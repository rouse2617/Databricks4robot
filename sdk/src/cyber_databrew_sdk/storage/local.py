"""Local filesystem backend."""

from __future__ import annotations

from pathlib import Path
from typing import IO, Any

from cyber_databrew_sdk.storage.backend import Backend, FileInfo


class LocalBackend(Backend):
    """POSIX local filesystem."""

    def open(self, path: str, mode: str = "rb") -> IO[Any]:
        return open(path, mode)

    def read(self, path: str) -> bytes:
        return Path(path).read_bytes()

    def write(self, path: str, data: bytes) -> int:
        Path(path).write_bytes(data)
        return len(data)

    def stat(self, path: str) -> FileInfo:
        p = Path(path)
        if not p.exists():
            raise FileNotFoundError(path)
        st = p.stat()
        return FileInfo(
            name=path,
            size=st.st_size,
            mtime=st.st_mtime,
            type="dir" if p.is_dir() else "file",
        )

    def listdir(self, path: str) -> list[FileInfo]:
        p = Path(path)
        if not p.is_dir():
            raise NotADirectoryError(path)
        results = []
        for entry in p.iterdir():
            st = entry.stat()
            results.append(FileInfo(
                name=entry.name,
                size=st.st_size,
                mtime=st.st_mtime,
                type="dir" if entry.is_dir() else "file",
            ))
        return results

    def copy(self, src: str, dst: str) -> None:
        import shutil
        shutil.copy2(src, dst)

    def delete(self, path: str) -> None:
        Path(path).unlink()

    def exists(self, path: str) -> bool:
        return Path(path).exists()
