import {
	batchAssetLimitError,
	exceedsBatchAssetLimit,
} from "./batchAssetLimits";

/** Split comma / whitespace / newline separated asset IDs. */
export function parseAssetIdInput(raw: string): string[] {
	return raw
		.split(/[\s,，;；]+/)
		.map((part) => part.trim())
		.filter(Boolean);
}

export function mergeAssetIds(
	selectedIds: string[],
	incoming: string[],
): string[] {
	const seen = new Set<string>();
	const merged: string[] = [];
	for (const id of [...selectedIds, ...incoming]) {
		const trimmed = id.trim();
		if (!trimmed || seen.has(trimmed)) continue;
		seen.add(trimmed);
		merged.push(trimmed);
	}
	return merged;
}

/** Merge selected IDs with uncommitted bulk-paste text (e.g. on deploy). */
export function mergePendingBulkPaste(
	selectedIds: string[],
	bulkPasteRaw: string,
): { assetIds: string[]; error?: string } {
	const incoming = parseAssetIdInput(bulkPasteRaw);
	if (incoming.length === 0) {
		return { assetIds: selectedIds };
	}
	const assetIds = mergeAssetIds(selectedIds, incoming);
	if (exceedsBatchAssetLimit(assetIds.length)) {
		return {
			assetIds: selectedIds,
			error: batchAssetLimitError(assetIds.length),
		};
	}
	return { assetIds };
}
