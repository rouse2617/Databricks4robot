"""Unit tests for AssetClient using respx to mock httpx."""

from datetime import datetime, timezone

import httpx
import respx

from asset_sdk.assets import AssetClient

MOCK_ASSET = {
    "asset_id": "550e8400-e29b-41d4-a716-446655440000",
    "mcap_file_id": "file-001",
    "t_start": 1000000000,
    "t_end": 2000000000,
    "reviewer": "alice",
    "status": "active",
    "duration_sec": 1.0,
    "owner": "team-a",
    "version": 1,
    "created_at": datetime.now(tz=timezone.utc).isoformat(),
    "updated_at": datetime.now(tz=timezone.utc).isoformat(),
}


@respx.mock
def test_get_asset():
    asset_id = MOCK_ASSET["asset_id"]
    respx.get(f"http://test/api/v1/assets/{asset_id}").mock(
        return_value=httpx.Response(200, json=MOCK_ASSET)
    )

    client = AssetClient(httpx.Client(base_url="http://test"))
    asset = client.get(asset_id)

    assert asset.asset_id == asset_id
    assert asset.status.value == "active"


@respx.mock
def test_list_assets():
    respx.get("http://test/api/v1/assets").mock(
        return_value=httpx.Response(200, json={"items": [MOCK_ASSET], "total": 1, "page": 1, "page_size": 20})
    )

    client = AssetClient(httpx.Client(base_url="http://test"))
    result = client.list()

    assert result.total == 1
    assert len(result.items) == 1
