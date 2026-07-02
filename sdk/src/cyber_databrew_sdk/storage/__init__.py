"""Storage module — multi-backend file operations.

Backends:
  - ``GCSBackend``   — Google Cloud Storage (gcsfs + transfer_manager)
  - ``LocalBackend`` — POSIX local filesystem
  - ``ProxyBackend`` — Backend-proxy (sign-url) for users without GCP

Usage via :class:`StorageManager`::

    from cyber_databrew_sdk.storage.manager import StorageManager
"""

from cyber_databrew_sdk.storage.backend import Backend, FileInfo
from cyber_databrew_sdk.storage.gcs import GCSBackend
from cyber_databrew_sdk.storage.local import LocalBackend
from cyber_databrew_sdk.storage.manager import StorageManager
from cyber_databrew_sdk.storage.proxy import ProxyBackend

__all__ = [
    "Backend",
    "FileInfo",
    "GCSBackend",
    "LocalBackend",
    "ProxyBackend",
    "StorageManager",
]
