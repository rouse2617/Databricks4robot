// CYB-4303: DurationDistributionCard smoke tests.
//
// Cover: initial fetch on mount, Segmented change triggers a fresh fetch with
// the new asset_type param, empty-state (total_assets=0) renders <Empty>,
// error path renders the error message.

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DurationDistributionResponse } from "../../api/dashboard";
import DurationDistributionCard from "./DurationDistributionCard";

const { durationDistributionMock } = vi.hoisted(() => ({
	durationDistributionMock: vi.fn(),
}));

vi.mock("../../api/dashboard", () => ({
	dashboardApi: {
		durationDistribution: durationDistributionMock,
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

function makeResponse(
	overrides: Partial<DurationDistributionResponse> = {},
): DurationDistributionResponse {
	return {
		asset_type: null,
		buckets: [
			{ label: "<1min", lo_ms: 0, hi_ms: 60_000, count: 12, total_ms: 250_000 },
			{
				label: "1-10min",
				lo_ms: 60_000,
				hi_ms: 600_000,
				count: 34,
				total_ms: 5_100_000,
			},
			{
				label: "10-30min",
				lo_ms: 600_000,
				hi_ms: 1_800_000,
				count: 56,
				total_ms: 44_000_000,
			},
			{
				label: "30-60min",
				lo_ms: 1_800_000,
				hi_ms: 3_600_000,
				count: 21,
				total_ms: 55_000_000,
			},
			{
				label: "60min+",
				lo_ms: 3_600_000,
				hi_ms: null,
				count: 3,
				total_ms: 15_000_000,
			},
		],
		total_assets: 126,
		total_ms: 119_350_000,
		mean_ms: 947_222,
		min_ms: 500,
		max_ms: 6_500_000,
		p50_ms: 850_000,
		p90_ms: 2_900_000,
		...overrides,
	};
}

describe("DurationDistributionCard", () => {
	beforeEach(() => {
		durationDistributionMock.mockReset();
	});

	it("fetches on mount and renders the histogram + stats", async () => {
		durationDistributionMock.mockResolvedValueOnce(makeResponse());

		render(<DurationDistributionCard />);

		await waitFor(() =>
			expect(durationDistributionMock).toHaveBeenCalledTimes(1),
		);
		// First call has no asset_type (segmented defaults to 全部/"").
		expect(durationDistributionMock).toHaveBeenLastCalledWith(undefined);

		// Bucket labels rendered.
		for (const label of ["<1min", "1-10min", "10-30min", "30-60min", "60min+"]) {
			expect(await screen.findByText(label)).toBeTruthy();
		}
		// Bucket count of the third bucket shows up.
		expect(await screen.findByText("56")).toBeTruthy();
		// Overall stats — 总资产 label + a value.
		expect(await screen.findByText("总资产")).toBeTruthy();
		expect(await screen.findByText("126")).toBeTruthy();
	});

	it("Segmented change triggers a fresh fetch with the new asset_type", async () => {
		durationDistributionMock.mockResolvedValue(makeResponse());

		render(<DurationDistributionCard />);
		await waitFor(() =>
			expect(durationDistributionMock).toHaveBeenCalledTimes(1),
		);

		// Segmented options render as accessible buttons whose text is the label.
		const rawMcapOption = await screen.findByText("raw_mcap");
		fireEvent.click(rawMcapOption);

		await waitFor(() =>
			expect(durationDistributionMock).toHaveBeenCalledTimes(2),
		);
		expect(durationDistributionMock).toHaveBeenLastCalledWith("raw_mcap");
	});

	it("shows the empty state when total_assets = 0", async () => {
		durationDistributionMock.mockResolvedValueOnce(
			makeResponse({
				total_assets: 0,
				total_ms: 0,
				buckets: [
					{ label: "<1min", lo_ms: 0, hi_ms: 60_000, count: 0, total_ms: 0 },
					{
						label: "1-10min",
						lo_ms: 60_000,
						hi_ms: 600_000,
						count: 0,
						total_ms: 0,
					},
					{
						label: "10-30min",
						lo_ms: 600_000,
						hi_ms: 1_800_000,
						count: 0,
						total_ms: 0,
					},
					{
						label: "30-60min",
						lo_ms: 1_800_000,
						hi_ms: 3_600_000,
						count: 0,
						total_ms: 0,
					},
					{
						label: "60min+",
						lo_ms: 3_600_000,
						hi_ms: null,
						count: 0,
						total_ms: 0,
					},
				],
			}),
		);

		render(<DurationDistributionCard />);
		expect(await screen.findByText("所选类型下无资产")).toBeTruthy();
	});

	it("shows the error message on fetch failure", async () => {
		durationDistributionMock.mockRejectedValueOnce(new Error("boom"));

		render(<DurationDistributionCard />);
		// The extractApiErrorMessage helper returns the message when present,
		// so "boom" should appear in the Empty description.
		expect(await screen.findByText(/boom|数据时长分布加载失败/)).toBeTruthy();
	});
});
