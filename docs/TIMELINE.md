# MyPaas — Implementation Timeline

Timeline implementasi MyPaas yang dipecah per-session untuk menjaga context Claude Code tetap focused. Setiap session ditargetkan selesai dalam 1-3 jam dengan deliverable yang jelas dan testable.

**Prinsip timeline:**
- Setiap session = 1 fokus area = 1 session Claude Code
- Setiap session punya **definition of done** yang bisa dites
- Setiap hari end-of-day = git commit + update CHANGELOG
- Kalau session overflow, **split** bukan paksain selesai

---

## Overview

| Fase | Hari | Fokus | Session |
|------|------|-------|---------|
| Foundation | Day 1-2 | Scaffold, schema, auth | 4 session |
| Core Deploy | Day 3-5 | Deploy engine lengkap | 6 session |
| Async & Webhook | Day 6-7 | Job queue, webhook | 3 session |
| Frontend | Day 8-10 | Dashboard UI | 6 session |
| Observability | Day 11-12 | Streaming, metrics, logs | 4 session |
| Advanced | Day 13-14 | Rollback, quota, polish | 4 session |
| Deploy | Day 15 | Self-deploy, CLI tool | 2 session |

**Total:** 29 session, ~70-100 jam coding time dalam 15 hari.

---

## Fase 1: Foundation (Day 1-2)

**Goal:** Project skeleton jalan, DB connected, auth bekerja end-to-end.

### Session 1.1 — Repository setup (1-2 jam)

**Scope:**
- Buat repo `mypaas` di GitHub (private)
- Copy `CLAUDE.md`, `docs/PRD.md`, `docs/TIMELINE.md`, `.env.example` ke repo
- Setup `.gitignore` comprehensive (Go, Node, IDE, OS)
- Buat struktur folder sesuai CLAUDE.md
- `README.md` placeholder dengan quick start
- `Makefile` dengan common commands
- `LICENSE` (MIT)

**Definition of done:**
- [ ] Repo di GitHub
- [ ] Struktur folder sesuai CLAUDE.md
- [ ] `git clone` di mesin baru langsung jelas strukturnya
- [ ] `make` tanpa argument menampilkan help

**Prompt Claude Code:**
> "Baca CLAUDE.md. Buat struktur folder sesuai section Repository structure. Setup .gitignore yang comprehensive untuk Go + SvelteKit + Docker. Buat Makefile dengan target: dev, test, lint, build, migrate-up, sqlc. README placeholder dengan quick start section."

---

### Session 1.2 — Go backend scaffold + DB (2-3 jam)

**Scope:**
- Init Go module: `go mod init github.com/nabil/mypaas/backend`
- `go.mod` dengan dependencies: chi, pgx, sqlc, golang-migrate, golang-jwt, oauth2, docker, go-git, godotenv, cron, uuid, slog
- `cmd/api/main.go` skeleton dengan graceful shutdown
- `internal/config/` untuk load env vars
- `internal/logger/` untuk slog setup
- `internal/db/` untuk pgx connection pool
- `internal/httpx/` untuk response helpers
- `internal/errs/` untuk error sentinels
- Migration `000001_init_schema.up.sql` — semua tabel per PRD section 6.1
- Migration `000001_init_schema.down.sql`
- Migration `000002_seed_ports.up.sql` — seed port 3001-9999
- `sqlc.yaml` config
- `docker-compose.dev.yml` dengan PostgreSQL
- Health check endpoint `/health` dan `/ready`
- Smoke test: binary compile, jalan, connect DB, migration sukses

**Definition of done:**
- [ ] `make migrate-up` sukses, semua tabel ada
- [ ] `go run cmd/api/main.go` jalan di port 8080
- [ ] `curl localhost:8080/health` return `{"data":{"status":"ok"}}`
- [ ] `docker compose -f docker-compose.dev.yml up -d` jalan
- [ ] Database ter-populate + port_registry terisi 3001-9999

**Prompt Claude Code:**
> "Baca CLAUDE.md dan PRD.md section 6 (Data model). Scaffold Go backend dengan dependency yang dibutuhkan. Buat migration 000001 (schema lengkap) dan 000002 (port seed). Setup sqlc config. docker-compose.dev.yml dengan PostgreSQL 16. Buat skeleton cmd/api/main.go dengan graceful shutdown + health endpoint. Jangan implement business logic apapun — fokus skeleton."

---

### Session 1.3 — SvelteKit scaffold (1-2 jam)

**Scope:**
- Init SvelteKit dengan TypeScript strict
- Install Tailwind CSS + konfigurasi
- Install Chart.js
- Folder structure sesuai CLAUDE.md
- API client skeleton di `lib/api/client.ts` dengan fetch wrapper
- Layout dasar dengan navigation placeholder
- Placeholder halaman semua route per PRD section 4.11
- Setup dark mode toggle (Tailwind class-based)
- Favicon + page title

**Definition of done:**
- [ ] `pnpm dev` jalan di port 5173
- [ ] Tailwind berfungsi
- [ ] Dark mode toggle bekerja
- [ ] Routing antar halaman bekerja
- [ ] `pnpm check` tanpa error

**Prompt Claude Code:**
> "Baca CLAUDE.md section frontend. Scaffold SvelteKit dengan TypeScript strict + Tailwind + Chart.js. Folder structure sesuai CLAUDE.md. Placeholder pages sesuai PRD section 4.11. Dark mode support dengan Tailwind class strategy. Belum implement fetching — fokus skeleton dengan hardcoded placeholder."

---

### Session 1.4 — Authentication end-to-end (3-4 jam)

**Scope backend:**
- `internal/auth/oauth.go` — GitHub OAuth flow (login, callback)
- `internal/auth/jwt.go` — JWT generation + validation (access + refresh token)
- `internal/auth/middleware.go` — JWT auth middleware untuk Chi
- `internal/user/` — User query via sqlc, service layer, handler
- Endpoint per PRD section 7.1 (`/auth/github/login`, `/auth/github/callback`, `/auth/refresh`, `/auth/me`, `/auth/logout`)
- Seed owner user manual via SQL migration atau CLI command
- Whitelist validation di OAuth callback

**Scope frontend:**
- `/login` page dengan tombol "Login with GitHub"
- Auth store (`lib/stores/auth.ts`)
- API client inject JWT dari store otomatis + refresh on 401
- Route guard di `+layout.ts`
- Dashboard `/` fetch `/auth/me` sebagai proof of login
- Logout button

**Definition of done:**
- [ ] Owner email ter-seed ke DB
- [ ] Visit `/auth/github/login` → redirect GitHub
- [ ] Login dengan email whitelisted → dapat JWT → masuk dashboard
- [ ] Login dengan email non-whitelisted → 403 halaman error
- [ ] Dashboard tampilkan email user + avatar
- [ ] Logout → kembali ke /login, JWT cleared
- [ ] Protected route tanpa token → redirect /login
- [ ] JWT expired → auto refresh via refresh token

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.1, 7.1, 4.11 FR-UI-1. Implement GitHub OAuth flow lengkap dengan whitelist validation, JWT (access + refresh), dan protected middleware. Seed owner email via SQL atau CLI. Frontend: /login page, auth store, API client dengan auto-refresh, route guard, logout. Test end-to-end manual dengan owner email."

---

## Fase 2: Core Deploy Engine (Day 3-5)

**Goal:** Bisa deploy project real (Dockerfile + Compose mode), accessible via subdomain.

### Session 2.1 — Project CRUD (2 jam)

**Scope:**
- Domain struct `Project` di `internal/project/domain.go`
- sqlc queries di `query/projects.sql`
- Service layer `internal/project/service.go`
- HTTP handler `internal/project/handler.go`
- Endpoints per PRD section 7.2
- Validasi: nama unique, alphanumeric + dash, 3-30 chars
- Generate subdomain otomatis dari nama
- Generate webhook secret (32 bytes random)
- Soft delete

**Definition of done:**
- [ ] `POST /projects` dengan form create project → 201 + project data
- [ ] `GET /projects` list project
- [ ] `GET /projects/:id` detail
- [ ] `PATCH /projects/:id` update (rename)
- [ ] `DELETE /projects/:id` soft delete
- [ ] Nama duplikat → 409
- [ ] Nama invalid → 400 dengan pesan jelas

**Prompt Claude Code:**
> "Baca CLAUDE.md (error pattern) + PRD section 4.2, 7.2. Implement project CRUD di internal/project/. Pakai sqlc query di query/projects.sql. Follow error pattern sentinel di internal/errs/. Validasi nama, generate webhook secret pakai crypto/rand. Belum implement deploy — fokus CRUD."

---

### Session 2.2 — Port registry + Docker client wrapper (2-3 jam)

**Scope:**
- `internal/port/service.go` — allocate, release, list available
- Query dengan `FOR UPDATE SKIP LOCKED` untuk atomic allocation
- Double-check: socket bind test sebelum confirm port available
- `internal/container/docker.go` — wrapper Docker Go client
- Operations: pull image, build image dari Dockerfile, run container, stop, remove, inspect, logs stream, stats
- Interface `ContainerManager` untuk testability

**Definition of done:**
- [ ] Unit test PortService: allocate → use → release cycle
- [ ] Concurrent allocate test (100 goroutine) → semua dapat port berbeda
- [ ] Port exhausted → return `ErrPortPoolExhausted`
- [ ] Integration test DockerClient: build image nginx hello, run, stop, remove
- [ ] Port stale detection: port di registry tapi dipakai proses lain → skip

**Prompt Claude Code:**
> "Baca CLAUDE.md. Implement internal/port/ dengan atomic allocation (FOR UPDATE SKIP LOCKED) + socket bind verification. Implement internal/container/ wrapper Docker client untuk Dockerfile mode operations. Interface ContainerManager untuk mock test. Integration test pakai Testcontainers."

---

### Session 2.3 — Caddy integration (1-2 jam)

**Scope:**
- `internal/caddy/client.go` — HTTP client ke Caddy Admin API `localhost:2019`
- Operations: add route, remove route, reload config, get config
- Route config: subdomain → 127.0.0.1:PORT
- Handle error: Caddy down, config invalid, reload fail
- Test dengan Caddy via docker-compose.dev.yml

**Definition of done:**
- [ ] Unit test dengan HTTP mock
- [ ] Integration test: add route → curl subdomain → get response → remove route → 404
- [ ] Error handling: Caddy down → return wrapped error

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.6. Implement internal/caddy/ client untuk Caddy Admin API. CRUD route untuk subdomain → port mapping. Integration test dengan Caddy container di docker-compose.dev.yml."

---

### Session 2.4 — Env vars management + encryption (1-2 jam)

**Scope:**
- `internal/crypto/aes_gcm.go` — AES-256-GCM encrypt/decrypt utility
- Key loaded dari `ENV_ENCRYPTION_KEY` env var
- sqlc queries di `query/env_vars.sql`
- Service `internal/envvar/service.go` — CRUD env vars dengan auto encrypt/decrypt
- Handler per PRD section 7.4
- Writer utility: `WriteEnvFile(vars, path)` untuk generate `.env` file saat deploy

**Definition of done:**
- [ ] Encrypt-decrypt roundtrip test
- [ ] Unique key constraint di DB enforced
- [ ] `PUT /projects/:id/env` bulk update
- [ ] `DELETE /projects/:id/env/:key` delete single
- [ ] Generated `.env` file format valid untuk Docker `--env-file`

**Prompt Claude Code:**
> "Baca CLAUDE.md security checklist + PRD section 4.2 FR-PROJ-6/7. Implement internal/crypto/ (AES-256-GCM), internal/envvar/ (CRUD dengan encryption). Writer utility generate .env file. Test encrypt-decrypt roundtrip + DB integration."

---

### Session 2.5 — Deploy engine Dockerfile mode (3-4 jam)

**Scope:**
- `internal/deployment/service.go` — orchestration
- `internal/deployment/git.go` — git clone wrapper via go-git
- Flow lengkap per PRD FR-DEPLOY-4 (13 langkah)
- State machine: queued → cloning → building → starting → configuring → running / failed
- Update status di DB per step
- Build workspace di `/tmp/mypaas/builds/{deploymentId}`, cleanup after
- Sinkron dulu — deploy endpoint wait sampai selesai (async di session berikutnya)
- Endpoint `POST /projects/:id/deploy`

**Definition of done:**
- [ ] Create project dengan repo public Dockerfile (misal github.com/docker/getting-started)
- [ ] Trigger deploy → selesai dalam <5 menit
- [ ] `curl http://{subdomain}.localhost` → response dari container
- [ ] Deploy ulang → container lama stop gracefully (SIGTERM), baru up
- [ ] Delete project → container + Caddy entry + port + volume cleaned
- [ ] Deploy tanpa Dockerfile → fail dengan `ErrDockerfileNotFound`

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.3. Implement internal/deployment/ Dockerfile mode lengkap (sinkron dulu). Git clone via go-git. Flow 13 langkah per FR-DEPLOY-4. State machine dengan update DB per step. Cleanup workspace. Test manual dengan repo Dockerfile simple."

---

### Session 2.6 — Deploy engine Compose mode (3-4 jam)

**Scope:**
- `internal/container/compose.go` — wrapper untuk exec `docker compose` CLI
- Operations: up, down, logs, ps, stats
- Inject `docker-compose.override.yml` untuk resource limits + port mapping
- Service validation: nama service utama yang di-form harus ada di compose file
- Parse compose file pakai `gopkg.in/yaml.v3`
- Update `internal/deployment/service.go` untuk handle Compose mode
- Flow per PRD FR-DEPLOY-5 (14 langkah)

**Definition of done:**
- [ ] Test repo multi-service: app (Node.js) + redis + postgres
- [ ] Deploy sukses, app accessible via subdomain
- [ ] App bisa connect redis & postgres (internal network)
- [ ] Delete → `docker compose down`, volume cleanup, port released
- [ ] Redeploy → down + up lagi, state preserved di volume
- [ ] Service utama tidak ada di compose → fail dengan error jelas

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.3 (FR-DEPLOY-5). Extend internal/container/ dengan Compose CLI wrapper. Generate docker-compose.override.yml untuk inject resource limits + port. Parse compose file validate service utama. Test manual dengan repo multi-service."

---

## Fase 3: Async & Webhook (Day 6-7)

### Session 3.1 — Async job queue (3 jam)

**Scope:**
- `internal/deployment/queue.go` — in-memory queue pakai channel
- Worker pool dengan concurrent limit (global max 2)
- Serialize per project (satu project nggak bisa 2 build bareng) via project-level lock
- Deduplication: kalau ada deploy pending untuk project sama, cancel yang lama
- Refactor deploy endpoint jadi async: return 202 + deploymentId
- Recovery saat startup: status `building`/`queued` yang tertinggal → mark `failed`
- Graceful shutdown: stop terima job baru, wait in-flight selesai (max 30 detik)

**Definition of done:**
- [ ] Deploy trigger return 202 dalam <100ms
- [ ] Polling `/deployments/:id` status update realtime
- [ ] 2 deploy cepat ke project sama → yang kedua menggantikan pending pertama
- [ ] 3 deploy ke project berbeda → 2 concurrent, 1 antri
- [ ] Restart API di tengah deploy → status recovery ke failed
- [ ] Graceful shutdown: wait in-flight max 30 detik sebelum force exit

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-DEPLOY-6/7. Refactor deployment jadi async. Worker pool dengan global limit 2, per-project serialize lock. Deduplication di queue. Recovery state pada startup. Graceful shutdown. Handler return 202."

---

### Session 3.2 — Webhook handler (2-3 jam)

**Scope:**
- Endpoint `POST /webhook/:projectId` public, bypass JWT
- Verify `X-Hub-Signature-256` dengan `hmac.Equal`
- Rate limit 10 req/menit per project (token bucket)
- Parse GitHub push event payload (ref, head_commit.id, repository)
- Filter branch sesuai config project
- Log delivery ke tabel `webhook_deliveries`
- Enqueue deployment baru dengan `triggered_by='webhook'`
- Test lokal pakai webhook simulator atau ngrok

**Definition of done:**
- [ ] Webhook valid signature + event push + branch match → deploy ter-enqueue
- [ ] Invalid signature → 401 + log attempt
- [ ] Event selain push → 200 OK (ignored, logged)
- [ ] Branch lain → 200 OK (ignored)
- [ ] Rate limit: 11 webhook dalam 1 menit → yang ke-11 429
- [ ] Test real via GitHub webhook → deploy trigger otomatis

**Prompt Claude Code:**
> "Baca CLAUDE.md security + PRD section 4.5. Implement internal/webhook/ handler. Signature verify dengan hmac.Equal. Rate limit token bucket per project. Parse minimal payload GitHub. Log ke webhook_deliveries. Bypass JWT via Chi routing. Test pakai webhook simulator."

---

### Session 3.3 — Build log streaming + audit log (2-3 jam)

**Scope:**
- Update `internal/deployment/service.go` — capture Docker build output via io.Pipe
- Simpan ke `deployments.build_log` incremental (append per line)
- `internal/audit/service.go` — log action ke tabel `audit_logs`
- Middleware Chi untuk auto-log: action, user_id, resource, IP, user agent
- Endpoint `GET /projects/:id/deployments` dengan pagination
- Endpoint `GET /deployments/:id` detail

**Definition of done:**
- [ ] Build log di DB bisa di-fetch via endpoint
- [ ] Audit log otomatis tercatat untuk action: project.created, project.deleted, deployment.triggered, user.login, dll
- [ ] Pagination deployment history bekerja (cursor atau offset)
- [ ] Deployment detail include full build log

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.4. Capture Docker build output real-time simpan ke DB append mode. Implement internal/audit/ dengan Chi middleware auto-log. Endpoint deployment history dengan pagination. Endpoint deployment detail."

---

## Fase 4: Frontend Dashboard (Day 8-10)

**Goal:** Dashboard fungsional lengkap.

### Session 4.1 — Dashboard + project list (2-3 jam)

**Scope:**
- Halaman `/` fetch `/projects`
- Komponen `ProjectCard.svelte` — nama, subdomain, status badge, last deployed, deploy mode indicator, quick action (restart, stop)
- Komponen `StatusBadge.svelte` — color per status
- Empty state: CTA ke New Project + tutorial singkat
- Skeleton loading
- Error state retry

**Definition of done:**
- [ ] Login → dashboard menampilkan list project
- [ ] Empty state saat belum ada project
- [ ] Status badge color benar (running, stopped, crashed, building, queued)
- [ ] Quick action bekerja inline (tidak navigate)
- [ ] Deploy mode indicator: "Dockerfile" atau "Compose"
- [ ] Responsive 1280px+

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.11 FR-UI-2. Implement dashboard page + ProjectCard + StatusBadge components. Empty state dengan CTA. Skeleton loading. Quick action inline (restart, stop). Responsive 1280px+. Data dari API real."

---

### Session 4.2 — New project form (2 jam)

**Scope:**
- Halaman `/projects/new`
- Form fields: nama, repo URL, branch, deploy mode (auto/Dockerfile/Compose), service utama (Compose), app port, memory limit, CPU limit
- Auto-detect: saat user paste repo URL, bisa ada tombol "Detect" yang fetch repo root (via backend) untuk cek Dockerfile vs compose
- Validasi client + server
- Loading state submit
- Error inline

**Definition of done:**
- [ ] Validation client bekerja (nama format, port range, dll)
- [ ] Submit sukses → redirect ke detail project
- [ ] Nama duplikat → error di field form
- [ ] Service utama hanya muncul kalau mode Compose
- [ ] Auto-detect mode bekerja (optional, kalau waktu cukup)

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.2, 4.11 FR-UI-3. Implement /projects/new form dengan semua field dari FR-PROJ-1. Validation client-side + server-side. Auto-detect mode bisa via tombol 'Detect' yang fetch ke backend endpoint baru /projects/detect-mode. Error inline."

---

### Session 4.3 — Project detail: Overview tab (2 jam)

**Scope:**
- Layout tab di `/projects/[id]`
- Tab Overview: status card, subdomain (clickable), last deployed info, commit hash, uptime, deploy mode indicator
- Action buttons: Deploy, Start, Stop, Restart
- Confirmation modal untuk action destructive
- Live update status via polling (SSE di Day 11)

**Definition of done:**
- [ ] Semua action bekerja end-to-end
- [ ] Status update setelah action (polling 2 detik)
- [ ] Subdomain clickable → buka di new tab
- [ ] Commit hash clickable → buka GitHub commit
- [ ] Confirmation modal untuk stop/restart

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-UI-4. Implement project detail Overview tab. Tab navigation component reusable. Action buttons dengan confirmation modal. Polling status 2 detik interval sementara (akan diganti SSE di Day 11)."

---

### Session 4.4 — Project detail: Environment + Settings tab (2-3 jam)

**Scope:**
- Tab Environment: table env vars dengan add/edit/delete inline, value masked (show/hide toggle)
- Tab Settings: rename project, change branch, change resource limits, webhook URL + secret (copy + regenerate), delete project dengan konfirmasi
- Konfirmasi delete project dengan typing nama project (seperti GitHub)
- Optimistic update untuk env vars

**Definition of done:**
- [ ] Env vars CRUD bekerja, persist di DB
- [ ] Mask/unmask value toggle
- [ ] Rename project → subdomain update, Caddy config updated
- [ ] Regenerate webhook secret → old secret invalid
- [ ] Delete project dengan typing confirmation
- [ ] Resource limit change → prompt redeploy untuk apply

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-UI-4. Implement Environment tab (table CRUD inline) + Settings tab (rename, branch, limits, webhook, delete). Mask value default, toggle show. Konfirmasi delete dengan typing nama project. Optimistic update env vars."

---

### Session 4.5 — Project detail: Deployments + Rollback (2 jam)

**Scope:**
- Tab Deployments: table history dengan kolom: date, commit, status, duration, triggered_by
- Expandable row → show build log
- Button Rollback per successful deployment
- Konfirmasi rollback modal
- Loading state saat rollback jalan
- Pagination

**Definition of done:**
- [ ] History table menampilkan deployment terurut DESC by started_at
- [ ] Expand row → tampil build log terformat
- [ ] Rollback button hanya muncul di deployment status 'running' sebelumnya (yang sukses)
- [ ] Rollback bekerja: container swap ke image lama tanpa rebuild
- [ ] Loading state visible saat rollback

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.4. Implement Deployments tab: history table, expandable row, rollback action. Backend: endpoint POST /deployments/:id/rollback yang swap container ke image_tag lama. Test rollback 2x berturut."

---

### Session 4.6 — Admin users page (1-2 jam)

**Scope:**
- Halaman `/admin/users` (owner only)
- List users + action: add, remove, view last login
- Form add user: email + role (owner/collaborator)
- Endpoint `/admin/users/*` per PRD section 7.7

**Definition of done:**
- [ ] Owner bisa akses halaman
- [ ] Non-owner redirect atau hide menu
- [ ] Add user → bisa langsung login dengan email itu
- [ ] Remove user → user tidak bisa login lagi (existing JWT tetap valid sampai expiry)

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 7.7. Implement admin users page + backend endpoint. Owner-only via role check middleware. List, add, remove user. Update internal/auth untuk check role."

---

## Fase 5: Observability (Day 11-12)

### Session 5.1 — Unified SSE stream (3 jam)

**Scope:**
- Endpoint `GET /projects/:id/stream`
- Typed event: metrics, log, deployment, status
- Implementation: goroutine per connection, channel fan-out
- Metrics scheduler: poll Docker Stats per 5 detik, broadcast ke subscriber project
- Log tail: attach ke docker logs --follow, broadcast ke subscriber
- Deployment event: dipush dari deployment service (wiring via event bus internal)
- Heartbeat comment tiap 30 detik untuk prevent idle timeout
- Close connection saat project dihapus

**Definition of done:**
- [ ] Buka stream → terima heartbeat
- [ ] Trigger deploy → terima events urut: deployment (building) → log lines → deployment (running) → metrics
- [ ] Multi-client: 2 browser buka stream sama → keduanya terima event yang sama
- [ ] Tutup browser → connection close, resource cleaned
- [ ] Project dihapus → connection close dengan event terminal

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.10, ADR-004/005. Implement unified SSE stream endpoint. Channel fan-out dengan goroutine per connection. Metrics scheduler + log tail goroutine. Event bus internal untuk deployment events. Heartbeat 30 detik. Test multi-client."

---

### Session 5.2 — Frontend: Logs tab (2-3 jam)

**Scope:**
- Tab Logs di project detail
- Load history via REST `/projects/:id/logs` (paginated)
- Connect SSE stream, filter event type 'log'
- UI: virtualized log viewer (untuk performa), auto-scroll, pause saat user scroll up
- Filter text input (client-side)
- Service selector dropdown (Compose mode) — filter per service
- Download logs button (export text file)

**Definition of done:**
- [ ] History ter-load saat tab dibuka
- [ ] Realtime stream dimulai setelah history
- [ ] Auto-scroll aktif, pause saat scroll up, resume saat scroll to bottom
- [ ] Filter text hide baris tidak match
- [ ] Service selector (Compose) filter per service
- [ ] Download logs export .txt
- [ ] Performance baik di 5000+ baris

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-MON-5/6/7, FR-UI-4. Implement Logs tab dengan virtualized viewer (svelte-virtual-list atau custom). History load via REST, realtime via SSE. Auto-scroll dengan pause detection. Filter text client-side. Service selector Compose. Export text."

---

### Session 5.3 — Frontend: Metrics tab (2 jam)

**Scope:**
- Tab Metrics di project detail
- Connect SSE stream, filter event 'metrics'
- Chart.js line chart: CPU% dan Memory MB, 60 data point terakhir (5 menit)
- Current value gauge (big number)
- Compose mode: tabs atau sections per service
- Uptime counter

**Definition of done:**
- [ ] Chart update realtime tiap 5 detik
- [ ] Current value gauge visible
- [ ] Compose mode menampilkan metrics per service
- [ ] Uptime counter update per detik
- [ ] Chart tidak freeze saat banyak data

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-MON-1/2/3. Implement Metrics tab: Chart.js line chart (60 data point rolling), current value gauge, uptime counter. Compose mode: per-service view. Data via SSE event 'metrics'."

---

### Session 5.4 — Prometheus metrics endpoint (1 jam)

**Scope:**
- `internal/monitoring/prometheus.go` — setup prometheus metrics
- Custom metrics: deployment_total, deployment_duration_seconds, active_containers, port_pool_utilization
- Endpoint `/metrics` dengan basic auth
- Standard metrics: go runtime, process

**Definition of done:**
- [ ] `curl /metrics` return format Prometheus valid
- [ ] Basic auth works
- [ ] Custom metrics increment saat deploy trigger/finish
- [ ] Standard go metrics exposed

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD FR-MON-9. Add prometheus/client_golang dependency. Implement /metrics dengan basic auth. Custom metrics: deployment_total (counter), deployment_duration_seconds (histogram), active_containers (gauge), port_pool_utilization (gauge)."

---

## Fase 6: Advanced Features (Day 13-14)

### Session 6.1 — Resource quota enforcement (2 jam)

**Scope:**
- `internal/quota/service.go` — calculate current usage, check before deploy
- Quota per user dari config env var
- Hook di deployment service: check quota sebelum allocate
- Endpoint `GET /me/quota` — usage breakdown
- Frontend: quota bar di dashboard + warning saat mendekati limit

**Definition of done:**
- [ ] Deploy ketika quota habis → 409 + error jelas
- [ ] Endpoint /me/quota return: total_memory, used_memory, total_cpu, used_cpu, project_count
- [ ] Dashboard tampilkan quota bar
- [ ] Warning muncul saat usage > 80%

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.8. Implement internal/quota/ dengan usage calculation. Hook di deployment service pre-allocation. Endpoint /me/quota. Frontend quota bar di dashboard dengan warning state."

---

### Session 6.2 — Audit log viewer (1-2 jam)

**Scope:**
- Halaman `/admin/audit-logs` (owner only)
- Filter: action, user, date range
- Table dengan detail expandable
- Export CSV

**Definition of done:**
- [ ] List audit logs terurut DESC
- [ ] Filter bekerja
- [ ] Expand row menampilkan metadata JSON
- [ ] Export CSV berjalan

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.4 (audit log). Implement /admin/audit-logs page owner-only. Filter by action, user, date. Expandable row untuk metadata. Export CSV client-side."

---

### Session 6.3 — Backup automation + scheduled cleanup (2 jam)

**Scope:**
- `internal/backup/service.go` — pg_dump ke `/var/lib/mypaas/backups/`
- Schedule daily via robfig/cron
- Retention: keep 7 daily + 4 weekly
- Optional: upload ke Cloudflare R2 via S3-compatible API (kalau `R2_*` env var diset)
- Scheduled cleanup: hapus image lama (tidak dipakai container running) seminggu sekali
- CLI command `mypaas backup` untuk manual trigger

**Definition of done:**
- [ ] Backup daily jalan otomatis
- [ ] File backup muncul di folder dengan naming yang benar
- [ ] Retention rotate otomatis
- [ ] R2 upload bekerja (kalau diset)
- [ ] Image cleanup tidak mengganggu container running

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD NFR-REL-4. Implement internal/backup/ pg_dump scheduler. Retention: 7 daily + 4 weekly. Optional R2 upload pakai aws-sdk-go-v2 (S3 compatible). Scheduled image cleanup (docker image prune filter 7d). Test manual."

---

### Session 6.4 — Polish pass (2-3 jam)

**Scope:**
- Bug list dari dogfooding
- Loading/error/empty state yang kurang
- Reusable action button dengan inline spinner/loading state untuk semua async actions
- Project detail tab navigation harus terasa client-side/SPA-like tanpa full layout reload
- Keyboard shortcut: `g d`, `g n`, `/`
- Toast notification system
- Persistent banner untuk ongoing deployment
- Error message improvements
- Accessibility: keyboard navigation, ARIA labels

**Definition of done:**
- [ ] Semua P0 user story bekerja tanpa error
- [ ] Tidak ada "undefined"/"null" bocor ke UI
- [ ] Error network di-handle dengan retry
- [ ] Semua action button async disable saat pending, menampilkan spinner, dan tidak double-submit
- [ ] Project detail tab switching mempertahankan header/tab shell, loading hanya muncul di konten tab
- [ ] Keyboard shortcut berfungsi
- [ ] Accessibility basic compliant

**Prompt Claude Code:**
> "Review semua halaman untuk loading/error/empty state consistency. Implement reusable Button/IconButton dengan inline spinner untuk semua async actions, prevent double-submit, dan pastikan project detail tab navigation terasa client-side/SPA-like tanpa full layout loading flash. Implement toast system + persistent banner. Keyboard shortcut per NFR-USE-4. Accessibility basic pass (ARIA, keyboard nav). Fix inconsistency. List issues sebelum fix, prioritize dengan owner."

---

### Session 6.5 — Resource efficiency gate before VM deploy (4-6 jam)

**Scope:**
Minimum pre-deploy gate:
- Resource profiles untuk New Project: static/no-runtime, Go small, Node/Python, Compose main, Compose side service
- Default memory/CPU limit mengikuti profile, bukan selalu 512MB
- Dashboard quota membedakan configured limit vs real Docker Stats usage
- Static no-container hosting path untuk project static/build output (serve langsung via Caddy filesystem)
- Shared PostgreSQL provisioning design + minimal implementation:
  - create database/user per project
  - inject `DATABASE_URL`
  - document sample CRUD app tanpa bundled Postgres container
- Env discovery untuk New Project:
  - scan `.env.example` / `.env.sample` / `.env.template` / `.env.local.example`
  - scan Compose interpolation `${VAR}` dari compose file
  - tampilkan key sebagai draft env yang user isi sebelum deploy
- Compose stale volume guard:
  - warning jika Docker volume/network/container lama dengan compose project name yang sama ditemukan
  - reset project volumes sebagai action eksplisit
  - hint troubleshooting untuk error DB auth/schema yang mengarah ke stale volume
- Dogfood ulang 5 sample projects dengan target configured app memory <= 2GB di luar platform DB

After-deploy design only:
- Idle sleep / wake-on-request ADR
- Autosizing recommendation ADR
- Optional single-host replicas ADR
- Kubernetes/multi-node autoscaling explicitly out of MVP

**Definition of done:**
- [ ] New Project form bisa memilih/menyarankan resource profile
- [ ] Default static/go/node/compose profile lebih hemat dari 512MB flat default
- [ ] Dashboard quota menampilkan configured limit dan real memory usage secara terpisah
- [ ] Static sample bisa berjalan tanpa nginx/app container dedicated
- [ ] CRUD DB sample bisa memakai shared PostgreSQL MyPaas
- [ ] New Project form auto-discover env keys dari `.env.example`/Compose variables dan user bisa mengisi value sebelum deploy
- [ ] Compose stale volume guard menampilkan warning/reset action saat resource lama ditemukan atau DB auth/schema gagal
- [ ] 5 sample projects lolos smoke test dengan total configured app memory <= 2GB
- [ ] ADR dibuat untuk idle sleep/wake-on-request, autosizing recommendation, dan optional single-host replicas
- [ ] PRD pre-deploy vs after-deploy grouping tetap sinkron dengan implementasi

**Prompt Claude Code:**
> "Baca PRD FR-PROJ-8..9, FR-LIFE-8..9, FR-QUOTA-4..10, NFR-PERF-7..8, dan Delivery goal grouping. Implement hanya pre-deploy gate: resource profiles, configured-vs-real quota display, static no-container hosting, minimal shared Postgres provisioning, env discovery dari `.env.example`/Compose variables, Compose stale volume guard/reset action, dan dogfood 5 sample projects <= 2GB configured app memory. Buat ADR untuk after-deploy items: idle sleep/wake-on-request, autosizing recommendation, optional single-host replicas. Jangan implement Kubernetes/multi-node autoscaling."

---

## Fase 7: Self-Deploy (Day 15)

### Session 7.1 — CLI tool (2 jam)

**Scope:**
- `cmd/cli/main.go` pakai urfave/cli atau cobra
- Command: `user add <email>`, `user list`, `user remove <id>`
- Command: `project list`, `project deploy <name>`, `project logs <name>`
- Command: `backup`
- Config: read JWT token dari `~/.mypaas/config.yml`
- Login flow CLI: generate token via API dengan special flag

**Definition of done:**
- [ ] `mypaas user add test@example.com` → user created
- [ ] `mypaas project list` → table projects
- [ ] `mypaas project deploy myapp` → trigger deploy, stream build log
- [ ] `mypaas backup` → manual backup

**Prompt Claude Code:**
> "Baca CLAUDE.md + PRD section 4.12. Implement CLI di cmd/cli/ pakai urfave/cli v2. Commands per FR-CLI-1. Config file ~/.mypaas/config.yml untuk JWT. Build binary: go build -o bin/mypaas cmd/cli/main.go."

---

### Session 7.2 — Deploy MyPaas ke VM (3-4 jam)

**Scope:**
- `Dockerfile` backend multi-stage (builder Go + runtime distroless)
- `Dockerfile` frontend multi-stage (builder Node + nginx)
- `docker-compose.prod.yml`: MyPaas backend + frontend + PostgreSQL + Caddy
- GitHub Actions: build + push image ke GHCR
- `scripts/deploy-to-vm.sh` untuk first-time deploy
- Setup VM:
  - Docker + Docker Compose
  - Cloudflare Tunnel wildcard hostname (via dashboard Cloudflare)
  - Clone repo, `.env` production
  - `docker compose -f docker-compose.prod.yml up -d`
- End-to-end test:
  - Akses dashboard via dashboard.nabilrizkinavisa.me
  - Login
  - Deploy 5 project real (static, Node, Python, Go, Compose)
  - Webhook test
  - Rollback test
- Cron backup setup

**Definition of done:**
- [ ] MyPaas accessible dari internet
- [ ] 5 project real deployed sukses dan accessible via subdomain
- [ ] Webhook trigger redeploy bekerja
- [ ] Rollback berhasil di 2 project
- [ ] Backup daily jalan (cek keesokan hari)
- [ ] VM restart → semua service auto-start
- [ ] CLAUDE.md dan PRD updated dengan link production

**Prompt Claude Code:**
> "Buat Dockerfile production untuk backend (Go multi-stage, distroless) dan frontend (Node → nginx). docker-compose.prod.yml dengan semua service. GitHub Actions build + push ke GHCR. scripts/deploy-to-vm.sh. Checklist end-to-end test post-deploy. Setup cron backup."

---

## Post Day 15 — Future work

Post-MVP enhancement:

- **F1:** Custom domain per project (selain subdomain default)
- **F2:** Branch preview deployment
- **F3:** Shared environment variables antar project
- **F4:** Grafana dashboard dari Prometheus metrics
- **F5:** Deployment notification ke Discord/Slack
- **F6:** Scheduled deployment (cron-like)
- **F7:** Self-deploy mode (MyPaas deploy dirinya sendiri via Git push)
- **F8:** Mobile full-feature UI

---

## Tips menggunakan timeline ini dengan Claude Code

**1. Satu session = satu Claude Code chat**

Jangan gabung multiple session dalam satu chat. Context membesar → hallucination naik. Selesai Session X.Y, buka chat baru untuk X.Y+1.

**2. Prompt opening tiap session**

```
Saya mulai Session X.Y - {nama session}.

Baca dulu:
- CLAUDE.md
- docs/PRD.md section {yang relevan}
- docs/TIMELINE.md Session X.Y
- File existing di package/folder terkait

Scope: {copy dari TIMELINE.md}.

Mulai dengan plan 3-5 langkah, konfirmasi ke saya sebelum coding.
```

**3. End of session checklist**

Setiap session selesai:
- [ ] Definition of done tercapai
- [ ] Git commit descriptive
- [ ] Test (manual atau automated) pass
- [ ] Update CHANGELOG.md
- [ ] Update CLAUDE.md kalau ada pattern/constraint baru

**4. Kalau session overflow**

Jangan paksain. Split jadi:
- Session X.Ya — bagian yang kelar
- Session X.Yb — lanjutan, chat baru

**5. Kalau stuck**

- Stop
- Revert atau simpan di branch lain
- Balik ke chat biasa (bukan Claude Code) diskusi arsitektur
- Setelah clarity, Claude Code session baru

**6. Testing mindset**

Jangan tunda test sampai akhir. Setiap session punya "Definition of done" yang testable — kalau tidak pass, jangan lanjut ke session berikutnya.

**7. Commit frequency**

Commit per-task, bukan per-session. Satu session bisa 3-5 commit kecil. Lebih mudah revert dan review.

---

*Last updated: 2026-04-20*
*Total estimasi: 15 hari kalender, ~70-100 jam coding time*
