#!/usr/bin/env bash

mypaas_update_status_enabled() {
  [[ "${MYPAAS_UPDATE_STATUS_ENABLED:-false}" == "true" ]]
}

mypaas_update_status_value() {
  printf '%s' "$1" | tr '\r\n=' '   '
}

mypaas_update_status_write() {
  local state="${1:-idle}"
  local phase="${2:-idle}"
  local message="${3:-}"
  local status_dir status_file tmp

  mypaas_update_status_enabled || return 0

  status_dir="${MYPAAS_UPDATE_STATUS_DIR:-/run/mypaas/update}"
  status_file="${MYPAAS_UPDATE_STATUS_FILE:-$status_dir/status}"
  mkdir -p "$status_dir"
  tmp="$(mktemp "$status_dir/.status.XXXXXX")"
  {
    printf 'state=%s\n' "$(mypaas_update_status_value "$state")"
    printf 'phase=%s\n' "$(mypaas_update_status_value "$phase")"
    printf 'channel=%s\n' "$(mypaas_update_status_value "${MYPAAS_UPDATE_CHANNEL:-unknown}")"
    printf 'current_sha=%s\n' "$(mypaas_update_status_value "${MYPAAS_UPDATE_CURRENT_SHA:-}")"
    printf 'target_sha=%s\n' "$(mypaas_update_status_value "${MYPAAS_UPDATE_TARGET_SHA:-}")"
    printf 'target_version=%s\n' "$(mypaas_update_status_value "${MYPAAS_UPDATE_TARGET_VERSION:-}")"
    printf 'message=%s\n' "$(mypaas_update_status_value "$message")"
    printf 'updated_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "$tmp"
  chmod 0644 "$tmp"
  mv -f "$tmp" "$status_file"
}
