from __future__ import annotations


VISUAL_CONTRACT_CSS = r"""
  /* Installer visual contract mirrors frontend/src/app.css and frontend/DESIGN.md. */
  :root {
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
    --control-border: color-mix(in oklch, var(--app-border) 68%, transparent);
    --control-border-hover: color-mix(in oklch, var(--app-border-strong) 88%, var(--app-border));
    --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 58%, transparent);
  }

  :root.dark {
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
    --control-border: color-mix(in oklch, var(--app-border) 62%, transparent);
    --control-border-hover: color-mix(in oklch, var(--app-border-strong) 86%, var(--app-border));
    --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 62%, transparent);
  }

  @media (prefers-color-scheme: dark) {
    :root:not(.light) {
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
      --control-border: color-mix(in oklch, var(--app-border) 62%, transparent);
      --control-border-hover: color-mix(in oklch, var(--app-border-strong) 86%, var(--app-border));
      --control-focus-ring: color-mix(in oklch, var(--app-accent-soft) 62%, transparent);
    }
  }

  body { background: var(--app-bg); }
  main { width: min(100%, 1180px); margin: 0 auto; padding: 0 14px 28px; }
  header { min-height: 56px; margin: 0; padding: 10px 0; border-bottom: 1px solid var(--workspace-divider); }
  .topline { min-height: 35px; }
  .brand-logo { width: 108px; }
  .installer-badge,
  .meta-chip,
  .step-count {
    min-height: 26px;
    border-color: var(--workspace-divider);
    border-radius: 5px;
    background: transparent;
    padding: 3px 7px;
  }
  .header-copy { margin-top: 12px; max-width: none; padding-bottom: 14px; }
  .header-copy h1 { font-size: 20px; line-height: 1.25; letter-spacing: -.015em; }
  .header-copy p { max-width: 72ch; font-size: 13px; line-height: 1.5; }

  .layout {
    display: grid;
    grid-template-columns: 12rem minmax(0, 1fr);
    gap: 0;
    align-items: stretch;
    border-bottom: 1px solid var(--workspace-divider);
  }
  .panel { border: 0; border-radius: 0; background: var(--app-surface); }
  form.panel { min-width: 0; border-left: 1px solid var(--workspace-divider); }
  .stepper {
    position: sticky;
    top: 0;
    align-self: start;
    display: grid;
    gap: 0;
    padding: 8px 8px 8px 0;
    background: var(--app-surface);
  }
  .step-tab {
    grid-template-columns: 24px minmax(0, 1fr);
    gap: 9px;
    min-height: 48px;
    border: 0;
    border-left: 2px solid transparent;
    border-radius: 0;
    padding: 7px 9px;
    background: transparent;
  }
  .step-tab.active { border-color: var(--app-ink); background: transparent; }
  .step-tab.done { border-color: var(--workspace-divider); }
  .step-number {
    width: 24px;
    height: 24px;
    border: 1px solid var(--workspace-divider);
    border-radius: 5px;
    background: transparent;
    font-size: 11px;
  }
  .step-tab.active .step-number { border-color: var(--app-ink); background: var(--app-ink); color: var(--app-surface); }
  .step-tab.done .step-number { background: transparent; color: var(--app-muted); }
  .step-title { font-size: 13px; font-weight: 650; }
  .step-body { margin-top: 1px; font-size: 11px; line-height: 1.35; }

  .panel-header {
    min-height: 62px;
    align-items: center;
    padding: 11px 16px;
    border-bottom: 1px solid var(--workspace-divider);
    background: var(--app-surface);
  }
  .panel-title h2 { margin-bottom: 2px; font-size: 15px; }
  .panel-title p { font-size: 12px; }
  .panel-body { padding: 0; }
  .wizard-step { padding: 14px 16px 16px; }

  .guide { gap: 0; margin: -14px -16px 14px; }
  .guide-card {
    border: 0;
    border-bottom: 1px solid var(--workspace-divider);
    border-radius: 0;
    background: transparent;
    padding: 12px 16px;
  }
  .guide-card strong { margin-bottom: 3px; font-size: 13px; }
  .guide-card p,
  .guide-card li { font-size: 12px; line-height: 1.5; }
  .guide-card ol { margin-top: 7px; }
  .guide-card li + li { margin-top: 4px; }
  .notice,
  .warning,
  .alert {
    margin: 0;
    border: 0;
    border-bottom: 1px solid var(--workspace-divider);
    border-radius: 0;
    padding: 9px 16px;
    background: transparent;
    font-size: 12px;
    line-height: 1.5;
  }
  .notice { color: var(--app-info); }
  .warning { color: var(--app-warning); }
  .alert { color: var(--app-danger); }

  .grid { gap: 12px 14px; }
  .field { gap: 5px; }
  label { font-size: 12px; font-weight: 600; }
  input {
    min-height: 36px;
    border-color: var(--control-border);
    border-radius: 5px;
    padding: 6px 9px;
    background: var(--control-bg);
    font-size: 14px;
  }
  input:hover { border-color: var(--control-border-hover); }
  input:focus { border-color: var(--app-ink); box-shadow: 0 0 0 3px var(--control-focus-ring); }
  .hint { font-size: 11px; line-height: 1.45; }

  button {
    min-height: 36px;
    border-radius: 5px;
    padding: 0 12px;
    font-size: 13px;
  }
  button.secondary { border-color: var(--control-border); background: transparent; }
  button.secondary:hover { border-color: var(--control-border-hover); background: transparent; }
  .theme-toggle { width: 32px; min-height: 32px; border-color: var(--control-border); background: transparent; }
  .theme-toggle:hover { border-color: var(--control-border-hover); background: transparent; }

  .example-grid { gap: 0; margin: 8px 0 0; border-top: 1px solid var(--workspace-divider); }
  .example-row {
    grid-template-columns: 8.5rem minmax(0, 1fr);
    gap: 10px;
    min-height: 34px;
    padding: 6px 0;
    border-bottom: 1px solid var(--workspace-divider);
    font-size: 12px;
  }
  code { border-color: var(--workspace-divider); background: transparent; }

  .review { gap: 0; margin-top: 8px; border-top: 1px solid var(--workspace-divider); }
  .review-row {
    grid-template-columns: 11rem minmax(0, 1fr);
    gap: 12px;
    min-height: 38px;
    align-items: center;
    margin: 0;
    padding: 7px 0;
    border-bottom: 1px solid var(--workspace-divider);
    font-size: 12px;
  }

  details { margin: 0 -16px -16px; border-top: 1px solid var(--workspace-divider); }
  summary { padding: 10px 16px; font-size: 12px; }
  details .panel-body { padding: 14px 16px 16px; }
  .preflight-row { margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--workspace-divider); }
  .preflight-status[data-state="ok"] { color: var(--app-success); }

  .actions {
    min-height: 56px;
    padding: 10px 16px;
    border-top: 1px solid var(--workspace-divider);
    background: var(--app-surface);
  }
  .action-hint { font-size: 11px; }

  @media (max-width: 900px) {
    main { padding: 0 12px 24px; }
    .layout { grid-template-columns: 1fr; }
    form.panel { border-left: 0; border-top: 1px solid var(--workspace-divider); }
    .stepper {
      position: static;
      grid-template-columns: repeat(5, minmax(150px, 1fr));
      overflow-x: auto;
      padding: 7px 0;
    }
    .step-tab { border-left: 0; border-bottom: 2px solid transparent; }
    .step-tab.active { border-bottom-color: var(--app-ink); }
    .step-tab.done { border-bottom-color: var(--workspace-divider); }
  }

  @media (max-width: 620px) {
    .install-meta .meta-chip { display: none; }
    .header-copy h1 { font-size: 18px; }
    .stepper { grid-template-columns: repeat(5, minmax(132px, 1fr)); }
    .panel-header { align-items: flex-start; gap: 7px; }
    .wizard-step { padding: 12px; }
    .guide { margin: -12px -12px 12px; }
    .guide-card, .notice, .warning, .alert { padding-inline: 12px; }
    details { margin: 0 -12px -12px; }
    .actions { align-items: stretch; }
  }
"""


COPY_REPLACEMENTS = {
    "<h1>Set up MyPaas</h1>": "<h1>Set up MyPaaS</h1>",
    "Complete five short steps. Nothing is saved until the final review, and the terminal installer continues automatically when you finish.": (
        "Connect the domain, GitHub sign-in, and Cloudflare routing. Configuration is written only after review."
    ),
    "<span><span class=\"step-title\">Restore</span><span class=\"step-body\">Upload backup</span></span>": (
        "<span><span class=\"step-title\">Restore</span><span class=\"step-body\">Optional backup</span></span>"
    ),
    "<span><span class=\"step-title\">Domain</span><span class=\"step-body\">Where MyPaas lives</span></span>": (
        "<span><span class=\"step-title\">Domain</span><span class=\"step-body\">Public URLs</span></span>"
    ),
    "<span><span class=\"step-title\">GitHub login</span><span class=\"step-body\">Owner sign-in</span></span>": (
        "<span><span class=\"step-title\">GitHub</span><span class=\"step-body\">OAuth app</span></span>"
    ),
    "<span><span class=\"step-title\">Routing</span><span class=\"step-body\">Cloudflare tunnel</span></span>": (
        "<span><span class=\"step-title\">Routing</span><span class=\"step-body\">Tunnel + wildcard</span></span>"
    ),
    "<span><span class=\"step-title\">Review</span><span class=\"step-body\">Confirm settings</span></span>": (
        "<span><span class=\"step-title\">Review</span><span class=\"step-body\">Write configuration</span></span>"
    ),
    "<strong>Restore from Backup (Optional)</strong>": "<strong>Restore an existing instance</strong>",
    "If you are migrating or recovering an existing MyPaas instance, you can upload a database backup file": (
        "Migrating or recovering MyPaaS? Upload a database backup file"
    ),
    "<strong>Use a domain that is already active in Cloudflare.</strong>": "<strong>Choose the public MyPaaS hostname</strong>",
    "Enter only the hostname. MyPaas uses it for the dashboard and automatically places projects below it.": (
        "Enter only the hostname. MyPaaS serves the dashboard here and project subdomains below it."
    ),
    "<strong>Before continuing</strong>": "<strong>Required before deployment</strong>",
    "You know the primary email of the GitHub account that will own MyPaas.": (
        "The owner GitHub account has a verified primary email."
    ),
    "<label for=\"OWNER_EMAIL\">Owner GitHub primary email</label>": (
        "<label for=\"OWNER_EMAIL\">Owner verified primary GitHub email</label>"
    ),
    "Only this whitelisted email can log in as the first owner.": (
        "Used for the first login only. After binding, MyPaaS identifies this account by GitHub numeric user ID."
    ),
    "<strong>Create one GitHub OAuth App.</strong>": "<strong>Connect one GitHub OAuth App</strong>",
    "<strong>1. Get the tunnel token</strong>": "<strong>1. Copy the tunnel connector</strong>",
    "Select the <strong>Docker</strong> connector and copy only the value after <code>--token</code>.": (
        "Select the <strong>Docker</strong> connector. You may paste the token or the full Add-a-replica command below."
    ),
    "<span class=\"hint\">Use a Cloudflare Zero Trust tunnel token, not an API token.</span>": (
        "<span class=\"hint\">Paste the Tunnel token or full Add-a-replica command. Cloudflare API tokens are not accepted.</span>"
    ),
    "<strong>Confirm the values below.</strong>": "<strong>Review before writing configuration</strong>",
}


def apply_visual_contract(document: str) -> str:
    """Apply the standalone installer's compact MyPaaS workspace grammar."""
    if "--workspace-divider" in document and "Installer visual contract mirrors" in document:
        return document
    marker = "  </style>\n</head>"
    if marker not in document:
        raise RuntimeError("install wizard style boundary was not found")
    document = document.replace(marker, VISUAL_CONTRACT_CSS + marker, 1)
    for source, target in COPY_REPLACEMENTS.items():
        document = document.replace(source, target, 1)
    return document
