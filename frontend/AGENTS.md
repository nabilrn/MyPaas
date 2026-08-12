# Frontend UI consistency rules

These rules apply to everything under `frontend/`. Keep the implementation small and reuse the existing Svelte/Tailwind primitives before adding abstractions or dependencies.

## Product visual direction

MyPaaS is a flat, neutral operational UI. Prefer consistency and information hierarchy over decorative variety.

- Keep normal surfaces neutral in light and dark mode.
- Use emerald as a restrained accent for primary actions, selected state, and success semantics; do not use it as decoration for generic utility actions.
- Use warning, danger, info, and success colors only when they communicate actual state.
- Do not imitate Vercel/Railway styling mechanically; preserve MyPaaS's own clean visual language.

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
- Primary page actions must normally keep a visible text label. Icon-only controls are for compact utility actions such as refresh, copy, reveal, expand, or repetitive row actions.
- Canonical action variants are `primary`, `secondary`, `ghost`, `danger`, and `ghostDanger`. Legacy IconButton aliases may exist for compatibility but new code should not use them.
- Standard controls should align to the 36px visual height. Keep the existing 44px coarse-pointer/touch target behavior.
- Use `LoaderCircle` from Lucide for loading indicators. Do not add custom border spinners.

## Icons

- `@lucide/svelte` is the single source for generic product/UI icons.
- Brand marks may use a local centralized SVG/component when Lucide intentionally does not provide the brand.
- Do not use emoji or Unicode glyphs as UI status/action icons.
- Icon semantics must match the operation: Stop is not Pause; Download is not Upload; Restart is not Refresh unless the action really means refresh.
- Decorative icons must use `aria-hidden="true"`; icon-only controls require a meaningful accessible label.

## Empty, status, and notification states

- Empty states should be quiet and neutral. Do not force one generic illustration/icon onto unrelated contexts.
- Status colors must represent real backend/runtime state; do not invent fake progress, unread state, release availability, or success.
- The global notification center is generic infrastructure for future release updates, deployment alerts, and resource notifications. Only render unread/update content when real data exists.

## Adding UI primitives

Before creating a new component, check whether `ActionButton`, `ActionLink`, `IconButton`, `SectionPanel`, `TableShell`, `PageHeader`, `EmptyState`, `StatusBadge`, or the shared surface utilities already solve the problem. Prefer a small extension to an existing primitive over a parallel component with slightly different styling.

Do not add a new UI framework or icon library merely to solve consistency. Svelte, Tailwind, and Lucide are sufficient for the current product scope.

## Frontend validation

For frontend-only changes, the required merge gate is:

1. frontend unit tests
2. Svelte/TypeScript checks
3. production frontend build

Fix failures before merging. Backend/infrastructure gates are required only when the change actually touches those contracts.
