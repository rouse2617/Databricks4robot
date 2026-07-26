const UUID_RE =
	/^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

/**
 * True when s is a standard UUID (e.g. a Grace video id). Used to detect when
 * a Grace UUID is used where a DataBrew 8-char asset_id is expected (CYB-4011).
 */
export function isUUID(s: string): boolean {
	return UUID_RE.test(s.trim());
}

/**
 * True when s is a valid asset_id: 8 ASCII alphanumeric chars or UUID.
 */
export function isCanonicalAssetId(s: string): boolean {
	const t = s.trim();
	return /^[0-9A-Za-z]{8}$/.test(t) || UUID_RE.test(t);
}
