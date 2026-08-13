#!/usr/bin/env python3
import html
import os
import re
import secrets
import stat
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import parse_qs, urlparse


HOST = os.environ.get("WIZARD_HOST", "127.0.0.1")
PORT = int(os.environ.get("WIZARD_PORT", "8787"))
TOKEN = os.environ.get("WIZARD_TOKEN", secrets.token_hex(16))
ENV_FILE = os.environ.get("WIZARD_ENV_FILE", ".env")
ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
BRAND_LOGO_PATH = os.path.join(ROOT_DIR, "frontend", "src", "assets", "new-assets", "logowithtext_black.png")


def default(name: str, fallback: str = "") -> str:
    return os.environ.get(f"WIZARD_DEFAULT_{name}", fallback)


DEFAULTS = {
    "PUBLIC_DOMAIN": default("PUBLIC_DOMAIN"),
    "OWNER_EMAIL": default("OWNER_EMAIL"),
    "GITHUB_CLIENT_ID": default("GITHUB_CLIENT_ID"),
    "GITHUB_CLIENT_SECRET": default("GITHUB_CLIENT_SECRET"),
    "GITHUB_CALLBACK_URL": default("GITHUB_CALLBACK_URL"),
    "CLOUDFLARE_TUNNEL_TOKEN": default("CLOUDFLARE_TUNNEL_TOKEN"),
    "POSTGRES_USER": default("POSTGRES_USER", "mypaas"),
    "POSTGRES_DB": default("POSTGRES_DB", "mypaas"),
    "POSTGRES_PASSWORD": default("POSTGRES_PASSWORD", secrets.token_hex(24)),
    "JWT_SECRET": default("JWT_SECRET", secrets.token_urlsafe(32)),
    "ENCRYPTION_KEY": default("ENCRYPTION_KEY", secrets.token_urlsafe(32)),
    "METRICS_PASSWORD": default("METRICS_PASSWORD", secrets.token_hex(18)),
    "PROJECT_NETWORK": default("PROJECT_NETWORK", "mypaas-prod"),
    "DOCKER_BIND_HOST": default("DOCKER_BIND_HOST", "172.17.0.1"),
}

RE_DOMAIN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9.-]{0,251}[A-Za-z0-9]$")
RE_EMAIL = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")
RE_URL_SAFE = re.compile(r"^[A-Za-z0-9._~-]+$")


def esc(value: str) -> str:
    return html.escape(value or "", quote=True)


def field(data: dict[str, list[str]], name: str) -> str:
    return data.get(name, [""])[0].strip()


def normalize_domain(value: str) -> str:
    value = value.strip()
    value = re.sub(r"^https?://", "", value, flags=re.IGNORECASE)
    value = value.split("/", 1)[0].strip().rstrip(".")
    return value.lower()


def safe_env_value(name: str, value: str) -> str:
    value = value.strip()
    if "\n" in value or "\r" in value or "\0" in value:
        raise ValueError(f"{name} cannot contain new lines")
    if re.search(r"\s", value):
        raise ValueError(f"{name} cannot contain whitespace")
    return value


def build_env(values: dict[str, str]) -> str:
    public_domain = normalize_domain(values["PUBLIC_DOMAIN"])
    if not RE_DOMAIN.match(public_domain):
        raise ValueError("PUBLIC_DOMAIN must be a hostname like mypaas.example.com")

    owner_email = values["OWNER_EMAIL"].strip().lower()
    if not RE_EMAIL.match(owner_email):
        raise ValueError("OWNER_EMAIL must be a valid GitHub primary email")

    github_callback = values["GITHUB_CALLBACK_URL"].strip() or f"https://{public_domain}/api/auth/github/callback"
    if not github_callback.startswith("https://"):
        raise ValueError("GITHUB_CALLBACK_URL must start with https://")

    postgres_password = safe_env_value("POSTGRES_PASSWORD", values["POSTGRES_PASSWORD"])
    if not RE_URL_SAFE.match(postgres_password):
        raise ValueError("POSTGRES_PASSWORD can only use A-Z, a-z, 0-9, '.', '_', '~', or '-'")

    clean = {
        "PUBLIC_DOMAIN": public_domain,
        "OWNER_EMAIL": owner_email,
        "GITHUB_CLIENT_ID": safe_env_value("GITHUB_CLIENT_ID", values["GITHUB_CLIENT_ID"]),
        "GITHUB_CLIENT_SECRET": safe_env_value("GITHUB_CLIENT_SECRET", values["GITHUB_CLIENT_SECRET"]),
        "GITHUB_CALLBACK_URL": safe_env_value("GITHUB_CALLBACK_URL", github_callback),
        "CLOUDFLARE_TUNNEL_TOKEN": safe_env_value("CLOUDFLARE_TUNNEL_TOKEN", values["CLOUDFLARE_TUNNEL_TOKEN"]),
        "POSTGRES_USER": safe_env_value("POSTGRES_USER", values["POSTGRES_USER"]),
        "POSTGRES_DB": safe_env_value("POSTGRES_DB", values["POSTGRES_DB"]),
        "POSTGRES_PASSWORD": postgres_password,
        "JWT_SECRET": safe_env_value("JWT_SECRET", values["JWT_SECRET"]),
        "ENCRYPTION_KEY": safe_env_value("ENCRYPTION_KEY", values["ENCRYPTION_KEY"]),
        "METRICS_PASSWORD": safe_env_value("METRICS_PASSWORD", values["METRICS_PASSWORD"]),
        "PROJECT_NETWORK": safe_env_value("PROJECT_NETWORK", values["PROJECT_NETWORK"]),
        "DOCKER_BIND_HOST": safe_env_value("DOCKER_BIND_HOST", values["DOCKER_BIND_HOST"]),
    }

    missing = [
        key
        for key in (
            "PUBLIC_DOMAIN",
            "OWNER_EMAIL",
            "GITHUB_CLIENT_ID",
            "GITHUB_CLIENT_SECRET",
            "CLOUDFLARE_TUNNEL_TOKEN",
        )
        if not clean[key]
    ]
    if missing:
        raise ValueError("Missing required fields: " + ", ".join(missing))

    return f"""ENVIRONMENT=production

POSTGRES_USER={clean["POSTGRES_USER"]}
POSTGRES_PASSWORD={clean["POSTGRES_PASSWORD"]}
POSTGRES_DB={clean["POSTGRES_DB"]}

API_HOST=127.0.0.1
API_PORT=8080
FRONTEND_URL=https://{clean["PUBLIC_DOMAIN"]}
PUBLIC_DOMAIN={clean["PUBLIC_DOMAIN"]}
OWNER_EMAIL={clean["OWNER_EMAIL"]}

GITHUB_CLIENT_ID={clean["GITHUB_CLIENT_ID"]}
GITHUB_CLIENT_SECRET={clean["GITHUB_CLIENT_SECRET"]}
GITHUB_CALLBACK_URL={clean["GITHUB_CALLBACK_URL"]}

CLOUDFLARE_TUNNEL_TOKEN={clean["CLOUDFLARE_TUNNEL_TOKEN"]}

JWT_SECRET={clean["JWT_SECRET"]}
ENCRYPTION_KEY={clean["ENCRYPTION_KEY"]}

DOCKER_SOCKET=/var/run/docker.sock
DOCKER_HOST=
DOCKER_BIND_HOST={clean["DOCKER_BIND_HOST"]}
PROJECT_NETWORK={clean["PROJECT_NETWORK"]}

USER_RAM_QUOTA_GB=6
USER_CPU_QUOTA=3
MAX_PROJECTS=20
PROJECT_DEFAULT_RAM_MB=512
PROJECT_DEFAULT_CPU=0.5

ENABLE_WEBHOOKS=true
ENABLE_METRICS=true
METRICS_USERNAME=mypaas
METRICS_PASSWORD={clean["METRICS_PASSWORD"]}
MAX_CONCURRENT_DEPLOYS=2
BUILD_TIMEOUT_MINUTES=30

SHARED_POSTGRES_ENABLED=true
SHARED_POSTGRES_HOST=postgres
SHARED_POSTGRES_PORT=5432
SHARED_POSTGRES_SSLMODE=disable

BACKUP_ENABLED=true
BACKUP_DIR=/var/lib/mypaas/backups
BACKUP_DAILY_AT=02:00
BACKUP_TIMEOUT_MINUTES=30
BACKUP_KEEP_DAILY=7
BACKUP_KEEP_WEEKLY=4
BACKUP_WEEKLY_DAY=sunday

IMAGE_CLEANUP_ENABLED=true
IMAGE_CLEANUP_UNTIL=168h
IMAGE_CLEANUP_WEEKDAY=sunday

LOG_LEVEL=info
LOG_FORMAT=json

CADDY_ADMIN=127.0.0.1:2019
CADDY_UPSTREAM_HOST={clean["DOCKER_BIND_HOST"]}
STATIC_ROOT=/var/lib/mypaas/static
CADDY_STATIC_ROOT=/var/lib/mypaas/static
CADDY_METRICS=true
STATD_SOCKET=/run/mypaas/statd.sock
"""


def write_env(content: str) -> None:
    directory = os.path.dirname(os.path.abspath(ENV_FILE))
    if directory:
        os.makedirs(directory, exist_ok=True)
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC
    fd = os.open(ENV_FILE, flags, stat.S_IRUSR | stat.S_IWUSR)
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        handle.write(content)


def form_html(error: str = "", values: dict[str, str] | None = None) -> bytes:
    values = values or DEFAULTS
    domain = values.get("PUBLIC_DOMAIN", "")
    callback_is_generated = not values.get("GITHUB_CALLBACK_URL", "")
    callback = values.get("GITHUB_CALLBACK_URL", "") or (f"https://{domain}/api/auth/github/callback" if domain else "")
    body = f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MyPaas Install Wizard</title>
  <style>
    @font-face {{
      font-family: "Inter Variable";
      font-style: normal;
      font-display: swap;
      font-weight: 100 900;
      src: url("https://cdn.jsdelivr.net/fontsource/fonts/inter:vf@5.3.0/latin-wght-normal.woff2") format("woff2-variations");
    }}
    :root {{
    color-scheme: light dark;
    font-family: "Inter Variable", "Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    --app-bg: #fafafa;
    --app-surface: #ffffff;
    --app-surface-muted: #f7f7f7;
    --app-surface-raised: #ffffff;
    --app-border: #e5e5e5;
    --app-border-strong: #d4d4d4;
    --app-ink: #171717;
    --app-muted: #525252;
    --app-subtle: #737373;
    --app-accent: #171717;
    --app-accent-strong: #0a0a0a;
    --app-accent-soft: #e5e5e5;
    --app-danger: #dc2626;
    --app-danger-soft: #fef2f2;
    --app-warning: #b45309;
    --app-warning-soft: #fffbeb;
    --app-info: #0369a1;
    --app-info-soft: #f0f9ff;
    --app-focus: rgb(23 23 23 / .14);
    background: var(--app-bg);
    color: var(--app-ink);
  }}
  * {{ box-sizing: border-box; }}
  body {{ margin: 0; min-height: 100vh; background: var(--app-bg); color: var(--app-ink); -webkit-font-smoothing: antialiased; }}
  main {{ width: min(100%, 1180px); margin: 0 auto; padding: 28px 20px 44px; }}
  header {{ display: grid; gap: 12px; margin-bottom: 18px; }}
  h1 {{ margin: 0; font-size: 26px; line-height: 1.18; font-weight: 680; letter-spacing: -.025em; }}
  h2 {{ margin: 0 0 5px; font-size: 16px; line-height: 1.35; font-weight: 650; letter-spacing: -.012em; }}
  h3 {{ margin: 0; font-size: 14px; line-height: 1.35; }}
  p {{ margin: 0; color: var(--app-muted); line-height: 1.55; }}
  a {{ color: var(--app-ink); text-underline-offset: 2px; text-decoration-color: var(--app-border-strong); }}
  a:hover {{ text-decoration-color: currentColor; }}
  .topline {{ display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 10px 14px; }}
  .product-brand {{ display: inline-flex; min-height: 34px; align-items: center; gap: 10px; }}
  .brand-logo {{ display: block; width: 116px; height: auto; object-fit: contain; object-position: left center; }}
  .installer-badge {{ display: inline-flex; min-height: 26px; align-items: center; border: 1px solid var(--app-border); border-radius: 6px; background: var(--app-surface-muted); padding: 4px 7px; color: var(--app-subtle); font-size: 11px; font-weight: 650; letter-spacing: .01em; }}
  :root.dark .brand-logo {{ filter: invert(1); }}
  @media (prefers-color-scheme: dark) {{ :root:not(.light) .brand-logo {{ filter: invert(1); }} }}
  .header-copy {{ display: grid; gap: 6px; max-width: 780px; }}
  .install-meta {{ display: flex; flex-wrap: wrap; gap: 8px; }}
  .meta-chip {{ display: inline-flex; min-height: 30px; align-items: center; gap: 6px; border: 1px solid var(--app-border); border-radius: 6px; background: var(--app-surface); padding: 5px 8px; color: var(--app-muted); font-size: 12px; }}
  .layout {{ display: grid; grid-template-columns: 246px minmax(0, 1fr); gap: 18px; align-items: start; }}
  .panel {{ border: 1px solid var(--app-border); border-radius: 8px; background: var(--app-surface); }}
  .panel-header {{ display: flex; align-items: flex-start; justify-content: space-between; gap: 18px; border-bottom: 1px solid var(--app-border); background: var(--app-surface); padding: 17px 18px; }}
  .panel-title {{ min-width: 0; }}
  .panel-title p {{ max-width: 68ch; color: var(--app-subtle); font-size: 13px; }}
  .step-count {{ flex: 0 0 auto; border: 1px solid var(--app-border); border-radius: 6px; background: var(--app-surface-muted); padding: 5px 8px; color: var(--app-subtle); font-size: 12px; font-weight: 600; }}
  .panel-body {{ padding: 18px; }}
  .grid {{ display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }}
  .field {{ display: flex; flex-direction: column; gap: 6px; min-width: 0; }}
  label {{ font-size: 12px; font-weight: 600; color: var(--app-muted); }}
  input {{ width: 100%; min-height: 42px; border: 1px solid var(--app-border); border-radius: 6px; padding: 8px 10px; font: inherit; background: var(--app-surface); color: var(--app-ink); transition: border-color .14s ease, box-shadow .14s ease; }}
  input::placeholder {{ color: var(--app-subtle); }}
  input:hover {{ border-color: var(--app-border-strong); }}
  input:focus {{ outline: none; border-color: var(--app-accent); box-shadow: 0 0 0 1px var(--app-accent), 0 0 0 4px var(--app-focus); }}
  form.was-validated .wizard-step:not([hidden]) input:invalid {{ border-color: var(--app-danger); }}
  .full {{ grid-column: 1 / -1; }}
  .hint {{ font-size: 12px; color: var(--app-subtle); line-height: 1.45; }}
  .alert {{ margin-bottom: 14px; border: 1px solid #fecaca; border-radius: 6px; background: var(--app-danger-soft); color: var(--app-danger); padding: 10px 12px; font-size: 13px; }}
  .notice {{ border: 1px solid #bae6fd; border-radius: 6px; background: var(--app-info-soft); color: var(--app-info); padding: 10px 12px; font-size: 13px; line-height: 1.5; }}
  .warning {{ border: 1px solid #fde68a; border-radius: 6px; background: var(--app-warning-soft); color: var(--app-warning); padding: 10px 12px; font-size: 13px; line-height: 1.5; }}
  details {{ border-top: 1px solid var(--app-border); }}
  summary {{ cursor: pointer; padding: 15px 18px; font-size: 13px; font-weight: 600; color: var(--app-muted); }}
  summary:focus-visible {{ outline: none; box-shadow: inset 0 0 0 2px var(--app-accent); }}
  button {{ min-height: 42px; border: 1px solid var(--app-accent); border-radius: 6px; background: var(--app-accent); color: var(--app-surface); padding: 0 16px; font: inherit; font-size: 13px; font-weight: 650; cursor: pointer; transition: background-color .14s ease, border-color .14s ease, box-shadow .14s ease, color .14s ease; }}
  button:hover {{ background: var(--app-accent-strong); border-color: var(--app-accent-strong); }}
  button:focus-visible {{ outline: none; box-shadow: 0 0 0 1px var(--app-accent), 0 0 0 4px var(--app-focus); }}
  button:disabled {{ cursor: not-allowed; opacity: .58; }}
  button.secondary {{ border-color: var(--app-border); background: var(--app-surface); color: var(--app-muted); }}
  button.secondary:hover {{ border-color: var(--app-border-strong); background: var(--app-surface-muted); color: var(--app-ink); }}
  .actions {{ display: flex; align-items: center; justify-content: space-between; gap: 12px; border-top: 1px solid var(--app-border); padding: 15px 18px; }}
  .actions-right {{ display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 10px; }}
  .action-hint {{ color: var(--app-subtle); font-size: 12px; }}
  ol {{ margin: 10px 0 0 20px; padding: 0; color: var(--app-muted); line-height: 1.55; }}
  li + li {{ margin-top: 8px; }}
  code {{ display: inline-block; max-width: 100%; overflow-wrap: anywhere; border: 1px solid var(--app-border); border-radius: 4px; background: var(--app-surface-muted); padding: 1px 4px; color: var(--app-ink); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; font-size: 12px; }}
  .stack {{ display: grid; gap: 14px; }}
  .stepper {{ position: sticky; top: 20px; display: grid; gap: 4px; padding: 8px; }}
  .step-tab {{ display: grid; grid-template-columns: 28px minmax(0, 1fr); gap: 10px; align-items: center; border: 1px solid transparent; border-radius: 6px; padding: 10px; color: var(--app-subtle); transition: background-color .14s ease, border-color .14s ease; }}
  .step-number {{ display: inline-flex; width: 28px; height: 28px; align-items: center; justify-content: center; border-radius: 6px; background: var(--app-surface-muted); color: var(--app-muted); font-size: 12px; font-weight: 700; }}
  .step-title {{ display: block; color: var(--app-ink); font-size: 13px; font-weight: 650; }}
  .step-body {{ display: block; margin-top: 2px; font-size: 12px; }}
  .step-tab.active {{ border-color: var(--app-border-strong); background: var(--app-surface-muted); color: var(--app-muted); }}
  .step-tab.active .step-number {{ background: var(--app-accent); color: var(--app-surface); }}
  .step-tab.done .step-number {{ background: var(--app-accent-soft); color: var(--app-ink); }}
  .wizard-step[hidden] {{ display: none; }}
  .guide {{ display: grid; gap: 12px; margin-bottom: 16px; }}
  .guide-card {{ border: 1px solid var(--app-border); border-radius: 8px; background: color-mix(in srgb, var(--app-surface-muted) 68%, var(--app-surface)); padding: 14px; }}
  .guide-card strong {{ display: block; margin-bottom: 4px; color: var(--app-ink); font-size: 13px; }}
  .example-grid {{ display: grid; gap: 8px; margin-top: 12px; }}
  .example-row {{ display: grid; grid-template-columns: 8rem minmax(0, 1fr); gap: 10px; align-items: center; font-size: 13px; }}
  .example-row span {{ color: var(--app-subtle); }}
  .review {{ display: grid; gap: 10px; margin-top: 12px; }}
  .review-row {{ display: grid; grid-template-columns: 11rem minmax(0, 1fr); gap: 12px; border-bottom: 1px solid var(--app-border); padding-bottom: 10px; }}
  .review-row span:first-child {{ color: var(--app-subtle); }}
  .review-row span:last-child {{ min-width: 0; overflow-wrap: anywhere; font-weight: 600; }}
  .theme-toggle {{ cursor: pointer; width: 30px; min-height: 30px; padding: 0; border: 1px solid var(--app-border); background: var(--app-surface); color: var(--app-muted); border-radius: 6px; display: inline-flex; align-items: center; justify-content: center; }}
  .theme-toggle:hover {{ border-color: var(--app-border-strong); background: var(--app-surface-muted); color: var(--app-ink); }}
  @media (max-width: 900px) {{
    main {{ padding: 20px 14px 34px; }}
    .layout {{ grid-template-columns: 1fr; }}
    .stepper {{ position: static; grid-template-columns: repeat(4, minmax(180px, 1fr)); overflow-x: auto; }}
    .grid {{ grid-template-columns: 1fr; }}
    .panel-header {{ flex-direction: column; }}
  }}
  @media (max-width: 620px) {{
    h1 {{ font-size: 23px; }}
    .stepper {{ grid-template-columns: 1fr; }}
    .actions {{ align-items: stretch; flex-direction: column; }}
    .actions-right, .actions-right button, .actions > button {{ width: 100%; }}
    .review-row, .example-row {{ grid-template-columns: 1fr; gap: 4px; }}
  }}
  @media (prefers-reduced-motion: reduce) {{
    *, *::before, *::after {{ scroll-behavior: auto !important; transition-duration: .01ms !important; }}
  }}
  :root.dark {{
    --app-bg: #0a0a0a;
    --app-surface: #141414;
    --app-surface-muted: #1a1a1a;
    --app-surface-raised: #171717;
    --app-border: #292929;
    --app-border-strong: #404040;
    --app-ink: #fafafa;
    --app-muted: #a3a3a3;
    --app-subtle: #737373;
    --app-accent: #fafafa;
    --app-accent-strong: #ffffff;
    --app-accent-soft: #333333;
    --app-danger: #f87171;
    --app-danger-soft: rgb(127 29 29 / .25);
    --app-warning: #fbbf24;
    --app-warning-soft: rgb(120 53 15 / .22);
    --app-info: #38bdf8;
    --app-info-soft: rgb(12 74 110 / .25);
    --app-focus: rgb(250 250 250 / .14);
  }}
  @media (prefers-color-scheme: dark) {{
    :root:not(.light) {{
      --app-bg: #0a0a0a;
      --app-surface: #141414;
      --app-surface-muted: #1a1a1a;
      --app-surface-raised: #171717;
      --app-border: #292929;
      --app-border-strong: #404040;
      --app-ink: #fafafa;
      --app-muted: #a3a3a3;
      --app-subtle: #737373;
      --app-accent: #fafafa;
      --app-accent-strong: #ffffff;
      --app-accent-soft: #333333;
      --app-danger: #f87171;
      --app-danger-soft: rgb(127 29 29 / .25);
      --app-warning: #fbbf24;
      --app-warning-soft: rgb(120 53 15 / .22);
      --app-info: #38bdf8;
      --app-info-soft: rgb(12 74 110 / .25);
      --app-focus: rgb(250 250 250 / .14);
    }}
  }}
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
  <main>
    <header>
      <div class="topline">
        <div class="product-brand"><img class="brand-logo" src="/brand/logo.png" alt="MyPaas"><span class="installer-badge">Installer</span></div>
        <div class="install-meta" aria-label="Install context">
          <button type="button" id="theme-toggle" class="theme-toggle" aria-label="Toggle theme">
            <svg id="theme-icon-sun" viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round" style="display:none"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
            <svg id="theme-icon-moon" viewBox="0 0 24 24" width="14" height="14" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
          </button>
          <span class="meta-chip">Environment <code>{esc(ENV_FILE)}</code></span>
          <span class="meta-chip">Fresh Linux VM</span>
        </div>
      </div>
      <div class="header-copy">
        <h1>Set up MyPaas</h1>
        <p>Complete four short steps. Nothing is saved until the final review, and the terminal installer continues automatically when you finish.</p>
      </div>
    </header>
    <div class="layout">
      <aside class="panel stepper" aria-label="Install steps" role="list">
        <div class="step-tab active" data-progress="0" role="listitem" aria-current="step">
          <span class="step-number">1</span>
          <span><span class="step-title">Domain</span><span class="step-body">Where MyPaas lives</span></span>
        </div>
        <div class="step-tab" data-progress="1" role="listitem">
          <span class="step-number">2</span>
          <span><span class="step-title">GitHub login</span><span class="step-body">Owner sign-in</span></span>
        </div>
        <div class="step-tab" data-progress="2" role="listitem">
          <span class="step-number">3</span>
          <span><span class="step-title">Routing</span><span class="step-body">Cloudflare tunnel</span></span>
        </div>
        <div class="step-tab" data-progress="3" role="listitem">
          <span class="step-number">4</span>
          <span><span class="step-title">Review</span><span class="step-body">Confirm settings</span></span>
        </div>
      </aside>

      <form class="panel" method="post" action="/save" aria-labelledby="step-heading">
        <input type="hidden" name="token" value="{esc(TOKEN)}">
        <div class="panel-header">
          <div class="panel-title">
            <h2 id="step-heading">Domain and owner</h2>
            <p id="step-description">Choose the hostname for the dashboard and the GitHub account that owns this installation.</p>
          </div>
          <span class="step-count" id="step-position">Step 1 of 4</span>
        </div>
        <div class="panel-body">
          {f'<div class="alert" role="alert">{esc(error)}</div>' if error else ''}

          <section class="wizard-step" data-step="0">
            <div class="guide">
              <div class="guide-card">
                <strong>Use a domain that is already active in Cloudflare.</strong>
                <p>Enter only the hostname. MyPaas uses it for the dashboard and automatically places projects below it.</p>
                <div class="example-grid">
                  <div class="example-row"><span>Dashboard</span><code id="example-dashboard">https://example.com</code></div>
                  <div class="example-row"><span>Project example</span><code id="example-project">https://todo.example.com</code></div>
                </div>
              </div>
              <div class="guide-card">
                <strong>Before continuing</strong>
                <ol>
                  <li>Cloudflare shows the zone as <strong>Active</strong>.</li>
                  <li>You can edit DNS records for that zone.</li>
                  <li>You know the primary email of the GitHub account that will own MyPaas.</li>
                </ol>
              </div>
              <div class="notice">Using <code>panel.example.com</code>? Project URLs become <code>project.panel.example.com</code>.</div>
            </div>
            <div class="grid">
              <div class="field">
                <label for="PUBLIC_DOMAIN">Public MyPaas domain</label>
                <input id="PUBLIC_DOMAIN" name="PUBLIC_DOMAIN" required inputmode="url" autocomplete="off" placeholder="example.com" value="{esc(domain)}">
                <span class="hint">Use the hostname only, without <code>https://</code>. For example: <code>example.com</code> or <code>panel.example.com</code>.</span>
              </div>
              <div class="field">
                <label for="OWNER_EMAIL">Owner GitHub primary email</label>
                <input id="OWNER_EMAIL" name="OWNER_EMAIL" required type="email" autocomplete="email" placeholder="you@example.com" value="{esc(values.get("OWNER_EMAIL", ""))}">
                <span class="hint">Only this whitelisted email can log in as the first owner.</span>
              </div>
            </div>
          </section>

          <section class="wizard-step" data-step="1" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>Create one GitHub OAuth App.</strong>
                <ol>
                  <li>Open <a href="https://github.com/settings/developers" target="_blank" rel="noopener">GitHub Developer settings</a> → <strong>OAuth Apps</strong> → <strong>New OAuth App</strong>.</li>
                  <li>Use <strong>MyPaas</strong> as the application name.</li>
                  <li>Homepage URL: <code id="github-homepage-example">https://your-domain</code></li>
                  <li>Callback URL: <code id="github-callback-example">https://your-domain/api/auth/github/callback</code></li>
                  <li>Copy the Client ID and generate one Client Secret, then paste both below.</li>
                </ol>
              </div>
            </div>
            <div class="grid">
            <div class="field">
              <label for="GITHUB_CLIENT_ID">OAuth Client ID</label>
              <input id="GITHUB_CLIENT_ID" name="GITHUB_CLIENT_ID" required autocomplete="off" value="{esc(values.get("GITHUB_CLIENT_ID", ""))}">
            </div>
            <div class="field">
              <label for="GITHUB_CLIENT_SECRET">OAuth Client Secret</label>
              <input id="GITHUB_CLIENT_SECRET" name="GITHUB_CLIENT_SECRET" required autocomplete="off" value="{esc(values.get("GITHUB_CLIENT_SECRET", ""))}">
            </div>
            <div class="field full">
              <label for="GITHUB_CALLBACK_URL">GitHub OAuth callback URL</label>
              <input id="GITHUB_CALLBACK_URL" name="GITHUB_CALLBACK_URL" required type="url" autocomplete="off" data-generated="{str(callback_is_generated).lower()}" value="{esc(callback)}">
              <span class="hint">Must match the callback URL in the GitHub OAuth app exactly.</span>
            </div>
            </div>
          </section>

          <section class="wizard-step" data-step="2" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>1. Get the tunnel token</strong>
                <ol>
                  <li>In Cloudflare Zero Trust open <strong>Networks → Tunnels</strong>.</li>
                  <li>Create a tunnel, or open the tunnel you want MyPaas to use.</li>
                  <li>Select the <strong>Docker</strong> connector and copy only the value after <code>--token</code>.</li>
                </ol>
              </div>
              <div class="warning">Use a <strong>Tunnel token</strong>, not a Cloudflare API token.</div>
              <div class="guide-card">
                <strong>2. Add two public hostname routes</strong>
                <p>Both routes use service type <code>HTTP</code> and service URL <code>caddy:80</code>.</p>
                <div class="example-grid">
                  <div class="example-row"><span>Dashboard route</span><code id="cf-root-example">your-domain</code></div>
                  <div class="example-row"><span>Project wildcard</span><code id="cf-wildcard-example">*.your-domain</code></div>
                </div>
              </div>
              <div class="guide-card">
                <strong>3. Confirm DNS</strong>
                <ol>
                  <li>Make sure the root/subdomain and wildcard records exist.</li>
                  <li>Both records should resolve through the tunnel target <code>&lt;tunnel-id&gt;.cfargotunnel.com</code> with proxy enabled.</li>
                  <li>If Cloudflare does not create the wildcard automatically, add that CNAME manually.</li>
                </ol>
              </div>
            </div>
            <div class="grid">
              <div class="field full">
              <label for="CLOUDFLARE_TUNNEL_TOKEN">Cloudflare Tunnel token</label>
              <input id="CLOUDFLARE_TUNNEL_TOKEN" name="CLOUDFLARE_TUNNEL_TOKEN" required autocomplete="off" value="{esc(values.get("CLOUDFLARE_TUNNEL_TOKEN", ""))}">
              <span class="hint">Use a Cloudflare Zero Trust tunnel token, not an API token.</span>
            </div>
          </div>
          </section>

          <section class="wizard-step" data-step="3" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>Confirm the values below.</strong>
                <p>Saving writes <code>{esc(ENV_FILE)}</code>. The browser wizard then closes and the terminal continues with directories, migrations, and startup.</p>
                <div class="review">
                  <div class="review-row"><span>Dashboard</span><span id="review-dashboard">-</span></div>
                  <div class="review-row"><span>Project URL pattern</span><span id="review-project">-</span></div>
                  <div class="review-row"><span>GitHub callback</span><span id="review-callback">-</span></div>
                  <div class="review-row"><span>Owner email</span><span id="review-owner">-</span></div>
                </div>
              </div>
            </div>
            <details>
              <summary>Advanced generated values</summary>
              <div class="panel-body grid">
                {advanced_field("POSTGRES_USER", "Postgres user", values)}
                {advanced_field("POSTGRES_DB", "Postgres database", values)}
                {advanced_field("POSTGRES_PASSWORD", "Postgres password", values)}
                {advanced_field("PROJECT_NETWORK", "Docker project network", values)}
                {advanced_field("DOCKER_BIND_HOST", "Docker bind host", values)}
                {advanced_field("METRICS_PASSWORD", "Metrics password", values)}
                <div class="field full">
                  <label for="JWT_SECRET">JWT secret</label>
                  <input id="JWT_SECRET" name="JWT_SECRET" required autocomplete="off" value="{esc(values.get("JWT_SECRET", ""))}">
                </div>
                <div class="field full">
                  <label for="ENCRYPTION_KEY">Env encryption key</label>
                  <input id="ENCRYPTION_KEY" name="ENCRYPTION_KEY" required autocomplete="off" value="{esc(values.get("ENCRYPTION_KEY", ""))}">
                </div>
              </div>
            </details>
          </section>
        </div>
        <div class="actions">
          <button class="secondary" type="button" id="back-button">Back</button>
          <span class="action-hint" id="action-hint">Nothing is saved until the final step.</span>
          <div class="actions-right">
            <button type="button" id="next-button">Continue</button>
            <button type="submit" id="submit-button" data-default-label="Save configuration" hidden>Save configuration</button>
          </div>
        </div>
      </form>
    </div>
  </main>
  <script>
    const steps = Array.from(document.querySelectorAll('.wizard-step'));
    const progress = Array.from(document.querySelectorAll('[data-progress]'));
    const form = document.querySelector('form');
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
    const titles = [
      ['Domain and owner', 'Choose where MyPaas lives and who owns the first account.'],
      ['GitHub login', 'Connect one OAuth App for owner sign-in.'],
      ['Cloudflare routing', 'Send the dashboard and project wildcard through the tunnel to Caddy.'],
      ['Review and save', 'Confirm the public URLs and owner before writing configuration.']
    ];
    let currentStep = 0;
    let callbackTouched = callback.dataset.generated !== 'true' && Boolean(callback.value);

    function cleanDomain() {{
      return domain.value.trim().replace(/^https?:\\/\\//i, '').replace(/\\/.*$/, '').replace(/\\.$/, '').toLowerCase();
    }}

    function setText(id, value) {{
      const node = document.getElementById(id);
      if (node) node.textContent = value;
    }}

    function updateDerivedText() {{
      const clean = cleanDomain() || 'mypaas.example.com';
      setText('example-dashboard', `https://${{clean}}`);
      setText('example-project', `https://todo.${{clean}}`);
      setText('github-homepage-example', `https://${{clean}}`);
      setText('github-callback-example', `https://${{clean}}/api/auth/github/callback`);
      setText('cf-root-example', clean);
      setText('cf-wildcard-example', `*.${{clean}}`);
      setText('review-dashboard', cleanDomain() ? `https://${{cleanDomain()}}` : '-');
      setText('review-project', cleanDomain() ? `https://<project>.${{cleanDomain()}}` : '-');
      setText('review-callback', callback.value || '-');
      setText('review-owner', owner.value || '-');
    }}

    function showStep(index) {{
      currentStep = Math.max(0, Math.min(index, steps.length - 1));
      steps.forEach((step, stepIndex) => step.hidden = stepIndex !== currentStep);
      progress.forEach((item, itemIndex) => {{
        item.classList.toggle('active', itemIndex === currentStep);
        item.classList.toggle('done', itemIndex < currentStep);
        if (itemIndex === currentStep) {{
          item.setAttribute('aria-current', 'step');
        }} else {{
          item.removeAttribute('aria-current');
        }}
      }});
      heading.textContent = titles[currentStep][0];
      description.textContent = titles[currentStep][1];
      stepPosition.textContent = `Step ${{currentStep + 1}} of ${{steps.length}}`;
      form.classList.remove('was-validated');
      backButton.hidden = currentStep === 0;
      nextButton.hidden = currentStep === steps.length - 1;
      submitButton.hidden = currentStep !== steps.length - 1;
      actionHint.textContent = currentStep === steps.length - 1
        ? 'Save writes the production configuration, then the terminal installer resumes.'
        : 'You can go back without losing the values you entered.';
      updateDerivedText();
    }}

    function validateCurrentStep() {{
      form.classList.add('was-validated');
      const invalid = Array.from(steps[currentStep].querySelectorAll('input[required]'))
        .find((input) => !input.checkValidity());
      if (!invalid) return true;
      invalid.reportValidity();
      return false;
    }}

    backButton.addEventListener('click', () => showStep(currentStep - 1));
    nextButton.addEventListener('click', () => {{
      if (validateCurrentStep()) showStep(currentStep + 1);
    }});
    form.addEventListener('submit', (event) => {{
      if (currentStep !== steps.length - 1) {{
        event.preventDefault();
        if (validateCurrentStep()) showStep(currentStep + 1);
        return;
      }}
      if (!validateCurrentStep()) {{
        event.preventDefault();
        return;
      }}
      submitButton.disabled = true;
      submitButton.textContent = 'Saving config...';
      backButton.disabled = true;
      nextButton.disabled = true;
    }});
    callback.addEventListener('input', () => callbackTouched = true);
    domain.addEventListener('input', () => {{
      const clean = cleanDomain();
      if (!callbackTouched) {{
        callback.value = clean ? `https://${{clean}}/api/auth/github/callback` : '';
      }}
      updateDerivedText();
    }});
    owner.addEventListener('input', updateDerivedText);
    callback.addEventListener('input', updateDerivedText);
    showStep(0);

    const themeToggle = document.getElementById('theme-toggle');
    const themeIconSun = document.getElementById('theme-icon-sun');
    const themeIconMoon = document.getElementById('theme-icon-moon');
    function updateThemeIcon() {{
      const isDark = document.documentElement.classList.contains('dark') || (!document.documentElement.classList.contains('light') && window.matchMedia('(prefers-color-scheme: dark)').matches);
      if (isDark) {{
        themeIconSun.style.display = 'block';
        themeIconMoon.style.display = 'none';
      }} else {{
        themeIconSun.style.display = 'none';
        themeIconMoon.style.display = 'block';
      }}
    }}
    if (themeToggle) {{
      updateThemeIcon();
      themeToggle.addEventListener('click', () => {{
        const isDark = document.documentElement.classList.contains('dark') || (!document.documentElement.classList.contains('light') && window.matchMedia('(prefers-color-scheme: dark)').matches);
        if (isDark) {{
          document.documentElement.classList.remove('dark');
          document.documentElement.classList.add('light');
          localStorage.setItem('theme', 'light');
        }} else {{
          document.documentElement.classList.remove('light');
          document.documentElement.classList.add('dark');
          localStorage.setItem('theme', 'dark');
        }}
        updateThemeIcon();
      }});
    }}
  </script>
</body>
</html>"""
    return body.encode("utf-8")


def advanced_field(name: str, label: str, values: dict[str, str]) -> str:
    return f"""<div class="field">
      <label for="{esc(name)}">{esc(label)}</label>
      <input id="{esc(name)}" name="{esc(name)}" required autocomplete="off" value="{esc(values.get(name, ""))}">
    </div>"""


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:
        print(f"{self.address_string()} - {fmt % args}")

    def send_html(self, body: bytes, status: int = 200) -> None:
        self.send_response(status)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def send_asset(self, path: str, content_type: str) -> None:
        try:
            with open(path, "rb") as handle:
                body = handle.read()
        except OSError:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Cache-Control", "public, max-age=3600")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        parsed = urlparse(self.path)
        if parsed.path == "/brand/logo.png":
            self.send_asset(BRAND_LOGO_PATH, "image/png")
            return
        if parsed.path == "/health":
            self.send_html(b"ok")
            return
        query = parse_qs(parsed.query)
        if query.get("token", [""])[0] != TOKEN:
            self.send_html(form_html("Invalid or missing wizard token. Use the URL printed by install-vm.sh."), 403)
            return
        self.send_html(form_html())

    def do_POST(self) -> None:
        parsed = urlparse(self.path)
        if parsed.path != "/save":
            self.send_error(404)
            return
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length).decode("utf-8")
        data = parse_qs(raw, keep_blank_values=True)
        values = {key: field(data, key) for key in DEFAULTS.keys()}
        if field(data, "token") != TOKEN:
            self.send_html(form_html("Invalid wizard token.", values), 403)
            return
        try:
            write_env(build_env(values))
        except ValueError as err:
            self.send_html(form_html(str(err), values), 400)
            return
        self.send_html(success_html())
        threading.Thread(target=self.server.shutdown, daemon=True).start()


def success_html() -> bytes:
    body = f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MyPaas Install Wizard Complete</title>
  <style>
    @font-face {{
      font-family: "Inter Variable";
      font-style: normal;
      font-display: swap;
      font-weight: 100 900;
      src: url("https://cdn.jsdelivr.net/fontsource/fonts/inter:vf@5.3.0/latin-wght-normal.woff2") format("woff2-variations");
    }}
    :root {{
    color-scheme: light dark;
    font-family: "Inter Variable", "Inter", ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    --app-bg: #fafafa;
    --app-surface: #ffffff;
    --app-surface-muted: #f7f7f7;
    --app-border: #e5e5e5;
    --app-ink: #171717;
    --app-muted: #525252;
    --app-success: #059669;
    --app-success-soft: #ecfdf5;
  }}
  * {{ box-sizing: border-box; }}
  body {{ margin: 0; min-height: 100vh; background: var(--app-bg); color: var(--app-ink); -webkit-font-smoothing: antialiased; }}
  main {{ width: min(100%, 680px); margin: 0 auto; padding: 48px 20px; }}
  section {{ display: grid; gap: 12px; border: 1px solid var(--app-border); border-radius: 8px; background: var(--app-surface); padding: 24px; }}
  .success-brand {{ display: flex; align-items: center; margin-bottom: 6px; }}
  .success-brand img {{ width: 116px; height: auto; }}
  :root.dark .success-brand img {{ filter: invert(1); }}
  @media (prefers-color-scheme: dark) {{ :root:not(.light) .success-brand img {{ filter: invert(1); }} }}
  .status-mark {{ display: inline-flex; width: 34px; height: 34px; align-items: center; justify-content: center; border: 1px solid color-mix(in srgb, var(--app-success) 30%, var(--app-border)); border-radius: 8px; background: var(--app-success-soft); color: var(--app-success); font-weight: 800; }}
  h1 {{ margin: 0; font-size: 24px; line-height: 1.2; font-weight: 680; letter-spacing: -.02em; }}
  p {{ margin: 0; color: var(--app-muted); line-height: 1.55; }}
  code {{ display: inline-block; max-width: 100%; overflow-wrap: anywhere; border: 1px solid var(--app-border); border-radius: 4px; background: var(--app-surface-muted); padding: 1px 4px; color: var(--app-ink); font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }}
  :root.dark {{
    --app-bg: #0a0a0a;
    --app-surface: #141414;
    --app-surface-muted: #1a1a1a;
    --app-border: #292929;
    --app-ink: #fafafa;
    --app-muted: #a3a3a3;
    --app-success: #34d399;
    --app-success-soft: rgb(6 78 59 / .25);
  }}
  @media (prefers-color-scheme: dark) {{
    :root:not(.light) {{
      --app-bg: #0a0a0a;
      --app-surface: #141414;
      --app-surface-muted: #1a1a1a;
      --app-border: #292929;
      --app-ink: #fafafa;
      --app-muted: #a3a3a3;
      --app-success: #34d399;
      --app-success-soft: rgb(6 78 59 / .25);
    }}
  }}
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
  <main>
    <section>
      <div class="success-brand"><img src="/brand/logo.png" alt="MyPaas"></div>
      <span class="status-mark" aria-hidden="true">✓</span>
      <h1>Production config saved</h1>
      <p>Production config was written to <code>{esc(ENV_FILE)}</code>. You can close this tab. The terminal installer will continue automatically.</p>
    </section>
  </main>
</body>
</html>"""
    return body.encode("utf-8")


def main() -> None:
    server = HTTPServer((HOST, PORT), Handler)
    # Output is handled by run-install-wizard.sh to avoid confusing local loopback URLs
    server.serve_forever()


if __name__ == "__main__":
    main()
