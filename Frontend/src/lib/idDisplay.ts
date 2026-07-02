const ASSET_STYLE_ID_LENGTH = 8;

export function toAssetStyleId(id?: string | null): string {
	const compact = (id ?? "").replace(/[^0-9A-Za-z]/g, "");
	if (!compact) return "--------";
	return compact.slice(0, ASSET_STYLE_ID_LENGTH);
}
