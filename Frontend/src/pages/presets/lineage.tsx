// CYB-4305: lineage preset for the shared `BatchAssetLookup` shell.
//
// Structure mirrors the durations / costs presets. Adds a depth Segmented
// (1 vs "all") that switches the column set (7 → 10 cols) and toggles the
// histogram card (categorical relation_types bars only for depth="all").

import { NodeIndexOutlined, PartitionOutlined } from "@ant-design/icons";
import { Col, Form, Segmented } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { ReactNode } from "react";
import type {
	AssetLineageBatchResponse,
	LineageBatchItem,
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

// LineageFilters is the preset's local filter shape. depth is either the
// number 1 or the string "all"; the server also accepts "1" as a string but
// the frontend only emits the two useful values.
export interface LineageFilters {
	depth: 1 | "all";
}

// Adapter: extract the frontend-shape response envelope from the raw API
// response. Stats has an optional `relation_type_counts` map that we drop
// into the numeric bag alongside the six counters.
function toResponseEnvelope(
	raw: AssetLineageBatchResponse,
): BatchLookupResponse<LineageBatchItem> {
	// The shell's `stats` bag is Record<string, number>; the relation-type
	// counts map lives in `relation_type_counts` on the raw response and is
	// consumed directly by the histogram (which reads response.items). Copy
	// the six scalar counters through; relation_type_counts is preserved on
	// the raw items[i].relation_types (for depth=all) so the histogram can
	// aggregate without pulling it off stats.
	const {
		matched_count,
		missing_count,
		has_parent_count,
		is_root_count,
		orphan_count,
		is_current_count,
	} = raw.stats;
	return {
		items: raw.items,
		missing_ids: raw.missing_ids,
		filtered_out_ids: raw.filtered_out_ids,
		stats: {
			matched_count,
			missing_count,
			has_parent_count,
			is_root_count,
			orphan_count,
			is_current_count,
		},
	};
}

// buildColumns switches between the depth=1 (7 cols) and depth=all (10 cols)
// layouts. Nulls render as `—`.
export function buildColumns(depth: 1 | "all"): ColumnsType<LineageBatchItem> {
	const em = "—";
	const base: ColumnsType<LineageBatchItem> = [
		{
			title: "input_id",
			dataIndex: "input_id",
			key: "input_id",
			defaultSortOrder: "ascend",
			sorter: (a, b) => a.input_id.localeCompare(b.input_id),
		},
		{ title: "asset_id", dataIndex: "asset_id", key: "asset_id" },
		{
			title: "grace_video_id",
			dataIndex: "grace_video_id",
			key: "grace_video_id",
			render: (v?: string) => v ?? em,
		},
		{
			title: "parent_asset_id",
			dataIndex: "parent_asset_id",
			key: "parent_asset_id",
			render: (v: string | null) => v ?? em,
		},
		{
			title: "root_asset_id",
			dataIndex: "root_asset_id",
			key: "root_asset_id",
			render: (v: string | null) => v ?? em,
		},
		{
			title: "is_current",
			dataIndex: "is_current",
			key: "is_current",
			render: (v: boolean) => (v ? "yes" : "no"),
		},
		{
			title: "revision",
			dataIndex: "revision",
			key: "revision",
			align: "right",
		},
	];
	if (depth === 1) {
		return base;
	}
	// depth="all" adds three columns off the ES projection. Counts render as
	// integer lengths; relation_types renders as a comma-joined string.
	return [
		...base,
		{
			title: "upstream_count",
			key: "upstream_count",
			align: "right",
			render: (_, r) => (r.upstream_ids?.length ?? 0).toString(),
		},
		{
			title: "downstream_count",
			key: "downstream_count",
			align: "right",
			render: (_, r) => (r.downstream_ids?.length ?? 0).toString(),
		},
		{
			title: "relation_types",
			key: "relation_types",
			render: (_, r) => (r.relation_types?.length ? r.relation_types.join(", ") : em),
		},
	];
}

// bucketRelationTypes rolls each item's `relation_types` slice into a single
// categorical bar chart. depth=1 items don't carry the field → this returns
// undefined so the shell suppresses the histogram card entirely.
export function bucketRelationTypes(
	items: LineageBatchItem[],
	filters?: LineageFilters,
): BucketRow[] | undefined {
	// Not depth=all → skip.
	if (!filters || filters.depth !== "all") return undefined;
	const counts = new Map<string, number>();
	for (const it of items) {
		for (const rt of it.relation_types ?? []) {
			counts.set(rt, (counts.get(rt) ?? 0) + 1);
		}
	}
	// Sort alphabetically so the bar order is deterministic across queries.
	return Array.from(counts.entries())
		.sort(([a], [b]) => a.localeCompare(b))
		.map(([label, count]) => ({ label, count }));
}

function renderFilters(
	filters: LineageFilters,
	setFilters: (next: LineageFilters) => void,
): ReactNode {
	return (
		<Col span={12}>
			<Form.Item label="深度">
				<Segmented<1 | "all">
					value={filters.depth}
					onChange={(v) => setFilters({ ...filters, depth: v })}
					options={[
						{
							value: 1,
							label: (
								<span>
									<NodeIndexOutlined /> 直接父/根
								</span>
							),
						},
						{
							value: "all",
							label: (
								<span>
									<PartitionOutlined /> 全链
								</span>
							),
						},
					]}
					data-testid="lineage-depth"
				/>
			</Form.Item>
		</Col>
	);
}

function extraStats(
	res: BatchLookupResponse<LineageBatchItem>,
): StatEntry[] {
	const s = res.stats;
	return [
		{ label: "已匹配", value: s.matched_count ?? 0 },
		{ label: "有 parent 的", value: s.has_parent_count ?? 0 },
		{ label: "是 root 的", value: s.is_root_count ?? 0 },
		{ label: "孤儿", value: s.orphan_count ?? 0 },
		{ label: "当前版本", value: s.is_current_count ?? 0 },
	];
}

async function fetchLineage(
	req: BatchLookupRequest<LineageFilters>,
): Promise<BatchLookupResponse<LineageBatchItem>> {
	const raw = await assetsApi.lookupLineage({
		ids: req.ids,
		id_type: req.id_type,
		depth: req.filters.depth,
	});
	return toResponseEnvelope(raw);
}

// depthToken returns the filename-safe form of the current depth: `1` or
// `All` (capitalized to hint at the switch when eyeballing filenames).
function depthToken(depth: 1 | "all"): string {
	return depth === "all" ? "All" : "1";
}

export const lineagePreset: BatchLookupPreset<
	LineageBatchItem,
	LineageFilters
> = {
	key: "lineage",
	placeholder: "aaaaaaaa\nbbbbbbbb\n019f8319-0ef3-7da0-815c-30f23d21e7f1",
	maxIds: MAX_IDS,
	fetch: fetchLineage,
	filters: {
		initial: { depth: 1 },
		render: renderFilters,
	},
	// Column set switches on depth — the shell resolves the callback each
	// render (see CYB-4304 pattern used by costs.tsx).
	columns: (filters: LineageFilters) => buildColumns(filters.depth),
	extraStats,
	// Only render the histogram card for depth="all"; the shell drops the
	// card entirely when this returns undefined.
	histogram: bucketRelationTypes,
	csv: {
		filename: (matchedCount: number, filters?: LineageFilters) => {
			const d = filters ? depthToken(filters.depth) : "1";
			return `asset-lineage-depth${d}-${matchedCount}.csv`;
		},
		header: [
			"input_id",
			"asset_id",
			"grace_video_id",
			"parent_asset_id",
			"root_asset_id",
			"logical_asset_id",
			"is_current",
			"revision",
			"upstream_ids",
			"downstream_ids",
			"relation_types",
		],
		row: (it) => [
			it.input_id,
			it.asset_id,
			it.grace_video_id ?? "",
			it.parent_asset_id ?? "",
			it.root_asset_id ?? "",
			it.logical_asset_id ?? "",
			it.is_current ? "true" : "false",
			it.revision,
			// Multi-value fields join with `|` per the CYB-4306 convention
			// (asset_algo's algo_keys use the same separator).
			(it.upstream_ids ?? []).join("|"),
			(it.downstream_ids ?? []).join("|"),
			(it.relation_types ?? []).join("|"),
		],
	},
};

// Backward-compat wrapper so unit tests can exercise the CSV helper without
// mounting the shell, matching the pattern shipped by durations / costs.
export function exportLineageCsv(
	items: LineageBatchItem[],
	matchedCount: number,
	filters?: LineageFilters,
): void {
	downloadCsv(items, matchedCount, lineagePreset.csv, filters);
}
