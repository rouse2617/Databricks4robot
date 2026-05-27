import { useEffect, useState, useMemo } from "react";
import { Button, Table, Tag, Select } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { listWorkflows, type WorkflowSummary } from "../api/workflowApi";

const STATUS_COLORS: Record<string, string> = {
  Succeeded: "success",
  Running: "processing",
  Pending: "warning",
  Failed: "error",
  Error: "error",
};

export default function WorkflowListPage() {
  const [items, setItems] = useState<WorkflowSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const navigate = useNavigate();

  const refresh = async () => {
    setLoading(true);
    try {
      const res = await listWorkflows();
      setItems(res.items || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const filtered = useMemo(
    () => (statusFilter ? items.filter((i) => i.status === statusFilter) : items),
    [items, statusFilter],
  );

  const columns = [
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      ellipsis: true,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (s: string) => (
        <Tag color={STATUS_COLORS[s] || "default"}>{s}</Tag>
      ),
    },
    {
      title: "节点数",
      dataIndex: "nodeCount",
      key: "nodeCount",
      width: 100,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (t: string) => (t ? new Date(t).toLocaleString() : "-"),
    },
    {
      title: "完成时间",
      dataIndex: "finishedAt",
      key: "finishedAt",
      width: 180,
      render: (t?: string) => (t ? new Date(t).toLocaleString() : "-"),
    },
    {
      title: "操作",
      key: "actions",
      width: 100,
      render: (_: unknown, record: WorkflowSummary) => (
        <Button
          type="link"
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            navigate(`/workflows/${record.name}`);
          }}
        >
          查看
        </Button>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          marginBottom: 16,
        }}
      >
        <h2 style={{ margin: 0 }}>流水线运行</h2>
        <Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
          刷新
        </Button>
      </div>
      <div style={{ marginBottom: 16, display: "flex", gap: 8 }}>
        <Select
          allowClear
          placeholder="状态筛选"
          style={{ width: 140 }}
          value={statusFilter}
          onChange={(val) => setStatusFilter(val)}
          options={[
            { label: "Running", value: "Running" },
            { label: "Succeeded", value: "Succeeded" },
            { label: "Failed", value: "Failed" },
            { label: "Error", value: "Error" },
            { label: "Pending", value: "Pending" },
          ]}
        />
      </div>
      <Table
        dataSource={filtered}
        columns={columns}
        rowKey="name"
        loading={loading}
        locale={{ emptyText: "暂无流水线运行" }}
        onRow={(record) => ({
          onClick: () => navigate(`/workflows/${record.name}`),
          style: { cursor: "pointer" },
        })}
        pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
      />
    </div>
  );
}
