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
    main {{ max-width: 1120px; margin: 0 auto; padding: 32px 20px 48px; }}
    header {{ margin-bottom: 22px; }}
    h1 {{ margin: 0; font-size: 28px; line-height: 1.15; letter-spacing: 0; }}
    h2 {{ margin: 0 0 12px; font-size: 16px; }}
    h3 {{ margin: 18px 0 8px; font-size: 14px; }}
    p {{ margin: 0; color: #4b5563; line-height: 1.55; }}
    a {{ color: #047857; }}
    .layout {{ display: grid; grid-template-columns: minmax(0, 1fr) 360px; gap: 18px; align-items: start; }}
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
    details {{ border-top: 1px solid #e5e7eb; }}
    summary {{ cursor: pointer; padding: 16px 18px; font-weight: 650; color: #374151; }}
    button {{ min-height: 42px; border: 1px solid #065f46; border-radius: 6px; background: #047857; color: #fff; padding: 0 16px; font-weight: 700; cursor: pointer; }}
    button:hover {{ background: #065f46; }}
    .actions {{ display: flex; justify-content: flex-end; border-top: 1px solid #e5e7eb; padding: 16px 18px; }}
    ol {{ margin: 10px 0 0 20px; padding: 0; color: #374151; line-height: 1.55; }}
    li + li {{ margin-top: 10px; }}
    code {{ background: #f3f4f6; border: 1px solid #e5e7eb; border-radius: 4px; padding: 1px 4px; font-size: 12px; }}
    .stack {{ display: grid; gap: 14px; }}
    @media (max-width: 900px) {{ .layout {{ grid-template-columns: 1fr; }} .grid {{ grid-template-columns: 1fr; }} }}
    @media (prefers-color-scheme: dark) {{
      :root {{ background: #030712; color: #f9fafb; }}
      .panel {{ background: #111827; border-color: #273244; }}
      .panel-header, details, .actions {{ border-color: #273244; }}
      p, label, .hint, ol {{ color: #cbd5e1; }}
      input {{ background: #030712; color: #f9fafb; border-color: #374151; }}
      code {{ background: #030712; border-color: #374151; }}
      .notice {{ background: rgba(30, 58, 138, .25); border-color: #1d4ed8; color: #bfdbfe; }}
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
      <form class="panel" method="post" action="/save">
        <input type="hidden" name="token" value="{esc(TOKEN)}">
        <div class="panel-header">
          <h2>Required credentials</h2>
          <p>Credential fields are intentionally visible so pasted tokens can be checked before saving.</p>
        </div>
        <div class="panel-body">
          {f'<div class="alert">{esc(error)}</div>' if error else ''}
          <div class="grid">
            <div class="field">
              <label for="PUBLIC_DOMAIN">Public domain</label>
              <input id="PUBLIC_DOMAIN" name="PUBLIC_DOMAIN" required placeholder="mypaas.example.com" value="{esc(domain)}">
              <span class="hint">Use the dashboard hostname, without https://.</span>
            </div>
            <div class="field">
              <label for="OWNER_EMAIL">Owner GitHub primary email</label>
              <input id="OWNER_EMAIL" name="OWNER_EMAIL" required placeholder="you@example.com" value="{esc(values.get("OWNER_EMAIL", ""))}">
              <span class="hint">This email is whitelisted as the first MyPaas owner.</span>
            </div>
            <div class="field">
              <label for="GITHUB_CLIENT_ID">GitHub OAuth Client ID</label>
              <input id="GITHUB_CLIENT_ID" name="GITHUB_CLIENT_ID" required value="{esc(values.get("GITHUB_CLIENT_ID", ""))}">
            </div>
            <div class="field">
              <label for="GITHUB_CLIENT_SECRET">GitHub OAuth Client Secret</label>
              <input id="GITHUB_CLIENT_SECRET" name="GITHUB_CLIENT_SECRET" required value="{esc(values.get("GITHUB_CLIENT_SECRET", ""))}">
            </div>
            <div class="field full">
              <label for="GITHUB_CALLBACK_URL">GitHub OAuth callback URL</label>
              <input id="GITHUB_CALLBACK_URL" name="GITHUB_CALLBACK_URL" required value="{esc(callback)}">
              <span class="hint">Must match the callback URL in the GitHub OAuth app exactly.</span>
            </div>
            <div class="field full">
              <label for="CLOUDFLARE_TUNNEL_TOKEN">Cloudflare Tunnel token</label>
              <input id="CLOUDFLARE_TUNNEL_TOKEN" name="CLOUDFLARE_TUNNEL_TOKEN" required value="{esc(values.get("CLOUDFLARE_TUNNEL_TOKEN", ""))}">
              <span class="hint">Use a Cloudflare Zero Trust tunnel token, not an API token.</span>
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
        <div class="actions">
          <button type="submit">Save .env and continue install</button>
        </div>
      </form>

      <aside class="stack">
        <section class="panel">
          <div class="panel-header"><h2>How to get GitHub OAuth credentials</h2></div>
          <div class="panel-body">
            <ol>
              <li>Open <a href="https://github.com/settings/developers" target="_blank" rel="noopener">GitHub Developer settings</a>.</li>
              <li>Choose <strong>OAuth Apps</strong>, then <strong>New OAuth App</strong>.</li>
              <li>Set Homepage URL to <code>https://your-domain</code>.</li>
              <li>Set Authorization callback URL to <code>https://your-domain/api/auth/github/callback</code>.</li>
              <li>After create, copy the Client ID and generate a Client Secret.</li>
            </ol>
          </div>
        </section>
        <section class="panel">
          <div class="panel-header"><h2>How to get the Cloudflare Tunnel token</h2></div>
          <div class="panel-body">
            <ol>
              <li>Open Cloudflare Zero Trust.</li>
              <li>Go to <strong>Networks</strong> -> <strong>Tunnels</strong>.</li>
              <li>Create or open a tunnel, choose the Docker connector, and copy the token from the run command.</li>
              <li>Add public hostnames for <code>your-domain</code> and <code>*.your-domain</code> to route to <code>http://caddy:80</code>.</li>
            </ol>
          </div>
        </section>
        <div class="notice">For remote VMs, keep this wizard on <code>127.0.0.1</code> and access it through SSH port forwarding.</div>
      </aside>
    </div>
  </main>
  <script>
    const domain = document.getElementById('PUBLIC_DOMAIN');
    const callback = document.getElementById('GITHUB_CALLBACK_URL');
    let callbackTouched = Boolean(callback.value);
    callback.addEventListener('input', () => callbackTouched = true);
    domain.addEventListener('input', () => {{
      if (callbackTouched) return;
      const clean = domain.value.trim().replace(/^https?:\\/\\//i, '').replace(/\\/.*$/, '').replace(/\\.$/, '');
      callback.value = clean ? `https://${{clean}}/api/auth/github/callback` : '';
    }});
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
