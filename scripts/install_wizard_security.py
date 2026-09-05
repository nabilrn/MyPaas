#!/usr/bin/env python3
"""Security primitives for the standalone MyPaas install wizard.

Keep this module standard-library-only so the installer remains usable before the
MyPaas application stack exists.
"""

from __future__ import annotations

import argparse
import os
import pathlib
import tarfile
import tempfile
from typing import BinaryIO

DEFAULT_MAX_BACKUP_BYTES = 1024 * 1024 * 1024  # 1 GiB compressed upload
DEFAULT_MAX_EXPANDED_BACKUP_BYTES = 4 * 1024 * 1024 * 1024  # 4 GiB extracted payload
ALLOWED_BACKUP_MEMBERS = frozenset({"database.sql", ".env"})


class BackupUploadError(ValueError):
    pass


def _normalized_member_name(name: str) -> str:
    while name.startswith("./"):
        name = name[2:]
    return name


def validate_backup_archive(
    path: str | os.PathLike[str],
    *,
    max_expanded_bytes: int = DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
) -> None:
    if max_expanded_bytes <= 0:
        raise BackupUploadError("expanded backup limit must be positive")

    seen: set[str] = set()
    expanded = 0
    try:
        with tarfile.open(path, "r:gz") as archive:
            members = archive.getmembers()
            if not members:
                raise BackupUploadError("backup archive is empty")
            for member in members:
                name = _normalized_member_name(member.name)
                if name not in ALLOWED_BACKUP_MEMBERS:
                    raise BackupUploadError(f"unexpected backup member: {name or '<empty>'}")
                if name in seen:
                    raise BackupUploadError(f"duplicate backup member: {name}")
                if not member.isfile():
                    raise BackupUploadError(f"backup member must be a regular file: {name}")
                expanded += member.size
                if expanded > max_expanded_bytes:
                    raise BackupUploadError("expanded backup exceeds the configured size limit")
                seen.add(name)
    except (tarfile.TarError, OSError) as exc:
        raise BackupUploadError("backup must be a readable .tar.gz archive") from exc

    missing = ALLOWED_BACKUP_MEMBERS - seen
    if missing:
        raise BackupUploadError("backup is missing required files: " + ", ".join(sorted(missing)))


def store_backup_upload(
    stream: BinaryIO,
    content_length: int,
    destination: str | os.PathLike[str],
    *,
    max_upload_bytes: int = DEFAULT_MAX_BACKUP_BYTES,
    max_expanded_bytes: int = DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
    chunk_bytes: int = 1024 * 1024,
) -> None:
    if content_length <= 0:
        raise BackupUploadError("backup upload is empty")
    if max_upload_bytes <= 0:
        raise BackupUploadError("backup upload limit must be positive")
    if content_length > max_upload_bytes:
        raise BackupUploadError("backup upload exceeds the configured size limit")
    if chunk_bytes <= 0:
        raise BackupUploadError("backup upload chunk size must be positive")

    destination_path = pathlib.Path(destination)
    destination_path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary_name = tempfile.mkstemp(
        prefix=destination_path.name + ".upload-",
        dir=str(destination_path.parent),
    )
    temporary_path = pathlib.Path(temporary_name)
    try:
        with os.fdopen(fd, "wb") as handle:
            remaining = content_length
            while remaining:
                block = stream.read(min(chunk_bytes, remaining))
                if not block:
                    raise BackupUploadError("backup upload ended before Content-Length bytes were received")
                handle.write(block)
                remaining -= len(block)
            handle.flush()
            os.fsync(handle.fileno())

        os.chmod(temporary_path, 0o600)
        validate_backup_archive(temporary_path, max_expanded_bytes=max_expanded_bytes)
        os.replace(temporary_path, destination_path)
        os.chmod(destination_path, 0o600)
    except Exception:
        try:
            temporary_path.unlink()
        except FileNotFoundError:
            pass
        raise


def main() -> None:
    parser = argparse.ArgumentParser(description="Validate a staged MyPaas restore backup")
    parser.add_argument("archive", type=pathlib.Path)
    parser.add_argument(
        "--max-expanded-bytes",
        type=int,
        default=DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
    )
    args = parser.parse_args()
    try:
        validate_backup_archive(args.archive, max_expanded_bytes=args.max_expanded_bytes)
    except BackupUploadError as exc:
        parser.exit(2, f"backup validation error: {exc}\n")


if __name__ == "__main__":
    main()
