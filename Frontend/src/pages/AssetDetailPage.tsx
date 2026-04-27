import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Tag, Button, Spin, Typography, message, Tabs, Empty,
} from "antd";
import {
  ArrowLeftOutlined, FileOutlined, TagOutlined, SendOutlined,
} from "@ant-design/icons";
import { assetsApi } from "../api/assets";
import type { Asset, AlgoEvent, AlgoStatus } from "../api/types";
import AssetPreviewHero from "../components/asset-detail/AssetPreviewHero";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";
import OverviewTab from "../components/asset-detail/OverviewTab";
import AlgoTab from "../components/asset-detail/AlgoTab";
import TagsTab from "../components/asset-detail/TagsTab";
import DeliveryHistoryTab from "../components/asset-detail/DeliveryHistoryTab";
import FilesTab from "../components/asset-detail/FilesTab";

const { Title, Text } = Typography;

const statusColor: Record<string, string> = {
  approved: "success", rejected: "error", superseded: "warning", archived: "default",
};

/** Parse algo_results map into structured algo info list. */
function parseAlgoResults(algoResults: Record<string, string> | undefined) {
  if (!algoResults) return [];
  const algos = new Map<string, Record<string, string>>();
  for (const [k, v] of Object.entries(algoResults)) {
    const colonIdx = k.indexOf(":");
    if (colonIdx === -1) continue;
    const algoKey = k.substring(0, colonIdx);
    const field = k.substring(colonIdx + 1);
    if (!algos.has(algoKey)) algos.set(algoKey, {});
    algos.get(algoKey)![field] = v;
  }
  return Array.from(algos.entries()).map(([key, fields]) => ({
    key,
    name: key.split("@")[0],
    version: key.split("@")[1] ?? "",
    status: (fields.status ?? "pending") as AlgoStatus,
    started_at: fields.started_at ?? fields.at,
    finished_at: fields.finished_at,
    method: fields.method,
    run_id: fields.run_id,
    output_uri: fields.output_uri,
    reason: fields.reason,
  }));
}

export default function AssetDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [asset, setAsset] = useState<Asset | null>(null);
  const [loading, setLoading] = useState(true);
  const [events, setEvents] = useState<AlgoEvent[]>([]);
  const [msg, msgCtx] = message.useMessage();

  const loadAsset = () => {
    if (!id) return;
    setLoading(true);
    assetsApi
      .get(id)
      .then(setAsset)
      .catch(() => msg.error("加载资产失败"))
      .finally(() => setLoading(false));
  };

  const loadEvents = () => {
    if (!id) return;
    assetsApi
      .listAlgoEvents(id)
      .then((evts) => setEvents(evts ?? []))
      .catch(() => setEvents([]));
  };

  const refresh = () => {
    loadAsset();
    loadEvents();
  };

  useEffect(() => {
    refresh();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  if (loading) {
    return (
      <div className="flex items-center justify-center" style={{ height: "60vh" }}>
        <Spin size="large" />
      </div>
    );
  }
  if (!asset) {
    return <Empty description="资产未找到" />;
  }

  const algoList = parseAlgoResults(asset.algo_results);

  const tabItems = [
    {
      key: "overview",
      label: "概览",
      children: <OverviewTab asset={asset} />,
    },
    {
      key: "algo",
      label: `算法处理 (${algoList.length})`,
      children: (
        <AlgoTab
          assetId={asset.asset_id}
          algoList={algoList}
          events={events}
          onRefresh={refresh}
        />
      ),
    },
    {
      key: "tags",
      label: <span><TagOutlined /> 标签</span>,
      children: (
        <TagsTab
          assetId={asset.asset_id}
          tags={asset.tags ?? {}}
          onUpdate={loadAsset}
        />
      ),
    },
    {
      key: "deliveries",
      label: <span><SendOutlined /> 交付历史</span>,
      children: <DeliveryHistoryTab assetId={asset.asset_id} />,
    },
    {
      key: "files",
      label: <span><FileOutlined /> 文件</span>,
      children: <FilesTab files={asset.files ?? {}} />,
    },
  ];

  return (
    <div>
      {msgCtx}

      {/* Header */}
      <div className="flex items-center gap-3 mb-4">
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(-1)}
          size="small"
        />
        <Title level={4} style={{ margin: 0 }}>
          资产详情
        </Title>
        <Tag color={statusColor[asset.status] ?? "default"}>{asset.status}</Tag>
        <Text type="secondary" className="text-xs font-mono">
          {asset.asset_id}
        </Text>
      </div>

      {/* Preview Hero */}
      <AssetPreviewHero
        asset={asset}
        previewManifest={buildPlaceholderPreviewManifest(asset)}
      />

      {/* Tabs */}
      <Tabs
        defaultActiveKey="overview"
        items={tabItems}
        size="small"
        style={{ marginTop: -8 }}
      />
    </div>
  );
}
