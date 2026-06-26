"""Authentication — providers, token management, and credential resolution.

Usage::

    from cyber_databrew_sdk.auth import AuthProvider, DatabrewTokenAuth

    auth = DatabrewTokenAuth(token="my-token")
    headers = auth.get_headers()
"""

from cyber_databrew_sdk.auth.providers import (
    AuthProvider,
    CompositeAuth,
    DatabrewTokenAuth,
    EmailAuth,
)
from cyber_databrew_sdk.auth.tokens import TokenResolver, TokenResult

__all__ = [
    "AuthProvider",
    "CompositeAuth",
    "DatabrewTokenAuth",
    "EmailAuth",
    "TokenResolver",
    "TokenResult",
]
