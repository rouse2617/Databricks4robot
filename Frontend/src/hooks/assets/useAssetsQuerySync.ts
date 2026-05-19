// ─── Assets Discovery Workbench — Bidirectional URL Sync Hook ───
// Keeps queryState and the browser URL in sync.
// On mount: parse URL → hydrate state.
// After hydration: serialize state changes → URL.
// On popstate: re-parse URL → hydrate state.
// Validates: Requirements R8

import { useEffect, useRef } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
	QueryState,
	RouterState,
} from "../../lib/assets/assetsDiscoveryTypes";
import {
	parsePreviewAssetIdFromUrl,
	parsePreviewSourceIdFromUrl,
	parsePreviewTopicFromUrl,
	parseQueryStateFromUrl,
	serializeQueryStateToUrl,
} from "../../lib/assets/assetsDiscoveryUrl";
import { parsePreviewURLState } from "../../lib/assets/previewURLState";

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
	previewSourceId: string | null,
	previewTopic: string | null,
	dispatch: React.Dispatch<AssetsDiscoveryAction>,
	previewTimeSec: number | null = null,
	ds: string | null = null,
	dsParams: Record<string, string> | null = null,
): void {
	const navigate = useNavigate();
	const location = useLocation();

	// Track last URL we wrote (avoid re-parsing our own changes)
	const lastWrittenUrlRef = useRef<string | null>(null);

	// Track previous page number to detect page vs filter changes
	const prevPageRef = useRef(queryState.page);

	// Track whether the initial hydration effect has run
	const didHydrateRef = useRef(false);

	// ── On mount: parse URL and hydrate state ──
	useEffect(() => {
		if (didHydrateRef.current) return;
		didHydrateRef.current = true;

		const sp = new URLSearchParams(location.search);
		const parsed = parseQueryStateFromUrl(sp);
		const previewId = parsePreviewAssetIdFromUrl(sp);
		const sourceId = parsePreviewSourceIdFromUrl(sp);
		const topic = parsePreviewTopicFromUrl(sp);
		const previewState = parsePreviewURLState(sp);

		dispatch({
			type: "URL_HYDRATE",
			payload: {
				queryState: parsed,
				previewAssetId: previewId,
				previewSourceId: sourceId,
				previewTopic: topic,
				previewTimeSec: previewState.time ?? null,
				ds: previewState.ds ?? null,
				dsParams: previewState.dsParams ?? null,
			},
		});
		dispatch({ type: "MARK_URL_HYDRATED" });
	}, [dispatch, location.search]);

	// ── After hydration: serialize queryState changes to URL ──
	useEffect(() => {
		if (!routerState.urlHydrated) return;

		const sp = serializeQueryStateToUrl(
			queryState,
			previewAssetId,
			previewSourceId,
			previewTopic,
			previewTimeSec,
			{ ds, dsParams },
		);
		const search = sp.toString();
		const nextSearch = search ? `?${search}` : "";
		const nextUrl = `${location.pathname}${nextSearch}`;
		const currentUrl = `${location.pathname}${location.search}`;

		// Always record the intended URL so the popstate effect does not re-hydrate.
		lastWrittenUrlRef.current = nextUrl;

		// Don't navigate if URL is already correct
		if (nextUrl === currentUrl) {
			prevPageRef.current = queryState.page;
			return;
		}

		// Use pushState for page changes (so back button works for pagination)
		// Use replace for filter/sort/other changes
		const isPageChange = queryState.page !== prevPageRef.current;
		navigate(
			{ pathname: location.pathname, search: nextSearch },
			{ replace: !isPageChange },
		);

		prevPageRef.current = queryState.page;
	}, [
		location.pathname,
		location.search,
		navigate,
		previewAssetId,
		previewSourceId,
		previewTopic,
		previewTimeSec,
		ds,
		dsParams,
		queryState,
		routerState.urlHydrated,
	]);

	// ── React Router location changes (back/forward + address-bar edits) ──
	useEffect(() => {
		if (!routerState.urlHydrated) return;

		const currentUrl = `${location.pathname}${location.search}`;
		if (lastWrittenUrlRef.current === currentUrl) {
			lastWrittenUrlRef.current = null;
			return;
		}

		const sp = new URLSearchParams(location.search);
		const parsed = parseQueryStateFromUrl(sp);
		const previewId = parsePreviewAssetIdFromUrl(sp);
		const sourceId = parsePreviewSourceIdFromUrl(sp);
		const topic = parsePreviewTopicFromUrl(sp);
		const previewState = parsePreviewURLState(sp);
		dispatch({
			type: "URL_HYDRATE",
			payload: {
				queryState: parsed,
				previewAssetId: previewId,
				previewSourceId: sourceId,
				previewTopic: topic,
				previewTimeSec: previewState.time ?? null,
				ds: previewState.ds ?? null,
				dsParams: previewState.dsParams ?? null,
			},
		});
	}, [dispatch, location.pathname, location.search, routerState.urlHydrated]);
}
