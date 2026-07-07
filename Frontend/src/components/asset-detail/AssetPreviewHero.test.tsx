// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { Asset } from "../../api/types";
import type { PreviewManifest } from "../../lib/assets/assetsDiscoveryTypes";
import AssetPreviewHero from "./AssetPreviewHero";

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

function makeAsset(overrides: Partial<Asset> = {}): Asset {
	return {
		asset_id: "asset-001",
		mcap_file_id: "mcap-001",
		start_timestamp_ns: 0,
		end_timestamp_ns: 1000000000,
		duration_ms: 7429,
		reviewer: "reviewer-a",
		status: "approved",
		lifecycle_state: "ready",
		owner: "owner-a",
		env: "indoor",
		delivery_count: 0,
		algo_results: {
			"algo@1.0.0:status": "pending",
		},
		tags: { priority: "medium", quality: "good" },
		files: { raw_mcap: "mcap-001" },
		lifecycle_meta: {},
		created_at: "2024-01-01T00:00:00Z",
		updated_at: "2024-01-02T00:00:00Z",
		version: 1,
		...overrides,
	};
}

describe("AssetPreviewHero", () => {
	it("renders native <video> when manifest exposes a preview URL", () => {
		const manifest: PreviewManifest = {
			thumbnailUrl: null,
			previewVideoUrl:
				"/api/v1/preview/assets/asset-001/segment.mp4?topic=%2Fcamera%2Ffront",
			mcapUrl: "https://example.com/a.mcap",
			availability: "ready",
			mode: "mcap",
			ds: "remote-file",
			dsParams: { url: "https://example.com/a.mcap", asset_id: "asset-001" },
		};
		const { container } = render(
			<MemoryRouter>
				<AssetPreviewHero asset={makeAsset()} previewManifest={manifest} />
			</MemoryRouter>,
		);
		const video = container.querySelector("video");
		expect(video).not.toBeNull();
		expect(video?.getAttribute("src")).toContain("segment.mp4");
		expect(screen.getByText("视频预览")).toBeTruthy();
	});

	it("falls back to placeholder when no preview video exists", () => {
		const manifest: PreviewManifest = {
			thumbnailUrl: null,
			previewVideoUrl: null,
			availability: "missing",
			mode: "none",
		};
		const { container } = render(
			<MemoryRouter>
				<AssetPreviewHero asset={makeAsset()} previewManifest={manifest} />
			</MemoryRouter>,
		);
		expect(container.querySelector("video")).toBeNull();
		expect(screen.getAllByText("暂无预览").length).toBeGreaterThan(0);
	});

	it("labels delivery_count as completed deliveries", () => {
		render(
			<MemoryRouter>
				<AssetPreviewHero
					asset={makeAsset({ delivery_count: 2 })}
					previewManifest={null}
				/>
			</MemoryRouter>,
		);
		expect(screen.getByText("已完成交付")).toBeTruthy();
		expect(screen.queryByText("Deliveries")).toBeNull();
	});
});
