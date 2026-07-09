# MyPaas — Product Requirements Document

**Author:** Nabil Rizki Navisa
**Status:** v2.0
**Last updated:** 20 April 2026
**Timeline:** 10-15 hari

---

## 1. Overview

### 1.1 Problem statement

Deployment project personal saat ini memiliki beberapa pain point:

- **GitHub Pages** terbatas pada static site, tidak bisa jalankan backend
- **GitHub Actions workflow** sering error dan butuh YAML yang ditulis manual per project
- **Railway / Render** punya free tier yang terbatas dan cold start
- **Manual VPS deployment** repetitive dan memakan waktu — setup Nginx, SSL, systemd service, dll per project
- **Project real** sering butuh lebih dari satu service (app + cache + message broker + database) — tidak cukup hanya single container

### 1.2 Solution

MyPaas adalah self-hosted personal deployment platform yang berjalan di VPS/VM sendiri, memungkinkan deployment project personal cukup dengan connect Git repository. Platform handle build, routing, SSL, subdomain, multi-service orchestration, dan lifecycle management secara otomatis.

### 1.3 Target user

Single user: pengembang (owner) yang deploy project personal. Bukan multi-tenant, bukan SaaS, bukan untuk publik luas. Mungkin share akses ke beberapa teman sebagai collaborator whitelisted.

### 1.4 Non-goals

Hal-hal berikut **bukan** tujuan MyPaas dan tidak akan di-implement:

- Multi-tenant SaaS dengan isolation enterprise-grade
- Billing, quota management, atau payment system
- Team collaboration features (PR review, org management, dll)
- Buildpack / auto-detect runtime — user bawa Dockerfile atau Compose sendiri
- Database provisioning otomatis di luar Compose
- High availability / multi-server / clustering
- Public signup / registration flow
- Kubernetes / Nomad / Docker Swarm orchestration

---

## 2. Context & konfigurasi lingkungan

### 2.1 Infrastructure existing

- **Host:** VM di Proxmox server (campus lab)
- **OS:** Ubuntu Server LTS
- **Network exposure:** Cloudflare Zero Trust Tunnel
- **Domain utama:** `nabilrizkinavisa.me`
- **DNS provider:** Cloudflare (domain sudah pakai Cloudflare nameserver)
- **VM specs minimum:** 4 vCPU, 8GB RAM, 80GB SSD

### 2.2 Implikasi Cloudflare Zero Trust

Karena VM di-expose via Cloudflare Tunnel, beberapa hal dihandle Cloudflare:

- SSL/TLS termination di level Cloudflare
- DDoS protection
- IP VM tidak exposed ke publik

Konsekuensinya untuk MyPaas:

- Tidak perlu Let's Encrypt di server (Cloudflare SSL cukup)
- Port 80/443 tidak perlu open ke publik
- Wildcard subdomain `*.nabilrizkinavisa.me` dikelola via Cloudflare Tunnel wildcard hostname
- Internal reverse proxy (Caddy) hanya untuk routing subdomain → service port internal

### 2.3 Stack keputusan

| Layer | Pilihan | Alasan |
|---|---|---|
| Backend | Go 1.22+ | RAM efisien (~30-50MB), cocok untuk VM 8GB, binary tunggal mudah deploy |
| HTTP router | Chi | Stdlib-compatible, tidak vendor lock-in, mature |
| Database | PostgreSQL 16 | Battle-tested, fitur lengkap |
| DB driver | pgx v5 | Performa tinggi, native PostgreSQL features |
| Query generator | sqlc | Type-safe SQL, zero runtime overhead, predictable |
| Migration | golang-migrate | Simple, CLI yang bisa dipakai manual |
| Frontend | SvelteKit | Bundle kecil, performa realtime baik |
| Container runtime | Docker + Docker Compose | Support multi-service per project |
| Internal reverse proxy | Caddy 2 | Config simple, dynamic reload via Admin API |
| Public exposure | Cloudflare Zero Trust Tunnel | Existing infra, wildcard support |
| Auth | GitHub OAuth 2.0 | Natural integration dengan webhook, tidak perlu Azure credential |

---

## 3. Architecture overview

### 3.1 Topology dalam satu VM

```
Internet
   ↓
Cloudflare (SSL + Zero Trust Tunnel)
   ↓
VM Ubuntu
   ├── Caddy (:80) — internal reverse proxy
   │
   ├── MyPaas Master
   │     ├── Go API binary (:8080)
   │     └── SvelteKit Dashboard (:3000)
   │
   ├── PostgreSQL (:5432)
   │
   └── User Project Containers
         ├── Project A (Dockerfile mode)
         │     └── app container (:3001 → internal :3000)
         │
         └── Project B (Compose mode)
               ├── app container (:3002 → internal :3000)
               ├── redis container (internal network only)
               └── postgres container (internal network only)
```

### 3.2 Request flows

**Dashboard access:**
```
user → dashboard.nabilrizkinavisa.me → Cloudflare → Tunnel → Caddy → Dashboard :3000
```

**API access:**
```
dashboard → api.nabilrizkinavisa.me → Cloudflare → Tunnel → Caddy → Go API :8080
```

**Deployed project access:**
```
user → {projectName}.nabilrizkinavisa.me → Cloudflare → Tunnel → Caddy → container :PORT
```

**Webhook:**
```
GitHub push → webhook.nabilrizkinavisa.me/{projectId} → Caddy → Go API → enqueue deploy
```

### 3.3 Subdomain allocation

- `dashboard.nabilrizkinavisa.me` — MyPaas dashboard UI
- `api.nabilrizkinavisa.me` — MyPaas Go API
- `webhook.nabilrizkinavisa.me` — GitHub webhook endpoint
- `{projectName}.nabilrizkinavisa.me` — user project subdomain (dynamic)

Wildcard hostname `*.nabilrizkinavisa.me` di Cloudflare Tunnel mengarah ke `localhost:80` (Caddy) di VM.

---

## 4. Functional requirements

### 4.1 Authentication

**FR-AUTH-1** Login dengan GitHub OAuth 2.0.
**FR-AUTH-2** Setelah OAuth callback, email user dicek ke tabel `users`. Jika tidak ada, login ditolak (HTTP 403).
**FR-AUTH-3** Session dikelola via JWT (access token dengan expiry 24 jam, refresh token 30 hari).
**FR-AUTH-4** Tidak ada flow register / signup di UI.
**FR-AUTH-5** Penambahan user dilakukan manual via SQL insert ke tabel `users` oleh owner, atau via CLI tool MyPaas.
**FR-AUTH-6** Semua endpoint API selain `/auth/*` dan `/webhook/*` harus terautentikasi.
**FR-AUTH-7** JWT secret di-rotate setiap 90 hari (semi-manual: owner regenerate secret, user re-login).

### 4.2 Project management

**FR-PROJ-1** User dapat membuat project baru dengan input:
- Nama project (alphanumeric + dash, 3-30 chars, harus unique)
- Git repository URL (HTTPS atau SSH)
- Branch yang di-deploy (default: `main`)
- Deploy mode: auto-detect (Dockerfile vs Compose)
- Service name utama (wajib diisi jika mode Compose)
- App port (port service utama yang akan di-proxy Caddy)

**FR-PROJ-2** User dapat melihat daftar semua project dengan informasi: nama, subdomain, status deployment, last deployed at, deploy mode (Dockerfile/Compose).
**FR-PROJ-3** User dapat melihat detail project termasuk history deployment lengkap.
**FR-PROJ-4** User dapat menghapus project. Penghapusan termasuk:
- Stop & remove semua container (Dockerfile: 1 container, Compose: N container)
- Remove volume terkait
- Remove Caddy entry
- Release port ke pool
- Hapus build workspace
- Soft delete record di DB

**FR-PROJ-5** User dapat rename project (subdomain otomatis berubah, Caddy config di-update).
**FR-PROJ-6** User dapat mengedit environment variables per project.
**FR-PROJ-7** Environment variables disimpan encrypted at rest di PostgreSQL (AES-256-GCM).
**FR-PROJ-8** Saat create project atau detect repo, MyPaas membaca template env dari repo dan menyiapkan draft environment variable:
- File yang discan: `.env.example`, `.env.sample`, `.env.template`, `.env.local.example`
- Compose interpolation yang discan: `${VAR}`, `${VAR:-default}`, `${VAR-default}`, `${VAR:?message}` di `docker-compose.yml` atau `compose.yml`
- MyPaas hanya membuat daftar key sebagai draft; value baru disimpan setelah user submit
- Key sensitif (`SECRET`, `TOKEN`, `PASSWORD`, `PASS`, `KEY`, `DATABASE_URL`, `DSN`, `PRIVATE`) wajib masked by default dan tidak boleh auto-copy sample value tanpa explicit user action
**FR-PROJ-9** New Project form memiliki step Environment untuk hasil discovery:
- User tinggal mengisi value untuk key yang ditemukan
- User dapat menambah, menghapus, atau mengubah key sebelum project dibuat
- Jika shared PostgreSQL dipilih, `DATABASE_URL` auto-generated dan ditandai sebagai managed value
- Jika Compose membawa bundled DB, MyPaas menampilkan catatan bahwa credential Postgres hanya berlaku saat volume DB pertama kali dibuat

### 4.3 Deployment engine

**FR-DEPLOY-1** Deployment support tiga mode:
- **Dockerfile mode:** repo memiliki `Dockerfile` di root
- **Compose mode:** repo memiliki `docker-compose.yml` atau `compose.yml` di root
- **Static mode:** repo memiliki output static dengan `index.html` di `dist`, `build`, `public`, atau root; diserve langsung oleh Caddy tanpa container app

**FR-DEPLOY-2** Auto-detection priority: Compose file > Dockerfile > Static output. Jika beberapa ada, prioritas tertinggi dipakai.
**FR-DEPLOY-3** Jika tidak ada Dockerfile, Compose file, maupun static `index.html`, deployment gagal dengan error message yang jelas.
**FR-DEPLOY-4** Flow deployment Dockerfile mode:
1. Terima trigger (manual deploy / webhook)
2. Enqueue deployment job (async)
3. Clone repo ke workspace `/tmp/mypaas/builds/{deploymentId}`
4. Validasi adanya Dockerfile
5. Generate `.env` file dari env vars yang ter-encrypted di DB
6. Build image: `docker build -t mypaas/{projectName}:{commitSha} .`
7. Jika container lama ada, stop gracefully (SIGTERM, max 30s) lalu remove
8. Allocate port dari port pool
9. Start container: `docker run -d --name mypaas-{projectName} -p 127.0.0.1:{allocatedPort}:{appPort} --memory 512m --cpus 0.5 --env-file .env --restart unless-stopped mypaas/{projectName}:{commitSha}`
10. Update Caddy config untuk subdomain → port mapping
11. Reload Caddy via Admin API
12. Update status deployment ke `running`
13. Cleanup build workspace

**FR-DEPLOY-5** Flow deployment Compose mode:
1. Terima trigger
2. Enqueue deployment job (async)
3. Clone repo ke workspace
4. Validasi adanya Compose file + validasi service utama yang dideklarasikan di form ada di Compose
5. Generate `.env` file di workspace
6. Generate `docker-compose.override.yml` dengan resource limits per service
7. Jika compose project lama running, `docker compose -p mypaas-{projectName} down`
8. Allocate port untuk service utama
9. Inject port mapping ke service utama via override
10. `docker compose -p mypaas-{projectName} up -d --build`
11. Update Caddy config → subdomain ke port yang di-allocate
12. Reload Caddy
13. Update status ke `running`
14. Cleanup build workspace (tapi simpan path untuk compose logs)

**FR-DEPLOY-5A** Flow deployment Static mode:
1. Terima trigger
2. Enqueue deployment job (async)
3. Clone repo ke workspace
4. Validasi static output memiliki `index.html` di `dist`, `build`, `public`, atau root
5. Copy static files ke `/var/lib/mypaas/static/{projectId}` secara atomic
6. Update Caddy config untuk subdomain -> file server root static project
7. Update status ke `running`
8. Cleanup build workspace

**FR-DEPLOY-6** Deployment async — request API tidak blocking, return 202 + deploymentId. Status tracked via SSE.
**FR-DEPLOY-7** Build queue: deployment di-serialize **per project** (satu project tidak bisa 2 build bareng). Project berbeda bisa parallel maksimum 2 concurrent build (CPU limit).
**FR-DEPLOY-8** Build log di-capture dan accessible realtime via SSE + history di DB.
**FR-DEPLOY-9** Deployment yang gagal tidak boleh mengganggu container yang sedang running (atomic switch: hanya swap container kalau build sukses).
**FR-DEPLOY-10** Build timeout: maksimal 15 menit. Kalau melebihi, proses di-kill dan status → `failed`.
**FR-DEPLOY-11** Image yang lama (tidak dipakai container running) dibersihkan otomatis seminggu sekali via scheduled job.

### 4.4 Deployment history & rollback

**FR-HIST-1** Setiap deployment tersimpan di DB dengan: commit SHA, status, triggered_by (manual/webhook), started_at, finished_at, build log, error message.
**FR-HIST-2** User dapat melihat history deployment dengan detail lengkap.
**FR-HIST-3** User dapat **rollback** ke deployment sukses sebelumnya:
- Image sebelumnya tetap ada di registry lokal
- Rollback = stop container sekarang, start container dari image lama
- Tidak perlu re-build
**FR-HIST-4** Retention: simpan maksimal 20 deployment terakhir per project. Deployment lama di-cleanup (termasuk image-nya jika bukan active).

### 4.5 Webhook

**FR-WEBHOOK-1** Setiap project generate webhook URL unik: `https://webhook.nabilrizkinavisa.me/{projectId}`.
**FR-WEBHOOK-2** Setiap project punya webhook secret unik yang di-generate otomatis (32 bytes random).
**FR-WEBHOOK-3** Go API verifikasi `X-Hub-Signature-256` header menggunakan HMAC SHA-256 dengan constant-time comparison (`hmac.Equal`).
**FR-WEBHOOK-4** Hanya event `push` di branch yang di-configure yang trigger redeploy.
**FR-WEBHOOK-5** Webhook yang signature-nya tidak valid di-reject dengan 401 dan di-log untuk audit.
**FR-WEBHOOK-6** Rate limit: maksimal 10 webhook per menit per project (prevent abuse).
**FR-WEBHOOK-7** Webhook di-queue dengan deduplication: jika ada deploy pending untuk project yang sama, webhook baru replace yang pending (cancel old, run new).

### 4.6 Subdomain & routing

**FR-ROUTE-1** Subdomain project di-generate dari nama project: `{projectName}.nabilrizkinavisa.me`.
**FR-ROUTE-2** Caddy config per project di-manage oleh Go API via Caddy Admin API (`localhost:2019`).
**FR-ROUTE-3** Saat project dideploy pertama kali, Caddy config ditambahkan dan di-reload.
**FR-ROUTE-4** Saat project dihapus, Caddy config di-remove.
**FR-ROUTE-5** Saat project di-rename, Caddy config lama di-remove, yang baru di-add.
**FR-ROUTE-6** Port internal tidak pernah di-expose ke Cloudflare Tunnel — hanya Caddy port 80 yang di-expose.

### 4.7 Container lifecycle

**FR-LIFE-1** Container user dijalankan dengan restart policy `unless-stopped`.
**FR-LIFE-2** User dapat melakukan start, stop, restart, redeploy dari dashboard.
**FR-LIFE-3** Auto-restart handled oleh Docker native.
**FR-LIFE-4** Port registry di PostgreSQL track port yang terpakai:
- Pool range: `3001-9999`
- Reserved: `80, 443, 8080, 5432, 22, 3000, 2019`
**FR-LIFE-5** Setiap container (Dockerfile mode) atau service utama (Compose mode) dijalankan dengan resource limit default:
- Memory: 512MB (customizable per project)
- CPU: 0.5 core (customizable per project)
**FR-LIFE-6** Compose mode: service selain service utama (redis, postgres, dll) punya default limit lebih kecil:
- Memory: 256MB
- CPU: 0.25 core
**FR-LIFE-7** Setiap project memiliki volume persistent di `/var/lib/mypaas/volumes/{projectId}` (untuk Compose mode, ini di-mount ke semua service yang butuh).
**FR-LIFE-8** Untuk Compose mode, MyPaas mendeteksi Docker volume/network/container existing dengan compose project name yang sama sebelum deploy pertama atau recreate:
- Jika volume existing ditemukan untuk project yang baru dibuat dari DB kosong/reset, dashboard menampilkan warning stale resource
- User dapat memilih reset project volumes sebelum deploy untuk menghindari credential lama, schema lama, atau data test lama
- Reset volume adalah action eksplisit dan tidak dilakukan diam-diam
**FR-LIFE-9** Jika deploy/runtime gagal dengan indikasi umum stale DB volume (contoh: `password authentication failed`, missing role/database, migration schema mismatch), dashboard menampilkan hint troubleshooting dan action menuju reset project volumes/logs.

### 4.8 Resource management & quota

**FR-QUOTA-1** Total resource limit per user (owner) configurable di `.env`:
- Max total memory: default 6GB (untuk VM 8GB)
- Max total CPU: default 3 core (untuk VM 4 vCPU)
- Max projects: default 20
**FR-QUOTA-2** Saat deploy, MyPaas cek apakah allocation baru masih di bawah quota. Kalau melebihi → reject dengan 409 Conflict.
**FR-QUOTA-3** User dapat customize memory/CPU limit per project via UI (dalam batas quota).
**FR-QUOTA-4** MyPaas wajib punya resource profile sebelum production deploy ke VM:
- Static/no-runtime: 64MB RAM, 0.10 CPU
- Go app kecil: 128MB RAM, 0.20 CPU
- Node/Python app: 256MB RAM, 0.35 CPU
- Compose main service: 256MB RAM, 0.35 CPU
- Compose DB/cache side service: 256MB RAM, 0.25 CPU
**FR-QUOTA-5** New Project form harus memilih/menyarankan profile berdasarkan deploy mode detection, dan user tetap bisa override limit dalam quota.
**FR-QUOTA-6** Dashboard quota harus membedakan configured limit vs real runtime usage dari Docker Stats supaya user tidak mengira semua limit langsung dipakai RAM fisik.
**FR-QUOTA-7** Shared database provisioning menjadi pre-production goal: MyPaas menyediakan satu PostgreSQL service platform, lalu dapat membuat database/user per project dan inject `DATABASE_URL`, agar project CRUD kecil tidak wajib membawa container Postgres sendiri.
Implementasi minimal pre-deploy:
- User opt-in saat create project.
- MyPaas membuat database dan role PostgreSQL deterministic per project.
- `DATABASE_URL` disimpan sebagai env var terenkripsi.
- Project container bergabung ke `PROJECT_NETWORK` supaya host `postgres` tetap privat di Docker network platform.
**FR-QUOTA-8** Static no-container hosting menjadi pre-production goal: project static/build-output dapat diserve langsung oleh Caddy dari filesystem, tanpa container nginx per project.
**FR-QUOTA-9** Idle sleep / wake-on-request menjadi staged optimization setelah resource profiles + shared DB stabil: project yang idle dapat distop otomatis dan dibangunkan saat request pertama, dengan status/cold-start state yang jelas di dashboard.
**FR-QUOTA-10** Autosizing berbasis metrics dimulai sebagai rekomendasi, bukan auto-enforcement: MyPaas membaca p95 memory/CPU aktual dan menyarankan limit lebih kecil/besar sebelum user apply.

### 4.9 Monitoring & observability

**FR-MON-1** Untuk setiap container running, dashboard menampilkan:
- CPU usage (%)
- Memory usage (MB / limit)
- Status (running / stopped / crashed / building / queued)
- Uptime (sejak container last started)
**FR-MON-2** Compose mode: tampilkan metrics per service, bukan aggregate.
**FR-MON-3** Data metrics di-stream via **SSE** ke dashboard, interval 5 detik.
**FR-MON-4** Data metrics diambil dari Docker Stats API.
**FR-MON-5** Container logs di-stream via **SSE** ke dashboard (typed event `log`).
**FR-MON-6** Compose mode: logs multi-service di-merge dengan prefix service name.
**FR-MON-7** Log history (maksimal 5000 baris terakhir) dapat di-load via REST endpoint dengan pagination.
**FR-MON-8** Build logs disimpan di PostgreSQL per deployment record.
**FR-MON-9** MyPaas expose own metrics di `/metrics` (Prometheus format) — future-proof untuk integrasi Grafana kalau diperlukan.

### 4.10 Unified SSE stream

**FR-SSE-1** Satu SSE endpoint per project: `GET /projects/{id}/stream`.
**FR-SSE-2** Server kirim typed events:
```
event: metrics
data: {"cpu":12.5,"memoryMb":234,"memoryLimitMb":512,"service":"app"}

event: log
data: {"service":"app","line":"[INFO] Server started","timestamp":"..."}

event: deployment
data: {"deploymentId":"...","status":"building","step":"Building Docker image..."}

event: status
data: {"status":"running","uptime":"2h 14m"}
```
**FR-SSE-3** Client subscribe sekali, filter event type di client side.
**FR-SSE-4** Connection idle timeout: 5 menit — kirim heartbeat tiap 30 detik.
**FR-SSE-5** Server close connection kalau project dihapus.

### 4.11 Dashboard UI

**FR-UI-1** Halaman login: tombol "Login with GitHub".
**FR-UI-2** Halaman dashboard utama: grid/list semua project dengan status indicator + quick action (restart, stop).
**FR-UI-3** Halaman "New Project": form multi-step atau single-page dengan auto-detect mode preview.
**FR-UI-4** Halaman project detail dengan tab:
- **Overview:** status, subdomain, last deployed, commit hash, uptime, action buttons (Deploy, Start, Stop, Restart)
- **Deployments:** history table dengan rollback button per entry
- **Logs:** realtime log stream + history + filter text + service selector (Compose mode)
- **Metrics:** CPU + memory graph per service (realtime)
- **Environment:** table env variables (key, value editable, mask by default)
- **Settings:** rename, change branch, change resource limits, webhook URL, webhook secret (regenerate), delete project (dengan konfirmasi)
**FR-UI-5** Dashboard harus responsive di desktop (1280px+) minimal.
**FR-UI-6** Mobile dashboard: read-only mode (view status, logs), actions disabled.
**FR-UI-7** Empty state: jika belum ada project, CTA langsung ke "New Project" dengan tutorial singkat.
**FR-UI-8** Global notification system: toast untuk sukses/error action, persistent banner untuk deployment ongoing.
**FR-UI-9** Dark mode support (toggle di navbar).
**FR-UI-10** Semua action button yang memicu request async wajib punya loading state yang terasa jelas:
- Button disable selama request in-flight
- Spinner inline atau progress affordance visual di dalam button
- Label berubah sesuai aksi (`Deploying...`, `Saving...`, `Deleting...`, dll)
- Double-click tidak boleh menghasilkan duplicate request
**FR-UI-11** Navigasi dashboard dan tab project harus terasa client-side/SPA-like:
- Tab project tidak boleh membuat header/layout utama blank atau flash ke loading penuh saat berpindah tab
- Data tab boleh refetch, tapi shell project, navbar, dan tab bar tetap stabil
- Navigasi internal pakai SvelteKit client navigation, bukan `location.href` atau full page reload kecuali untuk external URL/OAuth
- Data tab yang baru dibuka menampilkan skeleton/inline loading di area konten tab saja

### 4.12 CLI tool (optional tapi recommended)

**FR-CLI-1** CLI binary `mypaas` untuk operasi admin tanpa buka UI:
- `mypaas user add <email>` — add user ke whitelist
- `mypaas project list` — list semua project
- `mypaas project deploy <name>` — trigger deploy
- `mypaas project logs <name>` — tail logs
- `mypaas backup` — backup database manual
**FR-CLI-2** CLI communicate ke Go API lewat JWT (token disimpan di `~/.mypaas/config.yml`).

---

## 5. Non-functional requirements

### 5.1 Performance

**NFR-PERF-1** Dashboard page load < 2 detik.
**NFR-PERF-2** API response time p95 < 200ms (non-streaming endpoint).
**NFR-PERF-3** Deploy trigger → container running < 5 menit untuk image ukuran wajar (<500MB).
**NFR-PERF-4** Metrics SSE latency < 2 detik dari Docker Stats.
**NFR-PERF-5** Log streaming latency < 1 detik dari container stdout.
**NFR-PERF-6** MyPaas Go binary RAM idle < 80MB.
**NFR-PERF-7** Pre-production resource efficiency target: 5 sample projects pada VM 8GB tidak boleh menghabiskan lebih dari 2GB configured app memory limit di luar database platform MyPaas.
**NFR-PERF-8** Static project idle RAM target mendekati 0MB app-container RAM jika static no-container hosting aktif.

### 5.2 Reliability

**NFR-REL-1** Jika MyPaas process restart, state project harus recover dari PostgreSQL.
**NFR-REL-2** Jika container user crash, Docker auto-restart (max 5 kali dalam 1 menit, lalu stop).
**NFR-REL-3** Failed deployment tidak mengganggu container yang sedang running (blue-green style).
**NFR-REL-4** PostgreSQL backup otomatis harian ke filesystem + weekly ke Cloudflare R2 (kalau dikonfigurasi).
**NFR-REL-5** Graceful shutdown: SIGTERM → finish in-flight request → close DB → exit (max 30 detik).
**NFR-REL-6** Health check endpoint `/health` dan `/ready` untuk orchestration/monitoring tool.

### 5.3 Security

**NFR-SEC-1** Semua endpoint API (kecuali /auth, /webhook, /health) pakai JWT bearer token.
**NFR-SEC-2** Environment variables encrypted at rest (AES-256-GCM dengan key rotation support).
**NFR-SEC-3** Webhook signature verification wajib dengan constant-time comparison.
**NFR-SEC-4** Container user tidak punya akses ke Docker socket (kecuali jika user sendiri request dengan warning eksplisit).
**NFR-SEC-5** Port container di-bind ke `127.0.0.1`, bukan `0.0.0.0` (Caddy yang proxy).
**NFR-SEC-6** Rate limiting di level Caddy:
- Login endpoint: 5 req/menit per IP
- Webhook: 10 req/menit per project
- API umum: 100 req/menit per user
**NFR-SEC-7** CORS strict: hanya allow `dashboard.nabilrizkinavisa.me`.
**NFR-SEC-8** Secret di log di-sanitize (masking otomatis).
**NFR-SEC-9** SQL parameterized via sqlc (zero chance SQL injection).

### 5.4 Maintainability

**NFR-MAIN-1** Kode mengikuti Go idiomatic patterns + `gofmt` + `golangci-lint`.
**NFR-MAIN-2** Setiap package punya README singkat yang menjelaskan purpose & public API.
**NFR-MAIN-3** Test coverage minimum 70% di domain layer, 60% overall.
**NFR-MAIN-4** CI/CD via GitHub Actions: test, lint, build, push image.

### 5.5 Usability

**NFR-USE-1** UI mengikuti filosofi Vercel/Railway — clean, minimal, konteks per project jelas.
**NFR-USE-2** Error messages informatif dan actionable (mention next step).
**NFR-USE-3** Deployment progress visible dengan breakdown step (cloning → building → starting → running).
**NFR-USE-4** Keyboard shortcuts untuk power user: `g d` (dashboard), `g n` (new project), `/` (search).
**NFR-USE-5** Perceived latency untuk aksi dashboard harus rendah: user mendapat feedback visual dalam <100ms setelah klik, walaupun request backend masih berjalan.
**NFR-USE-6** Project detail navigation harus mempertahankan konteks visual: pergantian tab tidak boleh terasa seperti reload halaman penuh pada koneksi normal.

---

## 6. Data model

### 6.1 Schema PostgreSQL

```sql
-- User whitelist
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    github_id       VARCHAR(50),
    github_username VARCHAR(100),
    avatar_url      TEXT,
    role            VARCHAR(20) NOT NULL DEFAULT 'owner', -- owner, collaborator
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMP
);

-- Project metadata
CREATE TABLE projects (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id),
    name              VARCHAR(30) UNIQUE NOT NULL,
    repo_url          TEXT NOT NULL,
    branch            VARCHAR(100) NOT NULL DEFAULT 'main',
    subdomain         VARCHAR(100) UNIQUE NOT NULL,
    deploy_mode       VARCHAR(20) NOT NULL, -- 'dockerfile' | 'compose' | 'static'
    main_service      VARCHAR(100), -- for compose mode
    app_port          INT NOT NULL,
    webhook_secret    TEXT NOT NULL,
    allocated_port    INT UNIQUE,
    memory_limit_mb   INT NOT NULL DEFAULT 512,
    cpu_limit         NUMERIC(3,2) NOT NULL DEFAULT 0.5,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, stopped, crashed, building
    active_deployment_id UUID,
    created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMP
);

CREATE INDEX idx_projects_user_id ON projects(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_status ON projects(status) WHERE deleted_at IS NULL;

-- Deployment history
CREATE TABLE deployments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id),
    commit_sha    VARCHAR(40),
    commit_message TEXT,
    status        VARCHAR(20) NOT NULL, -- queued, cloning, building, starting, running, failed, stopped, rolled_back
    build_log     TEXT,
    error_msg     TEXT,
    image_tag     VARCHAR(255), -- for rollback
    triggered_by  VARCHAR(20) NOT NULL, -- manual, webhook, rollback
    triggered_by_user_id UUID REFERENCES users(id),
    started_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    finished_at   TIMESTAMP
);

CREATE INDEX idx_deployments_project_id ON deployments(project_id);
CREATE INDEX idx_deployments_status ON deployments(status);

-- Environment variables (encrypted)
CREATE TABLE env_vars (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id),
    key           VARCHAR(100) NOT NULL,
    value_encrypted TEXT NOT NULL,
    value_nonce   TEXT NOT NULL, -- for AES-GCM
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, key)
);

-- Port registry
CREATE TABLE port_registry (
    port          INT PRIMARY KEY,
    project_id    UUID REFERENCES projects(id),
    status        VARCHAR(20) NOT NULL DEFAULT 'available', -- available, in_use, reserved
    assigned_at   TIMESTAMP
);

-- Audit log
CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    action       VARCHAR(100) NOT NULL, -- project.created, project.deleted, deployment.triggered, dll
    resource_type VARCHAR(50),
    resource_id  UUID,
    metadata     JSONB,
    ip_address   INET,
    user_agent   TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Webhook delivery log (untuk debugging)
CREATE TABLE webhook_deliveries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id),
    github_delivery_id VARCHAR(100),
    signature_valid BOOLEAN NOT NULL,
    event_type    VARCHAR(50),
    branch        VARCHAR(100),
    processed     BOOLEAN NOT NULL DEFAULT FALSE,
    deployment_id UUID REFERENCES deployments(id),
    received_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 6.2 Port pool seed

Saat first run via migration, insert port `3001` sampai `9999` dengan status `available` ke `port_registry`.

---

## 7. API contract

### 7.1 Authentication

```
GET    /auth/github/login         — initiate OAuth flow (redirect to GitHub)
GET    /auth/github/callback      — OAuth callback, return JWT
POST   /auth/refresh              — refresh access token dengan refresh token
POST   /auth/logout               — invalidate tokens
GET    /auth/me                   — current user info
```

### 7.2 Project management

```
GET    /projects                  — list projects
POST   /projects                  — create project
GET    /projects/:id              — project detail
PATCH  /projects/:id              — update (rename, branch, resource limits)
DELETE /projects/:id              — delete project

POST   /projects/:id/deploy       — trigger manual deploy
POST   /projects/:id/start        — start container(s)
POST   /projects/:id/stop         — stop container(s)
POST   /projects/:id/restart      — restart container(s)
POST   /projects/:id/webhook-secret/regenerate — regenerate webhook secret
```

### 7.3 Deployments & rollback

```
GET    /projects/:id/deployments         — deployment history (paginated)
GET    /deployments/:id                  — deployment detail
POST   /deployments/:id/rollback         — rollback to this deployment
```

### 7.4 Environment variables

```
GET    /projects/:id/env                 — list env vars (value masked)
PUT    /projects/:id/env                 — bulk update env vars
DELETE /projects/:id/env/:key            — delete single env var
```

### 7.5 Logs & metrics

```
GET    /projects/:id/stream              — unified SSE stream
GET    /projects/:id/logs                — log history (paginated, filter by service)
```

### 7.6 Public endpoints

```
POST   /webhook/:projectId               — GitHub webhook (signature-verified)
GET    /health                           — liveness probe
GET    /ready                            — readiness probe
GET    /metrics                          — Prometheus metrics (requires basic auth)
```

### 7.7 Admin / user management

```
GET    /admin/users                      — list users (owner only)
POST   /admin/users                      — add user ke whitelist
DELETE /admin/users/:id                  — remove user
```

### 7.8 Response format

**Success:**
```json
{
  "data": { ... }
}
```

**Error:**
```json
{
  "error": {
    "code": "DOCKERFILE_NOT_FOUND",
    "message": "Dockerfile tidak ditemukan di root repository",
    "details": {
      "repo": "github.com/nabil/portfolio",
      "branch": "main"
    }
  }
}
```

---

## 8. User stories (prioritized)

### P0 — wajib ada

1. Owner login via GitHub OAuth dan hanya email whitelisted yang bisa masuk
2. Owner create project baru dengan repo URL, dapat subdomain otomatis
3. Owner trigger deploy manual dan project jalan di subdomain (Dockerfile mode)
4. Owner trigger deploy project Compose multi-service dengan Redis/Postgres
5. Owner lihat status container/service realtime di dashboard
6. Owner push ke GitHub → project otomatis redeploy via webhook
7. Owner lihat log container realtime (multi-service untuk Compose)
8. Owner lihat CPU dan memory usage per service realtime
9. Owner edit environment variables dan redeploy
10. Owner rollback ke deployment sebelumnya yang sukses
11. Owner delete project dan semua resource-nya dibersihkan
12. Owner customize memory/CPU limit per project
13. Owner deploy 5 sample projects dengan resource profiles hemat sebelum MyPaas production deploy ke VM

### P1 — prioritas tinggi

14. Owner rename project (subdomain ikut berubah)
15. Owner regenerate webhook secret
16. Owner lihat deployment history lengkap dengan build log
17. Audit log accessible untuk debugging
18. CLI tool untuk operasi admin
19. Dark mode UI
20. Shared PostgreSQL provisioning untuk project CRUD kecil
21. Static no-container hosting untuk project static/build output

### P2 — nice to have

22. Custom domain per project (selain subdomain default)
23. Branch preview deployment (deploy PR branch terpisah)
24. Shared environment variables antar project
25. Integration dengan Grafana via /metrics endpoint
26. Webhook delivery retry mechanism dengan backoff
27. Export project config sebagai YAML (portable)
28. Idle sleep / wake-on-request untuk app jarang diakses
29. Autosizing recommendation dari historical metrics

### P3 — future consideration

30. Team collaboration (multiple user per project)
31. Deployment notification ke Discord/Slack webhook
32. Scheduled deployment (cron-like)
33. Backup restore via UI

---

## 9. Delivery goal grouping

### 9.1 Pre-deploy goals — wajib selesai sebelum MyPaas live di VM/VPS

Pre-deploy adalah scope yang harus selesai sebelum owner menjalankan MyPaas sebagai platform live. Tujuannya membuat single-node MyPaas stabil, hemat resource, dan bisa dipakai dogfooding 5 sample project.

- Core deploy Dockerfile + Compose stabil: build, run, stop, restart, redeploy, cleanup, rollback.
- Caddy routing stabil untuk dashboard, API, webhook, dan project subdomain.
- Cloudflare Tunnel wildcard siap untuk dashboard/API/project access.
- GitHub OAuth + whitelist + refresh token siap.
- GitHub webhook receiver siap; webhook setup boleh manual via Settings selama auto-registration belum ada.
- Dashboard UX minimum siap: loading state, double-submit prevention, logs, metrics, env editor, settings, audit view.
- Resource profiles aktif; default limit tidak lagi flat 512MB untuk semua project.
- Dashboard quota membedakan configured limit dan real runtime usage.
- Static no-container hosting aktif untuk static/build-output sample.
- Shared PostgreSQL provisioning minimal aktif untuk CRUD DB sample kecil.
- Env discovery aktif untuk `.env.example`/Compose variables sehingga New Project form bisa menyiapkan key dan user tinggal mengisi value.
- Compose stale volume guard aktif, termasuk warning dan reset project volumes eksplisit untuk menghindari credential lama setelah reset DB/platform.
- 5 dogfooding sample project deploy sukses dengan configured app memory total <= 2GB di luar database platform MyPaas.
- Backup PostgreSQL lokal harian aktif.
- CLI minimal untuk add/list user, list/deploy/logs project, dan manual backup.

### 9.2 After-deploy goals — setelah MyPaas live dan stabil

After-deploy adalah improvement setelah platform sudah berjalan di VM/VPS dan dogfooding dasar lolos. Ini tidak boleh memblokir first live deploy.

- Auto-create GitHub webhook via GitHub API.
- Idle sleep / wake-on-request untuk app jarang diakses.
- Autosizing recommendation dari historical Docker Stats.
- Optional single-host replicas untuk app stateless, dengan Caddy load balancing.
- Custom domain per project.
- Branch/PR preview deployments.
- Webhook delivery retry mechanism.
- Shared env vars antar project.
- Grafana/Prometheus integration polish.
- Backup restore UI.
- Team collaboration.
- Notification integration ke Discord/Slack.

### 9.3 Explicitly out of MVP

- Kubernetes, Nomad, Docker Swarm, atau multi-node orchestration.
- Fully automatic horizontal autoscaling.
- Multi-region deploy.
- Managed database replacement. MyPaas hanya provisioning DB/user di PostgreSQL platform sendiri untuk scope awal.

---

## 10. Timeline

Detail di `docs/TIMELINE.md`. Ringkasan:

| Fase | Hari | Focus |
|---|---|---|
| Foundation | Day 1-2 | Project scaffold, DB schema, auth |
| Core Deploy | Day 3-5 | Deploy engine Dockerfile + Compose, port registry, Caddy |
| Async & Webhook | Day 6-7 | Job queue, webhook handler, build log streaming |
| Frontend | Day 8-10 | Dashboard UI lengkap |
| Observability | Day 11-12 | Unified SSE, metrics, logs, monitoring |
| Advanced Features | Day 13-14 | Rollback, rename, quota, dark mode |
| Polish & Deploy | Day 15 | Bug fix, CLI tool, self-deploy ke VM |

**Total: 15 hari kalender** (bisa compressed ke 10 hari kalau full-time, realistis 15 dengan kuliah/kegiatan lain).

---

## 11. Risks & mitigations

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Docker Compose CLI behavior berubah antar versi | Deploy tidak konsisten | Pin Docker version di VM, test compatibility di dev |
| Cloudflare Tunnel wildcard tidak support protocol tertentu | Feature terbatas | Test di Day 1, document limitation |
| Port exhaustion kalau banyak project | Deploy gagal | Port pool 7000 slot cukup untuk skala personal |
| Build memakan CPU berlebihan | VM lag | Limit concurrent build maksimal 2 |
| sqlc breaking change | Refactor query code | Pin versi, test migration saat update |
| PostgreSQL corrupt | Data loss | Daily backup + weekly offsite |
| Webhook spam dari GitHub | Service degradation | Rate limit + deduplication |
| Image storage penuh | Deploy stuck | Weekly cleanup + alert saat disk > 80% |
| Resource model flat 512MB/project membuat VM kecil cepat penuh | MyPaas tidak layak dipakai sebelum production deploy | Resource profiles, shared PostgreSQL, static no-container hosting, dan dashboard configured-vs-real usage wajib selesai sebelum VM deploy |

---

## 12. Open questions

Per status terbaru, semua sudah resolved:

- ✅ Cloudflare Tunnel wildcard — domain sudah di Cloudflare nameserver, support confirmed
- ✅ Encryption key untuk env vars — disimpan di env variable MyPaas sendiri (`ENV_ENCRYPTION_KEY`)
- ✅ Backup PostgreSQL — local daily + Cloudflare R2 weekly (free tier 10GB)
- ✅ Auth provider — GitHub OAuth (tidak perlu minta Azure credential)

---

## 13. Success criteria

Project dianggap berhasil jika:

- [ ] Owner bisa login via GitHub OAuth dengan whitelist DB
- [ ] Owner bisa deploy 5 project real berbeda:
  - [ ] 1 static site (HTML only)
  - [ ] 1 Node.js app (Dockerfile mode)
  - [ ] 1 Python FastAPI (Dockerfile mode)
  - [ ] 1 Go app (Dockerfile mode)
  - [ ] 1 Compose app dengan app + Redis + PostgreSQL
- [ ] Semua project accessible via subdomain dengan SSL
- [ ] 5 sample project memakai resource profiles hemat; configured app memory total tidak lebih dari 2GB di luar database platform MyPaas
- [ ] New Project form auto-discover env keys dari `.env.example`/Compose variables dan user bisa mengisi value sebelum deploy
- [ ] Compose app dengan bundled PostgreSQL memberi stale volume warning/reset action saat resource lama ditemukan atau auth DB gagal
- [ ] Dashboard membedakan configured memory limit dan real memory usage sehingga quota tidak membingungkan
- [ ] Push ke GitHub trigger redeploy otomatis untuk semua 5 project
- [ ] Dashboard menampilkan status, logs, dan metrics realtime untuk semua
- [ ] Semua action utama di dashboard punya spinner/loading state dan tidak bisa double-submit
- [ ] Navigasi tab project terasa client-side tanpa full-page blank/loading flash
- [ ] Rollback bekerja di minimal 2 project
- [ ] MyPaas sendiri di-deploy di VM lab dan stabil jalan 14 hari tanpa manual intervention
- [ ] CLI tool bisa add user + list project + trigger deploy
- [ ] Backup database jalan otomatis tanpa fail selama 7 hari
