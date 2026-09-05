#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import hmac
import importlib.util
import io
import json
import os
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from install_wizard_preflight import (
    PreflightError,
    check_cloudflare_tunnel,
    check_domain,
    check_github_oauth,
    extract_cloudflare_tunnel_token,
    validate_hostname,
    validate_https_callback,
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

# The installer is intentionally a single-process, single-setup-token workflow.
# Keep only keyed fingerprints of successful live checks in memory so OAuth and
# tunnel secrets are never retained as plaintext qualification state.
PREFLIGHT_STATE: dict[str, dict[str, object]] = {}


def _fingerprint(kind: str, *parts: str) -> str:
    material = "\0".join([kind, *parts]).encode("utf-8")
    return hmac.new(BASE.TOKEN.encode("utf-8"), material, hashlib.sha256).hexdigest()


def _domain_fingerprint(hostname: str) -> str:
    return _fingerprint("domain", validate_hostname(hostname))


def _github_fingerprint(client_id: str, client_secret: str, callback_url: str) -> str:
    client_id = client_id.strip()
    client_secret = client_secret.strip()
    callback = validate_https_callback(callback_url)
    return _fingerprint("github", client_id, client_secret, callback)


def _cloudflare_fingerprint(token_or_command: str) -> str:
    token = extract_cloudflare_tunnel_token(token_or_command)
    return _fingerprint("cloudflare", token)


def remember_domain_preflight(hostname: str, *, wildcard_resolved: bool) -> None:
    PREFLIGHT_STATE["domain"] = {
        "fingerprint": _domain_fingerprint(hostname),
        "wildcardResolved": bool(wildcard_resolved),
    }


def remember_github_preflight(client_id: str, client_secret: str, callback_url: str) -> None:
    PREFLIGHT_STATE["github"] = {
        "fingerprint": _github_fingerprint(client_id, client_secret, callback_url),
    }


def remember_cloudflare_preflight(token_or_command: str) -> None:
    PREFLIGHT_STATE["cloudflare"] = {
        "fingerprint": _cloudflare_fingerprint(token_or_command),
    }


def validate_save_preflights(values: dict[str, str]) -> None:
    try:
        expected_domain = _domain_fingerprint(values.get("PUBLIC_DOMAIN", ""))
        expected_github = _github_fingerprint(
            values.get("GITHUB_CLIENT_ID", ""),
            values.get("GITHUB_CLIENT_SECRET", ""),
            values.get("GITHUB_CALLBACK_URL", ""),
        )
        expected_cloudflare = _cloudflare_fingerprint(values.get("CLOUDFLARE_TUNNEL_TOKEN", ""))
    except PreflightError as exc:
        raise ValueError(str(exc)) from exc

    domain_state = PREFLIGHT_STATE.get("domain")
    if not domain_state or not hmac.compare_digest(str(domain_state.get("fingerprint", "")), expected_domain):
        raise ValueError("Domain settings changed after preflight. Go back and run the domain check again.")
    if not domain_state.get("wildcardResolved"):
        raise ValueError("Project wildcard DNS is not verified yet. Complete the Cloudflare routing step and retry.")

    github_state = PREFLIGHT_STATE.get("github")
    if not github_state or not hmac.compare_digest(str(github_state.get("fingerprint", "")), expected_github):
        raise ValueError("GitHub OAuth settings changed after preflight. Go back and test the GitHub configuration again.")

    cloudflare_state = PREFLIGHT_STATE.get("cloudflare")
    if not cloudflare_state or not hmac.compare_digest(str(cloudflare_state.get("fingerprint", "")), expected_cloudflare):
        raise ValueError("Cloudflare Tunnel settings changed after preflight. Go back and test the tunnel token again.")


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
  .preflight-status[data-state="stale"] { color: var(--app-subtle); }
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
              <button type="button" class="secondary" id="check-cloudflare-button">Test tunnel and routing</button>
              <span class="preflight-status" id="cloudflare-preflight-status" aria-live="polite"></span>
            </div>
""",
    )

    preflight_script = r"""
  <script>
    (() => {
      // Installer preflight gate contract: Continue performs the live check for
      // the current setup step and only advances after that exact input passes.
      const wizardToken = document.querySelector('input[name="token"]')?.value || '';
      const nextButton = document.getElementById('next-button');
      const domainButton = document.getElementById('check-domain-button');
      const githubButton = document.getElementById('check-github-button');
      const cloudflareButton = document.getElementById('check-cloudflare-button');
      const domainInput = document.getElementById('PUBLIC_DOMAIN');
      const githubClientId = document.getElementById('GITHUB_CLIENT_ID');
      const githubClientSecret = document.getElementById('GITHUB_CLIENT_SECRET');
      const githubCallback = document.getElementById('GITHUB_CALLBACK_URL');
      const cloudflareInput = document.getElementById('CLOUDFLARE_TUNNEL_TOKEN');
      let gateBusy = false;

      function statusNode(id, state, message) {
        const node = document.getElementById(id);
        if (!node) return;
        node.dataset.state = state;
        node.textContent = message;
      }

      function stale(statusId, message) {
        const node = document.getElementById(statusId);
        if (!node || !node.textContent) return;
        statusNode(statusId, 'stale', message);
      }

      function currentVisibleStep() {
        return Array.from(document.querySelectorAll('.wizard-step')).findIndex((step) => !step.hidden);
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

      async function runDomainCheck({ requireWildcard = false } = {}) {
        statusNode('domain-preflight-status', 'checking', requireWildcard ? 'Rechecking domain and project wildcard…' : 'Checking domain…');
        try {
          const result = await postPreflight('domain', { hostname: domainInput?.value || '' });
          if (requireWildcard && !result.wildcardResolved) {
            statusNode('domain-preflight-status', 'warning', result.message);
            statusNode('cloudflare-preflight-status', 'error', 'Tunnel token is valid, but project wildcard DNS is not visible yet. Add the wildcard route/DNS, then retry.');
            return false;
          }
          statusNode(
            'domain-preflight-status',
            result.wildcardResolved ? 'ok' : 'warning',
            result.message
          );
          return true;
        } catch (error) {
          statusNode('domain-preflight-status', 'error', error instanceof Error ? error.message : 'Domain check failed');
          if (requireWildcard) statusNode('cloudflare-preflight-status', 'error', 'Routing verification failed. Fix the domain or wildcard DNS, then retry.');
          return false;
        }
      }

      async function runGithubCheck() {
        statusNode('github-preflight-status', 'checking', 'Checking GitHub OAuth configuration…');
        try {
          const result = await postPreflight('github', {
            clientId: githubClientId?.value || '',
            clientSecret: githubClientSecret?.value || '',
            callbackUrl: githubCallback?.value || ''
          });
          statusNode('github-preflight-status', 'ok', result.message);
          return true;
        } catch (error) {
          statusNode('github-preflight-status', 'error', error instanceof Error ? error.message : 'GitHub check failed');
          return false;
        }
      }

      function normalizeTunnelInput(rawValue) {
        return rawValue.match(/(?:^|\s)(eyJ[A-Za-z0-9._=-]{20,})(?=\s|$)/)?.[1]
          || rawValue.match(/(eyJ[A-Za-z0-9._=-]{20,})/)?.[1]
          || '';
      }

      async function runCloudflareCheck({ verifyRouting = false } = {}) {
        statusNode('cloudflare-preflight-status', 'checking', verifyRouting ? 'Checking tunnel token and project routing…' : 'Checking tunnel token…');
        try {
          const rawValue = cloudflareInput?.value || '';
          const result = await postPreflight('cloudflare', { token: rawValue });
          const detected = normalizeTunnelInput(rawValue);
          if (cloudflareInput && detected) cloudflareInput.value = detected;
          statusNode('cloudflare-preflight-status', 'ok', result.message);
          if (!verifyRouting) return true;

          const domainReady = await runDomainCheck({ requireWildcard: true });
          if (!domainReady) return false;
          statusNode('cloudflare-preflight-status', 'ok', 'Tunnel token is valid and project wildcard DNS resolves from this VM.');
          return true;
        } catch (error) {
          statusNode('cloudflare-preflight-status', 'error', error instanceof Error ? error.message : 'Cloudflare check failed');
          return false;
        }
      }

      async function withButton(button, task) {
        if (!button || button.disabled || gateBusy) return;
        gateBusy = true;
        button.disabled = true;
        try {
          await task();
        } finally {
          button.disabled = false;
          gateBusy = false;
        }
      }

      function currentInputsAreValid() {
        const stepIndex = currentVisibleStep();
        const step = document.querySelector(`.wizard-step[data-step="${stepIndex}"]`);
        if (!step) return false;
        const invalid = Array.from(step.querySelectorAll('input[required]')).find((input) => !input.checkValidity());
        if (!invalid) return true;
        invalid.reportValidity();
        return false;
      }

      async function advanceThroughGate(stepIndex) {
        if (!currentInputsAreValid()) return;
        let passed = true;
        if (stepIndex === 1) passed = await runDomainCheck();
        if (stepIndex === 2) passed = await runGithubCheck();
        if (stepIndex === 3) passed = await runCloudflareCheck({ verifyRouting: true });
        if (passed && typeof showStep === 'function') showStep(stepIndex + 1);
      }

      domainButton?.addEventListener('click', () => withButton(domainButton, () => runDomainCheck()));
      githubButton?.addEventListener('click', () => withButton(githubButton, () => runGithubCheck()));
      cloudflareButton?.addEventListener('click', () => withButton(cloudflareButton, () => runCloudflareCheck({ verifyRouting: true })));

      // Capture before the base wizard's ordinary Continue handler. Step 0 remains
      // ungated because backup restore is optional; setup steps 1-3 are qualified.
      nextButton?.addEventListener('click', async (event) => {
        const stepIndex = currentVisibleStep();
        if (![1, 2, 3].includes(stepIndex)) return;
        event.preventDefault();
        event.stopImmediatePropagation();
        if (gateBusy) return;
        gateBusy = true;
        nextButton.disabled = true;
        const originalLabel = nextButton.textContent;
        nextButton.textContent = 'Checking…';
        try {
          await advanceThroughGate(stepIndex);
        } finally {
          nextButton.textContent = originalLabel;
          nextButton.disabled = false;
          gateBusy = false;
        }
      }, true);

      domainInput?.addEventListener('input', () => {
        stale('domain-preflight-status', 'Domain changed. Recheck before continuing.');
        stale('github-preflight-status', 'Callback may have changed. Recheck GitHub before continuing.');
      });
      githubClientId?.addEventListener('input', () => stale('github-preflight-status', 'GitHub settings changed. Recheck before continuing.'));
      githubClientSecret?.addEventListener('input', () => stale('github-preflight-status', 'GitHub settings changed. Recheck before continuing.'));
      githubCallback?.addEventListener('input', () => stale('github-preflight-status', 'GitHub settings changed. Recheck before continuing.'));
      cloudflareInput?.addEventListener('input', () => stale('cloudflare-preflight-status', 'Tunnel settings changed. Recheck before continuing.'));
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

    def _qualify_save_request(self) -> bool:
        raw_length = self.headers.get("Content-Length", "")
        try:
            length = int(raw_length)
        except ValueError:
            return True
        if length <= 0 or length > BASE.MAX_FORM_BYTES:
            return True

        raw = self.rfile.read(length)
        self.rfile = io.BytesIO(raw)
        try:
            data = parse_qs(raw.decode("utf-8"), keep_blank_values=True)
        except UnicodeDecodeError:
            return True

        if not BASE.token_matches(BASE.field(data, "token"), BASE.TOKEN):
            return True

        values = {key: BASE.field(data, key) for key in BASE.DEFAULTS.keys()}
        try:
            validate_save_preflights(values)
        except ValueError as exc:
            self.send_html(BASE.form_html(str(exc), values), 400)
            return False
        return True

    def do_POST(self) -> None:
        path = urlparse(self.path).path
        if path == "/save":
            if not self._qualify_save_request():
                return
            super().do_POST()
            return

        if path not in {"/preflight/domain", "/preflight/github", "/preflight/cloudflare"}:
            super().do_POST()
            return

        if not BASE.token_matches(self.headers.get("X-Wizard-Token"), BASE.TOKEN):
            self.send_json({"ok": False, "message": "Invalid wizard token."}, 403)
            return

        try:
            payload = self.read_json()
            if path == "/preflight/domain":
                hostname = str(payload.get("hostname", ""))
                result = check_domain(hostname)
                remember_domain_preflight(hostname, wildcard_resolved=bool(result.get("wildcardResolved")))
            elif path == "/preflight/github":
                client_id = str(payload.get("clientId", ""))
                client_secret = str(payload.get("clientSecret", ""))
                callback_url = str(payload.get("callbackUrl", ""))
                result = check_github_oauth(client_id, client_secret, callback_url)
                remember_github_preflight(client_id, client_secret, callback_url)
            else:
                token_or_command = str(payload.get("token", ""))
                result = check_cloudflare_tunnel(token_or_command)
                remember_cloudflare_preflight(token_or_command)
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
