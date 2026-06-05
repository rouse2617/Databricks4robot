// @vitest-environment jsdom

import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { Asset, McapFile } from "../api/types";
import McapFileDetailPage from "./McapFileDetailPage";

const apiMocks = vi.hoisted(() => ({
	getMcapFile: vi.fn(),
	listAssets: vi.fn(),
}));

vi.mock("../api/mcapFiles", () => ({
	mcapFilesApi: {
		get: apiMocks.getMcapFile,
	},
}));

vi.mock("../api/assets", () => ({
	assetsApi: {
		list: apiMocks.listAssets,
	},
}));

const mcapFile: McapFile = {
	mcap_file_id: "Y2hki98u",
	gcs_path: "gs://databrew-dev/mcap/Y2hki98u.mcap",
	size_bytes: 2048,
	raw_hash_md5: "abc123",
	ingest_state: "summarized",
	start_timestamp_ns: 1_700_000_000_000_000_000,
	end_timestamp_ns: 1_700_000_010_000_000_000,
	channel_count: 3,
	chunk_count: 9,
	owner: "qa",
	process_state: { summary: "ok" },
	created_at: "2026-06-01T01:00:00Z",
	updated_at: "2026-06-01T02:00:00Z",
	version: 2,
};

const asset: Asset = {
	asset_id: "asset001",
	mcap_file_id: "Y2hki98u",
	start_timestamp_ns: 1,
	end_timestamp_ns: 2,
	duration_sec: 10,
	reviewer: "",
	lifecycle_state: "ready",
	owner: "qa",
	env: "dev",
	delivery_count: 0,
	algo_results: { "count-lines:status": "ok" },
	tags: {},
	files: {},
	lifecycle_meta: {},
	created_at: "2026-06-01T01:00:00Z",
	updated_at: "2026-06-01T02:00:00Z",
	version: 1,
};

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
	Object.assign(navigator, {
		clipboard: {
			writeText: vi.fn(),
		},
	});
});

beforeEach(() => {
	apiMocks.getMcapFile.mockReset();
	apiMocks.listAssets.mockReset();
	apiMocks.getMcapFile.mockResolvedValue(mcapFile);
	apiMocks.listAssets.mockResolvedValue({ items: [asset], total: 1 });
});

function renderDetail(path = "/mcap-files/Y2hki98u") {
	return render(
		<MemoryRouter initialEntries={[path]}>
			<Routes>
				<Route path="/mcap-files/:id" element={<McapFileDetailPage />} />
			</Routes>
		</MemoryRouter>,
	);
}

describe("McapFileDetailPage", () => {
	it("loads mcap metadata and related assets from the route id", async () => {
		renderDetail();

		expect(await screen.findByText("MCAP 文件详情")).toBeTruthy();
		expect((await screen.findAllByText("Y2hki98u")).length).toBeGreaterThan(0);
		expect(
			screen.getByText("gs://databrew-dev/mcap/Y2hki98u.mcap"),
		).toBeTruthy();
		expect(await screen.findByText(/asset001/)).toBeTruthy();

		expect(apiMocks.getMcapFile).toHaveBeenCalledWith("Y2hki98u");
		await waitFor(() => {
			expect(apiMocks.listAssets).toHaveBeenCalledWith({
				mcap_file_id: "Y2hki98u",
				page: 1,
				page_size: 100,
			});
		});
	});

	it("renders a retryable error state when the mcap request fails", async () => {
		apiMocks.getMcapFile.mockRejectedValueOnce(new Error("not found"));

		renderDetail();

		expect(await screen.findByText("加载 MCAP 文件详情失败")).toBeTruthy();
		expect(screen.getByRole("button", { name: /重试/ })).toBeTruthy();
	});
});
