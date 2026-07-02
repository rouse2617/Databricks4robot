"""Storage backend abstract interface.

Each backend implements how to read/write/open/stat files on a specific
storage system (GCS, S3, local, proxy, …).  The :class:`StorageManager`
routes operations to the right backend based on the URI protocol.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from pathlib import Path
from typing import IO, Any


class FileInfo:
    """File metadata (similar to ``os.stat_result``)."""
    name: str
    size: int
    mtime: float | None
    type: str  # "file" | "dir"

    def __init__(self, name: str, size: int = 0, mtime: float | None = None, type: str = "file") -> None:
        self.name = name
        self.size = size
        self.mtime = mtime
        self.type = type


class Backend(ABC):
    """Abstract storage backend."""

    @abstractmethod
    def open(self, path: str, mode: str = "rb") -> IO[Any]:
        """Open a file for reading/writing (supports seek/tell)."""
        ...

    @abstractmethod
    def read(self, path: str) -> bytes:
        """Read entire file into memory."""
        ...

    @abstractmethod
    def write(self, path: str, data: bytes) -> int:
        """Write entire file."""
        ...

    @abstractmethod
    def stat(self, path: str) -> FileInfo:
        """Get file metadata."""
        ...

    @abstractmethod
    def listdir(self, path: str) -> list[FileInfo]:
        """List directory entries."""
        ...

    @abstractmethod
    def copy(self, src: str, dst: str) -> None:
        """Copy file."""
        ...

    @abstractmethod
    def delete(self, path: str) -> None:
        """Delete file."""
        ...

    @abstractmethod
    def exists(self, path: str) -> bool:
        """Check file existence."""
        ...

    def download(self, path: str, local_path: str | Path) -> Path:
        """Download to local file — default: read + write local."""
        local_path = Path(local_path)
        local_path.parent.mkdir(parents=True, exist_ok=True)
        local_path.write_bytes(self.read(path))
        return local_path

    def upload(self, local_path: str | Path, path: str) -> None:
        """Upload from local file — default: read local + write."""
        self.write(path, Path(local_path).read_bytes())
