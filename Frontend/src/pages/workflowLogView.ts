import type { WorkflowNodeStatus } from "../api/workflowApi";

export const LOG_ROW_HEIGHT = 20;

export interface LogLineEntry {
	text: string;
	index: number;
	searchMatch: boolean;
	isError: boolean;
}

export interface LogContentModel {
	lines: LogLineEntry[];
	totalLines: number;
	truncated: boolean;
	errorLineIndexes: number[];
}

// Keywords that mark a log line as an error/failure worth surfacing in the
// gutter, scrollbar ticks and "jump to first error" affordance.
const ERROR_LINE_PATTERN =
	/\b(error|err|fatal|panic|traceback|exception|failed|failure|exit status [1-9]|permissiondenied|denied|forbidden|unauthorized|timeout|timed out|oom|killed)\b/i;

/** Heuristically decide whether a single log line represents an error. */
export function isErrorLogLine(text: string): boolean {
	if (!text) return false;
	return ERROR_LINE_PATTERN.test(text);
}

// function countLines(value: string): number {
// 	if (value === "") return 0;
// 	let lines = 1;
// 	for (let index = 0; index < value.length; index += 1) {
// 		if (value.charCodeAt(index) === 10) {
// 			lines += 1;
// 		}
// 	}
// 	return lines;
// }

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

/**
 * Detect a long pod-name style prefix that is repeated at the start of most
 * log lines (e.g. "my-workflow-step-1234567890-abc:"). Returns "" when no
 * dominant prefix is found, so callers can leave content untouched.
 */
export function detectCommonLinePrefix(lines: string[]): string {
	const counts = new Map<string, number>();
	let considered = 0;
	for (const line of lines) {
		if (!line.trim()) continue;
		considered += 1;
		const match = line.match(/^([A-Za-z0-9_.:-]+)\s/);
		if (!match) continue;
		const token = match[1];
		counts.set(token, (counts.get(token) ?? 0) + 1);
	}
	if (considered === 0) return "";
	let best = "";
	let bestCount = 0;
	for (const [token, count] of counts) {
		if (count > bestCount) {
			best = token;
			bestCount = count;
		}
	}
	// Require the prefix to dominate and be long enough that it is clearly a
	// pod/step identifier rather than a meaningful first word of the log line.
	const threshold = Math.max(2, Math.ceil(considered * 0.6));
	if (bestCount >= threshold && best.length >= 8) return best;
	return "";
}

/** Remove a detected line prefix (and trailing whitespace) from a single line. */
export function stripLinePrefix(line: string, prefix: string): string {
	if (!prefix || !line.startsWith(prefix)) return line;
	return line.slice(prefix.length).replace(/^\s+/, "");
}

/** Apply {@link stripLinePrefix} to every line for a given prefix. */
export function stripLinePrefixes(lines: string[], prefix: string): string[] {
	if (!prefix) return lines;
	return lines.map((line) => stripLinePrefix(line, prefix));
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
	const pattern = normalized ? new RegExp(escapeRegex(normalized), "gi") : null;

	const entries: LogLineEntry[] = [];
	const errorLineIndexes: number[] = [];
	for (let i = 0; i < lines.length; i += 1) {
		const isError = isErrorLogLine(lines[i]);
		if (isError) errorLineIndexes.push(i);
		entries.push({
			text: lines[i],
			index: i,
			searchMatch: pattern ? pattern.test(lines[i]) : false,
			isError,
		});
	}

	return {
		lines: entries,
		totalLines: lines.length,
		truncated: _truncated ?? false,
		errorLineIndexes,
	};
}
