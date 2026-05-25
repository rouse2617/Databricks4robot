"""cyber-databrew Python SDK."""

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

# Backward-compat aliases for asset_sdk users
AssetClientSDK = CyberDatabrewClient
GraceClient = CyberDatabrewClient
DataCurationClient = CyberDatabrewClient
