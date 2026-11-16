#!/usr/bin/env bash
# Copyright IBM Corp. 2015, 2026
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/pkg/nomad_min/release}"
VERSION="$(tr -d '[:space:]' <"${ROOT_DIR}/nomad-min/UPSTREAM_VERSION")"
COMMIT="$(git -C "${ROOT_DIR}" rev-parse HEAD)"
UPSTREAM_COMMIT="$(cut -c1-12 "${ROOT_DIR}/nomad-min/UPSTREAM_COMMIT")"

[[ "$(uname -s)" == Linux ]] || { echo 'package execution checks require Linux' >&2; exit 1; }
for tool in jq go zstd tar sha256sum; do
  command -v "${tool}" >/dev/null || { echo "${tool} is required" >&2; exit 1; }
done

verify_tmp="$(mktemp -d)"
trap 'rm -rf "${verify_tmp}"' EXIT
mapfile -t rows < <(tail -n +2 "${OUT_DIR}/nomad-min-manifest.tsv")
[[ ${#rows[@]} == 2 ]] || { echo 'manifest must contain exactly two architectures' >&2; exit 1; }
seen=""
for row in "${rows[@]}"; do
  IFS=$'\t' read -r arch version commit package size digest compression build <<<"${row}"
  [[ "${arch}" == amd64 || "${arch}" == arm64 ]]
  [[ " ${seen} " != *" ${arch} "* ]]
  seen="${seen} ${arch}"
  [[ "${version}" == "${VERSION}" && "${commit}" == "${COMMIT:0:12}" ]]
  [[ "${package}" == "nomad-min-linux-${arch}.tar.zst" ]]
  [[ "$(wc -c <"${OUT_DIR}/${package}" | tr -d ' ')" == "${size}" ]]
  [[ "$(sha256sum "${OUT_DIR}/${package}" | awk '{print $1}')" == "${digest}" ]]
  (cd "${OUT_DIR}" && sha256sum -c "${package}.sha256")
  cmp "${OUT_DIR}/${package}" "${OUT_DIR}/nomad-min-linux-${arch}-${VERSION}.tar.zst"
  zstd -t "${OUT_DIR}/${package}"
  mkdir "${verify_tmp}/${arch}"
  zstd -dc "${OUT_DIR}/${package}" | tar -xf - -C "${verify_tmp}/${arch}"
  binary="${verify_tmp}/${arch}/usr/local/bin/nomad"
  jq -e --arg version "${VERSION}" --arg commit "${COMMIT:0:12}" --arg upstream "${UPSTREAM_COMMIT}" --arg arch "${arch}" \
    '.nomad == $version and .patch_commit == $commit and .upstream_commit == $upstream and .os == "linux" and .arch == $arch and .build_profile == "nomad_min" and .command_surface == ["agent", "version"]' \
    "${verify_tmp}/${arch}/opt/mim/runtime/NOMAD_MIN_VERSION.json" >/dev/null
  info="$(go version -m "${binary}")"
  grep -Eq 'GOOS=linux$' <<<"${info}"
  grep -Eq "GOARCH=${arch}$" <<<"${info}"
  grep -Eq 'CGO_ENABLED=0$' <<<"${info}"
  case "$(uname -m):${arch}" in
    x86_64:amd64|aarch64:arm64) output="$("${binary}" version)" ;;
    *:arm64) output="$(qemu-aarch64 "${binary}" version)" ;;
    *:amd64) output="$(qemu-x86_64 "${binary}" version)" ;;
  esac
  grep -Fxq "Nomad v${VERSION}" <<<"${output}"
  grep -Fxq "Revision ${COMMIT}" <<<"${output}"
  grep -Fxq 'BuildProfile nomad_min' <<<"${output}"
  printf '%s: %s\n%s\n' "${arch}" "${package}" "${output}"
done
echo '==> nomad_min packages verified'
