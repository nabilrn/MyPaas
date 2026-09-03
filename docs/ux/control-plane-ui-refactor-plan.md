# Control-plane UI reliability and refinement plan

**Status:** Proposed implementation plan  
**Baseline:** `origin/main` at `5a671c0`  
**Prepared:** 2026-09-02

## Objective

Make the MyPaaS dashboard easier to trust during deployment, failure, recovery, and host administration without replacing its existing compact visual system or expanding the product beyond its single-host boundary.

The refactor must prioritize:

1. one understandable operational state per project;
2. current state and recovery actions before optional integrations;
3. explicit impact for privileged or destructive actions;
4. honest loading, stale, partial, and insufficient-data states;
5. equivalent legibility and interaction hierarchy in light and dark themes;
6. responsive operational tables that preserve identity, state, and action.

## Inputs and current-code corrections

This plan combines a visual review of the authenticated dashboard in light and dark themes with a code review of the current frontend. The screenshots remain useful evidence, but current code is authoritative.

Several screenshot findings have already been improved on current `main` and must not be reimplemented:

- desktop navigation already expands as a non-reflowing hover/focus overlay in `Navbar.svelte`;
- Create Project already uses source-aware environment copy and a staged single-page analysis flow;
- host telemetry already derives CPU and network rates from successive samples instead of manufacturing history;
- Migration already represents preparing, failed, ready, and expired states;
- MCP token regeneration already requires an inline confirmation;
- Projects already has a deliberate compact layout below the desktop table breakpoint.

The remaining work below is grounded in gaps still present in current code.

## Confirmed problems

### 1. Project state is derived independently in several surfaces

Current state interpretation is split between:

- `frontend/src/routes/projects/[id]/+page.svelte` (`applicationStateFor`);
- `frontend/src/routes/projects/+page.svelte` (`getDerivedStatus` and primary-action selection);
- `frontend/src/lib/components/ProjectStatus.svelte`;
- `frontend/src/lib/components/StatusBadge.svelte`;
- `frontend/src/lib/utils/deploymentHistory.ts`.

The project overview's fallback for a pending project with a deployment record is always `Application is waiting`, even when the latest attempt failed. Static releases also inherit the backend `running` vocabulary even though they have no persistent runtime and display `n/a` uptime.

**User risk:** an operator can see `Pending`, `Failed`, an endpoint, and a Deploy action without a single statement explaining whether traffic is being served or what to do next.

### 2. Optional edge analytics outranks core operations

`ProjectObservability.svelte` is rendered before the latest deployment and project essentials. When Cloudflare is not configured, it renders the complete installation-level credential form through `CloudflareSetup.svelte` on every project overview.

**User risk:** deployment failure and runtime condition are pushed below an optional integration. A global owner credential workflow is also presented as if it were project-local configuration.

### 3. Charts render before history is meaningful

`ProjectObservability.svelte` sends the current history to `MultiServiceMetricChart` as soon as one metric item exists. Host charts have an initial `Collecting…` value, but project charts do not distinguish one current sample from a usable series.

**User risk:** a large empty plot can look like zero activity or broken telemetry instead of insufficient history.

### 4. High-trust actions use route-local confirmation patterns

MCP regeneration, webhook regeneration, Compose resource reset, rollback, database write mode, project lifecycle actions, shell sessions, backup, and migration each implement impact and confirmation differently. There is no shared confirmation primitive or impact vocabulary.

**User risk:** similarly styled buttons can have very different scope, reversibility, and side effects.

### 5. Table responsiveness is inconsistent

Projects has a compact fallback, while Containers, Users, Audit, and some database/environment tables primarily depend on fixed `min-width` tables and horizontal scrolling.

**User risk:** the rightmost state or action can leave the viewport, especially on laptop and mobile widths.

### 6. Theme tokens do not fully communicate control state

The base palette is centralized in `frontend/src/app.css`, but many routes still use route-local gray/neutral utilities. In dark mode, secondary text, row dividers, disabled controls, and editable input boundaries can converge visually. Read-only, disabled, editable, warning, and destructive states are not consistently distinguishable without reading nearby copy.

**User risk:** operators can miss unavailable controls, stale metadata, or the difference between a protected value and an editable secret.

## Target experience

### Project operational model

Introduce one frontend-owned operational view model derived from existing API data. Do not change backend lifecycle semantics merely to rename UI labels.

The view model should expose:

```ts
type ServingState = 'live' | 'offline' | 'degraded' | 'unknown';
type ReleaseState = 'not_deployed' | 'queued' | 'deploying' | 'succeeded' | 'failed';
type DesiredState = 'running' | 'stopped';

interface ProjectOperationalState {
  serving: ServingState;
  release: ReleaseState;
  desired: DesiredState;
  headline: string;
  detail: string;
  primaryAction: 'deploy' | 'retry' | 'start' | 'view_logs' | 'view_deployment';
  attention: 'none' | 'info' | 'warning' | 'danger';
}
```

Names may change during implementation, but one pure derivation function must own the matrix. Required cases:

| Project/runtime evidence | Latest deployment | UI result | Primary next action |
| --- | --- | --- | --- |
| no active deployment | none | Not deployed | Deploy |
| pending, no serving release | failed | Deployment failed; offline | View failure logs / Retry |
| active release still serving | newer attempt failed | Live; latest deploy failed | Review failed attempt |
| building | active pipeline | Deploying | View deployment |
| crashed | any | Crashed; offline | View logs |
| intentionally stopped | any | Stopped | Start |
| static active release | succeeded/running backend state | Live | Redeploy |
| runtime evidence unavailable | any | Status unknown | Retry status / View diagnostics |

The backend may continue returning `running` for an active static release. The presentation layer should use `Live`, omit runtime uptime, and show publish metadata instead.

### Project overview order

The default project overview should render in this order:

1. project identity, serving state, endpoint, and Deploy/recovery action;
2. attention banner only when action is required;
3. latest deployment/release summary;
4. runtime summary for container-backed projects or publish summary for static projects;
5. logs/recent failure entry point when relevant;
6. endpoint, environment, database, and deployment setup shortcuts;
7. edge analytics data when configured;
8. compact integration setup link when not configured.

Move installation-level Cloudflare credential entry to an owner-only administration/settings surface. Project pages should never contain the raw global credential form. A non-owner should see either available analytics or a neutral unavailable state, not a control that the backend will reject.

### Honest telemetry states

Use the current value immediately, but require enough samples before drawing a trend:

- zero samples: `Waiting for telemetry`;
- one sample: show current CPU/memory values and `Collecting history`;
- two or more samples: render a line, with an explicit time window/sample count where useful;
- reconnecting: preserve the last known value and label it stale/reconnecting;
- disconnected with no data: show recovery guidance;
- static project: do not render runtime telemetry.

Do not add fake points, repeat the first point, or delay the real subscription to make an animation look complete.

### High-trust action contract

Add a shared confirmation/impact pattern built from existing `ActionButton`, overlay, and focus utilities. It must support:

- object identity and scope;
- concise consequence;
- reversible/temporary/permanent label;
- confirm and cancel actions;
- independent pending state;
- focus placement, focus trap, Escape handling, and focus restoration;
- optional typed confirmation only for irreversible high-impact actions;
- success/error feedback without removing useful stale data.

Apply it according to risk, not mechanically to every mutation:

| Action | Required treatment |
| --- | --- |
| Deploy/start/restart | local pending state; no extra confirmation by default |
| Stop project | confirm when traffic is currently served; identify project |
| Enable DB write | identify database, explain 15-minute scope, show expiry after success |
| Delete DB row | identify table and primary key; permanent confirmation |
| Regenerate MCP/webhook token | state that old credential is immediately invalidated |
| Reset Compose resources | preserve existing strong confirmation and migrate to shared pattern |
| Rollback | identify source and target release and expected traffic change |
| End shell session | local pending state; no modal unless unsent/running work would be lost |
| Prepare migration | summarize pause/exclusion impact before starting |

### Operational table contract

Extend `TableShell` with reusable layout behavior rather than adding route-local card systems:

- desktop table with deliberate column geometry;
- sticky identity and/or action columns only where they materially improve use;
- compact row/list representation below the route's supported breakpoint;
- copy/title affordance for truncated hashes, image references, URLs, and UUIDs;
- consistent loading, empty, partial-error, pagination, and result-count placement;
- filters summarized as controls/chips without explanatory prose;
- row actions in one predictable right-aligned location.

Priority order on constrained widths is always: identity, critical state, primary action, then secondary metadata.

### Theme and accessibility contract

Keep the monochrome design language and semantic resource colors. Refine shared tokens before patching routes.

Required checks:

- body, supporting, placeholder, disabled, and technical text meet intended contrast in both themes;
- input boundaries remain visible without making the whole interface a border grid;
- editable, read-only, and disabled states are distinguishable by more than opacity;
- focus rings are visible on base, inset, warning, and overlay surfaces;
- success/warning/danger/info states include text or icon semantics, not color alone;
- reduced motion is respected for spinners, transitions, and pulsing status;
- controls preserve readable labels and coarse-pointer targets on mobile.

## Implementation work packages

Each package should be a separate narrow branch and PR. Do not combine all work into one implementation branch.

### PR 1 — `ux/project-operational-state`

**Outcome:** one tested source of truth for serving, release, desired, and recovery presentation.

Primary files:

- add `frontend/src/lib/utils/project-operational-state.ts`;
- add `frontend/src/lib/utils/project-operational-state.test.ts`;
- update `frontend/src/routes/projects/[id]/+page.svelte`;
- update `frontend/src/routes/projects/+page.svelte`;
- update `frontend/src/lib/components/ProjectStatus.svelte` and/or `StatusBadge.svelte` only if their public contract must change;
- reuse `frontend/src/lib/utils/deploymentHistory.ts` rather than duplicating release logic.

Acceptance criteria:

- the pending + failed case cannot render `Application is waiting` without mentioning failure;
- a failed newer deployment does not imply the active previous release is offline;
- static projects display release/serving language and no fake runtime uptime;
- project list, project header, attention banner, and deployment history agree;
- every state-matrix row has a unit test.

### PR 2 — `ux/project-overview-priority`

**Outcome:** core operations appear before optional analytics and duplicated project information is reduced.

Primary files:

- update `frontend/src/routes/projects/[id]/+page.svelte`;
- split or refine `frontend/src/lib/components/ProjectObservability.svelte`;
- move `CloudflareSetup.svelte` into the existing owner settings flow or an owner-only integration section;
- update `frontend/DESIGN.md` only if the resulting shared ordering changes its current contract.

Acceptance criteria:

- latest deployment/recovery is visible before optional Cloudflare setup;
- no raw installation-level credential form appears on a project overview;
- runtime and edge analytics have independent loading/error states;
- project header and essentials do not repeat the same facts without a distinct action;
- owner and non-owner variants are covered.

### PR 3 — `ux/telemetry-readiness`

**Outcome:** charts distinguish current values, collecting history, live history, stale data, and unavailable data.

Primary files:

- update `frontend/src/lib/components/ProjectObservability.svelte`;
- update `frontend/src/lib/components/MultiServiceMetricChart.svelte`;
- update `frontend/src/lib/components/CapacityMetricChart.svelte` if shared readiness presentation is useful;
- extend `frontend/src/lib/utils/project-metric-history.ts` and tests;
- preserve `frontend/src/lib/utils/host-telemetry.ts` rate derivation.

Acceptance criteria:

- one sample never appears as a meaningful trend line;
- last-known values remain visible during reconnect/background refresh;
- sample history resets when project/service identity changes;
- an unavailable stream has a retry or diagnostics path where one exists;
- no polling or SSE update triggers the global loader.

### PR 4 — `ux/high-trust-actions`

**Outcome:** privileged and destructive actions communicate scope and consequence consistently.

Primary files:

- add a shared confirmation/impact component under `frontend/src/lib/components/`;
- apply it first to database write/delete, MCP token regeneration, webhook regeneration, Compose reset, rollback, project stop, and migration preparation;
- keep all API calls in `frontend/src/lib/api/`;
- add focused component/utility tests and route-level tests for independent pending flags.

Acceptance criteria:

- double submit is impossible;
- cancel restores focus to the trigger;
- failure preserves the dialog/context and shows a recovery action;
- token invalidation, database write duration, and migration pause are explicit;
- benign actions do not acquire unnecessary confirmation friction.

### PR 5 — `ux/operational-table-responsiveness`

**Outcome:** identity, state, and action remain usable without relying on off-screen horizontal columns.

Primary files:

- extend `frontend/src/lib/components/TableShell.svelte` conservatively;
- update Containers, Ports, Users, Audit, Database Studio, and Environment tables;
- preserve the Projects compact representation as the reference pattern without copying route markup blindly.

Acceptance criteria:

- 390px, 768px, 1024px, and 1440px layouts have no clipped primary action;
- horizontal scrolling, where retained for technical datasets, is visibly intentional;
- long identifiers do not change column geometry;
- keyboard users can reach row actions and overflow controls;
- empty, loading, error, and partial-error states use stable dimensions.

### PR 6 — `ux/theme-state-parity`

**Outcome:** light and dark themes communicate the same hierarchy and control states.

Primary files:

- refine tokens/utilities in `frontend/src/app.css`;
- update shared field, status, table, action, and overlay primitives;
- remove route-local color exceptions only when the shared token fully owns the state;
- update `frontend/DESIGN.md` with any deliberate token/contrast contract changes.

Acceptance criteria:

- editable/read-only/disabled fields are visually distinct in both themes;
- secret inputs use neutral control surfaces unless a real warning exists;
- row separators and supporting text remain legible without creating a heavy grid;
- automated accessibility checks pass and a keyboard/focus visual pass is recorded;
- screenshots at the standard viewport set show equivalent information hierarchy.

### PR 7 — `ux/admin-operations-density`

**Outcome:** low-frequency administration pages become compact operational workflows rather than large instruction surfaces.

Scope only after the shared state/action/table primitives above are merged:

- Audit: human-readable primary event labels, filters/search where supported by the API, raw event/HTTP data in details;
- Backup: configuration state, destination, last result, next run, and the project-volume exclusion near the action;
- Migration: preflight/exclusion summary and package lifecycle using existing backend data only;
- MCP: credential metadata and client setup hierarchy without decorative dominance;
- Settings: dirty state, save/apply semantics, units, floors, and whether changes affect new or existing projects;
- Users: replace ambiguous `Protected` with an explicit primary-owner constraint and remove unnecessary pagination chrome for one page.

Do not invent history, scheduling, progress, actor identity, token usage, preflight detail, or configuration mutability that the backend does not provide. If a required value is missing, split the API contract into a separate `core/` branch rather than hiding backend work in the UX PR.

## Validation strategy

Every implementation PR must run the proportional frontend gate:

```text
cd frontend
pnpm test
pnpm check
pnpm build
```

Also add targeted browser coverage when the PR changes interaction or responsive behavior:

- desktop: 1440x900;
- laptop: 1024x768;
- mobile: 390x844;
- light and dark themes;
- keyboard-only navigation and visible focus;
- success, pending, disabled, failure, stale/partial, and empty states relevant to the PR.

Production evidence should remain non-destructive. Use deterministic mocked states for failed deploys, stale telemetry, unavailable integrations, confirmation failures, and dangerous actions.

## Branch and change discipline

- Branch from a fetched, up-to-date `origin/main`.
- Use `<domain>/<short-outcome>` names from `docs/engineering/branching.md`.
- Keep one branch to one domain and one outcome.
- Do not revive or stack work on a branch whose upstream was deleted.
- Rebase or fast-forward before final verification; do not merge unrelated local work.
- Update `frontend/DESIGN.md` only when changing the shared visual contract.
- Update product/runtime documentation only when actual behavior changes.
- Do not add a UI framework, icon library, websocket path, or parallel design system.

## Non-goals

- no Kubernetes, multi-node, HA, autoscaling, or hostile multi-tenant concepts;
- no broad backend redesign as part of visual cleanup;
- no Create Project wizard without new evidence overturning the current single-page contract;
- no decorative dashboard redesign, card stack, gradients, oversized headings, or marketing copy;
- no fake telemetry, progress, unread notification state, or release availability;
- no removal of security, validation, failure, stale-state, or destructive-action information to gain visual simplicity.

## Completion definition

The program is complete when:

- project state and next action are consistent across overview, inventory, and deployment history;
- optional integrations cannot displace deployment/runtime failure above the fold;
- high-trust actions share tested impact and pending behavior;
- charts never imply history before enough samples exist;
- primary actions remain available at supported viewport widths;
- light and dark themes communicate equivalent state hierarchy;
- all affected frontend gates and targeted interaction checks pass;
- no work package exceeds its declared branch domain or introduces undocumented backend behavior.

## Short implementation instruction

Start with `ux/project-operational-state`. Read `AGENTS.md`, `frontend/AGENTS.md`, `frontend/DESIGN.md`, and this plan; fetch `origin`, branch from current `origin/main`, implement only PR 1, run the frontend gate, and open a separate PR before starting the next package.
