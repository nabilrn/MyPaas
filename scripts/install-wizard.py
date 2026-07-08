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
CADDY_UPSTREAM_HOST=host.docker.internal
STATIC_ROOT=/var/lib/mypaas/static
CADDY_STATIC_ROOT=/var/lib/mypaas/static
CADDY_METRICS=true
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
    :root {{
      color-scheme: light dark;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: #f6f7f8;
      color: #111827;
    }}
    body {{ margin: 0; }}
    main {{ max-width: 1180px; margin: 0 auto; padding: 32px 20px 48px; }}
    header {{ margin-bottom: 22px; }}
    h1 {{ margin: 0; font-size: 28px; line-height: 1.15; letter-spacing: 0; }}
    h2 {{ margin: 0 0 12px; font-size: 16px; }}
    h3 {{ margin: 18px 0 8px; font-size: 14px; }}
    p {{ margin: 0; color: #4b5563; line-height: 1.55; }}
    a {{ color: #047857; }}
    .layout {{ display: grid; grid-template-columns: 250px minmax(0, 1fr); gap: 18px; align-items: start; }}
    .panel {{ border: 1px solid #d9dee6; border-radius: 8px; background: #fff; box-shadow: 0 1px 2px rgba(15, 23, 42, .04); }}
    .panel-header {{ border-bottom: 1px solid #e5e7eb; padding: 18px; }}
    .panel-body {{ padding: 18px; }}
    .grid {{ display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }}
    .field {{ display: flex; flex-direction: column; gap: 6px; }}
    label {{ font-size: 12px; font-weight: 650; color: #4b5563; }}
    input {{ min-height: 40px; border: 1px solid #cfd6df; border-radius: 6px; padding: 8px 10px; font: inherit; background: #fff; color: #111827; }}
    input:focus {{ outline: none; border-color: #047857; box-shadow: 0 0 0 3px rgba(4, 120, 87, .14); }}
    .full {{ grid-column: 1 / -1; }}
    .hint {{ font-size: 12px; color: #6b7280; }}
    .alert {{ margin-bottom: 14px; border: 1px solid #fecaca; border-radius: 6px; background: #fef2f2; color: #991b1b; padding: 10px 12px; font-size: 14px; }}
    .notice {{ border: 1px solid #bfdbfe; border-radius: 6px; background: #eff6ff; color: #1e3a8a; padding: 10px 12px; font-size: 13px; }}
    .warning {{ border: 1px solid #fde68a; border-radius: 6px; background: #fffbeb; color: #92400e; padding: 10px 12px; font-size: 13px; }}
    details {{ border-top: 1px solid #e5e7eb; }}
    summary {{ cursor: pointer; padding: 16px 18px; font-weight: 650; color: #374151; }}
    button {{ min-height: 42px; border: 1px solid #065f46; border-radius: 6px; background: #047857; color: #fff; padding: 0 16px; font-weight: 700; cursor: pointer; }}
    button:hover {{ background: #065f46; }}
    button.secondary {{ border-color: #d1d5db; background: #fff; color: #374151; }}
    button.secondary:hover {{ background: #f9fafb; }}
    .actions {{ display: flex; justify-content: space-between; gap: 12px; border-top: 1px solid #e5e7eb; padding: 16px 18px; }}
    .actions-right {{ display: flex; gap: 10px; }}
    ol {{ margin: 10px 0 0 20px; padding: 0; color: #374151; line-height: 1.55; }}
    li + li {{ margin-top: 10px; }}
    code {{ background: #f3f4f6; border: 1px solid #e5e7eb; border-radius: 4px; padding: 1px 4px; font-size: 12px; }}
    .stack {{ display: grid; gap: 14px; }}
    .stepper {{ position: sticky; top: 20px; display: grid; gap: 8px; padding: 10px; }}
    .step-tab {{ display: grid; grid-template-columns: 28px minmax(0, 1fr); gap: 10px; align-items: center; border: 1px solid transparent; border-radius: 8px; padding: 10px; color: #6b7280; }}
    .step-number {{ display: inline-flex; width: 28px; height: 28px; align-items: center; justify-content: center; border-radius: 7px; background: #f3f4f6; color: #4b5563; font-size: 12px; font-weight: 800; }}
    .step-title {{ display: block; color: #111827; font-size: 13px; font-weight: 750; }}
    .step-body {{ display: block; margin-top: 2px; font-size: 12px; }}
    .step-tab.active {{ border-color: #a7f3d0; background: #ecfdf5; color: #047857; }}
    .step-tab.active .step-number {{ background: #047857; color: #fff; }}
    .step-tab.done .step-number {{ background: #d1fae5; color: #047857; }}
    .wizard-step[hidden] {{ display: none; }}
    .guide {{ display: grid; gap: 12px; margin-bottom: 16px; }}
    .guide-card {{ border: 1px solid #e5e7eb; border-radius: 8px; background: #f9fafb; padding: 14px; }}
    .guide-card strong {{ color: #111827; }}
    .example-grid {{ display: grid; gap: 8px; margin-top: 12px; }}
    .example-row {{ display: grid; grid-template-columns: 8rem minmax(0, 1fr); gap: 10px; align-items: center; font-size: 13px; }}
    .review {{ display: grid; gap: 10px; }}
    .review-row {{ display: grid; grid-template-columns: 11rem minmax(0, 1fr); gap: 12px; border-bottom: 1px solid #eef2f7; padding-bottom: 10px; }}
    .review-row span:first-child {{ color: #6b7280; }}
    .review-row span:last-child {{ min-width: 0; overflow-wrap: anywhere; font-weight: 650; }}
    @media (max-width: 900px) {{ .layout {{ grid-template-columns: 1fr; }} .grid {{ grid-template-columns: 1fr; }} }}
    @media (prefers-color-scheme: dark) {{
      :root {{ background: #030712; color: #f9fafb; }}
      .panel {{ background: #111827; border-color: #273244; }}
      .panel-header, details, .actions {{ border-color: #273244; }}
      p, label, .hint, ol {{ color: #cbd5e1; }}
      input {{ background: #030712; color: #f9fafb; border-color: #374151; }}
      code {{ background: #030712; border-color: #374151; }}
      .notice {{ background: rgba(30, 58, 138, .25); border-color: #1d4ed8; color: #bfdbfe; }}
      .warning {{ background: rgba(146, 64, 14, .22); border-color: #92400e; color: #fde68a; }}
      button.secondary {{ border-color: #374151; background: #111827; color: #e5e7eb; }}
      button.secondary:hover {{ background: #1f2937; }}
      .step-title {{ color: #f9fafb; }}
      .step-number {{ background: #1f2937; color: #cbd5e1; }}
      .step-tab.active {{ border-color: #047857; background: rgba(4, 120, 87, .16); color: #a7f3d0; }}
      .step-tab.done .step-number {{ background: rgba(4, 120, 87, .25); color: #a7f3d0; }}
      .guide-card {{ background: #0b1220; border-color: #273244; }}
      .guide-card strong {{ color: #f9fafb; }}
      .review-row {{ border-color: #273244; }}
      .review-row span:first-child {{ color: #94a3b8; }}
    }}
  </style>
</head>
<body>
  <main>
    <header>
      <h1>MyPaas Install Wizard</h1>
      <p>Fill the production credentials once. The wizard writes <code>{esc(ENV_FILE)}</code>, shuts down, and the installer continues.</p>
    </header>
    <div class="layout">
      <aside class="panel stepper" aria-label="Install steps">
        <div class="step-tab active" data-progress="0">
          <span class="step-number">1</span>
          <span><span class="step-title">Domain</span><span class="step-body">Base URL and owner</span></span>
        </div>
        <div class="step-tab" data-progress="1">
          <span class="step-number">2</span>
          <span><span class="step-title">GitHub</span><span class="step-body">OAuth login</span></span>
        </div>
        <div class="step-tab" data-progress="2">
          <span class="step-number">3</span>
          <span><span class="step-title">Cloudflare</span><span class="step-body">Tunnel routing</span></span>
        </div>
        <div class="step-tab" data-progress="3">
          <span class="step-number">4</span>
          <span><span class="step-title">Review</span><span class="step-body">Save and deploy</span></span>
        </div>
      </aside>

      <form class="panel" method="post" action="/save">
        <input type="hidden" name="token" value="{esc(TOKEN)}">
        <div class="panel-header">
          <h2 id="step-heading">Domain and owner</h2>
          <p id="step-description">Start with the public domain MyPaas will control.</p>
        </div>
        <div class="panel-body">
          {f'<div class="alert">{esc(error)}</div>' if error else ''}

          <section class="wizard-step" data-step="0">
            <div class="guide">
              <div class="guide-card">
                <strong>You need a domain you control.</strong>
                <p>MyPaas uses this domain as its base address. The dashboard runs at <code>https://your-domain</code>, and deployed projects get subdomains under it.</p>
                <div class="example-grid">
                  <div class="example-row"><span>Dashboard</span><code id="example-dashboard">https://mypaas.example.com</code></div>
                  <div class="example-row"><span>Project</span><code id="example-project">https://todo.mypaas.example.com</code></div>
                </div>
              </div>
              <div class="guide-card">
                <strong>The domain must be active in Cloudflare DNS.</strong>
                <ol>
                  <li>If you bought the domain at Cloudflare Registrar, it already uses Cloudflare DNS.</li>
                  <li>If you bought it elsewhere, add the domain in Cloudflare, copy the two Cloudflare nameservers, then change the nameservers at your registrar.</li>
                  <li>You do not have to transfer registrar ownership to Cloudflare. Nameserver change is enough for MyPaas.</li>
                  <li>Wait until Cloudflare shows the domain as active before testing MyPaas routes.</li>
                </ol>
              </div>
              <div class="notice">Example: if you enter <code>mypaas.my.id</code>, a project named <code>crud</code> will route to <code>crud.mypaas.my.id</code>.</div>
            </div>
            <div class="grid">
              <div class="field">
                <label for="PUBLIC_DOMAIN">Public MyPaas domain</label>
                <input id="PUBLIC_DOMAIN" name="PUBLIC_DOMAIN" required placeholder="mypaas.example.com" value="{esc(domain)}">
                <span class="hint">Use the hostname only, without <code>https://</code>. A dedicated subdomain like <code>mypaas.example.com</code> is recommended.</span>
              </div>
              <div class="field">
                <label for="OWNER_EMAIL">Owner GitHub primary email</label>
                <input id="OWNER_EMAIL" name="OWNER_EMAIL" required placeholder="you@example.com" value="{esc(values.get("OWNER_EMAIL", ""))}">
                <span class="hint">Only this whitelisted email can log in as the first owner.</span>
              </div>
            </div>
          </section>

          <section class="wizard-step" data-step="1" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>Create a GitHub OAuth app.</strong>
                <ol>
                  <li>Open <a href="https://github.com/settings/developers" target="_blank" rel="noopener">GitHub Developer settings</a>.</li>
                  <li>Choose <strong>OAuth Apps</strong>, then <strong>New OAuth App</strong>.</li>
                  <li>Set Homepage URL to <code id="github-homepage-example">https://your-domain</code>.</li>
                  <li>Set Authorization callback URL to <code id="github-callback-example">https://your-domain/api/auth/github/callback</code>.</li>
                  <li>Create the app, copy the Client ID, then generate and copy a Client Secret.</li>
                </ol>
              </div>
            </div>
            <div class="grid">
            <div class="field">
              <label for="GITHUB_CLIENT_ID">OAuth Client ID</label>
              <input id="GITHUB_CLIENT_ID" name="GITHUB_CLIENT_ID" required value="{esc(values.get("GITHUB_CLIENT_ID", ""))}">
            </div>
            <div class="field">
              <label for="GITHUB_CLIENT_SECRET">OAuth Client Secret</label>
              <input id="GITHUB_CLIENT_SECRET" name="GITHUB_CLIENT_SECRET" required value="{esc(values.get("GITHUB_CLIENT_SECRET", ""))}">
            </div>
            <div class="field full">
              <label for="GITHUB_CALLBACK_URL">GitHub OAuth callback URL</label>
              <input id="GITHUB_CALLBACK_URL" name="GITHUB_CALLBACK_URL" required data-generated="{str(callback_is_generated).lower()}" value="{esc(callback)}">
              <span class="hint">Must match the callback URL in the GitHub OAuth app exactly.</span>
            </div>
            </div>
          </section>

          <section class="wizard-step" data-step="2" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>Create or reuse a Cloudflare Tunnel token.</strong>
                <ol>
                  <li>Open Cloudflare Zero Trust, then go to <strong>Networks</strong> -> <strong>Tunnels</strong>.</li>
                  <li>Create a tunnel or open an existing tunnel.</li>
                  <li>Choose the <strong>Docker</strong> connector.</li>
                  <li>Copy the token from the generated <code>cloudflared tunnel run --token ...</code> command.</li>
                </ol>
              </div>
              <div class="warning">Use the Tunnel token from the Docker connector command. This is not the same thing as a Cloudflare API token.</div>
              <div class="guide-card">
                <strong>Add Public Hostname routes in the tunnel.</strong>
                <ol>
                  <li>In the tunnel, open <strong>Public Hostnames</strong> or <strong>Published application routes</strong>.</li>
                  <li>Add hostname <code id="cf-root-example">your-domain</code>, service type <code>HTTP</code>, service URL <code>caddy:80</code>.</li>
                  <li>Add hostname <code id="cf-wildcard-example">*.your-domain</code>, service type <code>HTTP</code>, service URL <code>caddy:80</code>.</li>
                  <li>The wildcard route is what lets every deployed project use <code>project.your-domain</code>.</li>
                  <li>Do not point these routes to the VM public IP. The tunnel container reaches Caddy inside Docker by the <code>caddy:80</code> service name.</li>
                </ol>
              </div>
              <div class="guide-card">
                <strong>Check DNS records after adding routes.</strong>
                <ol>
                  <li>Open <strong>Cloudflare DNS</strong> -> <strong>Records</strong>.</li>
                  <li>If your Cloudflare zone is exactly this MyPaas domain, create CNAME records for <code>@</code> and <code>*</code>.</li>
                  <li>If your zone is a parent domain, create records for this subdomain and wildcard subdomain, for example <code>mypaas</code> and <code>*.mypaas</code>.</li>
                  <li>Point both records to your tunnel target: <code>&lt;tunnel-id&gt;.cfargotunnel.com</code>, with proxy enabled.</li>
                  <li>If Cloudflare says a wildcard route will not create a DNS record, create the wildcard CNAME manually.</li>
                </ol>
              </div>
            </div>
            <div class="grid">
              <div class="field full">
              <label for="CLOUDFLARE_TUNNEL_TOKEN">Cloudflare Tunnel token</label>
              <input id="CLOUDFLARE_TUNNEL_TOKEN" name="CLOUDFLARE_TUNNEL_TOKEN" required value="{esc(values.get("CLOUDFLARE_TUNNEL_TOKEN", ""))}">
              <span class="hint">Use a Cloudflare Zero Trust tunnel token, not an API token.</span>
            </div>
          </div>
          </section>

          <section class="wizard-step" data-step="3" hidden>
            <div class="guide">
              <div class="guide-card">
                <strong>Review before saving.</strong>
                <p>The installer will write <code>{esc(ENV_FILE)}</code>, prepare host directories, run migrations, then start MyPaas.</p>
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
                  <input id="JWT_SECRET" name="JWT_SECRET" required value="{esc(values.get("JWT_SECRET", ""))}">
                </div>
                <div class="field full">
                  <label for="ENCRYPTION_KEY">Env encryption key</label>
                  <input id="ENCRYPTION_KEY" name="ENCRYPTION_KEY" required value="{esc(values.get("ENCRYPTION_KEY", ""))}">
                </div>
              </div>
            </details>
          </section>
        </div>
        <div class="actions">
          <button class="secondary" type="button" id="back-button">Back</button>
          <div class="actions-right">
            <button type="button" id="next-button">Continue</button>
            <button type="submit" id="submit-button">Save .env and continue install</button>
          </div>
        </div>
      </form>
    </div>
  </main>
  <script>
    const steps = Array.from(document.querySelectorAll('.wizard-step'));
    const progress = Array.from(document.querySelectorAll('[data-progress]'));
    const heading = document.getElementById('step-heading');
    const description = document.getElementById('step-description');
    const backButton = document.getElementById('back-button');
    const nextButton = document.getElementById('next-button');
    const submitButton = document.getElementById('submit-button');
    const domain = document.getElementById('PUBLIC_DOMAIN');
    const owner = document.getElementById('OWNER_EMAIL');
    const callback = document.getElementById('GITHUB_CALLBACK_URL');
    const titles = [
      ['Domain and owner', 'Start with the public domain MyPaas will control.'],
      ['GitHub login', 'Create the OAuth app MyPaas uses for dashboard login.'],
      ['Cloudflare tunnel', 'Connect the public domain and wildcard project subdomains to this VM.'],
      ['Review and save', 'Check the generated production config before the installer continues.']
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
      }});
      heading.textContent = titles[currentStep][0];
      description.textContent = titles[currentStep][1];
      backButton.hidden = currentStep === 0;
      nextButton.hidden = currentStep === steps.length - 1;
      submitButton.hidden = currentStep !== steps.length - 1;
      updateDerivedText();
    }}

    function validateCurrentStep() {{
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
  </script>
</body>
</html>"""
    return body.encode("utf-8")


def advanced_field(name: str, label: str, values: dict[str, str]) -> str:
    return f"""<div class="field">
      <label for="{esc(name)}">{esc(label)}</label>
      <input id="{esc(name)}" name="{esc(name)}" required value="{esc(values.get(name, ""))}">
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

    def do_GET(self) -> None:
        parsed = urlparse(self.path)
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
    body {{ margin: 0; font-family: Inter, ui-sans-serif, system-ui, sans-serif; background: #f6f7f8; color: #111827; }}
    main {{ max-width: 680px; margin: 0 auto; padding: 48px 20px; }}
    section {{ border: 1px solid #d9dee6; border-radius: 8px; background: #fff; padding: 24px; }}
    h1 {{ margin: 0 0 10px; font-size: 24px; }}
    p {{ margin: 0; color: #4b5563; line-height: 1.55; }}
    code {{ background: #f3f4f6; border: 1px solid #e5e7eb; border-radius: 4px; padding: 1px 4px; }}
  </style>
</head>
<body>
  <main>
    <section>
      <h1>Saved</h1>
      <p>Production config was written to <code>{esc(ENV_FILE)}</code>. You can close this tab. The terminal installer will continue automatically.</p>
    </section>
  </main>
</body>
</html>"""
    return body.encode("utf-8")


def main() -> None:
    server = HTTPServer((HOST, PORT), Handler)
    print(f"Install wizard listening on http://{HOST}:{PORT}/?token={TOKEN}")
    server.serve_forever()


if __name__ == "__main__":
    main()
