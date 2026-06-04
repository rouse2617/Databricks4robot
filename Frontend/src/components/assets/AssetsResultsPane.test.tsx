// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { Asset } from "../../api/types";
import AssetsResultsPane from "./AssetsResultsPane";

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
	selectedColumns: [
		"asset_id",
		"duration",
		"env",
		"lifecycle_state",
		"algo",
		"updated_at",
	],
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
		render(
			<AssetsResultsPane {...defaultProps} total={100} totalApprox={true} />,
		);
		const elements = screen.getAllByText(/共 100\+ 条/);
		expect(elements.length).toBeGreaterThanOrEqual(1);
	});

	it("renders updated_at sortable header with current direction", () => {
		render(<AssetsResultsPane {...defaultProps} sort="-updated_at" />);
		expect(
			screen.getAllByRole("button", { name: "按更新时间排序" }).length,
		).toBeGreaterThanOrEqual(1);
	});

	it("calls onSortChange when updated_at header is clicked", () => {
		const onSortChange = vi.fn();
		render(
			<AssetsResultsPane
				{...defaultProps}
				sort="-updated_at"
				onSortChange={onSortChange}
			/>,
		);
		fireEvent.click(screen.getByRole("button", { name: "按更新时间排序" }));
		expect(onSortChange).toHaveBeenCalledWith("updated_at");
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

	it("shows failed algo indicator beside lifecycle when algos failed", () => {
		const asset = makeAsset({
			lifecycle_state: "ready",
			algo_results: {
				"hand_tracking@1.2.0:status": "failed",
				"env_analysis@1.0.0:status": "failed",
				"body_tracking@1.0.0:status": "ok",
			},
		});
		render(
			<AssetsResultsPane
				{...defaultProps}
				items={[asset]}
				selectedColumns={[
					"asset_id",
					"lifecycle_state",
					"updated_at",
				]}
			/>,
		);
		expect(screen.getByText("ready")).toBeTruthy();
		expect(screen.getByText("2 算法失败")).toBeTruthy();
	});

	it("hides failed algo indicator when all algos are healthy", () => {
		const asset = makeAsset({
			lifecycle_state: "ready",
			algo_results: {
				"hand_tracking@1.0.0:status": "ok",
			},
		});
		render(
			<AssetsResultsPane
				{...defaultProps}
				items={[asset]}
				selectedColumns={["asset_id", "lifecycle_state", "updated_at"]}
			/>,
		);
		expect(screen.queryByText(/算法失败/)).toBeNull();
	});

	it("highlights active preview row", () => {
		const { container } = render(
			<AssetsResultsPane {...defaultProps} activePreviewId="asset_001" />,
		);
		const highlightedRow = container.querySelector(
			".assets-active-preview-row",
		);
		expect(highlightedRow).toBeTruthy();
	});

	it("does not highlight rows when no activePreviewId", () => {
		const { container } = render(
			<AssetsResultsPane {...defaultProps} activePreviewId={null} />,
		);
		const highlightedRow = container.querySelector(
			".assets-active-preview-row",
		);
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

	it("opens preview when asset id text is clicked", () => {
		const onRowClick = vi.fn();
		render(<AssetsResultsPane {...defaultProps} onRowClick={onRowClick} />);
		fireEvent.click(screen.getByTestId("asset-id-link-asset_001"));
		expect(onRowClick).toHaveBeenCalledWith("asset_001");
	});

	it("calls onMcapClick when MCAP id text is clicked", () => {
		const onMcapClick = vi.fn();
		render(
			<AssetsResultsPane
				{...defaultProps}
				selectedColumns={["asset_id", "mcap_file_id"]}
				onMcapClick={onMcapClick}
			/>,
		);
		fireEvent.click(screen.getByLabelText("MCAP mcap_001"));
		expect(onMcapClick).toHaveBeenCalledWith("mcap_001");
	});

	it("calls onMcapClick from card view MCAP button", () => {
		const onMcapClick = vi.fn();
		render(
			<AssetsResultsPane
				{...defaultProps}
				viewMode="card"
				onMcapClick={onMcapClick}
			/>,
		);
		fireEvent.click(screen.getByText(/mcap_001/));
		expect(onMcapClick).toHaveBeenCalledWith("mcap_001");
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
