// ─── AssetsResultsPane — Section Component ───
// Toolbar + Ant Design Table + pagination for the assets discovery workbench.
// Validates: Requirements R6, R1

import { Table, Select, Button, Typography, Space, Tag, Tooltip } from "antd";
import { DownloadOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type { Asset } from "../../api/types";
import type { FetchStatus, ViewMode } from "../../lib/assets/assetsDiscoveryTypes";
import AlgoSummaryCell from "./AlgoSummaryCell";
import ResultsEmptyState from "./ResultsEmptyState";
import { formatDurationSeconds, getAssetStateColor, getLifecycleState } from "../../lib/assetPresentation";

const { Text } = Typography;

// ─── Status Color Map ───

// ─── Sort Options ───

const SORT_OPTIONS = [
  { value: "-updated_at", label: "更新时间 ↓" },
  { value: "-created_at", label: "创建时间 ↓" },
  { value: "duration_ms", label: "时长 ↑" },
];

// ─── Column Definitions ───

type ColumnBuilder = (onRowClick: (assetId: string) => void) => ColumnsType<Asset>[number];

const COLUMN_BUILDERS: Record<string, ColumnBuilder> = {
  asset_id: (onRowClick) => ({
    title: "Asset ID",
    dataIndex: "asset_id",
    width: 120,
    fixed: "left" as const,
    render: (id: string) => (
      <Tooltip title={id}>
        <a
          className="font-mono text-xs cursor-pointer"
          onClick={(e) => {
            e.stopPropagation();
            onRowClick(id);
          }}
        >
          {id.slice(0, 8)}…
        </a>
      </Tooltip>
    ),
  }),
  mcap_file_id: () => ({
    title: "MCAP",
    dataIndex: "mcap_file_id",
    width: 100,
    render: (v: string) => (
      <Tooltip title={v}>
        <span className="font-mono text-xs">{v?.slice(0, 8)}…</span>
      </Tooltip>
    ),
  }),
  duration: () => ({
    title: "时长",
    key: "duration",
    width: 70,
    render: (_: unknown, r: Asset) => formatDurationSeconds(r),
  }),
  env: () => ({
    title: "环境",
    key: "env",
    width: 80,
    render: (_: unknown, r: Asset) => r.env ?? "—",
  }),
  status: () => ({
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
  updated_at: () => ({
    title: "更新时间",
    dataIndex: "updated_at",
    width: 140,
    render: (v: string) => dayjs(v).format("MM-DD HH:mm"),
  }),
};

// ─── Build Columns from selectedColumns ───

function buildColumns(
  selectedColumns: string[],
  onRowClick: (assetId: string) => void,
): ColumnsType<Asset> {
  return selectedColumns
    .filter((col) => COLUMN_BUILDERS[col])
    .map((col) => COLUMN_BUILDERS[col](onRowClick));
}

// ─── Props ───

export interface AssetsResultsPaneProps {
  items: Asset[];
  total: number;
  totalApprox: boolean;
  fetchStatus: FetchStatus;
  error?: string | null;
  activeFilterCount?: number;
  sort: string;
  page: number;
  pageSize: number;
  viewMode: ViewMode;
  selectedColumns: string[];
  selectedIds: Set<string>;
  activePreviewId: string | null;
  onSortChange: (sort: string) => void;
  onPageChange: (page: number) => void;
  onSelectRow: (id: string) => void;
  onRowClick: (assetId: string) => void;
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
  sort,
  page,
  pageSize,
  selectedColumns,
  selectedIds,
  activePreviewId,
  onSortChange,
  onPageChange,
  onSelectRow,
  onRowClick,
  onClearFilters,
  onRetry,
  onExport,
  columnsConfigSlot,
}: AssetsResultsPaneProps) {
  const loading = fetchStatus === "loading";
  const columns = buildColumns(selectedColumns, onRowClick);
  const selectedRowKeys = Array.from(selectedIds);

  // ── Empty / Error states ──
  if (fetchStatus === "error") {
    return (
      <div style={{ flex: 1, minWidth: 0 }}>
        <ResultsEmptyState variant="error" error={error} onRetry={onRetry} />
      </div>
    );
  }

  if (fetchStatus === "success" && items.length === 0) {
    const variant = activeFilterCount > 0 ? "no_results" : "initial";
    return (
      <div style={{ flex: 1, minWidth: 0 }}>
        <ResultsEmptyState
          variant={variant}
          activeFilterCount={activeFilterCount}
          onClearFilters={onClearFilters}
        />
      </div>
    );
  }

  return (
    <div style={{ flex: 1, minWidth: 0 }}>
      {/* Results Toolbar */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 8,
        }}
      >
        {/* Results Count */}
        <Text type="secondary" style={{ fontSize: 13 }}>
          共 {total}{totalApprox ? "+" : ""} 条
        </Text>

        <Space size={8}>
          {/* Export Button (7.2) */}
          {onExport && (
            <Button
              size="small"
              icon={<DownloadOutlined />}
              onClick={onExport}
            >
              导出
            </Button>
          )}

          {/* Sort Selector */}
          <Select
            value={sort}
            onChange={onSortChange}
            options={SORT_OPTIONS}
            size="small"
            style={{ width: 140 }}
          />

          {/* Columns Config Slot */}
          {columnsConfigSlot}

          {/* ViewMode toggle placeholder */}
          <Text type="secondary" style={{ fontSize: 12 }}>
            {/* ViewMode toggle — Phase 2 */}
          </Text>
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
        rowClassName={(record) =>
          record.asset_id === activePreviewId ? "assets-active-preview-row" : ""
        }
        onRow={(record) => ({
          onClick: () => onRowClick(record.asset_id),
          style: { cursor: "pointer" },
          "aria-selected": selectedIds.has(record.asset_id),
        })}
        rowSelection={{
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
        }}
        pagination={{
          current: page,
          total,
          pageSize,
          showSizeChanger: false,
          showTotal: (t) => `共 ${t}${totalApprox ? "+" : ""} 条`,
          onChange: (p) => onPageChange(p),
        }}
      />

      {/* Active preview row highlight style */}
      <style>{`
        .assets-active-preview-row {
          background-color: #e6f4ff !important;
        }
        .assets-active-preview-row td {
          background-color: #e6f4ff !important;
        }
      `}</style>
    </div>
  );
}
