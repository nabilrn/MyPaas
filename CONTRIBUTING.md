# Contributing to MyPaaS

Thanks for contributing to MyPaaS.

## Ground rules

- Be respectful and constructive.
- Check existing issues/PRs before opening duplicate work.
- MyPaaS is intentionally a **single-host self-hosted PaaS** for an owner developer or small trusted team.
- Read `AGENTS.md`, `PRODUCT.md`, `ROADMAP.md`, and current architecture/security docs before proposing major platform changes.
- `docs/PRD.md` is historical and must not be treated as current product requirements.

Proposals for Kubernetes/Nomad/Swarm, distributed scheduling, automatic horizontal scaling, hostile multi-tenant isolation, or broad performance programs do not match the current product direction unless the direction is explicitly changed first.

## Opening issues

### Bug reports

Include:

- what failed;
- minimal reproduction steps;
- server OS/runtime (Podman or Docker compatibility mode);
- MyPaaS version/Git SHA when known;
- relevant secret-safe logs;
- expected vs observed behavior.

Never include tokens, cookies, passwords, registry credentials, decrypted environment values, or production `.env` contents.

### Feature requests

Describe the real application/operator problem first.

A good feature request explains:

- the workload or workflow that is blocked;
- why the existing Dockerfile/Compose/static/OCI primitives cannot handle it cleanly;
- the smallest reusable platform capability that would solve it;
- security/lifecycle implications.

Compatibility failures should be classified before becoming feature requests. A workload-specific upstream/configuration issue or host-resource limit is not automatically a MyPaaS feature gap.

## Submitting pull requests

1. Create a narrow branch from current `main` using the domain naming rules in `docs/engineering/branching.md`.
2. Read the existing implementation before editing architecture.
3. Keep the PR to one domain + one outcome.
4. Add/update targeted regression coverage for changed behavior.
5. Run checks proportional to the change.
6. Update the relevant current docs/ADR/compatibility record and `CHANGELOG.md` when product behavior changes.
7. Describe what changed, what was verified, and any intentional limitation in the PR.

Do not mix unrelated cleanup, benchmarks, redesigns, or speculative features into a defect fix.

## Current technical conventions

Detailed engineering rules live in `AGENTS.md`. Important high-level constraints include:

- backend: Go + Chi + pgx/sqlc;
- frontend: SvelteKit + TypeScript + pnpm;
- container orchestration: Docker-compatible CLI/socket contract;
- fresh supported Linux hosts: rootful Podman default, Docker Engine compatibility mode;
- streaming: existing SSE model;
- Caddy: project HTTP data plane + Unix-socket Admin API in production;
- Compose input is untrusted and must continue through the existing sanitization/validation boundary.

Do not add another deployment engine for templates or compatibility fixtures.

## Testing policy

Run tests that cover the behavior you changed.

Examples:

- Go/source behavior -> relevant Go tests, race checks when concurrency-sensitive;
- frontend behavior -> relevant Vitest/check/build;
- installer/runtime integration -> script regression + production Compose/Podman compatibility checks;
- routing lifecycle change -> targeted route/lifecycle tests and, when material, the affected real-VM qualification path;
- OSS compatibility fix -> rerun the affected application path.

Do not repeat broad k6/performance/resource-pressure matrices after unrelated changes.

A compatibility `PASS` is a correctness result for the declared workload scenario, not a throughput or server-capacity certification.

## Branch flow

- `main` is the accepted current product state.
- Normal work happens on a narrow branch and is reviewed through a PR to `main`.
- Use a disposable/temporary integration branch only when a specific validation genuinely requires it; there is no permanent staging-first requirement.
- Delete merged branches when practical.
- Start new work from updated `main`, not from an old experiment/qualification branch.

See `docs/engineering/branching.md` for branch prefixes and examples.

## Documentation source of truth

When documentation disagrees:

1. current code/schema/tests/installers/production config;
2. current architecture/security docs;
3. accepted ADRs;
4. `PRODUCT.md` / `ROADMAP.md`;
5. historical requirements/release notes.

Do not rewrite historical release records to pretend they describe current `main`.
