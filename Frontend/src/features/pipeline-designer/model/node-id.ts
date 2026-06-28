/** Stable unique IDs for designer canvas nodes. */

export function createPipelineNodeId(): string {
	if (
		typeof crypto !== "undefined" &&
		typeof crypto.randomUUID === "function"
	) {
		return `node-${crypto.randomUUID()}`;
	}
	return `node-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

/** Parse legacy `step-N` suffix for tests / migration helpers. */
export function parseLegacyStepCounter(id: string): number | null {
	const match = /^step-(\d+)$/.exec(id.trim());
	if (!match) return null;
	const value = Number(match[1]);
	return Number.isFinite(value) ? value : null;
}

export function maxLegacyStepCounter(ids: string[]): number {
	let max = 0;
	for (const id of ids) {
		const parsed = parseLegacyStepCounter(id);
		if (parsed !== null && parsed > max) max = parsed;
	}
	return max;
}
