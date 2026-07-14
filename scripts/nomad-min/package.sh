#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT_DIR}/pkg/nomad_min/release}"
WORK_DIR="${WORK_DIR:-${ROOT_DIR}/pkg/nomad_min/package-work}"
VERSION="${NOMAD_VERSION:-$(tr -d '[:space:]' <"${ROOT_DIR}/nomad-min/UPSTREAM_VERSION")}"
ZSTD_LEVEL="${ZSTD_LEVEL:-22}"
PATCH_COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short=12 HEAD)"
UPSTREAM_COMMIT="$(cut -c1-12 "${ROOT_DIR}/nomad-min/UPSTREAM_COMMIT")"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

for tool in go make tar zstd; do
  command -v "${tool}" >/dev/null 2>&1 || {
    echo "${tool} is required" >&2
    exit 1
  }
done

cd "${ROOT_DIR}"
make nomad-min
mkdir -p "${OUT_DIR}" "${WORK_DIR}"

manifest_new="${OUT_DIR}/.nomad-min-manifest.tsv.new"
printf 'arch\tnomad\tcommit\tpackage\tsize_bytes\tsha256\tcompression\tbuild\n' >"${manifest_new}"

for arch in amd64 arm64; do
  stage="${WORK_DIR}/stage-${arch}"
  binary="${ROOT_DIR}/pkg/nomad_min/linux_${arch}/nomad"
  package="nomad-min-linux-${arch}.tar.zst"
  versioned="nomad-min-linux-${arch}-${VERSION}.tar.zst"

  rm -rf "${stage}"
  mkdir -p "${stage}/usr/local/bin" "${stage}/opt/mim/runtime"
  cp "${binary}" "${stage}/usr/local/bin/nomad"
  chmod 0755 "${stage}/usr/local/bin/nomad"
  printf '{\n  "nomad": "%s",\n  "upstream_commit": "%s",\n  "patch_commit": "%s",\n  "os": "linux",\n  "arch": "%s",\n  "build_profile": "nomad_min",\n  "command_surface": ["agent", "version"],\n  "build_tags": ["hashicorpmetrics", "release", "nomad_min"],\n  "compression": "zstd --ultra -%s"\n}\n' \
    "${VERSION}" "${UPSTREAM_COMMIT}" "${PATCH_COMMIT}" "${arch}" "${ZSTD_LEVEL}" \
    >"${stage}/opt/mim/runtime/NOMAD_MIN_VERSION.json"

  COPYFILE_DISABLE=1 tar -C "${stage}" -cf - . \
    | zstd --ultra "-${ZSTD_LEVEL}" -T0 -f -o "${OUT_DIR}/${package}"
  zstd -t "${OUT_DIR}/${package}"
  cp "${OUT_DIR}/${package}" "${OUT_DIR}/${versioned}"

  digest="$(sha256_file "${OUT_DIR}/${package}")"
  size="$(wc -c <"${OUT_DIR}/${package}" | tr -d ' ')"
  printf '%s  %s\n' "${digest}" "${package}" >"${OUT_DIR}/${package}.sha256"
  printf '%s\t%s\t%s\t%s\t%s\t%s\tzstd --ultra -%s\tnomad_min agent-only stripped static\n' \
    "${arch}" "${VERSION}" "${PATCH_COMMIT}" "${package}" "${size}" "${digest}" "${ZSTD_LEVEL}" \
    >>"${manifest_new}"
done

mv "${manifest_new}" "${OUT_DIR}/nomad-min-manifest.tsv"
cat "${OUT_DIR}/nomad-min-manifest.tsv"
