import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import OverviewTab from "./OverviewTab";
import type { Asset } from "../../api/types";

// Mock matchMedia for antd responsive components
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

const mockAsset: Asset = {
  asset_id: "test-asset-001",
  mcap_file_id: "mcap-001",
  start_timestamp_ns: 1000000000,
  end_timestamp_ns: 2000000000,
  duration_ms: 1000,
  duration_sec: 1.0,
  reviewer: "alice",
  status: "approved",
  lifecycle_state: "ready",
  owner: "team-a",
  asset_type: "segment",
  delivery_count: 0,
  algo_results: {},
  tags: {},
  files: {},
  lifecycle_meta: {},
  retention_tier: "standard",
  version: 1,
  created_at: "2024-01-01T00:00:00Z",
  updated_at: "2024-01-02T00:00:00Z",
};

describe("OverviewTab", () => {
  it("renders asset details", () => {
    render(<OverviewTab asset={mockAsset} />);
    expect(screen.getByText("test-asset-001")).toBeTruthy();
    expect(screen.getByText("mcap-001")).toBeTruthy();
    expect(screen.getByText("alice")).toBeTruthy();
    expect(screen.getByText("team-a")).toBeTruthy();
  });

  it("renders lifecycle state and asset type", () => {
    render(<OverviewTab asset={mockAsset} />);
    expect(screen.getByText("ready")).toBeTruthy();
    expect(screen.getByText("segment")).toBeTruthy();
  });

  it("renders retention tier", () => {
    render(<OverviewTab asset={mockAsset} />);
    expect(screen.getByText("standard")).toBeTruthy();
  });
});
