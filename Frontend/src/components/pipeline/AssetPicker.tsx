import { Input, Table, message } from "antd";
import { useState } from "react";
import { searchApi } from "../../api/search";
import type { SearchAssetResult } from "../../api/search";

interface AssetPickerProps {
  /** Currently selected asset IDs (controlled) */
  selectedIds: string[];
  /** Called when selection changes */
  onSelectionChange: (ids: string[]) => void;
  /** Placeholder text for the search input */
  placeholder?: string;
  /** Table max height for scroll */
  maxHeight?: number;
}

/** Shared asset search + multi-select table.
 *  Wraps no Modal or Collapse — parent controls the container.
 */
export default function AssetPicker({
  selectedIds,
  onSelectionChange,
  placeholder = "搜索资产（输入 asset_id 或名称）",
  maxHeight = 200,
}: AssetPickerProps) {
  const [results, setResults] = useState<SearchAssetResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState("");

  const handleSearch = async (value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return;
    setQuery(trimmed);
    setLoading(true);
    try {
      const res = await searchApi.searchAssets({ q: trimmed, page_size: 50 });
      setResults(res.items);
    } catch {
      message.error("搜索资产失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <Input.Search
        placeholder={placeholder}
        onSearch={handleSearch}
        loading={loading}
        size="small"
      />
      {results.length > 0 ? (
        <Table
          rowKey="asset_id"
          dataSource={results}
          size="small"
          pagination={false}
          scroll={{ y: maxHeight }}
          rowSelection={{
            type: "checkbox",
            selectedRowKeys: selectedIds,
            onChange: (keys) => onSelectionChange(keys as string[]),
          }}
          columns={[
            { title: "Asset ID", dataIndex: "asset_id", width: 120 },
            { title: "类型", dataIndex: "asset_type", width: 70 },
            { title: "状态", dataIndex: "lifecycle_state", width: 70 },
            {
              title: "存储路径",
              dataIndex: "storage_uri",
              width: 180,
              render: (v: string | undefined) =>
                v ? (
                  <span
                    style={{
                      fontSize: 11,
                      fontFamily: '"SF Mono", monospace',
                      color: "#64748b",
                    }}
                  >
                    {v.length > 36 ? v.slice(0, 36) + "\u2026" : v}
                  </span>
                ) : (
                  <span style={{ fontSize: 11, color: "#aaa" }}>—</span>
                ),
            },
          ]}
        />
      ) : (
        <div style={{ color: "#999", textAlign: "center", padding: 12, fontSize: 12 }}>
          {query ? "未找到匹配的资产" : "输入关键字搜索资产，不选择则直接部署"}
        </div>
      )}
    </div>
  );
}
