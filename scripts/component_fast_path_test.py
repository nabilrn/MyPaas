from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]


class ComponentFastPathTests(unittest.TestCase):
    def text(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_frontend_only_ci_skips_heavy_jobs(self):
        workflow = self.text(".github/workflows/ci.yml")
        self.assertIn("name: Detect change scope", workflow)
        self.assertIn("frontend_only: ${{ steps.scope.outputs.frontend_only }}", workflow)
        self.assertGreaterEqual(
            workflow.count("if: needs.changes.outputs.frontend_only != 'true'"),
            3,
        )
        self.assertIn("name: Frontend checks", workflow)

    def test_frontend_only_publish_skips_api_build(self):
        workflow = self.text(".github/workflows/docker-publish.yml")
        self.assertIn("name: Detect frontend-only release", workflow)
        self.assertIn("if: steps.scope.outputs.frontend_only != 'true'", workflow)
        self.assertIn("name: Build and push Dashboard image", workflow)
        self.assertIn("mypaas-dashboard:${{ steps.release.outputs.sha }}", workflow)

    def test_dispatch_uses_dashboard_only_update_for_frontend_scope(self):
        dispatch = self.text("scripts/update-dispatch.sh")
        self.assertIn('if [[ "$path" != frontend/* ]]', dispatch)
        self.assertIn("Frontend-only update detected; using dashboard fast path", dispatch)
        self.assertIn('bash "$ROOT_DIR/scripts/update-dashboard.sh"', dispatch)
        self.assertIn('bash "$ROOT_DIR/scripts/update-vm.sh"', dispatch)

    def test_dashboard_updater_does_not_restart_dependencies(self):
        updater = self.text("scripts/update-dashboard.sh")
        self.assertIn("up -d --no-deps dashboard", updater)
        self.assertIn("Recreating dashboard only", updater)
        self.assertIn("restoring previous dashboard image", updater)
        self.assertNotIn(" up -d api", updater)
        self.assertNotIn(" migrate/migrate", updater)

    def test_systemd_update_service_uses_dispatcher(self):
        configure = self.text("scripts/configure-auto-update.sh")
        self.assertIn('ExecStart=/usr/bin/env bash "$root_q/scripts/update-dispatch.sh"', configure)
        self.assertIn("AUTO_UPDATE_INTERVAL_MINUTES 5", configure)
        self.assertIn("RandomizedDelaySec=30s", configure)


if __name__ == "__main__":
    unittest.main()
