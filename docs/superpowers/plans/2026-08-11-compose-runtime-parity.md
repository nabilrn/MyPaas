# Compose Runtime Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make repository/Compose deployments apply project env vars, refresh remote images, and verify the public service is ready before MyPaas marks the deployment running.

**Architecture:** Keep deployment orchestration in `internal/deployment` and Docker/Compose process details in `internal/container`. Extend the generated main-service override with the project env file, centralize Compose-up argv construction so pull policy is testable, and add a bounded readiness waiter based on Docker inspect state/health.

**Tech Stack:** Go 1.22+, Docker Compose v2 CLI, existing MyPaas deployment/container packages.

## Global Constraints

- No new dependencies.
- Preserve Compose security sanitization and host-port ownership.
- Do not log environment values.
- Rollback must not force-update upstream floating tags.
- Update CHANGELOG.md.

---

### Task 1: Lock runtime parity with regression tests

**Files:**
- Modify: `backend/internal/deployment/service_test.go`
- Modify: `backend/internal/container/docker_test.go`

**Interfaces:**
- Consumes: existing `writeComposeOverride`, Compose option construction, Docker state parsing.
- Produces: failing tests that define env injection, pull policy, and readiness semantics.

- [ ] Add a deployment test asserting the generated main-service override includes `env_file` for the MyPaas project env file.
- [ ] Add container tests asserting normal Compose up includes `--pull always` while `NoBuild` rollback does not.
- [ ] Add table-driven readiness tests for running/no-healthcheck, healthy, unhealthy, exited, and starting states.
- [ ] Open the PR so GitHub CI runs the test-only commit and confirm backend CI fails for the expected missing behavior.

### Task 2: Inject project env and refresh Compose images

**Files:**
- Modify: `backend/internal/deployment/service.go`
- Modify: `backend/internal/container/docker.go`

**Interfaces:**
- Consumes: `envFile` already created by deployment service and `ComposeUpOptions`.
- Produces: `writeComposeOverride(..., envFile, ...)` and deterministic Compose-up arguments.

- [ ] Extend `writeComposeOverride` to accept the env file path and emit it under the main service as `env_file`.
- [ ] Pass the resolved project env file from both normal Compose deploy and rollback paths.
- [ ] Extract Compose-up argument construction into a pure helper and add `--pull always` for normal deployments only.
- [ ] Run backend CI and confirm the env/pull tests pass.

### Task 3: Gate routing on main-service readiness

**Files:**
- Modify: `backend/internal/container/docker.go`
- Modify: `backend/internal/deployment/service.go`
- Modify: `backend/internal/container/docker_test.go`

**Interfaces:**
- Produces: `WaitComposeServiceReady(ctx, projectName, service, timeout) error`.

- [ ] Add Docker inspect decoding for container running state and optional health state.
- [ ] Implement a context-aware bounded polling loop: ready when running with no healthcheck or health=`healthy`; fail on exited/dead/unhealthy; continue polling on created/restarting/starting.
- [ ] Call readiness wait after `ComposeUp` and before Caddy route update/status persistence.
- [ ] Verify backend tests pass.

### Task 4: Documentation and full verification

**Files:**
- Modify: `CHANGELOG.md`

**Interfaces:** None.

- [ ] Document Compose runtime parity fix under Unreleased/current changelog section.
- [ ] Run/observe complete GitHub CI: backend, frontend, scripts.
- [ ] Review PR diff for secrets, unrelated refactors, and rollback behavior.
