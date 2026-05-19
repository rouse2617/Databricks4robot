"""Unit tests for ActionClient using respx to mock httpx."""

from datetime import datetime, timezone

import httpx
import respx

from asset_sdk.actions import ActionClient
from asset_sdk.types.action import ActionCreate

ASSET_ID = "550e8400-e29b-41d4-a716-446655440000"
ACTION_ID = "11111111-2222-3333-4444-555555555555"

NOW = datetime.now(tz=timezone.utc).isoformat()

MOCK_ACTION = {
    "action_id": ACTION_ID,
    "asset_id": ASSET_ID,
    "start_ns": 1_000_000,
    "end_ns": 2_000_000,
    "primary_label": "pickup",
    "labels": ["pickup", "left_hand"],
    "attrs": {},
    "source_type": "human",
    "source_name": "annotator-001",
    "is_deleted": False,
    "version": 1,
    "created_at": NOW,
    "updated_at": NOW,
}


@respx.mock
def test_create_action() -> None:
    respx.post(f"http://test/api/v1/assets/{ASSET_ID}/actions").mock(
        return_value=httpx.Response(201, json=MOCK_ACTION)
    )

    client = ActionClient(httpx.Client(base_url="http://test"))
    action = client.create(
        ASSET_ID,
        ActionCreate(
            start_ns=1_000_000,
            end_ns=2_000_000,
            primary_label="pickup",
            labels=["pickup", "left_hand"],
            source_name="annotator-001",
        ),
    )

    assert action.action_id == ACTION_ID
    assert action.primary_label == "pickup"
    assert action.labels == ["pickup", "left_hand"]
    assert action.source_type.value == "human"


@respx.mock
def test_list_actions_with_at_filter() -> None:
    route = respx.get(f"http://test/api/v1/assets/{ASSET_ID}/actions").mock(
        return_value=httpx.Response(
            200,
            json={"items": [MOCK_ACTION], "asset_id": ASSET_ID, "total": 1},
        )
    )

    client = ActionClient(httpx.Client(base_url="http://test"))
    result = client.at(ASSET_ID, 1_500_000)

    assert result.total == 1
    assert result.items[0].action_id == ACTION_ID
    # Verify filter was forwarded as ?at=...
    sent_url = route.calls.last.request.url
    assert sent_url.params.get("at") == "1500000"
    assert sent_url.params.get("limit") == "200"


@respx.mock
def test_list_actions_overlap_window() -> None:
    route = respx.get(f"http://test/api/v1/assets/{ASSET_ID}/actions").mock(
        return_value=httpx.Response(
            200, json={"items": [], "asset_id": ASSET_ID, "total": 0}
        )
    )
    client = ActionClient(httpx.Client(base_url="http://test"))
    client.list(ASSET_ID, from_ns=0, to_ns=5_000_000, label="pickup", limit=50)

    sent_url = route.calls.last.request.url
    assert sent_url.params.get("from") == "0"
    assert sent_url.params.get("to") == "5000000"
    assert sent_url.params.get("label") == "pickup"
    assert sent_url.params.get("limit") == "50"
