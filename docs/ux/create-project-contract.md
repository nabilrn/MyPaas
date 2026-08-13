# Create Project UX Contract

This is the audit source of truth for the MyPaaS Create Project flow. The Playwright audit harness should compare the rendered UI against this contract and collect evidence only. It must not redesign or fix the Create Project page.

## Intent

Create Project is a single-page, source-first flow:

1. The user provides a deploy source.
2. MyPaaS inspects the source automatically.
3. MyPaaS detects the runtime and deployment shape.
4. Required configuration is shown before optional configuration.
5. The user creates the project only after the current analysis is complete and valid.

Do not introduce a wizard. Keep the existing single-page flow.

## Required Behavior

- The Git repository input automatically starts repository inspection.
- Runtime and deployment type are detected, not asked initially.
- The project name may be auto-suggested from the repository or image reference until the user edits it.
- Deployment Type remains visible in the normal flow.
- Environment detection remains visible in the normal flow.
- Required configuration appears before optional configuration.
- Advanced settings are discoverable but secondary.
- Base Directory normally remains Advanced unless root detection fails or the user needs a manual override.
- Create cannot become ready while analysis is stale, incomplete, scheduled, or in flight.
- Re-analysis invalidates stale repository/runtime/environment results.
- Re-analysis must not leave a previous ready state available while new analysis is pending.
- Blocking Compose Doctor findings prevent readiness and surface an actionable message.
- Required environment values block readiness until provided or satisfied by a managed capability.
- Registry/image flows must expose the required container port when it cannot be inferred.

## Audit Expectations

The production audit answers: "What does the real deployed UI behave and look like?"

The mocked audit answers: "Does every important state behave correctly and reproducibly?"

Production audits are non-destructive by default. They may paste repository URLs, trigger repository inspection, trigger runtime detection, expand Advanced settings, use Re-analyze, and collect screenshots, console, network, ARIA, text, controls, timing, and geometry evidence. They must not create real projects, deploy applications, delete resources, or mutate production resources.

Mocked audits may simulate edge states that are unsafe or difficult to reproduce in production, including slow inspection, backend failures, timeouts, static detection, Dockerfile detection, Compose detection, missing required env values, Compose Doctor blockers, missing ports, nested base-directory cases, stale re-analysis results, and project creation failure.

## Evidence Checkpoints

Use stable checkpoint names where applicable:

- `00-initial`
- `01-source-entered`
- `02-analyzing`
- `03-runtime-detected`
- `04-configuration-required`
- `05-ready`
- `06-submitting`
- `07-created`

Non-destructive production audits normally omit submitting/created checkpoints.

At each useful checkpoint collect:

- screenshot
- current URL
- accessibility/ARIA representation
- important visible text
- visible controls
- Create button enabled or disabled state
- currently focused element
- console errors and warnings
- failed or relevant API requests
- important element geometry and bounding boxes

## Geometry Checks

Capture bounding boxes for:

- main page container
- Create Project form
- source selector
- repository input
- project name
- analysis timeline
- deployment type
- environment section
- Advanced trigger
- Advanced content
- Create Project CTA

Flag simple DOM geometry issues only:

- excessive horizontal whitespace
- inconsistent left alignment
- elements outside the viewport
- horizontal overflow
- overlapping controls
- tiny controls
- strange layout shifts

This is not a computer-vision system. Browser DOM geometry is sufficient.
