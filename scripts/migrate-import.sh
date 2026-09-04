#!/usr/bin/env bash
# migrate-import.sh — Run on the NEW VM from the MyPaas project root.
# Restores a MyPaas export archive and brings up the full stack.
set -euo pipefail

ARCHIVE="${1:?Usage: $0 <path-to-mypaas-export.tar.gz>}"
IMPORT_DIR="/tmp/mypaas-import"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*" >&2; exit 1; }

echo ""
echo "============================================"
echo "  MyPaas VM Migration — Import"
echo "============================================"
echo ""

# ── 1. Pre-flight ────────────────────────────────────────────────────
info "Pre-flight checks..."

command -v docker &>/dev/null       || fail "docker not found — install Docker first"
docker compose version &>/dev/null  || fail "docker compose plugin not found"
[ -f "$ARCHIVE" ]                   || fail "Archive not found: $ARCHIVE"
[ -f docker-compose.prod.yml ]      || fail "docker-compose.prod.yml not found — run from MyPaas project root"

ok "Pre-flight passed"

# ── 2. Extract archive ──────────────────────────────────────────────
info "Extracting archive..."
rm -rf "$IMPORT_DIR"
mkdir -p "$IMPORT_DIR"
tar xzf "$ARCHIVE" -C "$IMPORT_DIR"

# Verify critical files
[ -f "$IMPORT_DIR/databases/system.dump" ] || fail "system.dump missing from archive"
[ -f "$IMPORT_DIR/dot-env" ]               || fail ".env missing from archive"

if [ -f "$IMPORT_DIR/manifest.json" ]; then
    info "Export manifest:"
    cat "$IMPORT_DIR/manifest.json"
    echo ""
fi
ok "Archive extracted"

# ── 3. Restore .env ─────────────────────────────────────────────────
info "Restoring .env..."
if [ -f .env ]; then
    cp .env ".env.backup-$(date +%Y%m%d-%H%M%S)"
    warn "Existing .env backed up"
fi
cp "$IMPORT_DIR/dot-env" .env
ok ".env restored"

echo ""
echo -e "${YELLOW}════════════════════════════════════════════${NC}"
echo -e "${YELLOW}  REVIEW .env BEFORE CONTINUING${NC}"
echo -e "${YELLOW}════════════════════════════════════════════${NC}"
echo ""
echo "  Critical — these MUST stay the same:"
echo "    ENCRYPTION_KEY   (env vars won't decrypt if changed)"
echo "    JWT_SECRET"
echo ""
echo "  May need updating:"
echo "    CLOUDFLARE_TUNNEL_TOKEN  (if creating new tunnel)"
echo "    GITHUB_CALLBACK_URL      (if domain changed)"
echo "    DOCKER_BIND_HOST         (published-port identity binding)"
echo "    DOCKER_SOCKET            (usually /var/run/docker.sock)"
echo "    CONTROL_NETWORK          (default mypaas-control)"
echo "    PROJECT_NETWORK          (default mypaas-projects)"
echo "    ROUTING_NETWORK          (default mypaas-routing)"
echo ""
read -r -p "  Press Enter after reviewing .env to continue..."

# Reload .env
# shellcheck disable=SC1091
source .env

DB_USER="${POSTGRES_USER:?POSTGRES_USER not set}"
DB_NAME="${POSTGRES_DB:?POSTGRES_DB not set}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-mypaas-postgres-prod}"
CONTROL_NETWORK="${CONTROL_NETWORK:-mypaas-control}"
PROJECT_NETWORK="${PROJECT_NETWORK:-mypaas-projects}"
ROUTING_NETWORK="${ROUTING_NETWORK:-mypaas-routing}"

if [[ "$CONTROL_NETWORK" == "$PROJECT_NETWORK" || "$CONTROL_NETWORK" == "$ROUTING_NETWORK" || "$PROJECT_NETWORK" == "$ROUTING_NETWORK" ]]; then
    fail "CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct"
fi

# ── 4. Restore persistent directories ───────────────────────────────
info "Restoring persistent data..."

restore_dir() {
    local src="$IMPORT_DIR/$1" dest="/var/lib/mypaas/$1"
    if [ -d "$src" ]; then
        sudo mkdir -p "$dest"
        sudo cp -a "$src/." "$dest/"
        ok "  /var/lib/mypaas/$1"
    else
        warn "  $1 not in archive — skipping"
    fi
}

restore_engine_volumes() {
    local stage="/var/lib/mypaas/volumes/.migration-engine-volumes"
    [ -d "$stage" ] || return 0

    info "Restoring engine-managed Compose volumes..."
    local volume_dir volume_name mountpoint restored=0
    for volume_dir in "$stage"/*; do
        [ -d "$volume_dir" ] || continue
        volume_name="$(basename "$volume_dir")"
        [[ "$volume_name" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]] || fail "Unsafe volume name in migration package: $volume_name"

        docker volume create "$volume_name" >/dev/null
        mountpoint="$(docker volume inspect "$volume_name" --format '{{ .Mountpoint }}' 2>/dev/null || true)"
        [ -n "$mountpoint" ] || fail "Could not resolve mountpoint for restored volume: $volume_name"

        sudo mkdir -p "$mountpoint"
        sudo find "$mountpoint" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
        sudo cp -a "$volume_dir/." "$mountpoint/"
        ok "  named volume $volume_name"
        restored=$((restored + 1))
    done

    sudo rm -rf "$stage"
    if [ "$restored" -eq 0 ]; then
        warn "  engine-volume staging directory was empty"
    else
        ok "Restored $restored engine-managed Compose volume(s)"
    fi
}

sudo mkdir -p /var/lib/mypaas
restore_dir "volumes"
restore_engine_volumes
restore_dir "compose"
restore_dir "static"
sudo mkdir -p /var/lib/mypaas/backups
sudo mkdir -p /tmp/mypaas/builds

# ── 5. Create Docker-compatible networks ────────────────────────────
info "Ensuring MyPaas control/project/routing networks exist..."
for network in "$CONTROL_NETWORK" "$PROJECT_NETWORK" "$ROUTING_NETWORK"; do
    if docker network inspect "$network" >/dev/null 2>&1; then
        ok "Network $network already exists"
    else
        docker network create "$network" >/dev/null
        ok "Network $network created"
    fi
done

# ── 6. Start PostgreSQL ─────────────────────────────────────────────
info "Starting PostgreSQL..."
docker compose -f docker-compose.prod.yml --env-file .env up -d postgres

info "Waiting for PostgreSQL to be healthy..."
for i in $(seq 1 60); do
    if docker exec "$POSTGRES_CONTAINER" pg_isready -U "$DB_USER" &>/dev/null; then
        break
    fi
    if [ "$i" -eq 60 ]; then
        fail "PostgreSQL did not become ready in 60 seconds"
    fi
    sleep 1
done
ok "PostgreSQL is ready"

# ── 7. Restore system database ──────────────────────────────────────
info "Restoring MyPaas system database (${DB_NAME})..."

# Drop & recreate for clean restore
docker exec "$POSTGRES_CONTAINER" dropdb -U "$DB_USER" --if-exists "$DB_NAME"
docker exec "$POSTGRES_CONTAINER" createdb -U "$DB_USER" "$DB_NAME"

docker cp "$IMPORT_DIR/databases/system.dump" "${POSTGRES_CONTAINER}:/tmp/_system.dump"
docker exec "$POSTGRES_CONTAINER" pg_restore \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    --no-owner \
    --no-privileges \
    "/tmp/_system.dump"
docker exec "$POSTGRES_CONTAINER" rm "/tmp/_system.dump"
ok "System database restored"

# ── 8. Restore roles & shared project databases ─────────────────────
if [ -f "$IMPORT_DIR/databases/roles.sql" ]; then
    info "Restoring database roles..."
    docker cp "$IMPORT_DIR/databases/roles.sql" "${POSTGRES_CONTAINER}:/tmp/_roles.sql"
    # roles.sql may contain CREATE ROLE for postgres which already exists — ignore errors
    docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -f "/tmp/_roles.sql" 2>/dev/null || true
    docker exec "$POSTGRES_CONTAINER" rm "/tmp/_roles.sql"
    ok "Roles restored"
fi

SHARED_DBS=$(find "$IMPORT_DIR/databases" -name 'mypaas_p_*.dump' 2>/dev/null || true)
if [ -n "$SHARED_DBS" ]; then
    DB_COUNT=$(echo "$SHARED_DBS" | wc -l | tr -d ' ')
    info "Restoring ${DB_COUNT} shared project database(s)..."

    for dumpfile in $SHARED_DBS; do
        dbname=$(basename "$dumpfile" .dump)
        info "  Restoring ${dbname}..."

        docker exec "$POSTGRES_CONTAINER" dropdb -U "$DB_USER" --if-exists "$dbname"
        docker exec "$POSTGRES_CONTAINER" createdb -U "$DB_USER" "$dbname"

        docker cp "$dumpfile" "${POSTGRES_CONTAINER}:/tmp/_${dbname}.dump"
        docker exec "$POSTGRES_CONTAINER" pg_restore \
            -U "$DB_USER" \
            -d "$dbname" \
            --no-owner \
            --no-privileges \
            "/tmp/_${dbname}.dump" 2>/dev/null || warn "  Some warnings during ${dbname} restore (usually safe)"
        docker exec "$POSTGRES_CONTAINER" rm "/tmp/_${dbname}.dump"

        # Reassign ownership if the role exists
        role="${dbname}_user"
        role_exists=$(docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -A -c \
            "SELECT 1 FROM pg_roles WHERE rolname = '${role}'" 2>/dev/null || true)
        if [ "$role_exists" = "1" ]; then
            docker exec "$POSTGRES_CONTAINER" psql -U "$DB_USER" -d "$dbname" -c \
                "REASSIGN OWNED BY \"${DB_USER}\" TO \"${role}\"" 2>/dev/null || true
        fi
    done
    ok "Shared databases restored"
else
    ok "No shared project databases to restore"
fi

# ── 9. Start full stack ──────────────────────────────────────────────
info "Pulling and starting MyPaas stack..."
docker compose -f docker-compose.prod.yml --env-file .env pull
docker compose -f docker-compose.prod.yml --env-file .env up -d

# ── 10. Cleanup ──────────────────────────────────────────────────────
rm -rf "$IMPORT_DIR"

echo ""
echo "============================================"
echo -e "  ${GREEN}Import Complete!${NC}"
echo "============================================"
echo ""
echo "  MyPaas is starting up at https://${PUBLIC_DOMAIN:-your-domain}"
echo ""
echo "  NEXT STEPS:"
echo ""
echo "  1. Verify the dashboard is accessible"
echo ""
echo "  2. Redeploy all projects (images need to rebuild):"
echo "     mypaas config set api-url http://localhost:8080"
echo "     mypaas config set token <your-jwt>"
echo "     mypaas project list"
echo "     mypaas project deploy <name>   # repeat per project"
echo ""
echo "  3. Or redeploy ALL projects at once:"
echo "     for name in \$(mypaas project list | tail -n +2 | awk '{print \$1}'); do"
echo "       echo \"Deploying \$name...\""
echo "       mypaas project deploy \"\$name\""
echo "       sleep 5"
echo "     done"
echo ""
echo "  4. If you created a new Cloudflare Tunnel:"
echo "     - Update CLOUDFLARE_TUNNEL_TOKEN in .env"
echo "     - docker compose -f docker-compose.prod.yml --env-file .env up -d cloudflared"
echo ""
echo "  5. Run scripts/verify-production.sh and verify project subdomains"
echo ""
