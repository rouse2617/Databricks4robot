/**
 * True when s is a valid asset_id: 8 ASCII alphanumeric chars or UUID.
 */
export function isCanonicalAssetId(s: string): boolean {
	const t = s.trim();
	return (
		/^[0-9A-Za-z]{8}$/.test(t) ||
		/^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(
			t,
		)
	);
}
