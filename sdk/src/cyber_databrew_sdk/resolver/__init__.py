"""Resolver — maps business asset IDs to storage paths.

Each resolver knows how to talk to a specific source (Grace, internal DB, …)
and returns a ``gs://`` path for a given asset ID.

Usage::

    from cyber_databrew_sdk.resolver import GraceResolver

    resolver = GraceResolver()
    gcs_path = resolver.resolve("019ed9c5-...")
    # → "gs://bucket/path/file.mcap"
"""

from cyber_databrew_sdk.resolver.grace import GraceResolver

__all__ = ["GraceResolver"]
