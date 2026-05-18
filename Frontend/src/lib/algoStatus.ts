// Shared parsing/normalization for algo_results raw values.
// Used by both the algo matrix grid (AlgoProcessingPage) and the batch retry
// hook (useRetryAllFailed) so they always agree on which cells are "failed".

import type { CellStatus } from "../components/algo-matrix/AlgoStatusCell";

export type { CellStatus };

export function normalizeStatus(s: string): CellStatus {
	const lower = s.toLowerCase();
	if (lower === "ok" || lower === "success") return "ok";
	if (lower === "failed" || lower === "error") return "failed";
	if (lower === "running") return "running";
	if (lower === "pending") return "pending";
	if (lower === "blocked") return "blocked";
	return "none";
}

// Accepts either a plain status string ("ok", "failed", ...) or a JSON-encoded
// object with a `status` field. Returns the normalized CellStatus.
export function parseStatusFromRaw(raw?: string): CellStatus {
	if (!raw) return "none";
	try {
		const parsed = JSON.parse(raw);
		if (
			parsed &&
			typeof parsed === "object" &&
			typeof parsed.status === "string"
		) {
			return normalizeStatus(parsed.status);
		}
		return normalizeStatus(String(parsed));
	} catch {
		return normalizeStatus(raw);
	}
}

export function isFailedStatus(status: CellStatus): boolean {
	return status === "failed";
}

export interface AlgoResultDetail {
	status?: string;
	started_at?: string;
	finished_at?: string;
	method?: string;
	run_id?: string;
	output_uri?: string;
	reason?: string;
}

export function getAlgoResultDetail(
	algoResults: Record<string, string> | undefined,
	algoKey: string,
): AlgoResultDetail | null {
	if (!algoResults) return null;
	const detail: AlgoResultDetail = {};

	const legacyRaw = algoResults[algoKey];
	if (legacyRaw) {
		try {
			const parsed = JSON.parse(legacyRaw);
			if (parsed && typeof parsed === "object") {
				if (typeof parsed.status === "string") detail.status = parsed.status;
				if (typeof parsed.started_at === "string")
					detail.started_at = parsed.started_at;
				if (typeof parsed.finished_at === "string")
					detail.finished_at = parsed.finished_at;
				if (typeof parsed.method === "string") detail.method = parsed.method;
				if (typeof parsed.run_id === "string") detail.run_id = parsed.run_id;
				if (typeof parsed.output_uri === "string")
					detail.output_uri = parsed.output_uri;
				if (typeof parsed.reason === "string") detail.reason = parsed.reason;
			} else {
				detail.status = String(parsed);
			}
		} catch {
			detail.status = legacyRaw;
		}
	}

	const prefix = `${algoKey}:`;
	for (const [key, value] of Object.entries(algoResults)) {
		if (!key.startsWith(prefix)) continue;
		const field = key.slice(prefix.length) as keyof AlgoResultDetail;
		detail[field] = value;
	}
	return Object.keys(detail).length > 0 ? detail : null;
}

export function getAlgoStatusFromResults(
	algoResults: Record<string, string> | undefined,
	algoKey: string,
): CellStatus {
	return parseStatusFromRaw(getAlgoResultDetail(algoResults, algoKey)?.status);
}
