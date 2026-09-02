#!/usr/bin/env bash
set -euo pipefail

cleanup_advice() {
  echo "" >&2
  echo "=================================================================" >&2
  echo "❌ DEPLOYMENT FAILED!" >&2
  echo "The existing MyPaas installation was not intentionally removed." >&2
  echo "Review the error above and fix the reported runtime or configuration issue before retrying." >&2
  echo "Do not run scripts/uninstall-vm.sh unless you intentionally want to remove MyPaas state." >&2
  echo "=================================================================" >&2
}
trap cleanup_advice ERR

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env}"
DOCKER_BIN="${DOCKER_BIN:-docker}"
COMPOSE_BIN="${COMPOSE_BIN:-$DOCKER_BIN compose}"
API_IMAGE_REPO="${MYPAAS_API_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-api}"
DASHBOARD_IMAGE_REPO="${MYPAAS_DASHBOARD_IMAGE_REPO:-ghcr.io/nabilrn/mypaas-dashboard}"
SKIP_IMAGE_PULL="${MYPAAS_SKIP_IMAGE_PULL:-false}"
SKIP_MIGRATIONS="${MYPAAS_SKIP_MIGRATIONS:-auto}"
MIGRATION_PIDS_LIMIT="${MIGRATION_PIDS_LIMIT:-256}"
COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"
EXPLICIT_IMAGE_TAG_SET="${MYPAAS_IMAGE_TAG+x}"
EXPLICIT_IMAGE_TAG="${MYPAAS_IMAGE_TAG:-}"
EXPLICIT_BUILD_SHA_SET="${MYPAAS_BUILD_SHA+x}"
EXPLICIT_BUILD_SHA="${MYPAAS_BUILD_SHA:-}"
RESTORED_CONTROL_PLANE_DB=false

cd "$ROOT_DIR"

if [[ -f "/tmp/mypaas-restore.tar.gz" ]]; then
  echo "Extracting backup bundle..."
  TMP_EXTRACT=$(mktemp -d)
  tar -xzf /tmp/mypaas-restore.tar.gz -C "$TMP_EXTRACT"

  if [[ -f "$TMP_EXTRACT/.env" ]]; then
    cat "$TMP_EXTRACT/.env" >> "$ENV_FILE"
    echo "Restored .env from backup (merged)."
  fi

  if [[ -f "$TMP_EXTRACT/database.sql" ]]; then
    mv "$TMP_EXTRACT/database.sql" /tmp/mypaas-database.sql
  fi

  rm -rf "$TMP_EXTRACT"
  rm -f /tmp/mypaas-restore.tar.gz
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE. Copy .env.example to .env and set production values first." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

resolve_runtime_socket() {
  local configured="${DOCKER_SOCKET:-}"
  if [[ -n "$configured" && -S "$configured" ]]; then
    printf '%s' "$configured"
    return
  fi
  if [[ -S /run/podman/podman.sock ]]; then
    printf '%s' /run/podman/podman.sock
    return
  fi
  if [[ -S /var/run/docker.sock ]]; then
    printf '%s' /var/run/docker.sock
    return
  fi
  return 1
}

if ! RESOLVED_DOCKER_SOCKET="$(resolve_runtime_socket)"; then
  echo "No live Docker-compatible runtime socket found. Expected configured DOCKER_SOCKET, /run/podman/podman.sock, or /var/run/docker.sock." >&2
  exit 1
fi
if [[ "${DOCKER_SOCKET:-}" != "$RESOLVED_DOCKER_SOCKET" ]]; then
  echo "Using live container runtime socket $RESOLVED_DOCKER_SOCKET instead of configured ${DOCKER_SOCKET:-<empty>}."
fi
DOCKER_SOCKET="$RESOLVED_DOCKER_SOCKET"
DOCKER_HOST="unix://$DOCKER_SOCKET"
export DOCKER_SOCKET DOCKER_HOST

if [[ -n "$EXPLICIT_IMAGE_TAG_SET" ]]; then
  MYPAAS_IMAGE_TAG="$EXPLICIT_IMAGE_TAG"
  export MYPAAS_IMAGE_TAG
fi

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${PUBLIC_DOMAIN:?PUBLIC_DOMAIN is required}"
: "${OWNER_EMAIL:?OWNER_EMAIL is required}"
: "${GITHUB_CLIENT_ID:?GITHUB_CLIENT_ID is required}"
: "${GITHUB_CLIENT_SECRET:?GITHUB_CLIENT_SECRET is required}"
: "${GITHUB_CALLBACK_URL:?GITHUB_CALLBACK_URL is required}"
: "${JWT_SECRET:?JWT_SECRET is required}"
: "${ENCRYPTION_KEY:?ENCRYPTION_KEY is required}"
: "${DOCKER_SOCKET:?DOCKER_SOCKET is required}"
: "${CLOUDFLARE_TUNNEL_TOKEN:?CLOUDFLARE_TUNNEL_TOKEN is required}"

is_valid_public_domain() {
  local value="$1"
  [[ ${#value} -le 253 ]] || return 1
  [[ "$value" == *.* ]] || return 1
  [[ "$value" =~ ^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)+[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$ ]]
}

if ! is_valid_public_domain "$PUBLIC_DOMAIN"; then
  echo "PUBLIC_DOMAIN must be a bare DNS hostname like mypaas.example.com; URLs, paths, ports, credentials, queries, and malformed labels are not allowed." >&2
  exit 2
fi

if [[ "$SKIP_IMAGE_PULL" != "true" && "$SKIP_IMAGE_PULL" != "false" ]]; then
  echo "MYPAAS_SKIP_IMAGE_PULL must be true or false." >&2
  exit 2
fi
if [[ "$SKIP_MIGRATIONS" != "auto" && "$SKIP_MIGRATIONS" != "true" && "$SKIP_MIGRATIONS" != "false" ]]; then
  echo "MYPAAS_SKIP_MIGRATIONS must be auto, true, or false." >&2
  exit 2
fi
if [[ ! "$MIGRATION_PIDS_LIMIT" =~ ^[1-9][0-9]*$ ]]; then
  echo "MIGRATION_PIDS_LIMIT must be a positive integer." >&2
  exit 2
fi
if [[ ! "$COMPOSE_PARALLEL_LIMIT" =~ ^[1-9][0-9]*$ ]]; then
  echo "COMPOSE_PARALLEL_LIMIT must be a positive integer." >&2
  exit 2
fi
export COMPOSE_PARALLEL_LIMIT

SUDO=""
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
fi

if command -v systemctl >/dev/null 2>&1; then
  echo "Installing MyPaaS managed firewall helper..."
  bash "$ROOT_DIR/scripts/install-firewall-helper.sh"
  echo "Installing MyPaaS host update trigger..."
  MYPAAS_INSTALL_DIR="$ROOT_DIR" ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/configure-auto-update.sh"
else
  echo "systemd unavailable; firewall management and dashboard-triggered updates are disabled."
fi

for dir in \
  /var/lib/mypaas/volumes \
  /var/lib/mypaas/compose \
  /var/lib/mypaas/static \
  /var/lib/mypaas/backups \
  /tmp/mypaas/builds
do
  $SUDO mkdir -p "$dir"
done

CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"
PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"
ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"
if [[ "$CONTROL_NETWORK" == "$PROJECT_NETWORK" || "$CONTROL_NETWORK" == "$ROUTING_NETWORK" || "$PROJECT_NETWORK" == "$ROUTING_NETWORK" ]]; then
  echo "CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct." >&2
  exit 1
fi

port_is_listening() {
  local port="$1"
  ss -H -ltn "sport = :$port" 2>/dev/null | grep -q .
}

managed_container_running() {
  local name="$1"
  [[ "$($DOCKER_BIN inspect --format '{{.State.Running}}' "$name" 2>/dev/null || true)" == "true" ]]
}

preflight_managed_port() {
  local port="$1"
  local container="$2"

  if ! port_is_listening "$port"; then
    return
  fi
  if managed_container_running "$container"; then
    return
  fi

  echo "Port $port is already in use while $container is not running in the selected container runtime." >&2
  echo "This can indicate a stale container proxy or MyPaas state owned by another Docker-compatible engine." >&2
  echo "Refusing to continue before changing deployment state. Inspect the port owner and container runtime first." >&2
  exit 1
}

preflight_control_plane_ports() {
  if ! command -v ss >/dev/null 2>&1; then
    echo "Warning: ss is unavailable; skipping control-plane port ownership preflight." >&2
    return
  fi

  preflight_managed_port 5432 mypaas-postgres-prod
  preflight_managed_port 8080 mypaas-api
  preflight_managed_port 3000 mypaas-dashboard
  preflight_managed_port 80 mypaas-caddy-prod
}

preflight_control_plane_ports

for network in "$CONTROL_NETWORK" "$PROJECT_NETWORK" "$ROUTING_NETWORK"; do
  $DOCKER_BIN network inspect "$network" >/dev/null 2>&1 || $DOCKER_BIN network create "$network" >/dev/null
done

if [[ -z "${MYPAAS_IMAGE_TAG:-}" && -d "$ROOT_DIR/.git" ]]; then
  MYPAAS_IMAGE_TAG="$(git -c safe.directory="$ROOT_DIR" rev-parse HEAD)"
  export MYPAAS_IMAGE_TAG
  echo "Using MyPaas image tag ${MYPAAS_IMAGE_TAG:0:12} from the current checkout."
fi

# Build identity is derived from the immutable release tag by default. The
# updater overrides it only during rollback, when the runtime image tag is a
# temporary rollback-* tag but the owner-facing identity must remain the Git SHA
# that produced those images.
if [[ -n "$EXPLICIT_BUILD_SHA_SET" ]]; then
  MYPAAS_BUILD_SHA="$EXPLICIT_BUILD_SHA"
else
  MYPAAS_BUILD_SHA="${MYPAAS_IMAGE_TAG:-unknown}"
fi
export MYPAAS_BUILD_SHA

if [[ -n "${MYPAAS_IMAGE_TAG:-}" ]]; then
  if [[ "$SKIP_IMAGE_PULL" == "true" ]]; then
    echo "Using verified local MyPaas images for ${MYPAAS_IMAGE_TAG:0:12}..."
    if ! $DOCKER_BIN image inspect "$API_IMAGE_REPO:$MYPAAS_IMAGE_TAG" >/dev/null 2>&1; then
      echo "Missing local API image $API_IMAGE_REPO:$MYPAAS_IMAGE_TAG." >&2
      exit 1
    fi
    if ! $DOCKER_BIN image inspect "$DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG" >/dev/null 2>&1; then
      echo "Missing local dashboard image $DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG." >&2
      exit 1
    fi
  else
    echo "Pulling MyPaas release images for ${MYPAAS_IMAGE_TAG:0:12}..."
    if ! $DOCKER_BIN pull "$API_IMAGE_REPO:$MYPAAS_IMAGE_TAG"; then
      echo "Missing API image $API_IMAGE_REPO:$MYPAAS_IMAGE_TAG." >&2
      echo "Wait for the Docker publish workflow to finish, or set MYPAAS_IMAGE_TAG explicitly." >&2
      exit 1
    fi
    if ! $DOCKER_BIN pull "$DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG"; then
      echo "Missing dashboard image $DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG." >&2
      echo "Wait for the Docker publish workflow to finish, or set MYPAAS_IMAGE_TAG explicitly." >&2
      exit 1
    fi
  fi
fi

current_runtime_build_sha() {
  $DOCKER_BIN inspect --format '{{range .Config.Env}}{{println .}}{{end}}' mypaas-api 2>/dev/null \
    | sed -n 's/^MYPAAS_BUILD_SHA=//p' \
    | tail -n 1
}

should_skip_migrations() {
  local runtime_sha

  # A restored database must always be brought to the checkout schema.
  if [[ "$RESTORED_CONTROL_PLANE_DB" == "true" ]]; then
    return 1
  fi
  if [[ "$SKIP_MIGRATIONS" == "true" ]]; then
    return 0
  fi
  if [[ "$SKIP_MIGRATIONS" == "false" ]]; then
    return 1
  fi

  # Runtime rollback restores application images only. Running an older
  # checkout's `migrate up` cannot downgrade a schema and only adds another
  # failure point to the recovery path.
  if [[ "$SKIP_IMAGE_PULL" == "true" && "${MYPAAS_IMAGE_TAG:-}" == rollback-* ]]; then
    return 0
  fi

  # Normal updates should not launch a separate Go migration helper when the
  # migration tree is byte-for-byte unchanged from the currently running API.
  runtime_sha="$(current_runtime_build_sha)"
  if [[ "$runtime_sha" =~ ^[0-9a-fA-F]{40}$ ]] \
    && git -c safe.directory="$ROOT_DIR" cat-file -e "${runtime_sha}^{commit}" 2>/dev/null \
    && git -c safe.directory="$ROOT_DIR" diff --quiet "$runtime_sha" HEAD -- backend/migrations; then
    return 0
  fi

  return 1
}

echo "Starting PostgreSQL..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d postgres

echo "Waiting for PostgreSQL..."
until $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
  sleep 2
done

if [[ -f "/tmp/mypaas-database.sql" ]]; then
  echo "Found database backup, restoring..."
  cat /tmp/mypaas-database.sql | $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T -i postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
  rm -f /tmp/mypaas-database.sql
  RESTORED_CONTROL_PLANE_DB=true
  echo "Database restored successfully."
fi

MIGRATE_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"

if should_skip_migrations; then
  echo "Skipping migrations: control-plane migration tree is unchanged or this is an application rollback."
else
  echo "Running migrations..."
  $DOCKER_BIN run --rm \
    --pids-limit "$MIGRATION_PIDS_LIMIT" \
    --network "$CONTROL_NETWORK" \
    -v "$ROOT_DIR/backend/migrations:/migrations:ro" \
    migrate/migrate:latest \
    -path=/migrations \
    -database "$MIGRATE_DATABASE_URL" \
    up
fi

echo "Starting MyPaas control plane sequentially..."
if [[ "$SKIP_IMAGE_PULL" != "true" ]]; then
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull
fi

echo "Starting dashboard..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps dashboard

if managed_container_running mypaas-caddy-prod; then
  echo "Reloading caddy configuration from the current checkout without restarting the proxy..."
  # A single-file bind mount can keep pointing at the inode that existed before
  # git reset/checkout replaced Caddyfile.prod. Stream the checked-out config
  # into the running container so hot reload always uses this release's bytes.
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T caddy \
    sh -c 'cat > /tmp/mypaas-Caddyfile.next' < "$ROOT_DIR/Caddyfile.prod"
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T caddy \
    caddy reload --config /tmp/mypaas-Caddyfile.next --adapter caddyfile \
    --address unix//run/mypaas/caddy-admin.sock
else
  echo "Starting caddy..."
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps caddy
fi

echo "Starting api..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps api

echo "Starting cloudflared..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --no-deps cloudflared

if [[ "$RESTORED_CONTROL_PLANE_DB" == "true" ]]; then
  echo "Recreating API after database restore to trigger runtime reconciliation..."
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --force-recreate --no-deps api
fi

echo "MyPaas production stack is starting. Run scripts/verify-production.sh after the containers settle."
