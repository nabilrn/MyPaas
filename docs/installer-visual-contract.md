# Installer visual contract

The standalone browser installer uses the same product grammar as the authenticated MyPaaS workspace without importing the Svelte/Tailwind runtime.

## Source of truth

- `frontend/DESIGN.md` defines the product character, stroke grammar, control geometry, typography, and surface rules.
- `frontend/src/app.css` defines canonical light/dark application tokens.
- Administration Backup, MCP, Settings, and project Webhook are the canonical interaction references for provider credentials, secret handling, validation, derived values, and compact configuration layouts.
- `scripts/install_wizard_visual.py` renders the dependency-free installer directly from that grammar instead of post-processing the legacy installer DOM.
- `scripts/install_wizard_visual_test.py` fails when canonical token values or the workspace contract drift.

The installer remains dependency-free: it is rendered by the Python bootstrap server and must work before the MyPaaS dashboard is built or running.

## Geometry

- the installer is a full workspace, not a centered setup card;
- the top application header is 56px;
- the desktop setup rail uses the same 12rem structural width as authenticated secondary navigation;
- the rail and main workspace are separated by the canonical low-contrast divider;
- the main workspace consumes the available canvas and uses the same compact 14px outer inset as Administration;
- first-level sections are flat and separated by full-width 1px dividers;
- controls are 36px high on desktop and keep intrinsic widths where possible;
- provider configuration may use the Administration Backup pattern: editable values on the left, derived values and live validation on the right;
- rounded geometry is reserved for controls and inset interaction elements, not first-level cards.

## Flow

A fresh installation has four task steps:

1. Domain
2. GitHub
3. Cloudflare
4. Review

Restore is an alternate entry action, not a mandatory setup step. Opening Restore exposes the existing authenticated backup upload path without changing the fresh-install progress rail.

The root domain is the canonical onboarding model. The installer derives:

- dashboard: `https://example.com`
- projects: `*.example.com`

Do not advertise `panel.example.com` or other nested-hostname examples in the default flow. Advanced operators can still supply another valid hostname, but onboarding copy must not steer users toward deeper project hostnames.

## Provider interaction patterns

### GitHub

- owner email belongs to the GitHub step;
- owner copy states that the verified primary email is used for first binding and the durable owner identity is the GitHub numeric user ID;
- generated Homepage and Callback URLs are plain mono values with compact Copy actions, following Webhook/MCP patterns rather than bordered code chips;
- the Client Secret is masked and has an explicit reveal/hide action;
- link to official GitHub OAuth App documentation instead of keeping a long tutorial permanently visible.

### Cloudflare

- accept either the Tunnel token or the full Add-a-replica command;
- clearly state that Cloudflare API tokens are not accepted;
- show the derived dashboard hostname, wildcard hostname, and `http://caddy:80` service as compact rows;
- link to official Cloudflare Tunnel documentation;
- do not keep a permanent multi-section DNS tutorial in the happy path. Live validation owns failure-specific guidance.

## Validation

Installer copy must not claim that a diagnostic proves more than it actually verifies.

- Domain checks DNS resolution from the installer machine.
- GitHub preflight checks OAuth credentials and callback format; owner identity is verified during the actual sign-in.
- Cloudflare preflight briefly connects using the Tunnel token and then verifies project wildcard DNS.
- editing a qualified value makes that qualification stale and final save revalidates fingerprints server-side.

## Secrets

Secrets use the same interaction grammar as MCP/Webhook:

- masked by default;
- explicit reveal/hide control;
- never shown in Review;
- never echoed in validation errors or logs.

## Scope boundary

This contract changes presentation and explanatory copy only. Preflight network behavior, OAuth identity binding, backup handling, `.env` generation, and installer lifecycle remain owned by their respective implementation layers.
