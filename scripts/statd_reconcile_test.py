import unittest
from pathlib import Path


ROOT_DIR = Path(__file__).resolve().parents[1]
RECONCILE = ROOT_DIR / "scripts" / "reconcile-statd.sh"
UPDATER = ROOT_DIR / "scripts" / "update-vm.sh"
INSTALLER = ROOT_DIR / "scripts" / "install-vm.sh"


class StatdReconcileTest(unittest.TestCase):
    def test_reconcile_persists_socket_and_targets_pinned_release(self) -> None:
        reconcile = RECONCILE.read_text(encoding="utf-8")
        installer = INSTALLER.read_text(encoding="utf-8")

        self.assertIn('EXPECTED_STATD_VERSION="${STATD_VERSION:-v0.2.0}"', reconcile)
        self.assertIn('DEFAULT_STATD_SOCKET="/run/mypaas/statd.sock"', reconcile)
        self.assertIn('STATD_SOCKET=$DEFAULT_STATD_SOCKET', reconcile)
        self.assertIn('scripts/install-vm.sh" --statd-only', reconcile)
        self.assertIn('STATD_VERSION="${STATD_VERSION:-v0.2.0}"', installer)
        self.assertIn('systemctl is-active --quiet mypaas-statd', reconcile)

    def test_updater_reconciles_host_dependency_even_when_checkout_is_current(self) -> None:
        updater = UPDATER.read_text(encoding="utf-8")

        current_branch = updater.split('if [[ "$current_sha" == "$target_sha" ]]; then', 1)[1].split("\n  fi\n", 1)[0]
        self.assertIn("reconcile_statd", current_branch)
        self.assertIn("redeploy_current_for_env_drift", current_branch)
        self.assertIn("container_env_value", updater)
        self.assertIn("STATD_SOCKET", updater)

    def test_updater_reconciles_target_checkout_before_deploy(self) -> None:
        updater = UPDATER.read_text(encoding="utf-8")

        reset_index = updater.index('git_repo reset --hard "$target_sha"')
        reconcile_index = updater.index("if ! reconcile_statd; then", reset_index)
        deploy_index = updater.index('bash "$ROOT_DIR/scripts/deploy-to-vm.sh"', reconcile_index)
        self.assertLess(reset_index, reconcile_index)
        self.assertLess(reconcile_index, deploy_index)


if __name__ == "__main__":
    unittest.main()
