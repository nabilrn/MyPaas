from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]


class DockerPublishWorkflowContractTests(unittest.TestCase):
    def test_frontend_only_api_reuse_is_ancestor_safe_and_has_build_fallback(self):
        workflow = (ROOT / ".github/workflows/docker-publish.yml").read_text(encoding="utf-8")

        self.assertIn("fetch-depth: 0", workflow)
        self.assertIn("Resolve reusable API image", workflow)
        self.assertIn("git rev-list --first-parent HEAD^", workflow)
        self.assertIn('git diff --name-only "$candidate" HEAD', workflow)
        self.assertIn('if [[ "$path" != frontend/* ]]', workflow)
        self.assertIn('docker buildx imagetools inspect "$source_image"', workflow)
        self.assertIn("No safe reusable API image found; rebuilding API", workflow)
        self.assertIn("steps.api-reuse.outputs.source_sha == ''", workflow)
        self.assertIn("steps.api-reuse.outputs.source_sha != ''", workflow)
        self.assertIn("Alias reusable API image for frontend-only revision", workflow)


if __name__ == "__main__":
    unittest.main()
