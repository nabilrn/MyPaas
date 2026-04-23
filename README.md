# MyPaas — Self-Hosted Personal Deployment Platform

> Deploy your projects like Vercel/Railway, but on your own infrastructure.

MyPaas is a lightweight, self-hosted platform for deploying multi-service applications with automatic SSL, realtime logs, and a clean dashboard. Perfect for solo developers or small teams.

**Key features:**
- 🚀 **Auto-deploy** on git push via webhooks
- 🐳 **Docker + Compose** support — single container or multi-service
- 🔒 **SSL with Cloudflare** — no certificate management
- 📊 **Realtime dashboard** — logs, metrics, deployment history
- ⏮️ **Instant rollback** — go back to any previous deployment
- 🔐 **GitHub OAuth** — whitelisted collaborators only

---

## Quick Start

### Prerequisites
- **Go 1.22+** (backend)
- **Node.js 18+ & pnpm** (frontend)
- **PostgreSQL 16**
- **Docker + Docker Compose** (plugin version)
- **Caddy 2** (reverse proxy)

### 1. Clone & setup environment
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

**Production checklist:**
- [ ] All tests passing
- [ ] Environment variables set
- [ ] Database backups configured
- [ ] Cloudflare Tunnel token valid
- [ ] SSL certificates ready (via Cloudflare)

Deploy via:
```bash
docker compose -f docker-compose.prod.yml up -d
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

- 📖 **Docs:** See `docs/` for detailed guides
- 🐛 **Bug report:** File an issue with reproduction steps
- 💬 **Discussion:** Use GitHub Discussions
- 🔒 **Security:** Email security-related concerns (not issues)

---

**Last updated:** 2026-04-23
