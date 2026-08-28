#!/usr/bin/env bash
set -euo pipefail

project_network="${PROJECT_NETWORK:-mypaas-ci-projects}"
control_network="${CONTROL_NETWORK:-mypaas-ci-control}"
routing_network="${ROUTING_NETWORK:-mypaas-ci-routing}"
container_name="mypaas-ci-runtime-smoke"
shared_name="mypaas-ci-shared-smoke"
admin_name="mypaas-ci-admin-smoke"
route_name="mypaas-ci-route-smoke"
caddy_name="mypaas-ci-caddy-smoke"
compose_project="mypaas-ci-compose-smoke"
tmpdir="$(mktemp -d)"

cleanup() {
  docker rm -f "$container_name" "$shared_name" "$admin_name" "$route_name" "$caddy_name" >/dev/null 2>&1 || true
  docker compose -p "$compose_project" -f "$tmpdir/compose.yml" down -v --remove-orphans >/dev/null 2>&1 || true
  docker network rm "$routing_network" >/dev/null 2>&1 || true
  docker network rm "$project_network" >/dev/null 2>&1 || true
  docker network rm "$control_network" >/dev/null 2>&1 || true
  rm -rf "$tmpdir"
}
trap cleanup EXIT

docker version >/dev/null

for network in "$project_network" "$control_network" "$routing_network"; do
  docker network create "$network" >/dev/null
done

# Runtime command contract used by MyPaaS and the bounded static builder.
docker run -d \
  --name "$container_name" \
  --network "$project_network" \
  --label mypaas.smoke=true \
  --memory 128m \
  --cpus 0.50 \
  --pids-limit 64 \
  --restart unless-stopped \
  alpine:3.20 sleep 300 >/dev/null

docker ps -q --filter label=mypaas.smoke=true | grep -q .
docker inspect "$container_name" >/dev/null
docker stop --time 5 "$container_name" >/dev/null
docker start "$container_name" >/dev/null
docker restart "$container_name" >/dev/null

# Shared PostgreSQL is intentionally dual-homed; ordinary control services are
# not reachable from the project or routing networks.
docker run -d \
  --name "$shared_name" \
  --network "$control_network" \
  alpine:3.20 sleep 300 >/dev/null
docker network connect --alias shared-db "$project_network" "$shared_name"

docker run -d \
  --name "$admin_name" \
  --network "$control_network" \
  --network-alias control-admin \
  alpine:3.20 sleep 300 >/dev/null

docker run --rm --network "$project_network" alpine:3.20 \
  ping -c 1 -W 1 shared-db >/dev/null
if docker run --rm --network "$project_network" alpine:3.20 \
  ping -c 1 -W 1 control-admin >/dev/null 2>&1; then
  echo "project network unexpectedly reached a control-only service" >&2
  exit 1
fi
if docker run --rm --network "$routing_network" alpine:3.20 \
  ping -c 1 -W 1 control-admin >/dev/null 2>&1; then
  echo "routing network unexpectedly reached a control-only service" >&2
  exit 1
fi

# Final dynamic routing model:
# - allocated host ports remain stable runtime lookup keys;
# - selected runtime is explicitly attached to ROUTING_NETWORK;
# - alias mypaas-port-<allocated-port> is the Caddy data-plane identity;
# - project workloads do not join the control network;
# - Caddy Admin remains Unix-only.
mkdir -p "$tmpdir/caddy-run"
docker run -d \
  --name "$route_name" \
  --network "$project_network" \
  -p "127.0.0.1:18080:80" \
  nginx:1.27-alpine >/dev/null

docker inspect "$route_name" | grep -q '"HostPort": "18080"'
docker network connect --alias mypaas-port-18080 "$routing_network" "$route_name"

alias_ready=false
for _ in $(seq 1 20); do
  if docker run --rm --network "$routing_network" alpine:3.20 \
    wget -qO- http://mypaas-port-18080:80 2>/dev/null | grep -q 'Welcome to nginx'; then
    alias_ready=true
    break
  fi
  sleep 0.25
done
if [[ "$alias_ready" != "true" ]]; then
  echo "routing network alias did not become HTTP-reachable" >&2
  docker inspect "$route_name" >&2 || true
  docker network inspect "$routing_network" >&2 || true
  exit 1
fi

cat > "$tmpdir/Caddyfile" <<'EOF'
{
  admin unix//run/mypaas/caddy-admin.sock
}
:18081 {
  reverse_proxy mypaas-port-18080:80
}
EOF

# Model production lifecycle exactly: attach Caddy to both networks before the
# process starts.
docker create \
  --name "$caddy_name" \
  --network "$control_network" \
  --network-alias caddy-edge \
  -v "$tmpdir/Caddyfile:/etc/caddy/Caddyfile:ro" \
  -v "$tmpdir/caddy-run:/run/mypaas" \
  caddy:2-alpine >/dev/null
docker network connect --alias caddy-edge "$routing_network" "$caddy_name"
docker start "$caddy_name" >/dev/null

caddy_ready=false
for _ in $(seq 1 30); do
  if [[ -S "$tmpdir/caddy-run/caddy-admin.sock" ]] && \
    docker run --rm --network "$control_network" alpine:3.20 \
      wget -qO- http://caddy-edge:18081 2>/dev/null | grep -q 'Welcome to nginx'; then
    caddy_ready=true
    break
  fi
  sleep 0.25
done
if [[ "$caddy_ready" != "true" ]]; then
  echo "dual-homed Caddy could not route through the explicit runtime alias" >&2
  docker logs "$caddy_name" >&2 || true
  exit 1
fi

if docker run --rm --network "$routing_network" alpine:3.20 \
  wget -qO- http://caddy-edge:2019 >/dev/null 2>&1; then
  echo "Caddy Admin unexpectedly accepted TCP :2019 from the routing network" >&2
  exit 1
fi

# Compose contract: image refresh must work through the same Docker Compose
# provider used by MyPaaS before the main service is started. This specifically
# guards the rootful Podman + `docker compose pull --ignore-buildable` path.
cat > "$tmpdir/compose.yml" <<EOF
services:
  app:
    image: nginx:1.27-alpine
    ports:
      - "127.0.0.1:18082:80"
    networks:
      - default
      - mypaas_platform
networks:
  mypaas_platform:
    external: true
    name: "$project_network"
EOF

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" pull --ignore-buildable >/dev/null
docker compose -p "$compose_project" -f "$tmpdir/compose.yml" up -d >/dev/null
compose_ids="$(docker ps -aq --filter "label=com.docker.compose.project=$compose_project")"
test -n "$compose_ids"
# shellcheck disable=SC2086
docker inspect $compose_ids >/dev/null
compose_id="$(printf '%s\n' "$compose_ids" | head -n1)"
docker inspect "$compose_id" | grep -q '"HostPort": "18082"'
docker network connect --alias mypaas-port-18082 "$routing_network" "$compose_id"

compose_alias_ready=false
for _ in $(seq 1 20); do
  if docker run --rm --network "$routing_network" alpine:3.20 \
    wget -qO- http://mypaas-port-18082:80 2>/dev/null | grep -q 'Welcome to nginx'; then
    compose_alias_ready=true
    break
  fi
  sleep 0.25
done
if [[ "$compose_alias_ready" != "true" ]]; then
  echo "Compose runtime alias did not become HTTP-reachable" >&2
  docker inspect "$compose_id" >&2 || true
  docker network inspect "$routing_network" >&2 || true
  exit 1
fi

docker compose -p "$compose_project" -f "$tmpdir/compose.yml" stop >/dev/null
docker compose -p "$compose_project" -f "$tmpdir/compose.yml" start >/dev/null

echo "Podman Docker-compatibility smoke passed"
