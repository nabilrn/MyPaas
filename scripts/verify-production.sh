#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env}"
RUN_BACKUP="${RUN_BACKUP:-false}"
DOCKER_BIN="${DOCKER_BIN:-docker}"
COMPOSE_BIN="${COMPOSE_BIN:-$DOCKER_BIN compose}"

cd "$ROOT_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

: "${CLOUDFLARE_TUNNEL_TOKEN:?CLOUDFLARE_TUNNEL_TOKEN is required}"

echo "Checking production containers..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps

echo "Checking Cloudflare Tunnel container..."
if [[ "$($DOCKER_BIN inspect --format '{{.State.Running}}' mypaas-cloudflared 2>/dev/null)" != "true" ]]; then
  echo "Cloudflare Tunnel container is not running." >&2
  exit 1
fi

if [[ -n "${STATD_SOCKET:-}" ]]; then
  echo "Checking mypaas-statd service and socket..."
  if ! command -v systemctl >/dev/null 2>&1; then
    echo "STATD_SOCKET is configured but systemctl is unavailable." >&2
    exit 1
  fi
  if ! systemctl is-active --quiet mypaas-statd; then
    echo "mypaas-statd.service is not active." >&2
    exit 1
  fi
  if [[ ! -S "$STATD_SOCKET" ]]; then
    echo "mypaas-statd socket is missing or is not a Unix socket: $STATD_SOCKET" >&2
    exit 1
  fi
  if ! $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api test -S "$STATD_SOCKET"; then
    echo "mypaas-statd socket is not visible inside the API container: $STATD_SOCKET" >&2
    exit 1
  fi
else
  echo "Skipping mypaas-statd verification because STATD_SOCKET is empty."
fi

echo "Checking API health..."
curl -fsS http://127.0.0.1:8080/health >/dev/null
curl -fsS http://127.0.0.1:8080/ready >/dev/null

echo "Checking Caddy Admin API..."
curl -fsS http://127.0.0.1:2019/config/apps/http/servers/srv0/routes >/dev/null

echo "Checking CLI binary inside API container..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api /app/mypaas help >/dev/null

if [[ "$RUN_BACKUP" == "true" ]]; then
  echo "Running manual backup through CLI..."
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api /app/mypaas backup
else
  echo "Skipping manual backup. Set RUN_BACKUP=true to verify backup output."
fi

echo "Production verification passed."
