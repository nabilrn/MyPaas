#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${MYPAAS_INSTALL_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
EXPECTED_STATD_VERSION="${STATD_VERSION:-v0.2.0}"
DEFAULT_STATD_SOCKET="/run/mypaas/statd.sock"

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

run_root() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    command -v sudo >/dev/null 2>&1 || die "sudo is required to reconcile mypaas-statd"
    sudo "$@"
  fi
}

persist_socket_config() {
  local current=""
  [[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

  current="$(grep -E '^STATD_SOCKET=' "$ENV_FILE" | tail -n 1 | cut -d= -f2- || true)"
  if [[ -n "$current" ]]; then
    printf '%s' "$current"
    return
  fi

  if grep -qE '^STATD_SOCKET=' "$ENV_FILE"; then
    sed -i "s#^STATD_SOCKET=.*#STATD_SOCKET=$DEFAULT_STATD_SOCKET#" "$ENV_FILE"
  else
    printf '\nSTATD_SOCKET=%s\n' "$DEFAULT_STATD_SOCKET" >> "$ENV_FILE"
  fi
  printf '%s' "$DEFAULT_STATD_SOCKET"
}

statd_is_current() {
  local socket_path="$1"
  [[ -x /usr/local/bin/mypaas-statd ]] || return 1
  [[ "$(/usr/local/bin/mypaas-statd --version 2>/dev/null || true)" == "mypaas-statd ${EXPECTED_STATD_VERSION#v}" ]] || return 1
  command -v systemctl >/dev/null 2>&1 || return 1
  systemctl is-active --quiet mypaas-statd || return 1
  [[ -S "$socket_path" ]] || return 1
}

main() {
  local socket_path
  socket_path="$(persist_socket_config)"

  if statd_is_current "$socket_path"; then
    log "mypaas-statd $EXPECTED_STATD_VERSION is already healthy"
    return 0
  fi

  log "Reconciling mypaas-statd $EXPECTED_STATD_VERSION"
  ENV_FILE="$ENV_FILE" STATD_VERSION="$EXPECTED_STATD_VERSION" bash "$ROOT_DIR/scripts/install-vm.sh" --statd-only

  [[ "$(run_root /usr/local/bin/mypaas-statd --version)" == "mypaas-statd ${EXPECTED_STATD_VERSION#v}" ]] \
    || die "installed mypaas-statd version does not match $EXPECTED_STATD_VERSION"
  run_root systemctl is-active --quiet mypaas-statd || die "mypaas-statd.service is not active"
  [[ -S "$socket_path" ]] || die "mypaas-statd socket is missing: $socket_path"

  log "mypaas-statd host telemetry runtime is healthy"
}

main "$@"
