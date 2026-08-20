#!/usr/bin/env bash
set -euo pipefail

cleanup_advice() {
  echo "" >&2
  echo "=================================================================" >&2
  echo "❌ DEPLOYMENT FAILED!" >&2
  echo "To clean up the failed deployment and start fresh, please run:" >&2
  echo "   bash scripts/uninstall-vm.sh" >&2
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

SUDO=""
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
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

echo "Running migrations..."
$DOCKER_BIN run --rm \
  --network "$CONTROL_NETWORK" \
  -v "$ROOT_DIR/backend/migrations:/migrations:ro" \
  migrate/migrate:latest \
  -path=/migrations \
  -database "$MIGRATE_DATABASE_URL" \
  up

echo "Starting MyPaas..."
if [[ "$SKIP_IMAGE_PULL" != "true" ]]; then
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull
fi
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d
if [[ "$RESTORED_CONTROL_PLANE_DB" == "true" ]]; then
  echo "Recreating API after database restore to trigger runtime reconciliation..."
  $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --force-recreate api
fi

echo "MyPaas production stack is starting. Run scripts/verify-production.sh after the containers settle."
