// ─── Assets Discovery Workbench — Bidirectional URL Sync Hook ───
// Keeps queryState and the browser URL in sync.
// On mount: parse URL → hydrate state.
// After hydration: serialize state changes → URL.
// On popstate: re-parse URL → hydrate state.
// Validates: Requirements R8

import { useEffect, useRef } from "react";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
	QueryState,
	RouterState,
} from "../../lib/assets/assetsDiscoveryTypes";
import {
	parsePreviewAssetIdFromUrl,
	parseQueryStateFromUrl,
	serializeQueryStateToUrl,
} from "../../lib/assets/assetsDiscoveryUrl";

/**
 * Bidirectional sync between queryState and the browser URL.
 *
 * 1. On mount: parse `window.location.search` → dispatch URL_HYDRATE → MARK_URL_HYDRATED
 * 2. After hydration, on queryState change: serialize → replaceState (or pushState for page changes)
 * 3. On popstate: re-parse URL → dispatch URL_HYDRATE
 */
export function useAssetsQuerySync(
	queryState: QueryState,
	routerState: RouterState,
	previewAssetId: string | null,
	dispatch: React.Dispatch<AssetsDiscoveryAction>,
): void {
	// Track whether a URL change was initiated by us (to avoid re-parsing our own changes)
	const weChangedUrlRef = useRef(false);

	// Track previous page number to detect page vs filter changes
	const prevPageRef = useRef(queryState.page);

	// Track whether the initial hydration effect has run
	const didHydrateRef = useRef(false);

	// ── On mount: parse URL and hydrate state ──
	useEffect(() => {
		if (didHydrateRef.current) return;
		didHydrateRef.current = true;

		const sp = new URLSearchParams(window.location.search);
		const parsed = parseQueryStateFromUrl(sp);
		const previewId = parsePreviewAssetIdFromUrl(sp);

		dispatch({
			type: "URL_HYDRATE",
			payload: { queryState: parsed, previewAssetId: previewId },
		});
		dispatch({ type: "MARK_URL_HYDRATED" });
	}, [dispatch]);

	// ── After hydration: serialize queryState changes to URL ──
	useEffect(() => {
		if (!routerState.urlHydrated) return;

		const sp = serializeQueryStateToUrl(queryState, previewAssetId);
		const search = sp.toString();
		const newUrl = search
			? `${window.location.pathname}?${search}`
			: window.location.pathname;

		// Don't update if URL is already correct
		if (newUrl === `${window.location.pathname}${window.location.search}`) {
			prevPageRef.current = queryState.page;
			return;
		}

		weChangedUrlRef.current = true;

		// Use pushState for page changes (so back button works for pagination)
		// Use replaceState for filter/sort/other changes
		const isPageChange = queryState.page !== prevPageRef.current;
		if (isPageChange) {
			window.history.pushState(null, "", newUrl);
		} else {
			window.history.replaceState(null, "", newUrl);
		}

		prevPageRef.current = queryState.page;

		// Reset the flag after a microtask so popstate handler can check it
		queueMicrotask(() => {
			weChangedUrlRef.current = false;
		});
	}, [queryState, routerState.urlHydrated, previewAssetId]);

	// ── Listen for popstate (browser back/forward) ──
	useEffect(() => {
		const handlePopstate = () => {
			// If we initiated this URL change, skip re-parsing
			if (weChangedUrlRef.current) return;

			const sp = new URLSearchParams(window.location.search);
			const parsed = parseQueryStateFromUrl(sp);
			const previewId = parsePreviewAssetIdFromUrl(sp);
			dispatch({
				type: "URL_HYDRATE",
				payload: { queryState: parsed, previewAssetId: previewId },
			});
		};

		window.addEventListener("popstate", handlePopstate);
		return () => window.removeEventListener("popstate", handlePopstate);
	}, [dispatch]);
}
