import { useEffect, useState, useCallback } from "react";
import { Table, Tag, Typography, Button, Space, Select, Empty } from "antd";
import { ReloadOutlined, FileOutlined, CloudUploadOutlined } from "@ant-design/icons";
import { useSearchParams } from "react-router-dom";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { mcapFilesApi } from "../api/mcapFiles";
import type { McapFile } from "../api/types";

const { Title, Text } = Typography;

const ingestColor: Record<string, string> = {
  pending: "default",
  summarized: "success",
  failed: "error",
};

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

const stateFilterOptions = [
  { value: "", label: "全部状态" },
  { value: "pending", label: "pending" },
  { value: "summarized", label: "summarized" },
  { value: "failed", label: "failed" },
];

export default function McapFilesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [files, setFiles] = useState<McapFile[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(Number(searchParams.get("page")) || 1);
  const [stateFilter, setStateFilter] = useState("");

  const load = useCallback(async (p = page) => {
    setLoading(true);
    try {
      const data = await mcapFilesApi.list({ page: p, page_size: 20 });
      let items = data.items ?? [];
      // P2 #13: Client-side ingest_state filter
      if (stateFilter) {
        items = items.filter((f) => f.ingest_state === stateFilter);
      }
      setFiles(items);
      setTotal(data.total ?? 0);
    } catch {
      setFiles([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, stateFilter]);

  useEffect(() => {
    load(page);
  }, [page, stateFilter]);

  const columns: ColumnsType<McapFile> = [
    {
      title: "MCAP File ID",
      dataIndex: "mcap_file_id",
      width: 140,
      render: (id: string) => (
        <span className="font-mono text-xs">{id?.slice(0, 12)}…</span>
      ),
    },
    {
      title: "GCS Path",
      dataIndex: "gcs_path",
      ellipsis: true,
      responsive: ["md"] as any,
      render: (v: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>{v || "—"}</Text>
      ),
    },
    {
      title: "大小",
      dataIndex: "size_bytes",
      width: 100,
      render: (v: number) => formatBytes(v),
    },
    {
      title: "状态",
      dataIndex: "ingest_state",
      width: 100,
      render: (s: string) => (
        <Tag color={ingestColor[s] ?? "default"}>{s || "—"}</Tag>
      ),
    },
    {
      title: "Channels",
      dataIndex: "channel_count",
      width: 90,
      responsive: ["lg"] as any,
      render: (v: number) => v || "—",
    },
    {
      title: "Chunks",
      dataIndex: "chunk_count",
      width: 80,
      responsive: ["lg"] as any,
      render: (v: number) => v || "—",
    },
    {
      title: "Owner",
      dataIndex: "owner",
      width: 100,
      render: (v: string) => v || "—",
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      width: 140,
      render: (v: string) => v ? dayjs(v).format("MM-DD HH:mm") : "—",
    },
  ];

  // P1 #7: Empty state with guidance
  const emptyState = (
    <Empty
      image={<CloudUploadOutlined style={{ fontSize: 48, color: "#94A3B8" }} />}
      description={
        <div>
          <Text type="secondary">暂无 MCAP 文件记录</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            通过 SDK 上传 MCAP 文件或调用 POST /api/v1/mcap/upload/finalize 创建记录
          </Text>
        </div>
      }
    />
  );

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <Title level={4} style={{ margin: 0 }}>
          <FileOutlined style={{ marginRight: 8 }} />
          MCAP 文件
          <Text type="secondary" style={{ fontSize: 14, fontWeight: 400, marginLeft: 8 }}>
            ({total})
          </Text>
        </Title>
        <Space>
          <Select
            value={stateFilter}
            onChange={(v) => { setStateFilter(v); setPage(1); }}
            options={stateFilterOptions}
            style={{ width: 130 }}
            size="middle"
          />
          <Button icon={<ReloadOutlined />} onClick={() => load(page)}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        rowKey="mcap_file_id"
        columns={columns}
        dataSource={files}
        loading={loading}
        size="small"
        scroll={{ x: 800 }}
        locale={{ emptyText: emptyState }}
        pagination={{
          current: page,
          total,
          pageSize: 20,
          showSizeChanger: false,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p) => {
            setPage(p);
            setSearchParams({ page: String(p) });
          },
        }}
      />
    </div>
  );
}
