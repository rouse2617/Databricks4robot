// ─── AssetsResultsPane — Section Component ───
// Toolbar + Ant Design Table + pagination for the assets discovery workbench.
// Validates: Requirements R6, R1

import { CopyOutlined, DownloadOutlined } from "@ant-design/icons";
import { Button, message, Space, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type { Asset } from "../../api/types";
import {
	formatDurationSeconds,
	getAssetStateColor,
	getAssetType,
	getLifecycleState,
} from "../../lib/assetPresentation";
import type {
	FetchStatus,
	ViewMode,
} from "../../lib/assets/assetsDiscoveryTypes";
import { formatDateTime } from "../../lib/dateTime";
import { withSelectAllColumn } from "../../lib/tableSelection";
import AlgoSummaryCell from "./AlgoSummaryCell";
import AssetsCardView from "./AssetsCardView";
import ResultsEmptyState from "./ResultsEmptyState";

const { Text } = Typography;
// ─── Status Color Map ───

// ─── Column Definitions ───

type ColumnBuilder = (
	onRowClick: (assetId: string) => void,
	onMcapClick?: (mcapFileId: string) => void,
	sort?: string,
	onSortChange?: (sort: string) => void,
) => ColumnsType<Asset>[number];

async function copyToClipboard(value: string, label: string) {
	try {
		await navigator.clipboard.writeText(value);
		message.success(`已复制${label}`);
	} catch {
		message.error("复制失败");
	}
}

function formatShortId(value: string, keep = 8): string {
	return value.length > keep ? `${value.slice(0, keep)}…` : value;
}

function renderSortableTitle(
	label: string,
	sortField: string,
	sort: string | undefined,
	onSortChange: ((sort: string) => void) | undefined,
	defaultDirection: "asc" | "desc",
) {
	const asc = sortField;
	const desc = `-${sortField}`;
	const isActive = sort === desc || sort === asc;
	const activeDirection = sort === desc ? "desc" : "asc";
	const indicator = !isActive ? "↕" : activeDirection === "desc" ? "↓" : "↑";

	return (
		<button
			type="button"
			className="link-like-button"
			aria-label={`按${label}排序`}
			onClick={() => {
				if (!onSortChange) return;
				if (!isActive) {
					onSortChange(defaultDirection === "desc" ? desc : asc);
					return;
				}
				onSortChange(activeDirection === "desc" ? asc : desc);
			}}
			style={{ fontWeight: 600, color: "#111827", cursor: "pointer" }}
		>
			{label} {indicator}
		</button>
	);
}

function describeSort(sort: string): string {
	const isDesc = sort.startsWith("-");
	const field = isDesc ? sort.slice(1) : sort;
	const labelMap: Record<string, string> = {
		updated_at: "更新时间",
		duration_ms: "时长",
	};
	const fieldLabel = labelMap[field] ?? field;
	return `${fieldLabel} ${isDesc ? "↓" : "↑"}`;
}

const COLUMN_BUILDERS: Record<string, ColumnBuilder> = {
	asset_id: (onRowClick) => ({
		title: "Asset ID",
		dataIndex: "asset_id",
		width: 120,
		fixed: "left" as const,
		render: (id: string) => (
			<Space size={4}>
				<Tooltip title="点击打开预览">
					<button
						type="button"
						className="font-mono text-xs cursor-pointer link-like-button"
						aria-label={`Asset ${id}`}
						data-testid={`asset-id-link-${id}`}
						onClick={(e) => {
							e.stopPropagation();
							onRowClick(id);
						}}
					>
						{formatShortId(id)}
					</button>
				</Tooltip>
				<Tooltip title="复制 Asset ID">
					<button
						type="button"
						className="link-like-button"
						aria-label={`复制 Asset ${id}`}
						onClick={(e) => {
							e.stopPropagation();
							void copyToClipboard(id, " Asset ID");
						}}
					>
						<CopyOutlined style={{ fontSize: 12 }} />
					</button>
				</Tooltip>
			</Space>
		),
	}),
	mcap_file_id: (_onRowClick, onMcapClick) => ({
		title: "MCAP",
		dataIndex: "mcap_file_id",
		width: 100,
		render: (v: string) => (
			<Space size={4}>
				<Tooltip title={v || "—"}>
					<button
						type="button"
						className="font-mono text-xs cursor-pointer link-like-button"
						aria-label={`MCAP ${v}`}
						onClick={(e) => {
							e.stopPropagation();
							if (!v) return;
							onMcapClick?.(v);
						}}
					>
						{v ? formatShortId(v) : "—"}
					</button>
				</Tooltip>
				{v && (
					<Tooltip title="复制 MCAP ID">
						<button
							type="button"
							className="link-like-button"
							aria-label={`复制 MCAP ${v}`}
							onClick={(e) => {
								e.stopPropagation();
								void copyToClipboard(v, " MCAP ID");
							}}
						>
							<CopyOutlined style={{ fontSize: 12 }} />
						</button>
					</Tooltip>
				)}
			</Space>
		),
	}),
	duration: (_onRowClick, _onMcapClick, sort, onSortChange) => ({
		title: renderSortableTitle(
			"时长",
			"duration_ms",
			sort,
			onSortChange,
			"asc",
		),
		key: "duration",
		width: 70,
		render: (_: unknown, r: Asset) => formatDurationSeconds(r),
	}),
	asset_type: () => ({
		title: "资产类型",
		key: "asset_type",
		width: 100,
		render: (_: unknown, r: Asset) => getAssetType(r) || "—",
	}),
	retention_tier: () => ({
		title: "保留层级",
		key: "retention_tier",
		width: 100,
		render: (_: unknown, r: Asset) => {
			const tier = r.retention_tier;
			if (!tier) return "—";
			const colorMap: Record<string, string> = {
				standard: "blue",
				archive: "orange",
				cold: "default",
			};
			return <Tag color={colorMap[tier] ?? "default"}>{tier}</Tag>;
		},
	}),
	env: () => ({
		title: "环境",
		key: "env",
		width: 80,
		render: (_: unknown, r: Asset) => r.env ?? "—",
	}),
	lifecycle_state: () => ({
		title: "生命周期",
		width: 90,
		render: (_: unknown, r: Asset) => (
			<Tag color={getAssetStateColor(r)}>{getLifecycleState(r) || "—"}</Tag>
		),
	}),
	algo: () => ({
		title: "算法状态",
		key: "algo",
		width: 200,
		render: (_: unknown, record: Asset) => <AlgoSummaryCell record={record} />,
	}),
	tags: () => ({
		title: "标签",
		key: "tags",
		width: 120,
		render: (_: unknown, r: Asset) => {
			if (!r.tags || Object.keys(r.tags).length === 0) return "—";
			return (
				<Space size={2} wrap>
					{Object.entries(r.tags)
						.slice(0, 3)
						.map(([k, v]) => (
							<Tag key={k} style={{ fontSize: 11 }}>
								{k}:{v}
							</Tag>
						))}
				</Space>
			);
		},
	}),
	owner: () => ({
		title: "Owner",
		dataIndex: "owner",
		width: 120,
		ellipsis: { showTitle: false },
		render: (v: string) => {
			if (!v) return "—";
			return (
				<Tooltip title={v} placement="topLeft">
					<span
						style={{
							display: "inline-block",
							maxWidth: "100%",
							overflow: "hidden",
							textOverflow: "ellipsis",
							whiteSpace: "nowrap",
						}}
					>
						{v}
					</span>
				</Tooltip>
			);
		},
	}),
	expire_at: () => ({
		title: "过期时间",
		dataIndex: "expire_at",
		width: 130,
		render: (v: string | null | undefined) => (v ? dayjs(v).fromNow() : "—"),
	}),
	updated_at: (_onRowClick, _onMcapClick, sort, onSortChange) => ({
		title: renderSortableTitle(
			"更新时间",
			"updated_at",
			sort,
			onSortChange,
			"desc",
		),
		dataIndex: "updated_at",
		width: 180,
		render: (v: string) => formatDateTime(v),
	}),
};

// ─── Build Columns from selectedColumns ───

function buildColumns(
	selectedColumns: string[],
	onRowClick: (assetId: string) => void,
	onMcapClick?: (mcapFileId: string) => void,
	sort?: string,
	onSortChange?: (sort: string) => void,
): ColumnsType<Asset> {
	return selectedColumns
		.filter((col) => COLUMN_BUILDERS[col])
		.map((col) =>
			COLUMN_BUILDERS[col](onRowClick, onMcapClick, sort, onSortChange),
		);
}

// ─── Props ───

export interface AssetsResultsPaneProps {
	items: Asset[];
	total: number;
	totalApprox: boolean;
	fetchStatus: FetchStatus;
	error?: string | null;
	activeFilterCount?: number;
	hasActiveQueryText?: boolean;
	sort: string;
	page: number;
	pageSize: number;
	viewMode: ViewMode;
	selectedColumns: string[];
	selectedIds: Set<string>;
	activePreviewId: string | null;
	onSortChange: (sort: string) => void;
	onPageChange: (page: number, pageSize: number) => void;
	onSelectRow: (id: string) => void;
	onRowClick: (assetId: string) => void;
	onMcapClick?: (mcapFileId: string) => void;
	onClearFilters?: () => void;
	onRetry?: () => void;
	onExport?: () => void;
	columnsConfigSlot?: React.ReactNode;
}

// ─── Component ───

export default function AssetsResultsPane({
	items,
	total,
	totalApprox,
	fetchStatus,
	error,
	activeFilterCount = 0,
	hasActiveQueryText = false,
	sort,
	page,
	pageSize,
	viewMode,
	selectedColumns,
	selectedIds,
	activePreviewId,
	onSortChange,
	onPageChange,
	onSelectRow,
	onRowClick,
	onMcapClick,
	onClearFilters,
	onRetry,
	onExport,
	columnsConfigSlot,
}: AssetsResultsPaneProps) {
	const loading = fetchStatus === "loading";
	const columns = buildColumns(
		selectedColumns,
		onRowClick,
		onMcapClick,
		sort,
		onSortChange,
	);
	const selectedRowKeys = Array.from(selectedIds);

	// ── Card view ──
	if (viewMode === "card") {
		return (
			<AssetsCardView
				items={items}
				total={total}
				totalApprox={totalApprox}
				fetchStatus={fetchStatus}
				error={error}
				activeFilterCount={activeFilterCount}
				hasActiveQueryText={hasActiveQueryText}
				sort={sort}
				page={page}
				pageSize={pageSize}
				selectedIds={selectedIds}
				onSelectRow={onSelectRow}
				onRowClick={onRowClick}
				onMcapClick={onMcapClick}
				onSortChange={onSortChange}
				onPageChange={onPageChange}
				onClearFilters={onClearFilters}
				onRetry={onRetry}
			/>
		);
	}

	// ── Empty / Error states ──
	if (fetchStatus === "error") {
		return (
			<div style={{ flex: 1, minWidth: 0 }}>
				<ResultsEmptyState variant="error" error={error} onRetry={onRetry} />
			</div>
		);
	}

	if (fetchStatus === "success" && items.length === 0) {
		const variant =
			activeFilterCount > 0 || hasActiveQueryText ? "no_results" : "initial";
		return (
			<div style={{ flex: 1, minWidth: 0 }}>
				<ResultsEmptyState
					variant={variant}
					activeFilterCount={activeFilterCount + (hasActiveQueryText ? 1 : 0)}
					onClearFilters={onClearFilters}
				/>
			</div>
		);
	}

	return (
		<div style={{ flex: 1, minWidth: 0 }}>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					marginBottom: 8,
				}}
			>
				{/* Results Count */}
				<Text
					type="secondary"
					style={{ fontSize: 13, fontVariantNumeric: "tabular-nums" }}
				>
					共 {total}
					{totalApprox ? "+" : ""} 条
				</Text>
				<Text type="secondary" style={{ fontSize: 12 }}>
					当前排序：{describeSort(sort)}
				</Text>

				<Space size={8}>
					{/* Export Button (7.2) */}
					{onExport && (
						<Button size="small" icon={<DownloadOutlined />} onClick={onExport}>
							导出
						</Button>
					)}

					{/* Columns Config Slot */}
					{columnsConfigSlot}
				</Space>
			</div>

			{/* Table */}
			<Table
				rowKey="asset_id"
				columns={columns}
				dataSource={items}
				loading={loading}
				size="small"
				scroll={{ x: 900 }}
				rowClassName={(record, index) => {
					if (record.asset_id === activePreviewId)
						return "assets-active-preview-row";
					return index % 2 === 1 ? "assets-zebra-row" : "";
				}}
				onRow={(record) => ({
					onClick: () => onRowClick(record.asset_id),
					onKeyDown: (e) => {
						if (e.key === "Enter" || e.key === " ") {
							e.preventDefault();
							onRowClick(record.asset_id);
						}
					},
					style: { cursor: "pointer" },
					"aria-selected": selectedIds.has(record.asset_id),
					tabIndex: 0,
				})}
				rowSelection={withSelectAllColumn<Asset>({
					selectedRowKeys,
					onChange: (keys) => {
						const newKeys = new Set(keys as string[]);
						// Toggle added keys
						for (const k of newKeys) {
							if (!selectedIds.has(k)) onSelectRow(k);
						}
						// Toggle removed keys
						for (const k of selectedIds) {
							if (!newKeys.has(k)) onSelectRow(k);
						}
					},
				})}
				pagination={{
					current: page,
					total,
					pageSize,
					showSizeChanger: true,
					showQuickJumper: total > 200,
					pageSizeOptions: ["20", "50", "100", "200"],
					showTotal: (t) => `共 ${t}${totalApprox ? "+" : ""} 条`,
					onChange: (p, ps) => onPageChange(p, ps),
					onShowSizeChange: (p, ps) => onPageChange(p, ps),
				}}
			/>

			{/* Active preview row highlight + zebra + hover style */}
			<style>{`
        .assets-zebra-row > td {
          background-color: #FAFBFC;
        }
        .ant-table-tbody > tr.assets-zebra-row:hover > td,
        .ant-table-tbody > tr:hover > td {
          background-color: #EFF6FF !important;
        }
        .assets-active-preview-row > td,
        .ant-table-tbody > tr.assets-active-preview-row:hover > td {
          background-color: #DBEAFE !important;
        }
      `}</style>
		</div>
	);
}
