# New Project UX Refactor Design

## Goal

Make project creation source-first and detection-first so normal users provide a deploy source, review MyPaas' inferred plan, fix only missing requirements, and create the project without understanding platform internals.

## Scope

Frontend-only refactor of `frontend/src/routes/projects/new/+page.svelte` plus small testable UX helpers. Backend request/response contracts remain unchanged.

## Happy path

1. User chooses Git Repository or Container Registry.
2. For Git, entering a repository automatically inspects the repo, selects the default branch, and runs runtime detection when the current repo/branch/base-directory inspection is valid.
3. Project name is suggested from the repository/image slug when the user has not intentionally edited it.
4. The page shows detected runtime as a result, not as a question. Manual deploy-mode, port, base-directory, Compose overrides, and resource limits live under progressive-disclosure Advanced sections.
5. Required environment variables are visually prioritized. Optional/discovered values stay editable but do not dominate the happy path.
6. `Ready to create` is shown only when runtime resolution and all blocking requirements are complete.

## Key UX decisions

- Keep the current single-page layout; do not reintroduce a multi-step wizard.
- Replace the primary `Detect runtime` action with automatic analysis; retain a `Re-analyze` action for explicit retry.
- Do not expose `Base directory` by default. Blank means repository root. Never use `/` as the placeholder because absolute paths are rejected by validation.
- Treat deployment mode, container port, Compose file/workdir/overrides/profiles, and resource overrides as advanced configuration.
- Keep detected container port read-only in the normal flow.
- Keep Compose Doctor as the compatibility authority; advanced Compose file selection is collapsed while blocking Doctor findings remain visible and actionable.
- Show coding-agent handoff only when deployment analysis fails and the generated prompt is useful.
- Keep shared PostgreSQL available, but present it as an optional database capability instead of a small header checkbox.

## State model

The create CTA follows these states:

- `Waiting for source`
- `Analyzing deployment`
- `Needs configuration`
- `Ready to create`

For Git sources, `Ready to create` requires a current repository inspection and a resolved non-`auto` runtime. For non-static runtimes it also requires a resolved container port. Compose additionally requires no blocking Compose Doctor issue and no missing required env value.

## Testing

Focused Vitest coverage validates:

- deriving a safe project-name suggestion from repository/image input;
- preventing Git `auto` mode from reporting ready before runtime analysis finishes;
- requiring a resolved container port for detected non-static runtimes;
- existing payload/path validation, including repository-relative paths and runtime port bounds.

Existing project validation tests remain the source of truth for payload validation.

## Implementation boundary

This refactor intentionally does not change project creation APIs, backend detection, deployment orchestration, or persisted project fields. It changes how existing capabilities are presented and when the create action is considered ready.