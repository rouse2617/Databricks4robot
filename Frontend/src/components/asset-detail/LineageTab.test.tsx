// @vitest-environment jsdom
// CYB-3388 — LineageTab rendering rules:
//   - raw_mcap asset (asset_id === mcap_file_id): hide self-reference rows
//   - non-raw_mcap asset: show mcap_file_id / URI / ingest_state
//   - downstream shows only children (algo/delivery/eval are their own tabs)

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import { assetsApi } from "../../api/assets";
import LineageTab from "./LineageTab";

beforeAll(() => {
	// antd `<Row>/<Col>` calls responsiveObserver which uses window.matchMedia;
	// jsdom doesn't ship one, so we stub it out to keep the tests headless-safe.
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

vi.mock("../../api/assets", () => ({
	assetsApi: {
		getLineage: vi.fn(),
		getPipelineLineage: vi.fn(),
	},
}));

const rawMcapLineage = {
	asset_id: "qrZQGxpm",
	upstream: {
		mcap_file_id: "qrZQGxpm",
		mcap_uri: "gs://co-prod-gv-cybercap/raw/abc.mcap",
		ingest_state: "pending",
	},
	downstream: {
		algo_results: [
			{ algo_name: "transcode", algo_version: "1.0.0", status: "ok" },
		],
		deliveries: [{ delivery_id: "d1", customer_id: "cust-a" }],
		eval_results: [
			{ eval_name: "quality", metric_key: "score", metric_value: 0.9 },
		],
		children: [
			{ asset_id: "seg1", asset_type: "segment" },
			{ asset_id: "seg2", asset_type: "segment" },
		],
	},
};

const segmentLineage = {
	asset_id: "seg1",
	upstream: {
		mcap_file_id: "parentMcap",
		mcap_uri: "gs://bucket/x.mcap",
		ingest_state: "summarized",
	},
	downstream: {
		algo_results: [],
		deliveries: [],
		eval_results: [],
		children: [],
	},
};

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(assetsApi.getPipelineLineage).mockResolvedValue({} as never);
});

afterEach(cleanup);

describe("LineageTab — CYB-3388", () => {
	it("raw_mcap (self-sourced): retitles card to 存储 and hides mcap_file_id + ingest_state rows", async () => {
		vi.mocked(assetsApi.getLineage).mockResolvedValue(rawMcapLineage as never);
		render(
			<MemoryRouter>
				<LineageTab assetId="qrZQGxpm" assetType="raw_mcap" />
			</MemoryRouter>,
		);
		await waitFor(() => {
			expect(screen.getByText("存储")).toBeTruthy();
		});
		// URI still shown — it's the useful info
		expect(
			screen.getByText("gs://co-prod-gv-cybercap/raw/abc.mcap"),
		).toBeTruthy();
		// mcap_file_id (self-reference) hidden
		expect(screen.queryByText("MCAP File ID")).toBeNull();
		// ingest_state hidden (killing C ambiguity)
		expect(screen.queryByText("入库状态")).toBeNull();
	});

	it("segment (not self-sourced): shows 上游 card with mcap_file_id / URI / 入库状态", async () => {
		vi.mocked(assetsApi.getLineage).mockResolvedValue(segmentLineage as never);
		render(
			<MemoryRouter>
				<LineageTab assetId="seg1" assetType="segment" />
			</MemoryRouter>,
		);
		await waitFor(() => {
			expect(screen.getByText("上游")).toBeTruthy();
		});
		expect(screen.getByText("MCAP File ID")).toBeTruthy();
		expect(screen.getByText("parentMcap")).toBeTruthy();
		expect(screen.getByText("入库状态")).toBeTruthy();
	});

	it("downstream card only lists children (algo / deliveries / eval sections gone)", async () => {
		vi.mocked(assetsApi.getLineage).mockResolvedValue(rawMcapLineage as never);
		render(
			<MemoryRouter>
				<LineageTab assetId="qrZQGxpm" assetType="raw_mcap" />
			</MemoryRouter>,
		);
		await waitFor(() => {
			expect(screen.getByText("子资产")).toBeTruthy();
		});
		// Child asset ids appear
		expect(screen.getByText("seg1")).toBeTruthy();
		expect(screen.getByText("seg2")).toBeTruthy();
		// The old duplicated sections no longer render
		expect(screen.queryByText("算法处理")).toBeNull();
		expect(screen.queryByText("交付记录")).toBeNull();
		expect(screen.queryByText("评测结果")).toBeNull();
	});

	it("falls back to self-source detection via mcap_file_id === assetId even without assetType prop", async () => {
		vi.mocked(assetsApi.getLineage).mockResolvedValue(rawMcapLineage as never);
		render(
			<MemoryRouter>
				<LineageTab assetId="qrZQGxpm" />
			</MemoryRouter>,
		);
		await waitFor(() => {
			expect(screen.getByText("存储")).toBeTruthy();
		});
		expect(screen.queryByText("MCAP File ID")).toBeNull();
	});
});
