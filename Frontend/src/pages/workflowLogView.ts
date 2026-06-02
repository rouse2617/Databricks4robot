import type { WorkflowNodeStatus } from "../api/workflowApi";

export const LOG_MAX_RENDER_CHARS = 250_000;
export const LOG_MAX_RENDER_LINES = 2_000;

export type VisibleLogContent = {
	content: string;
	truncated: boolean;
	hiddenChars: number;
	hiddenLines: number;
	totalLines: number;
};

function countLines(value: string): number {
	if (value === "") return 0;
	let lines = 1;
	for (let index = 0; index < value.length; index += 1) {
		if (value.charCodeAt(index) === 10) {
			lines += 1;
		}
	}
	return lines;
}

export function normalizeLogContent(
	logContent: string,
	selectedNode: Pick<WorkflowNodeStatus, "podName"> | null,
) {
	const normalized = logContent.replace(/\r\n?/g, "\n");
	const podName = selectedNode?.podName?.trim();
	if (!podName || normalized.includes("\n")) {
		return normalized;
	}
	const escaped = podName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
	return normalized.replace(new RegExp(`(?!^)(${escaped})`, "g"), "\n$1");
}

export function prepareVisibleLogContent(
	logContent: string,
	selectedNode: Pick<WorkflowNodeStatus, "podName"> | null = null,
	maxChars = LOG_MAX_RENDER_CHARS,
	maxLines = LOG_MAX_RENDER_LINES,
): VisibleLogContent {
	const normalized = normalizeLogContent(logContent, selectedNode);
	const totalLines = countLines(normalized);
	const hiddenChars = Math.max(0, normalized.length - maxChars);
	const charLimited =
		hiddenChars > 0 ? normalized.slice(-maxChars) : normalized;
	const lines = charLimited.split("\n");
	const hiddenLines = Math.max(0, lines.length - maxLines);
	const visibleLines = hiddenLines > 0 ? lines.slice(-maxLines) : lines;
	return {
		content: visibleLines.join("\n"),
		truncated: hiddenChars > 0 || hiddenLines > 0,
		hiddenChars,
		hiddenLines,
		totalLines,
	};
}
