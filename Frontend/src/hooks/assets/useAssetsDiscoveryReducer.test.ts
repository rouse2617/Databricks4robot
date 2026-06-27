// @vitest-environment jsdom

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryApi } from "../../api/query";
import type { Asset } from "../../api/types";
import { useAssetsDiscoveryReducer } from "./useAssetsDiscoveryReducer";

vi.mock("../../api/query", () => ({
	queryApi: {
		run: vi.fn(),
	},
}));

describe("useAssetsDiscoveryReducer", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(queryApi.run).mockResolvedValue({
			items: [],
			total: 0,
			page: 1,
			page_size: 20,
			columns: [],
			debug_plan: { steps: [{ engine: "postgres", mode: "filter" }] },
		});
	});

	it("uses run for structured mode (list then facets)", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(2);
		});
		const firstCall = vi.mocked(queryApi.run).mock.calls[0]?.[0];
		const secondCall = vi.mocked(queryApi.run).mock.calls[1]?.[0];
		expect(firstCall?.facets).toBeUndefined();
		expect(secondCall?.facets?.length).toBeGreaterThan(0);
	});

	it("derives total from the count query when the list total is unreliable (ES down)", async () => {
		// Regression: the list query can return a `total` smaller than the rows
		// it actually returns when ES is unavailable. The dedicated count/facets
		// query carries the authoritative total (postgres count fallback) even
		// when no aggregation buckets come back (facets omitted).
		const listItems = Array.from(
			{ length: 6 },
			(_, i) => ({ asset_id: `a${i}` }) as unknown as Asset,
		);
		vi.mocked(queryApi.run)
			.mockResolvedValueOnce({
				items: listItems,
				total: 2,
				page: 1,
				page_size: 20,
				columns: [],
				debug_plan: { steps: [{ engine: "postgres", mode: "filter" }] },
			})
			.mockResolvedValueOnce({
				items: [listItems[0]],
				total: 6,
				page: 1,
				page_size: 1,
				columns: [],
				warnings: ["elasticsearch unavailable; used postgres count"],
			});

		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(2);
		});
		await waitFor(() => {
			expect(result.current[0].resultsState.total).toBe(6);
		});
		expect(result.current[0].resultsState.items).toHaveLength(6);
		expect(result.current[0].resultsState.totalApprox).toBe(false);
	});

	it("never reports a total smaller than the rows on the current page", async () => {
		// Even before the count query resolves, the list result must not show
		// `total < rows` (the original visible bug: "共 2 条" with 6 rows).
		const listItems = Array.from(
			{ length: 6 },
			(_, i) => ({ asset_id: `b${i}` }) as unknown as Asset,
		);
		vi.mocked(queryApi.run).mockResolvedValue({
			items: listItems,
			total: 2,
			page: 1,
			page_size: 20,
			columns: [],
			debug_plan: { steps: [{ engine: "postgres", mode: "filter" }] },
		});

		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(result.current[0].resultsState.items).toHaveLength(6);
		});
		expect(result.current[0].resultsState.total).toBeGreaterThanOrEqual(6);
	});

	it("does not refetch facets when only page changes", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});
		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(2);
		});
		vi.mocked(queryApi.run).mockClear();

		act(() => {
			result.current[1]({ type: "SET_PAGE", payload: { page: 2 } });
		});
		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(1);
		});
		expect(vi.mocked(queryApi.run).mock.calls[0]?.[0]?.facets).toBeUndefined();
	});

	it("uses query api for keyword mode", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({
				type: "SET_SEARCH_MODE",
				payload: { mode: "keyword" },
			});
			result.current[1]({
				type: "COMMIT_QUERY_TEXT",
				payload: {
					text: "forklift",
					tokens: [
						{
							id: "search_fulltext_1",
							field: "_fulltext",
							op: "ilike",
							value: "forklift",
							source: "search",
						},
					],
				},
			});
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(1);
		});
	});

	it("uses query api for semantic mode", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({
				type: "SET_SEARCH_MODE",
				payload: { mode: "semantic" },
			});
			result.current[1]({
				type: "COMMIT_QUERY_TEXT",
				payload: { text: "forklift", tokens: [] },
			});
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(1);
		});
	});

	it("uses query api for similar mode", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({
				type: "SET_SEARCH_MODE",
				payload: { mode: "similar" },
			});
			result.current[1]({
				type: "COMMIT_QUERY_TEXT",
				payload: { text: "aset0001", tokens: [] },
			});
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(1);
		});
	});
});
