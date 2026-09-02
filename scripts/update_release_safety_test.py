import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class UpdateReleaseSafetyContractTest(unittest.TestCase):
    def text(self, path):
        return (ROOT / path).read_text(encoding="utf-8")

    def test_production_compose_exposes_build_identity(self):
        compose = self.text("docker-compose.prod.yml")
        self.assertIn("MYPAAS_BUILD_SHA: ${MYPAAS_BUILD_SHA:-unknown}", compose)
        self.assertIn("ghcr.io/nabilrn/mypaas-api:${MYPAAS_IMAGE_TAG:-latest}", compose)
        self.assertIn("ghcr.io/nabilrn/mypaas-dashboard:${MYPAAS_IMAGE_TAG:-latest}", compose)

    def test_deploy_supports_local_rollback_without_remote_pull(self):
        deploy = self.text("scripts/deploy-to-vm.sh")
        self.assertIn('SKIP_IMAGE_PULL="${MYPAAS_SKIP_IMAGE_PULL:-false}"', deploy)
        self.assertIn('image inspect "$API_IMAGE_REPO:$MYPAAS_IMAGE_TAG"', deploy)
        self.assertIn('image inspect "$DASHBOARD_IMAGE_REPO:$MYPAAS_IMAGE_TAG"', deploy)
        self.assertIn('if [[ "$SKIP_IMAGE_PULL" != "true" ]]', deploy)

    def test_explicit_image_tag_overrides_env_file_pin(self):
        deploy = self.text("scripts/deploy-to-vm.sh")
        self.assertIn('EXPLICIT_IMAGE_TAG_SET="${MYPAAS_IMAGE_TAG+x}"', deploy)
        self.assertIn('EXPLICIT_IMAGE_TAG="${MYPAAS_IMAGE_TAG:-}"', deploy)
        self.assertIn('MYPAAS_IMAGE_TAG="$EXPLICIT_IMAGE_TAG"', deploy)

    def test_updater_verifies_target_and_restored_runtime(self):
        updater = self.text("scripts/update-vm.sh")
        self.assertIn('verify_stack "$docker_cmd" "$target_sha" "$target_sha"', updater)
        self.assertIn('MYPAAS_SKIP_IMAGE_PULL=true', updater)
        self.assertIn('MYPAAS_BUILD_SHA="$current_sha"', updater)
        self.assertIn('verify_stack "$docker_cmd" "$current_sha" "$rollback_tag"', updater)
        self.assertIn('previous runtime could not be verified after rollback', updater)

    def test_updater_deploys_after_each_checkout_reset(self):
        updater = self.text("scripts/update-vm.sh")
        deploy = self.text("scripts/deploy-to-vm.sh")

        target_reset = updater.index('git_repo reset --hard "$target_sha"')
        target_deploy = updater.index('bash "$ROOT_DIR/scripts/deploy-to-vm.sh"', target_reset)
        rollback_reset = updater.index('git_repo reset --hard "$current_sha"', target_deploy)
        rollback_deploy = updater.index('bash "$ROOT_DIR/scripts/deploy-to-vm.sh"', rollback_reset)

        self.assertLess(target_reset, target_deploy)
        self.assertLess(rollback_reset, rollback_deploy)
        self.assertIn("sh -c 'cat > /tmp/mypaas-Caddyfile.next'", deploy)
        self.assertIn('< "$ROOT_DIR/Caddyfile.prod"', deploy)
        self.assertIn(
            'caddy reload --config /tmp/mypaas-Caddyfile.next --adapter caddyfile',
            deploy,
        )
        self.assertNotIn(
            'caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile',
            deploy,
        )

    def test_updater_surfaces_final_verification_failure(self):
        updater = self.text("scripts/update-vm.sh")
        self.assertIn('verify_log="$(mktemp)"', updater)
        self.assertIn('MYPAAS_IMAGE_TAG="$expected_image_tag" MYPAAS_BUILD_SHA="$expected_build_sha"', updater)
        self.assertIn('Production verification failed after %s attempts. Last verifier output:', updater)
        self.assertIn('cat "$verify_log" >&2', updater)

    def test_updater_skips_migration_helper_when_tree_is_unchanged(self):
        updater = self.text("scripts/update-vm.sh")
        self.assertIn(
            'git_repo diff --quiet "$current_sha" "$target_sha" -- backend/migrations',
            updater,
        )
        self.assertIn("target_skip_migrations=true", updater)
        self.assertIn('MYPAAS_SKIP_MIGRATIONS="$target_skip_migrations"', updater)

    def test_updater_preflights_changed_migrations_before_checkout_reset(self):
        updater = self.text("scripts/update-vm.sh")
        preflight = updater.index('if ! migration_runner_ready "$docker_cmd"; then')
        reset = updater.index('git_repo reset --hard "$target_sha"')
        self.assertLess(preflight, reset)
        self.assertIn(
            "Migration helper could not start; leaving the running installation and checkout unchanged",
            updater,
        )

    def test_runtime_rollback_explicitly_skips_up_migrations(self):
        updater = self.text("scripts/update-vm.sh")
        self.assertIn(
            'MYPAAS_SKIP_IMAGE_PULL=true MYPAAS_SKIP_MIGRATIONS=true',
            updater,
        )

    def test_publish_workflow_requires_green_main_ci_and_keeps_rollback_aliases(self):
        workflow = self.text(".github/workflows/docker-publish.yml")
        self.assertIn('workflow_run:', workflow)
        self.assertIn('workflows: ["CI"]', workflow)
        self.assertIn("github.event.workflow_run.conclusion == 'success'", workflow)
        self.assertIn("github.event.workflow_run.event == 'push'", workflow)
        self.assertIn("github.event.workflow_run.head_branch == 'main'", workflow)
        self.assertIn('RELEASE_SHA: ${{ github.event.workflow_run.head_sha }}', workflow)
        self.assertIn('rollback_tag=rollback-${RELEASE_SHA:0:12}', workflow)
        self.assertIn('ref: ${{ steps.release.outputs.sha }}', workflow)
        self.assertIn('${{ env.REGISTRY }}/nabilrn/mypaas-api:${{ steps.release.outputs.rollback_tag }}', workflow)
        self.assertIn('${{ env.REGISTRY }}/nabilrn/mypaas-dashboard:${{ steps.release.outputs.rollback_tag }}', workflow)

    def test_post_update_verifier_checks_dashboard_identity_and_project_route(self):
        verify = self.text("scripts/verify-production.sh")
        self.assertIn("Checking dashboard reachability", verify)
        self.assertIn("EXPECTED_BUILD_SHA", verify)
        self.assertIn("EXPECTED_IMAGE_TAG", verify)
        self.assertIn("MYPAAS_BUILD_SHA", verify)
        self.assertIn("first_project_host", verify)
        self.assertIn("REQUIRE_PROJECT_ROUTE", verify)

    def test_owner_settings_exposes_build_sha(self):
        settings = self.text("backend/internal/settings/handler.go")
        self.assertIn('res["build_sha"]', settings)
        self.assertIn('os.Getenv("MYPAAS_BUILD_SHA")', settings)


if __name__ == "__main__":
    unittest.main()
