export interface EmbedLichtblickFlagOptions {
	search?: string;
	env?: ImportMetaEnv;
}

function parseTruthyFlag(v: string | null | undefined): boolean {
	if (!v) return false;
	const s = v.trim().toLowerCase();
	return s === "1" || s === "true" || s === "yes" || s === "on";
}

/**
 * Feature flag for enabling embedded Lichtblick player shell.
 *
 * Priority:
 * - URL query param `embed_player` / `embedPlayer` / `lichtblick` (truthy: 1/true/yes/on)
 * - Vite env `VITE_ENABLE_EMBED_LICHTBLICK_PLAYER` (same truthy values)
 */
export function isEmbedLichtblickPlayerEnabled(
	opts?: EmbedLichtblickFlagOptions,
): boolean {
	const search =
		opts?.search ??
		(typeof window !== "undefined" ? window.location.search : "");
	let sp: URLSearchParams;
	try {
		sp = new URLSearchParams(search.startsWith("?") ? search : `?${search}`);
	} catch {
		sp = new URLSearchParams();
	}

	const qp =
		sp.get("embed_player") ??
		sp.get("embedPlayer") ??
		sp.get("lichtblick") ??
		null;
	if (parseTruthyFlag(qp)) return true;
	if (qp !== null) return false; // explicit override

	const env = opts?.env ?? (import.meta as ImportMeta).env;
	return parseTruthyFlag(env.VITE_ENABLE_EMBED_LICHTBLICK_PLAYER);
}
