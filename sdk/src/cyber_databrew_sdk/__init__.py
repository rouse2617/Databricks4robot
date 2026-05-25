"""cyber-databrew Python SDK."""

import warnings

from cyber_databrew_sdk.auth import AuthProvider, CompositeAuth, EmailAuth, GraceTokenAuth
from cyber_databrew_sdk.client import CyberDatabrew, CyberDatabrewClient
from cyber_databrew_sdk.config import ConfigManager
from cyber_databrew_sdk.exceptions import (
    APIConnectionError,
    AuthenticationError,
    BadRequestError,
    ConflictError,
    CyberDatabrewError,
    NotFoundError,
    RateLimitError,
    ServerError,
    ValidationError,
)

__all__ = [
    "APIConnectionError",
    "AssetClientSDK",
    "AuthProvider",
    "AuthenticationError",
    "BadRequestError",
    "CompositeAuth",
    "ConfigManager",
    "ConflictError",
    "CyberDatabrew",
    "CyberDatabrewClient",
    "CyberDatabrewError",
    "DataCurationClient",
    "EmailAuth",
    "GraceClient",
    "GraceTokenAuth",
    "NotFoundError",
    "RateLimitError",
    "ServerError",
    "ValidationError",
]

# Backward-compat aliases for asset_sdk users — resolved via __getattr__
# with DeprecationWarning. Direct references (e.g. from cyber_databrew_sdk
# import AssetClientSDK) still work via PEP 562 module __getattr__.

import typing as _t


def __getattr__(name: str) -> _t.Any:
    if name in ("AssetClientSDK", "GraceClient", "DataCurationClient"):
        warnings.warn(
            f"{name} is deprecated, use CyberDatabrewClient instead",
            DeprecationWarning,
            stacklevel=2,
        )
        return CyberDatabrewClient
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
