// @vitest-environment jsdom

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import AnalyticsPage from "./AnalyticsPage";

// Polyfill matchMedia for Ant Design's responsive observer
beforeEach(() => {
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

// Mock echarts-for-react to avoid canvas issues in jsdom
vi.mock("echarts-for-react", () => ({
	default: (props: { style?: React.CSSProperties }) => (
		<div data-testid="echarts-mock" style={props.style}>
			ECharts Mock
		</div>
	),
}));

// Mock lakehouse API
const mockStatus = vi.fn();
const mockTables = vi.fn();
const mockSyncStatus = vi.fn();
const mockTrainingAssets = vi.fn();
const mockRecomputeCandidates = vi.fn();
const mockTagTimeline = vi.fn();
const mockQualityDistribution = vi.fn();
const mockCustomerReplay = vi.fn();

vi.mock("../api/lakehouse", () => ({
	lakehouseApi: {
		status: () => mockStatus(),
		tables: () => mockTables(),
		syncStatus: () => mockSyncStatus(),
		trainingAssets: () => mockTrainingAssets(),
		recomputeCandidates: () => mockRecomputeCandidates(),
		tagTimeline: () => mockTagTimeline(),
		qualityDistribution: () => mockQualityDistribution(),
		customerReplay: () => mockCustomerReplay(),
	},
}));

vi.mock("../lib/apiError", () => ({
	extractApiErrorMessage: (_err: unknown, fallback: string) => fallback,
}));

afterEach(() => {
	cleanup();
	vi.clearAllMocks();
});

function setupMocks(overrides?: {
	statusError?: boolean;
	tablesError?: boolean;
	trinoUnhealthy?: boolean;
	syncAlert?: boolean;
	syncSource?: "realtime" | "sync_reconciliation";
}) {
	if (overrides?.statusError) {
		mockStatus.mockRejectedValue(new Error("status error"));
	} else if (overrides?.trinoUnhealthy) {
		mockStatus.mockResolvedValue({
			enabled: true,
			healthy: false,
			catalog: "iceberg",
			schema: "robot",
			error: "trino query layer is disabled",
		});
	} else {
		mockStatus.mockResolvedValue({
			enabled: true,
			healthy: true,
			catalog: "iceberg",
			schema: "robot",
		});
	}

	if (overrides?.tablesError) {
		mockTables.mockRejectedValue(new Error("tables error"));
	} else {
		mockTables.mockResolvedValue({
			items: [
				{ table_name: "silver_assets_current", row_count: 1000 },
				{ table_name: "bronze_asset_algo_events", row_count: 5000 },
				{ table_name: "gold_dataset_snapshot_items", row_count: 200 },
			],
		});
	}

	const diffPct = overrides?.syncAlert ? 0.1 : 0.001;
	mockSyncStatus.mockResolvedValue({
		available: true,
		source: overrides?.syncSource ?? "sync_reconciliation",
		data: {
			dagster_run_id: "run-123",
			checked_at: "2025-01-15T10:00:00Z",
			pg_total_count: 1000,
			iceberg_total_count: overrides?.syncAlert ? 900 : 999,
			count_diff_pct: diffPct,
			pg_status_dist: { ready: 800, processing: 200 },
			iceberg_status_dist: { ready: 799, processing: 200 },
			status_diff: {
				ready: { pg: 800, iceberg: 799, diff: 1 },
				processing: { pg: 200, iceberg: 200, diff: 0 },
			},
			is_alert: overrides?.syncAlert ?? false,
			iceberg_max_seq: overrides?.syncSource === "realtime" ? 12345 : undefined,
		},
	});

	mockQualityDistribution.mockResolvedValue({
		items: [
			{ quality: "excellent", count: 50 },
			{ quality: "good", count: 100 },
			{ quality: "poor", count: 10 },
		],
	});

	mockTrainingAssets.mockResolvedValue({ items: [] });
	mockRecomputeCandidates.mockResolvedValue({ items: [] });
	mockTagTimeline.mockResolvedValue({ items: [] });
	mockCustomerReplay.mockResolvedValue({ items: [] });
}

describe("AnalyticsPage", () => {
	it("renders the page title", async () => {
		setupMocks();
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByText("湖仓验证")).toBeTruthy();
		});
	});

	it("renders visualization charts when data is available", async () => {
		setupMocks();
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByTestId("quality-distribution-chart")).toBeTruthy();
			expect(screen.getByTestId("table-row-count-chart")).toBeTruthy();
		});
	});

	it("renders sync status card with green light when data is consistent", async () => {
		setupMocks();
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByTestId("sync-status-card")).toBeTruthy();
			expect(screen.getByTestId("sync-traffic-light")).toBeTruthy();
			expect(screen.getAllByText("数据一致").length).toBeGreaterThanOrEqual(1);
		});
	});

	it("renders sync status card with red light when alert is triggered", async () => {
		setupMocks({ syncAlert: true });
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getAllByText("差异告警").length).toBeGreaterThanOrEqual(1);
		});
	});

	it("shows PG and Iceberg counts", async () => {
		setupMocks();
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByTestId("pg-count").textContent).toBe("1,000");
			expect(screen.getByTestId("iceberg-count").textContent).toBe("999");
		});
	});

	it("renders realtime sync source without requiring dagster metadata", async () => {
		setupMocks({ syncSource: "realtime" });
		mockSyncStatus.mockResolvedValue({
			available: true,
			source: "realtime",
			data: {
				dagster_run_id: "",
				checked_at: "2025-01-15T10:00:00Z",
				pg_total_count: 1000,
				iceberg_total_count: 999,
				count_diff_pct: 0.001,
				pg_status_dist: {},
				iceberg_status_dist: {},
				status_diff: {},
				is_alert: false,
				iceberg_max_seq: 12345,
			},
		});
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByTestId("sync-source").textContent).toBe("实时检查");
			expect(screen.getByText("12,345")).toBeTruthy();
		});
	});

	it("shows warning when both critical endpoints fail", async () => {
		setupMocks({ statusError: true, tablesError: true });
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByText("Trino 湖仓查询暂不可用")).toBeTruthy();
		});
	});

	it("does not call tables or training APIs when Trino reports unhealthy", async () => {
		setupMocks({ trinoUnhealthy: true });
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByText("湖仓验证")).toBeTruthy();
		});
		expect(mockTables).not.toHaveBeenCalled();
		expect(mockTrainingAssets).not.toHaveBeenCalled();
		expect(mockQualityDistribution).not.toHaveBeenCalled();
	});

	it("renders statistic cards with table row counts", async () => {
		setupMocks();
		render(<AnalyticsPage />);
		await waitFor(() => {
			expect(screen.getByText("Silver 当前资产")).toBeTruthy();
			expect(screen.getByText("Bronze 算法事件")).toBeTruthy();
			expect(screen.getByText("Gold 训练候选")).toBeTruthy();
		});
	});
});
