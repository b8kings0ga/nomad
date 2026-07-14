#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TAGS="hashicorpmetrics release nomad_min"
BANNED_RE='github.com/hashicorp/(consul|vault)|github.com/hashicorp/go-(checkpoint|discover|getter)|prometheus|datadog|circonus|aws-sdk|azure-sdk|cloud.google.com'

cd "${ROOT_DIR}"

echo '==> Checking patch formatting'
git diff --check

echo '==> Compiling the ordinary build'
CGO_ENABLED=0 go test -run '^$' .

echo '==> Compiling nomad_min'
CGO_ENABLED=0 go test -tags "${TAGS}" -run '^$' .

echo '==> Running focused ordinary-build regression tests'
CGO_ENABLED=0 go test \
  ./nomad/structs/config \
  ./client/serviceregistration \
  ./client/consul \
  ./client/vaultclient \
  ./client/allochealth \
  ./client/allocrunner/taskrunner \
  ./command/agent/consul

echo '==> Checking the linux/amd64 nomad_min dependency graph'
deps="$(CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go list -tags "${TAGS}" -deps .)"
if grep -Eiq "${BANNED_RE}" <<<"${deps}"; then
  grep -Ei "${BANNED_RE}" <<<"${deps}" >&2
  echo 'nomad_min contains a forbidden dependency' >&2
  exit 1
fi

echo '==> Building linux/amd64 and linux/arm64 nomad_min binaries'
make nomad-min

profile="$(pkg/nomad_min/linux_amd64/nomad version | awk '/^BuildProfile / {print $2}')"
[[ "${profile}" == "nomad_min" ]] || {
  echo "unexpected BuildProfile: ${profile:-missing}" >&2
  exit 1
}

echo '==> nomad_min verification passed'
