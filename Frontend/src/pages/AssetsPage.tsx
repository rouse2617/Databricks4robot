import { useEffect, useState } from "react";
import { Table, Tag, Button, Space, Input, Select, Typography, message } from "antd";
import { useNavigate } from "react-router-dom";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { assetsApi, type Asset } from "../api/assets";

const { Title } = Typography;
const { Search } = Input;

const statusColor: Record<string, string> = {
  active: "green",
  pending: "gold",
  archived: "blue",
  deleted: "red",
};

export default function AssetsPage() {
  const navigate = useNavigate();
  const [assets, setAssets] = useState<Asset[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<string>();
  const [msg, msgCtx] = message.useMessage();

  const load = async (p = page, s = status) => {
    setLoading(true);
    try {
      const data = await assetsApi.list({ page: p, page_size: 20, status: s });
      setAssets(data.items);
      setTotal(data.total);
    } catch {
      msg.error("Failed to load assets");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const columns: ColumnsType<Asset> = [
    {
      title: "Asset ID",
      dataIndex: "asset_id",
      render: (id: string) => (
        <Button type="link" onClick={() => navigate(`/assets/${id}`)} className="p-0 font-mono text-xs">
          {id.slice(0, 8)}…
        </Button>
      ),
    },
    { title: "MCAP File", dataIndex: "mcap_file_id", render: (v: string) => <span className="font-mono text-xs">{v.slice(0, 8)}…</span> },
    { title: "Duration (s)", dataIndex: "duration_sec", render: (v: number) => v.toFixed(2) },
    { title: "Reviewer", dataIndex: "reviewer" },
    {
      title: "Status",
      dataIndex: "status",
      render: (s: string) => <Tag color={statusColor[s] ?? "default"}>{s}</Tag>,
    },
    { title: "Owner", dataIndex: "owner" },
    { title: "Updated", dataIndex: "updated_at", render: (v: string) => dayjs(v).format("YYYY-MM-DD HH:mm") },
  ];

  return (
    <div>
      {msgCtx}
      <div className="flex justify-between items-center mb-4">
        <Title level={4} className="!mb-0">Assets</Title>
        <Space>
          <Select
            placeholder="Filter status"
            allowClear
            style={{ width: 140 }}
            onChange={(v) => { setStatus(v); load(1, v); setPage(1); }}
            options={["pending", "active", "archived", "deleted"].map((s) => ({ value: s, label: s }))}
          />
          <Search placeholder="Search tags…" onSearch={() => load(1)} style={{ width: 220 }} />
        </Space>
      </div>
      <Table
        rowKey="asset_id"
        columns={columns}
        dataSource={assets}
        loading={loading}
        pagination={{ current: page, total, pageSize: 20, onChange: (p) => { setPage(p); load(p); } }}
        size="middle"
      />
    </div>
  );
}
