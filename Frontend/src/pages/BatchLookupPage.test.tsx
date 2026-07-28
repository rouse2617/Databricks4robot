// CYB-4333: hub tests for the collapsed 批量查询 tab.
//
// Covers preset routing (default, explicit `?preset=`, unknown fallback),
// URL sync on Segmented click, extra-param preservation, and the two shell
// integration paths (submit → stats + table + CSV; costs preset renders too)
// that used to live in the deleted `AssetDurationLookup.test.tsx`.

import {
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import BatchLookupPage from "./BatchLookupPage";

const { lookupDurationsMock, lookupCostsMock } = vi.hoisted(() => ({
	lookupDurationsMock: vi.fn(),
	lookupCostsMock: vi.fn(),
}));

vi.mock("../api/assets", () => ({
	assetsApi: {
		lookupDurations: lookupDurationsMock,
		lookupCosts: lookupCostsMock,
	},
}));

// AntD Segmented + Table rely on matchMedia; jsdom does not implement it.
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

function LocationEcho() {
	const location = useLocation();
	return <output data-testid="location-search">{location.search}</output>;
}

function renderAt(entry: string): ReactNode {
	return render(
		<MemoryRouter initialEntries={[entry]}>
			<Routes>
				<Route
					path="/assets"
					element={
						<>
							<LocationEcho />
							<BatchLookupPage />
						</>
					}
				/>
			</Routes>
		</MemoryRouter>,
	);
}

beforeEach(() => {
	lookupDurationsMock.mockReset();
	lookupCostsMock.mockReset();
	vi.restoreAllMocks();
});

describe("BatchLookupPage preset routing", () => {
	it("renders the durations preset when `preset` is unset", () => {
		renderAt("/assets?view=lookup");
		// Durations preset ships filter inputs with these testids.
		expect(screen.getByTestId("min-duration-input")).toBeTruthy();
		expect(screen.getByTestId("max-duration-input")).toBeTruthy();
	});

	it("renders the costs preset when `preset=costs`", () => {
		renderAt("/assets?view=lookup&preset=costs");
		// Costs preset ships a date-range + group-by segmented control.
		// (RangePicker internally forwards data-testid to both inputs, so we
		// query the group-by Segmented — a single element — for the assertion.)
		expect(screen.getByTestId("cost-group-by")).toBeTruthy();
	});

	it("falls back to durations when `preset` is unknown", () => {
		renderAt("/assets?view=lookup&preset=bogus");
		expect(screen.getByTestId("min-duration-input")).toBeTruthy();
	});

	it("updates the URL when the Segmented picker changes", async () => {
		renderAt("/assets?view=lookup");
		// Click the 成本 option inside the picker.
		const picker = screen.getByTestId("batch-lookup-preset-picker");
		const costsOption = within(picker).getByText("成本");
		fireEvent.click(costsOption);

		await waitFor(() => {
			expect(screen.getByTestId("location-search").textContent).toContain(
				"preset=costs",
			);
		});
		// Costs preset filters now render (group-by Segmented is single).
		expect(screen.getByTestId("cost-group-by")).toBeTruthy();
	});

	it("preserves other query params through Segmented clicks", async () => {
		renderAt("/assets?view=lookup&other=keep-me");
		const picker = screen.getByTestId("batch-lookup-preset-picker");
		fireEvent.click(within(picker).getByText("成本"));
		await waitFor(() => {
			const search = screen.getByTestId("location-search").textContent ?? "";
			expect(search).toContain("preset=costs");
			expect(search).toContain("other=keep-me");
			expect(search).toContain("view=lookup");
		});
	});
});

describe("BatchLookupPage durations shell integration", () => {
	it("submits parsed ids and renders stats + results table", async () => {
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

		renderAt("/assets?view=lookup");
		const textarea = screen.getByTestId("ids-textarea") as HTMLTextAreaElement;
		fireEvent.change(textarea, {
			target: {
				value: "aaaaaaaa\n019f8319-0ef3-7da0-815c-30f23d21e7f1\nzzzzzzzz",
			},
		});
		fireEvent.click(screen.getByTestId("submit-button"));

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

		const table = await screen.findByTestId("results-table");
		const rows = within(table).getAllByRole("row");
		// Row 0 is the header row.
		expect(rows.length).toBe(1 + 2);
		expect(screen.getByText("matched")).toBeTruthy();
	});

	it("CSV export button triggers a download with the durations filename", async () => {
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

		renderAt("/assets?view=lookup");
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
