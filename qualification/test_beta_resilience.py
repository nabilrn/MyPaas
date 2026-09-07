import hashlib
import hmac
import json
import unittest

import beta_resilience


class BetaResilienceTest(unittest.TestCase):
    def test_fixture_payload_rotates_dockerfile_and_compose(self):
        first = beta_resilience.fixture_payload(0, "run", "https://github.com/example/repo", "main")
        second = beta_resilience.fixture_payload(1, "run", "https://github.com/example/repo", "main")
        self.assertEqual(first["deployMode"], "dockerfile")
        self.assertEqual(second["deployMode"], "compose")
        self.assertEqual(second["mainService"], "app")

    def test_failing_payload_uses_missing_registry_image(self):
        payload = beta_resilience.failing_payload("run", "ABC123")
        self.assertEqual(payload["sourceType"], "registry")
        self.assertIn("intentionally-missing", payload["imageRef"])

    def test_generated_project_names_fit_backend_limit(self):
        prefix = beta_resilience.project_prefix("20260814T143705Z-91ab957ba072")
        names = [
            beta_resilience.fixture_payload(0, prefix, "https://github.com/example/repo", "main")["name"],
            beta_resilience.fixture_payload(1, prefix, "https://github.com/example/repo", "main")["name"],
            beta_resilience.failing_payload(prefix, "20260814T143705Z-91ab957ba072")["name"],
        ]

        self.assertTrue(all(len(name) <= 30 for name in names), names)

    def test_duplicate_ports_are_reported(self):
        rows = [
            beta_resilience.ManagedProject("a", "dockerfile", "1", "a", "", 3001),
            beta_resilience.ManagedProject("b", "compose", "2", "b", "", 3001),
        ]
        self.assertEqual(beta_resilience.duplicate_ports(rows), [3001])

    def test_webhook_signature_matches_body(self):
        body = json.dumps({"ref": "refs/heads/main"}).encode()
        headers = beta_resilience.webhook_headers("fixture-secret", body)
        expected = hmac.new(b"fixture-secret", body, hashlib.sha256).hexdigest()
        self.assertEqual(headers["X-Hub-Signature-256"], "sha256=" + expected)
        self.assertEqual(headers["X-GitHub-Event"], "push")

    def test_evaluate_accepts_expected_success_and_failure(self):
        projects = [
            beta_resilience.ManagedProject("ok", "dockerfile", "1", "ok", "", 3001),
            beta_resilience.ManagedProject("bad", "intentional-failure", "2", "bad", "", 3002, True),
        ]
        deployments = [
            beta_resilience.DeploymentObservation("1", "ok", "initial-deploy", "d1", "running", 1.0),
            beta_resilience.DeploymentObservation("2", "bad", "initial-deploy", "d2", "failed", 1.0),
        ]
        failures = beta_resilience.evaluate(projects, deployments, [{"url": "https://ok", "ok": True}], [])
        self.assertEqual(failures, [])

    def test_evaluate_flags_server_error_read_probe(self):
        projects = [beta_resilience.ManagedProject("ok", "dockerfile", "1", "ok", "", 3001)]
        deployments = [beta_resilience.DeploymentObservation("1", "ok", "redeploy", "d1", "running", 1.0)]
        failures = beta_resilience.evaluate(
            projects,
            deployments,
            [],
            [{"path": "/api/projects/1/metrics", "status": 500, "serverError": True}],
        )
        self.assertTrue(any("read path failed" in failure for failure in failures))


if __name__ == "__main__":
    unittest.main()
