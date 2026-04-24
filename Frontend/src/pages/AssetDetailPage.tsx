import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Descriptions, Tag, Button, Spin, Typography, message, Card } from "antd";
import { ArrowLeftOutlined, DownloadOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { assetsApi, type Asset } from "../api/assets";

const { Title } = Typography;

const statusColor: Record<string, string> = {
  active: "green", pending: "gold", archived: "blue", deleted: "red",
};

export default function AssetDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [asset, setAsset] = useState<Asset | null>(null);
  const [loading, setLoading] = useState(true);
  const [msg, msgCtx] = message.useMessage();

  useEffect(() => {
    if (!id) return;
    assetsApi.get(id)
      .then(setAsset)
      .catch(() => msg.error("Failed to load asset"))
      .finally(() => setLoading(false));
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  if (loading) return <Spin size="large" className="flex justify-center mt-20" />;
  if (!asset) return <div>Asset not found</div>;

  return (
    <div>
      {msgCtx}
      <div className="flex items-center gap-3 mb-6">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>Back</Button>
        <Title level={4} className="!mb-0">Asset Detail</Title>
        <Tag color={statusColor[asset.status]}>{asset.status}</Tag>
      </div>

      <Card title="Metadata" className="mb-4">
        <Descriptions column={2} bordered size="small">
          <Descriptions.Item label="Asset ID"><code>{asset.asset_id}</code></Descriptions.Item>
          <Descriptions.Item label="MCAP File"><code>{asset.mcap_file_id}</code></Descriptions.Item>
          <Descriptions.Item label="t_start (ns)">{asset.t_start}</Descriptions.Item>
          <Descriptions.Item label="t_end (ns)">{asset.t_end}</Descriptions.Item>
          <Descriptions.Item label="Duration">{asset.duration_sec.toFixed(3)} s</Descriptions.Item>
          <Descriptions.Item label="Reviewer">{asset.reviewer}</Descriptions.Item>
          <Descriptions.Item label="Owner">{asset.owner}</Descriptions.Item>
          <Descriptions.Item label="Version">{asset.version}</Descriptions.Item>
          <Descriptions.Item label="Created">{dayjs(asset.created_at).format("YYYY-MM-DD HH:mm:ss")}</Descriptions.Item>
          <Descriptions.Item label="Updated">{dayjs(asset.updated_at).format("YYYY-MM-DD HH:mm:ss")}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Card title="Actions">
        <Button icon={<DownloadOutlined />} type="primary" disabled>
          Download MCAP (Phase 0.5)
        </Button>
      </Card>
    </div>
  );
}
