import { useEffect, useState } from "react";
import { Button, Table, Tag } from "antd";
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
        <h2 style={{ margin: 0 }}>运行记录</h2>
        <Button icon={<ReloadOutlined />} onClick={refresh} loading={loading}>
          刷新
        </Button>
      </div>
      <Table
        dataSource={items}
        columns={columns}
        rowKey="name"
        loading={loading}
        locale={{ emptyText: "暂无运行记录" }}
        onRow={(record) => ({
          onClick: () => navigate(`/workflows/${record.name}`),
          style: { cursor: "pointer" },
        })}
        pagination={false}
      />
    </div>
  );
}
