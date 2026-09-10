# Production Verification

`scripts/verify-production.sh` checks the MyPaaS control-plane/runtime invariants that can be verified locally on an installed VM.

Run it after installation, a manual recovery, or a host/runtime change that could affect the production contract.

## Run the verifier

From the installer-managed checkout:

```bash
cd ~/MyPaas
ENV_FILE=.env bash scripts/verify-production.sh
```

The Makefile wrapper is also available when the default paths apply:

```bash
make verify-prod
```

A successful run ends with:

```text
Production verification passed.
```

## What it checks

The current verifier checks:

- production Compose service state;
- `CONTROL_NETWORK`, `PROJECT_NETWORK`, and `ROUTING_NETWORK` separation;
- control-plane container membership/forbidden memberships;
- Cloudflare Tunnel container running state;
- `mypaas-statd` service/socket visibility when `STATD_SOCKET` is configured;
- API `/health` and `/ready` through the control network;
- `/metrics` authentication/availability when metrics are enabled;
- API ingress through local Caddy;
- dashboard reachability;
- dashboard HTML/cache behavior and referenced immutable SvelteKit assets;
- a concrete running API build SHA;
- optional exact expected build/image identity when supplied;
- Caddy Admin Unix-socket availability and the absence of public TCP port `2019`;
- an existing project route when one is discoverable;
- the bundled CLI inside the API image.

## Project-route behavior

By default, an unavailable existing project route is reported but does not fail control-plane verification. This prevents an unhealthy user workload from automatically being classified as a broken MyPaaS control plane.

To require an existing project route to be healthy:

```bash
REQUIRE_PROJECT_ROUTE=true \
ENV_FILE=.env \
bash scripts/verify-production.sh
```

Use this only when the verification scenario intentionally requires a healthy project workload.

## Exact release identity

Update workflows can provide expected build/image identity to the verifier. For manual investigation, inspect the running release before asserting that a VM is running a particular Git SHA.

Do not call an installation “exact-SHA qualified” merely because `/health` returns successfully.

## Optional backup check

The verifier does not trigger a backup by default. To include the bundled CLI backup operation:

```bash
RUN_BACKUP=true \
ENV_FILE=.env \
bash scripts/verify-production.sh
```

This creates backup output and therefore is not a purely read-only verification run.

## What it does not prove

A verifier pass does **not** establish:

- universal RPS or concurrent-user capacity;
- a maximum project count for the host;
- correctness of every deployed application;
- application-specific database consistency;
- external provider availability beyond the paths actually checked;
- browser UX/cross-browser behavior;
- migration correctness for data not covered by the migration package;
- hostile multi-tenant isolation.

Capacity and application behavior remain workload-specific.

## Troubleshooting

If verification fails, keep the first failing invariant and its surrounding logs. Useful host evidence can include:

```bash
docker compose -f docker-compose.prod.yml --env-file .env ps
journalctl -u mypaas-update.service
systemctl status mypaas-statd
```

On a Podman-first installation, the repository intentionally uses the Docker-compatible command surface. Do not replace documented commands with unrelated engine-local cleanup operations while diagnosing a running production host.

Never paste `.env`, credentials, tokens, cookies, decrypted project environment values, or database contents into a public issue.

## Related documents

- [Runtime verification evidence](../engineering/runtime-verification.md)
- [Architecture](../ARCHITECTURE.md)
- [Updates](update.md)
- [Backup and restore](backup-restore.md)
