#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE="${ENV_FILE:-.env}"
DOCKER_BIN="${DOCKER_BIN:-docker}"
COMPOSE_BIN="${COMPOSE_BIN:-$DOCKER_BIN compose}"

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

$DOCKER_BIN network inspect "${PROJECT_NETWORK:-mypaas-prod}" >/dev/null 2>&1 || $DOCKER_BIN network create "${PROJECT_NETWORK:-mypaas-prod}" >/dev/null

echo "Starting PostgreSQL..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d postgres

echo "Waiting for PostgreSQL..."
until $COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; do
  sleep 2
done

MIGRATE_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"

echo "Running migrations..."
$DOCKER_BIN run --rm \
  --network mypaas-prod \
  -v "$ROOT_DIR/backend/migrations:/migrations:ro" \
  migrate/migrate:latest \
  -path=/migrations \
  -database "$MIGRATE_DATABASE_URL" \
  up

echo "Building and starting MyPaas..."
$COMPOSE_BIN -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --build

echo "MyPaas production stack is starting. Run scripts/verify-production.sh after the containers settle."
