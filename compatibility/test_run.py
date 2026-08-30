import importlib.util
import json
import pathlib
import sys
import unittest

MODULE_PATH = pathlib.Path(__file__).with_name("run.py")
spec = importlib.util.spec_from_file_location("compatibility_run", MODULE_PATH)
runner = importlib.util.module_from_spec(spec)
assert spec.loader is not None
sys.modules[spec.name] = runner
spec.loader.exec_module(runner)


class CompatibilitySuiteTests(unittest.TestCase):
    def test_repository_catalog_validates_without_compose_execution(self):
        catalog = runner.load_catalog()
        self.assertEqual([], runner.validate_catalog(catalog, compose=False))
        self.assertGreaterEqual(len(catalog["applications"]), 10)

    def test_default_selection_excludes_heavy_but_includes_minio_route_candidate(self):
        catalog = runner.load_catalog()
        selected = runner.selected_apps(catalog, [], include_heavy=False, include_blocked=False)
        ids = {app["id"] for app in selected}
        self.assertNotIn("immich", ids)
        self.assertNotIn("appsmith", ids)
        self.assertIn("minio", ids)
        self.assertIn("excalidraw", ids)
        self.assertIn("n8n", ids)

    def test_explicit_heavy_selection_can_be_enabled(self):
        catalog = runner.load_catalog()
        selected = runner.selected_apps(catalog, ["immich"], include_heavy=True, include_blocked=False)
        self.assertEqual(["immich"], [app["id"] for app in selected])

    def test_unknown_id_is_rejected(self):
        catalog = runner.load_catalog()
        with self.assertRaises(runner.SuiteError):
            runner.selected_apps(catalog, ["does-not-exist"], False, False)

    def test_project_payload_uses_core_repo_for_local_manifests(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "n8n")
        payload = runner.project_payload(catalog, app, "compat-n8n-test")
        self.assertEqual("git", payload["sourceType"])
        self.assertEqual("compose", payload["deployMode"])
        self.assertEqual("https://github.com/nabilrn/MyPaas.git", payload["repoUrl"])
        self.assertEqual("main", payload["branch"])
        self.assertEqual("compatibility/manifests/n8n", payload["baseDirectory"])

    def test_project_payload_can_qualify_core_repo_candidate_branch(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "minio")
        payload = runner.project_payload(catalog, app, "compat-minio-test", "core/compose-http-routes")
        self.assertEqual("core/compose-http-routes", payload["branch"])
        external = next(item for item in catalog["applications"] if item["id"] == "drawdb")
        external_payload = runner.project_payload(catalog, external, "compat-drawdb-test", "core/compose-http-routes")
        self.assertEqual("main", external_payload["branch"])

    def test_registry_payload_does_not_require_repository(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "excalidraw")
        payload = runner.project_payload(catalog, app, "compat-excalidraw")
        self.assertEqual("registry", payload["sourceType"])
        self.assertEqual("", payload["repoUrl"])
        self.assertEqual("image", payload["deployMode"])
        self.assertTrue(payload["imageRef"])

    def test_repaired_template_smoke_contracts_are_explicit(self):
        catalog = runner.load_catalog()
        by_id = {app["id"]: app for app in catalog["applications"]}
        self.assertEqual("/server/ping", by_id["directus"]["execution"]["smokePath"])
        self.assertEqual("/healthz", by_id["n8n"]["execution"]["smokePath"])
        self.assertEqual("/ghost/", by_id["ghost"]["execution"]["smokePath"])
        self.assertEqual("/", by_id["paperless-ngx"]["execution"]["smokePath"])
        self.assertEqual("/", by_id["openclaw"]["execution"]["smokePath"])

    def test_repaired_product_and_qualification_manifests_lock_same_runtime_contracts(self):
        expected_snippets = {
            "directus": ["/server/ping", "start_period: 20s"],
            "n8n": ["docker.io/n8nio/n8n:2.36.7", "/healthz", "start_period: 20s"],
            "ghost": ["ghost:6-alpine", "mysql:8.0.44", "/ghost/", "start_period: 30s"],
            "paperless-ngx": [
                "PAPERLESS_DBENGINE: postgresql",
                "PAPERLESS_BIND_ADDR: 0.0.0.0",
                "http://localhost:8000/",
                "start_period: 30s",
            ],
            "openclaw": [
                "openclaw-bootstrap:",
                "condition: service_completed_successfully",
                "setup\", \"--baseline",
                "dist/docker-healthcheck.js",
            ],
            "umami": ["ghcr.io/umami-software/umami:3.3.1", "/api/heartbeat", "start_period: 30s"],
        }

        for app_id, snippets in expected_snippets.items():
            for root in ("compatibility/manifests", "templates/manifests"):
                path = runner.ROOT / root / app_id / "compose.yml"
                text = path.read_text(encoding="utf-8")
                for snippet in snippets:
                    self.assertIn(snippet, text, f"{path.relative_to(runner.ROOT)} missing {snippet!r}")

    def test_openclaw_bootstrap_resource_is_declared(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "openclaw")
        self.assertEqual(
            {"memoryLimitMb": 512, "cpuLimit": 0.5},
            app["execution"]["serviceResources"]["openclaw-bootstrap"],
        )

    def test_minio_declares_console_route_and_route_smoke(self):
        catalog = runner.load_catalog()
        app = next(item for item in catalog["applications"] if item["id"] == "minio")
        execution = runner.merged_execution(catalog, app)
        self.assertEqual(
            [{"name": "console", "service": "minio", "containerPort": 9001}],
            execution["additionalRoutes"],
        )
        self.assertEqual("console", execution["routeSmoke"][0]["route"])
        self.assertEqual("/", execution["routeSmoke"][0]["path"])

    def test_invalid_route_contract_is_reported(self):
        catalog = runner.load_catalog()
        clone = json.loads(json.dumps(catalog))
        minio = next(item for item in clone["applications"] if item["id"] == "minio")
        minio["execution"]["additionalRoutes"] = [
            {"name": "Console UI", "service": "minio", "containerPort": 9001},
            {"name": "console", "service": "", "containerPort": 70000},
        ]
        errors = runner.validate_catalog(clone)
        self.assertTrue(any("additionalRoutes[0].name is invalid" in error for error in errors))
        self.assertTrue(any("additionalRoutes[1].service is required" in error for error in errors))
        self.assertTrue(any("additionalRoutes[1].containerPort must be 1..65535" in error for error in errors))

    def test_invalid_duplicate_catalog_is_reported(self):
        catalog = runner.load_catalog()
        clone = json.loads(json.dumps(catalog))
        clone["applications"].append(clone["applications"][0])
        errors = runner.validate_catalog(clone)
        self.assertTrue(any("duplicate application id" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
