// Centralized status color mapping for algorithm tasks and asset lifecycle.
//
// All pages must import from here instead of redefining local maps. This is
// the canonical source for DESIGN.md §2 业务状态色映射.
//
// The asset-lifecycle helper lives in `assetPresentation.ts` and is re-exported
// here so callers only need to remember one module path.

import type { Asset } from "../api/types";

export { getAssetStateColor as assetLifecycleTagColor } from "./assetPresentation";

export type AlgoStatus = "ok" | "failed" | "running" | "pending" | "blocked";

/** Ant Design Tag color name for an algorithm task status. */
export function algoStatusTagColor(status: string): string {
	switch (status) {
		case "ok":
			return "success";
		case "failed":
			return "error";
		case "running":
			return "warning";
		// biome-ignore lint/complexity/noUselessSwitchCase: explicit enumeration documents the AlgoStatus contract
		case "pending":
		// biome-ignore lint/complexity/noUselessSwitchCase: explicit enumeration documents the AlgoStatus contract
		case "blocked":
		default:
			return "default";
	}
}

export interface AlgoStatusEntry {
	key: string;
	status: string;
}

/** Extract `algo_key:status` pairs from `asset.algo_results` flat map. */
export function extractAlgoStatuses(
	algo: Asset["algo_results"] | undefined,
): AlgoStatusEntry[] {
	if (!algo) return [];
	const out: AlgoStatusEntry[] = [];
	for (const [k, v] of Object.entries(algo)) {
		if (k.endsWith(":status")) {
			out.push({ key: k.replace(":status", ""), status: String(v) });
		}
	}
	return out;
}
