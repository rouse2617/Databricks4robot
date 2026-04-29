import { useState, useEffect } from "react";
import {
  Table, Tag, Button, Popconfirm, Card, Timeline, Typography,
  Modal, Select, Input, message,
} from "antd";
import {
  ReloadOutlined, PlayCircleOutlined, CheckCircleOutlined,
  CloseCircleOutlined, ClockCircleOutlined, LockOutlined,
  RocketOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import { assetsApi } from "../../api/assets";
import { algoRegistryApi, type AlgoRegistryItem } from "../../api/algoRegistry";
import type { AlgoEvent, AlgoStatus } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";

const { Text } = Typography;

const algoStatusConfig: Record<string, { color: string; icon: React.ReactNode; label: string }> = {
  ok: { color: "success", icon: <CheckCircleOutlined />, label: "完成" },
  failed: { color: "error", icon: <CloseCircleOutlined />, label: "失败" },
  running: { color: "warning", icon: <PlayCircleOutlined />, label: "运行中" },
  pending: { color: "default", icon: <ClockCircleOutlined />, label: "待处理" },
  blocked: { color: "default", icon: <LockOutlined />, label: "等待依赖" },
};

interface AlgoInfo {
  key: string;
  name: string;
  version: string;
  status: AlgoStatus;
  started_at?: string;
  finished_at?: string;
  method?: string;
  run_id?: string;
  output_uri?: string;
  reason?: string;
}

interface Props {
  assetId: string;
  algoList: AlgoInfo[];
  events: AlgoEvent[];
  eventsLoading?: boolean;
  hasMoreEvents?: boolean;
  onLoadMoreEvents?: () => void;
  onRefresh: () => void;
}

export default function AlgoTab({
  assetId,
  algoList,
  events,
  eventsLoading = false,
  hasMoreEvents = false,
  onLoadMoreEvents,
  onRefresh,
}: Props) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [startModalOpen, setStartModalOpen] = useState(false);
  const [algoKey, setAlgoKey] = useState("");
  const [method, setMethod] = useState("");
  const [starting, setStarting] = useState(false);
  const [registry, setRegistry] = useState<AlgoRegistryItem[]>([]);

  useEffect(() => {
    algoRegistryApi.list().then(setRegistry).catch(() => {});
  }, []);

  const handleReset = async (key: string) => {
    setActionLoading(key);
    try {
      await assetsApi.resetAlgo(assetId, key);
      message.success(`${key} 已重置为 pending`);
      onRefresh();
    } catch (err) {
      message.error(extractApiErrorMessage(err, "操作失败"));
    } finally {
      setActionLoading(null);
    }
  };

  const handleStartAlgo = async () => {
    if (!algoKey || !method) return;
    setStarting(true);
    try {
      await assetsApi.startAlgo(assetId, algoKey, { method });
      message.success(`算法 ${algoKey} 已启动`);
      setStartModalOpen(false);
      setAlgoKey("");
      setMethod("");
      onRefresh();
    } catch (err) {
      message.error(extractApiErrorMessage(err, "启动算法失败"));
    } finally {
      setStarting(false);
    }
  };

  return (
    <div>
      <div className="mb-3">
        <Button
          type="primary"
          icon={<RocketOutlined />}
          size="small"
          onClick={() => setStartModalOpen(true)}
        >
          启动算法
        </Button>
      </div>

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
              return <Tag color={cfg.color} icon={cfg.icon}>{cfg.label}</Tag>;
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
            render: (_: unknown, r: AlgoInfo) => {
              if (!r.started_at || !r.finished_at) return "—";
              const sec = dayjs(r.finished_at).diff(dayjs(r.started_at), "second");
              return `${sec}s`;
            },
          },
          {
            title: "产物 / 原因",
            key: "output",
            render: (_: unknown, r: AlgoInfo) => {
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
            render: (_: unknown, r: AlgoInfo) => {
              if (r.status === "failed" || r.status === "ok") {
                return (
                  <Popconfirm
                    title={`确认重置 ${r.key}？`}
                    onConfirm={() => handleReset(r.key)}
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
          {hasMoreEvents && (
            <div className="mt-3">
              <Button
                size="small"
                onClick={onLoadMoreEvents}
                loading={eventsLoading}
              >
                加载更多
              </Button>
            </div>
          )}
        </Card>
      )}

      {/* Start Algo Modal */}
      <Modal
        title="启动算法"
        open={startModalOpen}
        onCancel={() => {
          setStartModalOpen(false);
          setAlgoKey("");
          setMethod("");
        }}
        onOk={handleStartAlgo}
        confirmLoading={starting}
        okButtonProps={{ disabled: !algoKey || !method }}
        okText="启动"
        cancelText="取消"
      >
        <div className="mb-3">
          <label className="block text-sm mb-1">算法 Key</label>
          <Select
            placeholder="选择算法"
            value={algoKey || undefined}
            onChange={setAlgoKey}
            style={{ width: "100%" }}
            options={registry.map((r) => ({
              label: `${r.name}@${r.version}`,
              value: r.key,
            }))}
            showSearch
          />
        </div>
        <div>
          <label className="block text-sm mb-1">Method</label>
          <Input
            placeholder="输入 method（如 default, gpu, cpu）"
            value={method}
            onChange={(e) => setMethod(e.target.value)}
          />
        </div>
      </Modal>
    </div>
  );
}
