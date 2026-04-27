import { Table } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Asset } from "../../api/types";
import type { AlgoRegistryItem } from "../../api/algoRegistry";
import AlgoStatusCell, { type CellStatus } from "./AlgoStatusCell";
import AlgoStatusPopover from "./AlgoStatusPopover";

interface AlgoMatrixGridProps {
  assets: Asset[];
  algorithms: AlgoRegistryItem[];
  loading: boolean;
  total: number;
  page: number;
  pageSize: number;
  onPageChange: (page: number, pageSize: number) => void;
  onRefresh: () => void;
}

/** Parse the algo_results map value into a CellStatus */
function parseStatus(raw?: string): CellStatus {
  if (!raw) return "none";
  // algo_results values can be JSON strings or plain status strings
  try {
    const parsed = JSON.parse(raw);
    if (typeof parsed === "object" && parsed.status) {
      return normalizeStatus(parsed.status);
    }
    return normalizeStatus(String(parsed));
  } catch {
    return normalizeStatus(raw);
  }
}

function normalizeStatus(s: string): CellStatus {
  const lower = s.toLowerCase();
  if (lower === "ok" || lower === "success") return "ok";
  if (lower === "failed" || lower === "error") return "failed";
  if (lower === "running") return "running";
  if (lower === "pending") return "pending";
  if (lower === "blocked") return "blocked";
  return "none";
}

/** Extract detail object from algo_results value */
function parseDetail(raw?: string) {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (typeof parsed === "object") return parsed;
    return { status: raw };
  } catch {
    return { status: raw };
  }
}

export default function AlgoMatrixGrid({
  assets,
  algorithms,
  loading,
  total,
  page,
  pageSize,
  onPageChange,
  onRefresh,
}: AlgoMatrixGridProps) {
  const columns: ColumnsType<Asset> = [
    {
      title: "Asset ID",
      dataIndex: "asset_id",
      key: "asset_id",
      width: 220,
      fixed: "left",
      ellipsis: true,
      render: (id: string) => (
        <a href={`/assets/${id}`} style={{ fontFamily: "monospace", fontSize: 12 }}>
          {id.slice(0, 12)}…
        </a>
      ),
    },
    ...algorithms.map((algo) => ({
      title: (
        <div style={{ textAlign: "center", fontSize: 12 }}>
          <div>{algo.name}</div>
          <div style={{ color: "#999", fontSize: 10 }}>{algo.version}</div>
        </div>
      ),
      key: algo.key,
      width: 80,
      align: "center" as const,
      render: (_: unknown, asset: Asset) => {
        const rawValue = asset.algo_results?.[algo.key];
        const status = parseStatus(rawValue);
        const detail = parseDetail(rawValue);
        return (
          <AlgoStatusPopover
            assetId={asset.asset_id}
            algoKey={algo.key}
            detail={detail}
            onReset={onRefresh}
          >
            <span>
              <AlgoStatusCell status={status} onClick={() => {}} />
            </span>
          </AlgoStatusPopover>
        );
      },
    })),
  ];

  return (
    <Table<Asset>
      columns={columns}
      dataSource={assets}
      rowKey="asset_id"
      loading={loading}
      scroll={{ x: 220 + algorithms.length * 80 }}
      size="small"
      pagination={{
        current: page,
        pageSize,
        total,
        showSizeChanger: true,
        pageSizeOptions: ["20", "50", "100"],
        showTotal: (t) => `共 ${t} 条`,
        onChange: onPageChange,
      }}
    />
  );
}
