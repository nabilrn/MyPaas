# ADR-016: Flexible Compose Configuration

## Status

Accepted

## Context

MyPaas supported Docker Compose deploys from day one, but with a hard-coded
assumption: the compose file lives at the repository root and matches a fixed
list of candidate filenames (`docker-compose.yml`, `compose.yml`,
`docker-compose.prod.yml`, etc.). Detection (`project.detectComposeFile`) and
deploy (`deployment.findComposeFile`) duplicated this candidate list and both
scanned only the workspace root.

This blocked several real-world shapes:

- Compose file in a subdirectory (`infra/docker-compose.yml`, `docker/prod.yml`,
  monorepo package `apps/api/compose.yaml`).
- Custom compose filenames that don't match the hardcoded list
  (`app-stack.yaml`).
- Multiple `-f` files chained by the user (base + prod override + cache layer).
- Compose `profiles` to bring up only a subset of services per deploy.
- One repository hosting several MyPaas projects, each pointing at a different
  compose file.

There was also no persisted column for the compose path. Even if detection
could find a subdirectory file, webhook redeploys would re-scan the root and
fail with `ErrComposeFileNotFound`.

## Decision

Make compose deployment path-aware and profile-aware while preserving the
existing root-only behaviour as the default fallback.

### Data model

Add four nullable columns to `projects` (migration `000009`):

- `compose_file_path VARCHAR(255)` — repo-relative primary compose file.
- `compose_override_paths VARCHAR[]` — additional `-f` files (repo-relative).
- `compose_profiles VARCHAR[]` — `COMPOSE_PROFILES` values.
- `compose_workdir VARCHAR(255)` — repo-relative cwd override; defaults to the
  compose file's directory.

A `CHECK` constraint ensures compose_* fields are only populated on
`deploy_mode = 'compose'` rows. A `CHECK` constraint rejects absolute paths,
backslashes, and `..` segments in `compose_file_path` / `compose_workdir`.

All columns nullable means existing Dockerfile/static projects and existing
compose projects (where the file lives at the root) keep working unchanged —
NULL triggers the recursive discovery fallback.

### Shared discovery package

Introduce `internal/compose/` as the single source of truth for:

- `Discover(workspace)`: recursive `filepath.WalkDir` capped at depth 4,
  skipping `node_modules`, `vendor`, `.git`, `.next`, `dist`, `build`, etc.
  Returns ranked `Candidate` structs with `Path`, `Score`, `Depth`.
- `ValidateUserPath(path)`: rejects absolute paths, backslashes, and `..`
  segments. Used by the API layer and the resolver.
- `ResolveLayout(workspace, primary, overrides, workdir, ...)`: turns the
  cloned workspace + persisted project fields into an absolute `Layout` for
  `docker compose` — `WorkDir`, ordered `UserFiles`, MyPaas-generated
  `OverrideFile` / `SanitizedFile` placed inside `WorkDir`, and an absolute
  `EnvFile` path.

This removes the duplicated `composeFileCandidates` / `ignoredComposeCandidate`
helpers that previously lived in both `project/service.go` and
`deployment/service.go`.

### Container client

`container.ComposeUpOptions` gains `ComposeFiles []string` and
`Profiles []string`. The legacy `ComposeFile` / `OverrideFile` fields stay so
existing callers compile; `composeUpFiles` orders them primary → user files →
MyPaas override so MyPaas's port binding always wins.

`ComposeServices`, `ComposeBuildServices`, and the new
`WriteSanitizedComposeConfigMulti` accept variadic `-f` files via
`composeConfigArgsMulti`. The sanitized JSON is rendered once from the merged
user files (via `docker compose config --format json`), so user overrides are
already baked in — ComposeUp only needs `[sanitized, mypaas-override]`.

`COMPOSE_PROFILES` is exported via the subprocess env so docker compose
applies it consistently across `config` and `up`.

### API surface

- `POST /projects/detect-compose` returns the ranked candidate list for a
  repo+branch, without running full deploy-mode detection. Used to populate
  the create-form picker.
- `POST /projects` (create) and `PATCH /projects/{id}` (update) accept
  `composeFilePath`, `composeOverridePaths`, `composeProfiles`, and
  `composeWorkdir`. `Update` also makes `mainService` mutable for compose
  projects (previously create-only).
- The `Project` response includes the four new fields so the settings page can
  render them.

### Frontend

- The create form shows a "Compose configuration" panel when `deployMode` is
  `compose`: a path input, a working-directory override input, comma-separated
  override and profile inputs, and a "Scan for compose files" button that
  populates a clickable candidate list.
- The settings page adds a "Compose configuration" panel for compose projects
  with the same four fields, editable post-create.
- The `Project` and `DeployModeDetection` types include the new fields;
  `api.projects.detectCompose` is exposed.

### Build context resolution

The Compose Doctor previously resolved `build.context` against the workspace
root. Docker Compose actually resolves it against the compose file's
directory. `inspectComposePlan` now computes `composeDir` and passes it to
`composeServicePlanFromConfig` and `addComposeServiceIssues`, so subdir
compose files get accurate build-context existence checks.

## Consequences

- Repositories with compose files anywhere in the tree (subdirectory,
  monorepo package, `infra/`, `docker/`, `deploy/`) now deploy without
  manual restructuring.
- Users can chain multiple `-f` files and select `COMPOSE_PROFILES` from the
  dashboard, covering advanced compose setups (prod/cache overrides,
  worker-only deploys, etc.).
- One repository can host multiple MyPaas projects, each pointing at a
  different compose file via `compose_file_path`.
- Existing projects (root compose, Dockerfile, static) are unaffected — NULL
  compose fields trigger the same root-scan fallback as before, now via
  `compose.Discover` which ranks root files highest.
- Path safety is enforced at three layers: DB `CHECK` constraints,
  `compose.ValidateUserPath` in the service layer, and `compose.ResolveLayout`
  which stats every file inside the workspace before use.
- The duplicated candidate lists in `project/` and `deployment/` are gone;
  any future change to discovery rules lives in `internal/compose/` only.
