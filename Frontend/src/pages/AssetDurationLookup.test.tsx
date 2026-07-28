// CYB-4294: AssetDurationLookup page tests.
//
// Covers: pure helpers (parseIdBlob, bucketItems, exportDurationsCsv) plus a
// smoke render that submits the form and asserts stats/table + CSV download.

import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { parseIdBlob } from "../components/batch-lookup/types";
import AssetDurationLookup from "./AssetDurationLookup";
import { bucketItems, exportDurationsCsv } from "./presets/durations";

const { lookupDurationsMock } = vi.hoisted(() => ({
	lookupDurationsMock: vi.fn(),
}));

vi.mock("../api/assets", () => ({
	assetsApi: {
		lookupDurations: lookupDurationsMock,
	},
}));

// AntD components rely on matchMedia; jsdom does not implement it.
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

describe("parseIdBlob", () => {
	it("splits on whitespace and commas, trims, dedupes, and reports duplicates", () => {
		const result = parseIdBlob("a1, a2\n a3\n\na1 a4,,a2");
		expect(result.ids).toEqual(["a1", "a2", "a3", "a4"]);
		expect(result.duplicates).toBe(2); // a1 and a2 each show up twice
	});
	it("returns empty on empty input", () => {
		expect(parseIdBlob("")).toEqual({ ids: [], duplicates: 0 });
		expect(parseIdBlob("   \n  ")).toEqual({ ids: [], duplicates: 0 });
	});
});

describe("bucketItems", () => {
	it("groups items into the fixed 5 buckets", () => {
		const items = [
			{ input_id: "a", asset_id: "a", duration_ms: 0, duration_sec: 0, formatted: "0s" },
			{ input_id: "b", asset_id: "b", duration_ms: 30_000, duration_sec: 30, formatted: "30s" }, // <1min
			{ input_id: "c", asset_id: "c", duration_ms: 300_000, duration_sec: 300, formatted: "5m 0s" }, // 1-10min
			{ input_id: "d", asset_id: "d", duration_ms: 1_200_000, duration_sec: 1200, formatted: "20m 0s" }, // 10-30min
			{ input_id: "e", asset_id: "e", duration_ms: 2_700_000, duration_sec: 2700, formatted: "45m 0s" }, // 30-60min
			{ input_id: "f", asset_id: "f", duration_ms: 7_200_000, duration_sec: 7200, formatted: "2h 0m" }, // 60min+
		];
		const buckets = bucketItems(items);
		expect(buckets.map((b) => b.count)).toEqual([2, 1, 1, 1, 1]);
	});
});

describe("exportDurationsCsv", () => {
	beforeEach(() => {
		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
	});

	it("triggers an anchor.click with the expected filename", () => {
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});
		exportDurationsCsv(
			[
				{
					input_id: "aaaaaaaa",
					asset_id: "aaaaaaaa",
					duration_ms: 1_000,
					duration_sec: 1,
					formatted: "1s",
				},
			],
			1,
		);
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-durations-1.csv");
	});
});

describe("AssetDurationLookup page", () => {
	beforeEach(() => {
		lookupDurationsMock.mockReset();
		// Restore mocks between tests so anchor-click spies don't stack.
		vi.restoreAllMocks();
	});

	it("submits the parsed ids and renders stats + table rows on success", async () => {
		lookupDurationsMock.mockResolvedValue({
			items: [
				{
					input_id: "aaaaaaaa",
					asset_id: "aaaaaaaa",
					duration_ms: 5_000,
					duration_sec: 5,
					formatted: "5s",
				},
				{
					input_id: "019f8319-0ef3-7da0-815c-30f23d21e7f1",
					asset_id: "cccccccc",
					grace_video_id: "019f8319-0ef3-7da0-815c-30f23d21e7f1",
					duration_ms: 120_000,
					duration_sec: 120,
					formatted: "2m 0s",
				},
			],
			missing_ids: ["zzzzzzzz"],
			filtered_out_ids: [],
			stats: {
				matched_count: 2,
				missing_count: 1,
				filtered_out_count: 0,
				total_ms: 125_000,
				mean_ms: 62_500,
				min_ms: 5_000,
				max_ms: 120_000,
				p50_ms: 62_500,
				p90_ms: 108_500,
			},
		});

		render(
			<MemoryRouter>
				<AssetDurationLookup />
			</MemoryRouter>,
		);

		const textarea = screen.getByTestId("ids-textarea") as HTMLTextAreaElement;
		fireEvent.change(textarea, {
			target: {
				value: "aaaaaaaa\n019f8319-0ef3-7da0-815c-30f23d21e7f1\nzzzzzzzz",
			},
		});

		const button = screen.getByTestId("submit-button");
		fireEvent.click(button);

		await waitFor(() => {
			expect(lookupDurationsMock).toHaveBeenCalledTimes(1);
		});
		expect(lookupDurationsMock).toHaveBeenCalledWith(
			expect.objectContaining({
				ids: [
					"aaaaaaaa",
					"019f8319-0ef3-7da0-815c-30f23d21e7f1",
					"zzzzzzzz",
				],
			}),
		);

		// Table rendered with 2 data rows.
		const table = await screen.findByTestId("results-table");
		const rows = within(table).getAllByRole("row");
		// Row 0 is the header row.
		expect(rows.length).toBe(1 + 2);

		// Stats card shows matched_count=2 somewhere.
		expect(screen.getByText("matched")).toBeTruthy();
	});

	it("CSV export button triggers a download with the matched-count filename", async () => {
		lookupDurationsMock.mockResolvedValue({
			items: [
				{
					input_id: "aaaaaaaa",
					asset_id: "aaaaaaaa",
					duration_ms: 5_000,
					duration_sec: 5,
					formatted: "5s",
				},
			],
			missing_ids: [],
			filtered_out_ids: [],
			stats: {
				matched_count: 1,
				missing_count: 0,
				filtered_out_count: 0,
				total_ms: 5_000,
				mean_ms: 5_000,
				min_ms: 5_000,
				max_ms: 5_000,
				p50_ms: 5_000,
				p90_ms: 5_000,
			},
		});

		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});

		render(
			<MemoryRouter>
				<AssetDurationLookup />
			</MemoryRouter>,
		);
		const textarea = screen.getByTestId("ids-textarea") as HTMLTextAreaElement;
		fireEvent.change(textarea, { target: { value: "aaaaaaaa" } });
		fireEvent.click(screen.getByTestId("submit-button"));

		await waitFor(() => {
			expect(lookupDurationsMock).toHaveBeenCalledTimes(1);
		});

		const csvBtn = await screen.findByTestId("export-csv-button");
		fireEvent.click(csvBtn);

		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("asset-durations-1.csv");
	});
});
