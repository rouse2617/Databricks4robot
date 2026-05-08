"""asset-sdk: Python SDK for the data4cyber platform."""

from asset_sdk.client import AssetClientSDK

# Backward-compatible aliases for existing callers.
DataCurationClient = AssetClientSDK
GraceClient = AssetClientSDK

__all__ = ["AssetClientSDK", "DataCurationClient", "GraceClient"]
