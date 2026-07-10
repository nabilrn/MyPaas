# MyPaas — Self-Hosted Personal Deployment Platform

> Deploy your projects like Vercel/Railway, but on your own infrastructure.

MyPaas is a lightweight, self-hosted platform for deploying multi-service applications with automatic SSL, realtime logs, and a clean dashboard. Perfect for solo developers or small teams.

**Key features:**
- **Auto-deploy** on git push via webhooks
- **Docker + Compose** support — single container or multi-service
- **SSL with Cloudflare** — no certificate management
- **Realtime dashboard** — logs, metrics, deployment history
- **Instant rollback** — go back to any previous deployment
- **GitHub OAuth** — whitelisted collaborators only
- **VM install script** — prepare a fresh Linux VM and start MyPaas with one command

---

## Quick Start

### Production VM install
For a fresh Ubuntu/Debian VM, run the public bootstrap installer:
```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

The bootstrap installs Git when needed, checks out `main` into `~/MyPaas`, and starts the browser setup wizard. The installer prints a temporary HTTPS `*.trycloudflare.com` URL and removes it automatically after setup, so no inbound firewall rule or SSH port forwarding is required. It can be rerun safely when the checkout is clean.

Use the browser wizard when you want a guided setup for GitHub OAuth, Cloudflare DNS, Cloudflare Tunnel, owner email, and production secrets. If the temporary URL cannot be created, the installer prints this SSH fallback:
```bash
ssh -L 8787:127.0.0.1:8787 <user>@<vm-ip>
```

For non-interactive setup without the browser wizard, provide all required credentials:
```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env \
  INSTALL_WIZARD=false \
  PUBLIC_DOMAIN=mypaas.example.com \
  OWNER_EMAIL=you@example.com \
  GITHUB_CLIENT_ID=your_client_id \
  GITHUB_CLIENT_SECRET=your_client_secret \
  CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token \
  bash
```

The installer checks Docker + Compose, installs Docker on Ubuntu/Debian when needed, generates `.env`, prepares `/var/lib/mypaas`, runs migrations, and starts `docker-compose.prod.yml`. See [Deployment](#deployment) for non-interactive env flags and verification commands.

### Local development prerequisites
- **Go 1.22+** (backend)
- **Node.js 18+ & pnpm** (frontend)
- **PostgreSQL 16**
- **Docker + Docker Compose** (plugin version)
- **Caddy 2** (reverse proxy)

### 1. Clone and setup environment
```bash
git clone <your-repo-url> mypaas
cd mypaas
cp .env.example .env
# Edit .env with your config
```

### 2. Start infrastructure
```bash
make docker-up      # PostgreSQL + Redis in Docker
make migrate-up     # Run database migrations
```

### 3. Backend development
```bash
cd backend
air                 # Live reload server (http://localhost:8080)
```

In another terminal:
```bash
make test-backend   # Run tests
make lint-backend   # Run linter
```

### 4. Frontend development
```bash
cd frontend
pnpm install
pnpm dev            # Dev server (http://localhost:5173)
```

### 5. Build for production
```bash
make build          # Build backend binary + frontend
docker compose -f docker-compose.prod.yml up -d
```

---

## Project Structure

```
mypaas/
├── backend/              Go API server
│   ├── cmd/              Entry points (api, cli)
│   ├── internal/         Business logic (auth, deployment, container, etc.)
│   ├── query/            SQL queries (for sqlc)
│   └── migrations/       Database schema
├── frontend/             SvelteKit dashboard
│   └── src/routes/       Pages (projects, deployments, settings, etc.)
├── docs/                 Documentation & architecture
├── docker-compose.*yml   Dev & prod infrastructure
├── Caddyfile.*           Reverse proxy config
└── Makefile              Build targets
```

See `CLAUDE.md` for detailed structure.

---

## Make Targets

```bash
make dev            # Start dev environment (deps + auto-reload)
make test           # Run all tests
make lint           # Lint code
make build          # Build binaries
make migrate-up     # Run migrations
make sqlc           # Generate database code
make clean          # Remove build artifacts
make help           # Show all targets
```

---

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Core
DATABASE_URL=postgres://user:pass@localhost:5432/mypaas_dev
ENVIRONMENT=development

# GitHub OAuth
GITHUB_CLIENT_ID=your_id
GITHUB_CLIENT_SECRET=your_secret

# Cloudflare Tunnel
CLOUDFLARE_TUNNEL_TOKEN=your_token
CLOUDFLARE_ACCOUNT_ID=your_account_id

# JWT
JWT_SECRET=your_256bit_secret_base64_encoded

# Docker
DOCKER_SOCKET=/var/run/docker.sock
```

---

## Documentation

- **[PRD](docs/PRD.md)** — Product requirements & scope
- **[Architecture](docs/ARCHITECTURE.md)** — Technical design & diagrams
- **[Timeline](docs/TIMELINE.md)** — 15-day implementation plan
- **[ADRs](docs/adr/)** — Architecture decision records
- **[CLAUDE.md](CLAUDE.md)** — Codebase conventions & guidelines

---

## Development Workflow

1. **Branch:** `git checkout -b feature/foo`
2. **Code:** Follow conventions in [CLAUDE.md](CLAUDE.md)
3. **Test:** `make test`
4. **Lint:** `make lint`
5. **Commit:** Reference issue in message (e.g., `fix: prevent issue #42`)
6. **Push & PR:** Link to issue, describe changes
7. **Review:** Ensure tests pass + code review approval
8. **Merge:** Squash if cleanup commits, otherwise rebase

---

## Common Issues

**Port already in use?**
```bash
# Change in .env
API_PORT=8081
```

**Database won't migrate?**
```bash
# Reset and retry
make docker-reset
make migrate-up
```

**Frontend won't start?**
```bash
cd frontend
rm -rf node_modules pnpm-lock.yaml
pnpm install
pnpm dev
```

---

## Deployment

For a fresh Linux VM, bootstrap the repository and browser wizard in one command:
```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

For an existing checkout, run the installer directly:
```bash
INSTALL_WIZARD=true bash scripts/install-vm.sh
```

The bootstrap defaults to branch `main` and `~/MyPaas`. To pin another branch/tag or directory:
```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env MYPAAS_REF=<tag> MYPAAS_INSTALL_DIR=<path> bash
```

The wizard remains bound to `127.0.0.1` on the VM. During setup, the installer starts an ephemeral Cloudflare Quick Tunnel and prints a token-protected HTTPS URL. The tunnel is removed as soon as the wizard saves `.env`. If Quick Tunnel startup fails, use the printed SSH fallback:
```bash
ssh -L 8787:127.0.0.1:8787 <user>@<vm-ip>
```

The step-by-step wizard explains the public domain/subdomain model, Cloudflare nameserver/DNS requirements, GitHub OAuth setup, Cloudflare Tunnel token and public hostname routes, writes the production `.env`, shuts down, and lets the installer continue.

Cloudflare requirements before the dashboard can resolve:
- The MyPaas domain must be active in Cloudflare DNS. If the domain was bought elsewhere, add it to Cloudflare and change nameservers at the registrar; registrar transfer is not required.
- The tunnel must have public hostname routes for the root MyPaas domain and wildcard project domain, both routed to the MyPaas Caddy service (`HTTP` -> `caddy:80`).
- Cloudflare DNS must have proxied CNAME records for the root MyPaas host and wildcard host pointing to `<tunnel-id>.cfargotunnel.com`. If Cloudflare warns that a wildcard route will not create a DNS record, create the wildcard CNAME manually.

The installer checks Docker + Compose, generates a production `.env` with safe random secrets, creates MyPaas host directories, runs migrations, and starts the production Compose stack. For non-interactive installs, provide required values as environment variables:
```bash
PUBLIC_DOMAIN=mypaas.example.com \
OWNER_EMAIL=you@example.com \
GITHUB_CLIENT_ID=your_client_id \
GITHUB_CLIENT_SECRET=your_client_secret \
bash scripts/install-vm.sh
```

Useful installer flags:
```bash
SKIP_DEPLOY=true bash scripts/install-vm.sh          # prepare VM and .env only
FORCE_ENV=true bash scripts/install-vm.sh            # regenerate .env
SKIP_DOCKER_INSTALL=true bash scripts/install-vm.sh  # require Docker to already exist
INSTALL_WIZARD=true bash scripts/install-vm.sh       # use browser wizard for credentials
WIZARD_PUBLIC_TUNNEL=false INSTALL_WIZARD=true bash scripts/install-vm.sh  # require SSH forwarding instead
```

**Production checklist:**
- [ ] All tests passing
- [ ] Environment variables set
- [ ] Database backups configured
- [ ] Cloudflare Tunnel token valid
- [ ] SSL certificates ready (via Cloudflare)

Deploy via:
```bash
bash scripts/deploy-to-vm.sh
```

Verify after deploy or VM reboot:
```bash
bash scripts/verify-production.sh
RUN_BACKUP=true bash scripts/verify-production.sh
```

Verify local dogfooding routes:
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dogfood-smoke.ps1
```

Monitor:
- Dashboard: https://your-domain.com
- Logs: `docker compose logs -f`
- Metrics: Built-in dashboard

---

## License

MIT — See [LICENSE](LICENSE)

---

## Getting Help

- **Docs:** See `docs/` for detailed guides
- **Bug report:** File an issue with reproduction steps
- **Discussion:** Use GitHub Discussions
- **Security:** Email security-related concerns (not issues)

---

**Last updated:** 2026-07-10
