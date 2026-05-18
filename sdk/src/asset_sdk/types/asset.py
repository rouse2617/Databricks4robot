"""Pydantic models for Asset resources."""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Optional

from pydantic import BaseModel, Field


class AssetStatus(str, Enum):
    pending = "pending"
    active = "active"
    archived = "archived"
    deleted = "deleted"


class Asset(BaseModel):
    asset_id: str
    mcap_file_id: str
    t_start: int = Field(description="segment start in nanoseconds")
    t_end: int = Field(description="segment end in nanoseconds")
    reviewer: str
    status: AssetStatus
    duration_sec: float
    owner: str
    version: int
    created_at: datetime
    updated_at: datetime


class AssetCreate(BaseModel):
    mcap_file_id: str
    t_start: int
    t_end: int
    reviewer: str
    owner: str = ""


class AssetUpdate(BaseModel):
    status: Optional[AssetStatus] = None
    reviewer: Optional[str] = None


class AssetList(BaseModel):
    items: list[Asset]
    total: int
    page: int = 1
    page_size: int = 20
