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

cd "$ROOT_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE. Copy .env.example to .env and set production values first." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

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

if [[ -n "${MYPAAS_IMAGE_TAG:-}" ]]; then
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

echo "Starting PostgreSQL..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d postgres

echo "Waiting for PostgreSQL..."
until $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
  sleep 2
done

MIGRATE_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"

echo "Running migrations..."
$DOCKER_BIN run --rm \
  --network "$CONTROL_NETWORK" \
  -v "$ROOT_DIR/backend/migrations:/migrations:ro" \
  migrate/migrate:latest \
  -path=/migrations \
  -database "$MIGRATE_DATABASE_URL" \
  up

echo "Pulling and starting MyPaas..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d

echo "MyPaas production stack is starting. Run scripts/verify-production.sh after the containers settle."
