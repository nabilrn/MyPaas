# New Project UX Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify MyPaas project creation into a source-first, auto-analysis flow while keeping technical overrides available under Advanced sections.

**Architecture:** Keep backend contracts unchanged. Extract pure frontend UX decisions into `newProjectUx.ts` so readiness and source-derived defaults are testable, then wire the existing Svelte page to those helpers and progressively disclose advanced runtime/Compose/resource controls.

**Tech Stack:** SvelteKit, TypeScript, Vitest, Tailwind CSS.

## Global Constraints

- No backend API/schema changes.
- Keep single-page project creation.
- Blank base directory means repository root; never suggest `/`.
- Git projects cannot show `Ready to create` while deploy mode is still `auto`.
- Non-static runtimes require a resolved container port.
- Existing Compose Doctor blocking rules and env validation remain enforced.

---

### Task 1: Testable UX state helpers

**Files:**
- Create: `frontend/src/lib/utils/newProjectUx.ts`
- Create: `frontend/src/lib/utils/newProjectUx.test.ts`

**Interfaces:**
- Produces: `suggestProjectName(source: string): string`
- Produces: `runtimeAnalysisComplete(sourceType: 'git' | 'registry', deployMode: string): boolean`
- Produces: `projectCreationReadiness(input): { ready: boolean; state: string; reason: string }`

- [ ] **Step 1: Write failing tests** for repo/image slug suggestions and for Git `auto` mode being non-ready.
- [ ] **Step 2: Run `pnpm test --run src/lib/utils/newProjectUx.test.ts` and verify RED.**
- [ ] **Step 3: Implement the minimal pure helpers.**
- [ ] **Step 4: Re-run the focused tests and verify GREEN.**
- [ ] **Step 5: Commit `test/refactor` helper changes.**

### Task 2: Source-first automatic analysis

**Files:**
- Modify: `frontend/src/routes/projects/new/+page.svelte`
- Test: `frontend/src/lib/utils/newProjectUx.test.ts`

**Interfaces:**
- Consumes the Task 1 helpers.

- [ ] **Step 1: Add failing readiness tests** showing repository validation alone is insufficient and a resolved detected runtime is required.
- [ ] **Step 2: Verify RED in CI.**
- [ ] **Step 3: Wire automatic runtime detection after a current Git repository inspection, guard against stale/in-flight requests, and rename manual detection to `Re-analyze`.**
- [ ] **Step 4: Suggest project name from repo/image only until the user edits the name manually.**
- [ ] **Step 5: Replace page-local `canSubmit`/review labels with readiness helper output.**
- [ ] **Step 6: Verify focused and full frontend tests.**

### Task 3: Progressive disclosure of advanced controls

**Files:**
- Modify: `frontend/src/routes/projects/new/+page.svelte`

- [ ] **Step 1: Move base directory behind `Advanced source settings`; use an empty-root placeholder/copy.**
- [ ] **Step 2: Present detected runtime as the normal result and move deploy-mode override into `Advanced runtime settings` with the existing port override.**
- [ ] **Step 3: Collapse Compose file/workdir/override/profile controls into `Advanced Compose settings`.**
- [ ] **Step 4: Reduce healthy Compose Doctor to a concise ready summary while keeping blocking issues expanded/actionable.**
- [ ] **Step 5: Move resource profile/memory/CPU into a collapsed `Resources` details section showing the current recommendation in its summary.**
- [ ] **Step 6: Hide coding-agent handoff during successful normal analysis; show it only when analysis produced a deployability error/prompt.**

### Task 4: Environment hierarchy and final verification

**Files:**
- Modify: `frontend/src/routes/projects/new/+page.svelte`
- Modify if needed: `frontend/src/lib/utils/newProjectUx.test.ts`

- [ ] **Step 1: Surface missing required env variables ahead of optional values and keep existing env editor/import behavior.**
- [ ] **Step 2: Present shared PostgreSQL as an explicit optional database control with clearer copy.**
- [ ] **Step 3: Ensure Review focuses on route/source/runtime/readiness and treats port/resources as secondary metadata.**
- [ ] **Step 4: Run `pnpm test --run`, `pnpm check`, and `pnpm build` in GitHub CI.**
- [ ] **Step 5: Confirm backend/deployment-script CI remains green because no backend contract changed.**
- [ ] **Step 6: Open upstream PR to `nabilrn/MyPaas:main` with screenshots/copy summary omitted unless artifacts are available; document behavior and tests instead.**