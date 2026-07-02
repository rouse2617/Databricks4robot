import type { PipelineStats, PipelineTemplate } from "../api/pipelineApi";

export interface GroupedPipelineSet {
	title: string;
	icon: string;
	pipelines: PipelineTemplate[];
	count: number;
}

export function groupPipelinesWithStats(
	allPipelines: PipelineTemplate[],
	stats: PipelineStats | null,
	userTags?: Record<string, string[]>,
): GroupedPipelineSet[] {
	if (!stats) {
		// Fallback: first 5 as "most used", rest as "all"
		const mostUsed = allPipelines.slice(0, 5);
		const rest = allPipelines.slice(5);
		return [
			{
				title: "🔥 最常用",
				icon: "🔥",
				pipelines: mostUsed,
				count: mostUsed.length,
			},
			{ title: "📋 全部", icon: "📋", pipelines: rest, count: rest.length },
		];
	}

	// Get recommendation IDs for "most used"
	const recommendedIds = new Set(
		stats.recommendations.slice(0, 5).map((r) => r.pipelineId),
	);

	const mostUsed = allPipelines.filter((p) => recommendedIds.has(p.id));
	const rest = allPipelines.filter((p) => !recommendedIds.has(p.id));

	const groups: GroupedPipelineSet[] = [
		{
			title: "🔥 最常用",
			icon: "🔥",
			pipelines: mostUsed,
			count: mostUsed.length,
		},
	];

	// Add tagged group if user has tags
	if (userTags && Object.keys(userTags).length > 0) {
		const taggedIds = new Set(Object.values(userTags).flat());
		const tagged = allPipelines.filter((p) => taggedIds.has(p.id));
		if (tagged.length > 0) {
			groups.push({
				title: "📌 我的标记",
				icon: "📌",
				pipelines: tagged,
				count: tagged.length,
			});
		}
	}

	// Add "all" group
	groups.push({
		title: "📋 全部",
		icon: "📋",
		pipelines: rest,
		count: rest.length,
	});

	return groups;
}

export function filterPipelines(
	groups: GroupedPipelineSet[],
	query: string,
): GroupedPipelineSet[] {
	if (!query) return groups;

	const lowerQuery = query.toLowerCase();
	return groups
		.map((group) => ({
			...group,
			pipelines: group.pipelines.filter(
				(p) =>
					p.name.toLowerCase().includes(lowerQuery) ||
					p.id.toLowerCase().includes(lowerQuery),
			),
			count: group.pipelines.filter(
				(p) =>
					p.name.toLowerCase().includes(lowerQuery) ||
					p.id.toLowerCase().includes(lowerQuery),
			).length,
		}))
		.filter((g) => g.count > 0);
}

export function sortPipelinesByScore(
	pipelines: PipelineTemplate[],
	stats: PipelineStats | null,
): PipelineTemplate[] {
	if (!stats) return pipelines;

	const scoreMap = new Map(
		stats.recommendations.map((r) => [r.pipelineId, r.score]),
	);

	return [...pipelines].sort((a, b) => {
		const scoreA = scoreMap.get(a.id) ?? 0;
		const scoreB = scoreMap.get(b.id) ?? 0;
		return scoreB - scoreA;
	});
}
