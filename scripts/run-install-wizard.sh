#!/usr/bin/env bash
set -euo pipefail

WIZARD_SCRIPT="${WIZARD_SCRIPT:?WIZARD_SCRIPT is required}"
WIZARD_HOST="${WIZARD_HOST:-127.0.0.1}"
WIZARD_PORT="${WIZARD_PORT:-8787}"
WIZARD_TOKEN="${WIZARD_TOKEN:?WIZARD_TOKEN is required}"
WIZARD_PUBLIC_TUNNEL="${WIZARD_PUBLIC_TUNNEL:-true}"
WIZARD_TUNNEL_TIMEOUT="${WIZARD_TUNNEL_TIMEOUT:-120}"

WIZARD_PID=""
TUNNEL_PID=""
TUNNEL_NAME=""
TUNNEL_LOG=""

docker_cmd() {
  if docker ps >/dev/null 2>&1; then
    docker "$@"
    return
  fi
  sudo docker "$@"
}

print_local_access() {
  printf 'Local wizard URL: http://127.0.0.1:%s/?token=%s\n' "$WIZARD_PORT" "$WIZARD_TOKEN"
  printf 'SSH fallback: ssh -L %s:%s:%s <user>@<vm-ip>\n' "$WIZARD_PORT" "$WIZARD_HOST" "$WIZARD_PORT"
}

stop_quick_tunnel() {
  if [[ -n "$TUNNEL_PID" ]]; then
    kill "$TUNNEL_PID" >/dev/null 2>&1 || true
    wait "$TUNNEL_PID" >/dev/null 2>&1 || true
  fi
  if [[ -n "$TUNNEL_NAME" ]]; then
    docker_cmd rm -f "$TUNNEL_NAME" >/dev/null 2>&1 || true
  fi
  TUNNEL_PID=""
  TUNNEL_NAME=""
}

cleanup() {
  stop_quick_tunnel
  if [[ -n "$WIZARD_PID" ]]; then
    kill "$WIZARD_PID" >/dev/null 2>&1 || true
  fi
  [[ -z "$TUNNEL_LOG" ]] || rm -f "$TUNNEL_LOG"
}

start_quick_tunnel() {
  TUNNEL_NAME="mypaas-install-wizard-$$"
  TUNNEL_LOG="$(mktemp)"
  docker_cmd run --rm \
    --name "$TUNNEL_NAME" \
    --network host \
    cloudflare/cloudflared:latest \
    tunnel --no-autoupdate --url "http://127.0.0.1:$WIZARD_PORT" \
    >"$TUNNEL_LOG" 2>&1 &
  TUNNEL_PID=$!
}

wait_for_tunnel_url() {
  local elapsed=0 url=""
  while (( elapsed < WIZARD_TUNNEL_TIMEOUT )); do
    url="$(grep -Eo 'https://[A-Za-z0-9-]+\.trycloudflare\.com' "$TUNNEL_LOG" | head -n 1 || true)"
    if [[ -n "$url" ]]; then
      printf '%s' "$url"
      return
    fi
    kill -0 "$TUNNEL_PID" >/dev/null 2>&1 || return 1
    sleep 1
    elapsed=$((elapsed + 1))
  done
  return 1
}

main() {
  local public_url=""
  trap cleanup EXIT INT TERM
  [[ "$WIZARD_TUNNEL_TIMEOUT" =~ ^[1-9][0-9]*$ ]] || {
    printf 'ERROR: WIZARD_TUNNEL_TIMEOUT must be a positive integer\n' >&2
    return 1
  }

  python3 "$WIZARD_SCRIPT" &
  WIZARD_PID=$!

  if [[ "$WIZARD_PUBLIC_TUNNEL" == "true" ]]; then
    printf 'Creating temporary HTTPS wizard URL...\n'
    start_quick_tunnel
    public_url="$(wait_for_tunnel_url || true)"
  fi

  if [[ -n "$public_url" ]]; then
    printf '\nOpen this setup URL in your browser:\n%s/?token=%s\n\n' "$public_url" "$WIZARD_TOKEN"
    printf 'The temporary URL closes automatically after setup.\n'
  else
    stop_quick_tunnel
    printf 'WARN: Temporary HTTPS URL unavailable; use the SSH fallback.\n' >&2
    print_local_access
  fi

  wait "$WIZARD_PID"
}

main "$@"
