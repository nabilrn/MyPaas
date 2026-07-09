# CLAUDE.md — MyPaas

Konteks persistent untuk Claude Code di project ini. File ini **dibaca setiap session** — jaga agar tetap concise, actionable, dan up-to-date.

---

## Project overview

**MyPaas** adalah self-hosted personal deployment platform untuk satu user (owner) + beberapa collaborator whitelisted. Mengganti kebiasaan deploy manual via GitHub Actions untuk project personal, termasuk project yang butuh multi-service (app + cache + database + message broker).

**Scope ringkas:**
- Owner connect Git repository → MyPaas detect Dockerfile atau docker-compose.yml → build & run → accessible via subdomain dengan SSL otomatis (via Cloudflare)
- Push ke GitHub → otomatis redeploy via webhook
- Support rollback ke deployment sukses sebelumnya
- Dashboard ala Vercel/Railway — clean, realtime metrics & logs, multi-service aware

**Detail lengkap:** baca `docs/PRD.md`. Kalau ada konflik antara CLAUDE.md dan PRD, **PRD yang menang** — minta konfirmasi ke owner sebelum ambil keputusan.

---

## Tech stack (locked)

**Backend:**
- Go 1.22+
- Chi v5 (HTTP router, stdlib-compatible)
- pgx v5 (PostgreSQL driver)
- sqlc (type-safe SQL query generator)
- golang-migrate (database migration)
- golang-jwt/jwt v5 (JWT)
- golang.org/x/oauth2 (OAuth2 client)
- docker/docker (official Docker client)
- go-git/go-git v5 (Git operations)
- slog (structured logging, stdlib Go 1.21+)
- joho/godotenv (load .env)
- robfig/cron v3 (scheduler)
- google/uuid (UUID generation)

**Frontend:**
- SvelteKit (latest stable)
- TypeScript (strict mode)
- Tailwind CSS
- pnpm (bukan npm / yarn)
- Chart.js (untuk metrics graph)

**Infrastructure:**
- PostgreSQL 16
- Docker + Docker Compose (plugin, bukan docker-compose v1)
- Caddy 2 (internal reverse proxy, managed via Admin API)
- Cloudflare Zero Trust Tunnel (public exposure, wildcard subdomain)

**Tidak dipakai (jangan suggest):**
- Spring Boot, Kotlin, JVM-based stack (terlalu berat untuk VM 8GB)
- Gin, Echo, Fiber (pakai Chi — stdlib compatible)
- GORM, sqlx (pakai sqlc — type-safe, zero overhead)
- Kubernetes, Nomad, Docker Swarm
- Buildpack atau auto-detect runtime tanpa Dockerfile/Compose
- WebSocket (pakai SSE untuk semua streaming)
- React, Vue, Next.js untuk frontend (pakai SvelteKit)
- npm/yarn (selalu pnpm)
- Redis, RabbitMQ untuk job queue internal MyPaas (pakai in-memory queue + DB untuk scope awal)

---

## Repository structure

```
mypaas/
├── CLAUDE.md
├── README.md
├── CHANGELOG.md
├── LICENSE                      # MIT
├── .gitignore
├── .env.example
├── docker-compose.dev.yml
├── docker-compose.prod.yml
├── Caddyfile.dev
├── Caddyfile.prod
├── Makefile
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md
│   ├── TIMELINE.md
│   ├── adr/
│   └── workflows/
├── backend/
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   ├── sqlc.yaml
│   ├── migrations/
│   │   ├── 000001_init_schema.up.sql
│   │   ├── 000001_init_schema.down.sql
│   │   ├── 000002_seed_ports.up.sql
│   │   └── 000002_seed_ports.down.sql
│   ├── query/                   # sqlc query files
│   │   ├── users.sql
│   │   ├── projects.sql
│   │   ├── deployments.sql
│   │   ├── env_vars.sql
│   │   └── port_registry.sql
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go
│   │   └── cli/
│   │       └── main.go
│   └── internal/
│       ├── db/                  # generated sqlc code + connection pool
│       ├── config/
│       ├── auth/                # OAuth, JWT, middleware
│       ├── user/
│       ├── project/
│       ├── deployment/          # deploy engine, queue, orchestration
│       ├── container/           # Docker client wrapper (Dockerfile + Compose)
│       ├── caddy/
│       ├── webhook/
│       ├── monitoring/          # SSE streamer, metrics collector
│       ├── port/
│       ├── crypto/              # AES-GCM untuk env vars
│       ├── audit/
│       ├── httpx/
│       ├── errs/
│       └── logger/
└── frontend/
    ├── package.json
    ├── Dockerfile
    ├── svelte.config.js
    ├── tailwind.config.ts
    └── src/
        ├── app.html
        ├── routes/
        │   ├── +layout.svelte
        │   ├── +page.svelte
        │   ├── login/
        │   ├── projects/
        │   │   ├── new/
        │   │   └── [id]/
        │   │       ├── +page.svelte
        │   │       ├── deployments/
        │   │       ├── logs/
        │   │       ├── metrics/
        │   │       ├── env/
        │   │       └── settings/
        │   └── admin/users/
        └── lib/
            ├── api/
            ├── components/
            ├── stores/
            ├── types/
            └── utils/
```

**Aturan:**
- Backend pakai **package-by-feature**, bukan package-by-layer
- `internal/` untuk semua code yang tidak boleh di-import dari luar module
- sqlc-generated code di `internal/db/`, query SQL di `query/`, migration di `migrations/`

---

## Coding standards

### Naming conventions

**Go:**
- Package: `lowercase` satu kata — `deployment`, `container`
- Exported type: `PascalCase` — `Project`, `DeploymentStatus`
- Exported function: `PascalCase` — `DeployProject()`
- Unexported: `camelCase`
- Interface: suffix `-er` untuk single-method, descriptive name untuk multi-method
- Error variable: prefix `Err` — `ErrDockerfileNotFound`

**TypeScript/Svelte:**
- Component: `PascalCase.svelte`
- Function: `camelCase`
- Type/interface: `PascalCase`
- Constant: `SCREAMING_SNAKE_CASE`

**Database:**
- Table: `snake_case` plural
- Column: `snake_case`
- FK: `{table_singular}_id`
- Index: `idx_{table}_{columns}`

### Error handling

**Go — sentinel errors untuk domain, wrapped errors untuk context:**

```go
// internal/errs/errs.go
var (
    ErrDockerfileNotFound     = errors.New("dockerfile not found")
    ErrComposeFileNotFound    = errors.New("compose file not found")
    ErrNoDeployConfig         = errors.New("no deploy config found")
    ErrPortPoolExhausted      = errors.New("port pool exhausted")
    ErrProjectNameTaken       = errors.New("project name already taken")
    ErrEmailNotWhitelisted    = errors.New("email not in whitelist")
    ErrQuotaExceeded          = errors.New("resource quota exceeded")
)

// Wrap dengan konteks
if err := gitClient.Clone(ctx, url, path); err != nil {
    return fmt.Errorf("git clone %s: %w", url, err)
}

// Check sentinel dengan errors.Is
if errors.Is(err, errs.ErrDockerfileNotFound) {
    return httpx.Error(w, 400, "DOCKERFILE_NOT_FOUND", "...")
}
```

**HTTP layer:** translate domain error ke HTTP response. Jangan expose internal error.

**Frontend:** error sebagai data, bukan exception. API client return `{ data, error }`.

### Logging

**Format:** structured JSON via `slog`.

```go
logger.Info("deployment started",
    "projectId", project.ID,
    "deploymentId", deployment.ID,
    "commitSha", commit)
```

**Level:**
- `DEBUG` — internal detail
- `INFO` — lifecycle event
- `WARN` — recoverable issue
- `ERROR` — needs attention

**Wajib include:** `projectId`, `deploymentId`, `userId` untuk traceability.

**Jangan log:** webhook secrets, JWT, env var values, password, OAuth token.

### API conventions

**Response format:**

Success: `{ "data": { ... } }`
Error: `{ "error": { "code": "...", "message": "...", "details": { ... } } }`

**Status code:**
- `200/201/202/204` — success variants
- `400` validation, `401` unauth, `403` forbidden, `404` not found
- `409` conflict, `429` rate limit, `500` server error

### Testing

**Go:**
- Unit test untuk domain logic
- Integration test untuk repository pakai Testcontainers
- Handler test pakai `httptest`
- Table-driven untuk variasi input
- Target: >70% di `deployment/`, `auth/`, `container/`, `port/`

**Frontend:**
- Component test pakai Vitest
- E2E happy path pakai Playwright (Day 15)

---

## Common commands

### Backend

```bash
cd backend

go run ./cmd/api                  # Run dev
go run ./backend/cmd/api          # Run dev from repo root (requires go.work)
air                               # Live reload
go test ./...                     # Tests
go test -cover ./...              # Coverage
golangci-lint run                 # Lint
sqlc generate                     # Generate query code
migrate -path migrations -database "$DATABASE_URL" up       # Migrate
migrate create -ext sql -dir migrations -seq name           # New migration
go build -o bin/mypaas-api ./cmd/api                        # Build
```

### Frontend

```bash
cd frontend
pnpm install
pnpm dev
pnpm check
pnpm test
pnpm build
```

### Infrastructure

```bash
docker compose -f docker-compose.dev.yml up -d
docker compose -f docker-compose.dev.yml down
docker compose -f docker-compose.dev.yml down -v   # Reset DB
```

### Makefile

```bash
make dev              # Start dev dependencies
make test             # Run all tests
make lint             # Lint everything
make build            # Build binary + frontend
make migrate-up       # Run migrations
make sqlc             # Generate sqlc code
```

---

## Do & don't

### Do

✅ **Ask before large changes.** 5+ file atau refactor module besar → plan dulu, konfirmasi owner.

✅ **Read existing code first.** Cek utility di `internal/httpx/`, `internal/errs/` dulu.

✅ **Keep functions small.** Max 40 baris, max 5 parameter.

✅ **Handle context cancellation.** I/O operation wajib terima `context.Context`.

✅ **Close resources dengan defer.** Langsung setelah open.

✅ **Use sqlc untuk semua query.** Jangan raw SQL di handler/service.

✅ **Follow security checklist** saat handle credentials atau user input.

✅ **Update CHANGELOG.md** setiap feature selesai.

✅ **Write ADR** untuk architectural decision.

### Don't

❌ **Jangan pakai ORM.** Sudah commit ke sqlc.

❌ **Jangan install library baru tanpa konfirmasi.**

❌ **Jangan ignore error dengan `_`** tanpa alasan documented.

❌ **Jangan pakai `panic()` di business logic.** Hanya untuk unrecoverable startup state.

❌ **Jangan hardcode secret.** Via env var + `.env.example` entry.

❌ **Jangan commit secret.** `.env` di `.gitignore`.

❌ **Jangan pakai `any`/`interface{}`.** Pakai concrete type atau generic.

❌ **Jangan generate buildpack logic.** Dockerfile + Compose only.

❌ **Jangan pakai WebSocket.** Semua streaming SSE.

❌ **Jangan global state.** Dependency injection via constructor.

❌ **Jangan test framework behavior.** Test domain & integration.

---

## Security checklist

- [ ] Secret via env var, bukan hardcode
- [ ] Env var user encrypted AES-256-GCM
- [ ] Webhook signature verify dengan `hmac.Equal` (constant-time)
- [ ] JWT secret min 256-bit
- [ ] SQL via sqlc (parameterized)
- [ ] Docker socket mount hanya ke MyPaas, jangan ke container user
- [ ] Port bind ke `127.0.0.1`, bukan `0.0.0.0`
- [ ] Container user run dengan `--user` non-root
- [ ] Git credential encrypted, jangan masuk build log
- [ ] Log sanitization aktif (no secret/token/password)
- [ ] CORS strict ke origin dashboard
- [ ] Rate limit di login, webhook

---

## Constraints yang sering dilupakan

**Cloudflare Tunnel wildcard:**
- Domain `nabilrizkinavisa.me` sudah pakai Cloudflare nameserver
- Wildcard hostname `*.nabilrizkinavisa.me` → Caddy (`localhost:80`)
- SSL di-handle Cloudflare, jangan Let's Encrypt di Caddy

**Port pool:**
- Range: `3001-9999`
- Reserved: `80, 443, 8080, 5432, 22, 3000, 2019`
- Registry di PostgreSQL `port_registry`
- Allocate pakai `FOR UPDATE SKIP LOCKED`

**Docker daemon access:**
- MyPaas mount `/var/run/docker.sock`
- Compose: exec ke CLI `docker compose` (jangan reimplement)
- Docker client Go untuk single container

**Async deployment:**
- Trigger return 202 + deploymentId
- Queue in-memory + state di DB
- Serialize per project
- Global max 2 concurrent deploy

**Filesystem paths:**
- Build workspace: `/tmp/mypaas/builds/{deploymentId}` (cleanup after)
- Project volume: `/var/lib/mypaas/volumes/{projectId}` (persistent)
- Compose workspace: `/var/lib/mypaas/compose/{projectId}` (persistent untuk logs)
- Backup: `/var/lib/mypaas/backups/`

**Deploy mode detection:**
- Priority: `docker-compose.yml` atau `compose.yml` > `Dockerfile`
- Keduanya ada → Compose mode
- Tidak ada → error

**Resource limits default:**
- Per project main: 512MB RAM, 0.5 CPU (customizable)
- Compose non-main services: 256MB RAM, 0.25 CPU
- Per-user quota: 6GB RAM, 3 CPU (via .env)

---

## Reference documents

- `docs/PRD.md` — source of truth scope & requirement
- `docs/ARCHITECTURE.md` — detail teknis, diagram
- `docs/TIMELINE.md` — implementation timeline (15 hari)
- `docs/adr/` — architecture decision records
- `.env.example` — template env variables

---

## Decision log (quick reference)

Detail di `docs/adr/`.

- **ADR-001:** Backend = Go + Chi. RAM efisien untuk VM 8GB.
- **ADR-002:** Support Dockerfile + Docker Compose. Project real sering multi-service.
- **ADR-003:** Frontend = SvelteKit. Bundle kecil, realtime bagus.
- **ADR-004:** Semua streaming pakai SSE. One-way, simpler, auto-reconnect.
- **ADR-005:** Unified SSE stream per project. Satu connection lebih efisien.
- **ADR-006:** Caddy untuk reverse proxy. Admin API untuk dynamic reload.
- **ADR-007:** Cloudflare Tunnel wildcard. No manual DNS, no SSL management.
- **ADR-008:** Auth = GitHub OAuth + DB whitelist. No password management.
- **ADR-009:** Compose via exec CLI (`docker compose`). Simpler, battle-tested.
- **ADR-010:** sqlc untuk query, bukan ORM. Type-safe, zero overhead.
- **ADR-011:** In-memory queue + DB state. Single-node, cukup untuk personal.

---

## How to work with me (Claude Code)

**Prefer iteration:**
- Small chunk → review → adjust → next
- Jangan 10 file sekaligus tanpa konfirmasi

**Ask saat ambigu:**
- Jangan asumsi, tanya owner
- Multiple valid approach → present trade-off

**Use TodoWrite:**
- Task 3+ langkah → todo list dulu
- Update status saat progress

**Respect existing pattern:**
- Cek 2-3 file existing sebelum bikin baru
- Follow naming, structure, error handling yang ada

**Flag trade-off:**
- Decision signifikan → surface ke owner
- Jangan silent pick

**Generate idiomatic Go:**
- Accept interface, return struct
- Error sebagai nilai
- Context first parameter untuk I/O
- Goroutine saat genuinely concurrent

---

*Last updated: 2026-07-10*
*Maintainer: Nabil Rizki Navisa*
