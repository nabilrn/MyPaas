import subprocess
import unittest
from unittest import mock

import fixture_ref


class FixtureRefTest(unittest.TestCase):
    def remote(self, stdout: str, returncode: int = 0, stderr: str = ""):
        return mock.patch(
            "fixture_ref.subprocess.run",
            return_value=subprocess.CompletedProcess(
                args=["git", "ls-remote"],
                returncode=returncode,
                stdout=stdout,
                stderr=stderr,
            ),
        )

    def test_full_sha_resolves_to_candidate_branch(self):
        sha = "a" * 40
        output = (
            f"{sha}\trefs/heads/other\n"
            f"{sha}\trefs/heads/test/beta-readiness-candidate\n"
        )
        with self.remote(output):
            result = fixture_ref.resolve_fixture_ref("https://github.com/example/repo", sha)
        self.assertEqual(result.branch, "test/beta-readiness-candidate")
        self.assertEqual(result.resolved_sha, sha)

    def test_explicit_branch_must_match_expected_sha(self):
        expected = "a" * 40
        actual = "b" * 40
        with self.remote(f"{actual}\trefs/heads/test/beta-readiness-candidate\n"):
            with self.assertRaisesRegex(fixture_ref.FixtureRefError, "expected immutable SHA"):
                fixture_ref.resolve_fixture_ref(
                    "https://github.com/example/repo",
                    expected,
                    "test/beta-readiness-candidate",
                )

    def test_full_sha_without_matching_branch_fails_before_workload(self):
        with self.remote(f"{'b' * 40}\trefs/heads/main\n"):
            with self.assertRaisesRegex(fixture_ref.FixtureRefError, "not the head of any remote branch"):
                fixture_ref.resolve_fixture_ref("https://github.com/example/repo", "a" * 40)

    def test_branch_ref_is_verified_and_returns_resolved_sha(self):
        sha = "c" * 40
        with self.remote(f"{sha}\trefs/heads/main\n"):
            result = fixture_ref.resolve_fixture_ref("https://github.com/example/repo", "main")
        self.assertEqual(result.branch, "main")
        self.assertEqual(result.resolved_sha, sha)

    def test_remote_failure_is_actionable(self):
        with self.remote("", returncode=128, stderr="fatal: repository not found"):
            with self.assertRaisesRegex(fixture_ref.FixtureRefError, "failed to inspect fixture repository branches"):
                fixture_ref.resolve_fixture_ref("https://github.com/example/missing", "main")


if __name__ == "__main__":
    unittest.main()
