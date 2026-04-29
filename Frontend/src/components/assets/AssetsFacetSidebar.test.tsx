// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import AssetsFacetSidebar, {
  getCheckedValues,
  isSwitchActive,
  getInputValue,
  STATUS_OPTIONS,
  LIFECYCLE_OPTIONS,
  ASSET_TYPE_OPTIONS,
  ENV_OPTIONS,
  ALGO_STATUS_OPTIONS,
  PRIORITY_OPTIONS,
  QUALITY_OPTIONS,
  GROUP_KEYS,
  type AssetsFacetSidebarProps,
} from "./AssetsFacetSidebar";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

afterEach(cleanup);

// ─── Helper ───

function makeChip(overrides: Partial<FilterChip> = {}): FilterChip {
  return {
    id: "chip_1",
    field: "status",
    op: "eq",
    value: "approved",
    source: "facet",
    ...overrides,
  };
}

function defaultProps(overrides: Partial<AssetsFacetSidebarProps> = {}): AssetsFacetSidebarProps {
  return {
    activeFilters: [],
    expandedGroups: ["basic", "capture", "algorithm", "delivery", "tags"],
    rangeDrafts: {},
    dateDrafts: {},
    onToggleFacet: vi.fn(),
    onApplyRange: vi.fn(),
    onApplyDate: vi.fn(),
    onRangeDraftChange: vi.fn(),
    onDateDraftChange: vi.fn(),
    onToggleGroup: vi.fn(),
    ...overrides,
  };
}

// ─── Pure Helper Tests ───

describe("getCheckedValues", () => {
  it("returns empty array when no filters match", () => {
    expect(getCheckedValues([], "status")).toEqual([]);
  });

  it("extracts values for matching field", () => {
    const filters = [
      makeChip({ id: "1", field: "status", value: "approved" }),
      makeChip({ id: "2", field: "status", value: "rejected" }),
      makeChip({ id: "3", field: "env", value: "kitchen" }),
    ];
    expect(getCheckedValues(filters, "status")).toEqual(["approved", "rejected"]);
  });

  it("flattens array values", () => {
    const filters = [
      makeChip({ id: "1", field: "env", value: ["kitchen", "outdoor"] }),
    ];
    expect(getCheckedValues(filters, "env")).toEqual(["kitchen", "outdoor"]);
  });
});

describe("isSwitchActive", () => {
  it("returns false when no matching filter", () => {
    expect(isSwitchActive([], "has:delivery")).toBe(false);
  });

  it("returns true when matching filter exists", () => {
    const filters = [makeChip({ field: "has:delivery", value: "true" })];
    expect(isSwitchActive(filters, "has:delivery")).toBe(true);
  });
});

describe("getInputValue", () => {
  it("returns empty string when no matching filter", () => {
    expect(getInputValue([], "owner")).toBe("");
  });

  it("returns string value for matching filter", () => {
    const filters = [makeChip({ field: "owner", value: "alice" })];
    expect(getInputValue(filters, "owner")).toBe("alice");
  });

  it("returns first element for array value", () => {
    const filters = [makeChip({ field: "owner", value: ["alice", "bob"] })];
    expect(getInputValue(filters, "owner")).toBe("alice");
  });
});

// ─── Constants ───

describe("facet option constants", () => {
  it("exports correct status options", () => {
    expect(STATUS_OPTIONS).toEqual(["approved", "rejected", "superseded", "archived"]);
  });

  it("exports lifecycle options", () => {
    expect(LIFECYCLE_OPTIONS).toContain("ready");
    expect(LIFECYCLE_OPTIONS).toContain("created");
  });

  it("exports asset type options", () => {
    expect(ASSET_TYPE_OPTIONS).toEqual(["segment", "clip", "frame_set", "derived_asset"]);
  });

  it("exports correct env options", () => {
    expect(ENV_OPTIONS).toEqual(["kitchen", "outdoor", "warehouse", "office", "factory"]);
  });

  it("exports correct algo_status options", () => {
    expect(ALGO_STATUS_OPTIONS).toEqual(["ok", "failed", "running", "pending", "blocked"]);
  });

  it("exports correct priority options", () => {
    expect(PRIORITY_OPTIONS).toEqual(["critical", "high", "medium", "low"]);
  });

  it("exports correct quality options", () => {
    expect(QUALITY_OPTIONS).toEqual(["excellent", "good", "acceptable", "poor", "unusable"]);
  });

  it("exports 5 group keys", () => {
    expect(GROUP_KEYS).toEqual(["basic", "capture", "algorithm", "delivery", "tags"]);
  });
});

// ─── Component Rendering ───

describe("AssetsFacetSidebar", () => {
  it("renders all 5 group labels", () => {
    render(<AssetsFacetSidebar {...defaultProps()} />);
    expect(screen.getByText("基础")).toBeTruthy();
    expect(screen.getByText("采集")).toBeTruthy();
    expect(screen.getByText("算法")).toBeTruthy();
    expect(screen.getByText("交付")).toBeTruthy();
    expect(screen.getByText("标签")).toBeTruthy();
  });

  it("renders facet labels within expanded groups", () => {
    render(<AssetsFacetSidebar {...defaultProps()} />);
    expect(screen.getByText("生命周期")).toBeTruthy();
    expect(screen.getByText("环境 (cf_meta)")).toBeTruthy();
    expect(screen.getByText("算法状态")).toBeTruthy();
    expect(screen.getByText("有交付")).toBeTruthy();
    expect(screen.getByText("优先级 (tags_flat)")).toBeTruthy();
    expect(screen.getByText("质量 (tags_flat)")).toBeTruthy();
  });

  it("renders checkbox options for lifecycle facet", () => {
    render(<AssetsFacetSidebar {...defaultProps()} />);
    expect(screen.getByText("ready")).toBeTruthy();
  });

  it("renders checkbox options for env facet", () => {
    render(<AssetsFacetSidebar {...defaultProps()} />);
    for (const opt of ENV_OPTIONS) {
      expect(screen.getByText(opt)).toBeTruthy();
    }
  });

  it("renders checkbox options for algo_status facet", () => {
    render(<AssetsFacetSidebar {...defaultProps()} />);
    for (const opt of ALGO_STATUS_OPTIONS) {
      expect(screen.getByText(opt)).toBeTruthy();
    }
  });

  it("calls onToggleFacet when a checkbox is clicked", () => {
    const onToggleFacet = vi.fn();
    render(<AssetsFacetSidebar {...defaultProps({ onToggleFacet })} />);
    fireEvent.click(screen.getByText("approved"));
    expect(onToggleFacet).toHaveBeenCalledWith("status", "approved");
  });

  it("calls onToggleFacet to remove when unchecking a checked value", () => {
    const onToggleFacet = vi.fn();
    const filters = [makeChip({ id: "1", field: "status", value: "approved" })];
    render(
      <AssetsFacetSidebar {...defaultProps({ activeFilters: filters, onToggleFacet })} />,
    );
    // Click the already-checked "approved" to uncheck it
    fireEvent.click(screen.getByText("approved"));
    expect(onToggleFacet).toHaveBeenCalledWith("status", "approved");
  });

  it("calls onApplyRange when Apply button is clicked for duration", () => {
    const onApplyRange = vi.fn();
    const rangeDrafts = { duration_ms: { min: 9000, max: 12000 } };
    render(
      <AssetsFacetSidebar
        {...defaultProps({
          rangeDrafts,
          onApplyRange,
          expandedGroups: ["capture"],
        })}
      />,
    );
    // Find Apply buttons by role — Ant Design inserts spaces between CJK chars
    const applyButtons = screen.getAllByRole("button", { name: /应\s*用/ });
    fireEvent.click(applyButtons[0]);
    expect(onApplyRange).toHaveBeenCalledWith("duration_ms", 9000, 12000);
  });

  it("calls onToggleFacet when has:delivery switch is toggled", () => {
    const onToggleFacet = vi.fn();
    render(<AssetsFacetSidebar {...defaultProps({ onToggleFacet })} />);
    // Find the switch by its associated label
    const switchEl = screen.getByRole("switch");
    fireEvent.click(switchEl);
    expect(onToggleFacet).toHaveBeenCalledWith("has:delivery", "true");
  });

  it("renders only expanded groups' content", () => {
    render(
      <AssetsFacetSidebar {...defaultProps({ expandedGroups: ["basic"] })} />,
    );
    // Basic group content should be visible
    expect(screen.getByText("状态 (legacy)")).toBeTruthy();
    // Algorithm group content should not be visible (collapsed)
    expect(screen.queryByText("算法状态")).toBeNull();
  });
});
