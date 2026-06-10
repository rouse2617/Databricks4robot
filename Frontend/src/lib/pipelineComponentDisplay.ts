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
