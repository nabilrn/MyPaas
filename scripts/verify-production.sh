#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env}"
RUN_BACKUP="${RUN_BACKUP:-false}"
DOCKER_BIN="${DOCKER_BIN:-docker}"
COMPOSE_BIN="${COMPOSE_BIN:-$DOCKER_BIN compose}"
CADDY_ADMIN_SOCKET="${CADDY_ADMIN_SOCKET:-/run/mypaas/caddy-admin.sock}"
EXPECTED_BUILD_SHA="${EXPECTED_BUILD_SHA:-}"
EXPECTED_IMAGE_TAG="${EXPECTED_IMAGE_TAG:-$EXPECTED_BUILD_SHA}"
REQUIRE_PROJECT_ROUTE="${REQUIRE_PROJECT_ROUTE:-false}"

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
: "${PUBLIC_DOMAIN:?PUBLIC_DOMAIN is required}"

CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"
PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"
ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"
if [[ "$CONTROL_NETWORK" == "$PROJECT_NETWORK" || "$CONTROL_NETWORK" == "$ROUTING_NETWORK" || "$PROJECT_NETWORK" == "$ROUTING_NETWORK" ]]; then
  echo "CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct." >&2
  exit 1
fi

container_networks() {
  $DOCKER_BIN inspect --format '{{range $name, $_ := .NetworkSettings.Networks}}{{$name}} {{end}}' "$1"
}

container_env_value() {
  local container="$1"
  local key="$2"
  $DOCKER_BIN inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$container" 2>/dev/null \
    | sed -n "s/^${key}=//p" | tail -n 1
}

require_network() {
  local container="$1"
  local network="$2"
  local networks
  networks="$(container_networks "$container")"
  if [[ " $networks " != *" $network "* ]]; then
    echo "$container is not attached to required network $network (actual: $networks)." >&2
    exit 1
  fi
}

forbid_network() {
  local container="$1"
  local network="$2"
  local networks
  networks="$(container_networks "$container")"
  if [[ " $networks " == *" $network "* ]]; then
    echo "$container must not be attached to isolated network $network." >&2
    exit 1
  fi
}

caddy_admin_routes() {
  if curl -fsS --unix-socket "$CADDY_ADMIN_SOCKET" http://localhost/config/apps/http/servers/srv0/routes 2>/dev/null; then
    return 0
  fi
  if command -v sudo >/dev/null 2>&1; then
    sudo curl -fsS --unix-socket "$CADDY_ADMIN_SOCKET" http://localhost/config/apps/http/servers/srv0/routes
    return
  fi
  return 1
}

first_project_host() {
  local domain="$1"
  python3 -c '
import json, sys

domain = sys.argv[1].strip(".")
try:
    root = json.load(sys.stdin)
except Exception:
    raise SystemExit(1)
hosts = []
def walk(value):
    if isinstance(value, dict):
        for key, child in value.items():
            if key == "host" and isinstance(child, list):
                hosts.extend(str(item) for item in child)
            walk(child)
    elif isinstance(value, list):
        for child in value:
            walk(child)
walk(root)
for host in hosts:
    host = host.strip(".")
    if host != domain and host.endswith("." + domain) and "*" not in host:
        print(host)
        break
' "$domain"
}

dashboard_asset_paths() {
  python3 "$ROOT_DIR/scripts/extract_dashboard_assets.py"
}

echo "Checking production containers..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps

echo "Checking control/project/routing network boundaries..."
for container in mypaas-api mypaas-dashboard mypaas-cloudflared; do
  require_network "$container" "$CONTROL_NETWORK"
  forbid_network "$container" "$PROJECT_NETWORK"
  forbid_network "$container" "$ROUTING_NETWORK"
done

require_network mypaas-postgres-prod "$CONTROL_NETWORK"
require_network mypaas-postgres-prod "$PROJECT_NETWORK"
forbid_network mypaas-postgres-prod "$ROUTING_NETWORK"

require_network mypaas-caddy-prod "$CONTROL_NETWORK"
require_network mypaas-caddy-prod "$ROUTING_NETWORK"
forbid_network mypaas-caddy-prod "$PROJECT_NETWORK"

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
if [[ "${ENABLE_METRICS:-false}" == "true" ]]; then
  if [[ -z "${METRICS_USERNAME:-}" || -z "${METRICS_PASSWORD:-}" ]]; then
    echo "ENABLE_METRICS=true requires METRICS_USERNAME and METRICS_PASSWORD." >&2
    exit 1
  fi
  curl -fsS -u "$METRICS_USERNAME:$METRICS_PASSWORD" http://127.0.0.1:8080/metrics >/dev/null
fi

echo "Checking dashboard reachability..."
curl -fsS http://127.0.0.1:3000/ >/dev/null

echo "Checking dashboard release asset coherence..."
dashboard_headers="$(mktemp)"
# The SvelteKit root route intentionally redirects to /projects. Follow a small,
# bounded redirect chain so the verifier inspects the actual rendered HTML shell
# instead of treating the 307 response body as the dashboard document.
dashboard_html="$(curl -fsSL --max-redirs 5 -D "$dashboard_headers" -H "Host: $PUBLIC_DOMAIN" -H "Accept: text/html" http://127.0.0.1/)"
if ! grep -Eiq '^cache-control:.*no-store' "$dashboard_headers"; then
  rm -f "$dashboard_headers"
  echo "Dashboard HTML must be served with Cache-Control: no-store." >&2
  exit 1
fi
rm -f "$dashboard_headers"
dashboard_assets="$(printf '%s' "$dashboard_html" | dashboard_asset_paths)"
if [[ -z "$dashboard_assets" ]]; then
  echo "Dashboard HTML did not reference any immutable SvelteKit assets." >&2
  exit 1
fi
while IFS= read -r asset; do
  [[ -n "$asset" ]] || continue
  if ! curl -fsS -H "Host: $PUBLIC_DOMAIN" "http://127.0.0.1${asset}" >/dev/null; then
    echo "Dashboard HTML references an unavailable release asset: $asset" >&2
    exit 1
  fi
done <<< "$dashboard_assets"

echo "Checking running release identity..."
actual_build_sha="$(container_env_value mypaas-api MYPAAS_BUILD_SHA)"
if [[ -z "$actual_build_sha" || "$actual_build_sha" == "unknown" ]]; then
  echo "mypaas-api does not expose a concrete MYPAAS_BUILD_SHA." >&2
  exit 1
fi
if [[ -n "$EXPECTED_BUILD_SHA" && "$actual_build_sha" != "$EXPECTED_BUILD_SHA" ]]; then
  echo "Running build SHA $actual_build_sha does not match expected $EXPECTED_BUILD_SHA." >&2
  exit 1
fi
if [[ -n "$EXPECTED_IMAGE_TAG" ]]; then
  api_image="$($DOCKER_BIN inspect --format '{{.Config.Image}}' mypaas-api)"
  dashboard_image="$($DOCKER_BIN inspect --format '{{.Config.Image}}' mypaas-dashboard)"
  if [[ "$api_image" != *":$EXPECTED_IMAGE_TAG" ]]; then
    echo "API image $api_image does not match expected tag $EXPECTED_IMAGE_TAG." >&2
    exit 1
  fi
  if [[ "$dashboard_image" != *":$EXPECTED_IMAGE_TAG" ]]; then
    echo "Dashboard image $dashboard_image does not match expected tag $EXPECTED_IMAGE_TAG." >&2
    exit 1
  fi
fi

echo "Checking Caddy Admin Unix socket..."
if [[ ! -S "$CADDY_ADMIN_SOCKET" ]]; then
  echo "Caddy Admin socket is missing or is not a Unix socket: $CADDY_ADMIN_SOCKET" >&2
  exit 1
fi
routes_json="$(caddy_admin_routes)" || {
  echo "Caddy Admin API is not responsive through $CADDY_ADMIN_SOCKET." >&2
  exit 1
}
if ! $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api test -S "$CADDY_ADMIN_SOCKET"; then
  echo "Caddy Admin socket is not visible inside the API container: $CADDY_ADMIN_SOCKET" >&2
  exit 1
fi

if $DOCKER_BIN port mypaas-caddy-prod 2019/tcp 2>/dev/null | grep -q .; then
  echo "Caddy Admin TCP port 2019 must not be published." >&2
  exit 1
fi

echo "Checking an existing project route when available..."
project_host="$(printf '%s' "$routes_json" | first_project_host "$PUBLIC_DOMAIN" || true)"
if [[ -n "$project_host" ]]; then
  if curl -fsS -H "Host: $project_host" http://127.0.0.1/ >/dev/null; then
    echo "Verified project route $project_host through local Caddy."
  elif [[ "$REQUIRE_PROJECT_ROUTE" == "true" ]]; then
    echo "Existing project route $project_host is not healthy while REQUIRE_PROJECT_ROUTE=true." >&2
    exit 1
  else
    echo "Existing project route $project_host is currently unavailable; ignoring workload health for control-plane verification." >&2
  fi
elif [[ "$REQUIRE_PROJECT_ROUTE" == "true" ]]; then
  echo "No existing project route was found in Caddy while REQUIRE_PROJECT_ROUTE=true." >&2
  exit 1
else
  echo "No existing project route found; project-route verification skipped."
fi

echo "Checking CLI binary inside API container..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api /app/mypaas help >/dev/null

if [[ "$RUN_BACKUP" == "true" ]]; then
  echo "Running manual backup through CLI..."
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T api /app/mypaas backup
else
  echo "Skipping manual backup. Set RUN_BACKUP=true to verify backup output."
fi

echo "Production verification passed."
