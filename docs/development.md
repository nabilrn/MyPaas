# Development

This document describes the repository development workflow. It is separate from the production installation path in [Installation](installation.md).

Public MyPaaS support uses an Issues-only contribution flow, but the development workflow remains documented for maintainers, forks, and local investigation.

## Toolchain

The current repository CI uses:

- Go `1.25.5`;
- Node.js `22`;
- pnpm `10.22.0`.

The frontend also declares `pnpm@10.22.0` in `frontend/package.json`.

Depending on the target you run locally, you may also need:

- Docker-compatible `docker` + `docker compose` command surface;
- `migrate` for database migrations;
- `air` for Go live reload;
- `golangci-lint` for backend linting.

Use the repository CI/workflow files as the authority for exact CI setup.

## Discover repository targets

```bash
make help
```

## Start local development dependencies

```bash
make dev
```

`make dev` does **not** leave both application servers running in one process. It:

1. starts `docker-compose.dev.yml` dependencies;
2. runs database migrations;
3. tells you to start the API and dashboard in separate terminals.

Terminal 1:

```bash
make backend-dev
```

Terminal 2:

```bash
make frontend-dev
```

`backend-dev` runs the Go API with `air`. `frontend-dev` installs frontend packages with pnpm and starts the SvelteKit/Vite dev server.

## Tests

Run the repository test targets:

```bash
make test
```

This currently runs backend tests with the Go race detector and frontend tests.

Run individual sides when useful:

```bash
make test-backend
make test-frontend
```

Coverage helper:

```bash
make test-coverage
```

CI contains additional gates beyond `make test`; a local `make test` pass must not be described as equivalent to the full GitHub Actions qualification matrix.

## Checks and linting

```bash
make lint
```

This runs the configured Go lint path and Svelte frontend checks.

Frontend checks can be run directly with:

```bash
cd frontend
pnpm check
```

## Build

```bash
make build
```

Backend-only:

```bash
make build-backend
```

Current backend binaries are written to:

```text
backend/bin/mypaas-api
backend/bin/mypaas
backend/bin/mypaas-sqlite-helper
```

Frontend-only:

```bash
make build-frontend
```

The frontend build output is written under `frontend/build`.

## Database migrations and sqlc

Apply migrations:

```bash
make migrate-up
```

Roll migrations back in a development environment:

```bash
make migrate-down
```

Create a new migration:

```bash
make migrate-new
```

Regenerate sqlc code through the pinned repository helper:

```bash
make sqlc
```

Do not hand-edit generated sqlc output as a substitute for changing its schema/query source.

## Development containers

```bash
make docker-up
make docker-down
```

Destructive local reset, including development volumes:

```bash
make docker-reset
```

Use the reset target only when deleting local development state is intended.

## Production-related commands

Production install and verification targets exist in the Makefile:

```bash
make install-vm
make verify-prod
```

They are not substitutes for reading the production runbooks. `make install-vm` invokes the VM installer from the current checkout; stable public installation should use the pinned bootstrap path documented in [Installation](installation.md).

## Generated artifacts and qualification

Generated benchmark/test output should not be committed unless a specific retained evidence contract requires it. Product capacity claims must not be derived from one local VM, fixture count, RPS run, or concurrent-user run.

See [Runtime verification](engineering/runtime-verification.md) for retained qualification boundaries.

## Engineering conventions

Read [`../AGENTS.md`](../AGENTS.md) before making repository changes. It records current product boundaries and implementation constraints that should not be silently widened.

Public bug reports and feature proposals should use GitHub Issues as described in [`../CONTRIBUTING.md`](../CONTRIBUTING.md).
