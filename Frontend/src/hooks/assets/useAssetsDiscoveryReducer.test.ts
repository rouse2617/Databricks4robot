// @vitest-environment jsdom

import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryApi } from "../../api/query";
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

	it("uses run for structured mode", async () => {
		const { result } = renderHook(() => useAssetsDiscoveryReducer());

		act(() => {
			result.current[1]({ type: "MARK_URL_HYDRATED" });
		});

		await waitFor(() => {
			expect(queryApi.run).toHaveBeenCalledTimes(1);
		});
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
