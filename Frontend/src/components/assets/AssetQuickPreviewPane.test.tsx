// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { Asset } from "../../api/types";
import type {
	FetchStatus,
	PreviewManifest,
} from "../../lib/assets/assetsDiscoveryTypes";
import AssetQuickPreviewPane from "./AssetQuickPreviewPane";

// Ant Design Descriptions requires matchMedia for responsive layout
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

const noop = () => {};

function makeAsset(overrides: Partial<Asset> = {}): Asset {
	return {
		asset_id: "asset-001",
		mcap_file_id: "mcap-001",
		start_timestamp_ns: 0,
		end_timestamp_ns: 1000000000,
		duration_ms: 120000,
		reviewer: "reviewer-a",
		status: "approved",
		lifecycle_state: "ready",
		owner: "owner-a",
		env: "warehouse",
		delivery_count: 0,
		algo_results: {
			"hand_tracking@1.0.0:status": "ok",
			"body_tracking@1.0.0:status": "failed",
			"body_tracking@1.0.0:reason": "timeout",
		},
		tags: { priority: "high", quality: "good", scene: "indoor" },
		files: { thumbnail: "gs://bucket/thumb.jpg", raw: "" },
		lifecycle_meta: {},
		created_at: "2024-01-01T00:00:00Z",
		updated_at: "2024-01-02T00:00:00Z",
		version: 1,
		...overrides,
	};
}

const defaultManifest: PreviewManifest = {
	thumbnailUrl: null,
	previewVideoUrl: null,
	availability: "missing",
	mode: "none",
};

function renderPane(
	overrides: Partial<Parameters<typeof AssetQuickPreviewPane>[0]> = {},
) {
	const props = {
		activeAssetId: null as string | null,
		fetchStatus: "idle" as FetchStatus,
		asset: null as Asset | null,
		previewManifest: null as PreviewManifest | null,
		collapsed: false,
		onCollapse: noop,
		onOpenDetail: noop,
		onFindSimilar: noop,
		...overrides,
	};
	return render(<AssetQuickPreviewPane {...props} />);
}

describe("AssetQuickPreviewPane", () => {
	it("renders empty state when no asset is selected", () => {
		renderPane();
		expect(screen.getByText("点击行查看预览")).toBeTruthy();
	});

	it("renders collapsed state as thin bar with expand button", () => {
		const { container } = renderPane({
			collapsed: true,
			activeAssetId: "asset-001",
		});
		const bar = container.firstElementChild as HTMLElement;
		expect(bar.style.width).toBe("36px");
	});

	it("calls onCollapse when expand button is clicked in collapsed state", () => {
		const fn = vi.fn();
		renderPane({
			collapsed: true,
			activeAssetId: "asset-001",
			onCollapse: fn,
		});
		const btn = screen.getByTitle("展开预览");
		fireEvent.click(btn);
		expect(fn).toHaveBeenCalledOnce();
	});

	it("calls onCollapse when collapse button is clicked in expanded state", () => {
		const fn = vi.fn();
		renderPane({ onCollapse: fn });
		const btn = screen.getByRole("button", { name: "收起预览" });
		fireEvent.click(btn);
		expect(fn).toHaveBeenCalledOnce();
	});

	it("renders skeleton when loading", () => {
		const { container } = renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "loading",
		});
		expect(container.querySelector(".ant-skeleton")).toBeTruthy();
	});

	it("renders loaded content with asset data", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getByText("asset-001")).toBeTruthy();
		expect(screen.getByText("ready")).toBeTruthy();
		expect(screen.getByText("high")).toBeTruthy();
		expect(screen.getByText("good")).toBeTruthy();
	});

	it("renders summary descriptions", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getByText("mcap-001")).toBeTruthy();
		expect(screen.getByText("warehouse")).toBeTruthy();
		expect(screen.getByText("120.0s")).toBeTruthy();
		expect(screen.getByText("owner-a")).toBeTruthy();
		expect(screen.getByText("reviewer-a")).toBeTruthy();
	});

	it("calls onMcapClick when MCAP id is clicked", () => {
		const fn = vi.fn();
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
			onMcapClick: fn,
		});
		fireEvent.click(screen.getByLabelText("MCAP mcap-001"));
		expect(fn).toHaveBeenCalledWith("mcap-001");
	});

	it("renders algo summary with status counts", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getByText(/1 ok/)).toBeTruthy();
		expect(screen.getByText(/1 failed/)).toBeTruthy();
	});

	it("renders failure reason when algo has failed", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getByText(/timeout/)).toBeTruthy();
	});

	it("renders file keys with indicators", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getByText("thumbnail")).toBeTruthy();
		expect(screen.getByText("raw")).toBeTruthy();
	});

	it("renders preview unavailable placeholder", () => {
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
		});
		expect(screen.getAllByText("暂无预览").length).toBeGreaterThan(0);
	});

	it("renders 查看详情 button and calls onOpenDetail", () => {
		const fn = vi.fn();
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
			onOpenDetail: fn,
		});
		fireEvent.click(screen.getByText("查看详情"));
		expect(fn).toHaveBeenCalledWith("asset-001");
	});

	it("renders Find Similar button and calls onFindSimilar when clicked", () => {
		const fn = vi.fn();
		const asset = makeAsset();
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "success",
			asset,
			previewManifest: defaultManifest,
			onFindSimilar: fn,
		});
		fireEvent.click(screen.getByText("查找相似"));
		expect(fn).toHaveBeenCalledWith("asset-001");
	});

	it("shows error state when fetchStatus is error and no asset", () => {
		renderPane({
			activeAssetId: "asset-001",
			fetchStatus: "error",
			asset: null,
		});
		expect(screen.getByText("加载预览失败")).toBeTruthy();
	});
});
