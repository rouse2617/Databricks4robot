// @vitest-environment jsdom

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
	QueryState,
	RouterState,
} from "../../lib/assets/assetsDiscoveryTypes";
import { defaultAssetsDiscoveryState } from "../../lib/assets/assetsDiscoveryTypes";
import { useAssetsQuerySync } from "./useAssetsQuerySync";

// ─── Helpers ───

function defaultQueryState(): QueryState {
	return { ...defaultAssetsDiscoveryState.queryState };
}

function hydratedRouter(): RouterState {
	return { urlHydrated: true, currentPath: "/assets" };
}

function unhydratedRouter(): RouterState {
	return { urlHydrated: false, currentPath: "/assets" };
}

/** Navigate jsdom to a URL (actually updates window.location). */
function setUrl(url: string) {
	window.history.pushState(null, "", url);
}

// ─── Tests ───

describe("useAssetsQuerySync", () => {
	let pushStateSpy: ReturnType<typeof vi.spyOn>;
	let replaceStateSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		// Reset URL to clean state (real call, not mocked)
		setUrl("/assets");
		// Now install spies that track calls but still execute
		pushStateSpy = vi.spyOn(window.history, "pushState");
		replaceStateSpy = vi.spyOn(window.history, "replaceState");
	});

	afterEach(() => {
		pushStateSpy.mockRestore();
		replaceStateSpy.mockRestore();
		vi.restoreAllMocks();
		// Clean up URL
		window.history.pushState(null, "", "/assets");
	});

	it("dispatches URL_HYDRATE and MARK_URL_HYDRATED on mount", () => {
		// Set URL with params before mounting (before spies, so spy is clean)
		pushStateSpy.mockRestore();
		setUrl("/assets?page=3&sort=created_at");
		pushStateSpy = vi.spyOn(window.history, "pushState");

		const dispatch = vi.fn();
		renderHook(() =>
			useAssetsQuerySync(
				defaultQueryState(),
				unhydratedRouter(),
				null,
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: {
				queryState: expect.objectContaining({ page: 3, sort: "created_at" }),
				previewAssetId: null,
			},
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
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: { queryState: {}, previewAssetId: null },
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
				dispatch,
			),
		);

		// No URL sync calls should happen when not hydrated
		expect(pushStateSpy).not.toHaveBeenCalled();
		expect(replaceStateSpy).not.toHaveBeenCalled();
	});

	it("uses replaceState for filter changes after hydration", () => {
		const dispatch = vi.fn();
		const qs = { ...defaultQueryState(), sort: "created_at" };

		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		renderHook(() => useAssetsQuerySync(qs, hydratedRouter(), null, dispatch));

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
				useAssetsQuerySync(qs, router, preview ?? null, dispatch),
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

		// Change page from 1 to 2
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
				dispatch,
			),
		);

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: {
				queryState: {},
				previewAssetId: "abc-def-0000-1111-222233334444",
			},
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
		// Default state serializes to empty params, and location is /assets
		const dispatch = vi.fn();
		pushStateSpy.mockClear();
		replaceStateSpy.mockClear();

		renderHook(() =>
			useAssetsQuerySync(defaultQueryState(), hydratedRouter(), null, dispatch),
		);

		expect(pushStateSpy).not.toHaveBeenCalled();
		expect(replaceStateSpy).not.toHaveBeenCalled();
	});

	it("re-hydrates on popstate event", () => {
		const dispatch = vi.fn();
		renderHook(() =>
			useAssetsQuerySync(defaultQueryState(), hydratedRouter(), null, dispatch),
		);

		// Clear mount dispatches
		dispatch.mockClear();

		// Simulate browser back: update URL then fire popstate
		setUrl("/assets?page=5");

		act(() => {
			window.dispatchEvent(new PopStateEvent("popstate"));
		});

		expect(dispatch).toHaveBeenCalledWith({
			type: "URL_HYDRATE",
			payload: {
				queryState: expect.objectContaining({ page: 5 }),
				previewAssetId: null,
			},
		});
	});

	it("cleans up popstate listener on unmount", () => {
		const dispatch = vi.fn();
		const removeEventSpy = vi.spyOn(window, "removeEventListener");

		const { unmount } = renderHook(() =>
			useAssetsQuerySync(defaultQueryState(), hydratedRouter(), null, dispatch),
		);

		unmount();

		expect(removeEventSpy).toHaveBeenCalledWith(
			"popstate",
			expect.any(Function),
		);

		removeEventSpy.mockRestore();
	});
});
