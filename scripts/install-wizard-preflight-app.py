#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
import os
from pathlib import Path
from urllib.parse import urlparse

from install_wizard_preflight import (
    PreflightError,
    check_cloudflare_tunnel,
    check_domain,
    check_github_oauth,
    extract_cloudflare_tunnel_token,
)
from install_wizard_visual import apply_visual_contract


SCRIPT_DIR = Path(__file__).resolve().parent
BASE_SCRIPT = Path(os.environ.get("WIZARD_BASE_SCRIPT", SCRIPT_DIR / "install-wizard.py")).resolve()
MAX_PREFLIGHT_BODY_BYTES = 64 * 1024


def _load_base_module():
    spec = importlib.util.spec_from_file_location("mypaas_install_wizard_base", BASE_SCRIPT)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load install wizard from {BASE_SCRIPT}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


BASE = _load_base_module()
ORIGINAL_FORM_HTML = BASE.form_html
ORIGINAL_BUILD_ENV = BASE.build_env


def _inject_into_step(document: str, step: int, fragment: str) -> str:
    marker = f'<section class="wizard-step" data-step="{step}"'
    start = document.find(marker)
    if start < 0:
        raise RuntimeError(f"install wizard step {step} was not found")
    end = document.find("</section>", start)
    if end < 0:
        raise RuntimeError(f"install wizard step {step} is not closed")
    return document[:end] + fragment + document[end:]


def form_html(error: str = "", values: dict[str, str] | None = None) -> bytes:
    document = ORIGINAL_FORM_HTML(error, values).decode("utf-8")
    preflight_css = """
  .preflight-row { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin-top: 14px; }
  .preflight-row button { min-height: 36px; padding: 0 12px; }
  .preflight-status { min-width: 0; font-size: 12px; line-height: 1.45; color: var(--app-subtle); }
  .preflight-status[data-state="checking"] { color: var(--app-muted); }
  .preflight-status[data-state="ok"] { color: #047857; }
  .preflight-status[data-state="warning"] { color: var(--app-warning); }
  .preflight-status[data-state="error"] { color: var(--app-danger); }
  :root.dark .preflight-status[data-state="ok"] { color: #34d399; }
  @media (prefers-color-scheme: dark) { :root:not(.light) .preflight-status[data-state="ok"] { color: #34d399; } }
"""
    document = document.replace("  </style>\n</head>", preflight_css + "  </style>\n</head>", 1)

    document = _inject_into_step(
        document,
        1,
        """
            <div class="preflight-row">
              <button type="button" class="secondary" id="check-domain-button">Check domain</button>
              <span class="preflight-status" id="domain-preflight-status" aria-live="polite"></span>
            </div>
""",
    )
    document = _inject_into_step(
        document,
        2,
        """
            <div class="preflight-row">
              <button type="button" class="secondary" id="check-github-button">Test GitHub configuration</button>
              <span class="preflight-status" id="github-preflight-status" aria-live="polite"></span>
            </div>
""",
    )
    document = _inject_into_step(
        document,
        3,
        """
            <div class="preflight-row">
              <button type="button" class="secondary" id="check-cloudflare-button">Test tunnel token</button>
              <span class="preflight-status" id="cloudflare-preflight-status" aria-live="polite"></span>
            </div>
""",
    )

    preflight_script = r"""
  <script>
    (() => {
      const wizardToken = document.querySelector('input[name="token"]')?.value || '';

      function statusNode(id, state, message) {
        const node = document.getElementById(id);
        if (!node) return;
        node.dataset.state = state;
        node.textContent = message;
      }

      async function postPreflight(kind, payload) {
        const response = await fetch(`/preflight/${kind}`, {
          method: 'POST',
          cache: 'no-store',
          credentials: 'same-origin',
          headers: {
            'Content-Type': 'application/json',
            'X-Wizard-Token': wizardToken
          },
          body: JSON.stringify(payload)
        });
        let result;
        try {
          result = await response.json();
        } catch {
          throw new Error(`Preflight returned ${response.status}`);
        }
        if (!response.ok || !result.ok) throw new Error(result.message || `Preflight returned ${response.status}`);
        return result;
      }

      async function withButton(button, statusId, task) {
        if (!button || button.disabled) return;
        button.disabled = true;
        statusNode(statusId, 'checking', 'Checking…');
        try {
          await task();
        } catch (error) {
          statusNode(statusId, 'error', error instanceof Error ? error.message : 'Check failed');
        } finally {
          button.disabled = false;
        }
      }

      const domainButton = document.getElementById('check-domain-button');
      domainButton?.addEventListener('click', () => withButton(domainButton, 'domain-preflight-status', async () => {
        const hostname = document.getElementById('PUBLIC_DOMAIN')?.value || '';
        const result = await postPreflight('domain', { hostname });
        statusNode(
          'domain-preflight-status',
          result.wildcardResolved ? 'ok' : 'warning',
          result.message
        );
      }));

      const githubButton = document.getElementById('check-github-button');
      githubButton?.addEventListener('click', () => withButton(githubButton, 'github-preflight-status', async () => {
        const result = await postPreflight('github', {
          clientId: document.getElementById('GITHUB_CLIENT_ID')?.value || '',
          clientSecret: document.getElementById('GITHUB_CLIENT_SECRET')?.value || '',
          callbackUrl: document.getElementById('GITHUB_CALLBACK_URL')?.value || ''
        });
        statusNode('github-preflight-status', 'ok', result.message);
      }));

      const cloudflareButton = document.getElementById('check-cloudflare-button');
      cloudflareButton?.addEventListener('click', () => withButton(cloudflareButton, 'cloudflare-preflight-status', async () => {
        const tokenInput = document.getElementById('CLOUDFLARE_TUNNEL_TOKEN');
        const rawValue = tokenInput?.value || '';
        const result = await postPreflight('cloudflare', { token: rawValue });
        const detected = rawValue.match(/(?:^|\s)(eyJ[A-Za-z0-9._=-]{20,})(?=\s|$)/)?.[1]
          || rawValue.match(/(eyJ[A-Za-z0-9._=-]{20,})/)?.[1];
        if (tokenInput && detected) tokenInput.value = detected;
        statusNode('cloudflare-preflight-status', 'ok', result.message);
      }));
    })();
  </script>
"""
    document = document.replace("</body>", preflight_script + "</body>", 1)
    document = apply_visual_contract(document)
    return document.encode("utf-8")


def build_env(values: dict[str, str]) -> str:
    normalized = dict(values)
    raw_tunnel_value = normalized.get("CLOUDFLARE_TUNNEL_TOKEN", "").strip()
    if raw_tunnel_value:
        try:
            normalized["CLOUDFLARE_TUNNEL_TOKEN"] = extract_cloudflare_tunnel_token(raw_tunnel_value)
        except PreflightError as exc:
            # Preserve legacy opaque tokens that were already accepted by the installer.
            # A pasted command necessarily contains whitespace, so fail closed for malformed commands.
            if any(char.isspace() for char in raw_tunnel_value):
                raise ValueError(str(exc)) from exc
    return ORIGINAL_BUILD_ENV(normalized)


BASE.form_html = form_html
BASE.build_env = build_env


class Handler(BASE.Handler):
    def send_json(self, payload: dict[str, object], status: int = 200) -> None:
        body = json.dumps(payload, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Cache-Control", "no-store")
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def read_json(self) -> dict[str, object]:
        raw_length = self.headers.get("Content-Length", "")
        try:
            length = int(raw_length)
        except ValueError as exc:
            raise PreflightError("Invalid preflight request") from exc
        if length <= 0 or length > MAX_PREFLIGHT_BODY_BYTES:
            raise PreflightError("Invalid preflight request size")
        try:
            payload = json.loads(self.rfile.read(length).decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError) as exc:
            raise PreflightError("Invalid preflight request") from exc
        if not isinstance(payload, dict):
            raise PreflightError("Invalid preflight request")
        return payload

    def do_POST(self) -> None:
        path = urlparse(self.path).path
        if path not in {"/preflight/domain", "/preflight/github", "/preflight/cloudflare"}:
            super().do_POST()
            return

        if not BASE.token_matches(self.headers.get("X-Wizard-Token"), BASE.TOKEN):
            self.send_json({"ok": False, "message": "Invalid wizard token."}, 403)
            return

        try:
            payload = self.read_json()
            if path == "/preflight/domain":
                result = check_domain(str(payload.get("hostname", "")))
            elif path == "/preflight/github":
                result = check_github_oauth(
                    str(payload.get("clientId", "")),
                    str(payload.get("clientSecret", "")),
                    str(payload.get("callbackUrl", "")),
                )
            else:
                result = check_cloudflare_tunnel(str(payload.get("token", "")))
        except PreflightError as exc:
            self.send_json({"ok": False, "message": str(exc)}, 400)
            return
        except Exception as exc:
            # Do not include exception text: network/container errors can contain credentials.
            print(f"installer preflight failed: {type(exc).__name__}")
            self.send_json({"ok": False, "message": "Preflight check failed unexpectedly."}, 500)
            return

        self.send_json(result)


def main() -> None:
    server = BASE.HTTPServer((BASE.HOST, BASE.PORT), Handler)
    server.serve_forever()


if __name__ == "__main__":
    main()
