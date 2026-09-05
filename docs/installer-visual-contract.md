# Installer visual contract

The standalone browser installer uses the same product grammar as the authenticated MyPaaS workspace without importing the Svelte/Tailwind runtime.

## Source of truth

- `frontend/DESIGN.md` defines the product character, stroke grammar, control geometry, typography, and surface rules.
- `frontend/src/app.css` defines canonical light/dark application tokens.
- `scripts/install_wizard_visual.py` mirrors the small subset of those tokens required by the dependency-free installer.
- `scripts/install_wizard_visual_test.py` fails when canonical token values drift away from the installer copy.

The installer remains dependency-free: it is rendered by the Python bootstrap server and must work before the MyPaaS dashboard is built or running.

## Geometry

- no floating first-level card stack;
- one flat workspace surface;
- desktop step rail uses the same 12rem structural width as authenticated secondary navigation;
- the rail and form are separated by the canonical low-contrast workspace divider;
- controls are 36px high on desktop;
- rounded geometry is reserved for controls, badges, and technical/inset elements;
- content sections are separated with full-width 1px dividers instead of alternating fills.

## Copy

Installer copy is short and task-oriented. It must not claim that a diagnostic proves more than it actually verifies.

The owner field describes a **verified primary GitHub email** because the first successful login matches that verified email against the whitelist. After that first binding, the durable identity is GitHub's numeric user ID; changing the account's primary email must not transfer ownership or lock out the already-bound account.

Cloudflare setup explicitly accepts either the Tunnel token itself or the full Add-a-replica command because the preflight layer normalizes the command before saving configuration.

## Scope boundary

This contract changes presentation and explanatory copy only. Preflight network behavior, OAuth identity binding, backup handling, `.env` generation, and installer lifecycle remain owned by their respective implementation stages.
