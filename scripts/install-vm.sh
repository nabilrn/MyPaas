#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
SKIP_DOCKER_INSTALL="${SKIP_DOCKER_INSTALL:-false}"
SKIP_DEPLOY="${SKIP_DEPLOY:-false}"
FORCE_ENV="${FORCE_ENV:-false}"
INSTALL_WIZARD="${INSTALL_WIZARD:-false}"
WIZARD_HOST="${WIZARD_HOST:-127.0.0.1}"
WIZARD_PORT="${WIZARD_PORT:-8787}"
WIZARD_PUBLIC_TUNNEL="${WIZARD_PUBLIC_TUNNEL:-true}"

cd "$ROOT_DIR"

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

sudo_cmd() {
  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

prompt_required() {
  local var_name="$1"
  local prompt="$2"
  local secret="${3:-false}"
  local value="${!var_name:-}"

  if [[ -n "$value" ]]; then
    printf '%s' "$value"
    return
  fi

  if [[ ! -t 0 ]]; then
    die "$var_name is required in non-interactive mode"
  fi

  while [[ -z "$value" ]]; do
    if [[ "$secret" == "true" ]]; then
      read -r -s -p "$prompt: " value
      printf '\n' >&2
    else
      read -r -p "$prompt: " value
    fi
  done

  printf '%s' "$value"
}

prompt_optional() {
  local var_name="$1"
  local prompt="$2"
  local default_value="$3"
  local value="${!var_name:-}"

  if [[ -n "$value" ]]; then
    printf '%s' "$value"
    return
  fi

  if [[ ! -t 0 ]]; then
    printf '%s' "$default_value"
    return
  fi

  read -r -p "$prompt [$default_value]: " value
  printf '%s' "${value:-$default_value}"
}

random_base64() {
  local bytes="${1:-32}"
  openssl rand -base64 "$bytes" | tr -d '\n'
}

random_hex() {
  local bytes="${1:-24}"
  openssl rand -hex "$bytes" | tr -d '\n'
}

ensure_docker_network() {
  local network_name="$1"
  local docker_cmd
  docker_cmd="$(docker_prefix)"
  $docker_cmd network inspect "$network_name" >/dev/null 2>&1 || $docker_cmd network create "$network_name" >/dev/null
}

docker_network_gateway() {
  local network_name="$1"
  local docker_cmd gateway
  docker_cmd="$(docker_prefix)"
  gateway="$($docker_cmd network inspect "$network_name" --format '{{(index .IPAM.Config 0).Gateway}}' 2>/dev/null || true)"
  printf '%s' "${gateway:-0.0.0.0}"
}

validate_url_safe_password() {
  local value="$1"
  if [[ "$value" =~ [^A-Za-z0-9._~-] ]]; then
    die "POSTGRES_PASSWORD contains characters that are unsafe for DATABASE_URL. Use A-Z, a-z, 0-9, '.', '_', '~', or '-'"
  fi
}

ensure_openssl() {
  if command_exists openssl; then
    return
  fi
  command_exists apt-get || die "openssl is required"
  sudo_cmd apt-get update
  sudo_cmd apt-get install -y openssl
}

ensure_python3() {
  if command_exists python3; then
    return
  fi
  command_exists apt-get || die "python3 is required for INSTALL_WIZARD=true"
  sudo_cmd apt-get update
  sudo_cmd apt-get install -y python3
}

install_docker_debian() {
  if ! command_exists curl || ! command_exists gpg; then
    sudo_cmd apt-get update
    sudo_cmd apt-get install -y ca-certificates curl gnupg
  fi

  # shellcheck disable=SC1091
  source /etc/os-release
  local distro="${ID:-}"
  local codename="${VERSION_CODENAME:-}"

  if [[ "$distro" != "ubuntu" && "$distro" != "debian" ]]; then
    die "automatic Docker install only supports Ubuntu/Debian. Install Docker manually, then rerun with SKIP_DOCKER_INSTALL=true"
  fi
  if [[ -z "$codename" ]]; then
    die "could not detect OS codename for Docker apt repository"
  fi

  sudo_cmd install -m 0755 -d /etc/apt/keyrings
  sudo_cmd rm -f /etc/apt/keyrings/docker.gpg
  curl -fsSL "https://download.docker.com/linux/$distro/gpg" | sudo_cmd gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo_cmd chmod a+r /etc/apt/keyrings/docker.gpg

  local arch
  arch="$(dpkg --print-architecture)"
  printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/%s %s stable\n' "$arch" "$distro" "$codename" \
    | sudo_cmd tee /etc/apt/sources.list.d/docker.list >/dev/null

  sudo_cmd apt-get update
  sudo_cmd apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin git openssl

  if command_exists systemctl; then
    sudo_cmd systemctl enable --now docker >/dev/null 2>&1 || true
  fi
}

ensure_dependencies() {
  [[ "$(uname -s)" == "Linux" ]] || die "install-vm.sh must run on a Linux VM"
  ensure_openssl

  if ! command_exists docker || ! docker compose version >/dev/null 2>&1; then
    if [[ "$SKIP_DOCKER_INSTALL" == "true" ]]; then
      die "Docker with Compose plugin is required"
    fi

    log "Installing Docker Engine and Compose plugin"
    command_exists apt-get || die "automatic dependency install requires apt-get. Install Docker manually, then rerun with SKIP_DOCKER_INSTALL=true"
    install_docker_debian
  fi
}

docker_prefix() {
  if docker ps >/dev/null 2>&1; then
    printf 'docker'
    return
  fi
  if command_exists sudo && sudo docker ps >/dev/null 2>&1; then
    printf 'sudo docker'
    return
  fi
  die "current user cannot access Docker. Add the user to the docker group or run with sudo"
}

run_install_wizard() {
  ensure_python3

  local public_domain owner_email github_client_id github_client_secret callback_url cloudflare_tunnel_token
  local postgres_user postgres_db postgres_password jwt_secret encryption_key project_network docker_bind_host metrics_password
  local wizard_token

  public_domain="${PUBLIC_DOMAIN:-}"
  owner_email="${OWNER_EMAIL:-}"
  github_client_id="${GITHUB_CLIENT_ID:-}"
  github_client_secret="${GITHUB_CLIENT_SECRET:-}"
  callback_url="${GITHUB_CALLBACK_URL:-}"
  cloudflare_tunnel_token="${CLOUDFLARE_TUNNEL_TOKEN:-}"
  postgres_user="${POSTGRES_USER:-mypaas}"
  postgres_db="${POSTGRES_DB:-mypaas}"
  postgres_password="${POSTGRES_PASSWORD:-$(random_hex 24)}"
  validate_url_safe_password "$postgres_password"
  jwt_secret="${JWT_SECRET:-$(random_base64 32)}"
  encryption_key="${ENCRYPTION_KEY:-$(random_base64 32)}"
  metrics_password="${METRICS_PASSWORD:-$(random_hex 18)}"
  project_network="${PROJECT_NETWORK:-mypaas-prod}"
  ensure_docker_network "$project_network"
  docker_bind_host="${DOCKER_BIND_HOST:-$(docker_network_gateway "$project_network")}"
  wizard_token="${WIZARD_TOKEN:-$(random_hex 16)}"

  log "Starting install wizard"

  WIZARD_ENV_FILE="$ENV_FILE" \
  WIZARD_TOKEN="$wizard_token" \
  WIZARD_HOST="$WIZARD_HOST" \
  WIZARD_PORT="$WIZARD_PORT" \
  WIZARD_DEFAULT_PUBLIC_DOMAIN="$public_domain" \
  WIZARD_DEFAULT_OWNER_EMAIL="$owner_email" \
  WIZARD_DEFAULT_GITHUB_CLIENT_ID="$github_client_id" \
  WIZARD_DEFAULT_GITHUB_CLIENT_SECRET="$github_client_secret" \
  WIZARD_DEFAULT_GITHUB_CALLBACK_URL="$callback_url" \
  WIZARD_DEFAULT_CLOUDFLARE_TUNNEL_TOKEN="$cloudflare_tunnel_token" \
  WIZARD_DEFAULT_POSTGRES_USER="$postgres_user" \
  WIZARD_DEFAULT_POSTGRES_DB="$postgres_db" \
  WIZARD_DEFAULT_POSTGRES_PASSWORD="$postgres_password" \
  WIZARD_DEFAULT_JWT_SECRET="$jwt_secret" \
  WIZARD_DEFAULT_ENCRYPTION_KEY="$encryption_key" \
  WIZARD_DEFAULT_METRICS_PASSWORD="$metrics_password" \
  WIZARD_DEFAULT_PROJECT_NETWORK="$project_network" \
  WIZARD_DEFAULT_DOCKER_BIND_HOST="$docker_bind_host" \
  WIZARD_SCRIPT="$ROOT_DIR/scripts/install-wizard.py" \
  WIZARD_PUBLIC_TUNNEL="$WIZARD_PUBLIC_TUNNEL" \
  bash "$ROOT_DIR/scripts/run-install-wizard.sh"
}

write_env_file() {
  if [[ -f "$ENV_FILE" && "$FORCE_ENV" != "true" ]]; then
    log "Using existing $ENV_FILE"
    return
  fi

  if [[ "$INSTALL_WIZARD" == "true" ]]; then
    run_install_wizard
    return
  fi

  log "Generating production .env"

  local public_domain owner_email github_client_id github_client_secret callback_url cloudflare_tunnel_token
  local postgres_user postgres_db postgres_password jwt_secret encryption_key project_network
  local docker_bind_host

  public_domain="$(prompt_required PUBLIC_DOMAIN "Public dashboard domain, e.g. mypaas.example.com")"
  owner_email="$(prompt_required OWNER_EMAIL "Owner GitHub primary email")"
  github_client_id="$(prompt_required GITHUB_CLIENT_ID "GitHub OAuth Client ID")"
  github_client_secret="$(prompt_required GITHUB_CLIENT_SECRET "GitHub OAuth Client Secret")"
  callback_url="$(prompt_optional GITHUB_CALLBACK_URL "GitHub OAuth callback URL" "https://$public_domain/api/auth/github/callback")"
  cloudflare_tunnel_token="$(prompt_required CLOUDFLARE_TUNNEL_TOKEN "Cloudflare Tunnel token")"

  postgres_user="$(prompt_optional POSTGRES_USER "Postgres user" "mypaas")"
  postgres_db="$(prompt_optional POSTGRES_DB "Postgres database" "mypaas")"
  postgres_password="${POSTGRES_PASSWORD:-$(random_hex 24)}"
  validate_url_safe_password "$postgres_password"
  jwt_secret="${JWT_SECRET:-$(random_base64 32)}"
  encryption_key="${ENCRYPTION_KEY:-$(random_base64 32)}"
  project_network="${PROJECT_NETWORK:-mypaas-prod}"
  ensure_docker_network "$project_network"
  docker_bind_host="${DOCKER_BIND_HOST:-$(docker_network_gateway "$project_network")}"

  umask 077
  cat > "$ENV_FILE" <<EOF
ENVIRONMENT=production

POSTGRES_USER=$postgres_user
POSTGRES_PASSWORD=$postgres_password
POSTGRES_DB=$postgres_db

API_HOST=127.0.0.1
API_PORT=8080
FRONTEND_URL=https://$public_domain
PUBLIC_DOMAIN=$public_domain
OWNER_EMAIL=$owner_email

GITHUB_CLIENT_ID=$github_client_id
GITHUB_CLIENT_SECRET=$github_client_secret
GITHUB_CALLBACK_URL=$callback_url

CLOUDFLARE_TUNNEL_TOKEN=$cloudflare_tunnel_token

JWT_SECRET=$jwt_secret
ENCRYPTION_KEY=$encryption_key

DOCKER_SOCKET=/var/run/docker.sock
DOCKER_HOST=
DOCKER_BIND_HOST=$docker_bind_host
PROJECT_NETWORK=$project_network

USER_RAM_QUOTA_GB=${USER_RAM_QUOTA_GB:-6}
USER_CPU_QUOTA=${USER_CPU_QUOTA:-3}
MAX_PROJECTS=${MAX_PROJECTS:-20}
PROJECT_DEFAULT_RAM_MB=${PROJECT_DEFAULT_RAM_MB:-512}
PROJECT_DEFAULT_CPU=${PROJECT_DEFAULT_CPU:-0.5}

ENABLE_WEBHOOKS=true
ENABLE_METRICS=true
METRICS_USERNAME=${METRICS_USERNAME:-mypaas}
METRICS_PASSWORD=${METRICS_PASSWORD:-$(random_hex 18)}
MAX_CONCURRENT_DEPLOYS=${MAX_CONCURRENT_DEPLOYS:-2}
BUILD_TIMEOUT_MINUTES=${BUILD_TIMEOUT_MINUTES:-30}

SHARED_POSTGRES_ENABLED=${SHARED_POSTGRES_ENABLED:-true}
SHARED_POSTGRES_HOST=postgres
SHARED_POSTGRES_PORT=5432
SHARED_POSTGRES_SSLMODE=disable

BACKUP_ENABLED=${BACKUP_ENABLED:-true}
BACKUP_DIR=/var/lib/mypaas/backups
BACKUP_DAILY_AT=${BACKUP_DAILY_AT:-02:00}
BACKUP_TIMEOUT_MINUTES=${BACKUP_TIMEOUT_MINUTES:-30}
BACKUP_KEEP_DAILY=${BACKUP_KEEP_DAILY:-7}
BACKUP_KEEP_WEEKLY=${BACKUP_KEEP_WEEKLY:-4}
BACKUP_WEEKLY_DAY=${BACKUP_WEEKLY_DAY:-sunday}

IMAGE_CLEANUP_ENABLED=${IMAGE_CLEANUP_ENABLED:-true}
IMAGE_CLEANUP_UNTIL=${IMAGE_CLEANUP_UNTIL:-168h}
IMAGE_CLEANUP_WEEKDAY=${IMAGE_CLEANUP_WEEKDAY:-sunday}

LOG_LEVEL=info
LOG_FORMAT=json

CADDY_ADMIN=127.0.0.1:2019
CADDY_UPSTREAM_HOST=$docker_bind_host
STATIC_ROOT=/var/lib/mypaas/static
CADDY_STATIC_ROOT=/var/lib/mypaas/static
CADDY_METRICS=true
EOF
}

prepare_host() {
  log "Preparing host directories"
  for dir in \
    /var/lib/mypaas/volumes \
    /var/lib/mypaas/compose \
    /var/lib/mypaas/static \
    /var/lib/mypaas/backups \
    /tmp/mypaas/builds
  do
    sudo_cmd mkdir -p "$dir"
  done

  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
  ensure_docker_network "${PROJECT_NETWORK:-mypaas-prod}"
}

main() {
  ensure_dependencies
  write_env_file
  prepare_host

  if [[ "$SKIP_DEPLOY" == "true" ]]; then
    log "Install preparation complete. Skipping deploy because SKIP_DEPLOY=true"
    return
  fi

  local docker_cmd
  docker_cmd="$(docker_prefix)"
  log "Starting MyPaas production stack"
  DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" COMPOSE_FILE="$COMPOSE_FILE" ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh"

  log "Install complete"
  printf 'Dashboard: https://%s\n' "$(grep -E '^PUBLIC_DOMAIN=' "$ENV_FILE" | cut -d= -f2-)"
  printf 'Run verification: ENV_FILE=%q bash scripts/verify-production.sh\n' "$ENV_FILE"
}

main "$@"
