# Changelog

All notable changes to MyPaas will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure and setup
- Rollback endpoint and dashboard action for Dockerfile deployments
- Webhook secret regeneration endpoint and settings UI action
- Project app port can be edited from settings and through the update API
- GitHub webhook endpoint with HMAC signature verification, branch filtering, rate limiting, delivery logging, and Dockerfile deployment trigger
- Deployment startup recovery that marks interrupted queued/building deployments as failed and resets stuck project build states
- Per-user quota endpoint and enforcement for project count, configured memory, and configured CPU limits
- Dockerfile container metrics endpoint and dashboard chart for CPU, memory, and uptime
- Strict CORS middleware for configured dashboard origins
- Prometheus-compatible `/metrics` endpoint with optional Basic Auth credentials
- Authenticated project SSE stream endpoint at `/projects/{id}/stream` for status, metrics, logs, and deployment events
- Project Logs tab that loads recent history, streams new lines over SSE, filters by text/service, and exports visible logs
- MVP Compose deployment support for project creation, deploy, lifecycle actions, logs, metrics, cleanup, and main-service port routing
- Deploy mode detection endpoint and New Project form wiring that prefers Compose files over Dockerfile for `auto` projects
- Active webhook secret display/copy flow in project settings
- Refresh token cookie flow with 30-day refresh lifetime and frontend automatic refresh retry
- Audit log sqlc queries, authenticated mutation middleware, owner-only audit log API, and dashboard viewer
- Daily PostgreSQL backup scheduler with weekly snapshots, retention cleanup, and scoped unused MyPaas image pruning
- Dependency-free `mypaas` CLI with config, admin user, project list/deploy/logs, and manual backup commands
- MVP dogfooding sample projects for Node.js, Python FastAPI, and Go Dockerfile deployments
- Static no-container deployment mode that publishes `dist`, `build`, `public`, or root `index.html` output through Caddy file serving
- Opt-in shared PostgreSQL provisioning for new projects, creating a per-project database/user and injecting encrypted `DATABASE_URL`
- Seed owner GitHub email into the user whitelist during migration
- Env discovery for `.env.example`, `.env.sample`, `.env.template`, `.env.local.example`, and Compose `${VAR}` interpolation in the detect-mode API
- New Project Environment step for discovered/manual env vars, sensitive key masking, managed shared `DATABASE_URL`, and encrypted env persistence during create
- Compose resource audit/reset API and settings UI for clearing stale project containers, volumes, networks, routes, and allocated ports before deploy
- Per-service Compose log and metrics collection, including multi-service log filters and metrics service selection in the dashboard
- Realtime deployment build-log SSE events surfaced in the Logs tab before runtime container logs are available
- Compose rollback path for the main service using per-commit immutable image tags for buildable services and target-commit Compose config for image-only services
- Periodic Caddy route reconciliation from DB-running projects so API/Caddy restarts restore project routes automatically
- Dogfood and production verification scripts for routed sample projects, production health, Caddy Admin API, CLI presence, and optional manual backup
- VM deploy helper script that creates MyPaas host directories, runs migrations, and starts the production Compose stack
- VM install script that checks Docker/Compose, generates production secrets, prepares host storage, and runs the production deploy helper
- Detect-mode app port inference from Dockerfile `EXPOSE`/port env, Compose service ports/expose/env, and static mode defaults
- Sidebar dashboard shell with mobile navigation fallback
- Reusable dashboard pagination control for deployment history and admin tables
- Shared dashboard `TableShell` and `ErrorState` components for consistent table loading, empty, retry, and footer states
- Shared dashboard `SecretField` component for consistent environment variable hidden, revealed, dirty, copy, discard, reveal, and delete states
- Shared dashboard `DeployControlPanel` component for project status, deploy/restart/stop actions, logs access, and route/runtime metadata
- Project-local Impeccable design workflow context and live-mode config for future UI polish passes
- Environment variable `.env` paste/upload importer with preview, duplicate/invalid detection, and overwrite confirmation
- GitHub webhook setup help dialog in project settings with payload URL, secret, and event configuration guidance
- VM install browser wizard for first-time production credentials, including GitHub OAuth, Cloudflare DNS, and Tunnel route setup guidance
- New Project repository inspection with branch dropdown selection and repository structure preview before runtime detection
- Compose Doctor preflight in detect-mode with public service/port recommendation, required env detection, build context checks, host-port warnings, and unsafe Compose config flags
- Detect-mode env discovery now scans nested env example files for monorepo-style repositories without turning Dockerfile build/image defaults into project env vars
- DB Studio Lite for project databases, with PostgreSQL/MySQL/MariaDB connection discovery, schema/table browsing, paginated rows, temporary write mode, and guarded insert/update/delete by primary key

### Changed
- Limit concurrent deployment workers using `MAX_CONCURRENT_DEPLOYS`
- Settings now shows the routable `/api/webhook/{projectId}` GitHub webhook URL and clearer webhook secret copy behavior
- Caddy dev/prod config now proxies `/webhook/*` directly to the API for GitHub webhook delivery
- Project rename now updates the active Caddy route when a deployed project has an allocated port
- Project Overview now reads live metrics snapshots instead of hardcoded CPU, memory, and uptime values
- Include `git` and Docker CLI in the backend production runtime image
- Dockerfile deploy and rollback now start a replacement container on a fresh port before switching Caddy and removing the previous stable container
- Manual and webhook deploy triggers now reuse an active deployment for the same project instead of creating duplicate concurrent work
- Dashboard project list now shows quota usage bars for memory, CPU, and project count
- PRD and timeline now include dashboard UX goals for async button spinners, double-submit prevention, and SPA-like project tab navigation
- Dashboard async actions now use a shared spinner button with per-action pending state and double-submit prevention
- Frontend dashboard received a PaaS-style polish pass with denser project inventory, project command surface, refined tabs, compact status badges, neutral action system, and redesigned project settings/env/metrics/admin views
- Environment variable keys are normalized to uppercase in create/edit flows and persisted uppercase by the API
- Deployment history, audit logs, and admin users now share table shell state handling and accessible pagination controls
- Project inventory now uses the shared table shell with pagination, and environment rows use the shared secret field pattern
- Project detail pages now use the shared deploy control panel instead of page-local duplicated command and metadata markup
- New Project no longer pre-fills app port `3000`; the field now distinguishes detected, manual, static, and fallback port states
- Metrics now fetch before the chart library loads, avoid overlapping refreshes, preserve stale data on refresh failure, and show chart-specific loading/error states
- Project overview now loads project/deployment data independently from Docker metrics so slow stats collection does not block the overview
- Logs now expose separate history retry and live-stream reconnect states with stable terminal loading placeholders
- Risky dashboard actions now use inline confirmation states for rollback, webhook secret regeneration, Compose reset, and whitelist user removal
- Settings now has explicit load failure retry and inline Compose resource check errors instead of a permanent skeleton or toast-only recovery
- Dashboard visual system now uses semantic app surface tokens, a native system font stack, softer sidebar active states, consistent brand focus rings, and quieter top-level headers for a more refined PaaS control-plane feel
- New Project now fills non-sensitive discovered env defaults while keeping sensitive values blank for `.env` import or manual entry
- PRD and timeline now require a pre-VM-deploy resource efficiency gate with resource profiles, separate configured-vs-real memory reporting, shared PostgreSQL provisioning, and static no-container hosting
- PRD now groups goals into pre-deploy, after-deploy, and explicitly out-of-MVP work to keep Kubernetes/autoscaling out of the first VM deploy scope
- New Project auto detection now validates the selected branch only, auto-applies Compose Doctor recommendations, and blocks create when required Compose env values or blocking Compose issues remain unresolved
- PRD and timeline now include New Project env discovery from `.env.example`/Compose variables plus Compose stale volume warnings and explicit reset actions as pre-deploy goals
- Dashboard quota now separates configured memory/CPU allocation from best-effort live Docker Stats runtime usage
- After-deploy ADRs for idle sleep/wake-on-request, autosizing recommendations, and optional single-host replicas

### Deprecated

### Removed

### Fixed
- Ignore the Linux Docker socket `DOCKER_HOST` value for local Windows Docker CLI calls and use the non-deprecated `docker stop --timeout` flag
- Treat missing Docker containers as empty log output instead of logging an internal server error while a project has not deployed successfully yet
- Bind Caddy Admin API inside dev/prod containers on `0.0.0.0:2019` so the API can manage routes through the published local port or Docker network
- Avoid Caddy wildcard route conflicts during dynamic project route updates and proxy deployed containers through configurable `CADDY_UPSTREAM_HOST`
- Serve static projects correctly from Dockerized Caddy when the API runs on Windows by using container path separators and Caddy Admin route operations that match live API behavior
- Replace Caddy route arrays with `PATCH` instead of `PUT` to avoid Admin API `key already exists: routes` conflicts
- Make Docker project port binding configurable with `DOCKER_BIND_HOST` so containerized Caddy can reach local project upstreams
- Use HTTP local project URLs in development instead of hardcoded production HTTPS domains
- Clear `allocated_port` when projects are soft-deleted so reused ports do not violate `projects_allocated_port_key`
- Return a conflict for duplicate admin whitelist users and render users without GitHub avatars cleanly
- Route Caddy `/api/*` and `/webhook/*` with explicit `handle` blocks so dashboard fallback cannot intercept backend requests
- Backend runtime image now includes `pg_dump`, and production compose mounts `/var/lib/mypaas/backups` into the API container
- Backend build target now emits both `mypaas-api` and the `mypaas` CLI binary
- Backend production image now includes the `mypaas` CLI binary for in-container backup and verification commands
- Compose deployment override now replaces the main service `ports` list so app-local ports like `8080:8080` do not conflict with the MyPaas API
- Compose commands now use the generated project `.env` and filter MyPaas internal env vars so values like the platform `DATABASE_URL` cannot leak into deployed apps
- Project soft-delete now releases name/subdomain uniqueness via active-only unique indexes so deleted projects can be recreated with the same name
- New Project now uses a single-screen create flow instead of the four-step wizard, with detect, runtime, resources, env, and plan visible together
- Pass `PUBLIC_DOMAIN` into the production Caddy container so `Caddyfile.prod` can adapt successfully
- Added a no-op `/firebase-messaging-sw.js` static worker to quiet stale Firebase Messaging service worker probes on reused browser origins
- Project create/update now persists `resource_profile`, returns it in API responses, and the dashboard resource forms apply profile defaults instead of a flat 512MB default
- DB Studio now connects the API container to the actual Compose service network and targets the database container IP, fixing custom network database hosts like `db`
- Static projects bypass Docker lifecycle/log collection while still supporting route start/stop/restart and zero-runtime metrics snapshots
- Dockerfile containers and Compose main services can join `PROJECT_NETWORK` so shared platform services remain private on the Docker network
- Compose deploys now warn in build logs when Docker resources exist before the first tracked active deployment
- Production API Docker build now uses Go 1.23 to match the current module dependency floor
- Backend and frontend Docker builds now ignore local artifacts such as `node_modules`, Svelte build output, and host binaries so containers can be recreated cleanly
- Encrypted environment variables can be revealed through an authenticated decrypt endpoint, with 404 handling for missing keys
- Sidebar navigation now keeps the active menu item highlighted across nested project and admin routes
- Dashboard P0 UX states now handle deployment load failures, env var load failures/empty state, admin user load failures/empty state, env overwrite drafts, and New Project env-key Enter behavior
- Project detail header status now follows the project SSE stream so deployment completion appears without polling or manual reload

### Security

---

## [0.1.0] - 2026-04-23

### Added
- Initial project setup
- Directory structure
- Makefile with development targets
- Docker Compose configurations (dev & prod)
- Caddyfile for reverse proxy
- Environment configuration templates
- GitHub Actions workflows placeholder
- Project documentation structure

[Unreleased]: https://github.com/nabilrizkinavisa/mypaas/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/nabilrizkinavisa/mypaas/releases/tag/v0.1.0
