"""Generic pagination types.

Adapted from the ItemPaged[T] pattern used by Azure SDK and OpenAI.
Returns typed Page containers with cursor/next-page awareness.
"""

from __future__ import annotations

from typing import Any, Generic, TypeVar

T = TypeVar("T")


class Page(Generic[T]):
    """A single page of results with optional cursor for next page."""

    items: list[T]
    total: int | None
    next_cursor: str | None
    _raw: dict[str, Any]

    def __init__(
        self,
        items: list[T],
        *,
        total: int | None = None,
        next_cursor: str | None = None,
        raw: dict[str, Any] | None = None,
    ) -> None:
        self.items = items
        self.total = total
        self.next_cursor = next_cursor
        self._raw = raw or {}

    def has_next(self) -> bool:
        return self.next_cursor is not None

    def __len__(self) -> int:
        return len(self.items)

    def __iter__(self):
        return iter(self.items)

    def __getitem__(self, index):
        return self.items[index]
