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
