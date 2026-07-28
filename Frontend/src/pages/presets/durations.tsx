// CYB-4304: durations preset for the shared `BatchAssetLookup` shell.
//
// This is the CYB-4294 batch duration-lookup logic re-expressed as a
// `BatchLookupPreset`. The shell (see `components/batch-lookup/BatchAssetLookup.tsx`)
// consumes this object; the page shim at `pages/AssetDurationLookup.tsx` just
// mounts `<BatchAssetLookup preset={durationsPreset} />`.

import { Col, Form, InputNumber } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { ReactNode } from "react";
import type {
	AssetDurationsResponse,
	DurationLookupItem,
} from "../../api/assets";
import { assetsApi } from "../../api/assets";
import { downloadCsv } from "../../components/batch-lookup/BatchAssetLookup";
import type {
	BatchLookupPreset,
	BatchLookupRequest,
	BatchLookupResponse,
	BucketRow,
	StatEntry,
} from "../../components/batch-lookup/types";

const MAX_IDS = 5000;

// Histogram bucket boundaries (ms). Boundaries are `[lo, hi)` — the last
// bucket is open-ended. Matches CYB-4294 spec.md.
export const HISTOGRAM_BUCKETS: Array<{
	label: string;
	loMs: number;
	hiMs: number; // Number.POSITIVE_INFINITY for the top bucket
}> = [
	{ label: "<1min", loMs: 0, hiMs: 60_000 },
	{ label: "1-10min", loMs: 60_000, hiMs: 600_000 },
	{ label: "10-30min", loMs: 600_000, hiMs: 1_800_000 },
	{ label: "30-60min", loMs: 1_800_000, hiMs: 3_600_000 },
	{ label: "60min+", loMs: 3_600_000, hiMs: Number.POSITIVE_INFINITY },
];

export function bucketItems(items: DurationLookupItem[]): BucketRow[] {
	const counts = HISTOGRAM_BUCKETS.map((b) => ({ label: b.label, count: 0 }));
	for (const it of items) {
		for (let i = 0; i < HISTOGRAM_BUCKETS.length; i += 1) {
			const b = HISTOGRAM_BUCKETS[i];
			if (it.duration_ms >= b.loMs && it.duration_ms < b.hiMs) {
				counts[i].count += 1;
				break;
			}
		}
	}
	return counts;
}

// Format a millisecond count into a compact human string. Used for the
// preset's aggregate stat cards (总时长 / 均值 / 最小 / 最大 / P50 / P90).
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

// Durations filter shape: two optional ms bounds. Both null/undefined means
// no server-side filtering.
export interface DurationsFilters {
	min_duration_ms?: number;
	max_duration_ms?: number;
}

// Adapter: cast the typed `DurationStats` object into the shell's generic
// `Record<string, number>` bag. Every field on DurationStats is a number,
// so a spread produces a valid Record<string, number> at runtime — the
// unknown-cast keeps the compiler happy across the generic boundary.
function toResponseEnvelope(
	raw: AssetDurationsResponse,
): BatchLookupResponse<DurationLookupItem> {
	return {
		items: raw.items,
		missing_ids: raw.missing_ids,
		filtered_out_ids: raw.filtered_out_ids,
		stats: { ...raw.stats } as unknown as Record<string, number>,
	};
}

const columns: ColumnsType<DurationLookupItem> = [
	{ title: "input_id", dataIndex: "input_id", key: "input_id" },
	{ title: "asset_id", dataIndex: "asset_id", key: "asset_id" },
	{
		title: "grace_video_id",
		dataIndex: "grace_video_id",
		key: "grace_video_id",
		render: (v?: string) => v ?? "-",
	},
	{
		title: "duration_ms",
		dataIndex: "duration_ms",
		key: "duration_ms",
		sorter: (a, b) => a.duration_ms - b.duration_ms,
		defaultSortOrder: "descend",
		align: "right",
	},
	{ title: "formatted", dataIndex: "formatted", key: "formatted" },
];

function renderFilters(
	filters: DurationsFilters,
	setFilters: (next: DurationsFilters) => void,
): ReactNode {
	return (
		<>
			<Col span={8}>
				<Form.Item label="最短时长">
					<InputNumber
						value={filters.min_duration_ms ?? null}
						onChange={(v) =>
							setFilters({
								...filters,
								min_duration_ms: v == null ? undefined : Number(v),
							})
						}
						min={0}
						addonAfter="ms"
						style={{ width: "100%" }}
						placeholder="0 = 不限"
						data-testid="min-duration-input"
					/>
				</Form.Item>
			</Col>
			<Col span={8}>
				<Form.Item label="最长时长">
					<InputNumber
						value={filters.max_duration_ms ?? null}
						onChange={(v) =>
							setFilters({
								...filters,
								max_duration_ms: v == null ? undefined : Number(v),
							})
						}
						min={0}
						addonAfter="ms"
						style={{ width: "100%" }}
						placeholder="0 = 不限"
						data-testid="max-duration-input"
					/>
				</Form.Item>
			</Col>
		</>
	);
}

function validateFilters(filters: DurationsFilters): string | null {
	const { min_duration_ms, max_duration_ms } = filters;
	if (
		min_duration_ms != null &&
		max_duration_ms != null &&
		min_duration_ms > max_duration_ms
	) {
		return "最短时长不能大于最长时长";
	}
	return null;
}

function extraStats(
	res: BatchLookupResponse<DurationLookupItem>,
): StatEntry[] {
	const s = res.stats;
	return [
		{ label: "总时长", value: formatDurationMs(s.total_ms ?? 0) },
		{ label: "均值", value: formatDurationMs(s.mean_ms ?? 0) },
		{ label: "最小", value: formatDurationMs(s.min_ms ?? 0) },
		{ label: "最大", value: formatDurationMs(s.max_ms ?? 0) },
		{ label: "P50", value: formatDurationMs(s.p50_ms ?? 0) },
		{ label: "P90", value: formatDurationMs(s.p90_ms ?? 0) },
	];
}

async function fetchDurations(
	req: BatchLookupRequest<DurationsFilters>,
): Promise<BatchLookupResponse<DurationLookupItem>> {
	const raw = await assetsApi.lookupDurations({
		ids: req.ids,
		id_type: req.id_type,
		...(req.filters.min_duration_ms != null &&
		req.filters.min_duration_ms > 0
			? { min_duration_ms: req.filters.min_duration_ms }
			: {}),
		...(req.filters.max_duration_ms != null &&
		req.filters.max_duration_ms > 0
			? { max_duration_ms: req.filters.max_duration_ms }
			: {}),
	});
	return toResponseEnvelope(raw);
}

export const durationsPreset: BatchLookupPreset<
	DurationLookupItem,
	DurationsFilters
> = {
	key: "durations",
	placeholder: "aaaaaaaa\nbbbbbbbb\n019f8319-0ef3-7da0-815c-30f23d21e7f1",
	maxIds: MAX_IDS,
	fetch: fetchDurations,
	filters: {
		initial: { min_duration_ms: undefined, max_duration_ms: undefined },
		render: renderFilters,
		validate: validateFilters,
	},
	columns,
	extraStats,
	histogram: bucketItems,
	csv: {
		filename: (matchedCount: number) => `asset-durations-${matchedCount}.csv`,
		header: [
			"input_id",
			"asset_id",
			"grace_video_id",
			"duration_ms",
			"formatted",
		],
		row: (it) => [
			it.input_id,
			it.asset_id,
			it.grace_video_id ?? "",
			it.duration_ms,
			it.formatted,
		],
	},
};

// Backward-compat wrapper for CYB-4294 unit tests that exercise the CSV
// helper directly. The shell (BatchAssetLookup) uses `preset.csv.*` at
// runtime; this thin adapter routes those fields into `downloadCsv` so a
// test can still assert filename/anchor behavior without mounting the shell.
export function exportDurationsCsv(
	items: DurationLookupItem[],
	matchedCount: number,
): void {
	downloadCsv(items, matchedCount, durationsPreset.csv);
}
