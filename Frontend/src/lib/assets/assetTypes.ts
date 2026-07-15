// Single source of truth for the built-in asset_type option set.
//
// CYB-3305: sidebar facet was missing raw_mcap / action / frame; AddFilterPopover
// held a duplicate copy of the list. Both now import from here.
//
// mergeAssetTypeOptions merges the built-in set with any keys present in the
// ES aggregation buckets, so the facet UI shows every asset_type that actually
// exists in the data (not just the ones we hard-coded).

export const ASSET_TYPE_OPTIONS = [
	"segment",
	"clip",
	"frame_set",
	"derived_asset",
	"raw_mcap",
	"action",
	"frame",
];

export function mergeAssetTypeOptions(
	counts?: Record<string, number>,
): string[] {
	if (!counts) return ASSET_TYPE_OPTIONS;
	const seen = new Set<string>(ASSET_TYPE_OPTIONS);
	const merged = [...ASSET_TYPE_OPTIONS];
	for (const key of Object.keys(counts)) {
		if (!key || seen.has(key)) continue;
		seen.add(key);
		merged.push(key);
	}
	return merged;
}
