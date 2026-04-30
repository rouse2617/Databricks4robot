import { useEffect, useState } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";
import {
  Tag, Button, Spin, Typography, message, Tabs, Empty,
} from "antd";
import {
  ArrowLeftOutlined, FileOutlined, TagOutlined, SendOutlined,
} from "@ant-design/icons";
import { assetsApi } from "../api/assets";
import type { Asset, AlgoEvent, AlgoStatus, AssetEvent } from "../api/types";
import AssetPreviewHero from "../components/asset-detail/AssetPreviewHero";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";
import OverviewTab from "../components/asset-detail/OverviewTab";
import AlgoTab from "../components/asset-detail/AlgoTab";
import AssetEventsTab from "../components/asset-detail/AssetEventsTab";
import TagsTab from "../components/asset-detail/TagsTab";
import DeliveryHistoryTab from "../components/asset-detail/DeliveryHistoryTab";
import FilesTab from "../components/asset-detail/FilesTab";
import { getAssetStateColor, getLifecycleState } from "../lib/assetPresentation";
import {
  consumeStoredReturnUrl,
  clearStoredAssetDetailReturn,
  isSafeInternalReturnUrl,
  type AssetDetailLocationState,
} from "../lib/assets/assetWorkbenchNavigation";

const { Title, Text } = Typography;

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
  const location = useLocation();
  const returnTo = (location.state as AssetDetailLocationState | null)?.assetsReturnTo;
  const [asset, setAsset] = useState<Asset | null>(null);
  const [loading, setLoading] = useState(true);
  const [algoEvents, setAlgoEvents] = useState<AlgoEvent[]>([]);
  const [algoEventsCursor, setAlgoEventsCursor] = useState<number | null>(null);
  const [algoEventsLoading, setAlgoEventsLoading] = useState(false);
  const [allEvents, setAllEvents] = useState<AssetEvent[]>([]);
  const [allEventsCursor, setAllEventsCursor] = useState<number | null>(null);
  const [allEventsLoading, setAllEventsLoading] = useState(false);
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

  const loadAlgoEvents = (cursor?: number) => {
    if (!id) return;
    setAlgoEventsLoading(true);
    assetsApi
      .listAlgoEvents(id, undefined, cursor, 20)
      .then((resp) => {
        setAlgoEvents((prev) => (cursor ? [...prev, ...(resp.items ?? [])] : (resp.items ?? [])));
        setAlgoEventsCursor(resp.next_cursor ?? null);
      })
      .catch(() => {
        if (!cursor) setAlgoEvents([]);
      })
      .finally(() => setAlgoEventsLoading(false));
  };

  const loadAllEvents = (cursor?: number) => {
    if (!id) return;
    setAllEventsLoading(true);
    assetsApi
      .listEvents(id, cursor ? { cursor, limit: 20 } : { limit: 20 })
      .then((resp) => {
        setAllEvents((prev) => (cursor ? [...prev, ...(resp.items ?? [])] : (resp.items ?? [])));
        setAllEventsCursor(resp.next_cursor ?? null);
      })
      .catch(() => {
        if (!cursor) setAllEvents([]);
      })
      .finally(() => setAllEventsLoading(false));
  };

  const refresh = () => {
    loadAsset();
    loadAlgoEvents();
    loadAllEvents();
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
          events={algoEvents}
          eventsLoading={algoEventsLoading}
          hasMoreEvents={algoEventsCursor !== null}
          onLoadMoreEvents={() => {
            if (algoEventsCursor !== null) loadAlgoEvents(algoEventsCursor);
          }}
          onRefresh={refresh}
        />
      ),
    },
    {
      key: "events",
      label: `全部事件 (${allEvents.length})`,
      children: (
        <AssetEventsTab
          events={allEvents}
          loading={allEventsLoading}
          hasMore={allEventsCursor !== null}
          onLoadMore={() => {
            if (allEventsCursor !== null) loadAllEvents(allEventsCursor);
          }}
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
          onUpdate={refresh}
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
          onClick={() => {
            if (returnTo && isSafeInternalReturnUrl(returnTo)) {
              clearStoredAssetDetailReturn();
              navigate(returnTo);
              return;
            }
            const stored = consumeStoredReturnUrl();
            if (stored) {
              navigate(stored);
              return;
            }
            if (typeof window !== "undefined" && window.history.length > 1) {
              navigate(-1);
              return;
            }
            navigate("/assets");
          }}
          size="small"
          aria-label="返回"
        />
        <Title level={4} style={{ margin: 0 }}>
          资产详情
        </Title>
        <Tag color={getAssetStateColor(asset)}>{getLifecycleState(asset) || "—"}</Tag>
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
