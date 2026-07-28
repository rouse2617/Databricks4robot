// Argo workflow-level priority (wf.Spec.Priority) presented as three tiers.
//
// Higher priority is admitted to Running first when the controller's
// parallelism queue is saturated; it never preempts already-running workflows
// and is a no-op when there is spare capacity. Values are only comparable among
// pools sharing one controller/namespace queue. Stored top-level in a target's
// resourceDefaults.priority (the pool default); a per-dispatch override may set
// it on a single batch (filter_json.priority).

export type PriorityTier = "low" | "normal" | "high";

export const WORKFLOW_PRIORITY: Record<PriorityTier, number> = {
	low: -100,
	normal: 0,
	high: 100,
};

// priorityTier buckets any stored integer into a tier by sign, so a custom
// value (or a future raw-integer knob) still renders sensibly.
export function priorityTier(value?: number | null): PriorityTier {
	if (typeof value !== "number" || Number.isNaN(value)) return "normal";
	if (value < 0) return "low";
	if (value > 0) return "high";
	return "normal";
}

// canonicalPriority maps a stored value to its tier's canonical integer, so a
// Select bound to {-100, 0, 100} always has a matching option.
export function canonicalPriority(value?: number | null): number {
	return WORKFLOW_PRIORITY[priorityTier(value)];
}

export const PRIORITY_TIER_LABEL: Record<PriorityTier, string> = {
	low: "低",
	normal: "普通",
	high: "高",
};

// antd Tag color per tier — two ramps only: grey (default) for low/normal,
// blue for high (the attention case).
const PRIORITY_TIER_TAG_COLOR: Record<PriorityTier, string | undefined> = {
	low: "default",
	normal: "default",
	high: "blue",
};

// priorityBadge resolves an effective priority value to a tag label + color for
// display (pool manager, dispatch form, batch list/detail).
export function priorityBadge(value?: number | null): {
	label: string;
	color?: string;
} {
	const tier = priorityTier(value);
	return {
		label: PRIORITY_TIER_LABEL[tier],
		color: PRIORITY_TIER_TAG_COLOR[tier],
	};
}

// batchTargetId extracts a batch's execution target id from its filter_json,
// which may store it under any of these aliases (matches the backend).
export function batchTargetId(
	filterJson?: Record<string, unknown>,
): string | undefined {
	for (const key of [
		"targetId",
		"target_id",
		"executionTargetId",
		"execution_target_id",
	]) {
		const v = filterJson?.[key];
		if (typeof v === "string" && v.trim() !== "") return v.trim();
	}
	return undefined;
}

// effectivePriorityFromFilter resolves a batch's effective priority for display:
// the per-batch override in filter_json.priority wins, else the pool default.
export function effectivePriorityFromFilter(
	filterJson: Record<string, unknown> | undefined,
	poolDefault: number | undefined,
): number | undefined {
	const raw = filterJson?.priority;
	if (typeof raw === "number") return raw;
	if (
		typeof raw === "string" &&
		raw.trim() !== "" &&
		!Number.isNaN(Number(raw))
	) {
		return Number(raw);
	}
	return poolDefault;
}

// Options for the pool-default Select (concrete tiers, high → normal → low).
export const PRIORITY_TIER_OPTIONS = [
	{ value: WORKFLOW_PRIORITY.high, label: "高 — 急单,繁忙时优先" },
	{ value: WORKFLOW_PRIORITY.normal, label: "普通(默认)" },
	{ value: WORKFLOW_PRIORITY.low, label: "低 — 后台大批量,繁忙时让路" },
];
