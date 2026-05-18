"""Pydantic models for Action resources (mcap → seg → action)."""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Any, Optional

from pydantic import BaseModel, Field


class ActionSourceType(str, Enum):
    human = "human"
    algo = "algo"
    rule = "rule"
    system = "system"


class Action(BaseModel):
    action_id: str
    asset_id: str
    start_ns: int = Field(description="half-open interval start (ns)")
    end_ns: int = Field(description="half-open interval end (ns)")
    action_index: Optional[int] = None
    primary_label: Optional[str] = None
    labels: list[str] = Field(default_factory=list)
    description: Optional[str] = None
    attrs: dict[str, Any] = Field(default_factory=dict)
    source_type: ActionSourceType = ActionSourceType.human
    source_name: Optional[str] = None
    source_version: Optional[str] = None
    run_id: Optional[str] = None
    confidence: Optional[float] = None
    external_id: Optional[str] = None
    tenant_id: Optional[str] = None
    project_id: Optional[str] = None
    is_deleted: bool = False
    version: int = 1
    created_at: datetime
    updated_at: datetime


class ActionCreate(BaseModel):
    start_ns: int
    end_ns: int
    action_index: Optional[int] = None
    primary_label: Optional[str] = None
    labels: list[str] = Field(default_factory=list)
    description: Optional[str] = None
    attrs: dict[str, Any] = Field(default_factory=dict)
    source_type: ActionSourceType = ActionSourceType.human
    source_name: Optional[str] = None
    source_version: Optional[str] = None
    run_id: Optional[str] = None
    confidence: Optional[float] = None
    external_id: Optional[str] = None


class ActionList(BaseModel):
    items: list[Action] = Field(default_factory=list)
    asset_id: str
    total: int = 0
