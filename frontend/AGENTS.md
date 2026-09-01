# Frontend agent instructions

These rules apply to everything under `frontend/`.

## Mandatory visual source of truth

Before making ANY UI/layout/styling change, read [`DESIGN.md`](./DESIGN.md).

`DESIGN.md` is authoritative for:

- application-shell geometry;
- page/workspace spacing;
- surfaces, borders, radius, and elevation;
- navigation states;
- typography and color;
- operational tables;
- metrics/charts;
- loading-state scope;
- full-canvas tools such as ERD and Shell;
- responsive behavior and visual anti-patterns.

Do not invent route-local visual rules that contradict `DESIGN.md`. If a requested change intentionally changes the visual contract, update `DESIGN.md` first or in the same change.

In particular, do not reintroduce generic SaaS card stacks, first-level rounded boxes, strong border grids, large outer page padding, or full-page loading for local operations.

## Implementation discipline

- Keep implementation small and reuse existing Svelte/Tailwind primitives before adding abstractions or dependencies.
- Prefer fixing a shared primitive/token when an issue is systemic instead of patching many routes separately.
- Do not add a new UI framework or icon library. Svelte, Tailwind, and Lucide are sufficient for the current product scope.
- Do not create a second visual system for one feature.

## Application shell ownership

- Desktop primary navigation is the existing 56px rail.
- Hover/keyboard focus expands desktop navigation as an overlay; expansion must never resize or reflow the main workspace.
- Selecting a navigation item collapses the overlay.
- Do not add a persistent manual minimize/expand toggle or sidebar-width preference.
- Sidebar owns navigation only.
- Global topbar owns breadcrumbs/page context, notifications, appearance/theme control, account access, and suitable route-level primary actions.
- Mobile navigation must expose the same authorized destinations as desktop navigation.
- Root routes must not repeat the same title in topbar and body.
- Project observability belongs on Project Detail, not a standalone Metrics route.
- Do not keep duplicate route-level breadcrumbs hidden with CSS.

## Shared UI primitives

Before creating a new component, check whether these already solve the problem:

- `ActionButton`
- `ActionLink`
- `IconButton`
- `SectionPanel`
- `TableShell`
- `PageHeader`
- `EmptyState`
- `StatusBadge`
- `SegmentedChoice`
- `InfoDisclosure`
- shared field/surface utilities

Prefer a small extension to an existing primitive over a parallel component with slightly different styling.

## Controls

- Reuse `ActionButton` for text buttons, `ActionLink` for text navigation actions, and `IconButton` for icon-only utility/repeated actions.
- Visible workflow actions normally combine a Lucide icon with a readable label: New project, Refresh, Deploy, Start, Stop, Save, Retry, Import, Download, etc.
- Icon-only controls are reserved for compact utility/chrome actions such as notifications, appearance, overflow, clear input, reveal/hide, copy in a dense context, or row expand/collapse.
- Canonical action variants are `primary`, `secondary`, `ghost`, `danger`, and `ghostDanger`.
- Use `LoaderCircle` from Lucide for local loading indicators. Do not add custom border spinners.

## Icons

- `@lucide/svelte` is the single source for generic product/UI icons.
- Brand marks may use a centralized local component when Lucide intentionally does not provide the brand.
- Do not use emoji or Unicode glyphs as UI status/action icons.
- Icon semantics must match the actual operation.
- Decorative icons use `aria-hidden="true"`; icon-only controls require a meaningful accessible label.

## Loading behavior

Follow the scope contract in `DESIGN.md`.

- Global main-content loading is for initial authenticated route/resource loading only.
- Repository inspection/detection is local state and MUST NOT blank the whole page.
- Mutations such as deploy/start/stop/restart, DB writes, shell input, explicit refresh, and form submissions use local progress.
- Polling, SSE, and background telemetry never trigger the global blocking loader.
- Preserve the mounted route while global initial loading is visually active so lifecycle requests can execute.

## Data and telemetry integrity

- Chart color communicates the resource being measured, not generic brand emphasis.
- Never manufacture fake history from a current percentage.
- Network counters are cumulative unless an API explicitly returns a rate; derive rates from successive valid samples and elapsed time.
- Reset network baseline if counters decrease.
- Preserve last-known telemetry during successful background refreshes.
- Do not present policy quotas as if they are physical host capacity.
- Do not fabricate unread state, release availability, progress, or success.

## Status behavior

- Status colors represent real backend/runtime state only.
- Do not default every state to a tinted rounded badge.
- Do not repeat the same state redundantly when another nearby control already communicates it clearly.

## Create Project workflow

Preserve the source-first workflow:

- repository inspection;
- branch resolution;
- deployment detection;
- environment scan;
- staged presentation of real results.

Rules:

- Real repository/detection requests start immediately.
- Minimum visual duration may make a real operation readable but must never delay when work begins or fabricate backend work.
- Deployment Type remains in the normal flow, not hidden inside Advanced settings.
- Environment detection remains visible in the normal flow.
- Advanced settings contain only genuine overrides/diagnostics and require a clear disclosure trigger.
- Do not constrain the whole workflow with an arbitrary narrow route-level max width.

## Database / ERD

- Database Studio and schema design follow the full-workspace contract in `DESIGN.md`.
- ERD canvas owns pan/zoom interaction.
- Preserve pointer-anchored zoom behavior for mouse/trackpad gestures.
- Do not re-wrap the design canvas in generic cards or introduce page scroll around a full-canvas schema workspace.

## Host Shell

- Host Shell is an operational workspace, not a dashboard card.
- Preserve owner-only authorization and existing backend safety/session boundaries.
- Terminal output/input loading stays local; shell commands must not trigger the global page loader.
- Do not add decorative terminal wrappers that reduce usable workspace height.

## Accessibility

- Preserve keyboard access and visible focus states.
- Hover-only behavior requires an equivalent focus/keyboard path where applicable.
- Do not rely on color alone for critical state.
- Avoid noisy live announcements during polling/background refresh.

## Frontend validation

For frontend-only changes, the required merge gate is:

1. frontend unit tests;
2. Svelte/TypeScript checks;
3. production frontend build.

Fix failures before merging. Backend/infrastructure gates are required only when the change actually touches those contracts.
