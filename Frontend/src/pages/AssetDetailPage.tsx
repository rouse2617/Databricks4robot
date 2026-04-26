import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Descriptions, Tag, Button, Spin, Typography, message, Card, Tabs,
  Table, Space, Popconfirm, Timeline, Empty,
} from "antd";
import {
  ArrowLeftOutlined, ReloadOutlined, PlayCircleOutlined,
  CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined,
  LockOutlined, FileOutlined, TagOutlined, SendOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import { assetsApi } from "../api/assets";
import type { Asset, AlgoEvent, AlgoStatus } from "../api/types";
import AssetPreviewHero from "../components/asset-detail/AssetPreviewHero";
import { buildPlaceholderPreviewManifest } from "../hooks/assets/useAssetPreview";

const { Title, Text } = Typography;

const statusColor: Record<string, string> = {
  approved: "success", rejected: "error", superseded: "warning", archived: "default",
};

const algoStatusConfig: Record<string, { color: string; icon: React.ReactNode; label: string }> = {
  ok: { color: "success", icon: <CheckCircleOutlined />, label: "完成" },
  failed: { color: "error", icon: <CloseCircleOutlined />, label: "失败" },
  running: { color: "warning", icon: <PlayCircleOutlined />, label: "运行中" },
  pending: { color: "default", icon: <ClockCircleOutlined />, label: "待处理" },
  blocked: { color: "default", icon: <LockOutlined />, label: "等待依赖" },
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
    started_at: fields.started_at ?? fields.at, // fallback to "at" field
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
  const [actionLoading, setActionLoading] = useState<string | null>(null);
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

  useEffect(() => {
    loadAsset();
    loadEvents();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleAlgoAction = async (algoKey: string, _action: "reset") => {
    if (!id) return;
    setActionLoading(algoKey);
    try {
      await assetsApi.resetAlgo(id, algoKey);
      msg.success(`${algoKey} 已重置为 pending`);
      loadAsset();
      loadEvents();
    } catch (err: any) {
      msg.error(err?.response?.data?.message ?? "操作失败");
    } finally {
      setActionLoading(null);
    }
  };

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
      children: (
        <Card size="small">
          <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
            <Descriptions.Item label="Asset ID">
              <code className="text-xs">{asset.asset_id}</code>
            </Descriptions.Item>
            <Descriptions.Item label="MCAP File">
              <code className="text-xs">{asset.mcap_file_id}</code>
            </Descriptions.Item>
            <Descriptions.Item label="起始时间 (ns)">
              {asset.start_timestamp_ns}
            </Descriptions.Item>
            <Descriptions.Item label="结束时间 (ns)">
              {asset.end_timestamp_ns ?? "—"}
            </Descriptions.Item>
            <Descriptions.Item label="时长">
              {asset.duration_sec ? `${asset.duration_sec.toFixed(3)} s` : "—"}
            </Descriptions.Item>
            <Descriptions.Item label="类型">{asset.type ?? "—"}</Descriptions.Item>
            <Descriptions.Item label="环境">{asset.env ?? "—"}</Descriptions.Item>
            <Descriptions.Item label="任务">{asset.task ?? "—"}</Descriptions.Item>
            <Descriptions.Item label="审核人">{asset.reviewer ?? "—"}</Descriptions.Item>
            <Descriptions.Item label="Owner">{asset.owner ?? "—"}</Descriptions.Item>
            <Descriptions.Item label="版本">{asset.version}</Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {dayjs(asset.created_at).format("YYYY-MM-DD HH:mm:ss")}
            </Descriptions.Item>
            <Descriptions.Item label="更新时间">
              {dayjs(asset.updated_at).format("YYYY-MM-DD HH:mm:ss")}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      ),
    },
    {
      key: "algo",
      label: `算法处理 (${algoList.length})`,
      children: (
        <div>
          <Table
            rowKey="key"
            dataSource={algoList}
            size="small"
            pagination={false}
            columns={[
              {
                title: "算法",
                dataIndex: "key",
                width: 180,
                render: (k: string) => <code className="text-xs">{k}</code>,
              },
              {
                title: "状态",
                dataIndex: "status",
                width: 100,
                render: (s: AlgoStatus) => {
                  const cfg = algoStatusConfig[s] ?? algoStatusConfig.pending;
                  return (
                    <Tag color={cfg.color} icon={cfg.icon}>
                      {cfg.label}
                    </Tag>
                  );
                },
              },
              {
                title: "开始时间",
                dataIndex: "started_at",
                width: 160,
                render: (v: string) => (v ? dayjs(v).format("MM-DD HH:mm:ss") : "—"),
              },
              {
                title: "耗时",
                key: "duration",
                width: 80,
                render: (_: unknown, r: any) => {
                  if (!r.started_at || !r.finished_at) return "—";
                  const ms = dayjs(r.finished_at).diff(dayjs(r.started_at), "second");
                  return `${ms}s`;
                },
              },
              {
                title: "产物 / 原因",
                key: "output",
                render: (_: unknown, r: any) => {
                  if (r.status === "failed" && r.reason) {
                    return <Text type="danger" className="text-xs">{r.reason}</Text>;
                  }
                  if (r.output_uri) {
                    return <Text className="text-xs font-mono">{r.output_uri.slice(-40)}</Text>;
                  }
                  if (r.status === "blocked") {
                    return <Text type="secondary">等待上游算法完成</Text>;
                  }
                  return "—";
                },
              },
              {
                title: "操作",
                key: "action",
                width: 80,
                render: (_: unknown, r: any) => {
                  if (r.status === "failed" || r.status === "ok") {
                    return (
                      <Popconfirm
                        title={`确认重置 ${r.key}？`}
                        onConfirm={() => handleAlgoAction(r.key, "reset")}
                      >
                        <Button
                          size="small"
                          icon={<ReloadOutlined />}
                          loading={actionLoading === r.key}
                        >
                          重置
                        </Button>
                      </Popconfirm>
                    );
                  }
                  return null;
                },
              },
            ]}
          />

          {/* Event timeline */}
          {events.length > 0 && (
            <Card title="状态变更历史" size="small" className="mt-4">
              <Timeline
                items={events.slice(0, 20).map((e) => ({
                  color:
                    e.new_status === "ok" ? "green" :
                    e.new_status === "failed" ? "red" :
                    e.new_status === "running" ? "blue" : "gray",
                  children: (
                    <div>
                      <Text strong className="text-xs">{e.algo_key}</Text>
                      <Text className="text-xs ml-2">
                        {e.prev_status ?? "—"} → {e.new_status}
                      </Text>
                      {e.reason && (
                        <Text type="danger" className="text-xs ml-2">
                          ({e.reason})
                        </Text>
                      )}
                      <Text type="secondary" className="text-xs ml-2">
                        {dayjs(e.created_at).format("MM-DD HH:mm:ss")}
                      </Text>
                    </div>
                  ),
                }))}
              />
            </Card>
          )}
        </div>
      ),
    },
    {
      key: "tags",
      label: (
        <span>
          <TagOutlined /> 标签
        </span>
      ),
      children: (
        <Card size="small">
          {asset.tags && Object.keys(asset.tags).length > 0 ? (
            <Space wrap>
              {Object.entries(asset.tags).map(([k, v]) => (
                <Tag key={k} closable={false}>
                  {k}: {v}
                </Tag>
              ))}
            </Space>
          ) : (
            <Empty description="暂无标签" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Card>
      ),
    },
    {
      key: "deliveries",
      label: (
        <span>
          <SendOutlined /> 交付历史
        </span>
      ),
      children: (
        <Card size="small">
          {asset.delivery_count && asset.delivery_count > 0 ? (
            <Descriptions size="small" column={2}>
              <Descriptions.Item label="交付次数">{asset.delivery_count}</Descriptions.Item>
              <Descriptions.Item label="最近交付时间">
                {asset.last_delivered_at ? dayjs(asset.last_delivered_at).format("YYYY-MM-DD HH:mm") : "—"}
              </Descriptions.Item>
              <Descriptions.Item label="最近交付客户">
                {asset.last_delivered_to ?? "—"}
              </Descriptions.Item>
            </Descriptions>
          ) : (
            <Empty description="该资产尚未交付" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
          <Text type="secondary" className="text-xs mt-2 block">
            完整交付历史需要 /assets/:id/deliveries 接口实现
          </Text>
        </Card>
      ),
    },
    {
      key: "files",
      label: (
        <span>
          <FileOutlined /> 文件
        </span>
      ),
      children: (
        <Card size="small">
          <Text type="secondary" className="text-xs mb-2 block">
            文件引用注册表
          </Text>
          {asset.files && Object.keys(asset.files).length > 0 ? (
            <Descriptions bordered size="small" column={1}>
              {Object.entries(asset.files).map(([k, v]) => (
                <Descriptions.Item key={k} label={k}>
                  <code className="text-xs">{v}</code>
                </Descriptions.Item>
              ))}
            </Descriptions>
          ) : (
            <Empty description="暂无文件引用" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Card>
      ),
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
