from __future__ import annotations

import hmac
import os
import tarfile
import tempfile
from typing import BinaryIO


DEFAULT_MAX_BACKUP_BYTES = 512 * 1024 * 1024
DEFAULT_MAX_EXPANDED_BACKUP_BYTES = 2 * 1024 * 1024 * 1024
REQUIRED_BACKUP_MEMBERS = {".env", "database.sql"}
COPY_CHUNK_BYTES = 1024 * 1024


class BackupUploadError(ValueError):
    """Raised when an installer backup upload is malformed or unsafe."""


class BackupTooLargeError(BackupUploadError):
    """Raised when the compressed or expanded backup exceeds the configured limit."""


def token_matches(provided: str | None, expected: str) -> bool:
    if not provided or not expected:
        return False
    return hmac.compare_digest(provided, expected)


def parse_content_length(raw_value: str | None, max_bytes: int) -> int:
    if not raw_value:
        raise BackupUploadError("Content-Length is required")
    try:
        length = int(raw_value)
    except ValueError as exc:
        raise BackupUploadError("Content-Length must be an integer") from exc
    if length <= 0:
        raise BackupUploadError("Backup upload is empty")
    if length > max_bytes:
        raise BackupTooLargeError(f"Backup exceeds the {max_bytes} byte upload limit")
    return length


def validate_backup_archive(path: str, max_expanded_bytes: int = DEFAULT_MAX_EXPANDED_BACKUP_BYTES) -> None:
    names: set[str] = set()
    expanded_bytes = 0
    try:
        with tarfile.open(path, mode="r:gz") as archive:
            for member in archive:
                if member.name not in REQUIRED_BACKUP_MEMBERS:
                    raise BackupUploadError(f"Unexpected backup member: {member.name}")
                if not member.isfile():
                    raise BackupUploadError(f"Backup member must be a regular file: {member.name}")
                if member.name in names:
                    raise BackupUploadError(f"Duplicate backup member: {member.name}")
                if member.size < 0:
                    raise BackupUploadError(f"Backup member has invalid size: {member.name}")
                expanded_bytes += member.size
                if expanded_bytes > max_expanded_bytes:
                    raise BackupTooLargeError(
                        f"Expanded backup exceeds the {max_expanded_bytes} byte limit"
                    )
                names.add(member.name)
    except BackupUploadError:
        raise
    except (OSError, tarfile.TarError) as exc:
        raise BackupUploadError("Backup must be a valid gzip-compressed tar archive") from exc

    if names != REQUIRED_BACKUP_MEMBERS:
        missing = sorted(REQUIRED_BACKUP_MEMBERS.difference(names))
        raise BackupUploadError("Backup is missing required members: " + ", ".join(missing))


def receive_backup(
    stream: BinaryIO,
    content_length: str | None,
    destination: str,
    *,
    max_bytes: int = DEFAULT_MAX_BACKUP_BYTES,
    max_expanded_bytes: int = DEFAULT_MAX_EXPANDED_BACKUP_BYTES,
) -> None:
    length = parse_content_length(content_length, max_bytes)
    directory = os.path.dirname(os.path.abspath(destination)) or "."
    os.makedirs(directory, exist_ok=True)
    fd, temp_path = tempfile.mkstemp(prefix="mypaas-restore-", suffix=".tar.gz.part", dir=directory)

    try:
        with os.fdopen(fd, "wb") as output:
            remaining = length
            while remaining:
                chunk = stream.read(min(COPY_CHUNK_BYTES, remaining))
                if not chunk:
                    raise BackupUploadError("Backup upload ended before Content-Length bytes were received")
                output.write(chunk)
                remaining -= len(chunk)
            output.flush()
            os.fsync(output.fileno())

        os.chmod(temp_path, 0o600)
        validate_backup_archive(temp_path, max_expanded_bytes=max_expanded_bytes)
        os.replace(temp_path, destination)
        temp_path = ""
    finally:
        if temp_path:
            try:
                os.remove(temp_path)
            except FileNotFoundError:
                pass
