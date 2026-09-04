# MyPaaS Frontend Design Contract

`frontend/DESIGN.md` is the visual and copywriting source of truth for the authenticated MyPaaS application.

Any change to layout, spacing, surfaces, strokes, typography, navigation, tables, charts, loading, copy, or interaction affordances MUST follow this file. Do not invent route-local grammar when an existing canonical screen already solves the same problem.

---

## 1. Product character

MyPaaS is a compact operational workspace.

It is:

- compact rather than spacious;
- flat rather than elevated;
- monochrome by default;
- information-dense without tiny text;
- structured by alignment, typography, and thin strokes;
- explicit about real runtime state;
- written in short, direct language.

Avoid decorative gradients, card stacks, large shadows, oversized whitespace, filler copy, unnecessary neutral fills, and jargon that describes internal architecture instead of the user's task.

---

## 2. Application shell

### Desktop

- top header: **56px**;
- collapsed primary rail: **56px**;
- expanded primary navigation may overlay to roughly **240px** and MUST NOT reflow content;
- Administration and project-detail families use a structural **12rem** secondary sidebar.

### Canonical outer inset

Routes with a secondary sidebar own one parent inset:

- horizontal: **14px** (`px-3.5`);
- vertical: **12px** (`py-3`).

Child routes MUST NOT add a second route-level outer-padding wrapper.

### Project-detail readable gutters

`/projects/:id`, `/projects/:id/deployments`, and `/projects/:id/logs` are the reference for project-detail geometry.

- parent main: `px-3.5 py-3`;
- ordinary rows/body content: **16px** horizontal (`px-4`);
- first-level title/header bars: **20px** horizontal (`px-5`) where the existing header geometry uses it;
- leaf title optical offset: **16px** (`pt-4`) after the parent's top inset.

Overview, Deployments, and Logs already have the correct optical position. Environment, Database, General, Source, Resources, Webhook, and Danger zone align to them without duplicating the operational header.

### Administration gutters

Administration uses the same rhythm:

- parent main: `px-3.5 py-3`;
- route heading: `px-5`;
- body/setting rows: `px-4`;
- first-level dividers remain full width inside the content column.

Do not fix perceived misalignment with arbitrary route-local `mt-*`, `px-*`, or nested `page-shell` values.

---

## 3. Canonical stroke grammar

**Project Overview is the source of truth for strokes and section boundaries.**

The canonical boundary is `--workspace-divider` or an optically equivalent low-contrast 1px line.

Rules:

- use thin, low-contrast dividers;
- ordinary first-level sections remain flat;
- use borders to separate rows/sections, not to wrap every object;
- avoid strong box grids, nested rectangles, and repeated outer outlines;
- dark and light themes must have equivalent optical hierarchy;
- if another route conflicts with Overview's stroke strength, Overview wins.

Inset controls, dialogs, overlays, and technical output may retain an explicit boundary because they are a different interaction/content layer.

---

## 4. Page anatomy and information ownership

An ordinary authenticated leaf follows:

```text
global header / breadcrumb
secondary sidebar | page heading or operational header
                    first section
                    next section
```

### Page heading

- title: `text-lg`, semibold;
- subtitle: at most one short sentence;
- no nested application header;
- no title card.

### Project operational header

Project lifecycle actions may appear in the shared operational header only where they are a primary task:

- Overview;
- Deployments.

Logs, Environment, Database, General, Source, Resources, Webhook, and Danger zone MUST NOT repeat project name/status/Deploy actions.

### Route ownership

- Overview: operational summary and observability;
- Deployments: deployment history, deployment output, rollback;
- Logs: application log filtering/export;
- Environment: environment variables;
- Database: Database Studio;
- General: identity/public URL/deployment type;
- Source: repository/image/source configuration;
- Resources: allocation/profile limits;
- Webhook: endpoint/secret/setup;
- Danger zone: deletion.

Do not duplicate sibling-route information unless required to perform the current task.

---

## 5. Surfaces

Ordinary authenticated sections share one workspace surface.

Do not create hierarchy by alternating neutral backgrounds. Hierarchy comes from:

- alignment;
- typography;
- 1px strokes;
- semantic state dots/icons;
- real data color.

Reserved alternate surfaces:

- semantic warning/error/success/info states;
- overlays/dialogs;
- technical output;
- chart data marks;
- primary/destructive controls.

First-level rounded card stacks and decorative shadows are prohibited.

---

## 6. Settings and compact content

Settings/configuration rows use:

```text
[label] [value/control] [optional actions]
```

Desktop targets:

- label column: roughly **9–12rem**;
- row vertical padding: roughly **10–12px**;
- divider between logical rows;
- controls use the smallest width that remains usable.

Normal inputs/selects should usually stay between `max-w-sm` and `max-w-xl`. Do not stretch a simple input across a 1400px canvas.

### Compact Administration content

Short Administration workflows do not need to consume the full desktop canvas.

- General, Migration, and MCP may use a readable `max-w-4xl` / `max-w-5xl` content region while preserving the parent route gutter;
- this max width limits readable content, not the outer shell or sidebar geometry;
- Host stats may use a compact 3-column row;
- resource defaults should read as `preset | memory | CPU` rather than widely separated fields;
- single-field sections and update actions stay near their related value.

---

## 7. Typography and brand

Primary family: Inter Variable. Technical identifiers/output: IBM Plex Mono.

Targets:

- body/control/table text: **14–15px**;
- section titles: **15–16px**;
- supporting text: **13px**;
- technical mono: **12–13px**;
- true page/object titles: **18–24px** when needed.

Compactness comes from layout, not unreadable text.

Canonical vector assets:

- `src/assets/brand/mypaas-icon.svg` — application mark;
- `src/assets/brand/mypaas-logo.svg` — wordmark;
- `src/assets/brand/mypaas-favicon.svg` — explicit white browser favicon.

Do not reintroduce legacy raster logo variants.

---

## 8. Color semantics

Default chrome is monochrome. Color encodes meaning.

- CPU: `--chart-cpu`;
- memory/RAM: `--chart-memory`;
- storage: `--chart-storage`;
- network: `--chart-network`.

Success/warning/danger/info colors are reserved for real state. Resource telemetry should not be reduced to decorative gray when semantic tokens exist.

---

## 9. Controls

Canonical desktop control geometry:

- height: **36px**;
- text: **14px**;
- icon-only: **36×36px**;
- coarse-pointer target: **44px**.

Adjacent controls align on the same edges. Idle controls inherit the workspace fill; hover/focus raises boundary/text contrast instead of adding decorative fills.

Use shared `ActionButton`, `ActionLink`, `IconButton`, and `.field` before creating route-local controls.

---

## 10. Canonical table grammar

**The Projects inventory table is the visual source of truth for operational tables.**

Containers and Audit Logs MUST feel like the same table family rather than separate admin widgets.

Preserve the Projects table qualities:

- roughly **60px** (`h-[3.75rem]`) rows when primary + secondary metadata are present;
- compact 13–14px primary text;
- secondary metadata directly beneath the primary value;
- deliberate `table-fixed` column geometry;
- technical values use mono/tabular numerals when helpful;
- quiet header and row strokes;
- contextual action/chevron at the far right;
- no decorative card around the table.

Order:

```text
table context / heading
toolbar when needed
column header
rows
pagination only when needed
```

Pagination chrome MUST NOT appear when the complete dataset demonstrably fits on one page.

### Projects inventory semantics

Keep Projects as the baseline rather than redesigning it route-by-route.

- `Updated` remains the concise project-change timestamp.
- Do not pair it with a redundant `Uptime / release` column.
- Runtime-backed projects use a compact **Limits** column for configured memory/CPU allocation.
- `Usage` explicitly identifies CPU (`% CPU`) rather than displaying an ambiguous percentage.
- Static projects do not pretend to have runtime usage/allocation.

### Runtime icon semantics

Runtime icons are meaningful, not generic file/package placeholders:

- Static → **web/globe**;
- Dockerfile → **Docker**;
- Docker Compose → **Compose**;
- OCI image → **Docker/container**.

Icons remain small, monochrome, and aligned with Repository/Branch icon treatment.

### Containers State vs Health

Container lifecycle and health are separate dimensions:

- **State**: running, exited, paused, restarting, dead, etc.;
- **Health**: Healthy, Unhealthy, Starting, or No check.

Never render `Running` and `Healthy` as one mixed status cell. A container without a healthcheck is `No check`, not Healthy and not Unhealthy.

### Audit Logs

Audit rows follow the same density and primary/secondary hierarchy as Projects. Routine probe events may be hidden by default, but hidden-row counts must be explicit and pagination/result copy must not claim that hidden rows are visible.

Destructive/security/failure evidence must never be hidden merely to make the table cleaner.

---

## 11. Metrics and charts

Charts are data visualization, not decorative cards.

- use the workspace surface;
- data marks may use semantic resource color;
- grid lines remain faint;
- never fabricate history;
- preserve last-known telemetry during background refresh failures.

Overview time-series charts should have meaningful vertical space, roughly **96–120px** when compact.

Runtime CPU/memory usage shows:

```text
used / allocation    percentage
semantic progress bar
```

Do not repeat the same allocation immediately beneath the bar.

---

## 12. Deployment status and output

Deployment status copy must be short and outcome-oriented.

Preferred pattern:

```text
Deployment live
Your app is running.
```

Avoid architecture-heavy narration such as “selected runtime is active” when a simple outcome says the same thing.

### Deployment output

Deployment/build logs use the canonical technical output surface.

- opening a deployment output should land at the latest output;
- while the user remains near the bottom, new output may continue following naturally;
- if the user scrolls upward, polling MUST NOT force them back to the bottom;
- when newer output is below the viewport, expose a compact **Latest** / scroll-to-latest action;
- empty output copy is short (`Waiting for output.` / `No output captured.`).

---

## 13. Technical output surface

Host Shell is the palette reference for technical output, including:

- project logs;
- deployment/build logs;
- audit metadata JSON;
- MCP setup prompt;
- migration commands;
- repository inspection/config examples.

Technical output may use a dark `console-surface` because it represents a different content mode, not a hierarchy trick.

---

## 14. Toasts

Toasts are lightweight confirmations, not floating cards.

Canonical toast:

- neutral workspace surface;
- thin `--workspace-divider` border;
- small radius (`rounded-md`);
- compact `px-3 py-2` geometry;
- 13px text;
- small semantic icon is the primary color signal;
- compact close control near the message;
- no large pastel background or heavy shadow;
- keep copy to one line when practical.

---

## 15. UI copywriting

User-facing UI copy describes **what happened, what exists, or what the user can do**. It does not expose internal architecture terminology merely because the implementation uses it.

### Direct-copy rule

Prefer:

- `Review activity and changes.`
- `Your app is running.`
- `Waiting for output.`
- `MyPaaS may restart.`

Avoid:

- `Review control-plane changes.`
- `selected runtime is active`;
- long explanations that restate the visible status;
- internal component names used as user concepts.

`control-plane` is valid in engineering docs, comments, architecture, and code. It SHOULD NOT appear in ordinary user-facing dashboard copy. Replace it with the concrete context: MyPaaS, services, settings, activity, deployment, app, or server.

Status presentation should normally be:

1. short status/title;
2. at most one short supporting sentence when it adds information.

---

## 16. Migration and MCP

### Migration

Migration is a focused workflow:

1. package status;
2. Prepare or Download;
3. when ready, restore command + Copy.

Do not add a generic `What is included?` filler disclosure to the main screen. Operational caveats belong next to the action they affect.

### MCP

MCP must explain what the connected agent can actually do. Present a compact capability summary derived from the real tool surface, covering projects, deployments/lifecycle, observability, and environment variables. Keep token/client/setup controls compact and task-oriented.

---

## 17. Empty states and redundancy

An empty page becomes simpler, not noisier.

Do not repeat:

- `0 variables` plus `No variables`;
- `0 visible` plus `No logs`;
- one-page pagination;
- adjacent dates that communicate the same event;
- sidebar links as navigation-only content blocks.

An empty state is normally a short title plus one actionable sentence.

---

## 18. Database Studio and full-canvas tools

Database Studio is a normal project leaf. Do not add a nested black application header. Connection/table browsing use ordinary section grammar.

Schema Design and Host Shell are genuine full-canvas tools and may consume useful workspace width/height while preserving global shell, stroke, control, and technical-output contracts.

---

## 19. Theme, loading, responsive, accessibility

- theme state is correct before first paint;
- local operations use local loading state rather than blocking the whole application;
- polling/SSE refresh does not discard usable last-known data without reason;
- desktop is the primary density target;
- mobile exposes the same authorized destinations;
- tables may use an intentional compact mobile representation instead of shrinking type below the floor;
- keyboard access and visible focus are preserved;
- hover-only affordances require keyboard/focus equivalents;
- color is never the only signal for critical state;
- background polling should not create noisy live announcements.

---

## 20. Explicit anti-patterns

Do not introduce:

- first-level rounded card stacks;
- arbitrary route-local outer padding/top margins;
- nested application headers;
- repeated project lifecycle bars on non-operational leaves;
- strong nested border grids;
- alternating neutral section fills;
- controls stretched across the desktop without need;
- pagination for a one-page dataset;
- repeated zero-state copy;
- generic runtime icons that hide the deployment mode;
- mixed container lifecycle + health in one cell;
- large pastel toast cards;
- fake telemetry history;
- user-facing internal jargon where direct language works;
- explanatory copy that only restates the label directly above it.

---

## 21. Change rule

When changing authenticated UI:

1. read this file first;
2. identify the parent route family and its canonical geometry;
3. use **Project Overview for stroke grammar**;
4. use **Projects inventory for operational table grammar**;
5. identify one owner for each piece of data/action;
6. fix shared behavior centrally before adding route-local CSS;
7. audit loading, empty, error, disabled, destructive, focus, responsive, and polling states;
8. run frontend unit tests, Svelte/TypeScript checks, production build, and repository CI before merge.

The target is one coherent operational workspace: **one geometry, one stroke language, one table family, one control system, one technical-output palette, short direct copy, and no redundant state.**
