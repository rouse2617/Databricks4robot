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

function escapeRegex(value: string): string {
	return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function splitRepeatedPrefix(value: string, prefix: string) {
	const trimmed = prefix.trim();
	if (!trimmed || value.includes("\n")) return value;
	return value.replace(
		new RegExp(`(?!^)(${escapeRegex(trimmed)})`, "g"),
		"\n$1",
	);
}

function inferRepeatedLogPrefix(value: string) {
	const match = value.match(/^([A-Za-z0-9_.:-]+)\s/);
	if (!match) return "";
	const prefix = match[1];
	const occurrences =
		value.match(new RegExp(escapeRegex(prefix), "g"))?.length ?? 0;
	return occurrences > 1 ? prefix : "";
}

export function normalizeLogContent(
	logContent: string,
	selectedNode: Pick<WorkflowNodeStatus, "podName"> | null,
) {
	const normalized = logContent.replace(/\r\n?/g, "\n");
	if (normalized.includes("\n")) {
		return normalized;
	}
	const podName = selectedNode?.podName?.trim();
	if (podName) {
		return splitRepeatedPrefix(normalized, podName);
	}
	return splitRepeatedPrefix(normalized, inferRepeatedLogPrefix(normalized));
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
