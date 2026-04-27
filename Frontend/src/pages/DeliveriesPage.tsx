import { useEffect, useState, useCallback } from "react";
import { Typography, Table, Select, Button, Tag, Space, message } from "antd";
import { ReloadOutlined, PlusOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { deliveriesApi } from "../api/deliveries";
import type { Delivery } from "../api/types";
import CreateDeliveryModal from "../components/deliveries/CreateDeliveryModal";

const { Title } = Typography;

const STATUS_OPTIONS = [
  { label: "全部", value: "" },
  { label: "草稿", value: "draft" },
  { label: "待交付", value: "pending" },
  { label: "已交付", value: "delivered" },
  { label: "已接受", value: "accepted" },
  { label: "已拒绝", value: "rejected" },
];

const STATUS_COLOR: Record<string, string> = {
  draft: "default",
  pending: "processing",
  delivered: "success",
  accepted: "green",
  rejected: "error",
};

export default function DeliveriesPage() {
  const navigate = useNavigate();
  const [msg, msgCtx] = message.useMessage();
  const [items, setItems] = useState<Delivery[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await deliveriesApi.list({
        page,
        page_size: pageSize,
        status: status || undefined,
      });
      setItems(res.items ?? []);
      setTotal(res.total);
    } catch {
      msg.error("加载交付列表失败");
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, status, msg]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const columns = [
    {
      title: "交付 ID",
      dataIndex: "delivery_id",
      key: "delivery_id",
      width: 220,
      ellipsis: true,
    },
    {
      title: "客户",
      dataIndex: "customer_id",
      key: "customer_id",
      width: 160,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (val: string) => (
        <Tag color={STATUS_COLOR[val] ?? "default"}>{val}</Tag>
      ),
    },
    {
      title: "交付时间",
      dataIndex: "delivered_at",
      key: "delivered_at",
      width: 180,
      render: (val?: string) => val ?? "—",
    },
    {
      title: "资产数",
      dataIndex: "asset_count",
      key: "asset_count",
      width: 80,
    },
    {
      title: "Owner",
      dataIndex: "owner",
      key: "owner",
      width: 120,
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 180,
    },
  ];

  return (
    <div>
      {msgCtx}
      <Title level={4} style={{ margin: "0 0 16px 0" }}>
        交付管理
      </Title>

      {/* Toolbar: status filter + refresh + create */}
      <Space style={{ marginBottom: 12 }} wrap>
        <Select
          value={status}
          onChange={(v) => {
            setStatus(v);
            setPage(1);
          }}
          options={STATUS_OPTIONS}
          style={{ width: 140 }}
          placeholder="状态过滤"
        />
        <Button icon={<ReloadOutlined />} onClick={fetchData}>
          刷新
        </Button>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalOpen(true)}
        >
          新建交付
        </Button>
      </Space>

      <Table
        rowKey="delivery_id"
        columns={columns}
        dataSource={items}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
        onRow={(record) => ({
          onClick: () => navigate(`/deliveries/${record.delivery_id}`),
          style: { cursor: "pointer" },
        })}
        size="middle"
        scroll={{ x: 1000 }}
      />

      <CreateDeliveryModal
        open={modalOpen}
        assetIds={[]}
        onClose={() => setModalOpen(false)}
        onSuccess={(deliveryId) => {
          setModalOpen(false);
          navigate(`/deliveries/${deliveryId}`);
        }}
      />
    </div>
  );
}
