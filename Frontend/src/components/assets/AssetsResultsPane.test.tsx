// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach, beforeAll } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import AssetsResultsPane from "./AssetsResultsPane";
import type { Asset } from "../../api/types";

// Ant Design Table requires matchMedia for responsive columns
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

afterEach(cleanup);

// ─── Helpers ───

function makeAsset(overrides: Partial<Asset> = {}): Asset {
  return {
    asset_id: "asset_001",
    mcap_file_id: "mcap_001",
    start_timestamp_ns: 0,
    end_timestamp_ns: 1000000000,
    duration_ms: 12500,
    reviewer: "reviewer1",
    status: "approved",
    lifecycle_state: "ready",
    owner: "owner1",
    env: "warehouse",
    delivery_count: 0,
    algo_results: {},
    tags: {},
    files: {},
    lifecycle_meta: {},
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-02T00:00:00Z",
    version: 1,
    ...overrides,
  };
}

const defaultProps = {
  items: [makeAsset()],
  total: 1,
  totalApprox: false,
  fetchStatus: "success" as const,
  sort: "-updated_at",
  page: 1,
  pageSize: 20,
  viewMode: "table" as const,
  selectedColumns: ["asset_id", "duration", "env", "status", "algo", "updated_at"],
  selectedIds: new Set<string>(),
  activePreviewId: null,
  onSortChange: vi.fn(),
  onPageChange: vi.fn(),
  onSelectRow: vi.fn(),
  onRowClick: vi.fn(),
};

// ─── Tests ───

describe("AssetsResultsPane", () => {
  it("renders results count in toolbar", () => {
    render(<AssetsResultsPane {...defaultProps} total={42} />);
    // Toolbar count + pagination both show the count; use getAllByText
    const elements = screen.getAllByText(/共 42 条/);
    expect(elements.length).toBeGreaterThanOrEqual(1);
  });

  it("renders approximate count with + suffix", () => {
    render(<AssetsResultsPane {...defaultProps} total={100} totalApprox={true} />);
    const elements = screen.getAllByText(/共 100\+ 条/);
    expect(elements.length).toBeGreaterThanOrEqual(1);
  });

  it("renders sort selector with correct value", () => {
    render(<AssetsResultsPane {...defaultProps} sort="-updated_at" />);
    expect(screen.getByText("更新时间 ↓")).toBeTruthy();
  });

  it("renders table with data", () => {
    render(<AssetsResultsPane {...defaultProps} />);
    // Ant Design Table renders header + measure row, so use getAllByText
    const headers = screen.getAllByText("Asset ID");
    expect(headers.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("12.5s")).toBeTruthy();
    expect(screen.getByText("warehouse")).toBeTruthy();
    expect(screen.getByText("ready")).toBeTruthy();
  });

  it("renders algo summary cell with dash when no algo results", () => {
    render(<AssetsResultsPane {...defaultProps} />);
    expect(screen.getByText("—")).toBeTruthy();
  });

  it("renders algo summary with counts when algo results present", () => {
    const asset = makeAsset({
      algo_results: {
        "hand_tracking@1.0.0:status": "ok",
        "body_tracking@1.0.0:status": "ok",
        "sam2@1.0.0:status": "failed",
      },
    });
    render(<AssetsResultsPane {...defaultProps} items={[asset]} />);
    expect(screen.getByText(/2 ok/)).toBeTruthy();
    expect(screen.getByText(/1 failed/)).toBeTruthy();
  });

  it("highlights active preview row", () => {
    const { container } = render(
      <AssetsResultsPane {...defaultProps} activePreviewId="asset_001" />,
    );
    const highlightedRow = container.querySelector(".assets-active-preview-row");
    expect(highlightedRow).toBeTruthy();
  });

  it("does not highlight rows when no activePreviewId", () => {
    const { container } = render(
      <AssetsResultsPane {...defaultProps} activePreviewId={null} />,
    );
    const highlightedRow = container.querySelector(".assets-active-preview-row");
    expect(highlightedRow).toBeNull();
  });

  it("calls onRowClick when a data row is clicked", () => {
    const onRowClick = vi.fn();
    const { container } = render(
      <AssetsResultsPane {...defaultProps} onRowClick={onRowClick} />,
    );
    // Target the actual data row (has data-row-key attribute)
    const row = container.querySelector("tr[data-row-key='asset_001']");
    expect(row).toBeTruthy();
    if (row) fireEvent.click(row);
    expect(onRowClick).toHaveBeenCalledWith("asset_001");
  });

  it("only renders columns from selectedColumns", () => {
    render(
      <AssetsResultsPane
        {...defaultProps}
        selectedColumns={["asset_id", "env"]}
      />,
    );
    // Asset ID appears in header + measure row
    const assetIdHeaders = screen.getAllByText("Asset ID");
    expect(assetIdHeaders.length).toBeGreaterThanOrEqual(1);
    // 环境 column header present
    const envHeaders = screen.getAllByText("环境");
    expect(envHeaders.length).toBeGreaterThanOrEqual(1);
    // 时长 and 状态 should NOT be present
    expect(screen.queryByText("时长")).toBeNull();
    expect(screen.queryByText("状态")).toBeNull();
  });
});
