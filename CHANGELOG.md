# Changelog

All notable changes to MyPaas will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure and setup
- Rollback endpoint and dashboard action for Dockerfile deployments
- Webhook secret regeneration endpoint and settings UI action
- GitHub webhook endpoint with HMAC signature verification, branch filtering, rate limiting, delivery logging, and Dockerfile deployment trigger
- Deployment startup recovery that marks interrupted queued/building deployments as failed and resets stuck project build states
- Per-user quota endpoint and enforcement for project count, configured memory, and configured CPU limits
- Dockerfile container metrics endpoint and dashboard chart for CPU, memory, and uptime

### Changed
- Limit concurrent deployment workers using `MAX_CONCURRENT_DEPLOYS`
- Include `git` and Docker CLI in the backend production runtime image
- Dockerfile deploy and rollback now start a replacement container on a fresh port before switching Caddy and removing the previous stable container
- Manual and webhook deploy triggers now reuse an active deployment for the same project instead of creating duplicate concurrent work
- Dashboard project list now shows quota usage bars for memory, CPU, and project count

### Deprecated

### Removed

### Fixed

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
