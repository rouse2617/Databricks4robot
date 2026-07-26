import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Asset } from "../../api/types";
import OverviewTab from "./OverviewTab";

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
		expect(screen.getAllByText("test-asset-001").length).toBeGreaterThan(0);
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

	// CYB-4011: Grace video id row on the asset overview.
	it("renders grace_video_id with a Grace link when present", () => {
		render(
			<OverviewTab
				asset={{
					...mockAsset,
					grace_video_id: "019f9893-3456-7376-ae68-30a89227eb46",
				}}
			/>,
		);
		expect(
			screen.getByText("019f9893-3456-7376-ae68-30a89227eb46"),
		).toBeTruthy();
		const link = screen
			.getAllByRole("link")
			.find(
				(a) =>
					a.getAttribute("href") ===
					"https://grace.cyberorigin.ai/videos/019f9893-3456-7376-ae68-30a89227eb46",
			);
		expect(link).toBeTruthy();
	});

	it("shows placeholder when grace_video_id is absent", () => {
		render(<OverviewTab asset={mockAsset} />);
		expect(screen.getByText("未关联")).toBeTruthy();
	});
});
