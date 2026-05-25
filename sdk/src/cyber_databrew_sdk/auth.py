"""Pluggable authentication strategies.

Pattern adapted from Stripe's RequestorOptions + OpenAI's
api_key_provider. AuthProvider is a Protocol so users
can supply custom auth without subclassing.
"""

from __future__ import annotations

import os
from typing import Protocol


class AuthProvider(Protocol):
    """Pluggable authentication strategy.

    Implementations return headers that are injected into
    every API request.
    """

    def get_headers(self) -> dict[str, str]: ...


class GraceTokenAuth:
    """Primary auth: inject X-Grace-Token header.

    Token defaults from CYBER_DATABREW_TOKEN, then GRACE_TOKEN
    environment variable.
    """

    def __init__(self, token: str | None = None) -> None:
        self._token = (
            token
            or os.environ.get("CYBER_DATABREW_TOKEN")
            or os.environ.get("GRACE_TOKEN", "")
        )

    def get_headers(self) -> dict[str, str]:
        return {"X-Grace-Token": self._token}


class EmailAuth:
    """Audit-only auth: inject X-User-Email header.

    This is NOT a security credential. The email is used solely
    for request logging / audit trails on the backend.

    Email defaults from CYBER_DATABREW_EMAIL environment variable.
    """

    def __init__(self, email: str | None = None) -> None:
        self._email = email or os.environ.get("CYBER_DATABREW_EMAIL", "")

    def get_headers(self) -> dict[str, str]:
        return {"X-User-Email": self._email}


class CompositeAuth:
    """Compose multiple AuthProviders into a single header set.

    Later providers override earlier ones on key conflict.
    Empty-value headers are filtered out.
    """

    def __init__(self, *providers: AuthProvider) -> None:
        self._providers = providers

    def get_headers(self) -> dict[str, str]:
        result: dict[str, str] = {}
        for provider in self._providers:
            result.update(
                {k: v for k, v in provider.get_headers().items() if v}
            )
        return result
