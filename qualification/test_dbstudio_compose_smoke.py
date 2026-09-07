import unittest

import dbstudio_compose_smoke


class DBStudioComposeSmokeTest(unittest.TestCase):
    def test_fixture_payload_uses_compose_app_contract(self):
        payload = dbstudio_compose_smoke.fixture_payload(
            "postgres",
            "qualification/fixtures/dbstudio/postgres",
            "beta-db",
            "https://github.com/example/repo",
            "main",
        )
        self.assertEqual(payload["deployMode"], "compose")
        self.assertEqual(payload["branch"], "main")
        self.assertEqual(payload["mainService"], "app")
        self.assertEqual(payload["appPort"], 80)
        self.assertEqual(payload["composeFilePath"], "compose.yml")

    def test_generated_project_names_fit_backend_limit(self):
        prefix = dbstudio_compose_smoke.project_prefix("20260814T143857Z")
        names = [
            dbstudio_compose_smoke.fixture_payload(engine, "fixtures", prefix, "https://github.com/example/repo", "main")["name"]
            for engine in ("postgres", "mysql", "mariadb")
        ]

        self.assertTrue(all(len(name) <= 30 for name in names), names)

    def test_status_requires_connected_expected_driver_and_read_only_default(self):
        status = {
            "configured": True,
            "connected": True,
            "connection": {"driver": "postgres"},
            "writeAccess": None,
        }
        self.assertEqual(dbstudio_compose_smoke.evaluate_status(status, "postgres"), [])

    def test_status_rejects_implicit_write_access(self):
        status = {
            "configured": True,
            "connected": True,
            "connection": {"driver": "mysql"},
            "writeAccess": {"id": "unexpected"},
        }
        failures = dbstudio_compose_smoke.evaluate_status(status, "mysql")
        self.assertTrue(any("writeAccess" in failure for failure in failures))

    def test_status_rejects_wrong_driver(self):
        status = {
            "configured": True,
            "connected": True,
            "connection": {"driver": "mysql"},
            "writeAccess": None,
        }
        failures = dbstudio_compose_smoke.evaluate_status(status, "mariadb")
        self.assertTrue(any("expected 'mariadb'" in failure for failure in failures))

    def test_report_does_not_contain_api_token_field(self):
        result = dbstudio_compose_smoke.EngineResult(
            engine="postgres",
            expected_driver="postgres",
            deployment_status="running",
            configured=True,
            connected=True,
            actual_driver="postgres",
            write_access_absent=True,
            schemas_readable=True,
        )
        report = {
            "runId": "run",
            "gitSha": "abc",
            "fixtureRef": "abc",
            "fixtureBranch": "test/beta-readiness-candidate",
            "fixtureResolvedSha": "abc",
            "pass": True,
            "engines": [result.__dict__],
        }
        text = dbstudio_compose_smoke.markdown(report)
        self.assertNotIn("token", text.lower())
        self.assertIn("Fixture branch", text)
        self.assertIn("postgres", text)


if __name__ == "__main__":
    unittest.main()
