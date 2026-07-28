// CYB-4306: costs preset for the shared `BatchAssetLookup` shell.
//
// Structure mirrors CYB-4304's durations preset. Everything cost-specific
// (endpoint call, filters render, columns, histogram buckets, csv shape)
// lives here so the shell doesn't need any awareness of what a "cost" is.

import { Col, DatePicker, Form, Segmented } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import type { ReactNode } from "react";
import type {
	AssetCostByAlgo,
	AssetCostItem,
	AssetCostsResponse,
	AssetCostGroupBy,
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

const { RangePicker } = DatePicker;

const MAX_IDS = 5000;
const MAX_WINDOW_DAYS = 90;

// Cost buckets ($). Boundaries are `[lo, hi)` — the last one is open-ended.
// Chosen for FinOps skew: most assets cost cents per run, so the first
// bucket collects the majority and the last two buckets flag outliers.
export const COST_BUCKETS: Array<{ label: string; lo: number; hi: number }> = [
	{ label: "<$0.10", lo: 0, hi: 0.1 },
	{ label: "$0.10-$1", lo: 0.1, hi: 1 },
	{ label: "$1-$10", lo: 1, hi: 10 },
	{ label: "$10-$100", lo: 10, hi: 100 },
	{ label: "$100+", lo: 100, hi: Number.POSITIVE_INFINITY },
];

export function bucketCostItems(items: AssetCostItem[]): BucketRow[] {
	const counts = COST_BUCKETS.map((b) => ({ label: b.label, count: 0 }));
	for (const it of items) {
		for (let i = 0; i < COST_BUCKETS.length; i += 1) {
			const b = COST_BUCKETS[i];
			if (it.total_cost_usd >= b.lo && it.total_cost_usd < b.hi) {
				counts[i].count += 1;
				break;
			}
		}
	}
	return counts;
}

// Format a USD amount. Uses 4 decimals for < $10 (most items cost cents),
// 2 decimals above. `-` when v is NaN/undefined.
export function formatUsd(v: number | undefined | null): string {
	if (v == null || !Number.isFinite(v)) return "-";
	const abs = Math.abs(v);
	return abs < 10 ? `$${v.toFixed(4)}` : `$${v.toFixed(2)}`;
}

function formatMinutes(seconds: number): string {
	if (!Number.isFinite(seconds) || seconds < 0) return "0.0 min";
	return `${(seconds / 60).toFixed(1)} min`;
}

// Costs filter shape. Both dates required (validated below), group_by
// defaults to "asset".
export interface CostsFilters {
	start_at: string | undefined; // ISO
	end_at: string | undefined; // ISO
	group_by: AssetCostGroupBy;
}

function toResponseEnvelope(
	raw: AssetCostsResponse,
): BatchLookupResponse<AssetCostItem> {
	return {
		items: raw.items,
		missing_ids: raw.missing_ids,
		filtered_out_ids: raw.filtered_out_ids,
		stats: { ...raw.stats } as unknown as Record<string, number>,
	};
}

// buildColumns constructs the sortable column set for a given group_by.
// Called from the exported `costsPreset.columns` initializer with the
// preset's initial group_by. The asset_algo mode has an extra algo_key
// column between grace_video_id and total_cost_usd; the asset mode does
// not (by_algo is null, so the column would be all "-").
export function buildColumns(groupBy: AssetCostGroupBy): ColumnsType<AssetCostItem> {
	const baseHead: ColumnsType<AssetCostItem> = [
		{ title: "input_id", dataIndex: "input_id", key: "input_id" },
		{ title: "asset_id", dataIndex: "asset_id", key: "asset_id" },
		{
			title: "grace_video_id",
			dataIndex: "grace_video_id",
			key: "grace_video_id",
			render: (v?: string) => v ?? "-",
		},
	];
	const algoCol: ColumnsType<AssetCostItem> =
		groupBy === "asset_algo"
			? [
					{
						title: "algo_key",
						key: "algo_key",
						render: (_, r) => {
							const byAlgo = r.by_algo ?? [];
							if (byAlgo.length === 0) return "-";
							if (byAlgo.length === 1) return byAlgo[0].algo_key;
							return byAlgo.map((a: AssetCostByAlgo) => a.algo_key).join(", ");
						},
					},
				]
			: [];
	const tail: ColumnsType<AssetCostItem> = [
		{
			title: "total_cost_usd",
			dataIndex: "total_cost_usd",
			key: "total_cost_usd",
			sorter: (a, b) => a.total_cost_usd - b.total_cost_usd,
			defaultSortOrder: "descend",
			align: "right",
			render: (v: number) => formatUsd(v),
		},
		{
			title: "gpu_min",
			dataIndex: "gpu_sec",
			key: "gpu_min",
			align: "right",
			sorter: (a, b) => a.gpu_sec - b.gpu_sec,
			render: (v: number) => formatMinutes(v),
		},
		{
			title: "cpu_min",
			dataIndex: "cpu_sec",
			key: "cpu_min",
			align: "right",
			sorter: (a, b) => a.cpu_sec - b.cpu_sec,
			render: (v: number) => formatMinutes(v),
		},
		{
			title: "run_count",
			dataIndex: "run_count",
			key: "run_count",
			align: "right",
			sorter: (a, b) => a.run_count - b.run_count,
		},
	];
	return [...baseHead, ...algoCol, ...tail];
}

function renderFilters(
	filters: CostsFilters,
	setFilters: (next: CostsFilters) => void,
): ReactNode {
	const start = filters.start_at ? dayjs(filters.start_at) : null;
	const end = filters.end_at ? dayjs(filters.end_at) : null;
	return (
		<>
			<Col span={12}>
				<Form.Item label="时间窗">
					<RangePicker
						value={[start, end] as [Dayjs | null, Dayjs | null]}
						showTime
						onChange={(dates) => {
							const [s, e] = (dates ?? [null, null]) as [
								Dayjs | null,
								Dayjs | null,
							];
							setFilters({
								...filters,
								start_at: s?.toISOString() ?? undefined,
								end_at: e?.toISOString() ?? undefined,
							});
						}}
						placeholder={["开始时间", "结束时间"]}
						style={{ width: "100%" }}
						data-testid="cost-date-range"
					/>
				</Form.Item>
			</Col>
			<Col span={4}>
				<Form.Item label="分组">
					<Segmented<AssetCostGroupBy>
						value={filters.group_by}
						onChange={(v) => setFilters({ ...filters, group_by: v })}
						options={[
							{ label: "按 asset", value: "asset" },
							{ label: "按 asset × 算法", value: "asset_algo" },
						]}
						data-testid="cost-group-by"
					/>
				</Form.Item>
			</Col>
		</>
	);
}

export function validateCostsFilters(filters: CostsFilters): string | null {
	const { start_at, end_at } = filters;
	if (!start_at || !end_at) return "开始时间和结束时间必填";
	const s = dayjs(start_at);
	const e = dayjs(end_at);
	if (!s.isValid() || !e.isValid()) return "开始时间和结束时间必填";
	if (!e.isAfter(s)) return "结束时间必须晚于开始时间";
	const days = e.diff(s, "day", true);
	if (days > MAX_WINDOW_DAYS) return `时间窗不能超过 ${MAX_WINDOW_DAYS} 天`;
	return null;
}

function extraStats(
	res: BatchLookupResponse<AssetCostItem>,
): StatEntry[] {
	const s = res.stats;
	return [
		{ label: "总花销", value: formatUsd(s.total_cost_usd ?? 0) },
		{ label: "均值", value: formatUsd(s.mean_cost_usd ?? 0) },
		{ label: "P50", value: formatUsd(s.p50_cost_usd ?? 0) },
		{ label: "P90", value: formatUsd(s.p90_cost_usd ?? 0) },
		{
			label: "总 GPU-min",
			value: Math.round((s.total_gpu_sec ?? 0) / 60).toString(),
		},
		{
			label: "总 CPU-min",
			value: Math.round((s.total_cpu_sec ?? 0) / 60).toString(),
		},
		{ label: "总运行次数", value: (s.total_run_count ?? 0).toString() },
	];
}

async function fetchCosts(
	req: BatchLookupRequest<CostsFilters>,
): Promise<BatchLookupResponse<AssetCostItem>> {
	if (!req.filters.start_at || !req.filters.end_at) {
		// Shell shouldn't fire when validate returns non-null; belt-and-braces.
		throw new Error("start_at 和 end_at 必填");
	}
	const raw = await assetsApi.lookupCosts({
		ids: req.ids,
		id_type: req.id_type,
		start_at: req.filters.start_at,
		end_at: req.filters.end_at,
		group_by: req.filters.group_by,
	});
	return toResponseEnvelope(raw);
}

// Build filename token from an ISO date. Falls back to "unknown" so the CSV
// button never crashes on a malformed filter (validate ought to catch it
// first).
function dateToken(iso: string | undefined): string {
	if (!iso) return "unknown";
	const d = dayjs(iso);
	return d.isValid() ? d.format("YYYY-MM-DD") : "unknown";
}

export const costsPreset: BatchLookupPreset<AssetCostItem, CostsFilters> = {
	key: "costs",
	placeholder: "aaaaaaaa\nbbbbbbbb\n019f8319-0ef3-7da0-815c-30f23d21e7f1",
	maxIds: MAX_IDS,
	fetch: fetchCosts,
	filters: {
		initial: {
			start_at: dayjs().subtract(7, "day").toISOString(),
			end_at: dayjs().toISOString(),
			group_by: "asset",
		},
		render: renderFilters,
		validate: validateCostsFilters,
	},
	// Column set depends on group_by — asset_algo mode adds an algo_key
	// column between grace_video_id and total_cost_usd. Shell resolves
	// this on every render via typeof-function check.
	columns: (filters: CostsFilters) => buildColumns(filters.group_by),
	extraStats,
	histogram: bucketCostItems,
	csv: {
		filename: (matchedCount: number, filters?: CostsFilters) => {
			const start = dateToken(filters?.start_at);
			const end = dateToken(filters?.end_at);
			return `asset-costs-${start}-${end}-${matchedCount}.csv`;
		},
		header: [
			"input_id",
			"asset_id",
			"grace_video_id",
			"algo_key",
			"total_cost_usd",
			"gpu_min",
			"cpu_min",
			"run_count",
		],
		row: (it) => [
			it.input_id,
			it.asset_id,
			it.grace_video_id ?? "",
			// When by_algo is present, emit a "|"-joined list of algo keys so
			// the CSV row still fits one line per asset. asset-only mode
			// leaves this empty.
			(it.by_algo ?? []).map((a: AssetCostByAlgo) => a.algo_key).join("|"),
			it.total_cost_usd,
			(it.gpu_sec / 60).toFixed(2),
			(it.cpu_sec / 60).toFixed(2),
			it.run_count,
		],
	},
};

// Backward-compat wrapper for unit tests that exercise the CSV helper
// directly without mounting the shell — matches the exportDurationsCsv
// pattern shipped in CYB-4294.
export function exportCostsCsv(
	items: AssetCostItem[],
	matchedCount: number,
	filters?: CostsFilters,
): void {
	downloadCsv(items, matchedCount, costsPreset.csv, filters);
}
