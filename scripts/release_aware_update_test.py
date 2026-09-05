from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]


class ReleaseAwareUpdateContractTests(unittest.TestCase):
    def text(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_stable_release_channel_is_default(self):
        env = self.text(".env.example")
        self.assertIn("AUTO_UPDATE_CHANNEL=release", env)
        self.assertIn("AUTO_UPDATE_INCLUDE_PRERELEASES=false", env)
        self.assertIn("AUTO_UPDATE_REF=main", env)

    def test_update_policy_persists_release_channel(self):
        configure = self.text("scripts/configure-auto-update.sh")
        self.assertIn("AUTO_UPDATE_CHANNEL=$channel", configure)
        self.assertIn("AUTO_UPDATE_INCLUDE_PRERELEASES=$include_prereleases", configure)
        self.assertIn('STATUS_DIR="/run/mypaas/update"', configure)
        self.assertIn('install -d -m 0755 "$STATUS_DIR"', configure)

    def test_dispatcher_resolves_release_tags_but_keeps_main_as_dev_channel(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('CHANNEL="${AUTO_UPDATE_CHANNEL:-release}"', dispatch)
        self.assertIn('https://github.com/$RELEASE_REPOSITORY/releases/latest', dispatch)
        self.assertIn('TARGET_REF="refs/tags/$tag"', dispatch)
        self.assertIn('main)', dispatch)
        self.assertIn('TARGET_REF="$REF"', dispatch)

    def test_status_helper_is_atomic_and_phase_aware(self):
        helper = self.text("scripts/update-status.sh")
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('status_parent="$(dirname "$status_file")"', helper)
        self.assertIn('mktemp "$status_parent/.status.XXXXXX"', helper)
        self.assertIn('mv -f "$tmp" "$status_file"', helper)
        self.assertIn("printf 'phase=%s\\n'", helper)
        for state in ("succeeded", "failed", "rolled_back", "blocked"):
            self.assertIn(f"write_status {state}", dispatch)

    def test_dispatcher_reports_real_update_phases(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        dashboard = self.text("scripts/update-dashboard.sh")
        full = self.text("scripts/update-vm.sh")
        for phase in ("resolving_release", "validating_target", "checking_images", "preflight", "applying", "rolling_back", "complete"):
            self.assertIn(phase, dispatch)
        self.assertIn('status_phase updating verifying "Verifying the updated MyPaas control plane"', full)
        self.assertIn('status_phase updating verifying "Verifying the updated MyPaas dashboard"', dashboard)

    def test_dispatcher_preserves_child_phase_and_requires_checkout_advance(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn("latest_status_phase()", dispatch)
        self.assertIn('actual_sha="$(git_repo rev-parse HEAD)"', dispatch)
        self.assertIn('if [[ "$actual_sha" != "$TARGET_SHA" ]]', dispatch)
        self.assertIn("Updater left the installation unchanged", dispatch)
        self.assertIn('write_status failed "$(latest_status_phase)"', dispatch)

    def test_dispatcher_expands_shallow_history_before_ancestry_guard(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('git_repo rev-parse --is-shallow-repository', dispatch)
        self.assertIn('git_repo fetch --unshallow "$REMOTE"', dispatch)
        self.assertLess(dispatch.index("ensure_complete_history"), dispatch.index('git_repo merge-base --is-ancestor "$CURRENT_SHA" "$TARGET_SHA"'))

    def test_dispatcher_refuses_non_descendant_release(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('git_repo merge-base --is-ancestor "$CURRENT_SHA" "$TARGET_SHA"', dispatch)
        self.assertIn("Refusing to move to a target that is not a descendant", dispatch)

    def test_frontend_fast_path_uses_exact_dashboard_sha(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('target_dashboard="$DASHBOARD_IMAGE_REPO:$TARGET_SHA"', dispatch)
        self.assertIn('MYPAAS_DASHBOARD_IMAGE_TAG="$TARGET_SHA"', dispatch)
        self.assertIn('bash "$ROOT_DIR/scripts/update-dashboard.sh"', dispatch)

    def test_lock_contention_preserves_active_updater_status(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        lock_block = dispatch.split('if ! mkdir "$LOCK_DIR"', 1)[1].split('LOCK_HELD=true', 1)[0]
        self.assertIn("Another MyPaas update is already running; skipping", lock_block)
        self.assertNotIn("write_status", lock_block)

    def test_dashboard_can_only_read_dedicated_update_status_mount(self):
        compose = self.text("docker-compose.prod.yml")
        dashboard = compose.split("  dashboard:\n", 1)[1].split("\n  caddy:\n", 1)[0]
        self.assertIn("/run/mypaas/update:/run/mypaas/update:ro", dashboard)
        self.assertNotIn("/run/mypaas:/run/mypaas", dashboard)
        self.assertIn("MYPAAS_UPDATE_STATUS_FILE: /run/mypaas/update/status", dashboard)
        self.assertIn("/etc/mypaas/update.env:/etc/mypaas/update.env:ro", dashboard)
        self.assertNotIn("/etc:/etc", dashboard)

    def test_release_status_route_is_owner_authenticated_and_descendant_aware(self):
        route = self.text("frontend/src/routes/internal/system-update/+server.ts")
        header = self.text("frontend/src/lib/components/AppHeader.svelte")
        self.assertIn("apiRequest('/auth/me', cookie)", route)
        self.assertIn("user?.role !== 'owner'", route)
        self.assertIn("/run/mypaas/update/status", route)
        self.assertIn("/etc/mypaas/update.env", route)
        self.assertIn("AUTO_UPDATE_INCLUDE_PRERELEASES", route)
        self.assertIn("api.github.com/repos/nabilrn/MyPaas/releases", route)
        self.assertIn("api.github.com/repos/nabilrn/MyPaas/compare/", route)
        self.assertIn("comparison.status === 'ahead'", route)
        self.assertIn("GITHUB_CACHE_TTL_MS = 5 * 60_000", route)
        self.assertIn("redirect: 'error'", route)
        self.assertIn("phaseCandidate", route)
        self.assertIn("<ReleaseNotification enabled={user?.role === 'owner'} />", header)

    def test_system_update_page_is_server_rendered_and_survives_restart(self):
        server = self.text("frontend/src/routes/admin/update/+page.server.ts")
        page = self.text("frontend/src/routes/admin/update/+page.svelte")
        bell = self.text("frontend/src/lib/components/ReleaseNotification.svelte")
        self.assertIn("fetch('/internal/system-update'", server)
        self.assertIn("Update progress", page)
        self.assertIn("queuedLocally", page)
        self.assertIn("connectionLost", page)
        self.assertIn("window.location.reload()", page)
        self.assertIn("pollingFast ? 1000 : 30_000", page)
        self.assertIn("baselineStatusKey", page)
        self.assertIn("statusAdvanced", page)
        self.assertIn("statusTargetsAvailableRelease", page)
        self.assertIn("release?.available && !busy && !statusTargetsAvailableRelease", page)
        self.assertIn("goto('/admin/update')", bell)
        self.assertNotIn("api.admin.triggerUpdate", bell)

    def test_release_publish_targets_qualified_main_images(self):
        workflow = self.text(".github/workflows/release-publish.yml")
        self.assertIn("RELEASE_SOURCE_SHA: ${{ github.event.pull_request.base.sha }}", workflow)
        self.assertIn("Release request may only change", workflow)
        self.assertIn('docker manifest inspect "ghcr.io/nabilrn/mypaas-api:$TARGET_SHA"', workflow)
        self.assertIn('docker manifest inspect "ghcr.io/nabilrn/mypaas-dashboard:$TARGET_SHA"', workflow)
        self.assertIn('--target "$TARGET_SHA"', workflow)


if __name__ == "__main__":
    unittest.main()
