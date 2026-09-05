from __future__ import annotations


VISUAL_CONTRACT_CSS = r"""
  /* Standalone installer implementation of frontend/DESIGN.md. */
  :root {
    color-scheme: light;
    font-family: "Inter Variable", "Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    --app-bg: #fafafa;
    --app-surface: #ffffff;
    --app-surface-muted: #f7f7f7;
    --app-surface-raised: #ffffff;
    --app-border: #dedede;
    --app-border-strong: #c9c9c9;
    --app-ink: #171717;
    --app-muted: #525252;
    --app-subtle: #737373;
    --workspace-divider: color-mix(in oklch, var(--app-border) 82%, transparent);
    --app-accent: #171717;
    --app-accent-strong: #0a0a0a;
    --app-accent-soft: #e5e5e5;
    --app-danger: #dc2626;
    --app-warning: #d97706;
    --app-info: #0284c7;
    --app-success: #059669;
    --control-bg: transparent;
    --control-bg-disabled: color-mix(in oklch, var(--app-surface-muted) 82%, var(--app-surface));
    --control-border: color-mix(in oklch, var(--app-border) 68%, transparent);
    --control-border-hover: color-mix(in oklch, var(--app-border-strong) 88%, var(--app-border));
    --control-border-focus: var(--app-ink);
    --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 58%, transparent);
    background: var(--app-surface);
    color: var(--app-ink);
  }

  :root.dark {
    color-scheme: dark;
    --app-bg: #0a0a0a;
    --app-surface: #141414;
    --app-surface-muted: #1a1a1a;
    --app-surface-raised: #171717;
    --app-border: #303030;
    --app-border-strong: #4a4a4a;
    --app-ink: #fafafa;
    --app-muted: #a3a3a3;
    --app-subtle: #737373;
    --workspace-divider: color-mix(in oklch, var(--app-border) 82%, transparent);
    --app-accent: #fafafa;
    --app-accent-strong: #ffffff;
    --app-accent-soft: #333333;
    --app-danger: #f87171;
    --app-warning: #fbbf24;
    --app-info: #38bdf8;
    --app-success: #34d399;
    --control-bg: transparent;
    --control-bg-disabled: color-mix(in oklch, var(--app-surface-muted) 78%, var(--app-surface));
    --control-border: color-mix(in oklch, var(--app-border) 62%, transparent);
    --control-border-hover: color-mix(in oklch, var(--app-border-strong) 86%, var(--app-border));
    --control-border-focus: var(--app-ink);
    --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 62%, transparent);
  }

  @media (prefers-color-scheme: dark) {
    :root:not(.light) {
      color-scheme: dark;
      --app-bg: #0a0a0a;
      --app-surface: #141414;
      --app-surface-muted: #1a1a1a;
      --app-surface-raised: #171717;
      --app-border: #303030;
      --app-border-strong: #4a4a4a;
      --app-ink: #fafafa;
      --app-muted: #a3a3a3;
      --app-subtle: #737373;
      --workspace-divider: color-mix(in oklch, var(--app-border) 82%, transparent);
      --app-accent: #fafafa;
      --app-accent-strong: #ffffff;
      --app-accent-soft: #333333;
      --app-danger: #f87171;
      --app-warning: #fbbf24;
      --app-info: #38bdf8;
      --app-success: #34d399;
      --control-bg: transparent;
      --control-bg-disabled: color-mix(in oklch, var(--app-surface-muted) 78%, var(--app-surface));
      --control-border: color-mix(in oklch, var(--app-border) 62%, transparent);
      --control-border-hover: color-mix(in oklch, var(--app-border-strong) 86%, var(--app-border));
      --control-border-focus: var(--app-ink);
      --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 62%, transparent);
    }
  }

  * { box-sizing: border-box; }
  [hidden] { display: none !important; }
  html, body { margin: 0; min-height: 100%; background: var(--app-surface); color: var(--app-ink); }
  body { min-height: 100vh; -webkit-font-smoothing: antialiased; }
  button, input { font: inherit; }
  button { cursor: pointer; }
  button:disabled { cursor: not-allowed; opacity: .55; }
  a { color: inherit; text-underline-offset: 3px; }
  h1, h2, h3, p, dl, dt, dd { margin: 0; }
  code, .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }

  /* Authenticated shell rule: chrome and workspace use one surface. */
  .app-shell,
  .app-header,
  .workspace,
  .setup-rail,
  .setup-main,
  .setup-form,
  .page-heading,
  .step-primary,
  .step-aside,
  .restore-panel {
    background: var(--app-surface);
  }

  /* Desktop installer owns exactly one viewport. Long content scrolls inside the active pane. */
  .app-shell {
    height: 100dvh;
    min-height: 100dvh;
    display: grid;
    grid-template-rows: 56px minmax(0, 1fr);
    overflow: hidden;
  }
  @supports not (height: 100dvh) {
    .app-shell { height: 100vh; min-height: 100vh; }
  }

  .app-header {
    display: flex;
    min-width: 0;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    border-bottom: 1px solid var(--app-border);
    padding: 0 14px;
  }
  .brand-lockup { display: inline-flex; min-width: 0; align-items: center; gap: 9px; }
  .brand-logo { display: block; width: 112px; height: auto; object-fit: contain; object-position: left center; }
  :root.dark .brand-logo { filter: invert(1); }
  @media (prefers-color-scheme: dark) { :root:not(.light) .brand-logo { filter: invert(1); } }
  .installer-label { color: var(--app-subtle); font-size: 12px; font-weight: 600; }
  .header-actions { display: flex; align-items: center; gap: 2px; }

  .icon-button,
  .ghost-button,
  .secondary-button,
  .primary-button {
    display: inline-flex;
    min-height: 36px;
    align-items: center;
    justify-content: center;
    gap: 7px;
    border-radius: 6px;
    padding: 0 12px;
    font-size: 14px;
    font-weight: 600;
    transition: border-color .14s ease, background-color .14s ease, color .14s ease, box-shadow .14s ease;
  }
  .icon-button {
    width: 36px;
    padding: 0;
    border: 1px solid transparent;
    background: transparent;
    color: var(--app-muted);
  }
  .icon-button svg { display: block; width: 18px; height: 18px; }
  .icon-button:hover, .ghost-button:hover { background: var(--app-surface-muted); color: var(--app-ink); }
  .ghost-button { border: 1px solid transparent; background: transparent; color: var(--app-muted); }
  .secondary-button { border: 1px solid var(--control-border); background: transparent; color: var(--app-ink); }
  .secondary-button:hover { border-color: var(--control-border-hover); }
  .primary-button { border: 1px solid var(--app-ink); background: var(--app-ink); color: var(--app-surface); }
  .primary-button:hover { border-color: var(--app-accent-strong); background: var(--app-accent-strong); }
  .icon-button:focus-visible,
  .ghost-button:focus-visible,
  .secondary-button:focus-visible,
  .primary-button:focus-visible,
  .field-input:focus-visible,
  .secret-toggle:focus-visible,
  .backup-dropzone:focus-visible,
  .backup-file-clear:focus-visible,
  .step-tab:focus-visible {
    outline: none;
    box-shadow: 0 0 0 3px var(--control-focus-ring);
  }

  .workspace {
    display: grid;
    min-height: 0;
    height: calc(100dvh - 56px);
    grid-template-columns: 12rem minmax(0, 1fr);
    overflow: hidden;
  }
  @supports not (height: 100dvh) {
    .workspace { height: calc(100vh - 56px); }
  }

  .setup-rail {
    min-width: 0;
    min-height: 0;
    overflow-y: auto;
    border-right: 1px solid var(--workspace-divider);
    padding: 16px 12px;
  }
  .rail-group-label {
    padding: 0 8px;
    color: var(--app-subtle);
    font-size: 11px;
    font-weight: 650;
    letter-spacing: .08em;
    text-transform: uppercase;
  }
  .stepper { display: grid; gap: 4px; margin-top: 8px; }
  .step-tab {
    appearance: none;
    width: 100%;
    display: flex;
    min-height: 36px;
    align-items: center;
    gap: 10px;
    border: 0;
    border-left: 2px solid transparent;
    background: transparent;
    padding: 8px 10px;
    color: var(--app-muted);
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    transition: color .14s ease, border-color .14s ease;
  }
  .step-tab:hover:not(:disabled) { color: var(--app-ink); }
  .step-tab.active { border-left-color: var(--app-ink); color: var(--app-ink); }
  .step-tab.done { color: var(--app-muted); }
  .step-tab:disabled { opacity: .42; }
  .step-icon { display: inline-flex; width: 16px; height: 16px; flex: 0 0 16px; align-items: center; justify-content: center; }
  .step-icon svg { width: 16px; height: 16px; stroke: currentColor; }
  .step-title { display: block; line-height: 1.25; }
  .step-body { display: none; }

  .setup-main {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 12px 14px;
  }
  .setup-form {
    flex: 1 1 auto;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    overflow: hidden;
  }
  .page-heading {
    flex: 0 0 auto;
    display: flex;
    min-height: 70px;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    border-bottom: 1px solid var(--workspace-divider);
    padding: 15px 20px 12px;
  }
  .page-heading h2 { font-size: 18px; line-height: 1.3; font-weight: 650; letter-spacing: -.015em; }
  .page-heading p { margin-top: 4px; max-width: 70ch; color: var(--app-muted); font-size: 13px; line-height: 1.5; }
  .step-position { flex: 0 0 auto; color: var(--app-subtle); font-size: 12px; line-height: 1.5; }

  .error-banner {
    flex: 0 0 auto;
    border-bottom: 1px solid color-mix(in oklch, var(--app-danger) 34%, var(--workspace-divider));
    padding: 10px 16px;
    color: var(--app-danger);
    font-size: 13px;
    line-height: 1.45;
  }
  .wizard-step[hidden] { display: none; }
  .wizard-step:not([hidden]) {
    display: block;
    flex: 1 1 auto;
    min-height: 0;
    overflow: auto;
    overscroll-behavior: contain;
  }
  .step-grid { display: grid; min-height: 100%; grid-template-columns: minmax(0, 1.45fr) minmax(20rem, .75fr); }
  .step-primary { min-width: 0; border-right: 1px solid var(--workspace-divider); padding: 16px; }
  .step-aside { min-width: 0; padding: 16px; }
  .section-heading { margin-bottom: 14px; }
  .section-heading h3 { font-size: 14px; font-weight: 650; }
  .section-heading p { margin-top: 4px; color: var(--app-muted); font-size: 12px; line-height: 1.5; }

  .field-grid { display: grid; gap: 14px; }
  .field-grid.two { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .field-group { min-width: 0; }
  .field-label { display: block; margin-bottom: 5px; color: var(--app-muted); font-size: 13px; font-weight: 600; }
  .field-hint { margin-top: 5px; color: var(--app-subtle); font-size: 12px; line-height: 1.45; }
  .field-input {
    width: 100%;
    min-height: 36px;
    border: 1px solid var(--control-border);
    border-radius: 6px;
    background: var(--control-bg);
    color: var(--app-ink);
    padding: 6px 10px;
    font-size: 14px;
    outline: none;
    transition: border-color .14s ease, box-shadow .14s ease, background-color .14s ease;
  }
  .field-input:hover:not(:disabled):not([readonly]) { border-color: var(--control-border-hover); }
  .field-input:focus { border-color: var(--control-border-focus); box-shadow: 0 0 0 3px var(--control-focus-ring); }
  .field-input[readonly], .field-input:disabled { background: var(--control-bg-disabled); color: var(--app-muted); }
  .field-input.mono { font-size: 13px; }
  .secret-wrap { position: relative; }
  .secret-wrap .field-input { padding-right: 52px; }
  .secret-toggle {
    position: absolute;
    top: 0;
    right: 0;
    width: 48px;
    height: 36px;
    border: 0;
    background: transparent;
    color: var(--app-muted);
    font-size: 0;
    font-weight: 600;
  }
  .secret-toggle::after { content: "Show"; font-size: 11px; }
  .secret-toggle[data-revealed="true"]::after { content: "Hide"; }
  .secret-toggle:hover { color: var(--app-ink); }

  /* One separator owner per section: the value list owns the provider divider. */
  .provider-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
  .provider-header strong { display: block; font-size: 14px; }
  .provider-header p { margin-top: 3px; color: var(--app-muted); font-size: 12px; line-height: 1.45; }
  .docs-link { flex: 0 0 auto; color: var(--app-muted); font-size: 12px; font-weight: 600; text-decoration: none; }
  .docs-link:hover { color: var(--app-ink); text-decoration: underline; }

  .value-list { margin-top: 12px; border-top: 1px solid var(--workspace-divider); }
  .value-row {
    display: grid;
    min-width: 0;
    grid-template-columns: 9rem minmax(0, 1fr) auto;
    align-items: center;
    gap: 12px;
    min-height: 42px;
    border-bottom: 1px solid var(--workspace-divider);
    padding: 7px 0;
  }
  .value-row > span:first-child { color: var(--app-muted); font-size: 12px; }
  .value-row .value { min-width: 0; overflow-wrap: anywhere; color: var(--app-ink); font-size: 13px; }
  .copy-button { min-height: 30px; border: 0; background: transparent; color: var(--app-muted); padding: 0 6px; font-size: 12px; font-weight: 600; }
  .copy-button:hover { color: var(--app-ink); }

  .validation-box { margin-top: 14px; }
  .validation-title { font-size: 13px; font-weight: 650; }
  .validation-copy { margin-top: 4px; color: var(--app-muted); font-size: 12px; line-height: 1.5; }
  .preflight-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-top: 12px; }
  .preflight-status { min-width: 0; color: var(--app-subtle); font-size: 12px; line-height: 1.45; }
  .preflight-status[data-state="checking"] { color: var(--app-muted); }
  .preflight-status[data-state="ok"] { color: var(--app-success); }
  .preflight-status[data-state="warning"] { color: var(--app-warning); }
  .preflight-status[data-state="error"] { color: var(--app-danger); }
  .preflight-status[data-state="stale"] { color: var(--app-subtle); }

  .review-list { border-top: 1px solid var(--workspace-divider); }
  .review-row {
    display: grid;
    grid-template-columns: 11rem minmax(0, 1fr);
    gap: 12px;
    min-height: 44px;
    align-items: center;
    border-bottom: 1px solid var(--workspace-divider);
    padding: 8px 0;
  }
  .review-row dt { color: var(--app-muted); font-size: 12px; }
  .review-row dd { min-width: 0; overflow-wrap: anywhere; font-size: 13px; }

  .advanced { border-top: 1px solid var(--workspace-divider); }
  .advanced summary { cursor: pointer; list-style: none; padding: 11px 16px; color: var(--app-muted); font-size: 12px; font-weight: 600; }
  .advanced summary::-webkit-details-marker { display: none; }
  .advanced[open] summary { border-bottom: 1px solid var(--workspace-divider); }
  .advanced-body { padding: 16px; }

  .form-actions {
    flex: 0 0 auto;
    display: grid;
    min-height: 58px;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    gap: 12px;
    border-top: 1px solid var(--workspace-divider);
    padding: 10px 16px;
  }
  .form-actions > .secondary-button { justify-self: start; }
  .action-hint { color: var(--app-subtle); font-size: 11px; line-height: 1.4; text-align: center; }
  .action-right { display: flex; justify-self: end; align-items: center; gap: 8px; }

  .restore-panel {
    flex: 0 0 auto;
    max-height: min(40dvh, 22rem);
    overflow: auto;
    margin-bottom: 12px;
  }
  .restore-content { display: grid; gap: 12px; padding: 0 2px 2px; }
  .restore-copy strong { font-size: 13px; }
  .restore-copy p { margin-top: 3px; color: var(--app-muted); font-size: 12px; line-height: 1.45; }
  .backup-file-input { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; clip-path: inset(50%); }
  .backup-dropzone {
    min-height: 104px;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    border: 1px dashed var(--control-border);
    border-radius: 6px;
    background: transparent;
    color: var(--app-muted);
    padding: 16px;
    text-align: left;
    transition: border-color .14s ease, color .14s ease, box-shadow .14s ease;
  }
  .backup-dropzone:hover, .backup-dropzone[data-dragging="true"] { border-color: var(--app-border-strong); color: var(--app-ink); }
  .backup-dropzone svg { width: 18px; height: 18px; flex: 0 0 18px; }
  .backup-dropzone strong { display: block; color: var(--app-ink); font-size: 13px; }
  .backup-dropzone span { display: block; margin-top: 2px; color: var(--app-subtle); font-size: 12px; }
  .backup-file-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 10px; min-height: 38px; }
  .backup-file-meta { min-width: 0; }
  .backup-file-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--app-ink); font-size: 12px; font-weight: 600; }
  .backup-file-size { margin-top: 2px; color: var(--app-subtle); font-size: 11px; }
  .backup-file-clear { border: 0; background: transparent; color: var(--app-muted); padding: 6px 8px; font-size: 12px; font-weight: 600; }
  .backup-file-clear:hover { color: var(--app-ink); }
  .backup-actions { display: flex; min-width: 0; align-items: center; justify-content: space-between; gap: 12px; }
  .restore-status { min-width: 0; color: var(--app-subtle); font-size: 12px; line-height: 1.45; }

  @media (max-width: 960px) {
    .app-shell { height: auto; min-height: 100dvh; overflow: visible; }
    .workspace { height: auto; min-height: calc(100dvh - 56px); grid-template-columns: 1fr; overflow: visible; }
    .setup-rail { overflow: visible; border-right: 0; border-bottom: 1px solid var(--workspace-divider); padding: 8px 14px; }
    .rail-group-label { display: none; }
    .stepper { grid-template-columns: repeat(4, minmax(9rem, 1fr)); margin-top: 0; overflow-x: auto; }
    .step-tab { border-left: 0; border-bottom: 2px solid transparent; }
    .step-tab.active { border-bottom-color: var(--app-ink); }
    .setup-main { min-height: calc(100dvh - 104px); overflow: visible; padding-top: 10px; }
    .setup-form { min-height: 0; overflow: visible; }
    .wizard-step:not([hidden]) { overflow: visible; }
  }

  @media (max-width: 760px) {
    .step-grid { grid-template-columns: 1fr; }
    .step-primary { border-right: 0; border-bottom: 1px solid var(--workspace-divider); }
    .field-grid.two { grid-template-columns: 1fr; }
    .value-row { grid-template-columns: 7rem minmax(0, 1fr) auto; }
    .review-row { grid-template-columns: 1fr; gap: 4px; }
  }

  @media (max-width: 560px) {
    .app-header { padding-inline: 10px; }
    .installer-label { display: none; }
    .setup-main { padding: 8px 10px; }
    .stepper { grid-template-columns: repeat(4, minmax(7.5rem, 1fr)); }
    .page-heading { padding-inline: 14px; }
    .step-primary, .step-aside, .advanced-body { padding: 14px; }
    .form-actions { grid-template-columns: 1fr; align-items: stretch; }
    .form-actions > .secondary-button, .action-right { width: 100%; justify-self: stretch; }
    .action-hint { grid-row: 1; }
    .action-right button { width: 100%; }
    .backup-actions { align-items: stretch; flex-direction: column; }
    .backup-actions .secondary-button { width: 100%; }
  }

  @media (any-pointer: coarse) {
    .icon-button, .ghost-button, .secondary-button, .primary-button, .field-input { min-height: 44px; }
    .secret-toggle { height: 44px; }
  }
"""


BASE_SCRIPT_JS = r"""
  <script>
    const steps = Array.from(document.querySelectorAll('.wizard-step'));
    const progress = Array.from(document.querySelectorAll('[data-progress]'));
    const form = document.getElementById('setup-form');
    const wizardToken = form.querySelector('input[name="token"]').value;
    const heading = document.getElementById('step-heading');
    const description = document.getElementById('step-description');
    const stepPosition = document.getElementById('step-position');
    const actionHint = document.getElementById('action-hint');
    const backButton = document.getElementById('back-button');
    const nextButton = document.getElementById('next-button');
    const submitButton = document.getElementById('submit-button');
    const domain = document.getElementById('PUBLIC_DOMAIN');
    const owner = document.getElementById('OWNER_EMAIL');
    const callback = document.getElementById('GITHUB_CALLBACK_URL');
    const restoreToggle = document.getElementById('restore-toggle');
    const restorePanel = document.getElementById('restore-panel');
    const titles = [
      ['Domain', 'Choose the public hostname for the dashboard and project URLs.'],
      ['GitHub', 'Connect the owner account and one GitHub OAuth App.'],
      ['Cloudflare', 'Connect the tunnel and verify dashboard plus project routing.'],
      ['Review', 'Confirm the public configuration before installation continues.']
    ];
    let currentStep = 0;
    let maxVisitedStep = 0;
    let callbackTouched = callback.dataset.generated !== 'true' && Boolean(callback.value);

    function cleanDomain() {
      return domain.value.trim().replace(/^https?:\/\//i, '').replace(/\/.*$/, '').replace(/\.$/, '').toLowerCase();
    }

    function setText(id, value) {
      const node = document.getElementById(id);
      if (node) node.textContent = value;
    }

    function updateDerivedText() {
      const clean = cleanDomain();
      const display = clean || 'example.com';
      setText('domain-dashboard', `https://${display}`);
      setText('domain-projects', `*.${display}`);
      setText('github-homepage-example', `https://${display}`);
      setText('github-callback-example', `https://${display}/api/auth/github/callback`);
      setText('cf-root-example', display);
      setText('cf-wildcard-example', `*.${display}`);
      setText('review-dashboard', clean ? `https://${clean}` : '-');
      setText('review-project', clean ? `https://<project>.${clean}` : '-');
      setText('review-callback', callback.value || '-');
      setText('review-owner', owner.value || '-');
    }

    function syncStepNavigation() {
      progress.forEach((item, itemIndex) => {
        item.classList.toggle('active', itemIndex === currentStep);
        item.classList.toggle('done', itemIndex < currentStep);
        item.disabled = itemIndex > maxVisitedStep;
        item.setAttribute('aria-disabled', String(itemIndex > maxVisitedStep));
        if (itemIndex === currentStep) item.setAttribute('aria-current', 'step');
        else item.removeAttribute('aria-current');
      });
    }

    function showStep(index) {
      const target = Math.max(0, Math.min(index, steps.length - 1));
      currentStep = target;
      maxVisitedStep = Math.max(maxVisitedStep, target);
      steps.forEach((step, stepIndex) => step.hidden = stepIndex !== currentStep);
      syncStepNavigation();
      heading.textContent = titles[currentStep][0];
      description.textContent = titles[currentStep][1];
      stepPosition.textContent = `Step ${currentStep + 1} of ${steps.length}`;
      form.classList.remove('was-validated');
      backButton.hidden = currentStep === 0;
      nextButton.hidden = currentStep === steps.length - 1;
      submitButton.hidden = currentStep !== steps.length - 1;
      actionHint.textContent = currentStep === steps.length - 1
        ? 'Saving writes the production configuration and resumes the terminal installer.'
        : 'You can go back without losing the values entered here.';
      updateDerivedText();
    }

    progress.forEach((item) => {
      item.addEventListener('click', () => {
        const target = Number(item.dataset.progress);
        if (!Number.isInteger(target) || target > maxVisitedStep) return;
        showStep(target);
      });
    });

    function validateCurrentStep() {
      form.classList.add('was-validated');
      const invalid = Array.from(steps[currentStep].querySelectorAll('input[required]')).find((input) => !input.checkValidity());
      if (!invalid) return true;
      invalid.reportValidity();
      return false;
    }

    function copyNodeText(id, button) {
      const node = document.getElementById(id);
      const value = node?.textContent?.trim();
      if (!value || value === '-') return;
      navigator.clipboard.writeText(value).then(() => {
        const previous = button.textContent;
        button.textContent = 'Copied';
        setTimeout(() => button.textContent = previous, 1200);
      }).catch(() => {});
    }

    document.querySelectorAll('[data-copy-target]').forEach((button) => {
      button.addEventListener('click', () => copyNodeText(button.dataset.copyTarget, button));
    });

    document.querySelectorAll('[data-secret-target]').forEach((button) => {
      button.addEventListener('click', () => {
        const input = document.getElementById(button.dataset.secretTarget);
        if (!input) return;
        const reveal = input.type === 'password';
        input.type = reveal ? 'text' : 'password';
        button.dataset.revealed = String(reveal);
        button.setAttribute('aria-label', reveal ? 'Hide secret' : 'Reveal secret');
        button.title = reveal ? 'Hide secret' : 'Reveal secret';
      });
    });

    restoreToggle?.addEventListener('click', () => {
      const open = restorePanel.hidden;
      restorePanel.hidden = !open;
      restoreToggle.setAttribute('aria-expanded', String(open));
    });

    const uploadBackupBtn = document.getElementById('upload-backup-btn');
    const backupFile = document.getElementById('backup-file');
    const backupDropzone = document.getElementById('backup-dropzone');
    const backupStatus = document.getElementById('backup-status');
    const backupFileRow = document.getElementById('backup-file-row');
    const backupFileName = document.getElementById('backup-file-name');
    const backupFileSize = document.getElementById('backup-file-size');
    const backupFileClear = document.getElementById('backup-file-clear');
    let selectedBackup = null;

    function formatBytes(bytes) {
      if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
      const units = ['B', 'KB', 'MB', 'GB'];
      const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
      return `${(bytes / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
    }

    function setBackupStatus(message, state = '') {
      backupStatus.textContent = message;
      backupStatus.style.color = state === 'error'
        ? 'var(--app-danger)'
        : state === 'ok'
          ? 'var(--app-success)'
          : 'var(--app-subtle)';
    }

    function clearBackupSelection() {
      selectedBackup = null;
      backupFile.value = '';
      backupFileRow.hidden = true;
      uploadBackupBtn.disabled = true;
      setBackupStatus('');
    }

    function chooseBackup(file) {
      if (!file) return;
      if (!file.name.toLowerCase().endsWith('.tar.gz')) {
        clearBackupSelection();
        setBackupStatus('Choose a MyPaaS .tar.gz backup.', 'error');
        return;
      }
      selectedBackup = file;
      backupFileName.textContent = file.name;
      backupFileSize.textContent = formatBytes(file.size);
      backupFileRow.hidden = false;
      uploadBackupBtn.disabled = false;
      setBackupStatus('Ready to upload.');
    }

    backupDropzone?.addEventListener('click', () => backupFile.click());
    backupDropzone?.addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        backupFile.click();
      }
    });
    backupFile?.addEventListener('change', () => chooseBackup(backupFile.files?.[0]));
    backupFileClear?.addEventListener('click', clearBackupSelection);

    ['dragenter', 'dragover'].forEach((type) => backupDropzone?.addEventListener(type, (event) => {
      event.preventDefault();
      event.stopPropagation();
      backupDropzone.dataset.dragging = 'true';
    }));
    ['dragleave', 'drop'].forEach((type) => backupDropzone?.addEventListener(type, (event) => {
      event.preventDefault();
      event.stopPropagation();
      backupDropzone.dataset.dragging = 'false';
    }));
    backupDropzone?.addEventListener('drop', (event) => chooseBackup(event.dataTransfer?.files?.[0]));

    uploadBackupBtn?.addEventListener('click', async () => {
      if (!selectedBackup) {
        setBackupStatus('Choose a MyPaaS backup first.', 'error');
        return;
      }
      const file = selectedBackup;
      uploadBackupBtn.disabled = true;
      uploadBackupBtn.textContent = 'Uploading…';
      setBackupStatus('Validating backup…');
      try {
        const res = await fetch('/upload-backup', {
          method: 'POST',
          body: file,
          headers: {
            'Content-Type': 'application/octet-stream',
            'X-Wizard-Token': wizardToken
          }
        });
        if (res.ok) {
          const html = await res.text();
          document.open();
          document.write(html);
          document.close();
          return;
        }
        setBackupStatus('Backup rejected. Check that the archive was created by MyPaaS.', 'error');
      } catch {
        setBackupStatus('Upload failed. Try again.', 'error');
      } finally {
        uploadBackupBtn.disabled = !selectedBackup;
        uploadBackupBtn.textContent = 'Upload backup';
      }
    });

    backButton.addEventListener('click', () => showStep(currentStep - 1));
    nextButton.addEventListener('click', () => {
      if (validateCurrentStep()) showStep(currentStep + 1);
    });
    form.addEventListener('submit', (event) => {
      if (currentStep !== steps.length - 1) {
        event.preventDefault();
        if (validateCurrentStep()) showStep(currentStep + 1);
        return;
      }
      if (!validateCurrentStep()) {
        event.preventDefault();
        return;
      }
      submitButton.disabled = true;
      submitButton.textContent = 'Saving…';
      backButton.disabled = true;
      nextButton.disabled = true;
    });

    callback.addEventListener('input', () => callbackTouched = true);
    domain.addEventListener('input', () => {
      const clean = cleanDomain();
      if (!callbackTouched) callback.value = clean ? `https://${clean}/api/auth/github/callback` : '';
      updateDerivedText();
    });
    owner.addEventListener('input', updateDerivedText);
    callback.addEventListener('input', updateDerivedText);
    showStep(0);

    const themeToggle = document.getElementById('theme-toggle');
    const themeIconSun = document.getElementById('theme-icon-sun');
    const themeIconMoon = document.getElementById('theme-icon-moon');
    function updateThemeIcon() {
      const isDark = document.documentElement.classList.contains('dark') || (!document.documentElement.classList.contains('light') && window.matchMedia('(prefers-color-scheme: dark)').matches);
      themeIconSun.style.display = isDark ? 'block' : 'none';
      themeIconMoon.style.display = isDark ? 'none' : 'block';
    }
    themeToggle?.addEventListener('click', () => {
      const isDark = document.documentElement.classList.contains('dark') || (!document.documentElement.classList.contains('light') && window.matchMedia('(prefers-color-scheme: dark)').matches);
      document.documentElement.classList.toggle('dark', !isDark);
      document.documentElement.classList.toggle('light', isDark);
      localStorage.setItem('theme', isDark ? 'light' : 'dark');
      updateThemeIcon();
    });
    updateThemeIcon();
  </script>
"""


def _secret_input(base, name: str, label: str, value: str, *, hint: str = "") -> str:
    hint_html = f'<p class="field-hint">{base.esc(hint)}</p>' if hint else ""
    return f"""
      <div class="field-group">
        <label class="field-label" for="{base.esc(name)}">{base.esc(label)}</label>
        <div class="secret-wrap">
          <input class="field-input mono" id="{base.esc(name)}" name="{base.esc(name)}" required type="password" autocomplete="new-password" value="{base.esc(value)}">
          <button class="secret-toggle" type="button" data-secret-target="{base.esc(name)}" aria-label="Reveal secret" title="Reveal secret"></button>
        </div>
        {hint_html}
      </div>
"""


def _advanced_field(base, name: str, label: str, values: dict[str, str], *, secret: bool = False) -> str:
    if secret:
        return _secret_input(base, name, label, values.get(name, ""))
    return f"""
      <div class="field-group">
        <label class="field-label" for="{base.esc(name)}">{base.esc(label)}</label>
        <input class="field-input mono" id="{base.esc(name)}" name="{base.esc(name)}" required type="text" autocomplete="off" value="{base.esc(values.get(name, ""))}">
      </div>
"""


def _nav_icon(kind: str) -> str:
    icons = {
        "domain": '<svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 0 20"/><path d="M12 2a15.3 15.3 0 0 0 0 20"/></svg>',
        "github": '<svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M15 6H9a6 6 0 0 0-6 6v1a6 6 0 0 0 6 6h1"/><path d="M14 19h1a6 6 0 0 0 6-6v-1a6 6 0 0 0-6-6"/><path d="M8 12h8"/></svg>',
        "cloudflare": '<svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect width="6" height="6" x="9" y="9" rx="1"/><path d="M12 2v7"/><path d="M12 15v7"/><path d="m19 5-4.5 4.5"/><path d="m9.5 14.5-4.5 4.5"/><path d="M2 12h7"/><path d="M15 12h7"/></svg>',
        "review": '<svg viewBox="0 0 24 24" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>',
    }
    return icons[kind]


def render_form_html(base, error: str = "", values: dict[str, str] | None = None) -> bytes:
    """Render the dependency-free installer with the authenticated workspace grammar."""
    values = values or base.DEFAULTS
    domain = values.get("PUBLIC_DOMAIN", "")
    callback_is_generated = not values.get("GITHUB_CALLBACK_URL", "")
    callback = values.get("GITHUB_CALLBACK_URL", "") or (f"https://{domain}/api/auth/github/callback" if domain else "")
    error_html = f'<div class="error-banner" role="alert">{base.esc(error)}</div>' if error else ""

    advanced_fields = "".join(
        [
            _advanced_field(base, "POSTGRES_USER", "Postgres user", values),
            _advanced_field(base, "POSTGRES_DB", "Postgres database", values),
            _advanced_field(base, "POSTGRES_PASSWORD", "Postgres password", values, secret=True),
            _advanced_field(base, "PROJECT_NETWORK", "Project network", values),
            _advanced_field(base, "DOCKER_BIND_HOST", "Docker bind host", values),
            _advanced_field(base, "METRICS_PASSWORD", "Metrics password", values, secret=True),
            _advanced_field(base, "JWT_SECRET", "JWT secret", values, secret=True),
            _advanced_field(base, "ENCRYPTION_KEY", "Environment encryption key", values, secret=True),
        ]
    )

    body = f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MyPaaS Installer</title>
  <style>
    @font-face {{
      font-family: "Inter Variable";
      font-style: normal;
      font-display: swap;
      font-weight: 100 900;
      src: url("https://cdn.jsdelivr.net/fontsource/fonts/inter:vf@5.3.0/latin-wght-normal.woff2") format("woff2-variations");
    }}
{VISUAL_CONTRACT_CSS}
  </style>
</head>
<body>
  <script>
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {{
      document.documentElement.classList.add('dark');
    }} else if (savedTheme === 'light') {{
      document.documentElement.classList.add('light');
    }}
  </script>

  <div class="app-shell">
    <header class="app-header">
      <div class="brand-lockup">
        <img class="brand-logo" src="/brand/logo.svg" alt="MyPaaS">
        <span class="installer-label">Installer</span>
      </div>
      <div class="header-actions">
        <button type="button" id="restore-toggle" class="ghost-button" aria-expanded="false" aria-controls="restore-panel">Restore backup</button>
        <button type="button" id="theme-toggle" class="icon-button" aria-label="Toggle theme" title="Toggle theme">
          <svg id="theme-icon-sun" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
          <svg id="theme-icon-moon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>
        </button>
      </div>
    </header>

    <div class="workspace">
      <aside class="setup-rail" aria-label="Setup steps">
        <p class="rail-group-label">Setup</p>
        <nav class="stepper" aria-label="Install progress">
          <button type="button" class="step-tab active" data-progress="0" aria-current="step"><span class="step-icon">{_nav_icon("domain")}</span><span class="step-title">Domain</span></button>
          <button type="button" class="step-tab" data-progress="1" disabled aria-disabled="true"><span class="step-icon">{_nav_icon("github")}</span><span class="step-title">GitHub</span></button>
          <button type="button" class="step-tab" data-progress="2" disabled aria-disabled="true"><span class="step-icon">{_nav_icon("cloudflare")}</span><span class="step-title">Cloudflare</span></button>
          <button type="button" class="step-tab" data-progress="3" disabled aria-disabled="true"><span class="step-icon">{_nav_icon("review")}</span><span class="step-title">Review</span></button>
        </nav>
      </aside>

      <main class="setup-main">
        <section id="restore-panel" class="restore-panel" hidden>
          <div class="restore-content">
            <div class="restore-copy">
              <strong>Restore an existing MyPaaS instance</strong>
              <p>Upload a MyPaaS <span class="mono">.tar.gz</span> backup instead of creating a fresh configuration.</p>
            </div>
            <input class="backup-file-input" type="file" id="backup-file" accept=".tar.gz,application/gzip">
            <div id="backup-dropzone" class="backup-dropzone" role="button" tabindex="0" aria-controls="backup-file" data-dragging="false">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12"/><path d="m7 8 5-5 5 5"/><path d="M5 13v6h14v-6"/></svg>
              <div><strong>Drop backup here or choose a file</strong><span>MyPaaS .tar.gz backups only</span></div>
            </div>
            <div id="backup-file-row" class="backup-file-row" hidden>
              <div class="backup-file-meta"><p id="backup-file-name" class="backup-file-name"></p><p id="backup-file-size" class="backup-file-size"></p></div>
              <button type="button" id="backup-file-clear" class="backup-file-clear">Remove</button>
            </div>
            <div class="backup-actions">
              <div id="backup-status" class="restore-status" aria-live="polite"></div>
              <button type="button" id="upload-backup-btn" class="secondary-button" disabled>Upload backup</button>
            </div>
          </div>
        </section>

        <form id="setup-form" class="setup-form" method="post" action="/save" aria-labelledby="step-heading">
          <input type="hidden" name="token" value="{base.esc(base.TOKEN)}">
          <div class="page-heading">
            <div>
              <h2 id="step-heading">Domain</h2>
              <p id="step-description">Choose the public hostname for the dashboard and project URLs.</p>
            </div>
            <span class="step-position" id="step-position">Step 1 of 4</span>
          </div>
          {error_html}

          <section class="wizard-step" data-step="0">
            <div class="step-grid">
              <div class="step-primary">
                <div class="section-heading"><h3>Public domain</h3><p>Use the root domain. MyPaaS creates project hostnames directly below it.</p></div>
                <div class="field-group" style="max-width: 44rem;">
                  <label class="field-label" for="PUBLIC_DOMAIN">Domain</label>
                  <input class="field-input mono" id="PUBLIC_DOMAIN" name="PUBLIC_DOMAIN" required inputmode="url" autocomplete="off" placeholder="example.com" value="{base.esc(domain)}">
                  <p class="field-hint">Hostname only, without <span class="mono">https://</span>.</p>
                </div>
              </div>
              <aside class="step-aside">
                <div class="provider-header"><div><strong>Public URLs</strong><p>Derived from the domain.</p></div><a class="docs-link" href="https://developers.cloudflare.com/dns/zone-setups/reference/domain-status/" target="_blank" rel="noopener noreferrer">Cloudflare DNS docs ↗</a></div>
                <div class="value-list">
                  <div class="value-row"><span>Dashboard</span><span class="value mono" id="domain-dashboard">https://example.com</span></div>
                  <div class="value-row"><span>Projects</span><span class="value mono" id="domain-projects">*.example.com</span></div>
                </div>
                <div class="validation-box"><p class="validation-title">Domain check</p><p class="validation-copy">Resolve the domain from this machine. Wildcard routing is verified after the tunnel is configured.</p><div class="preflight-row"><button type="button" class="secondary-button" id="check-domain-button">Check domain</button><span class="preflight-status" id="domain-preflight-status" aria-live="polite"></span></div></div>
              </aside>
            </div>
          </section>

          <section class="wizard-step" data-step="1" hidden>
            <div class="step-grid">
              <div class="step-primary">
                <div class="section-heading"><h3>Owner and OAuth credentials</h3><p>Use the verified primary email on the GitHub account that will own this instance.</p></div>
                <div class="field-grid" style="max-width: 52rem;">
                  <div class="field-group"><label class="field-label" for="OWNER_EMAIL">Owner GitHub email</label><input class="field-input" id="OWNER_EMAIL" name="OWNER_EMAIL" required type="email" autocomplete="email" placeholder="you@example.com" value="{base.esc(values.get("OWNER_EMAIL", ""))}"><p class="field-hint">Verified during the first GitHub sign-in. After binding, MyPaaS identifies the owner by GitHub numeric user ID.</p></div>
                  <div class="field-grid two"><div class="field-group"><label class="field-label" for="GITHUB_CLIENT_ID">OAuth Client ID</label><input class="field-input mono" id="GITHUB_CLIENT_ID" name="GITHUB_CLIENT_ID" required autocomplete="off" spellcheck="false" value="{base.esc(values.get("GITHUB_CLIENT_ID", ""))}"></div>{_secret_input(base, "GITHUB_CLIENT_SECRET", "OAuth Client Secret", values.get("GITHUB_CLIENT_SECRET", ""))}</div>
                  <div class="field-group"><label class="field-label" for="GITHUB_CALLBACK_URL">Callback URL</label><input class="field-input mono" id="GITHUB_CALLBACK_URL" name="GITHUB_CALLBACK_URL" required type="url" autocomplete="off" data-generated="{str(callback_is_generated).lower()}" value="{base.esc(callback)}"><p class="field-hint">Must match the GitHub OAuth App callback exactly.</p></div>
                </div>
              </div>
              <aside class="step-aside">
                <div class="provider-header"><div><strong>GitHub OAuth App</strong><p>Create one OAuth App and paste its credentials on the left.</p></div><a class="docs-link" href="https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app" target="_blank" rel="noopener noreferrer">GitHub docs ↗</a></div>
                <div class="value-list">
                  <div class="value-row"><span>App name</span><span class="value">MyPaaS</span></div>
                  <div class="value-row"><span>Homepage URL</span><span class="value mono" id="github-homepage-example">https://example.com</span><button class="copy-button" type="button" data-copy-target="github-homepage-example">Copy</button></div>
                  <div class="value-row"><span>Callback URL</span><span class="value mono" id="github-callback-example">https://example.com/api/auth/github/callback</span><button class="copy-button" type="button" data-copy-target="github-callback-example">Copy</button></div>
                </div>
                <div class="validation-box"><p class="validation-title">OAuth check</p><p class="validation-copy">Checks Client ID, Client Secret, and callback format. Owner identity is verified during sign-in.</p><div class="preflight-row"><button type="button" class="secondary-button" id="check-github-button">Test GitHub</button><span class="preflight-status" id="github-preflight-status" aria-live="polite"></span></div></div>
              </aside>
            </div>
          </section>

          <section class="wizard-step" data-step="2" hidden>
            <div class="step-grid">
              <div class="step-primary"><div class="section-heading"><h3>Cloudflare Tunnel</h3><p>Paste the Tunnel token or the full command copied from Add a replica. MyPaaS extracts the <span class="mono">eyJ…</span> token automatically.</p></div><div style="max-width: 52rem;">{_secret_input(base, "CLOUDFLARE_TUNNEL_TOKEN", "Tunnel token or Add-a-replica command", values.get("CLOUDFLARE_TUNNEL_TOKEN", ""), hint="Use a Cloudflare Tunnel token, not a Cloudflare API token.")}</div></div>
              <aside class="step-aside">
                <div class="provider-header"><div><strong>Published routes</strong><p>Create the dashboard and wildcard routes in the same tunnel.</p></div><a class="docs-link" href="https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/" target="_blank" rel="noopener noreferrer">Cloudflare Tunnel docs ↗</a></div>
                <div class="value-list"><div class="value-row"><span>Dashboard</span><span class="value mono" id="cf-root-example">example.com</span></div><div class="value-row"><span>Projects</span><span class="value mono" id="cf-wildcard-example">*.example.com</span></div><div class="value-row"><span>Service</span><span class="value mono">http://caddy:80</span></div></div>
                <div class="validation-box"><p class="validation-title">Tunnel and routing check</p><p class="validation-copy">Briefly connects with the token, then confirms project wildcard DNS from this machine.</p><div class="preflight-row"><button type="button" class="secondary-button" id="check-cloudflare-button">Test tunnel</button><span class="preflight-status" id="cloudflare-preflight-status" aria-live="polite"></span></div></div>
              </aside>
            </div>
          </section>

          <section class="wizard-step" data-step="3" hidden>
            <div class="step-grid">
              <div class="step-primary"><div class="section-heading"><h3>Installation summary</h3><p>Secrets stay hidden. Saving writes the production configuration and continues the terminal installer.</p></div><dl class="review-list"><div class="review-row"><dt>Dashboard</dt><dd class="mono" id="review-dashboard">-</dd></div><div class="review-row"><dt>Project URLs</dt><dd class="mono" id="review-project">-</dd></div><div class="review-row"><dt>Owner</dt><dd id="review-owner">-</dd></div><div class="review-row"><dt>GitHub callback</dt><dd class="mono" id="review-callback">-</dd></div><div class="review-row"><dt>GitHub secret</dt><dd>Configured</dd></div><div class="review-row"><dt>Cloudflare Tunnel</dt><dd>Configured</dd></div></dl></div>
              <aside class="step-aside"><div class="provider-header"><div><strong>Ready to install</strong><p>Domain, GitHub, and Cloudflare checks must match the values being saved.</p></div></div><div class="validation-box"><p class="validation-title">What happens next</p><p class="validation-copy">MyPaaS writes the configuration, closes this wizard, then the terminal continues with startup and migrations.</p></div></aside>
            </div>
            <details class="advanced"><summary>Advanced generated values</summary><div class="advanced-body field-grid two">{advanced_fields}</div></details>
          </section>

          <div class="form-actions">
            <button class="secondary-button" type="button" id="back-button" hidden>Back</button>
            <span class="action-hint" id="action-hint">You can go back without losing the values entered here.</span>
            <div class="action-right"><button class="primary-button" type="button" id="next-button">Continue</button><button class="primary-button" type="submit" id="submit-button" data-default-label="Install MyPaaS" hidden>Install MyPaaS</button></div>
          </div>
        </form>
      </main>
    </div>
  </div>
{BASE_SCRIPT_JS}
</body>
</html>"""
    return body.encode("utf-8")
