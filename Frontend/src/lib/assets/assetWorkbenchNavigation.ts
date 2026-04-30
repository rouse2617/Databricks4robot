import type { NavigateFunction } from "react-router-dom";

export type AssetDetailLocationState = {
  assetsReturnTo?: string;
};

/** sessionStorage key — bump suffix if stored shape changes */
export const ASSET_DETAIL_RETURN_SESSION_KEY = "d4c_asset_detail_return_v1";

/** App top-level route segments (same-origin return targets). */
const TOP_LEVEL_ROUTES = new Set([
  "dashboard",
  "assets",
  "mcap-files",
  "algo",
  "deliveries",
  "analytics",
  "events",
  "tags",
  "settings",
  "login",
]);

/** Current URL is asset detail `/assets/:id`. */
export function isAssetDetailPath(pathAndSearch: string): boolean {
  const path = pathAndSearch.split("?")[0] ?? "";
  const segments = path.split("/").filter(Boolean);
  return segments[0] === "assets" && segments.length >= 2;
}

/** `/assets` or `/assets?...` (discovery workbench only). */
export function isAssetsWorkbenchReturnUrl(pathAndSearch: string): boolean {
  const path = pathAndSearch.split("?")[0] ?? "";
  const segments = path.split("/").filter(Boolean);
  return segments.length === 1 && segments[0] === "assets";
}

/**
 * Internal app URLs safe to return to after leaving asset detail.
 * Rejects open redirects, `/assets/:id`, and unknown top-level paths.
 */
export function isSafeInternalReturnUrl(pathAndSearch: string): boolean {
  if (!pathAndSearch.startsWith("/") || pathAndSearch.startsWith("//")) {
    return false;
  }
  const path = pathAndSearch.split("?")[0] ?? "";
  if (path.includes("..")) return false;
  const segments = path.split("/").filter(Boolean);
  if (segments.length === 0) return false;
  if (segments[0] === "assets" && segments.length >= 2) return false;
  return TOP_LEVEL_ROUTES.has(segments[0]!);
}

/** Clear session slot (e.g. after explicit state-based return). */
export function clearStoredAssetDetailReturn(): void {
  if (typeof window === "undefined") return;
  try {
    sessionStorage.removeItem(ASSET_DETAIL_RETURN_SESSION_KEY);
  } catch {
    /* ignore */
  }
}

/**
 * Remember current location before navigating to asset detail.
 * Skips when already on a detail page so jumping detail→detail keeps the original return.
 */
export function rememberReturnUrlBeforeAssetDetail(): void {
  if (typeof window === "undefined") return;
  const pathAndSearch = `${window.location.pathname}${window.location.search}`;
  if (isAssetDetailPath(pathAndSearch)) return;
  if (!isSafeInternalReturnUrl(pathAndSearch)) return;
  try {
    sessionStorage.setItem(ASSET_DETAIL_RETURN_SESSION_KEY, pathAndSearch);
  } catch {
    /* quota / private mode */
  }
}

/** Read and remove stored return URL if valid. */
export function consumeStoredReturnUrl(): string | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = sessionStorage.getItem(ASSET_DETAIL_RETURN_SESSION_KEY);
    if (!raw) return null;
    sessionStorage.removeItem(ASSET_DETAIL_RETURN_SESSION_KEY);
    return isSafeInternalReturnUrl(raw) ? raw : null;
  } catch {
    return null;
  }
}

/**
 * Navigate to full-screen asset detail after persisting the current page for "back".
 * Optional `state` (e.g. explicit `assetsReturnTo`) takes precedence over session when returning.
 */
export function navigateToAssetDetail(
  navigate: NavigateFunction,
  assetId: string,
  options?: { state?: AssetDetailLocationState },
): void {
  rememberReturnUrlBeforeAssetDetail();
  navigate(`/assets/${assetId}`, options);
}
