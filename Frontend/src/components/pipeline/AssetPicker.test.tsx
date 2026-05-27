// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { searchApi } from "../../api/search";
import type { SearchAssetResult } from "../../api/search";
import AssetPicker from "./AssetPicker";

vi.mock("../../api/search", () => ({
  searchApi: {
    searchAssets: vi.fn(),
  },
}));

function makeSearchResult(overrides: Partial<SearchAssetResult>): SearchAssetResult {
  return {
    asset_id: "ast-000",
    mcap_file_id: "mcap-000",
    start_timestamp_ns: 0,
    end_timestamp_ns: 0,
    reviewer: "test",
    owner: "test",
    delivery_count: 0,
    algo_results: {},
    tags: {},
    files: {},
    lifecycle_meta: {},
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    version: 1,
    ...overrides,
  };
}

const mockResults: SearchAssetResult[] = [
  makeSearchResult({
    asset_id: "ast-001",
    asset_type: "dataset",
    lifecycle_state: "active",
    storage_uri: "gs://bucket/path/to/dataset-v1",
  }),
  makeSearchResult({
    asset_id: "ast-002",
    asset_type: "model",
    lifecycle_state: "archived",
    storage_uri: "s3://another-bucket/very/long/path/that/should/be/truncated/by/the/asset/picker/component",
  }),
];

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

afterEach(() => {
  vi.restoreAllMocks();
  cleanup();
});

describe("AssetPicker", () => {
  it("renders search input with default placeholder", () => {
    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    expect(
      screen.getByPlaceholderText("搜索资产（输入 asset_id 或名称）"),
    ).toBeTruthy();
  });

  it("shows empty state with hint text when no query and no results", () => {
    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    expect(
      screen.getByText("输入关键字搜索资产，不选择则直接部署"),
    ).toBeTruthy();
  });

  it("calls searchApi.searchAssets on search and displays results", async () => {
    vi.mocked(searchApi.searchAssets).mockResolvedValue({
      items: mockResults,
      total: 2,
      page: 1,
      page_size: 50,
    });

    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    expect(searchButton).toBeTruthy();

    fireEvent.change(input, { target: { value: "ast" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(searchApi.searchAssets).toHaveBeenCalledWith({
        q: "ast",
        page_size: 50,
      });
    });

    expect(screen.getByText("ast-001")).toBeTruthy();
    expect(screen.getByText("ast-002")).toBeTruthy();
    expect(screen.getByText("dataset")).toBeTruthy();
    expect(screen.getByText("model")).toBeTruthy();
  });

  it("truncates long storage_uri values", async () => {
    vi.mocked(searchApi.searchAssets).mockResolvedValue({
      items: mockResults,
      total: 2,
      page: 1,
      page_size: 50,
    });

    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    fireEvent.change(input, { target: { value: "ast" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(screen.getByText("gs://bucket/path/to/dataset-v1")).toBeTruthy();
    });

    // Long URI gets truncated by component logic + Ant Table column width
    const longUriCell = screen.getByText(/^s3:\/\/another-bucket/);
    expect(longUriCell.textContent!.length).toBeLessThan(
      "s3://another-bucket/very/long/path/that/should/be/truncated/by/the/asset/picker/component"
        .length,
    );
    expect(longUriCell.textContent).toMatch(/…$/);
  });

  it("shows dash for missing storage_uri", async () => {
    vi.mocked(searchApi.searchAssets).mockResolvedValue({
      items: [
        makeSearchResult({
          asset_id: "ast-003",
          asset_type: "dataset",
          lifecycle_state: "active",
        }),
      ],
      total: 1,
      page: 1,
      page_size: 50,
    });

    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    fireEvent.change(input, { target: { value: "no-uri" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(screen.getByText("—")).toBeTruthy();
    });
  });

  it("calls onSelectionChange when rows are selected", async () => {
    vi.mocked(searchApi.searchAssets).mockResolvedValue({
      items: mockResults,
      total: 2,
      page: 1,
      page_size: 50,
    });

    const onSelectionChange = vi.fn();
    render(
      <AssetPicker
        selectedIds={[]}
        onSelectionChange={onSelectionChange}
      />,
    );

    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    fireEvent.change(input, { target: { value: "ast" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(screen.getByText("ast-001")).toBeTruthy();
    });

    const checkboxes = document.querySelectorAll(
      ".ant-table-row input[type='checkbox']",
    );
    expect(checkboxes.length).toBeGreaterThan(0);
    fireEvent.click(checkboxes[0]);

    await waitFor(() => {
      expect(onSelectionChange).toHaveBeenCalled();
      expect(onSelectionChange.mock.calls[0][0]).toContain("ast-001");
    });
  });

  it("shows '未找到匹配的资产' when search yields no results", async () => {
    vi.mocked(searchApi.searchAssets).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50,
    });

    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    fireEvent.change(input, { target: { value: "zzznonexistent" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(screen.getByText("未找到匹配的资产")).toBeTruthy();
    });
  });

  it("handles search API failure gracefully", async () => {
    vi.mocked(searchApi.searchAssets).mockRejectedValue(
      new Error("Network error"),
    );

    render(<AssetPicker selectedIds={[]} onSelectionChange={() => {}} />);
    const input = screen.getByPlaceholderText(
      "搜索资产（输入 asset_id 或名称）",
    );
    const searchButton = input.parentElement?.querySelector("button");
    fireEvent.change(input, { target: { value: "ast" } });
    fireEvent.click(searchButton!);

    await waitFor(() => {
      expect(searchApi.searchAssets).toHaveBeenCalled();
    });
  });

  it("accepts custom placeholder and maxHeight props", () => {
    render(
      <AssetPicker
        selectedIds={[]}
        onSelectionChange={() => {}}
        placeholder="自定义搜索"
        maxHeight={400}
      />,
    );
    expect(screen.getByPlaceholderText("自定义搜索")).toBeTruthy();
  });
});
