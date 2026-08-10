# New Project Visual Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify the New Project page into one visually coherent creation surface with exception-driven diagnostics, concise on-demand help, and a compact review summary while preserving all deployment behavior.

**Architecture:** Keep existing state, API calls, validation helpers, and submit payload semantics in `+page.svelte`; refactor only presentation/disclosure hierarchy. Introduce one lightweight reusable `InfoDisclosure.svelte` only for concise optional explanations. Add a small pure presentation helper in project validation only if needed for testable setup-summary behavior; otherwise keep readiness logic unchanged.

**Tech Stack:** SvelteKit, TypeScript, Tailwind CSS, Lucide Svelte, Vitest, existing MyPaas UI primitives (`surface`, `field`, `ActionButton`, `IconButton`).

## Global Constraints

- Do not change backend contracts or deployment semantics.
- Preserve source validation and automatic runtime detection.
- Preserve manual runtime and container-port override capability.
- Preserve Compose preflight, env discovery/import, shared PostgreSQL, resource profiles, readiness validation, and coding-agent handoff.
- Use one primary creation `surface`; section hierarchy should rely on typography, whitespace, and dividers rather than nested cards.
- Keep required configuration visible when it blocks creation; never hide a required registry container port inside Advanced.
- Repository tree and detailed healthy Compose diagnostics are hidden by default.
- Use existing design tokens and components before introducing new visual primitives.
- Optional explanations use concise accessible disclosure, not permanently visible helper paragraphs.

---

### Task 1: Lock behavior with page-level presentation tests

**Files:**
- Modify: `frontend/src/lib/validation/project.test.ts`
- Modify only if helper is justified: `frontend/src/lib/validation/project.ts`

**Interfaces:**
- Consumes: existing `projectCreationReadiness(...)`, `suggestProjectName(...)`, `resolveProjectAppPort(...)`.
- Produces: tests proving registry-port blocking and Git auto-analysis readiness remain unchanged during the visual refactor.

- [ ] **Step 1: Add a failing regression test for registry readiness copy/actionability**

Add a test asserting registry source with no `appPort` returns `Needs configuration` and a reason that explicitly tells the user to set a container port, while a valid registry port becomes ready.

- [ ] **Step 2: Run the focused validation test file and verify RED only if production behavior is insufficient**

Run: `cd frontend && pnpm test --run src/lib/validation/project.test.ts`

Expected: Either the new assertion fails for the intended missing behavior, or if current behavior already covers it, do not change production helper logic and treat it as a regression characterization test.

- [ ] **Step 3: Make the minimal helper change only if Step 2 exposed a behavior gap**

Keep readiness semantics unchanged apart from making the blocking reason actionable enough for the simplified UI.

- [ ] **Step 4: Run focused tests until green**

Run: `cd frontend && pnpm test --run src/lib/validation/project.test.ts`
Expected: 0 failures.

- [ ] **Step 5: Commit**

Commit message: `test(project): lock simplified creation readiness behavior`

---

### Task 2: Add concise accessible info disclosure

**Files:**
- Create: `frontend/src/lib/components/InfoDisclosure.svelte`
- Create: `frontend/src/lib/components/InfoDisclosure.test.ts` if component-test setup supports Svelte component rendering; otherwise test through `svelte-check` and page behavior without adding a new test framework.

**Interfaces:**
- Produces: `<InfoDisclosure label="...">help text</InfoDisclosure>` with keyboard-operable button semantics, `aria-expanded`, `aria-controls`, Escape-close behavior, and concise inline disclosure content.

- [ ] **Step 1: Write the failing component behavior test when existing frontend test tooling supports Svelte rendering**

Test trigger label, initial collapsed state, click expansion, `aria-expanded`, and Escape collapse. If the repository has no component-rendering utility, document that constraint in the commit and rely on Svelte static accessibility checks instead of adding a dependency.

- [ ] **Step 2: Run the focused test and verify RED**

Run the smallest supported Vitest command for the new component test.

- [ ] **Step 3: Implement minimal `InfoDisclosure.svelte`**

Use Lucide `Info`, a native button, deterministic local panel id, concise inline panel, and no external popover dependency. Reuse existing focus/border/color tokens.

- [ ] **Step 4: Run focused tests / `pnpm check` and verify green**

Expected: test passes when present; `svelte-check` reports no accessibility/type errors.

- [ ] **Step 5: Commit**

Commit message: `feat(ui): add accessible info disclosure`

---

### Task 3: Collapse New Project into one main surface

**Files:**
- Modify: `frontend/src/routes/projects/new/+page.svelte`

**Interfaces:**
- Consumes: all current state variables and handlers unchanged.
- Produces default visual flow: Source → Detected setup → Environment → Advanced, with one compact sticky Review/Create summary.

- [ ] **Step 1: Add a failing structural regression test if an existing page-source test pattern exists**

Assert that the normal flow no longer renders separate section-panel headings for Resources and Runtime as independent cards, and that a single `Advanced` disclosure exists. If there is no page-source test pattern, do not add brittle DOM-text snapshot tests; proceed with compile-time verification and preserve Task 1 behavior tests.

- [ ] **Step 2: Replace stacked `SectionPanel` usage with one main `surface`**

Keep breadcrumbs/PageHeader. Inside one form surface, create sections separated by `border-t` dividers and heading rows. Remove `SectionPanel` wrappers for Source, Runtime, Resources, and Environment.

- [ ] **Step 3: Simplify Source section**

Keep project name, source segmented control, repository/image input, branch. Replace permanent route/registry explanation paragraphs with terse secondary text or `InfoDisclosure`. Move project directory and repository tree to Advanced.

- [ ] **Step 4: Replace runtime cards with compact Detected setup row**

For ready Git detection, render one line such as `Docker Compose · api · :3000` plus status icon and `Re-analyze`. For loading/error/unresolved states, render one concise actionable line. Required registry port must appear inline in this section when missing.

- [ ] **Step 5: Make Compose diagnostics exception-driven**

Healthy Compose plan: show only `Compose ready`. Blocking/warning issues: show concise actionable normal-flow warning. Move services, ports, env summaries, detailed issues, candidate scans, and override controls into Advanced.

- [ ] **Step 6: Simplify Environment section**

Keep missing-required-env banner/rows, editable env table, `.env` import, localhost warnings. Replace standalone database card with a compact checkbox row and optional `InfoDisclosure`. Empty state becomes quiet text.

- [ ] **Step 7: Consolidate one Advanced disclosure**

Include headings for Source, Runtime, Compose, Resources, Diagnostics. Move base directory, repository tree, deployment-mode override, port override (except unresolved required registry port), Compose overrides, static frontend override, resource tuning, and detailed Compose Doctor here.

- [ ] **Step 8: Shrink sticky Review summary**

Keep only public hostname, source + branch/image, detected runtime, readiness state, and Create button. Remove duplicated CPU, memory, profile, DB, Compose file/profile, and port-explanation rows.

- [ ] **Step 9: Shorten page and helper copy**

Use task-oriented header description. Remove infrastructure tutorial paragraphs from normal flow. Keep blocking/error copy explicit.

- [ ] **Step 10: Run frontend checks**

Run:
- `cd frontend && pnpm test --run`
- `cd frontend && pnpm check`
- `cd frontend && pnpm build`

Expected: all tests pass, `svelte-check` 0 errors/0 warnings, production build exits 0.

- [ ] **Step 11: Commit**

Commit message: `refactor(project): simplify new project visual hierarchy`

---

### Task 4: Full repository verification and PR

**Files:**
- No additional production files unless verification exposes a defect.

**Interfaces:**
- Produces: a green branch ready for PR and squash merge to `main`.

- [ ] **Step 1: Re-read the design spec against the diff**

Verify every success criterion: one main surface; healthy setup compressed; diagnostics hidden by default; explanation on demand; required config visible; compact review; existing design system reused.

- [ ] **Step 2: Run full frontend verification fresh**

Run:
- `cd frontend && pnpm test --run`
- `cd frontend && pnpm check`
- `cd frontend && pnpm build`

- [ ] **Step 3: Run backend and deployment-script checks through repository CI**

Open PR to `main`; use the repository CI as authoritative cross-stack verification.

- [ ] **Step 4: Inspect CI and review threads**

Do not merge while any required workflow fails or any blocking review thread remains unresolved.

- [ ] **Step 5: Squash merge after green verification**

Use expected head SHA guard. Confirm resulting PR state is `merged: true` and record merge SHA.
