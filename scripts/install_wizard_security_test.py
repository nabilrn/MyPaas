import io
import os
import tarfile
import tempfile
import unittest
from pathlib import Path

import install_wizard_security as security


class InstallWizardSecurityTest(unittest.TestCase):
    def make_archive(self, path: Path, members: list[tuple[str, bytes]]) -> bytes:
        with tarfile.open(path, "w:gz") as archive:
            for name, body in members:
                info = tarfile.TarInfo(name)
                info.size = len(body)
                info.mode = 0o600
                archive.addfile(info, io.BytesIO(body))
        return path.read_bytes()

    def test_valid_backup_is_stored_with_private_permissions(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "source.tar.gz"
            payload = self.make_archive(
                source,
                [("database.sql", b"select 1;\n"), (".env", b"PUBLIC_DOMAIN=example.com\n")],
            )
            destination = root / "restore.tar.gz"

            security.store_backup_upload(io.BytesIO(payload), len(payload), destination)

            self.assertEqual(destination.read_bytes(), payload)
            self.assertEqual(os.stat(destination).st_mode & 0o777, 0o600)
            security.validate_backup_archive(destination)

    def test_archive_rejects_unexpected_members(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            archive_path = Path(tmp) / "bad.tar.gz"
            self.make_archive(
                archive_path,
                [
                    ("database.sql", b"select 1;\n"),
                    (".env", b"PUBLIC_DOMAIN=example.com\n"),
                    ("extra.txt", b"unexpected"),
                ],
            )
            with self.assertRaisesRegex(security.BackupUploadError, "unexpected backup member"):
                security.validate_backup_archive(archive_path)

    def test_archive_rejects_links(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            archive_path = Path(tmp) / "bad.tar.gz"
            with tarfile.open(archive_path, "w:gz") as archive:
                database = tarfile.TarInfo("database.sql")
                database.type = tarfile.SYMTYPE
                database.linkname = "/etc/passwd"
                archive.addfile(database)
                env = tarfile.TarInfo(".env")
                env.size = 3
                archive.addfile(env, io.BytesIO(b"A=1"))
            with self.assertRaisesRegex(security.BackupUploadError, "regular file"):
                security.validate_backup_archive(archive_path)

    def test_archive_requires_database_and_env(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            archive_path = Path(tmp) / "bad.tar.gz"
            self.make_archive(archive_path, [("database.sql", b"select 1;\n")])
            with self.assertRaisesRegex(security.BackupUploadError, "missing required files"):
                security.validate_backup_archive(archive_path)

    def test_upload_rejects_declared_size_over_limit_before_reading(self) -> None:
        class FailOnRead(io.BytesIO):
            def read(self, *args, **kwargs):
                raise AssertionError("oversized request should be rejected before reading")

        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            with self.assertRaisesRegex(security.BackupUploadError, "exceeds"):
                security.store_backup_upload(
                    FailOnRead(b""),
                    11,
                    destination,
                    max_upload_bytes=10,
                )
            self.assertFalse(destination.exists())

    def test_upload_rejects_truncated_body_without_replacing_destination(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / "restore.tar.gz"
            destination.write_bytes(b"known-good-existing")
            with self.assertRaisesRegex(security.BackupUploadError, "ended before"):
                security.store_backup_upload(io.BytesIO(b"short"), 10, destination)
            self.assertEqual(destination.read_bytes(), b"known-good-existing")


if __name__ == "__main__":
    unittest.main()
