/**
 * True when s is the current backend asset_id format: 8 ASCII alphanumeric chars.
 */
export function isCanonicalAssetId(s: string): boolean {
	const t = s.trim();
	return /^[0-9A-Za-z]{8}$/.test(t);
}
