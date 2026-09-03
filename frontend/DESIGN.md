# MyPaaS Frontend Design Contract

`frontend/DESIGN.md` is the visual source of truth for the authenticated MyPaaS application.

Any change to layout, spacing, surfaces, typography, navigation, tables, charts, loading behavior, or interaction chrome MUST follow this document. If a requested design intentionally changes this contract, update this file first or in the same change. Do not invent a parallel visual grammar inside an individual route.

MyPaaS is an operational single-host control plane. It should feel like a compact desktop workspace, not a generic SaaS dashboard assembled from cards.

---

## 1. Product character

The UI is:

- operational rather than promotional;
- compact rather than spacious;
- flat rather than elevated;
- monochrome by default;
- information-dense without using tiny text;
- structured by alignment, internal padding, dividers, and background tone;
- explicit about real runtime state.

The UI is NOT:

- a stack of rounded dashboard cards;
- a collection of isolated stat tiles;
- a decorative "premium SaaS" layout with large whitespace;
- a wireframe made from visible borders around every container;
- an excuse to add colors, gradients, shadows, badges, or copy that do not communicate state.

Reference quality: a professional database/infrastructure console where navigation, tables, editors, terminal views, and canvas tools share one continuous workspace. Do not mechanically clone another product.

---

## 2. Application shell

### Desktop

- Top header height: **56px**.
- Collapsed navigation rail width: **56px**.
- The rail is always present on authenticated desktop routes.
- Hover/focus may expand navigation to approximately **240px** as an overlay.
- Expanded navigation MUST NOT resize or reflow main content.
- Main workspace begins immediately after the 56px rail and immediately below the 56px header.

### Workspace geometry

Authenticated operational content is **edge-to-edge inside the application workspace**.

```text
GOOD
+------+----------------------------------------------------+
| rail | header                                             |
|      +----------------------------------------------------+
|      | workspace section header                          |
|      |----------------------------------------------------|
|      | content                                            |
|      |----------------------------------------------------|
|      | next section                                       |
+------+----------------------------------------------------+

BAD
+------+----------------------------------------------------+
| rail | header                                             |
|      +----------------------------------------------------+
|      |   gap   + rounded card +                          |
|      |         +--------------+   gap                    |
|      |         + rounded card +                          |
+------+----------------------------------------------------+
```

Rules:

- No generic outer page margin between rail and operational content.
- No generic outer page padding above/below authenticated operational workflows.
- Do not add route-level centered `max-w-*` containers for normal operational screens.
- Internal document-like content, dialogs, onboarding, and narrow forms may constrain their own readable width when genuinely useful.

---

## 3. Spacing ownership

Whitespace belongs **inside content**, not between connected workspace sections.

### Page level

- Consecutive operational sections use **0px external gap**.
- Do not separate first-level sections with `mb-4`, `mb-5`, `gap-4`, `space-y-6`, etc.
- Use a divider when two first-level regions need separation.

### Internal rhythm

Preferred internal rhythm:

- 4px: micro relationships;
- 8px: compact inline controls;
- 12px: small groups / toolbar relationships;
- 16px: normal content grouping;
- 20px: section content padding when the content needs breathing room.

Do not solve hierarchy by continuously increasing whitespace.

---

## 4. Surface hierarchy

The application uses three surface classes conceptually.

### A. Workspace surface

Examples: Projects inventory, Host resources, Containers, Ports, Users, Audit, project detail sections, Database Studio.

Rules:

- flat;
- same base background as the workspace;
- **no full rectangular outline at first level**;
- **no outer rounded-card silhouette at first level**;
- separated from adjacent first-level content by subtle structural dividers;
- no shadow.

`SectionPanel` and `TableShell` are workspace sections by default.

### B. Inset/control surface

Examples: input fields, textarea, segmented controls, search boxes, a terminal viewport, code surfaces, compact internal option groups.

Rules:

- border is allowed when it defines an actual control boundary;
- small radius is allowed;
- subtle alternate background is allowed;
- no decorative shadow.

### C. Overlay surface

Examples: popover, account menu, notification panel, modal, expanded sidebar, context menu.

Rules:

- border allowed;
- radius allowed;
- shadow allowed because the surface is actually floating;
- must clearly layer above the workspace.

### Anti-pattern: border nesting

Avoid this:

```text
page border
  -> section border
      -> metric border
          -> chart border
              -> grid border
```

At most one visible boundary should normally define a given level of hierarchy.

---

## 5. Dividers and strokes

Dividers communicate structure. They must not dominate the screen.

- Use 1px structural dividers.
- Prefer low-contrast mixed/transparent neutral borders rather than raw strong gray everywhere.
- Header bottom divider: subtle.
- Sidebar right divider: subtle.
- Section header bottom divider: visible enough to establish hierarchy, but quieter than a form-control border.
- Table row/column dividers: even quieter.
- Avoid full outer borders around first-level workspace regions.

A screen should still look clean if viewed from a distance. If the first thing visible is a grid of rectangles, stroke density is too high.

---

## 6. Navigation

### Sidebar

- Default desktop state is the 56px icon rail.
- Active item uses **background/fill + text/icon tone**, not an outlined box.
- Idle items must not show persistent borders.
- Hover may use a quiet background change.
- Workspace and Administration groups may use one quiet separator.
- Do not add a manual sidebar collapse button.

### Header

- Header owns breadcrumbs/page context, notifications, appearance, account, and route-level primary actions where appropriate.
- Root routes should not duplicate the same page title inside body content.
- Breadcrumbs on project routes use `Projects / project / section` hierarchy.

---

## 7. Typography

Primary family: Inter Variable.
Technical identifiers: IBM Plex Mono.

Minimum normal UI targets:

- primary body/control/table text: **14–15px**;
- section titles: **15–16px**;
- metadata/supporting text: **13px**;
- technical mono: **12–13px**;
- object/page titles: **20–24px** where a real object title is necessary.

Rules:

- Do not make normal interface copy 10–11px.
- Compactness comes from layout/chrome, not unreadable type.
- Prefer font weights 400/500/600.
- Do not bold every hierarchy level.
- Streaming metrics use tabular numerals.

---

## 8. Color

Default chrome is monochrome neutral in both light and dark themes.

Color is semantic, not decorative.

Current resource semantics:

- RAM / memory: green;
- CPU: blue;
- storage: amber;
- network: violet.

State semantics may use success/warning/danger/info colors when a real state exists.

Do not use green or any other accent as a generic brand/action/selection color across the application.

Primary workflow actions use monochrome inversion.

---

## 9. Tables

Operational tables are first-class workspace content, not cards containing miniature cards.

### Hierarchy and toolbar

Table-scoped utilities belong to the table chrome, not to a separate form or card above it.

The visual order is:

```text
table title / context
table toolbar
column header
rows
pagination / result count
```

The toolbar owns controls that operate on the displayed dataset, including:

- search;
- filters;
- sorting;
- refresh;
- row-count controls;
- table-scoped primary actions.

Rules:

- Place the toolbar immediately above the column header inside the same `TableShell` workspace.
- Do not create a separate bordered panel merely to hold search/filter controls for a table.
- Toolbar controls use the same shared field tokens and standard control height as adjacent actions.
- Idle toolbar controls stay visually quiet; focus/active state may become the strongest normal control boundary.
- A toolbar without controls must not reserve an empty row.
- Table title/context and table utilities may wrap independently on narrow widths without clipping the primary action.

### Geometry

- Prefer stable deliberate column geometry for predictable datasets.
- Do not let browser auto-layout produce erratic widths for operational inventory tables.
- Short fields such as Status, Branch, Uptime, Updated, and Action should not receive excessive width.
- Project/repository/runtime fields may receive more width.
- Truncate long technical identifiers instead of stretching the table.

### Alignment

- headers: centered by default;
- names/textual identity: left aligned;
- status/action: centered when appropriate;
- numeric metrics: right aligned with tabular numerals;
- technical identifiers: use mono where useful.

### Strokes and states

- table header: one quiet tonal step from the body;
- row separators: subtle;
- column separators: quieter than row separators;
- idle row: neutral table surface;
- hover row: subtle neutral surface change;
- selected row: stronger neutral surface change, still monochrome;
- outer table outline: normally unnecessary when table already belongs to a workspace section.

The table grid must never dominate the data. From a distance, content and hierarchy should be more visible than individual cell boundaries.

Typical row height: roughly **52–60px** for ordinary operational inventory.

---

## 10. Metrics and charts

Charts should behave as data visualizations, not decorative cards.

- The plot area should be visually more important than labels around it.
- Do not wrap every chart in another visible bordered rectangle.
- Grid lines must be faint.
- Use real bounded rolling history only; never fabricate history from one current value.
- Polling/background samples must not trigger blocking page loaders.
- Preserve last-known telemetry while a successful background refresh is in progress.
- Resource color follows the semantic mapping above.

---

## 11. Loading-state contract

Loading states are classified by scope.

### Global blocking loader

Use only for **initial authenticated route/resource loading** where the page cannot meaningfully render yet.

Behavior:

- sidebar remains visible;
- header remains visible;
- child route stays mounted invisibly/out-of-flow so lifecycle requests can execute;
- main workspace visually shows only the MyPaaS loading indicator;
- old-route requests must not keep the new route blocked.

### Local operation state

Use local loading/progress for:

- repository inspection/detection;
- deploy/start/stop/restart;
- form submissions;
- DB write actions;
- explicit refresh;
- shell session commands;
- pagination;
- polling/SSE/background telemetry.

These MUST NOT blank the whole workspace.

Repository inspection is a local workflow state even if it performs network requests.

---

## 12. Full-canvas tools

ERD/schema design and Host Shell are workspace tools, not cards.

### ERD / schema design

- may occupy the full workspace below the global header;
- starts after the desktop rail on desktop;
- page body should not scroll while the canvas is active;
- canvas owns pan/zoom interaction;
- mouse wheel / trackpad gestures may zoom where defined;
- pointer position should anchor zoom behavior;
- toolbars remain compact and structural.

### Shell

- terminal should consume the useful workspace height;
- toolbar/session status stays compact;
- command input belongs at the workspace edge/bottom;
- do not embed the terminal inside multiple card wrappers.

---

## 13. Controls and icons

Reuse existing primitives before adding new ones:

- `ActionButton`
- `ActionLink`
- `IconButton`
- `SectionPanel`
- `TableShell`
- `EmptyState`
- existing field/surface utilities

Use `@lucide/svelte` for generic UI icons.

- Text workflow actions generally combine icon + label.
- Icon-only controls are for compact utility actions.
- Standard visual control height is about 36px while retaining appropriate coarse-pointer targets.
- Do not use emoji/Unicode glyphs as product icons.

### Input hierarchy

Editable text inputs, search fields, selects, comboboxes, and textareas share one neutral control grammar.

Idle state:

- background remains close to the parent surface;
- boundary is visible but low contrast;
- an idle input must not become the strongest rectangle in its section.

Hover state:

- boundary becomes slightly clearer;
- background may shift by only one neutral tonal step;
- do not introduce semantic or decorative accent color.

Focus state:

- focus is the highest-contrast normal input state;
- use the shared monochrome focus border and restrained focus ring;
- the focus indicator must remain visible in light and dark themes;
- do not use success green or another semantic color as generic focus chrome.

Disabled and read-only states:

- use a muted control surface and muted text rather than opacity alone;
- remain distinguishable from editable idle state;
- preserve legibility of existing values.

Sizing:

- ordinary fields target **36px** visual height;
- field text remains **14px**;
- horizontal padding is normally **10–12px**;
- table-toolbar fields align with adjacent `ActionButton size="sm"` controls;
- coarse-pointer environments may increase the hit target without changing desktop density.

Route-local input palettes that recreate border/background/focus colors are prohibited when the shared `.field` primitive owns the state.

---

## 14. Status presentation

- Do not make every state a rounded colored pill.
- Ordinary runtime state should usually be a status dot + readable neutral text.
- Strong tinted badges are reserved for states that require stronger emphasis.
- Do not repeat identical state information in multiple nearby components without a functional reason.

---

## 15. Copywriting inside the product

Operational UI copy should be factual and short.

Prefer:

- `CPU, memory, storage, and network.`
- `Search projects`
- `Not configured`
- `Deployment history`

Avoid explanatory filler that restates obvious UI structure or attempts to sound sophisticated.

Do not invent claims, fake progress, fake release information, or narrative telemetry descriptions.

---

## 16. Responsive behavior

Desktop remains the primary density target, but smaller widths must stay usable.

- Sidebar becomes topbar-controlled mobile navigation.
- Large tables may switch to a deliberate compact/mobile representation rather than compressing every desktop column.
- Do not shrink normal text below the typography floor just to preserve desktop geometry.
- Full-canvas tools remain usable within the available viewport.

---

## 17. Accessibility and interaction

- Preserve keyboard access and visible focus states.
- Icon-only buttons require meaningful accessible labels.
- Decorative icons use `aria-hidden="true"`.
- Hover-only behavior must have a keyboard/focus equivalent where needed.
- Color must not be the only indicator of critical state.
- Loading regions should expose appropriate busy/status semantics without producing noisy announcements during polling.

---

## 18. Explicit anti-patterns

Do not introduce any of these without first changing this contract:

- page-level `gap-4` / `gap-6` card stacks;
- first-level rounded rectangles around every section;
- `border-gray-200` on every possible container;
- nested card-inside-card hierarchy for normal operational content;
- shadows on ordinary sections/tables/toolbars;
- wide empty action rows that consume useful viewport height;
- browser-guessed table widths for stable operational schemas;
- tiny metadata everywhere;
- decorative accent colors;
- full-page spinner for repository inspection or mutations;
- fake telemetry history;
- isolated metric cards when the data naturally belongs to one shared workspace;
- route-local visual systems that bypass shared primitives/tokens.

---

## 19. Change rule

When implementing UI work:

1. Read this file first.
2. Identify the existing shared primitive/token that owns the behavior.
3. Fix the shared grammar when the issue is systemic; do not patch dozens of routes individually.
4. Keep route-specific overrides only when the workflow is genuinely different.
5. If the requested design contradicts this document, update this document deliberately rather than silently diverging.
6. Validate frontend unit tests, Svelte/TypeScript checks, and production frontend build.

The goal is a coherent control-plane workspace whose visual system remains stable as features are added.
