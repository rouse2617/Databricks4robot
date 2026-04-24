"""McapClient: finalize and iterate MCAP messages."""

from __future__ import annotations

from typing import Generator, Optional

import httpx


class McapClient:
    _base = "/api/v1/mcap"

    def __init__(self, http: httpx.Client) -> None:
        self._http = http

    def finalize_upload(self, mcap_file_id: str) -> dict:
        """Finalize an already uploaded MCAP file."""
        fin_resp = self._http.post(
            "/api/v1/mcap/upload/finalize",
            json={"mcap_file_id": mcap_file_id},
        )
        fin_resp.raise_for_status()
        return fin_resp.json()

    def iter_messages(
        self,
        mcap_file_id: str,
        *,
        topics: Optional[list[str]] = None,
        t_start: Optional[int] = None,
        t_end: Optional[int] = None,
    ) -> Generator[dict, None, None]:
        """Iterate decoded messages from an MCAP segment via the backend API."""
        params: dict[str, object] = {}
        if topics:
            params["topics"] = ",".join(topics)
        if t_start is not None:
            params["t_start"] = t_start
        if t_end is not None:
            params["t_end"] = t_end

        r = self._http.get(f"{self._base}/{mcap_file_id}/messages", params=params)
        r.raise_for_status()
        for msg in r.json().get("messages", []):
            yield msg
