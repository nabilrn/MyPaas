#!/usr/bin/env bash
set -euo pipefail

cleanup_advice() {
  printf '\n=================================================================\n' >&2
  printf '❌ INSTALLATION FAILED!\n' >&2
  printf 'To clean up the failed installation and start fresh, please run:\n' >&2
  printf '   bash scripts/uninstall-vm.sh\n' >&2
  printf '=================================================================\n' >&2
}
trap cleanup_advice ERR

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

USE_PODMAN="${USE_PODMAN:-false}"
MIGRATE_URL="${MIGRATE_URL:-}"
INSTALL_STATD="${INSTALL_STATD:-true}"
STATD_ONLY="${STATD_ONLY:-false}"
STATD_INSTALL_MODE="${STATD_INSTALL_MODE:-release}"
STATD_VERSION="${STATD_VERSION:-v0.2.0}"
STATD_RELEASE_BASE_URL="${STATD_RELEASE_BASE_URL:-https://github.com/nabilrn/mypaas-statd/releases/download}"
STATD_REPO_URL="${STATD_REPO_URL:-https://github.com/nabilrn/mypaas-statd.git}"
STATD_REF="${STATD_REF:-main}"
STATD_DIR="${STATD_DIR:-/opt/mypaas-statd}"

while [[ $# -gt 0 ]]; do
  case $1 in
    --migrate-url)
      MIGRATE_URL="$2"
      shift 2
      ;;
    --podman)
      USE_PODMAN="true"
      shift
      ;;
    --statd-only)
      STATD_ONLY="true"
      shift
      ;;
    *)
      break
      ;;
  esac
done

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

validate_network_names() {
  local control_network="$1"
  local project_network="$2"
  local routing_network="$3"
  if [[ "$control_network" == "$project_network" || "$control_network" == "$routing_network" || "$project_network" == "$routing_network" ]]; then
    die "CONTROL_NETWORK, PROJECT_NETWORK, and ROUTING_NETWORK must be distinct"
  fi
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

ensure_nixpacks() {
  if command_exists nixpacks; then
    return
  fi
  log "Installing Nixpacks CLI..."
  if ! command_exists curl; then
    command_exists apt-get || die "curl is required to install nixpacks"
    sudo_cmd apt-get update
    sudo_cmd apt-get install -y curl
  fi
  curl -sSL https://nixpacks.com/install.sh | sudo_cmd bash
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
  printf 'deb [arch=%s signed-by=%s] https://download.docker.com/linux/%s %s stable\n' "$arch" /etc/apt/keyrings/docker.gpg "$distro" "$codename" \
    | sudo_cmd tee /etc/apt/sources.list.d/docker.list >/dev/null

  sudo_cmd apt-get update
  if [[ "$USE_PODMAN" == "true" ]]; then
    sudo_cmd apt-get install -y podman catatonit docker-ce-cli docker-compose-plugin git openssl
    if command_exists systemctl; then
      sudo_cmd systemctl enable --now podman.socket >/dev/null 2>&1 || true
    fi
    sudo_cmd ln -sf /run/podman/podman.sock /var/run/docker.sock || true
  else
    sudo_cmd apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin git openssl
    if command_exists systemctl; then
      sudo_cmd systemctl enable --now docker >/dev/null 2>&1 || true
    fi
  fi
}

ensure_dependencies() {
  [[ "$(uname -s)" == "Linux" ]] || die "install-vm.sh must run on a Linux VM"
  ensure_openssl
  ensure_nixpacks

  if [[ "$USE_PODMAN" == "true" ]]; then
    if ! command_exists podman || ! docker compose version >/dev/null 2>&1 || ! command_exists catatonit; then
      log "Installing Podman Engine and Compose plugin"
      command_exists apt-get || die "automatic dependency install requires apt-get."
      install_docker_debian
    fi
  else
    if ! command_exists docker || ! docker compose version >/dev/null 2>&1; then
      if [[ "$SKIP_DOCKER_INSTALL" == "true" ]]; then
        die "Docker with Compose plugin is required"
      fi

      log "Installing Docker Engine and Compose plugin"
      command_exists apt-get || die "automatic dependency install requires apt-get. Install Docker manually, then rerun with SKIP_DOCKER_INSTALL=true"
      install_docker_debian
    fi
  fi
}

install_statd_release() {
  local machine arch artifact release_url checksum_url
  machine="$(uname -m)"
  case "$machine" in
    x86_64|amd64)
      arch="amd64"
      ;;
    *)
      die "mypaas-statd $STATD_VERSION release artifacts currently support linux-amd64 only; set STATD_INSTALL_MODE=source for an explicit source build"
      ;;
  esac

  if command_exists apt-get; then
    sudo_cmd apt-get update
    sudo_cmd apt-get install -y ca-certificates curl tar coreutils
  else
    command_exists curl || die "curl is required to install the mypaas-statd release"
    command_exists tar || die "tar is required to install the mypaas-statd release"
    command_exists sha256sum || die "sha256sum is required to verify the mypaas-statd release"
  fi

  artifact="mypaas-statd-linux-${arch}.tar.gz"
  release_url="${STATD_RELEASE_BASE_URL%/}/${STATD_VERSION}/${artifact}"
  checksum_url="${STATD_RELEASE_BASE_URL%/}/${STATD_VERSION}/SHA256SUMS"
  log "Installing verified mypaas-statd release $STATD_VERSION"

  (
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT
    curl -fL "$release_url" -o "$tmpdir/$artifact"
    curl -fL "$checksum_url" -o "$tmpdir/SHA256SUMS"
    grep -E "^[0-9a-fA-F]{64}  ${artifact}$" "$tmpdir/SHA256SUMS" > "$tmpdir/SHA256SUMS.selected" \
      || die "SHA256SUMS does not contain a checksum for $artifact"
    (cd "$tmpdir" && sha256sum -c SHA256SUMS.selected >/dev/null)
    tar -C "$tmpdir" -xzf "$tmpdir/$artifact"
    [[ "$(cat "$tmpdir/VERSION")" == "$STATD_VERSION" ]] || die "mypaas-statd release VERSION does not match $STATD_VERSION"
    [[ "$("$tmpdir/mypaas-statd" --version)" == "mypaas-statd ${STATD_VERSION#v}" ]] \
      || die "mypaas-statd release binary version does not match $STATD_VERSION"
    sudo_cmd install -Dm0755 "$tmpdir/mypaas-statd" /usr/local/bin/mypaas-statd
    sudo_cmd install -Dm0644 "$tmpdir/mypaas-statd.service" /etc/systemd/system/mypaas-statd.service
  )
}

install_statd_source() {
  if command_exists apt-get; then
    sudo_cmd apt-get update
    sudo_cmd apt-get install -y git make gcc libc6-dev
  else
    command_exists git || die "git is required to install mypaas-statd from source"
    command_exists make || die "make is required to install mypaas-statd from source"
    command_exists cc || command_exists gcc || die "a C compiler is required to install mypaas-statd from source"
  fi

  log "Installing mypaas-statd from $STATD_REPO_URL ($STATD_REF)"
  if [[ -d "$STATD_DIR/.git" ]]; then
    sudo_cmd git -C "$STATD_DIR" fetch --prune --tags origin
  elif [[ -e "$STATD_DIR" ]]; then
    die "$STATD_DIR exists but is not a git checkout"
  else
    sudo_cmd git clone "$STATD_REPO_URL" "$STATD_DIR"
    sudo_cmd git -C "$STATD_DIR" fetch --prune --tags origin
  fi

  if sudo_cmd git -C "$STATD_DIR" rev-parse --verify --quiet "origin/$STATD_REF^{commit}" >/dev/null; then
    sudo_cmd git -C "$STATD_DIR" checkout --detach "origin/$STATD_REF"
  elif sudo_cmd git -C "$STATD_DIR" rev-parse --verify --quiet "$STATD_REF^{commit}" >/dev/null; then
    sudo_cmd git -C "$STATD_DIR" checkout --detach "$STATD_REF"
  else
    die "STATD_REF does not resolve to a fetched branch, tag, or commit: $STATD_REF"
  fi
  sudo_cmd make -C "$STATD_DIR" install PREFIX=/usr/local SYSTEMD_UNIT_DIR=/etc/systemd/system
}

install_statd() {
  if [[ "$INSTALL_STATD" != "true" ]]; then
    log "Skipping mypaas-statd install because INSTALL_STATD=false"
    return
  fi

  command_exists systemctl || die "mypaas-statd install requires systemd. Set INSTALL_STATD=false to skip."
  [[ -f /sys/fs/cgroup/cgroup.controllers ]] || die "mypaas-statd requires cgroup v2. Set INSTALL_STATD=false to skip."

  case "$STATD_INSTALL_MODE" in
    release)
      install_statd_release
      ;;
    source)
      install_statd_source
      ;;
    *)
      die "STATD_INSTALL_MODE must be release or source"
      ;;
  esac

  sudo_cmd systemctl daemon-reload
  sudo_cmd systemctl enable mypaas-statd >/dev/null
  sudo_cmd systemctl restart mypaas-statd
  for _ in {1..20}; do
    [[ -S /run/mypaas/statd.sock ]] && break
    sleep 0.1
  done
  [[ -S /run/mypaas/statd.sock ]] || die "mypaas-statd started but /run/mypaas/statd.sock was not created"
  if [[ "$STATD_INSTALL_MODE" == "release" ]]; then
    [[ "$(sudo_cmd /usr/local/bin/mypaas-statd --version)" == "mypaas-statd ${STATD_VERSION#v}" ]] \
      || die "installed mypaas-statd version does not match $STATD_VERSION"
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
  local postgres_user postgres_db postgres_password jwt_secret encryption_key control_network project_network routing_network docker_bind_host metrics_password
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
  control_network="${CONTROL_NETWORK:-mypaas-control}"
  project_network="${PROJECT_NETWORK:-mypaas-projects}"
  routing_network="${ROUTING_NETWORK:-mypaas-routing}"
  validate_network_names "$control_network" "$project_network" "$routing_network"
  ensure_docker_network "$control_network"
  ensure_docker_network "$project_network"
  ensure_docker_network "$routing_network"
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

  if ! grep -q '^CONTROL_NETWORK=' "$ENV_FILE"; then
    printf '\nCONTROL_NETWORK=%s\n' "$control_network" >> "$ENV_FILE"
  fi
  if grep -q '^ROUTING_NETWORK=' "$ENV_FILE"; then
    sed -i "s#^ROUTING_NETWORK=.*#ROUTING_NETWORK=$routing_network#" "$ENV_FILE"
  else
    printf 'ROUTING_NETWORK=%s\n' "$routing_network" >> "$ENV_FILE"
  fi
  if grep -q '^CADDY_ADMIN=' "$ENV_FILE"; then
    sed -i 's#^CADDY_ADMIN=.*#CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock#' "$ENV_FILE"
  else
    printf 'CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock\n' >> "$ENV_FILE"
  fi
  if grep -q '^CADDY_UPSTREAM_HOST=' "$ENV_FILE"; then
    sed -i 's#^CADDY_UPSTREAM_HOST=.*#CADDY_UPSTREAM_HOST=runtime#' "$ENV_FILE"
  else
    printf 'CADDY_UPSTREAM_HOST=runtime\n' >> "$ENV_FILE"
  fi
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
  local postgres_user postgres_db postgres_password jwt_secret encryption_key control_network project_network routing_network
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
  control_network="${CONTROL_NETWORK:-mypaas-control}"
  project_network="${PROJECT_NETWORK:-mypaas-projects}"
  routing_network="${ROUTING_NETWORK:-mypaas-routing}"
  validate_network_names "$control_network" "$project_network" "$routing_network"
  ensure_docker_network "$control_network"
  ensure_docker_network "$project_network"
  ensure_docker_network "$routing_network"
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
CONTROL_NETWORK=$control_network
PROJECT_NETWORK=$project_network
ROUTING_NETWORK=$routing_network

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

AUTO_UPDATE_ENABLED=${AUTO_UPDATE_ENABLED:-false}
AUTO_UPDATE_INTERVAL_MINUTES=${AUTO_UPDATE_INTERVAL_MINUTES:-30}
AUTO_UPDATE_REF=${AUTO_UPDATE_REF:-main}
AUTO_UPDATE_IMAGE_WAIT_SECONDS=${AUTO_UPDATE_IMAGE_WAIT_SECONDS:-300}

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

CADDY_ADMIN=unix//run/mypaas/caddy-admin.sock
CADDY_UPSTREAM_HOST=runtime
STATIC_ROOT=/var/lib/mypaas/static
CADDY_STATIC_ROOT=/var/lib/mypaas/static
CADDY_METRICS=true
STATD_SOCKET=/run/mypaas/statd.sock
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

  local control_network project_network routing_network
  control_network="${CONTROL_NETWORK:-mypaas-control}"
  project_network="${PROJECT_NETWORK:-mypaas-projects}"
  routing_network="${ROUTING_NETWORK:-mypaas-routing}"
  validate_network_names "$control_network" "$project_network" "$routing_network"
  ensure_docker_network "$control_network"
  ensure_docker_network "$project_network"
  ensure_docker_network "$routing_network"
  install_statd
}

main() {
  if [[ "$STATD_ONLY" == "true" ]]; then
    [[ "$(uname -s)" == "Linux" ]] || die "mypaas-statd install requires Linux"
    install_statd
    log "mypaas-statd $STATD_VERSION installation complete"
    return
  fi

  ensure_dependencies

  if [[ -n "$MIGRATE_URL" ]]; then
    log "Downloading migration package..."
    if ! command_exists curl; then
      sudo_cmd apt-get update && sudo_cmd apt-get install -y curl
    fi
    curl -fL "$MIGRATE_URL" -o /tmp/mypaas-export.tar.gz

    log "Running import script..."
    bash "$ROOT_DIR/scripts/migrate-import.sh" /tmp/mypaas-export.tar.gz

    log "Migration import complete! Starting MyPaas..."
    prepare_host
    local docker_cmd
    docker_cmd="$(docker_prefix)"
    DOCKER_BIN="$docker_cmd" COMPOSE_BIN="$docker_cmd compose" COMPOSE_FILE="$COMPOSE_FILE" ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/deploy-to-vm.sh"
    ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/configure-auto-update.sh"

    log "Migration successfully deployed on new VM!"
    exit 0
  fi

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
  ENV_FILE="$ENV_FILE" bash "$ROOT_DIR/scripts/configure-auto-update.sh"

  log "Install complete"
  printf 'Dashboard: https://%s\n' "$(grep -E '^PUBLIC_DOMAIN=' "$ENV_FILE" | cut -d= -f2-)"
  printf 'Run verification: ENV_FILE=%q bash scripts/verify-production.sh\n' "$ENV_FILE"
}

main "$@"
