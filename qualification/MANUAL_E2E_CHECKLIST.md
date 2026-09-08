# MyPaaS Manual E2E Checklist

Manual stable-release feature qualification for the **actual product surface on `main`**.

This checklist is intentionally human-driven. Tick a box only after the expected result is visible. If a step fails, leave it unchecked and write what happened in the **Failure notes** area for that section.

> Baseline used to build this checklist: `438fa43565aa19492fa314880350be40d2c262af`
>
> Remote MCP (`/mcp`) from draft PR #317 is **not part of this baseline**. The MCP checks here cover the local STDIO bridge that exists on `main`.

---

## How to use this file

- [ ] I am testing a disposable/non-critical MyPaaS instance or I understand which steps are destructive.
- [ ] I recorded the exact build/revision being tested below.
- [ ] I will tick a checkbox only after observing the final state, not merely after clicking a button.
- [ ] If something fails, I will leave it unchecked and add a short note, screenshot, response/error text, or reproduction detail.
- [ ] I will not fix bugs while running this checklist; failures are captured first and fixed in a separate pass.

### Test run

| Field | Value |
| --- | --- |
| Date | |
| Tester | |
| MyPaaS version/tag | |
| Git SHA / installed revision | |
| Base URL | |
| Runtime | Podman / Docker |
| Browser | |
| Host / VM notes | |

### Test inputs

Fill these once so the rest of the checklist stays short.

| Fixture | Value |
| --- | --- |
| Git repo with Dockerfile | |
| Git repo with Compose | |
| Git repo with static site / static SPA | |
| Known-good OCI image + internal port | |
| Monorepo/base-directory repo, if available | |
| DB-backed project, if available | |
| Secondary account, if available | |
| Cloudflare analytics credentials, optional | |
| S3/R2 backup credentials, optional | |

---

# A. Platform and authentication

## A1. Basic availability

- [ ] Dashboard/login page opens without a browser error.
- [ ] `/api/health` returns a healthy response.
- [ ] `/api/ready` reports ready while PostgreSQL is available.
- [ ] Refreshing an authenticated page does not unexpectedly log me out.

**Failure notes**

- 

## A2. GitHub authentication

- [ ] GitHub login redirects to GitHub and returns to MyPaaS successfully.
- [ ] After login, the correct account/avatar/identity is shown.
- [ ] GitHub repository picker loads repositories accessible to the logged-in account.
- [ ] Logout returns me to the public/login state.
- [ ] Logging in again works after logout.

**Failure notes**

- 

---

# B. Main navigation and access

## B1. Owner navigation

As the owner:

- [ ] Projects opens.
- [ ] Containers opens.
- [ ] Shell opens.
- [ ] Administration opens.
- [ ] Project detail navigation exposes Overview, Deployments, Logs, Environment, Database, Settings, Webhook, and Danger zone.
- [ ] Administration exposes General, Users, System update, Backup, Migration, MCP, and Audit logs.

**Failure notes**

- 

## B2. Secondary user access — optional but recommended

If a non-owner/secondary user is available:

- [ ] Secondary user can log in successfully.
- [ ] Owner-only Shell is not exposed as an ordinary usable action to the secondary user.
- [ ] Owner-only Administration is not exposed as an ordinary usable action to the secondary user.
- [ ] Containers behaves consistently with what the UI/navigation offers to the secondary user.
- [ ] Direct navigation to owner-only URLs is rejected safely rather than exposing privileged data.

**Failure notes**

- 

---

# C. Repository inspection and Create Project

## C1. Git repository inspection

Use the **New project** flow with a Git source.

- [ ] Selecting/pasting a valid repository starts repository inspection.
- [ ] Default branch is populated correctly.
- [ ] Branch choices are populated and switching branch re-runs source validation.
- [ ] Repository tree/base-directory choices are usable.
- [ ] Auto detection reports a clear deploy mode rather than leaving the form stuck in analysis.
- [ ] Environment-variable discovery appears when the repository declares variables.
- [ ] Invalid/unreachable repository gives an actionable error and does not create a project.

**Failure notes**

- 

## C2. Monorepo / base directory — if available

- [ ] Selecting a base directory changes inspection to that subdirectory.
- [ ] Changing base directory invalidates old detection and re-checks the repository.
- [ ] The detected deploy mode belongs to the selected subdirectory, not the repository root.
- [ ] Creating the project preserves the selected base directory.
- [ ] Reloading Project Settings shows the same saved base directory.

**Failure notes**

- 

## C3. Compose inspection — if Compose fixture is available

- [ ] Compose file is detected.
- [ ] Available Compose file candidates can be selected when more than one exists.
- [ ] Services are detected.
- [ ] Main service is selected or can be selected.
- [ ] Public app port is detected or can be supplied manually.
- [ ] Required environment variables are shown.
- [ ] Missing required Compose environment values block project creation with a clear reason.
- [ ] Blocking Compose issues prevent creation instead of allowing a predictably broken project.

**Failure notes**

- 

---

# D. Dockerfile project

Use a disposable Git repository containing a production Dockerfile.

## D1. Create

- [ ] Auto detection selects Dockerfile for a repo whose relevant root contains a Dockerfile and no higher-priority Compose configuration.
- [ ] Project name validation works.
- [ ] App port is detected correctly or accepts a valid manual value.
- [ ] Resource profile can be selected.
- [ ] Project is created successfully.
- [ ] Project appears in the Projects list.
- [ ] Project detail loads without stale/unsaved state warnings.

## D2. First deployment

- [ ] Clicking Deploy creates a deployment record.
- [ ] Deployment page shows build/deploy progress.
- [ ] Successful deployment ends in a running/success state.
- [ ] Public project URL opens the deployed application.
- [ ] Project Overview and latest-deployment state agree with the deployment page.

**Failure notes**

- 

---

# E. Docker Compose project

Use a disposable multi-service Compose repository.

## E1. Create and deploy

- [ ] Auto detection selects Compose.
- [ ] Correct Compose file is selected.
- [ ] Correct main service is selected.
- [ ] Correct public app port is selected.
- [ ] Required environment values can be filled before creation.
- [ ] Project creates successfully.
- [ ] Deployment completes successfully.
- [ ] Main public URL reaches the intended main service.
- [ ] Supporting services start as expected.

## E2. Compose runtime visibility

- [ ] Overview runtime telemetry shows service-level data when available.
- [ ] Service visibility/filter control works when more than one service emits metrics.
- [ ] Settings can load Compose runtime resource information without a 5xx/error state.
- [ ] Reset Compose resource overrides works when there are overrides to reset.

**Failure notes**

- 

---

# F. Static project

Use a static site or a known static SPA.

- [ ] Auto detection selects Static for a supported static project.
- [ ] Static project can be created without requiring an application runtime port.
- [ ] Static project deploys successfully.
- [ ] Public URL serves the static site.
- [ ] Redeploying updated static content replaces the public content successfully.
- [ ] Static project Overview does **not** show container CPU/RAM runtime usage.
- [ ] Stop/start behavior, if exposed, behaves consistently with static routing semantics and does not invent a persistent app container.

**Failure notes**

- 

---

# G. OCI image project

Use a known-good image and its internal application port.

- [ ] Container Registry source can be selected.
- [ ] Image reference can be entered without Git repository fields being required.
- [ ] Project is created in image mode.
- [ ] Deployment/pull succeeds.
- [ ] Project reaches running state.
- [ ] Public URL reaches the image workload.
- [ ] Changing the image reference in Settings can be saved.
- [ ] Redeploy uses the newly saved image reference.

**Failure notes**

- 

---

# H. Project lifecycle

Run on a disposable non-static runtime project unless noted.

- [ ] Deploy from a stopped/pending project succeeds.
- [ ] Stop changes the project to stopped and public application is no longer treated as running.
- [ ] Start brings the project back without requiring a fresh build.
- [ ] Restart completes and the application becomes reachable again.
- [ ] Repeated clicks while an action is pending do not queue accidental duplicate lifecycle actions.
- [ ] Project status in the control panel agrees with the actual application state.
- [ ] Refreshing the browser after each lifecycle action shows the same state.

**Failure notes**

- 

---

# I. Project settings

## I1. Source settings

- [ ] Git branch change is validated against the repository before save.
- [ ] Saving a valid branch succeeds.
- [ ] Reloading Settings preserves the saved branch.
- [ ] Invalid branch/source change shows an error and does not silently save invalid state.
- [ ] Base-directory change is re-inspected before save.
- [ ] Runtime app port can be changed for non-static projects and persists after reload.
- [ ] Compose main service/file/override/profile/workdir values, when used, persist after reload.
- [ ] Static frontend path, when used, persists after reload.

## I2. Resource settings

- [ ] Resource profile can be changed and saved.
- [ ] Memory limit can be changed and saved.
- [ ] CPU limit can be changed and saved.
- [ ] Reloading Settings shows the saved resource values.
- [ ] A subsequent deployment/runtime uses the new resource settings rather than only changing the UI.
- [ ] Invalid resource values are rejected with a clear message.

**Failure notes**

- 

---

# J. Environment variables

Use a disposable secret/value.

- [ ] Environment page lists configured variable keys without exposing values by default.
- [ ] Add/update environment variables succeeds.
- [ ] Refreshing the page preserves the variables.
- [ ] Reveal returns the correct value only when explicitly requested.
- [ ] Editing an existing variable persists the new value.
- [ ] Deleting a variable removes it from the list.
- [ ] Redeploy after an environment change makes the new value available to the workload.
- [ ] Secrets are not printed in ordinary deployment/log UI unexpectedly.

**Failure notes**

- 

---

# K. Logs and live stream

- [ ] Logs page loads recent application logs.
- [ ] New log lines appear while the application is running/producing logs.
- [ ] Navigating away and back does not create visibly duplicated stream output.
- [ ] Browser refresh reconnects to the stream.
- [ ] Logs for stopped/unavailable workloads fail gracefully instead of hanging forever.
- [ ] Deployment build/output logs remain distinguishable from normal runtime logs where applicable.

**Failure notes**

- 

---

# L. Metrics and edge analytics

## L1. Runtime metrics

For a non-static runtime project:

- [ ] Runtime metrics connect/live-refresh.
- [ ] CPU usage is displayed against the project's allocation.
- [ ] Memory usage is displayed against the project's allocation.
- [ ] Compose service metrics are attributable to the correct service when available.
- [ ] Temporary telemetry loss shows reconnecting/unavailable state rather than fabricated zero usage.

For a static project:

- [ ] Container runtime CPU/RAM section is absent.

## L2. Cloudflare analytics — optional

- [ ] When Cloudflare is not configured, UI clearly says analytics are optional/not configured.
- [ ] Valid Cloudflare configuration can be saved.
- [ ] Once configured, project analytics can load request/bandwidth/error data.
- [ ] Analytics failure does not break project runtime telemetry or the whole Overview page.

**Failure notes**

- 

---

# M. Deployment history and rollback

Create at least two known-good revisions/deployments of the same disposable project.

- [ ] Deployment history lists newest deployments correctly.
- [ ] Individual deployment details open correctly.
- [ ] Commit/image identity shown in history corresponds to what was deployed.
- [ ] Failed deployment is clearly distinguishable from successful/running history.
- [ ] Rollback can be triggered to a prior successful deployment.
- [ ] Rollback reaches a terminal success state.
- [ ] Public application content/version matches the rolled-back deployment.
- [ ] Project Overview and active deployment state agree after rollback.

**Failure notes**

- 

---

# N. Compose additional HTTP routes

Use a Compose project with an additional HTTP service/port.

- [ ] Additional route configuration loads for the Compose project.
- [ ] A valid additional route can be saved.
- [ ] Project Overview displays the additional endpoint.
- [ ] Additional endpoint hostname opens the intended service and port.
- [ ] Main project route continues to reach the main service.
- [ ] Stop removes/disables additional endpoint reachability consistently with project state.
- [ ] Start restores the additional endpoint.
- [ ] Restart preserves the additional endpoint.
- [ ] Redeploy preserves the configured route contract.
- [ ] Deleting the project removes the additional route.

**Failure notes**

- 

---

# O. Shared PostgreSQL — optional

Create a disposable project with **Shared Postgres** enabled.

- [ ] Project creation successfully provisions shared PostgreSQL state.
- [ ] Managed database environment/connection is available to the workload.
- [ ] Workload can connect to the database.
- [ ] Restart/redeploy does not unexpectedly lose database data.
- [ ] Project deletion cleans project-owned shared database resources according to the product behavior.

**Failure notes**

- 

---

# P. Database Studio

Use a disposable database/table with non-critical rows.

## P1. Read mode

- [ ] Database Studio detects the configured database connection.
- [ ] Driver/database identity displayed is correct.
- [ ] Schemas load.
- [ ] Tables load.
- [ ] Table search filters table names.
- [ ] Selecting a table loads rows.
- [ ] Row pagination/infinite loading can load beyond the first page when enough rows exist.
- [ ] String row search narrows the selected table's rows as expected.
- [ ] Changing table clears the previous table's row search state.

## P2. Controlled write mode

Current UI exposes a time-bounded write session and existing-row editing. Use a disposable row.

- [ ] Database Studio starts read-only.
- [ ] Enabling write mode creates a visible 15-minute write session.
- [ ] A table with a primary key and editable columns allows editing an existing row.
- [ ] Updating one safe field persists to the database.
- [ ] `NULL` behavior works on a nullable field when tested.
- [ ] Database-generated current-time behavior works on a compatible temporal field when tested.
- [ ] Revoking write mode returns the UI to read-only.
- [ ] After revoke, an ordinary edit cannot be submitted as if write mode were still active.

> Do not invent an Insert/Delete UI test here unless the current UI explicitly exposes those controls. The backend has mutation endpoints, but this checklist follows the user-visible source on this baseline.

**Failure notes**

- 

---

# Q. GitHub webhook deployment

Use a disposable Git-backed project.

- [ ] Webhook page shows a payload URL.
- [ ] Webhook secret can be revealed/copied intentionally.
- [ ] A correctly configured GitHub push webhook reaches MyPaaS.
- [ ] Webhook status records the delivery.
- [ ] A signed push to the configured branch queues a deployment.
- [ ] The webhook-triggered deployment reaches the expected terminal state.
- [ ] Push to a different branch does not incorrectly deploy the configured branch.
- [ ] Invalid signature does not trigger deployment and is represented as a delivery issue/rejected delivery.
- [ ] Regenerating the webhook secret invalidates the previous secret as expected.

**Failure notes**

- 

---

# R. Containers workspace

- [ ] Containers page loads host runtime inventory.
- [ ] Search matches container name/image/project/service/port information.
- [ ] State filter works.
- [ ] Runtime/project-group filter works.
- [ ] Pagination/page-size controls work when enough containers exist.
- [ ] Expanding a row shows container identity/runtime/network details.
- [ ] Running/stopped/health information matches the actual project runtime state.
- [ ] Container workspace remains metadata/read-only; project lifecycle actions are performed from the project surface.

If a secondary non-owner account exists:

- [ ] Containers access behaves consistently with the fact that the primary navigation exposes or hides it for that user.

**Failure notes**

- 

---

# S. Shell — owner only

Use harmless commands only.

- [ ] Shell page opens a session and reaches Connected state.
- [ ] `pwd` returns output.
- [ ] A second simple command can be entered without starting a new session.
- [ ] Output copy works.
- [ ] Start a harmless long-running command such as `sleep 30`.
- [ ] Clicking **Interrupt** stops the running command and returns the session to usable input.
- [ ] Pressing `Ctrl+C` also interrupts a harmless long-running command.
- [ ] After interrupt, a new command still works in the same session.
- [ ] Ending the session changes it to Ended/no-active-session state.
- [ ] Starting a new session after ending the previous one works.
- [ ] Leaving the Shell page does not leave an obviously active abandoned session.

**Failure notes**

- 

---

# T. Administration — General

Use temporary values and restore the intended defaults after testing.

## T1. Platform limits

- [ ] General settings load current effective values.
- [ ] User RAM quota can be changed within allowed range and persists after refresh.
- [ ] User CPU quota can be changed within allowed range and persists after refresh.
- [ ] Maximum projects can be changed within allowed range and persists after refresh.
- [ ] Build timeout can be changed within allowed range and persists after refresh.
- [ ] Invalid/out-of-range settings are rejected clearly.

## T2. Resource profile defaults

- [ ] Static memory default can be changed and persists.
- [ ] Static CPU default accepts `0.01` CPU and persists.
- [ ] Go-small profile defaults can be changed within supported bounds.
- [ ] Node/Python profile defaults can be changed within supported bounds.
- [ ] Compose-main profile defaults can be changed within supported bounds.
- [ ] Opening New Project after changing a profile uses the current configured defaults rather than stale hard-coded UI values.

## T3. Host stats

- [ ] Host statistics load.
- [ ] Host RAM/CPU allocation values are plausible for the current VM.
- [ ] Storage/network fields either show valid values or a clear unavailable state.

**Failure notes**

- 

---

# U. Administration — Users

Use a disposable secondary email/account when possible.

- [ ] User list loads.
- [ ] Adding an allowed user succeeds.
- [ ] Added user can authenticate through the intended GitHub flow.
- [ ] Removing the disposable user succeeds.
- [ ] Removed user no longer retains ordinary application access after the relevant session/auth boundary is re-evaluated.
- [ ] Owner cannot accidentally remove/lock out the required owner identity through an ordinary unsafe path.

**Failure notes**

- 

---

# V. Administration — Audit logs

Perform several known actions first: settings update, project update, deploy, env change, etc.

- [ ] Audit logs page loads.
- [ ] Recent known actions appear.
- [ ] Actor/identity is correct.
- [ ] Target/action metadata is understandable enough to identify what changed.
- [ ] Secret values are not exposed in ordinary audit metadata.
- [ ] Pagination works when enough audit records exist.

**Failure notes**

- 

---

# W. Local MCP bridge on current `main`

This baseline uses the local STDIO MCP bridge in `backend/cmd/mcp`, not the draft remote `/mcp` endpoint.

## W1. Token management

- [ ] Administration → MCP loads the configured API token state.
- [ ] Reveal/hide token works intentionally.
- [ ] Copy token works.
- [ ] Regenerating the MCP token produces a new token.
- [ ] Old token stops authenticating after regeneration.

## W2. Local agent bridge — optional

Configure the local bridge with the displayed API target/token.

- [ ] Agent connects through the STDIO bridge.
- [ ] Read-only `list projects` works.
- [ ] Read-only project inspection works.
- [ ] Deployment history/logs/metrics reads work for an existing project.
- [ ] Quota/host-stat read works when exposed to the bridge.
- [ ] Environment variable **names/listing** works without accidental secret exposure.
- [ ] Perform one explicit disposable mutation (for example a harmless env update) and verify the same state appears in the dashboard.

**Failure notes**

- 

---

# X. Backup

## X1. Local/manual backup download

- [ ] Backup page loads.
- [ ] Manual backup download starts successfully.
- [ ] Downloaded file is non-empty and has the expected archive type.
- [ ] Creating a backup does not leave projects visibly broken/stopped afterward.

## X2. S3 / Cloudflare R2 — optional

Use disposable/approved backup credentials.

- [ ] S3-compatible endpoint/bucket/region/access credentials can be entered.
- [ ] **Test connection** validates the bucket successfully.
- [ ] Saving is not enabled until connection validation succeeds.
- [ ] Saving valid credentials reports configured state.
- [ ] Secret key is not repopulated into the browser as plaintext after reload.
- [ ] Triggering backup produces expected object/archive behavior in the configured bucket.
- [ ] Invalid credentials fail validation clearly without being treated as configured.

**Failure notes**

- 

---

# Y. Migration package

> This operation intentionally pauses running project runtimes briefly while capturing state. Use a disposable/staging instance.

- [ ] Migration page starts in a clear idle/not-prepared state.
- [ ] Preparing a migration asks for confirmation.
- [ ] Preparation enters `preparing` state and status polling progresses.
- [ ] Package reaches `ready` state.
- [ ] Running project runtimes are usable again after preparation completes.
- [ ] Download package link works with the generated download token.
- [ ] Downloaded package is non-empty.
- [ ] Generated destination install/migration command can be copied.
- [ ] Package reports expiration information.

### Destination restore — optional real VM test

- [ ] Fresh destination VM can consume the generated migration package.
- [ ] Platform database/state is restored.
- [ ] Persistent project data required by the tested workloads is restored.
- [ ] Compose named-volume data used by tested projects is restored.
- [ ] Static project content is restored.
- [ ] Restored project routes become reachable on the destination after normal reconciliation.

**Failure notes**

- 

---

# Z. System update — isolated upgrade test only

> **Destructive/operational test. Do not run merely to complete the checklist.** Run only when a newer qualified release is intentionally available and this instance can be upgraded safely.

- [ ] System Update page shows the actual installed revision.
- [ ] Available release/tag/SHA is displayed correctly when an update exists.
- [ ] Release notes/highlights correspond to the offered release.
- [ ] Queueing update enters queued/in-progress state.
- [ ] Navigation-away protection appears while update is active.
- [ ] Dashboard reconnects after its own service restart during update.
- [ ] Update reaches `succeeded`, or a clear `failed`/`rolled_back`/`blocked` terminal state.
- [ ] Successful update reports the new installed revision.
- [ ] Existing projects remain available after the update.
- [ ] Existing environment/persistent state remains intact after the update.
- [ ] Core smoke checks (login, projects list, one project URL, deploy action) still pass after update.

**Failure notes**

- 

---

# AA. Project deletion and cleanup

Use the disposable projects created by this checklist. For data you need to keep, do not run this section.

## AA1. Runtime project

- [ ] Delete requires the intended confirmation in the UI.
- [ ] Deleted project disappears from Projects list.
- [ ] Its primary public route no longer resolves to the deleted workload.
- [ ] Its application runtime/container is no longer left running.
- [ ] Reloading the old project URL does not restore a ghost project.

## AA2. Compose project

- [ ] Compose workload containers/resources owned by the project are cleaned according to MyPaaS behavior.
- [ ] Additional HTTP routes no longer serve the deleted project.
- [ ] Project-specific Compose runtime state does not remain presented as an active project.

## AA3. Static project

- [ ] Static project disappears from Projects list.
- [ ] Static public route no longer serves the deleted project.
- [ ] Deleted static project is not resurrected by normal route reconciliation.

**Failure notes**

- 

---

# AB. Restart and reconciliation — recommended staging/VM test

This validates behavior that a browser-only happy path cannot prove.

- [ ] Keep at least one Dockerfile/image project running before restart.
- [ ] Keep at least one Compose project running before restart.
- [ ] Keep at least one static project deployed before restart.
- [ ] Restart the MyPaaS control-plane stack/API in the normal supported way.
- [ ] API returns healthy/ready again.
- [ ] Previously running runtime projects recover or reconcile to the correct state.
- [ ] Static project route is still served/reconciled correctly.
- [ ] Compose additional routes, if configured, are restored correctly.
- [ ] Projects that were intentionally stopped are not incorrectly started as running workloads.
- [ ] No deployment remains permanently stuck only because the control plane restarted.

### Full VM reboot — optional

- [ ] Reboot the staging VM.
- [ ] MyPaaS stack comes back through the configured host startup mechanism.
- [ ] Dashboard login works.
- [ ] Existing projects and state are present.
- [ ] Expected public routes recover.
- [ ] Persistent data survives.

**Failure notes**

- 

---

# AC. Final regression pass

After all applicable sections:

- [ ] Login/auth still works.
- [ ] Projects list loads without error.
- [ ] Create Project opens and repository inspection still works.
- [ ] At least one runtime project is reachable publicly.
- [ ] At least one static project is reachable publicly.
- [ ] Logs still load.
- [ ] Runtime metrics still load for non-static workload.
- [ ] Containers inventory still loads for the intended user role.
- [ ] Administration pages still load for owner.
- [ ] No disposable project that should have been deleted remains visible as active.
- [ ] No unexpected test secret is left in a project environment.
- [ ] Test-only users/credentials/resources that should be removed have been cleaned up.

**Failure notes**

- 

---

# Failure summary

Only list failures worth fixing/reviewing. Keep each item short; detailed evidence can live in an issue/PR later.

| ID | Section / checkbox | What failed | Reproduction / evidence | Blocking stable? |
| --- | --- | --- | --- | --- |
| E2E-001 | | | | |
| E2E-002 | | | | |
| E2E-003 | | | | |
| E2E-004 | | | | |
| E2E-005 | | | | |

---

# Final run verdict

Tick exactly one when the run is complete.

- [ ] **PASS** — all applicable core product flows passed; no stable blocker found.
- [ ] **PASS WITH NOTES** — core flows work; failures are non-blocking or optional integrations.
- [ ] **FAIL** — one or more confirmed failures block stable release.
- [ ] **INCOMPLETE** — important sections were not tested yet.

### Stable blockers

- 

### Non-blocking notes

- 

### Sections intentionally skipped / N/A

- 
