# Create Project Production UX Audit - 2026-08-13

Production audit target: `https://nabilrn.space/projects/new`

Primary browser: Firefox via Playwright `1.62.1`

Artifact root: `frontend/artifacts/create-project-audit/`

## Run Summary

- Mode: production
- Runs: 4
- Checkpoints: 36
- Console errors: 0
- Console warnings: 0
- Failed requests: 2
- Geometry findings: 0

Viewports:

- Desktop: 1366x768
- Large desktop: 1440x900
- Mobile: 390x844

Scenarios:

- `non-destructive-main`
- `invalid-repository-error`

Production audit remained non-destructive. It did not submit Create Project, create a real project, deploy an application, delete resources, or mutate production resources.

## Findings

### Evidence Gap - Re-analysis Checkpoint Timing

- Category: STATE
- Scenario: `non-destructive-main`
- Checkpoints: `07-reanalyze-triggered`, `08-readiness`
- Evidence:
  - Earlier local artifacts sampled `08-readiness` while the UI still showed `Analyzing deployment`.
  - That sample did not prove a product bug because the previous `Needs configuration` text could satisfy the harness wait before the new Re-analyze request completed.
- Expected audit behavior:
  - Re-analysis evidence should wait for the new deployment-detection request and final reveal state before classifying the product state.
- Corrective action:
  - The Playwright harness now waits for a non-`inspectOnly` `/api/projects/detect-mode` response after clicking Re-analyze and then waits for busy analysis copy to clear before taking the `08-readiness` checkpoint.
- Product finding status:
  - The corrected rerun settled `08-readiness` back to `Needs configuration`; do not treat the earlier `Analyzing deployment` sample as a confirmed UI bug.
- Confidence: high

### P2 - MyPaas Repository Correctly Blocks Create, But The Normal Flow Is Very Long And Dense

- Category: FLOW
- Scenario: `non-destructive-main`
- Checkpoints: `06-advanced-closed`, `08-readiness`
- Evidence:
  - `production-firefox-desktop-non-destructive-main/audit.json`
  - Create remains disabled.
  - Visible blockers include `Mounting /var/run/docker.sock into app containers is not allowed.`
  - Required env values missing: `CLOUDFLARE_TUNNEL_TOKEN`, `ENCRYPTION_KEY`, `GITHUB_CLIENT_SECRET`, `JWT_SECRET`, `POSTGRES_PASSWORD`.
  - The same checkpoint contains many Compose Doctor warning/info lines.
- Expected behavior:
  - Required blockers should be impossible to miss and should outrank non-blocking diagnostics.
- Observed behavior:
  - The required blockers are present, but compete with a large amount of warning/info diagnostic content.
- UX impact:
  - The user may understand that Create is blocked, but still need to scan too much text to identify the exact next action.
- Likely source area:
  - Create Project normal-flow rendering and Compose Doctor diagnostic presentation in `frontend/src/routes/projects/new/+page.svelte`.
- Recommended correction:
  - Keep the blocking Compose issue and missing required env summary prominent. Move non-blocking Compose warnings/info farther behind diagnostics or collapse them more aggressively.
- Confidence: medium

### P4 - Invalid Repository Error Is Safe But Technically Worded

- Category: VALIDATION
- Scenario: `invalid-repository-error`
- Checkpoints: `08-readiness`
- Evidence:
  - `production-firefox-desktop-invalid-repository-error/audit.json`
  - UI shows `validation failed: failed to inspect remote branches: fatal: could not read Username for 'https://github.com': No such device or address`.
  - Create remains disabled.
  - Network contains expected `400` responses from `/api/projects/detect-mode`.
- Expected behavior:
  - Invalid or inaccessible repository errors should explain the user action needed.
- Observed behavior:
  - The low-level Git error is surfaced directly.
- UX impact:
  - The user can infer repository access failed, but the message is more implementation-oriented than task-oriented.
- Likely source area:
  - Backend repository inspection error mapping and frontend error presentation.
- Recommended correction:
  - Translate this into user-facing copy such as repository not found, private repository inaccessible, or GitHub credentials required, while preserving technical detail in diagnostics.
- Confidence: high

## Non-Issues Observed

- No console errors or warnings were captured.
- No simple DOM geometry issues were captured after the corrected rerun. Earlier mobile overlap artifacts were superseded by the re-analysis wait fix and are not treated as a product finding.
- Invalid repository handling is non-destructive and keeps Create disabled.
- Production audit did not perform project creation.
- Auth bootstrap storage state is ignored by Git and must not be committed.
