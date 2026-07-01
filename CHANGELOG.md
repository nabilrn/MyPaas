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

### Changed
- Limit concurrent deployment workers using `MAX_CONCURRENT_DEPLOYS`
- Include `git` and Docker CLI in the backend production runtime image
- Dockerfile deploy and rollback now start a replacement container on a fresh port before switching Caddy and removing the previous stable container
- Manual and webhook deploy triggers now reuse an active deployment for the same project instead of creating duplicate concurrent work
- Dashboard project list now shows quota usage bars for memory, CPU, and project count

### Deprecated

### Removed

### Fixed
- Ignore the Linux Docker socket `DOCKER_HOST` value for local Windows Docker CLI calls and use the non-deprecated `docker stop --timeout` flag
- Treat missing Docker containers as empty log output instead of logging an internal server error while a project has not deployed successfully yet
- Bind Caddy Admin API inside dev/prod containers on `0.0.0.0:2019` so the API can manage routes through the published local port or Docker network
- Avoid Caddy wildcard route conflicts during dynamic project route updates and proxy deployed containers through configurable `CADDY_UPSTREAM_HOST`
- Replace Caddy route arrays with `PATCH` instead of `PUT` to avoid Admin API `key already exists: routes` conflicts
- Make Docker project port binding configurable with `DOCKER_BIND_HOST` so containerized Caddy can reach local project upstreams
- Use HTTP local project URLs in development instead of hardcoded production HTTPS domains
- Clear `allocated_port` when projects are soft-deleted so reused ports do not violate `projects_allocated_port_key`

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
