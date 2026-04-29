import { useEffect, useState, useCallback, useRef } from "react";
import {
  Typography,
  Descriptions,
  Tag,
  Table,
  Spin,
  Button,
  message,
} from "antd";
import { ArrowLeftOutlined, ReloadOutlined } from "@ant-design/icons";
import { useParams, useNavigate } from "react-router-dom";
import { deliveriesApi } from "../api/deliveries";
import { assetsApi } from "../api/assets";
import type { Delivery, DeliveryItem, Asset } from "../api/types";
import { formatDurationSeconds, getAssetStateColor, getLifecycleState } from "../lib/assetPresentation";

const { Title } = Typography;

const STATUS_COLOR: Record<string, string> = {
  draft: "default",
  pending: "processing",
  delivered: "success",
  accepted: "green",
  rejected: "error",
};

export default function DeliveryDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [msg, msgCtx] = message.useMessage();

  const [delivery, setDelivery] = useState<Delivery | null>(null);
  const [loading, setLoading] = useState(true);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [assetsLoading, setAssetsLoading] = useState(false);

  // antd useMessage may return a fresh reference per render; capture in ref so
  // fetchDelivery's identity stays stable (otherwise useEffect re-fires every
  // render → infinite refetch).
  const msgRef = useRef(msg);
  msgRef.current = msg;

  const fetchDelivery = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    try {
      const d = await deliveriesApi.get(id);
      setDelivery(d);
    } catch {
      msgRef.current.error("加载交付详情失败");
    } finally {
      setLoading(false);
    }
  }, [id]);

  const fetchAssets = useCallback(async () => {
    if (!id) return;
    setAssetsLoading(true);
    try {
      const items: DeliveryItem[] = await deliveriesApi.listItems(id);
      // Fetch each asset in parallel
      const assetResults = await Promise.allSettled(
        items.map((item) => assetsApi.get(item.asset_id)),
      );
      setAssets(
        assetResults
          .filter(
            (r): r is PromiseFulfilledResult<Asset> =>
              r.status === "fulfilled",
          )
          .map((r) => r.value),
      );
    } catch {
      // Keep the page usable even if the related-assets call fails.
      setAssets([]);
    } finally {
      setAssetsLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchDelivery();
    fetchAssets();
  }, [fetchDelivery, fetchAssets]);

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!delivery) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Typography.Text type="secondary">交付记录未找到</Typography.Text>
      </div>
    );
  }

  const assetColumns = [
    {
      title: "资产 ID",
      dataIndex: "asset_id",
      key: "asset_id",
      width: 240,
      ellipsis: true,
      render: (val: string) => (
        <a onClick={() => navigate(`/assets/${val}`)}>{val}</a>
      ),
    },
    {
      title: "生命周期",
      key: "status",
      width: 100,
      render: (_: unknown, asset: Asset) => (
        <Tag color={getAssetStateColor(asset)}>{getLifecycleState(asset) || "—"}</Tag>
      ),
    },
    {
      title: "时长 (s)",
      key: "duration_sec",
      width: 100,
      render: (_: unknown, asset: Asset) => formatDurationSeconds(asset),
    },
    { title: "Owner", dataIndex: "owner", key: "owner", width: 120 },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      key: "updated_at",
      width: 180,
    },
  ];

  return (
    <div>
      {msgCtx}

      {/* Header */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          marginBottom: 16,
        }}
      >
        <Button
          icon={<ArrowLeftOutlined />}
          type="text"
          onClick={() => navigate("/deliveries")}
        />
        <Title level={4} style={{ margin: 0 }}>
          交付详情
        </Title>
        <Tag color={STATUS_COLOR[delivery.status] ?? "default"}>
          {delivery.status}
        </Tag>
        <Button
          icon={<ReloadOutlined />}
          size="small"
          onClick={() => {
            fetchDelivery();
            fetchAssets();
          }}
        >
          刷新
        </Button>
      </div>

      {/* Basic info */}
      <Descriptions
        bordered
        column={2}
        size="small"
        style={{ marginBottom: 24 }}
      >
        <Descriptions.Item label="交付 ID">
          {delivery.delivery_id}
        </Descriptions.Item>
        <Descriptions.Item label="客户 ID">
          {delivery.customer_id}
        </Descriptions.Item>
        <Descriptions.Item label="合同号">
          {delivery.contract_id ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label="Owner">
          {delivery.owner}
        </Descriptions.Item>
        <Descriptions.Item label="资产数">
          {delivery.asset_count}
        </Descriptions.Item>
        <Descriptions.Item label="交付时间">
          {delivery.delivered_at ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label="Manifest URI" span={2}>
          {delivery.manifest_uri ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label="备注" span={2}>
          {delivery.note ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {delivery.created_at}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {delivery.updated_at}
        </Descriptions.Item>
      </Descriptions>

      {/* Related assets */}
      <Title level={5} style={{ marginBottom: 12 }}>
        关联资产
      </Title>
      <Table
        rowKey="asset_id"
        columns={assetColumns}
        dataSource={assets}
        loading={assetsLoading}
        pagination={false}
        size="small"
        scroll={{ x: 800 }}
        locale={{ emptyText: "暂无关联资产" }}
      />
    </div>
  );
}
