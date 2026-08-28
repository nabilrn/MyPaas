---
name: mypaas-dashboard-no-slop
description: Use when creating, refactoring, or visually polishing MyPaaS dashboard UI. Enforces compact operational UX, removes AI-style explanatory clutter, and preserves information hierarchy without hiding important system state.
---

# MyPaaS Dashboard No-Slop UX

MyPaaS is an operational control plane, not a marketing page. Dashboard surfaces should feel compact, calm, legible, and tool-like.

Load this skill together with:

- `.agents/skills/mypaas-dashboard-state-audit/SKILL.md`
- `.agents/skills/mypaas-svelte/SKILL.md`

State correctness wins over cosmetic simplification. Never remove security, destructive-action, failure, stale-state, or blocking validation information merely to make a page look cleaner.

## Core rule

Prefer **state + data + action** over explanatory prose.

Bad:

```text
Project overview
Here you can view all of the important information about your project and monitor its current status.
Your project is currently running and accessible from the public URL below.
```

Good:

```text
my-app                         Running
my-app.example.com             ↗
CPU 4%   RAM 214 MB
Restart   Redeploy   Stop
```

## Information hierarchy

For normal operational pages, use this order:

1. object identity;
2. current state;
3. primary operational data;
4. primary actions;
5. errors/warnings requiring attention;
6. secondary configuration behind compact sections or disclosure.

Do not place generic explanatory copy above the state or action it is supposed to explain.

## Copy budget

Default limits unless the workflow genuinely requires more:

- page subtitle: 0-1 short sentence;
- panel description: omit when the title/data already explain the panel;
- helper text: one short line next to the affected control;
- empty state: title + one actionable sentence;
- warning: concise reason + consequence + next action;
- cards: avoid free-form descriptive paragraphs.

If the same fact appears in a heading, badge, field label, and paragraph, keep the most useful representation and remove the duplicates.

## Visual anti-patterns

Avoid AI-generated dashboard tropes:

- a card for every small piece of information;
- stacked heading + subtitle + helper paragraph + footer explanation;
- decorative gradients or glowing effects;
- excessive pill-shaped controls/badges;
- unnecessary icons beside obvious labels;
- oversized whitespace around low-information content;
- generic headings such as `Overview`, `Details`, or `Information` when a precise object/state label exists;
- repeated prose explaining standard CRUD controls;
- large hero-like sections inside authenticated operational pages.

Keep the existing monochrome visual language. Semantic warning/error/success color is allowed when it conveys state.

## Layout rules

- Prefer one strong page header, not nested marketing-style headers.
- Use dense rows/tables for comparable operational data.
- Use cards/panels only when they establish a real grouping boundary.
- Keep common actions visible without scrolling where practical.
- Put destructive actions near the object they affect, with confirmation when required.
- Keep advanced/rare settings secondary.
- Mobile layouts must preserve labels and actionable state; do not depend on hover.

## Progressive disclosure

Move long explanations to one of these surfaces instead of leaving them inline:

- tooltip for terminology;
- small `Why?`/help disclosure for unusual behavior;
- docs link for conceptual explanation;
- expandable advanced section for rarely edited configuration.

Do not hide information required to safely complete the current task.

## Operational state vocabulary

Prefer short state labels:

- `Running`
- `Stopped`
- `Deploying`
- `Failed`
- `Update available`
- `Disconnected`
- `Read-only`
- `Required`
- `Optional`

Avoid sentences such as `Your project is currently in a running state`.

## Forms

- Labels describe the value; helper text describes only a non-obvious constraint.
- Required/optional status should not be repeated in multiple places.
- Validation belongs directly next to the affected field.
- Do not explain standard text inputs, selects, or buttons.
- Advanced fields remain collapsed/secondary unless they block the current workflow.

## Tables and metrics

- Prefer scannable numeric alignment and short labels.
- Units must be explicit.
- Avoid prose cards for values that can be shown as rows or compact metric cells.
- Do not turn compatibility evidence into benchmark/capacity marketing.

## Required audit before a visual refactor

For each page being changed:

1. list every visible sentence/description/helper;
2. mark whether it is required for safety, validation, empty/error state, or non-obvious behavior;
3. delete or collapse duplicate/explanatory copy;
4. verify primary state and primary action remain visually dominant;
5. verify loading/error/empty/disabled states with `mypaas-dashboard-state-audit`;
6. run `pnpm check`, relevant frontend tests, and `pnpm build`.

## Stop conditions

Do not continue broad visual cleanup when:

- the change starts altering backend behavior;
- a state-management defect is discovered;
- the page needs product decisions rather than styling;
- the refactor expands into unrelated pages.

Split those into separate work.
