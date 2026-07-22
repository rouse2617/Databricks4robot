import dayjs from "dayjs";
import type { BatchJob } from "../api/batchJobApi";
import type { PipelineRun } from "../api/pipelineApi";
import { listRunChildren } from "../api/runApi";

export function batchJobCreatedAtMs(job: BatchJob): number {
	const value = dayjs(job.createdAt).valueOf();
	return Number.isFinite(value) ? value : 0;
}

export function sortBatchJobsByCreatedDesc(jobs: BatchJob[]): BatchJob[] {
	return [...jobs].sort(
		(a, b) => batchJobCreatedAtMs(b) - batchJobCreatedAtMs(a),
	);
}

function isTerminalBatchJob(job: BatchJob): boolean {
	return job.status === "completed" || job.status === "failed";
}

/**
 * Completion timestamp for display: the job's own finishedAt, or the latest
 * subtask finish (runFinishedAt) as a fallback for terminal jobs whose
 * finished_at was never stamped (legacy / async-finalized batches). Undefined
 * for non-terminal jobs (so running/paused show "—").
 */
export function batchJobCompletionAt(job: BatchJob): string | undefined {
	return (
		job.finishedAt ?? (isTerminalBatchJob(job) ? job.runFinishedAt : undefined)
	);
}

/**
 * Real run duration in seconds — the subtask run span (first subtask start →
 * last subtask finish), which excludes submit/queue/pause waiting. This is NOT
 * finished_at - created_at (wall-clock incl. queueing). Null when the batch is
 * not terminal, has no run span, or the span is negative.
 */
export function batchJobRunDurationSeconds(job: BatchJob): number | null {
	if (!isTerminalBatchJob(job) || !job.runStartedAt || !job.runFinishedAt) {
		return null;
	}
	const secs = dayjs(job.runFinishedAt).diff(dayjs(job.runStartedAt), "second");
	return secs >= 0 ? secs : null;
}

export function formatDurationSeconds(secs: number): string {
	if (secs < 60) return `${secs}s`;
	const m = Math.floor(secs / 60);
	const s = secs % 60;
	return `${m}m ${s}s`;
}

// ─── Asset ID export (CYB-3800) ─────────────────────────────────────────────
// Extracted from BatchJobDetailPage so the batch-list page can reuse the same
// dedup + CSV + clipboard logic for the multi-select export flow.

/**
 * Backend run rows carry asset_ids on each PipelineRun. Return the union across
 * all rows with empty / whitespace-only ids dropped and insertion order
 * preserved. When `filterFn` is supplied (CYB-3821), only runs it returns true
 * for contribute their asset_ids — used by the "仅成功 / 仅失败" export
 * variants to narrow the export to a subset of child-run statuses.
 */
export function extractAssetIds(
	runs: PipelineRun[],
	filterFn?: (run: PipelineRun) => boolean,
): string[] {
	const ids = new Set<string>();
	for (const item of runs) {
		if (filterFn && !filterFn(item)) continue;
		if (item.assetIds && item.assetIds.length > 0) {
			for (const id of item.assetIds) {
				const trimmed = id?.trim();
				if (trimmed) ids.add(trimmed);
			}
		}
	}
	return Array.from(ids);
}

// CYB-3821: status filters used by the batch-export dropdown. Kept as a
// discriminated union of pure predicates so the UI can label them without
// duplicating the truth about which run.status values count as success/failure.
export type BatchExportStatusFilter = "all" | "succeeded" | "failed";

// CYB-3821a: backend serializes run status in TitleCase ("Succeeded",
// "Failed", "Running", "Pending"), not lowercase — the aggregate summary
// fields (succeededCount etc.) misled the initial implementation into
// comparing against "succeeded"/"failed" and always returning zero.
// toLowerCase() covers both future casings and any legacy rows.
export function statusFilterPredicate(
	filter: BatchExportStatusFilter,
): ((run: PipelineRun) => boolean) | undefined {
	switch (filter) {
		case "succeeded":
			return (run) => run.status?.toLowerCase() === "succeeded";
		case "failed":
			return (run) => run.status?.toLowerCase() === "failed";
		case "all":
			return undefined;
	}
}

/**
 * Trigger a browser download of a single-column asset_id CSV.
 * `filenameSuffix` is placed after `asset-ids-` in the filename (e.g.
 * `"batch-abc12345"` → `asset-ids-batch-abc12345.csv`).
 */
export function exportAssetIdsCsv(
	assetIds: string[],
	filenameSuffix: string,
): void {
	const header = "assetId\n";
	// CSV: escape embedded double quotes by doubling them.
	const rows = assetIds
		.map((id) => `"${id.replace(/"/g, '""')}"`)
		.join("\n");
	const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8" });
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = `asset-ids-${filenameSuffix}.csv`;
	// Firefox requires the anchor to be in the DOM before .click() fires.
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
	URL.revokeObjectURL(url);
}

/**
 * Copy the asset_ids as a newline-joined string to the clipboard. Tries the
 * modern Clipboard API first, falls back to a hidden textarea + execCommand
 * so it still works on http origins / older browsers.
 */
export async function copyAssetIdsToClipboard(
	assetIds: string[],
): Promise<boolean> {
	const text = assetIds.join("\n");
	if (navigator.clipboard && navigator.clipboard.writeText) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {
			// fall through to legacy path
		}
	}
	try {
		const ta = document.createElement("textarea");
		ta.value = text;
		ta.style.position = "fixed";
		ta.style.opacity = "0";
		document.body.appendChild(ta);
		ta.select();
		const ok = document.execCommand("copy");
		document.body.removeChild(ta);
		return ok;
	} catch {
		return false;
	}
}

/**
 * Fetch every child run for a batch, iterating pages up to `total`. Server
 * caps pageSize at 100 (CYB-3491). Returns the asset_ids union across all
 * pages. Uses the same runId endpoint that the detail page uses.
 */
export async function fetchAllBatchAssetIds(
	batchId: string,
	filterFn?: (run: PipelineRun) => boolean,
): Promise<string[]> {
	const pageSize = 100;
	// First page tells us `total` — keep pulling pages until we've either
	// collected total items or a page comes back empty (safety valve so a
	// truncated `total` on the server side does not spin forever).
	const first = await listRunChildren(batchId, { page: 1, pageSize });
	const collected: PipelineRun[] = [...first.items];
	const total = first.total ?? first.items.length;
	let page = 2;
	while (collected.length < total) {
		const part = await listRunChildren(batchId, { page, pageSize });
		if (part.items.length === 0) break;
		collected.push(...part.items);
		page += 1;
	}
	return extractAssetIds(collected, filterFn);
}

/**
 * Fetch asset_ids across many batches concurrently and return a deduped union.
 * Concurrency is bounded (default 4) so we don't slam the backend when the
 * user selects a large number of batches.
 */
export async function fetchAssetIdsForBatches(
	batchIds: string[],
	options: {
		concurrency?: number;
		filterFn?: (run: PipelineRun) => boolean;
	} = {},
): Promise<string[]> {
	const concurrency = options.concurrency ?? 4;
	const union = new Set<string>();
	let cursor = 0;
	async function worker(): Promise<void> {
		while (cursor < batchIds.length) {
			const idx = cursor;
			cursor += 1;
			const ids = await fetchAllBatchAssetIds(batchIds[idx], options.filterFn);
			for (const id of ids) union.add(id);
		}
	}
	const workers = Array.from(
		{ length: Math.min(concurrency, batchIds.length) },
		() => worker(),
	);
	await Promise.all(workers);
	return Array.from(union);
}
