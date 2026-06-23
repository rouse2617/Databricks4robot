import type { WorkflowNodeStatus } from "../api/workflowApi";

export const LOG_ROW_HEIGHT = 20;

export interface LogLineEntry {
	text: string;
	index: number;
	searchMatch: boolean;
}

export interface LogContentModel {
	lines: LogLineEntry[];
	totalLines: number;
	truncated: boolean;
}

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
		const splitBySelectedPod = splitRepeatedPrefix(normalized, podName);
		if (splitBySelectedPod !== normalized) return splitBySelectedPod;
	}
	return splitRepeatedPrefix(normalized, inferRepeatedLogPrefix(normalized));
}

/** Split normalized log content into individual lines. */
export function splitIntoLines(content: string): string[] {
	const raw = content.endsWith("\n") ? content.slice(0, -1) : content;
	return raw === "" ? [] : raw.split("\n");
}

/** Build a LogContentModel from raw lines and an optional search query. */
export function buildContentModel(
	lines: string[],
	search: string,
	_truncated?: boolean,
): LogContentModel {
	const normalized = search.trim();
	const pattern = normalized
		? new RegExp(escapeRegex(normalized), "gi")
		: null;

	const entries: LogLineEntry[] = [];
	for (let i = 0; i < lines.length; i += 1) {
		entries.push({
			text: lines[i],
			index: i,
			searchMatch: pattern ? pattern.test(lines[i]) : false,
		});
	}

	return {
		lines: entries,
		totalLines: lines.length,
		truncated: _truncated ?? false,
	};
}
