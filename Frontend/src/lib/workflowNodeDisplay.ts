import type { WorkflowNodeStatus } from "../api/workflowApi";
import type { Pipeline } from "../components/pipeline/types";

export function truncateMiddle(text: string, maxLength: number): string {
	if (maxLength < 4 || text.length <= maxLength) {
		return text;
	}
	const edge = Math.floor((maxLength - 3) / 2);
	return `${text.slice(0, edge)}...${text.slice(text.length - edge)}`;
}

const UUID_PATTERN =
	/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function addPipelineNodeLabel(
	lookup: Map<string, string>,
	key: string | undefined,
	label: string,
) {
	const normalized = key?.trim();
	if (!normalized || lookup.has(normalized)) return;
	lookup.set(normalized, label);
}

function addPipelineNodeLabelVariants(
	lookup: Map<string, string>,
	nodeId: string,
	label: string,
) {
	const normalized = nodeId.trim();
	if (!normalized) return;
	addPipelineNodeLabel(lookup, normalized, label);
	addPipelineNodeLabel(lookup, `step-${normalized}`, label);

	const withoutStep = normalized.startsWith("step-")
		? normalized.slice("step-".length)
		: normalized;
	if (
		normalized.startsWith("step-") &&
		(withoutStep.startsWith("node-") || UUID_PATTERN.test(withoutStep))
	) {
		addPipelineNodeLabel(lookup, withoutStep, label);
	}

	if (withoutStep.startsWith("node-")) {
		const generatedSuffix = withoutStep.slice("node-".length);
		addPipelineNodeLabel(lookup, `step-${withoutStep}`, label);
		if (UUID_PATTERN.test(generatedSuffix)) {
			addPipelineNodeLabel(lookup, generatedSuffix, label);
			addPipelineNodeLabel(lookup, `step-${generatedSuffix}`, label);
		}
		return;
	}

	if (UUID_PATTERN.test(withoutStep)) {
		addPipelineNodeLabel(lookup, `node-${withoutStep}`, label);
		addPipelineNodeLabel(lookup, `step-node-${withoutStep}`, label);
	}
}

export function buildPipelineNodeLabelLookup(
	pipeline?: Pipeline | null,
): Map<string, string> {
	const lookup = new Map<string, string>();
	if (!pipeline?.nodes?.length) {
		return lookup;
	}
	for (const node of pipeline.nodes) {
		const label = node.component?.name?.trim() || node.id;
		addPipelineNodeLabelVariants(lookup, node.id, label);
	}
	return lookup;
}

function workflowNodeLookupKeys(node: WorkflowNodeStatus): string[] {
	const keys = [
		node.templateName?.trim(),
		node.displayName?.trim(),
		node.name?.split(".").pop()?.trim(),
	];
	const out: string[] = [];
	const seen = new Set<string>();
	for (const key of keys) {
		if (!key || seen.has(key)) continue;
		seen.add(key);
		out.push(key);
	}
	return out;
}

export function getWorkflowNodeDisplayText(
	node: WorkflowNodeStatus,
	pipelineLabels?: Map<string, string> | null,
): string {
	if (pipelineLabels && pipelineLabels.size > 0) {
		for (const key of workflowNodeLookupKeys(node)) {
			const label = pipelineLabels.get(key);
			if (label) {
				return label;
			}
		}
	}
	return node.displayName || node.templateName || node.name;
}

export function getWorkflowNodePodName(node: WorkflowNodeStatus): string {
	if (node.podName?.trim()) {
		return node.podName.trim();
	}
	const type = (node.type ?? "").toLowerCase();
	if (type === "pod" && node.id) {
		return node.id;
	}
	return node.name || node.id || "—";
}
