import dayjs from "dayjs";
import type { BatchJob } from "../api/batchJobApi";
import type { PipelineRun } from "../api/pipelineApi";
import { listRunChildren } from "../api/runApi";

export function batchJobCreatedAtMs(job: BatchJob): number {
	const value = dayjs(job.createdAt).valueOf();
	return Number.isFinite(value) ? value : 0;
}

/**
 * Human-friendly ms duration used across the batch surfaces:
 * `Xh Ym Ns` / `Ym Ns` / `Ns` / `Nms` for sub-second, `-` for invalid input.
 * Moved from the durations preset for reuse by the batch job list column
 * (CYB-4350). The presets file re-exports this so existing call sites keep
 * working without churn.
 */
export function formatDurationMs(ms: number): string {
	if (!Number.isFinite(ms) || ms < 0) return "-";
	if (ms < 1000) return `${ms}ms`;
	const totalSec = Math.floor(ms / 1000);
	const h = Math.floor(totalSec / 3600);
	const m = Math.floor((totalSec % 3600) / 60);
	const s = totalSec % 60;
	if (h > 0) return `${h}h ${m}m ${s}s`;
	if (m > 0) return `${m}m ${s}s`;
	return `${s}s`;
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

/**
 * Wait time in seconds = wall-clock (creation → runFinishedAt) minus real run
 * span. Captures the time spent in queue / submit / pause not actually
 * executing subtasks. Null when the batch lacks either a run span or a
 * creation timestamp, or when the wait would be negative (which would only
 * happen with a malformed backend timestamp).
 *
 * Used by the BatchJobList "耗时（含等待）" column to surface the wait time in
 * a Tooltip next to the actual run duration (CYB-4477).
 */
export function batchJobTotalWaitSeconds(job: BatchJob): number | null {
	const real = batchJobRunDurationSeconds(job);
	if (real === null || !job.createdAt || !job.runFinishedAt) return null;
	const wall = dayjs(job.runFinishedAt).diff(dayjs(job.createdAt), "second");
	return wall > real ? wall - real : null;
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

// CYB-3822: server-side status query value. Backend expects Argo TitleCase
// (batchItemRunStatusExpr in pipeline_repo.go rewrites bi.status → "Succeeded"
// / "Failed" / …). undefined for "all" means the caller should omit the query
// param entirely. Kept alongside statusFilterPredicate so the client-side
// predicate stays as a safety net if a legacy deploy ignores the param.
export function statusFilterServerValue(
	filter: BatchExportStatusFilter,
): string | undefined {
	switch (filter) {
		case "succeeded":
			return "Succeeded";
		case "failed":
			return "Failed";
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
	options: {
		filterFn?: (run: PipelineRun) => boolean;
		// CYB-3822: server-side status filter (Argo TitleCase). When set the
		// backend applies WHERE status=? so only matching rows come back, and
		// each page's item count reflects the filter — total shrinks from
		// e.g. 10000 → 42 for a "just failed" scan of a mostly-successful batch.
		status?: string;
	} = {},
): Promise<string[]> {
	const pageSize = 100;
	// First page tells us `total` — keep pulling pages until we've either
	// collected total items or a page comes back empty (safety valve so a
	// truncated `total` on the server side does not spin forever).
	const first = await listRunChildren(batchId, {
		page: 1,
		pageSize,
		status: options.status,
	});
	const collected: PipelineRun[] = [...first.items];
	const total = first.total ?? first.items.length;
	let page = 2;
	while (collected.length < total) {
		const part = await listRunChildren(batchId, {
			page,
			pageSize,
			status: options.status,
		});
		if (part.items.length === 0) break;
		collected.push(...part.items);
		page += 1;
	}
	return extractAssetIds(collected, options.filterFn);
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
		// CYB-3822: passed through to each batch's fetchAllBatchAssetIds.
		status?: string;
		// CYB-3822: invoked once per batch after its fetch completes. UI uses
		// this to render "X/Y 批次已拉取" progress so the loading toast does
		// not look like a hang on large multi-batch exports.
		onBatchDone?: (done: number, total: number) => void;
	} = {},
): Promise<string[]> {
	const concurrency = options.concurrency ?? 4;
	const union = new Set<string>();
	let cursor = 0;
	let doneCount = 0;
	const totalBatches = batchIds.length;
	async function worker(): Promise<void> {
		while (cursor < batchIds.length) {
			const idx = cursor;
			cursor += 1;
			const ids = await fetchAllBatchAssetIds(batchIds[idx], {
				filterFn: options.filterFn,
				status: options.status,
			});
			for (const id of ids) union.add(id);
			doneCount += 1;
			options.onBatchDone?.(doneCount, totalBatches);
		}
	}
	const workers = Array.from(
		{ length: Math.min(concurrency, batchIds.length) },
		() => worker(),
	);
	await Promise.all(workers);
	return Array.from(union);
}
