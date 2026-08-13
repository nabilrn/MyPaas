# Frontend UI consistency rules

These rules apply to everything under `frontend/`. Keep the implementation small and reuse the existing Svelte/Tailwind primitives before adding abstractions or dependencies.

## Product visual direction

MyPaaS is a flat monochrome operational UI. Prefer consistency, information hierarchy, real data, and useful density over decorative variety.

- Normal product chrome is white, black, and neutral gray in light/dark mode.
- Do not use green/emerald as generic brand decoration, generic selection color, or generic primary-action color.
- Primary workflow actions use monochrome inversion: dark control on light surfaces, light control on dark surfaces.
- Color is reserved for actual state and data semantics. Current resource mapping is RAM=green, CPU=blue, storage=amber, network=violet.
- Warning, danger, info, and success colors are allowed only when they communicate real state.
- Keep normal cards flat and neutral. Do not imitate another PaaS mechanically; preserve MyPaaS's own compact operational language.
- Do not make the interface feel premium by adding whitespace. MyPaaS is an operational control plane and should expose useful information per viewport.

## Shell ownership

- Desktop sidebar owns primary navigation, user identity/account access, and appearance/theme control.
- Global topbar owns only page context/breadcrumbs and the notification bell.
- Mobile navigation must still expose account and appearance controls because the desktop sidebar is hidden.
- The notification center is generic infrastructure for real platform/release/deployment/resource events. Never fabricate unread state or release availability.
- Root routes must not repeat the same title in both the topbar and page body. Deep routes may use the topbar for breadcrumb context such as `Projects / project / Deployments`. Project observability belongs on Project Detail, not a standalone Metrics route.
- Do not keep a second route-level breadcrumb implementation hidden with CSS. Navigation context must have one owner.

## Layout and density

- Operational routes use the shared `page-shell`. Do not introduce route-specific centered `max-w-*` wrappers for the whole page unless the content is genuinely document/dialog-like.
- Field-level `max-w-*` is allowed when it improves readability. Constrain the field, not the entire operational workflow.
- Parent layouts own external spacing between sections. Components own only their internal padding/gaps.
- Avoid nested cards used only to create more whitespace. Use dividers and layout grids when the content belongs to one surface.
- Prefer a restrained spacing rhythm around 8px inline, 12px small detail, 16px normal content, 20px sections, and 24px page spacing instead of arbitrary route-local gap/mb values.
- Operational table rows should normally land around 52–60px effective height. Do not use `px-5 py-4` card-like rows by default.
- Prefer the shared `.data-table` grammar for ordinary operational/admin tables.

## Surfaces and elevation

Use only the established surface hierarchy:

- `.surface`: normal card/content container. Flat, bordered, no shadow.
- `.surface-muted`: subtle inset/secondary grouping. No shadow.
- `.overlay`: floating UI such as menus, popovers, notification panels, and modal surfaces. Shadow is allowed here.

Do not add arbitrary `shadow-*` classes to normal cards, tables, stat tiles, headers, or toolbars. Semantic alerts may use tinted backgrounds, but an alert tone is not a new card/elevation type.
Use `.console-surface` and `.code-surface` for terminal/code content instead of recreating dark boxes per route.

## Controls

- Reuse `ActionButton` for text buttons, `ActionLink` for text navigation actions, and `IconButton` for icon-only utility/repeated actions before writing a generic raw control.
- Visible workflow actions should normally combine a Lucide icon with a readable text label: New project, Refresh, Deploy, Start, Stop, Save, Retry, Import, Download, and similar actions.
- Icon-only controls are reserved for compact chrome/utility actions such as notifications, overflow menus, clear input, reveal/hide, copy in a dense context, row expand/collapse, or sidebar collapse.
- Canonical action variants are `primary`, `secondary`, `ghost`, `danger`, and `ghostDanger`. Legacy IconButton aliases may exist for compatibility but new code should not use them.
- Standard controls should align to the 36px visual height. Keep the existing 44px coarse-pointer/touch target behavior.
- Use `LoaderCircle` from Lucide for loading indicators. Do not add custom border spinners.

## Icons

- `@lucide/svelte` is the single source for generic product/UI icons.
- Brand marks may use a local centralized SVG/component when Lucide intentionally does not provide the brand.
- Do not use emoji or Unicode glyphs as UI status/action icons.
- Icon semantics must match the operation: Stop is not Pause; Download is not Upload; Restart is not Refresh unless the action really means refresh.
- Decorative icons must use `aria-hidden="true"`; icon-only controls require a meaningful accessible label.

## Typography

- Product UI uses bundled Inter Variable with the system sans stack as fallback.
- Technical identifiers such as commit SHAs, domains, env keys, ports, filenames, branches, and repository identifiers use IBM Plex Mono with a system monospace fallback.
- Live metric numbers use tabular numerals so streaming/background updates do not visually shift digits.
- Prefer weights 400, 500, and 600. Avoid making every hierarchy level bold.
- Normal body, controls, and primary table values should generally be 14–15px. Section titles should generally be 15–16px. Page/object titles should generally be 20–24px.
- Metadata is normally 12–13px. Do not introduce generic 10–11px text; reserve micro typography for exceptional technical annotations only.
- A route should not shrink most supporting text to `text-xs` simply to create hierarchy. Use weight, tone, alignment, and spacing as well.

## Status and chips

- Do not default to a tinted rounded badge for every state.
- Ordinary operational state should usually be a semantic `.status-dot` plus neutral readable text.
- Tinted pills are reserved for states that genuinely need stronger emphasis, such as critical/failed/warning or a special workflow state.
- Do not represent the same state redundantly in multiple columns when another contextual control already communicates it. For example, Projects inventory intentionally does not have a separate Status column because Start/Stop/Deploy and local error cues already expose actionable state.

## Metrics and charts

- Chart color communicates the resource being measured, not generic brand emphasis.
- Never manufacture fake history from a current percentage. Overview trend charts must use real bounded rolling samples or clearly render a current-value meter instead.
- Network counters are cumulative unless an API explicitly returns a rate. Derive transfer rate only from successive valid samples and elapsed time; reset the baseline if a counter decreases.
- Preserve last-known telemetry during successful background refreshes. Do not show a spinner for every host/project telemetry sample; loading indicators are for initial connection or explicit recovery only.
- Do not present policy quotas such as maximum project count as equivalent to physical RAM/CPU/storage/network capacity.

## Empty, status, and notification states

- Empty states should be quiet and neutral. Do not force one generic illustration/icon onto unrelated contexts.
- Status colors must represent real backend/runtime state; do not invent fake progress, unread state, release availability, or success.
- The global notification center only renders unread/update content when real data exists.

## Create Project

- Preserve the current source-first state machine: repository inspection, branch resolution, deployment detection, environment scan, and staged presentation are real workflow states.
- Real repository/detection requests start immediately. Minimum visual duration may make a real operation readable, but must not delay when work begins or fabricate backend work.
- Deployment Type stays in the normal flow, not hidden inside Advanced settings.
- Environment detection stays visible in the normal flow.
- Advanced settings must have a clearly visible disclosure trigger and contain only genuine overrides/diagnostics.
- Do not reintroduce a narrow route-level max-width for this operational workflow.

## Adding UI primitives

Before creating a new component, check whether `ActionButton`, `ActionLink`, `IconButton`, `SectionPanel`, `TableShell`, `PageHeader`, `EmptyState`, `StatusBadge`, `SegmentedChoice`, `InfoDisclosure`, or the shared surface utilities already solve the problem. Prefer a small extension to an existing primitive over a parallel component with slightly different styling.

Do not add a new UI framework or icon library merely to solve consistency. Svelte, Tailwind, and Lucide are sufficient for the current product scope.

## Frontend validation

For frontend-only changes, the required merge gate is:

1. frontend unit tests
2. Svelte/TypeScript checks
3. production frontend build

Fix failures before merging. Backend/infrastructure gates are required only when the change actually touches those contracts.
