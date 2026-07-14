#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TAGS="hashicorpmetrics release nomad_min"
BANNED_RE='github.com/hashicorp/(consul|vault)|github.com/hashicorp/go-(checkpoint|discover|getter)|prometheus|datadog|circonus|aws-sdk|azure-sdk|cloud.google.com'

cd "${ROOT_DIR}"

verify_tmp="$(mktemp -d "${TMPDIR:-/tmp}/nomad-min-verify.XXXXXX")"
trap 'rm -rf "${verify_tmp}"' EXIT

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
  ./client/allochealth
CGO_ENABLED=0 go test -run '^$' ./client/allocrunner/taskrunner

echo '==> Running focused nomad_min taskrunner regression tests'
CGO_ENABLED=0 go build -tags "${TAGS}" ./client/allocrunner/taskrunner
CGO_ENABLED=0 go test -tags "${TAGS}" \
  ./client/allocrunner/taskrunner/getter \
  ./client/allocrunner/taskrunner/template

# The integration tests require a Consul executable and are timing-sensitive
# on macOS. Run them only in environments that deliberately provide Consul;
# otherwise retain the ordinary-build compile regression.
if [[ "$(go env GOOS)" == "linux" ]] && command -v consul >/dev/null 2>&1; then
  CGO_ENABLED=0 go test ./command/agent/consul
else
  CGO_ENABLED=0 go test -run '^$' ./command/agent/consul
fi

echo '==> Checking the linux/amd64 nomad_min dependency graph'
deps="$(CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go list -tags "${TAGS}" -deps .)"
if grep -Eiq "${BANNED_RE}" <<<"${deps}"; then
  grep -Ei "${BANNED_RE}" <<<"${deps}" >&2
  echo 'nomad_min contains a forbidden dependency' >&2
  exit 1
fi

echo '==> Building linux/amd64 and linux/arm64 nomad_min binaries'
make nomad-min

host_binary="${verify_tmp}/nomad-min-host"
CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -tags "${TAGS}" -o "${host_binary}" .
profile="$("${host_binary}" version | awk '/^BuildProfile / {print $2}')"
[[ "${profile}" == "nomad_min" ]] || {
  echo "unexpected BuildProfile: ${profile:-missing}" >&2
  exit 1
}

echo '==> Checking removed runtime configuration is rejected'
cat >"${verify_tmp}/event-stream.hcl" <<EOF
data_dir = "${verify_tmp}/event-stream-data"
bind_addr = "127.0.0.1"

ports {
  http = 17646
  rpc  = 17647
  serf = 17648
}

advertise {
  http = "127.0.0.1:17646"
  rpc  = "127.0.0.1:17647"
  serf = "127.0.0.1:17648"
}

server {
  enabled             = true
  bootstrap_expect    = 1
  enable_event_broker = true
  event_buffer_size   = 100
}
EOF
if output="$("${host_binary}" agent -config="${verify_tmp}/event-stream.hcl" 2>&1)"; then
  echo 'nomad_min accepted the removed event stream configuration' >&2
  exit 1
fi
grep -Fq 'nomad_min: unsupported feature event stream' <<<"${output}" || {
  printf '%s\n' "${output}" >&2
  echo 'nomad_min did not report the expected event stream error' >&2
  exit 1
}

echo '==> nomad_min verification passed'
