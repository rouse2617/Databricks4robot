import { describe, expect, it, vi } from "vitest";

const getMock = vi.fn();

vi.mock("./client", () => ({
	apiClient: {
		get: getMock,
	},
}));

describe("algoRunsApi.list", async () => {
	const { algoRunsApi } = await import("./algoRuns");

	it("sends page/page_size pagination params", async () => {
		getMock.mockResolvedValueOnce({
			data: { items: [], total: 0, page: 2, page_size: 25 },
		});

		await algoRunsApi.list({
			page: 2,
			page_size: 25,
			algo_name: "hand_track",
			status: "failed",
			started_after: "2026-05-22T00:00:00Z",
			started_before: "2026-05-23T00:00:00Z",
		});

		expect(getMock).toHaveBeenCalledWith(
			"/algo-runs?page=2&page_size=25&algo_name=hand_track&status=failed&started_after=2026-05-22T00%3A00%3A00Z&started_before=2026-05-23T00%3A00%3A00Z",
		);
	});
});
