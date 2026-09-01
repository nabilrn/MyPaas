import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class UpdateTriggerContractTest(unittest.TestCase):
    def test_manual_update_path_is_always_installed(self) -> None:
        configure = (ROOT / "scripts" / "configure-auto-update.sh").read_text(encoding="utf-8")

        self.assertIn('PATH_FILE="/etc/systemd/system/mypaas-update.path"', configure)
        self.assertIn("PathExists=$REQUEST_FILE", configure)
        self.assertIn("systemctl enable --now mypaas-update.path", configure)
        self.assertIn("dashboard-triggered updates remain available", configure)

    def test_deploy_reconciles_host_update_units(self) -> None:
        deploy = (ROOT / "scripts" / "deploy-to-vm.sh").read_text(encoding="utf-8")

        self.assertIn('bash "$ROOT_DIR/scripts/configure-auto-update.sh"', deploy)

    def test_uninstall_removes_host_update_units_and_policy(self) -> None:
        uninstall = (ROOT / "scripts" / "uninstall-vm.sh").read_text(encoding="utf-8")

        for unit in ("mypaas-update.service", "mypaas-update.timer", "mypaas-update.path"):
            self.assertIn(unit, uninstall)
        self.assertIn("systemctl stop mypaas-update.timer mypaas-update.path mypaas-update.service", uninstall)
        self.assertIn("/etc/mypaas/update.env", uninstall)
        self.assertIn("/run/mypaas/update.request", uninstall)


if __name__ == "__main__":
    unittest.main()
