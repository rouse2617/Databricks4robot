"""Top-level GraceClient that exposes sub-clients for each resource."""

from __future__ import annotations

import os
from typing import Optional

import httpx

from grace_sdk.assets import AssetClient
from grace_sdk.mcap import McapClient
from grace_sdk.delivery import DeliveryClient


class GraceClient:
    """Entry point for the grace-sdk.

    Usage::

        from grace_sdk import GraceClient

        client = GraceClient(base_url="http://localhost:8080", token="dev-token")
        asset = client.assets.get("asset-uuid")
    """

    def __init__(
        self,
        base_url: Optional[str] = None,
        token: Optional[str] = None,
        timeout: float = 30.0,
    ) -> None:
        self._base_url = (base_url or os.environ.get("GRACE_BASE_URL", "http://localhost:8080")).rstrip("/")
        self._token = token or os.environ.get("GRACE_TOKEN", "")

        self._http = httpx.Client(
            base_url=self._base_url,
            headers={
                "X-Grace-Token": self._token,
                "Content-Type": "application/json",
            },
            timeout=timeout,
        )

        self.assets = AssetClient(self._http)
        self.mcap = McapClient(self._http)
        self.delivery = DeliveryClient(self._http)

    def close(self) -> None:
        self._http.close()

    def __enter__(self) -> "GraceClient":
        return self

    def __exit__(self, *_: object) -> None:
        self.close()
