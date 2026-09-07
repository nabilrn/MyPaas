#!/usr/bin/env bash
set -euo pipefail

SQLC_VERSION="1.31.1"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CACHE_ROOT="${XDG_CACHE_HOME:-${HOME}/.cache}/mypaas/sqlc/${SQLC_VERSION}"

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "unsupported OS for pinned sqlc: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture for pinned sqlc: $(uname -m)" >&2
    exit 1
    ;;
esac

case "${os}_${arch}" in
  linux_amd64) expected_sha256="497ae4fcdfa64c5b0c311ffe4c2bd991e43991e82e5367792ed78bc2dca27354" ;;
  linux_arm64) expected_sha256="b7cae247740d0c51a1e657479e5b2d21e6fef428f596682a01bc55bf4ab8a23d" ;;
  darwin_amd64) expected_sha256="c5af76772e3785d21663a62697056b383f07629979b1bd25b93872e73dbd519b" ;;
  darwin_arm64) expected_sha256="21602158c99eb1f2bae197a66abfb1941d1e9e50b23125bb193349c6b1acc71e" ;;
  *)
    echo "no checksum configured for ${os}_${arch}" >&2
    exit 1
    ;;
esac

archive="sqlc_${SQLC_VERSION}_${os}_${arch}.tar.gz"
archive_path="${CACHE_ROOT}/${archive}"
bin_path="${CACHE_ROOT}/sqlc"
url="https://github.com/sqlc-dev/sqlc/releases/download/v${SQLC_VERSION}/${archive}"

mkdir -p "${CACHE_ROOT}"

if [[ ! -f "${archive_path}" ]]; then
  command -v curl >/dev/null 2>&1 || {
    echo "curl is required to download sqlc" >&2
    exit 1
  }
  tmp_archive="${archive_path}.tmp.$$"
  trap 'rm -f "${tmp_archive:-}"' EXIT
  curl -fsSL "${url}" -o "${tmp_archive}"
  mv "${tmp_archive}" "${archive_path}"
  trap - EXIT
fi

if command -v sha256sum >/dev/null 2>&1; then
  echo "${expected_sha256}  ${archive_path}" | sha256sum -c - >/dev/null
elif command -v shasum >/dev/null 2>&1; then
  actual_sha256="$(shasum -a 256 "${archive_path}" | awk '{print $1}')"
  if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
    echo "sqlc archive checksum mismatch" >&2
    exit 1
  fi
else
  echo "sha256sum or shasum is required to verify sqlc" >&2
  exit 1
fi

if [[ ! -x "${bin_path}" ]]; then
  tar -xzf "${archive_path}" -C "${CACHE_ROOT}" sqlc
  chmod +x "${bin_path}"
fi

version_output="$(${bin_path} version)"
if [[ "${version_output}" != *"${SQLC_VERSION}"* ]]; then
  echo "unexpected sqlc version: ${version_output}" >&2
  exit 1
fi

if [[ $# -eq 0 ]]; then
  set -- generate
fi

cd "${REPO_ROOT}/backend"
exec "${bin_path}" "$@"
