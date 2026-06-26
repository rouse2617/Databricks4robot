"""User management — CRUD operations for DataBrew platform users."""

from __future__ import annotations

import logging
from dataclasses import dataclass

_logger = logging.getLogger(__name__)


@dataclass
class UserInfo:
    """Platform user information."""
    id: str
    email: str
    role: str
    enabled: bool = True


class UserManager:
    """Manage platform users.

    Requires admin-level DataBrew token.
    """

    def __init__(self, requestor) -> None:
        self._requestor = requestor

    def get(self, user_id: str) -> UserInfo:
        """Get user by ID."""
        # TODO: implement
        raise NotImplementedError

    def list(self, page: int = 1, page_size: int = 20) -> list[UserInfo]:
        """List users."""
        raise NotImplementedError

    def create(self, email: str, role: str = "user") -> UserInfo:
        """Create a new user."""
        raise NotImplementedError

    def disable(self, user_id: str) -> None:
        """Disable a user."""
        raise NotImplementedError
