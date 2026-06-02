import type { Deployment, PipelineTemplate } from "../../api/pipelineApi";

export const TEMPLATE_PAGE_SIZE = 10;
export const DEPLOYMENT_PAGE_SIZE = 10;
export const SIDEBAR_TEMPLATE_LIMIT = 6;
export const COMPACT_TEMPLATE_LIMIT = 8;

export function sortByCreatedDesc<T extends { createdAt: string }>(
	items: T[],
): T[] {
	return [...items].sort(
		(a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
	);
}

export function dedupeTemplatesByName(
	items: PipelineTemplate[],
): PipelineTemplate[] {
	const byName = new Map<string, PipelineTemplate>();
	for (const item of items) {
		const key = item.name.trim().toLowerCase() || item.id;
		const prev = byName.get(key);
		if (
			!prev ||
			item.version > prev.version ||
			(item.version === prev.version &&
				new Date(item.createdAt).getTime() > new Date(prev.createdAt).getTime())
		) {
			byName.set(key, item);
		}
	}
	return sortByCreatedDesc(Array.from(byName.values()));
}

export function prepareDeployments(items: Deployment[]): Deployment[] {
	return sortByCreatedDesc(items);
}
