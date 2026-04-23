# Architecture — MyPaas

> Technical design, component interaction, and deployment flow.

**Status:** v1.0
**Last updated:** 2026-04-23

---

## 1. High-Level Architecture

```
┌─────────────────┐
│  Git Push       │ (GitHub webhook)
└────────┬────────┘
         │
         ▼
┌─────────────────────────┐
│  MyPaas API             │
│  - Auth & validation    │
│  - Build orchestration  │
│  - State management     │
└────────┬────────────────┘
         │
    ┌────┴─────┬──────────┬──────────┐
    ▼          ▼          ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Docker │ │ Caddy  │ │ Postgres│ │ Logs   │
│ Daemon │ │ Proxy  │ │ DB      │ │ Sink   │
└────────┘ └────────┘ └────────┘ └────────┘
    │
    ▼
┌──────────────────────┐
│ User Container       │
│ (Dockerfile/Compose) │
└──────────────────────┘
```

---

## 2. Component Details

### 2.1 MyPaas API (Go + Chi)
- **Responsibility:** Orchestrate builds, manage state, serve dashboard
- **Key flows:**
  - Validate webhook → queue deployment
  - Build & push image
  - Update Caddy routing dynamically
  - Stream logs via SSE
  - Collect metrics from containers

### 2.2 Caddy Reverse Proxy
- **Responsibility:** Route requests to user containers + dashboard
- **Dynamic routing:** Caddy Admin API (2019)
- **SSL:** Handled by Cloudflare Tunnel (no Let's Encrypt)
- **Subdomains:** `project-name.nabilrizkinavisa.me` → user container

### 2.3 PostgreSQL
- **Responsibility:** Persistent state (users, projects, env vars, deployments)
- **Migrations:** golang-migrate (versioned SQL)
- **Queries:** sqlc (type-safe, generated Go code)

### 2.4 Docker Daemon
- **Responsibility:** Build & run containers
- **Mount:** `/var/run/docker.sock` (single socket access)
- **CLI:** `docker compose` for multi-service (not SDK reimplementation)

---

## 3. Deployment Flow

```
1. GitHub push
   └─ Webhook → POST /webhook/github

2. Validation
   └─ Verify signature
   └─ Fetch repo config (Dockerfile / Compose)

3. Queue
   └─ Create deployment record
   └─ Return 202 + deploymentId (async)

4. Build (in-memory queue, max 2 concurrent)
   └─ Clone repo to /tmp/mypaas/builds/{deploymentId}
   └─ docker build -t mypaas/proj-{id}:{sha}
   └─ Write logs to SSE stream

5. Deploy
   └─ docker run (single) OR docker compose up (multi)
   └─ Allocate port from registry (3001–9999)
   └─ Update Caddy routing
   └─ Start monitoring (CPU, RAM, uptime)

6. Cleanup
   └─ Delete /tmp/mypaas/builds/{id}
   └─ Trim old images
```

---

## 4. Data Flow

### 4.1 Request Flow
```
Client → Caddy (TLS) → MyPaas API (HTTP)
                    ↓
              PostgreSQL
              Docker API
              Caddy Admin API
```

### 4.2 Log Flow
```
User Container logs
         ↓
    docker logs (streaming)
         ↓
    MyPaas log collector
         ↓
    In-memory buffer (10MB)
         ↓
    SSE → Browser → Dashboard
```

### 4.3 Metric Flow
```
User Container (cgroup stats)
         ↓
    docker stats (every 10s)
         ↓
    MyPaas metric collector
         ↓
    PostgreSQL (time-series)
         ↓
    API → Dashboard (Chart.js graph)
```

---

## 5. Key Design Decisions

### 5.1 SSE Over WebSocket
- **Why:** Simpler, one-way, auto-reconnect, lower overhead
- **Trade-off:** Can't send server→client realtime without polling

### 5.2 In-Memory Queue + DB State
- **Why:** Single-node MVP, simpler than Redis
- **Trade-off:** Lost deployments if API crashes (acceptable for MVP)
- **Future:** Migrate to Redis if needed

### 5.3 Caddy Admin API vs. nginx.conf rewrites
- **Why:** Dynamic routing without reloading
- **Trade-off:** Caddy-specific (can't swap easily)

### 5.4 sqlc vs. ORM
- **Why:** Type-safe, zero-overhead, migrations explicit
- **Trade-off:** Must write SQL manually

---

## 6. Scaling Considerations

### 6.1 Single-Node Constraints
- Max 2 concurrent deployments (configurable)
- 6 GB RAM per user (enforced)
- ~10–15 projects per user (rough limit)

### 6.2 Multi-Node Future
- Move deployment queue to Redis
- Shared PostgreSQL
- Multiple API instances behind load balancer
- Shared volume for compose workspaces

---

## 7. Security Architecture

### 7.1 Authentication
```
GitHub OAuth → JWT token (15 min) + refresh token (DB)
             → HTTP-only cookie
             → CSRF token (header)
```

### 7.2 Secrets Management
```
User-provided env vars
    ↓
AES-256-GCM encryption (key from .env)
    ↓
PostgreSQL (encrypted column)
    ↓
Inject at runtime (decrypted in memory)
```

### 7.3 Webhook Verification
```
GitHub secret → HMAC-SHA256 signature
    ↓
Header: X-Hub-Signature-256
    ↓
Verify with constant-time comparison (hmac.Equal)
```

---

## 8. Deployment Modes

### 8.1 Dockerfile Mode
```
docker build -f Dockerfile -t mypaas/proj-{id}:{sha} .
docker run \
  --name mypaas-proj-{id} \
  --memory 512m --cpus 0.5 \
  -p 127.0.0.1:{port}:3000 \
  mypaas/proj-{id}:{sha}
```

### 8.2 Docker Compose Mode
```
docker compose -f docker-compose.yml up -d
(all services on internal network)
(main service exposed via port)
```

---

## 9. Monitoring & Observability

### 9.1 Application Logs
- **Format:** JSON (slog)
- **Level:** DEBUG, INFO, WARN, ERROR
- **Fields:** projectId, deploymentId, userId (traceability)

### 9.2 Deployment Logs
- **Source:** docker logs (streaming)
- **Storage:** In-memory buffer + PostgreSQL (truncated)
- **Access:** Real-time SSE + historical queries

### 9.3 Metrics
- **CPU, RAM, uptime:** docker stats
- **Caddy metrics:** Caddy prometheus endpoint (future)

---

## References

- `docs/PRD.md` — Requirements & scope
- `docs/TIMELINE.md` — Implementation schedule
- `docs/adr/` — Architecture decision records
- `CLAUDE.md` — Code conventions & tech stack

---

**Maintained in:** `docs/ARCHITECTURE.md`
