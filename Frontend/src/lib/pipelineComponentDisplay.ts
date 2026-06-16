import type { PipelineComponentAPI } from "../api/pipelineComponentApi";

export function imageHasTag(image: string): boolean {
	if (image.includes("@")) return true;
	const lastSlash = image.lastIndexOf("/");
	const lastColon = image.lastIndexOf(":");
	return lastColon > lastSlash && lastColon < image.length - 1;
}

export function formatComponentImage(image: string, tag?: string): string {
	if (!tag || imageHasTag(image)) return image;
	return `${image}:${tag}`;
}

/** Prefer stable system ids (sys-*) when API returns duplicate names. */
export function dedupePipelineComponentsByName(
	items: PipelineComponentAPI[],
): PipelineComponentAPI[] {
	const byName = new Map<string, PipelineComponentAPI>();

	const score = (item: PipelineComponentAPI): number => {
		let value = 0;
		if (item.id.startsWith("sys-")) value += 4;
		if (item.source === "system") value += 2;
		return value;
	};

	for (const item of items) {
		const key = item.name.trim().toLowerCase();
		if (!key) continue;
		const existing = byName.get(key);
		if (!existing || score(item) > score(existing)) {
			byName.set(key, item);
		}
	}

	return Array.from(byName.values());
}

/** Strip artificial zero-padding some ingest paths append to short git SHAs. */
export function normalizeGitCommit(value?: string): string {
	const trimmed = (value || "").trim();
	if (!trimmed) return "";
	const lower = trimmed.toLowerCase();
	if (!/^[0-9a-f]+$/.test(lower)) return trimmed;
	if (lower.length >= 40 && /^[0-9a-f]{4,12}0+$/.test(lower)) {
		return lower.replace(/0+$/, "");
	}
	return lower;
}

export function formatCommitDisplay(value?: string): string {
	const normalized = normalizeGitCommit(value);
	if (!normalized) return "-";
	if (normalized.length <= 12) return normalized;
	return `${normalized.slice(0, 10)}…${normalized.slice(-8)}`;
}
