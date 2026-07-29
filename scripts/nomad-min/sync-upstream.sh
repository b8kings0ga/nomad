#!/usr/bin/env bash
# Copyright IBM Corp. 2015, 2026
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BASE_FILE="${ROOT_DIR}/nomad-min/UPSTREAM_COMMIT"
VERSION_FILE="${ROOT_DIR}/nomad-min/UPSTREAM_VERSION"
REMOTE="${NOMAD_MIN_UPSTREAM_REMOTE:-upstream}"
TARGET=""
WORKTREE=""
BRANCH=""

usage() {
  cat <<'EOF'
Usage: scripts/nomad-min/sync-upstream.sh [options]

Options:
  --target REF       Upstream tag or commit. Defaults to latest stable tag.
  --worktree PATH    Destination worktree. Defaults to a temporary directory.
  --branch NAME      Create NAME in the synchronized worktree.
  --remote NAME      Git remote containing HashiCorp Nomad (default: upstream).
  -h, --help         Show this help.

The current nomad_min commit stack is replayed onto the requested upstream
release. Conflicts are intentionally left in the worktree for inspection.
EOF
}

while (($#)); do
  case "$1" in
    --target) TARGET="${2:?missing target}"; shift 2 ;;
    --worktree) WORKTREE="${2:?missing worktree}"; shift 2 ;;
    --branch) BRANCH="${2:?missing branch}"; shift 2 ;;
    --remote) REMOTE="${2:?missing remote}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

for tool in git sort; do
  command -v "${tool}" >/dev/null 2>&1 || {
    echo "${tool} is required" >&2
    exit 1
  }
done

[[ -f "${BASE_FILE}" && -f "${VERSION_FILE}" ]] || {
  echo "nomad_min upstream metadata is missing" >&2
  exit 1
}

PATCH_BASE="$(tr -d '[:space:]' <"${BASE_FILE}")"
PATCH_HEAD="$(git -C "${ROOT_DIR}" rev-parse HEAD)"
git -C "${ROOT_DIR}" merge-base --is-ancestor "${PATCH_BASE}" "${PATCH_HEAD}" || {
  echo "recorded upstream commit ${PATCH_BASE} is not an ancestor of HEAD" >&2
  exit 1
}

if [[ -z "${TARGET}" ]]; then
  TARGET="$(git -C "${ROOT_DIR}" ls-remote --tags --refs "${REMOTE}" 'v*' \
    | awk '{sub("refs/tags/", "", $2); print $2}' \
    | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' \
    | sort -V \
    | tail -1)"
fi
[[ -n "${TARGET}" ]] || {
  echo "unable to resolve the latest stable upstream release" >&2
  exit 1
}

if [[ "${TARGET}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  git -C "${ROOT_DIR}" fetch --force --no-tags "${REMOTE}" \
    "refs/tags/${TARGET}:refs/tags/${TARGET}"
else
  git -C "${ROOT_DIR}" fetch --force --no-tags "${REMOTE}" "${TARGET}"
fi

TARGET_COMMIT="$(git -C "${ROOT_DIR}" rev-parse "${TARGET}^{commit}")"
TARGET_VERSION="${TARGET#v}"
if [[ "${TARGET_COMMIT}" == "${PATCH_BASE}" ]]; then
  echo "nomad_min is already based on ${TARGET} (${TARGET_COMMIT})"
  echo "NOMAD_MIN_SYNC_CHANGED=false"
  echo "NOMAD_MIN_SYNC_TARGET=${TARGET}"
  exit 0
fi

mapfile -t PATCH_COMMITS < <(git -C "${ROOT_DIR}" rev-list --reverse "${PATCH_BASE}..${PATCH_HEAD}")
((${#PATCH_COMMITS[@]} > 0)) || {
  echo "no nomad_min commits found after ${PATCH_BASE}" >&2
  exit 1
}

if [[ -z "${WORKTREE}" ]]; then
  WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/nomad-min-sync.XXXXXX")"
  rmdir "${WORKTREE}"
fi
[[ ! -e "${WORKTREE}" ]] || {
  echo "worktree destination already exists: ${WORKTREE}" >&2
  exit 1
}

git -C "${ROOT_DIR}" worktree add --no-checkout --detach "${WORKTREE}" "${TARGET_COMMIT}"
git -C "${WORKTREE}" sparse-checkout init --no-cone
git -C "${WORKTREE}" sparse-checkout set '/*' '!/ui/' '!/website/'
git -C "${WORKTREE}" checkout --detach "${TARGET_COMMIT}"

for commit in "${PATCH_COMMITS[@]}"; do
  echo "==> Replaying $(git -C "${ROOT_DIR}" show -s --format='%h %s' "${commit}")"
  if ! git -C "${WORKTREE}" cherry-pick "${commit}"; then
    echo >&2
    echo "nomad_min replay conflicted at ${commit}" >&2
    echo "worktree: ${WORKTREE}" >&2
    git -C "${WORKTREE}" diff --name-only --diff-filter=U >&2
    exit 1
  fi
done

printf '%s\n' "${TARGET_COMMIT}" >"${WORKTREE}/nomad-min/UPSTREAM_COMMIT"
printf '%s\n' "${TARGET_VERSION}" >"${WORKTREE}/nomad-min/UPSTREAM_VERSION"
git -C "${WORKTREE}" add nomad-min/UPSTREAM_COMMIT nomad-min/UPSTREAM_VERSION
if ! git -C "${WORKTREE}" diff --cached --quiet; then
  git -C "${WORKTREE}" commit -m "build: sync nomad_min to ${TARGET}"
fi

if [[ -n "${BRANCH}" ]]; then
  git -C "${WORKTREE}" switch -c "${BRANCH}"
fi

echo "NOMAD_MIN_SYNC_TARGET=${TARGET}"
echo "NOMAD_MIN_SYNC_COMMIT=$(git -C "${WORKTREE}" rev-parse HEAD)"
echo "NOMAD_MIN_SYNC_WORKTREE=${WORKTREE}"
echo "NOMAD_MIN_SYNC_CHANGED=true"
