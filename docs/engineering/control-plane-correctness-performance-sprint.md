# Control-plane correctness and performance sprint

Status: coordination plan only  
Baseline: `4c6559a6f3471f837b7a0b0cffd41c62bdbe0ada`  
Scope: current single-host MyPaaS control plane  

This document is the execution contract for the next correctness/performance sprint. It exists so different models/agents can work on one bounded phase at a time without inventing new scope.

The coordination branch/PR that carries this document is **not** an implementation branch. Source fixes must use the narrow phase branches declared below and must target current `origin/main` unless this document explicitly says otherwise.

## How to invoke a phase

The operator may switch models between phases. The literal instruction:

```text
Kerjakan Phase <ID> saja.
```

means:

1. read root `AGENTS.md` and this document;
2. read any phase-specific repository instructions named below;
3. fetch current `origin/main` and verify the phase is still applicable;
4. execute **only** the requested phase;
5. do not automatically continue to another phase;
6. do not merge any PR;
7. report evidence using the evidence labels in this document.

If a phase is audit-only, do not edit source. If a phase is implementation, use its declared branch/domain and tests. If the required evidence can only come from the production VM or real browsers, generate exact copy-paste measurement steps and stop instead of fabricating a runtime verdict.

## Evidence labels

Every audit result must classify claims as one of:

- **SOURCE-CONFIRMED** — directly proven from current repository source/tests/configuration.
- **RUNTIME-HYPOTHESIS** — source makes the behavior plausible, but actual production/browser behavior is not yet measured.
- **RUNTIME-EVIDENCE-REQUIRED** — a decision is blocked until the operator supplies VM/browser/production evidence.
- **RUNTIME-CONFIRMED** — based on explicit operator-provided runtime output attached to the phase discussion.

Never upgrade a hypothesis to a confirmed claim because it is likely.

## Global constraints

These apply to every phase.

- Preserve MyPaaS as a **single-host self-hosted PaaS**.
- Preserve the Podman-first Docker-compatible runtime contract.
- Do not add Kubernetes, HA control-plane work, read replicas, sharding, Redis, or distributed infrastructure for this sprint.
- Do not add app-specific compatibility logic for Wago or any other named application.
- Do not convert `mypaas-statd` into a generic data/read service. It remains host/runtime telemetry.
- Browsers communicate with the MyPaaS API, never directly with statd or the container engine.
- Do not weaken engine-socket, Caddy Admin socket, persistent-volume, secret, or filesystem security boundaries.
- Do not create benchmark theater or repeat unrelated throughput matrices.
- Do not optimize request count merely because DevTools shows many Svelte chunks. Measure transferred bytes, blocking behavior, server latency, and actual waterfalls.
- Do not add a PostgreSQL index until a real query pattern and measurement justify it.
- Do not redesign unrelated UI while fixing a correctness/performance defect.
- Use tests proportional to the change and existing repository gates.
- One implementation branch = one domain + one outcome.

## Known starting evidence

The following is already established at sprint creation time. Agents must re-check current source before relying on it because `main` may advance.

### Static project inventory primary action

Current project inventory action derivation converts a healthy `live + succeeded + no attention` state to `Stop` without considering deployment mode. The shared operational state still uses `Deploy again` for a live static project. This makes the inventory primary action too generic while the backend's static Stop capability remains valid.

Expected UX contract for this sprint:

```text
static + live            -> Deploy again (primary)
container-backed + live  -> Stop (primary)
stopped                  -> Start
failed/deploying/etc.    -> state-derived action
```

Static Stop may remain available as a secondary/overflow lifecycle action if the current UI exposes it there; this sprint does not remove the backend capability to unpublish a static route.

### SQLite / Wago specimen

Compatibility specimen: `https://github.com/howlil/wago`

Current Wago source declares a SQLite application store and uses Node's built-in `node:sqlite` / `DatabaseSync`. The application database is `/app/data/wago.db`, and `/app/data` is intended to be persistent runtime data.

Current MyPaaS SQLite DB Studio discovery begins from recognized encrypted project environment variables such as SQLite URLs/paths or driver hints. If no recognized SQLite path is present in project env, runtime SQLite connection resolution does not proceed.

This is evidence of a generic discovery gap to investigate. It is **not** permission to add a `wago` special case.

### Containers first-load read path

Current Containers route initiates its load on mount and treats the first inventory request as initial main-content loading. The frontend calls `/api/admin/containers`.

Current backend container inventory does more than metadata discovery: it runs a container list and also attempts `stats --no-stream` for running containers, with per-container fallback for unmatched metrics. Therefore an expensive telemetry operation is on the same response path as inventory metadata.

Whether this is the dominant production latency source must be measured. The architectural smell is source-confirmed; production timing is not.

### Browser/navigation observations supplied by operator

The operator has observed:

- Firefox stops vertical scrolling when table content ends.
- Chromium leaves additional page height/scroll below the content.
- first navigation to some routes, especially Containers, feels slow;
- DevTools can show dozens of Svelte JS/CSS requests, many cached;
- Firefox showed aborted requests during navigation in captured evidence.

Do not infer the exact CSS owner or conclude full-document navigation solely from those screenshots. Those remain runtime/browser hypotheses until traced and reproduced.

### PostgreSQL

The schema already has basic indexes for projects, deployments, audit logs, webhook deliveries, etc. Some deployment reads combine `WHERE project_id = ...` with `ORDER BY started_at DESC`, and some add status predicates. Composite indexes may be candidates, but current deployment retention is bounded and no new index is authorized without evidence.

Replication/read replicas are explicitly out of scope for this sprint.

---

# Phase 0 — Baseline read-path and defect map

**Recommended model:** GPT-5.6 Luna, reasoning high  
**Mode:** READ ONLY  
**Branch:** none  
**PR:** none  

## Goal

Produce one current-source baseline proving where each reported issue is owned before implementation starts.

## Mandatory reads

- `AGENTS.md`
- `PRODUCT.md`
- `frontend/AGENTS.md`
- `frontend/DESIGN.md`
- current implementations/tests relevant to Projects inventory, root layout/TableShell, DB Studio SQLite, Containers, statd metrics, database/query layer

## Required work

Map:

1. static project inventory action call graph;
2. application shell/page/table vertical sizing and overflow ownership;
3. SQLite discovery from project config through runtime helper;
4. initial route loading/fetching for primary dashboard routes;
5. Containers handler/service/subprocess path;
6. statd responsibilities and current fallback path;
7. PostgreSQL query/index patterns and connection pool configuration.

For primary frontend routes, produce a compact matrix containing:

```text
route
initial requests
sequential/parallel
first-render blockers
background polling/SSE
source of truth
expensive subprocess/runtime reads
possible duplicate fetches
```

## Runtime evidence handling

The agent has no production SSH requirement. For anything that cannot be proven from source, provide exact commands/browser steps the operator can run later.

## Output

- SOURCE-CONFIRMED findings
- RUNTIME-HYPOTHESIS findings
- RUNTIME-EVIDENCE-REQUIRED list
- exact files/functions involved
- recommended implementation ordering
- contradictions with this plan, if current main changed

## Stop condition

Stop after the audit. **Do not edit source. Do not create a branch. Do not continue to Phase 1.**

Operator invocation:

```text
Kerjakan Phase 0 saja.
```

---

# Phase 1 — Correct static project inventory primary action

**Recommended model:** GPT-5.6 Luna, reasoning high  
**Mode:** implementation  
**Branch:** `ux/static-inventory-primary-action`  
**PR target:** current `main`  

## Goal

Make project inventory primary lifecycle actions deployment-mode aware without changing Project Detail semantics or removing valid static lifecycle capabilities.

## Required behavior

```text
healthy live static project            -> primary action: Deploy again
healthy live non-static project        -> primary action: Stop
stopped project                         -> Start
no deployment                           -> Deploy
failed/in-progress/unknown states       -> existing operational state behavior
```

## Constraints

- Do not change backend static Stop behavior.
- Do not make Project Detail lose `Deploy again`.
- Do not duplicate the whole operational-state state machine in a route component.
- Prefer a small shared inventory-action derivation change.
- Preserve desktop/mobile inventory consistency.
- Read `frontend/AGENTS.md` and `frontend/DESIGN.md` before editing.

## Regression coverage

At minimum prove:

- live static => Deploy again;
- live Dockerfile/container-backed => Stop;
- live Compose => Stop;
- live image deployment => Stop if image mode is represented as a container-backed mode in current types;
- stopped => Start;
- Project Detail remains unaffected by the inventory-specific derivation.

Do not invent deployment modes that do not exist in current types.

## Validation

Frontend-only gate:

```bash
cd frontend
pnpm test
pnpm check
pnpm build
```

## PR expectation

Suggested title:

```text
fix: keep static inventory action deployment-aware
```

Do not merge.

## Stop condition

Stop after opening/updating the Phase 1 PR and reporting test results. Do not begin browser-layout work.

Operator invocation:

```text
Kerjakan Phase 1 saja.
```

---

# Phase 2 — Cross-browser vertical layout/scroll contract

**Recommended model:** GPT-5.6 Luna, reasoning high for investigation; GPT-5.5 high if the owning fix is cross-layer or ambiguous  
**Mode:** audit first; implement only when owner is proven  
**Branch if implementation is justified:** `ux/cross-browser-scroll-contract`  

## Goal

Make the application shell terminate at the same logical content boundary in Firefox and Chromium without route-specific height hacks.

## Investigation scope

Trace shared ownership of:

- `html`, `body`, Svelte root height;
- `100vh`, `100dvh`, `min-height`, `h-screen`, `min-h-screen`;
- flex parents/children and `min-height: 0` behavior;
- global `overflow`, `overflow-y`, nested scroll containers;
- fixed navigation/topbar/main offsets;
- `.page-shell`;
- `TableShell`;
- footer/pagination wrappers;
- browser-specific scrollbar/layout differences.

Inspect multiple table routes before concluding the problem is route-local.

## Required evidence

SOURCE-CONFIRMED:
- exact shared CSS/layout chain that can produce the extra height, or explicit proof that source alone cannot determine it.

If real browser evidence is required, generate a minimal measurement checklist for both Firefox and Chromium, including which DOM nodes to inspect and values to capture (`clientHeight`, `scrollHeight`, computed height/min-height/overflow, bounding rectangles).

The agent must not claim the bug is fixed solely because unit tests/build pass.

## Implementation rule

Only implement when the owning rule is sufficiently proven. Prefer one shared shell/primitive correction over repeated route patches.

## Validation

```bash
cd frontend
pnpm test
pnpm check
pnpm build
```

Plus operator-provided Firefox + Chromium verification if the environment used by the agent cannot run both browsers reliably.

## PR expectation

Suggested title:

```text
fix: normalize control-plane vertical scroll geometry
```

Do not merge.

## Stop condition

If source evidence is insufficient, stop with **RUNTIME-EVIDENCE-REQUIRED** and exact browser measurement steps. Do not guess a CSS fix.

Operator invocation:

```text
Kerjakan Phase 2 saja.
```

---

# Phase 3 — Generic SQLite discovery for persistent runtime databases

**Recommended model:** GPT-5.5, reasoning high  
**Mode:** design + implementation only after re-verifying specimen/current code  
**Branch:** `core/sqlite-persistent-discovery`  

## Goal

Allow DB Studio to discover real persistent SQLite applications such as the Wago pattern when the SQLite database path is not declared through MyPaaS-recognized project environment variables, without adding app-specific logic or arbitrary filesystem scanning.

## Compatibility specimen

`https://github.com/howlil/wago`

Re-verify current Wago source before implementing. Do not assume the specimen is unchanged.

Expected current pattern at sprint creation:

```text
Node node:sqlite / DatabaseSync
SQLite DB: /app/data/wago.db
persistent data root: /app/data
```

## Required design questions

The implementation must explicitly resolve:

1. What is the precedence between explicit env-derived SQLite paths and inferred runtime discovery?
2. Which persistent mount types/directories are eligible?
3. How narrowly are candidate database files discovered?
4. How is a SQLite file positively identified rather than guessed from an arbitrary filename?
5. What candidate count/depth/size/time bounds prevent broad container scans?
6. What happens with zero candidates?
7. What happens with multiple valid SQLite candidates?
8. How is ambiguity presented to DB Studio without silently picking the wrong DB?
9. How does this work for Dockerfile/image/Compose runtime candidates while preserving current container ownership boundaries?
10. Can discovery reuse the existing isolated SQLite helper instead of executing application code?

## Security invariants

- Never scan arbitrary host filesystem.
- Never scan outside eligible project runtime persistent mounts.
- Keep the existing requirement that DB Studio SQLite targets persistent storage.
- Do not follow a path outside the inspected persistent mount boundary.
- Do not execute untrusted project code to discover the DB.
- Do not add `if project == wago` or repository-name heuristics.
- Keep helper isolation (`network none`, read-only helper container, dropped capabilities, no-new-privileges or current stronger equivalent).
- Never log DB contents/secrets.

## Preferred behavior

Explicit configuration remains authoritative. Runtime inference is a conservative fallback, not a replacement for explicit env detection.

A generic solution may inspect eligible persistent mounts and identify bounded SQLite candidates using file/database evidence. The exact design must come from current implementation and tests; do not implement an unbounded recursive scan just because it would find Wago.

## Required regression tests

Include at least:

- existing env-derived SQLite cases remain unchanged;
- Wago-shaped persistent mount with `/app/data/wago.db` can be discovered generically;
- database in writable container layer is rejected;
- database in tmpfs is rejected;
- non-SQLite files are not claimed;
- multiple SQLite candidates produce deterministic safe behavior (explicit selection/ambiguity, according to the approved design);
- traversal/symlink/boundary cases cannot escape eligible mount ownership;
- server DB connection precedence remains correct.

## Validation

Backend tests proportional to affected packages, then standard backend gate if the change crosses shared DB Studio/runtime code:

```bash
cd backend
go test ./...
go test -race ./...
go build ./cmd/api
```

If scripts/Compose contracts are touched, run their existing repository gates too.

## Runtime evidence

If actual Wago runtime mount/container details are needed, produce copy-paste `docker inspect`/Podman-compatible commands for the operator. Do not require SSH access from the agent.

## PR expectation

Suggested title:

```text
fix: discover persistent SQLite databases conservatively
```

Do not merge.

## Stop condition

Stop if a generic secure contract cannot be proven. Never fall back to an app-specific workaround.

Operator invocation:

```text
Kerjakan Phase 3 saja.
```

---

# Phase 4 — Dashboard fetch/navigation/read-path audit

**Recommended model:** GPT-5.6 Luna, reasoning high  
**Mode:** READ ONLY  
**Branch:** none  

## Goal

Map actual initial-navigation and background-read behavior for the whole control-plane UI before performance changes are made.

## Routes to cover

At minimum inspect every primary navigation destination available in current main, plus Project Detail and its high-use child routes. Use current navigation source rather than a hard-coded old route list.

## For each route record

- data requests fired on first entry;
- which request owns initial main-content loading;
- serial vs parallel requests;
- duplicate project/user/auth reads;
- `/me` behavior and where it is owned;
- background polling/SSE subscriptions;
- cleanup on navigation/unmount;
- expensive Docker/Podman/host subprocesses caused by a read;
- source of truth: PostgreSQL / Docker-compatible engine / statd / filesystem / Caddy;
- whether data can be stale and for how long;
- whether first useful render truly needs the data;
- whether SvelteKit preloading/prefetch behavior is being used or defeated;
- any code path that can fall back to full-document navigation before hydration.

## Asset analysis

Distinguish:

- number of requests;
- transferred bytes;
- cached immutable chunks;
- blocking chunks;
- oversized static assets;
- actual navigation/server waterfalls.

Do not recommend bundle surgery simply because request count is high.

## Containers special case

Trace and classify separately:

```text
inventory metadata
runtime status metadata
CPU/RAM/PID telemetry
```

Evaluate the contract:

```text
fast container metadata -> Docker/Podman engine
telemetry                -> statd / existing telemetry abstraction
browser                  -> MyPaaS API only
```

Do not assume statd should own container names/images/Compose labels or PostgreSQL product state.

## Runtime measurement pack

Because the agent may not have SSH, output exact operator commands/steps needed to measure hypotheses, for example:

- authenticated API timing for expensive routes;
- host-side timing of `docker ps -a` versus `docker stats --no-stream`;
- counts of running containers;
- browser Network navigation type and timing;
- repeated first visit versus warm navigation;
- Firefox versus Chromium evidence.

Do not include credentials/cookies in committed docs or logs. Use placeholders where authentication is required.

## Output

Rank findings:

```text
P0 confirmed user-visible defect
P1 measurable avoidable latency
P2 harmless/low-value cleanup
NO ACTION
```

For every optimization candidate give expected benefit and evidence needed.

## Stop condition

Audit only. **Do not edit source, add caching, add prefetch, or split endpoints in this phase.**

Operator invocation:

```text
Kerjakan Phase 4 saja.
```

---

# Phase 5 — Containers fast metadata + non-blocking telemetry

**Recommended model:** GPT-5.5, reasoning high  
**Mode:** conditional implementation  
**Branch:** `core/container-inventory-fast-path`  

## Entry criteria

Do not execute this phase merely because it exists. Phase 4 and/or operator runtime evidence must confirm that expensive telemetry is materially delaying Containers first useful render or that the current combined contract creates a reproducible blocking defect.

If entry criteria are not met, return `DEFER / NO ACTION`.

## Goal

Make Containers inventory metadata available without waiting for expensive runtime telemetry, while preserving truthful CPU/RAM state and existing runtime compatibility.

## Architecture boundary

Preferred separation if supported by Phase 4 evidence:

```text
Container inventory metadata
    -> Docker-compatible engine
    -> names, IDs, image, state/status, Compose project/service

Container telemetry
    -> statd where supported / existing safe fallback where required
    -> CPU, memory, PID/resource metrics

Browser
    -> MyPaaS API only
```

Do not move product state or generic runtime metadata into statd.

## UX/loading contract

- initial route shell/table metadata should not be blanked by background telemetry;
- telemetry can populate asynchronously;
- polling/SSE/background telemetry must not trigger global blocking loader;
- missing telemetry must be represented truthfully, never fabricated as zero;
- navigation away must not leave orphaned timers/subscriptions.

## Backend constraints

- preserve context cancellation;
- bound subprocess/statd timeouts;
- avoid N+1 per-container subprocesses in the steady state where possible;
- preserve Podman Docker-compatible behavior;
- do not expose engine socket/runtime service directly to browser.

## Tests

Add behavioral coverage for:

- inventory metadata succeeds even when telemetry fails/is slow;
- metricsAvailable/unknown state stays truthful;
- telemetry path does not change metadata ownership;
- cancellation/error behavior;
- frontend initial loading ends when required inventory is available, not when background telemetry finishes;
- background refresh does not trigger global loader.

## Validation

Run affected backend + frontend gates. If runtime orchestration contract changes, include existing Podman compatibility gate.

Do not run unrelated benchmark matrices.

## PR expectation

Suggested title:

```text
perf: decouple container inventory from telemetry
```

Do not merge.

## Stop condition

Stop after PR and evidence. Production timing must be rechecked by operator after deployment; repo tests alone cannot prove production latency improvement.

Operator invocation:

```text
Kerjakan Phase 5 saja.
```

---

# Phase 6 — PostgreSQL query/index/pool audit

**Recommended model:** GPT-5.6 Luna, reasoning medium/high for audit; GPT-5.5 only if an evidence-backed schema change is approved  
**Mode:** READ ONLY by default  
**Branch:** none unless a migration is justified  

## Goal

Determine whether PostgreSQL contributes materially to dashboard latency and whether any index/pool changes are justified.

## Audit scope

Inspect:

- current migrations/schema;
- all query files and generated call patterns;
- Projects reads;
- deployment history/latest/active/rollback queries;
- audit logs;
- webhook deliveries;
- env vars;
- ports;
- users/auth;
- pagination/counts;
- database connection/pool configuration.

Produce a query/index matrix:

```text
query/use case
WHERE predicates
ORDER BY
LIMIT/OFFSET
existing usable index
candidate issue
expected table growth
measurement required
```

## Measurement rule

No new index based only on code aesthetics.

For candidates, generate representative:

```sql
EXPLAIN (ANALYZE, BUFFERS)
...
```

commands for the operator to run against representative data. Avoid leaking sensitive values in shared evidence; use safe IDs/placeholders where possible.

Consider actual retention/bounded table size. An index that is theoretically better but meaningless for current/expected table size should be `NO ACTION`.

## Explicitly out of scope

- replication/read replicas;
- sharding;
- partitioning;
- Redis caching;
- HA PostgreSQL;
- speculative DB parameter tuning.

## Conditional implementation

Only if operator-provided EXPLAIN/latency evidence proves a meaningful deficiency, GPT-5.5 may create a narrow branch such as:

```text
core/postgres-query-index
```

with the smallest migration/query/pool change justified by evidence.

Any migration requires forward/down behavior consistent with repository conventions and proportional backend/script verification.

## Stop condition

By default this phase ends with an audit and measurement pack. Do not create migrations unless evidence is supplied and the operator explicitly continues the phase with that evidence.

Operator invocation:

```text
Kerjakan Phase 6 saja.
```

---

# Phase 7 — Runtime evidence reconciliation

**Recommended model:** GPT-5.5, reasoning high  
**Mode:** analysis of operator-provided evidence; no SSH assumption  
**Branch:** none  

## Goal

Reconcile source audits with real VM/browser evidence before final merge decisions.

## Inputs

Use only evidence actually supplied by the operator, such as:

- authenticated route timings;
- `docker ps` and `docker stats` timing;
- container counts;
- `docker inspect` for Wago runtime/mounts;
- Firefox/Chromium DOM height/scroll measurements;
- PostgreSQL `EXPLAIN (ANALYZE, BUFFERS)` output;
- network navigation timings;
- final CI results from phase PRs.

## Rules

- Do not claim SSH was performed if it was not.
- Do not infer missing production measurements.
- Distinguish production defect from local browser/cache behavior.
- Reject optimizations whose predicted bottleneck is contradicted by runtime evidence.
- If an implementation PR was based on a false hypothesis, recommend closing/reworking it rather than defending sunk work.

## Output

For each phase:

```text
CONFIRMED / REJECTED / INCONCLUSIVE
source evidence
runtime evidence
remaining blocker
merge recommendation
production re-test required?
```

## Stop condition

No source edits. No merge.

Operator invocation:

```text
Kerjakan Phase 7 saja.
```

---

# Phase 8 — Release-blocking sprint review

**Recommended model:** GPT-5.5, reasoning high  
**Mode:** final reviewer  
**Branch:** none  

## Goal

Review every implementation PR produced by this sprint before merge.

## Review checklist

For each PR verify:

- solves the confirmed root cause, not only a symptom;
- stays inside its declared phase scope;
- no Wago/app-specific workaround;
- no architecture expansion beyond single-host product boundary;
- no statd responsibility creep;
- no browser-to-runtime bypass;
- no engine/socket/filesystem/security regression;
- no secret leakage;
- no frontend `DESIGN.md` violation;
- no new global loading caused by background data;
- no stale polling/SSE cleanup bug;
- no speculative index/caching/dependency;
- tests assert behavior, not brittle implementation strings where avoidable;
- CI/gates are proportional and green;
- VM/browser qualification is requested only where the changed contract requires it.

## Required output

```text
BLOCKERS
MAJOR
MINOR
EVIDENCE GAPS
MERGE ORDER
MERGE VERDICT PER PR
```

Do not merge. The operator decides when to authorize merges after reviewing this phase.

Operator invocation:

```text
Kerjakan Phase 8 saja.
```

---

# Recommended execution order

Default order:

```text
Phase 0  baseline source audit
Phase 1  static inventory correctness
Phase 2  browser scroll contract
Phase 3  SQLite generic discovery
Phase 4  read-path/fetch audit
Phase 6  PostgreSQL audit (can run in parallel with Phase 4)
        -> operator supplies required VM/browser/EXPLAIN evidence
Phase 5  only if Phase 4/runtime evidence justifies container fast-path implementation
Phase 7  reconcile runtime evidence
Phase 8  release-blocking review
```

Phase 1 is already strongly source-confirmed and can proceed after Phase 0 verification. Phase 5 and any Phase 6 migration are explicitly conditional.

## Model-switching guidance

Use model capability where it adds value rather than using the strongest model for mechanical exploration.

```text
GPT-5.6 Luna
- Phase 0 repository exploration/read-path map
- Phase 1 narrow frontend implementation
- Phase 2 source/layout investigation
- Phase 4 route/fetch audit
- Phase 6 query/index inventory

GPT-5.5
- Phase 3 security-sensitive generic SQLite discovery design/implementation
- Phase 5 cross-layer inventory/telemetry contract
- Phase 7 source + runtime evidence synthesis
- Phase 8 final release-blocking review
- Phase 2 or Phase 6 implementation when evidence requires a more complex cross-layer decision
```

A model switch does not change phase scope. The new model must re-read this document and current source rather than trusting previous agent reasoning.

# Definition of sprint done

The sprint is complete when:

1. static project inventory primary action matches deployment semantics;
2. Firefox/Chromium page-height behavior is either fixed and browser-verified or explicitly documented as inconclusive with evidence gap;
3. Wago-shaped persistent SQLite is handled by a generic secure discovery contract or the proposed generic solution is explicitly rejected with reason;
4. route/fetch audit identifies actual blocking/duplicate/background reads and avoids request-count superstition;
5. Containers first useful render is improved only if evidence demonstrates the current telemetry coupling is a material problem;
6. PostgreSQL receives only evidence-backed changes; replication remains out of scope;
7. affected CI gates pass;
8. runtime/browser evidence is reconciled for changes that cannot be qualified from repository tests alone;
9. no implementation PR is merged solely because it was produced during the sprint.

Do not continue inventing optimization work after these acceptance criteria are satisfied.
