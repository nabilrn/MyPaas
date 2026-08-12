# Frontend UI consistency rules

These rules apply to everything under `frontend/`. Keep the implementation small and reuse the existing Svelte/Tailwind primitives before adding abstractions or dependencies.

## Product visual direction

MyPaaS is a flat monochrome operational UI. Prefer consistency, information hierarchy, and real data over decorative variety.

- Normal product chrome is white, black, and neutral gray in light/dark mode.
- Do not use green/emerald as generic brand decoration, generic selection color, or generic primary-action color.
- Primary workflow actions use monochrome inversion: dark control on light surfaces, light control on dark surfaces.
- Color is reserved for actual state and data semantics. Current resource mapping is RAM=green, CPU=blue, storage=amber, network=violet.
- Warning, danger, info, and success colors are allowed only when they communicate real state.
- Keep normal cards flat and neutral. Do not imitate another PaaS mechanically; preserve MyPaaS's own compact operational language.

## Shell ownership

- Desktop sidebar owns primary navigation, user identity/account access, and appearance/theme control.
- Global topbar owns only page context/breadcrumbs and the notification bell.
- Mobile navigation must still expose account and appearance controls because the desktop sidebar is hidden.
- The notification center is generic infrastructure for real platform/release/deployment/resource events. Never fabricate unread state or release availability.

## Layout

- Operational routes use the shared `page-shell`. Do not introduce route-specific centered `max-w-*` wrappers for the whole page unless the content is genuinely document/dialog-like.
- Field-level `max-w-*` is allowed when it improves readability. Constrain the field, not the entire operational workflow.
- Parent layouts own external spacing between sections. Components own only their internal padding/gaps.
- Avoid nested cards used only to create more whitespace. Use dividers and layout grids when the content belongs to one surface.

## Surfaces and elevation

Use only the established surface hierarchy:

- `.surface`: normal card/content container. Flat, bordered, no shadow.
- `.surface-muted`: subtle inset/secondary grouping. No shadow.
- `.overlay`: floating UI such as menus, popovers, notification panels, and modal surfaces. Shadow is allowed here.

Do not add arbitrary `shadow-*` classes to normal cards, tables, stat tiles, headers, or toolbars. Semantic alerts may use tinted backgrounds, but an alert tone is not a new card/elevation type.

## Controls

- Reuse `ActionButton` for text buttons, `ActionLink` for text navigation actions, and `IconButton` for icon-only utility/repeated actions before writing a generic raw control.
- Visible workflow actions should normally combine a Lucide icon with a readable text label: New project, Refresh, Deploy, Start, Stop, Save, Retry, Import, Download, and similar actions.
- Icon-only controls are reserved for compact chrome/utility actions such as notifications, overflow menus, clear input, reveal/hide, or sidebar collapse.
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

- Product UI uses Inter Variable once bundled through the frontend dependency graph; use the system sans stack only as fallback.
- Technical identifiers such as commit SHAs, domains, env keys, ports, and filenames use IBM Plex Mono once bundled; keep a system monospace fallback.
- Live metric numbers use tabular numerals so polling does not visually shift digits.
- Prefer weights 400, 500, and 600. Avoid making every hierarchy level bold.

## Metrics and charts

- Chart color communicates the resource being measured, not generic brand emphasis.
- Never manufacture fake history from a current percentage. Overview trend charts must use real bounded rolling samples or clearly render a current-value meter instead.
- Network counters are cumulative unless an API explicitly returns a rate. Derive transfer rate only from successive valid samples and elapsed time; reset the baseline if a counter decreases.
- Do not present policy quotas such as maximum project count as equivalent to physical RAM/CPU/storage/network capacity.

## Empty, status, and notification states

- Empty states should be quiet and neutral. Do not force one generic illustration/icon onto unrelated contexts.
- Status colors must represent real backend/runtime state; do not invent fake progress, unread state, release availability, or success.
- The global notification center only renders unread/update content when real data exists.

## Adding UI primitives

Before creating a new component, check whether `ActionButton`, `ActionLink`, `IconButton`, `SectionPanel`, `TableShell`, `PageHeader`, `EmptyState`, `StatusBadge`, or the shared surface utilities already solve the problem. Prefer a small extension to an existing primitive over a parallel component with slightly different styling.

Do not add a new UI framework or icon library merely to solve consistency. Svelte, Tailwind, and Lucide are sufficient for the current product scope.

## Frontend validation

For frontend-only changes, the required merge gate is:

1. frontend unit tests
2. Svelte/TypeScript checks
3. production frontend build

Fix failures before merging. Backend/infrastructure gates are required only when the change actually touches those contracts.
