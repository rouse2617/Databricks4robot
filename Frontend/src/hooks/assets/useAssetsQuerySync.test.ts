// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import type { To } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
	QueryState,
	RouterState,
} from "../../lib/assets/assetsDiscoveryTypes";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { useAssetsQuerySync } from "./useAssetsQuerySync";

const mockNavigate = vi.fn(
	(
		to: To | { pathname?: string; search?: string },
		options?: { replace?: boolean },
	) => {
		const pathname =
			typeof to === "string"
				? (to.split("?")[0] ?? "/assets")
				: (to.pathname ?? window.location.pathname);
		const search =
			typeof to === "string"
				? to.includes("?")
					? `?${to.split("?")[1]}`
					: ""
				: (to.search ?? "");
		const url = `${pathname}${search}`;
		if (options?.replace) {
			window.history.replaceState(null, "", url);
		} else {
			window.history.pushState(null, "", url);
		}
	},
);

vi.mock("react-router-dom", async () => {
	const actual =
		await vi.importActual<typeof import("react-router-dom")>(
			"react-router-dom",
		);
	return {
		...actual,
		useNavigate: () => mockNavigate,
		useLocation: () => ({
			pathname: window.location.pathname,
			search: window.location.search,
		}),
	};
});

function defaultQueryState(): QueryState {
	return { ...defaultAssetsDiscoveryState.queryState };
}

function hydratedRouter(): RouterState {
	return { urlHydrated: true, currentPath: "/assets" };
}

function unhydratedRouter(): RouterState {
	return { urlHydrated: false, currentPath: "/assets" };
}

function setUrl(url: string) {
	window.history.pushState(null, "", url);
}

describe("useAssetsQuerySync", () => {
	let pushStateSpy: ReturnType<typeof vi.spyOn>;
	let replaceStateSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		mockNavigate.mockClear();
		setUrl("/assets");
		pushStateSpy = vi.spyOn(window.history, "pushState");
		replaceStateSpy = vi.spyOn(window.history, "replaceState");
	});

	afterEach(() => {
		pushStateSpy.mockRestore();
		replaceStateSpy.mockRestore();
		vi.restoreAllMocks();
		window.history.pushState(null, "", "/assets");
	});

	it("dispatches URL_HYDRATE and MARK_URL_HYDRATED on mount", () => {
		pushStateSpy.mockRestore();
		setUrl("/assets?page=3&sort=created_at");
		pushStateSpy = vi.spyOn(window.history, "pushState");

		const dispatch = vi.fn();
		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: expect.objectContaining({ page: 3, sort: "created_at" }),
				previewAssetId: null,
				previewSourceId: null,
				previewTopic: null,
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
		expect(dispatch).toHaveBeenCalledWith({ type: "MARK_URL_HYDRATED" });
	});

	it("dispatches URL_HYDRATE with empty object for clean URL", () => {
		const dispatch = vi.fn();
		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: {},
				previewAssetId: null,
				previewSourceId: null,
				previewTopic: null,
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
	});

	it("does not push/replace URL before hydration", () => {
		const dispatch = vi.fn();
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		renderHook(() =>
			useAssetsQuerySync(
				{ ...defaultQueryState(), page: 5 },
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(pushStateSpy).not.toHaveBeenCalled();
		expect(replaceStateSpy).not.toHaveBeenCalled();
	});

	it("uses replaceState for filter changes after hydration", () => {
		const dispatch = vi.fn();
		const qs = { ...defaultQueryState(), sort: "created_at" };
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		renderHook(() =>
			useAssetsQuerySync(qs, hydratedRouter(), null, null, null, dispatch),
		);

		expect(replaceStateSpy).toHaveBeenCalledWith(
			null,
			"",
			expect.stringContaining("sort=created_at"),
		);
	});

	it("uses pushState for page changes after hydration", () => {
		const dispatch = vi.fn();
		const { rerender } = renderHook(
			({ qs, router, preview }) =>
				useAssetsQuerySync(qs, router, preview ?? null, null, null, dispatch),
			{
				initialProps: {
					qs: { ...defaultQueryState(), page: 1 },
					router: hydratedRouter(),
					preview: null as string | null,
				},
			},
		);

		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		rerender({
			qs: { ...defaultQueryState(), page: 2 },
			router: hydratedRouter(),
			preview: null,
		});

		expect(pushStateSpy).toHaveBeenCalledWith(
			null,
			"",
			expect.stringContaining("page=2"),
		);
	});

	it("dispatches URL_HYDRATE with preview id from query string", () => {
		pushStateSpy.mockRestore();
		setUrl("/assets?preview=abc-def-0000-1111-222233334444");
		pushStateSpy = vi.spyOn(window.history, "pushState");
		const dispatch = vi.fn();

		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: {},
				previewAssetId: "abc-def-0000-1111-222233334444",
				previewSourceId: null,
				previewTopic: null,
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
	});

	it("dispatches URL_HYDRATE with preview topic from query string", () => {
		pushStateSpy.mockRestore();
		setUrl("/assets?preview=asset-1&preview_topic=%2Fcamera%2Ffront");
		pushStateSpy = vi.spyOn(window.history, "pushState");
		const dispatch = vi.fn();

		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: {},
				previewAssetId: "asset-1",
				previewSourceId: null,
				previewTopic: "/camera/front",
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
	});

	it("dispatches URL_HYDRATE with preview source from query string", () => {
		pushStateSpy.mockRestore();
		setUrl("/assets?preview=asset-1&preview_source=live_topic_0");
		pushStateSpy = vi.spyOn(window.history, "pushState");
		const dispatch = vi.fn();

		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: {},
				previewAssetId: "asset-1",
				previewSourceId: "live_topic_0",
				previewTopic: null,
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
	});

	it("hydrates preview time, ds and dsParams from URL", () => {
		pushStateSpy.mockRestore();
		setUrl(
			"/assets?time=2025-01-02T03:04:05Z&ds=remote-file&ds.asset_id=asset-1&ds.url=http%3A%2F%2Fa&ds.url=http%3A%2F%2Fb",
		);
		pushStateSpy = vi.spyOn(window.history, "pushState");
		const dispatch = vi.fn();

		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: {},
				previewAssetId: "asset-1",
				previewSourceId: null,
				previewTopic: null,
				previewTimeSec: 1735787045,
				ds: "remote-file",
				dsParams: { asset_id: "asset-1", url: "http://a,http://b" },
			}),
		});
	});

	it("uses replaceState when preview id changes after hydration", () => {
		const dispatch = vi.fn();
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		const { rerender } = renderHook(
			({ preview }) =>
				useAssetsQuerySync(
					defaultQueryState(),
					hydratedRouter(),
					preview,
					null,
					null,
					dispatch,
				),
			{
				initialProps: { preview: null as string | null },
			},
		);

		rerender({ preview: "id-1" });

		expect(replaceStateSpy).toHaveBeenCalledWith(
			null,
			"",
			expect.stringContaining("preview=id-1"),
		);
	});

	it("does not update URL when serialized URL matches current", () => {
		const dispatch = vi.fn();
		setUrl("/assets?preview_layout_version=2");
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				hydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		expect(pushStateSpy).not.toHaveBeenCalled();
		expect(replaceStateSpy).not.toHaveBeenCalled();
	});

	it("re-hydrates on popstate event", () => {
		const dispatch = vi.fn();
		const { rerender } = renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				hydratedRouter(),
				null,
				null,
				null,
				dispatch,
			),
		);

		dispatch.mockClear();
		setUrl("/assets?page=5");
		act(() => rerender());

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: expect.objectContaining({
				queryState: expect.objectContaining({ page: 5 }),
				previewAssetId: null,
				previewSourceId: null,
				previewTopic: null,
				previewTimeSec: null,
				ds: null,
				dsParams: null,
			}),
		});
	});

	it("does not update URL on rerender when location matches last write", () => {
		const dispatch = vi.fn();
		const { rerender } = renderHook(
			({ qs }) =>
				useAssetsQuerySync(qs, hydratedRouter(), null, null, null, dispatch),
			{ initialProps: { qs: { ...defaultQueryState(), sort: "created_at" } } },
		);

		expect(replaceStateSpy).toHaveBeenCalled();
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		act(() => rerender({ qs: { ...defaultQueryState(), sort: "created_at" } }));
		expect(pushStateSpy).not.toHaveBeenCalled();
		expect(replaceStateSpy).not.toHaveBeenCalled();
	});

	it("updates URL when preview time or ds params change", () => {
		const dispatch = vi.fn();
		const { rerender } = renderHook(
			({ previewTimeSec, dsParams }) =>
				useAssetsQuerySync(
					defaultQueryState(),
					hydratedRouter(),
					"asset-1",
					null,
					null,
					dispatch,
					previewTimeSec,
					"remote-file",
					dsParams,
				),
			{
				initialProps: {
					previewTimeSec: null as number | null,
					dsParams: { asset_id: "asset-1" } as Record<string, string> | null,
				},
			},
		);

		replaceStateSpy.mockClear();
		rerender({
			previewTimeSec: 12.5,
			dsParams: { zeta: "2", asset_id: "asset-1" },
		});

		expect(replaceStateSpy).toHaveBeenCalledWith(
			null,
			"",
			expect.stringContaining("preview_time=12.5"),
		);
		expect(replaceStateSpy).toHaveBeenCalledWith(
			null,
			"",
			expect.stringContaining("ds.asset_id=asset-1"),
		);
	});
});
