# MyPaaS Frontend Design Contract

`frontend/DESIGN.md` is the visual source of truth for the authenticated MyPaaS application.

Any change to layout, spacing, surfaces, typography, navigation, tables, charts, loading behavior, or interaction chrome MUST follow this document. Do not invent a route-local visual grammar that conflicts with these rules.

MyPaaS is an operational single-host control plane. It should feel like a compact desktop workspace, not a generic SaaS dashboard assembled from cards.

---

## 1. Product character

The authenticated UI is:

- operational rather than promotional;
- compact rather than spacious;
- flat rather than elevated;
- monochrome by default;
- information-dense without tiny text;
- structured by alignment, padding, typography, and strokes;
- explicit about real runtime state.

Avoid decorative gradients, card stacks, unnecessary shadows, oversized whitespace, decorative accent colors, and explanatory filler.

---

## 2. Application shell

### Desktop

- Top header height: **56px**.
- Collapsed navigation rail width: **56px**.
- Hover/focus may expand navigation to approximately **240px** as an overlay.
- Expanded navigation MUST NOT resize or reflow main content.
- Main workspace begins immediately after the 56px rail and below the 56px header.

### Workspace geometry

Authenticated operational content is edge-to-edge inside the workspace.

- No generic outer page margin between rail and operational content.
- No generic route-level `max-w-*` wrapper for ordinary operational pages.
- Consecutive first-level operational regions have **0px external gap**.
- Use a structural divider instead of whitespace/card separation.
- Dialogs, onboarding, and document-like content may constrain their own readable width.

---

## 3. Dashboard surface invariant

This rule applies to **every authenticated dashboard route**, including Projects, Project Detail, Deployments, Logs, Environment, Settings, Database Studio, Containers, Users, Audit, backup/migration, MCP, and other administration pages.

### One workspace fill

Ordinary authenticated workspace regions use **one base workspace surface**.

The following MUST NOT create hierarchy by changing neutral background tone:

- first-level sections;
- section headers and bodies;
- metric cells;
- table toolbar/header/body rows;
- pagination strips;
- shortcut/configuration cells;
- idle inputs and secondary controls;
- ordinary hover states;
- nested neutral panels used only for grouping.

Hierarchy is communicated with:

- 1px borders/dividers;
- stronger or quieter stroke contrast;
- alignment and spacing;
- typography;
- state dots/icons;
- focus/selection indicators.

A neutral section must not be `#141414` next to another neutral section at `#0a0a0a` merely to make them look separate. If two regions need separation, keep their fill identical and draw the boundary.

### Permitted fill exceptions

A different fill is allowed only when the surface has a different semantic/layer role:

- semantic warning/error/success/info feedback;
- primary or destructive workflow actions;
- true overlays such as modal/popover/context menu;
- technical output/terminal surfaces defined below;
- chart data marks/fills that encode a measured resource;
- navigation active-state treatment when required by the navigation contract.

Do not turn these exceptions into generic section backgrounds.

### Divider grids

A `gap-px` operational grid may use the divider color as the gap substrate, but **every cell uses the same workspace fill**. The divider must read as a stroke, not as a contrasting card matrix.

---

## 4. Strokes and elevation

- Structural dividers are 1px.
- Header/sidebar/section/table strokes must remain subordinate to content.
- First-level workspace regions do not use rounded outer card silhouettes.
- First-level workspace regions do not use shadows.
- Controls may use a small radius because their boundary is interactive.
- Overlays may use border, radius, and shadow because they genuinely float above the workspace.
- Avoid border nesting where every parent and child draws a full rectangle.

---

## 5. Navigation

### Sidebar

- Default desktop state is the 56px icon rail.
- Expansion is a non-reflowing hover/focus overlay.
- Sidebar owns navigation only.
- Active navigation may use a neutral fill because it communicates selection, not section hierarchy.
- Idle navigation items do not show persistent borders.

### Header

Header owns breadcrumbs/page context, notifications, appearance/theme control, account access, and suitable route-level primary actions.

Root routes should not repeat the same title in the header and body.

---

## 6. Typography

Primary family: Inter Variable.
Technical identifiers/output: IBM Plex Mono.

Targets:

- body/control/table text: **14–15px**;
- section titles: **15–16px**;
- supporting text: **13px**;
- technical mono: **12–13px**;
- real object/page titles: **20–24px** when needed.

Compactness comes from layout and chrome, not unreadable type.

---

## 7. Color

Default chrome is monochrome neutral in light and dark themes. Color is semantic, not decorative.

Resource semantics:

- RAM / memory: green;
- CPU: blue;
- storage: amber;
- network: violet.

Success/warning/danger/info colors are reserved for real state. Primary workflow actions use monochrome inversion.

---

## 8. Controls

Editable text inputs, search fields, selects, comboboxes, and textareas share one neutral grammar.

### Idle

- fill inherits the parent workspace surface;
- boundary is visible but low contrast;
- idle control must not become a contrasting rectangle.

### Hover

- fill remains unchanged;
- boundary/text may become slightly clearer;
- no decorative accent color.

### Focus

- focus is the strongest normal boundary state;
- use the shared monochrome focus border/ring;
- focus remains visible in light and dark themes.

### Disabled/read-only

Prefer muted text and boundary treatment. A muted fill is allowed only when necessary to make non-editability unambiguous; it must remain restrained.

Ordinary fields target **36px** visual height and **14px** text.

Reuse `ActionButton`, `ActionLink`, `IconButton`, `SegmentedChoice`, and shared field utilities rather than creating route-local button/input palettes.

---

## 9. Tables

Operational tables are workspace content, not cards.

Order:

```text
table title/context
table toolbar
column header
rows
pagination/result count
```

Rules:

- toolbar, header, rows, and pagination use the same workspace fill;
- row/column separators define the table structure;
- idle and hover rows keep the same fill;
- selected rows keep the same fill and use a stronger stroke/selection marker;
- stable datasets use deliberate column geometry;
- long technical identifiers truncate rather than stretching the table;
- headers are centered by default, textual identity left aligned, numeric metrics right aligned;
- typical row height is roughly **52–60px**.

The grid must never dominate the data.

---

## 10. Metrics and charts

Charts are data visualizations, not decorative cards.

- Chart containers participate in the shared workspace fill.
- Do not create a neutral contrasting plot rectangle merely for decoration.
- Resource-colored marks/fills are allowed because they encode data.
- Grid lines are faint.
- Use real bounded rolling history only.
- Never fabricate history from one current value.
- Preserve last-known telemetry during successful background refresh.
- Network counters are cumulative unless the API explicitly returns a rate.

---

## 11. Canonical technical/output surface

**Host Shell output is the canonical palette for technical output throughout the authenticated dashboard.**

Every terminal/log/build-output/command-output/JSON-dump/technical `<pre>` surface uses the same technical background, border, foreground, muted text, and selection treatment as Host Shell output.

This includes, where applicable:

- Host Shell output;
- project logs;
- deployment/build logs;
- audit metadata JSON;
- MCP setup prompt/output;
- migration commands/output;
- repository inspection technical details;
- configuration command examples.

Technical output is an intentional exception to the one-workspace-fill rule because it represents a different content mode, not a section hierarchy level.

Do not create per-route terminal palettes such as one page using `neutral-900`, another `gray-950`, and another a muted workspace fill.

---

## 12. Theme paint and navigation

Theme state must be correct **before first paint**.

- `app.html` applies the stored theme, or system preference when no explicit theme exists, before Svelte head/hydration.
- The hydrated theme store must reconcile with that prepaint state rather than repainting the opposite theme.
- Full document reloads, stale-asset recovery, and direct deep links must not flash the opposite theme.
- Ordinary client-side route navigation must not show a global blocking loader merely because navigation is in progress.

Global main-content loading is reserved for initial authenticated account/resource loading where the workspace cannot meaningfully render yet.

Repository inspection, mutations, refreshes, pagination, polling, SSE, and other local operations use local progress only.

---

## 13. Full-canvas tools

ERD/schema design and Host Shell are workspace tools, not cards.

### ERD / schema design

- may occupy the full workspace below the global header;
- starts after the desktop rail on desktop;
- page body does not scroll while the canvas is active;
- canvas owns pan/zoom interaction;
- pointer position anchors zoom behavior;
- shell/header chrome follows the same workspace fill and stroke contract.

### Host Shell

- terminal consumes the useful workspace height;
- toolbar/session status stays compact;
- command input belongs at the workspace edge/bottom;
- terminal/output area defines the canonical technical palette.

---

## 14. Status and semantic feedback

- Ordinary runtime state is usually a status dot + readable text.
- Strong tinted badges are reserved for states requiring emphasis.
- Semantic warning/error/success/info surfaces may use restrained semantic tint.
- Do not repeat the same state in multiple nearby components without a functional reason.
- Never hide destructive, security, stale-state, blocking validation, or failure information to make a page look cleaner.

---

## 15. Copywriting

Operational copy is factual and short.

Prefer state + data + action. Avoid generic explanatory paragraphs that restate the interface.

Default budget:

- page subtitle: 0–1 short sentence;
- panel description: omit if title/data already explain it;
- helper text: one short line for a non-obvious constraint;
- empty state: title + one actionable sentence;
- warning: concise reason + consequence + next action.

---

## 16. Responsive and accessibility

- Desktop remains the primary density target.
- Mobile navigation exposes the same authorized destinations.
- Tables may use a deliberate compact/mobile representation instead of shrinking typography below the floor.
- Preserve keyboard access and visible focus states.
- Hover-only behavior requires an equivalent keyboard/focus path where needed.
- Color is never the only indicator of critical state.
- Avoid noisy live announcements during polling/background refresh.

---

## 17. Explicit anti-patterns

Do not introduce:

- first-level card stacks separated by large gaps;
- alternating neutral section fills;
- dark-mode neutral blocks that visually stripe the page;
- neutral hover fills that are stronger than the surrounding workspace;
- table headers/toolbars/pagination with independent neutral fills;
- route-local input/button/console palettes;
- nested border rectangles at every hierarchy level;
- decorative shadows on ordinary sections;
- full-page spinner for normal client-side navigation or local operations;
- fake telemetry history;
- decorative accent colors;
- tiny metadata everywhere.

---

## 18. Change rule

When implementing authenticated UI work:

1. Read this file first.
2. Identify the shared primitive or root workspace contract that owns the behavior.
3. Fix systemic behavior centrally rather than patching dozens of routes.
4. Keep route-specific overrides only when the workflow is genuinely different.
5. Audit semantic, loading, empty, disabled, focus, and error states after visual changes.
6. Run frontend unit tests, Svelte/TypeScript checks, and production build before merge.

The goal is one coherent operational workspace: **same neutral fill, structural strokes, semantic color only when meaningful, and one canonical technical-output palette.**
