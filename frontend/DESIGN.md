# MyPaaS Frontend Design Contract

`frontend/DESIGN.md` is the visual source of truth for the authenticated MyPaaS application.

Any change to layout, spacing, surfaces, typography, navigation, tables, charts, loading behavior, route chrome, or interaction affordances MUST follow this document. Do not invent route-local visual grammar when the same problem exists elsewhere.

MyPaaS is an operational single-host control plane. It should feel like one compact desktop workspace, not a collection of unrelated SaaS pages.

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

Avoid decorative gradients, card stacks, unnecessary shadows, oversized whitespace, filler copy, and decorative accent colors.

---

## 2. Application shell

### Desktop shell

- Top header height: **56px**.
- Collapsed primary navigation rail width: **56px**.
- Hover/focus expansion may overlay to approximately **240px**.
- Expanded primary navigation MUST NOT resize or reflow main content.
- Main workspace starts immediately after the primary rail and below the top header.

### Secondary sidebar

Administration and project-detail families use a local secondary sidebar.

- Canonical desktop width: **12rem**.
- The sidebar is structural navigation, not a card.
- Its right divider uses `--workspace-divider` and must remain visible in dark mode.
- Sidebar navigation is the canonical route map for sibling destinations.
- Do not duplicate sidebar destinations as low-value shortcut cards in page content.

### Canonical outer page inset

For route families with a secondary sidebar, the content column owns exactly one outer inset:

- horizontal: **14px** (`px-3.5`);
- vertical: **12px** (`py-3`).

The `/projects/:id` project-detail layout is the reference implementation for this parent inset.

Child routes MUST NOT add another route-level outer padding wrapper. A nested `page-shell` inside these families is normalized to full width with zero extra padding.

Do not compensate for inconsistent child anatomy by changing this outer inset route-by-route.

### Canonical project-detail horizontal gutter

For every `/projects/:id/*` route, the final left/right content alignment is defined by the geometry already used by **Overview, Deployments, and Logs**. Those three routes are the canonical visual reference for project-detail horizontal spacing.

The parent `main` inset and the inner content gutter are two separate layers:

- parent project `main`: **14px** horizontal (`px-3.5`);
- ordinary section/body/row content inside the route surface: **16px** horizontal (`px-4` or equivalent);
- shared page/section heading bars may use **20px** horizontal (`px-5` / `.panel-header`) when matching Deployments/Logs header geometry.

Rules:

- first-level route surfaces and divider lines remain full width inside the parent project content column;
- do **not** fix alignment by wrapping the entire leaf page in another `px-4`, because that incorrectly indents the surface/divider itself;
- place horizontal padding on the readable content, heading, toolbar, or row inside the surface;
- Environment, Database data view, General, Source, Resources, Webhook, and Danger zone MUST align their readable content to the same gutter family as Overview, Deployments, and Logs;
- Schema Design remains a full-canvas exception because it deliberately leaves the ordinary project-detail surface.

### Canonical Administration horizontal gutter

Administration uses the **same horizontal rhythm** as the project-detail reference above. `/admin/settings`, `/admin/users`, `/admin/backup`, `/admin/migration`, `/admin/mcp`, and `/admin/audit-logs` do not define a separate spacing system.

- parent admin `main`: **14px** horizontal (`px-3.5`);
- Administration route heading: **20px** horizontal (`px-5`);
- ordinary admin section rows/body content: **16px** horizontal (`px-4` or equivalent);
- shared `TableShell` / `.panel-header` / toolbar geometry remains canonical for Users and Audit logs;
- first-level section borders/dividers remain full width inside the admin content column.

Do not zero out row padding in Administration to make a section look wider. The surface may be full width; the readable content inside it still follows the 16px/20px gutter contract.

If a future Administration page looks horizontally inconsistent, compare its heading/body alignment against project Overview, Deployments, and Logs before inventing a route-local padding value.

If a future project-detail page looks horizontally inconsistent, compare it against `/projects/:id`, `/projects/:id/deployments`, and `/projects/:id/logs` before changing any padding value.

---

## 3. Route anatomy

Every ordinary authenticated leaf route follows the same composition:

```text
breadcrumb / global header
secondary sidebar | route content
                    page heading or operational header
                    first section
                    next section
```

### Page heading

A normal leaf page heading is:

- title: `text-lg`, semibold;
- subtitle: at most one short sentence;
- left aligned to the same x-origin as the readable section content below it;
- horizontally padded using the canonical project-detail gutter when inside `/projects/:id/*`;
- separated from following content by deliberate section rhythm, not a floating card.

Do not create a black nested title strip, rounded title card, or second application header inside a route.

### Operational header exception

A project operational action bar may exist only where lifecycle action is a primary task:

- Project Overview;
- Deployments.

Logs, Environment, Database, General, Source, Resources, Webhook, and Danger zone MUST NOT repeat project name/status plus `Deploy again` above their own content. Breadcrumb + project sidebar already establish context.

### Administration heading ownership

`/admin/+layout.svelte` owns the Administration page title and subtitle. Child admin routes MUST NOT render a second page-level title for the same route. Child routes begin with their first meaningful section.

---

## 4. Dashboard surface invariant

This applies to every authenticated dashboard route, including Project Overview, Deployments, Logs, Environment, Database Studio, project settings leaves, Containers, Users, Audit logs, Backup, Migration, MCP, and Administration General.

Ordinary authenticated regions use **one base workspace surface**.

The following MUST NOT create hierarchy by changing neutral background tone:

- first-level sections;
- section headers and bodies;
- metric cells;
- table toolbar/header/body rows;
- pagination strips;
- ordinary setting rows;
- idle inputs and secondary controls;
- neutral hover states.

Hierarchy is communicated by:

- 1px dividers;
- stronger/quieter stroke contrast;
- alignment;
- typography;
- state dots/icons;
- semantic data marks.

Different fills are reserved for real semantic/layer changes: warnings/errors/success/info, primary/destructive actions, overlays, technical output, and chart data marks.

---

## 5. Sections, rows, and settings geometry

### First-level sections

- No rounded outer card silhouette.
- No shadow.
- No large external card gap.
- Use a bottom divider between consecutive sections.
- Section header padding is typically `12px 16px`.
- Section body rows use the same x-origin.

### Canonical settings row

Settings/configuration pages use one row grammar:

```text
[label column] [value / control column] [optional row actions]
```

Desktop target:

- label column: **9–12rem**;
- row vertical padding: **12px**;
- divider between rows;
- content column remains flexible but controls do not stretch arbitrarily.

### Control width rule

Editable controls should use the smallest width that preserves usability.

- ordinary select/input: `max-w-md` to `max-w-xl`;
- repository/URL/technical text may use a wider readable column;
- a single simple select MUST NOT span a 1400px desktop canvas;
- destructive confirmation input should normally stay `max-w-xl`;
- full width is reserved for search bars, logs, tables, code/JSON editors, and genuinely canvas-wide tools.

### General-information ownership

Project General owns project identity and public endpoint information only. Source details belong to Source. Resource limits belong to Resources. Do not repeat sibling-route data merely because it is available.

---

## 6. Navigation and redundancy

Sidebar is the primary navigation source for sibling routes.

A page may link to a sibling route only when the link is attached to unique summary information, for example:

- Latest deployment → `View all` Deployments;
- an actionable failure → `View logs`;
- a real summary metric → detail page.

Do NOT add navigation-only blocks such as `Project settings` or `Database Studio` when the secondary sidebar already exposes those routes and the block adds no unique status/data.

Do not repeat the same nearby state, count, label, or action twice. Examples to avoid:

- `0 variables` plus an empty state saying there are no variables;
- `171 visible` plus `Showing 171 of 171 lines`;
- one-page datasets with disabled Previous/Page 1/Next chrome;
- project name in breadcrumb, repeated project bar, and repeated General row when the intermediate bar adds no operational value.

---

## 7. Typography

Primary family: Inter Variable.
Technical identifiers/output: IBM Plex Mono.

Targets:

- body/control/table text: **14–15px**;
- section titles: **15–16px**;
- supporting text: **13px**;
- technical mono: **12–13px**;
- true object/page titles: **18–24px** when needed.

Compactness comes from layout and chrome, not unreadable type.

---

## 8. Color semantics

Default chrome is monochrome neutral. Color must encode meaning.

Resource semantics are stable throughout the product:

- CPU: **blue** (`--chart-cpu`);
- memory/RAM: **green** (`--chart-memory`);
- storage: **amber** (`--chart-storage`);
- network: **violet** (`--chart-network`).

Resource progress bars, chart marks, and matching resource indicators SHOULD use these tokens. Do not reduce real resource telemetry to neutral gray when semantic tokens already exist.

Success/warning/danger/info colors are reserved for real state. Primary workflow actions use monochrome inversion.

---

## 9. Controls

Inputs and actions belong to one shared control system.

### Geometry

- canonical desktop height: **36px**;
- canonical text: **14px**;
- icon-only controls: **36×36px**;
- `ActionButton` and `ActionLink` size variants may change horizontal padding, not height/font size;
- adjacent controls align on the same top/bottom edges;
- coarse-pointer layouts use **44px** targets consistently.

### Visual state

Idle controls inherit the workspace fill. Hover raises boundary/text contrast without decorative fill. Focus uses the shared monochrome focus treatment. Disabled/read-only controls remain restrained.

Use existing shared controls instead of route-local button/input palettes.

---

## 10. Tables and pagination

Operational tables are workspace content, not cards.

Order:

```text
table context / section heading
toolbar
column header
rows
pagination/result count only when needed
```

Rules:

- toolbar/header/rows/pagination share the workspace fill;
- separators define the grid;
- stable datasets use deliberate column geometry;
- technical identifiers truncate rather than stretching the table;
- textual identity is left aligned; numeric/status columns may center/right align as appropriate;
- typical row height: roughly **52–60px**.

### Pagination visibility

Do not render pagination chrome when the dataset demonstrably fits on one page. A single-row Users table must not show `Previous · Page 1 · Next` or a redundant `Owners: 1-1` footer.

---

## 11. Metrics and charts

Charts are data visualization, not decorative cards.

- plot areas use the workspace fill;
- data marks may use semantic resource/metric color;
- grid lines remain faint;
- do not fabricate history;
- preserve last-known telemetry during successful background refresh;
- network counters remain cumulative unless the API explicitly returns a rate.

### Overview chart density

Overview is an observability dashboard. When time-series data exists, charts should use meaningful vertical space. A chart compressed to roughly 50px high is too small for a wide desktop dashboard.

Compact Overview charts should target approximately **96–120px** height while keeping the section itself compact.

### Runtime usage

CPU and memory usage show:

```text
used / allocation    percentage
semantic progress bar
```

Do not repeat the allocation again underneath the same bar.

---

## 12. Empty states and low-value chrome

An empty page should become simpler, not noisier.

- Do not repeat zero counts next to an empty-state sentence that communicates the same fact.
- Disable or hide actions that cannot operate on empty data when keeping them visible adds no orientation value.
- Notes such as `latest 5000 lines kept in memory` belong to non-empty log state, not an empty console.
- Empty states remain concise: title + one actionable sentence.

---

## 13. Project-detail route contract

The project secondary sidebar is the persistent navigation model for all `/projects/:id/*` leaves.

Routes:

- Overview
- Deployments
- Logs
- Environment
- Database
- General
- Source
- Resources
- Webhook
- Danger zone

Rules:

- Overview owns operational summary/observability.
- Deployments owns deployment history and deployment lifecycle context.
- Logs owns log filtering/export and technical output.
- Environment owns environment variable management.
- Database owns Database Studio.
- General owns identity/public URL/deployment type.
- Source owns repository/image/branch/base-directory/runtime-source configuration.
- Resources owns resource profile and limits.
- Webhook owns webhook endpoint/secret/setup.
- Danger zone owns deletion only.

No route should restate data whose canonical owner is a sibling route unless the duplicate is required to execute the current task.

---

## 14. Administration route contract

Administration uses the same outer geometry and inner readable-content gutters as the project-detail source of truth for General, Users, Backup, Migration, MCP, and Audit logs.

- Parent layout owns route heading/subtitle and gives it the canonical **20px** horizontal heading gutter.
- Child pages begin at the same content x-origin.
- Ordinary admin rows use the canonical **16px** horizontal content gutter while their parent border/divider remains full width.
- General uses flat rows, not a rounded settings card silhouette.
- Backup/Migration/MCP use the same divider-based section grammar.
- Users/Audit use the shared table grammar, which already owns its header/toolbar/cell gutters.

MCP token actions that act on the same secret belong in the same row/action group. Avoid separate action strips for Reveal/Copy/Regenerate when they operate on one token.

---

## 15. Database Studio

Database Studio is a project leaf, not a separate embedded application.

- Do not render an additional dark/black nested application header.
- `Database Studio` uses the normal project leaf heading grammar.
- `Schema design` is a contextual page action aligned with that heading.
- Connection status and table browsing are ordinary sections below it.
- The table/data workspace may still consume the available width because it is a real data tool.

---

## 16. Audit logs

Audit logs should emphasize meaningful control-plane mutations.

- Actor/IP belongs in a dedicated compact column or expanded metadata, not repeated as secondary text under every action when it reduces scanability.
- High-frequency probe/detection events should not visually dominate actual mutations. If the backend returns them, the UI should filter or classify repetitive read/probe events when doing so does not hide meaningful security/destructive activity.
- Expanded metadata uses the canonical technical-output surface.

Never hide destructive/security/failure audit evidence merely to make the table look cleaner.

---

## 17. Canonical technical/output surface

Host Shell output is the canonical palette for technical output across authenticated MyPaaS.

This includes:

- project logs;
- deployment/build logs;
- audit metadata JSON;
- MCP setup prompt;
- migration commands;
- repository inspection details;
- configuration examples.

Technical output may use its own surface because it represents a different content mode, not a hierarchy trick.

---

## 18. Theme paint and loading

Theme state must be correct before first paint.

- stored/system theme is applied before Svelte hydration;
- ordinary client-side navigation does not trigger a global blocking loader;
- local operations use local loading state;
- polling/SSE/background refresh must not destroy usable last-known data unnecessarily.

---

## 19. Full-canvas tools

ERD/schema design and Host Shell are workspace tools, not cards.

They may consume the useful workspace height/width and own their interaction model while preserving the global shell, divider, control, and technical-output contracts.

---

## 20. Responsive and accessibility

- Desktop is the primary density target.
- Mobile exposes the same authorized destinations.
- Tables may use a deliberate compact/mobile representation rather than shrinking type below the floor.
- Preserve keyboard access and visible focus.
- Hover-only behavior requires keyboard/focus equivalent where needed.
- Color is never the only signal for critical state.
- Avoid noisy live announcements during polling/background refresh.

---

## 21. Explicit anti-patterns

Do not introduce:

- first-level rounded card stacks;
- route-local outer padding that conflicts with the parent family inset;
- nested application headers inside leaf pages;
- repeated project operational bars on non-operational settings/data leaves;
- sibling-route shortcut cards with no unique data;
- controls stretched across the full desktop canvas without a functional reason;
- pagination for a one-page dataset;
- repeated zero counts + empty-state copy;
- alternating neutral section fills;
- route-local control/console palettes;
- nested border rectangles at every level;
- decorative shadows on ordinary sections;
- fake telemetry history;
- semantic resource telemetry rendered as meaningless neutral gray;
- explanatory copy that only restates the label directly above it.

---

## 22. Change rule

When implementing authenticated UI work:

1. Read this file first.
2. Identify the parent route family and its canonical outer inset.
3. Identify the single owner of each piece of information and each action.
4. Fix shared behavior centrally before adding route-specific CSS.
5. Keep route-specific exceptions only when workflow semantics genuinely differ.
6. Audit loading, empty, disabled, focus, error, destructive, and responsive states.
7. Run frontend unit tests, Svelte/TypeScript checks, production build, and repository CI before merge.

The target is one coherent operational workspace: **one outer geometry, one section grammar, one control system, one technical-output palette, semantic color where data requires it, and no redundant navigation or state.**
