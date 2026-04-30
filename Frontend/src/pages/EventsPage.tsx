// ─── EventsPage — General event stream overview ───
// P2-FE-2: Shows recent asset_events in a timeline/table view.
// Fetches from /api/v1/assets/:id/events for a given asset.

import { useState, useEffect, useCallback } from "react";
import { Card, Table, Tag, Input, Space, Typography, Empty, Spin, Select } from "antd";
import { ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { assetsApi } from "../api/assets";
import type { AssetEvent } from "../api/types";

dayjs.extend(relativeTime);

const { Title, Text } = Typography;

// ─── Event type color mapping ───

function eventTypeColor(eventType: string): string {
  if (eventType.startsWith("algo_")) return "purple";
  if (eventType.startsWith("tag_")) return "cyan";
  if (eventType === "asset_created") return "green";
  if (eventType === "asset_updated") return "blue";
  if (eventType === "asset_lifecycle_changed") return "orange";
  if (eventType === "asset_delivered") return "gold";
  return "default";
}

// ─── Table columns ───

const columns = [
  {
    title: "Event Seq",
    dataIndex: "event_seq",
    key: "event_seq",
    width: 100,
    sorter: (a: AssetEvent, b: AssetEvent) => a.event_seq - b.event_seq,
  },
  {
    title: "类型",
    dataIndex: "event_type",
    key: "event_type",
    width: 180,
    render: (val: string) => <Tag color={eventTypeColor(val)}>{val}</Tag>,
  },
  {
    title: "Asset ID",
    dataIndex: "asset_id",
    key: "asset_id",
    width: 280,
    ellipsis: true,
    render: (val: string) => (
      <a href={`/assets/${val}`} style={{ fontFamily: "monospace", fontSize: 12 }}>
        {val}
      </a>
    ),
  },
  {
    title: "来源",
    dataIndex: "event_source",
    key: "event_source",
    width: 100,
    render: (val: string) => <Text type="secondary">{val || "—"}</Text>,
  },
  {
    title: "发布状态",
    dataIndex: "publish_state",
    key: "publish_state",
    width: 100,
    render: (val: string) => {
      const color = val === "published" ? "green" : val === "pending" ? "orange" : "red";
      return <Tag color={color}>{val || "—"}</Tag>;
    },
  },
  {
    title: "Payload",
    dataIndex: "event_payload",
    key: "event_payload",
    ellipsis: true,
    render: (val: Record<string, unknown> | undefined) => (
      <Text type="secondary" style={{ fontSize: 11, fontFamily: "monospace" }}>
        {val ? JSON.stringify(val).slice(0, 120) : "—"}
      </Text>
    ),
  },
  {
    title: "时间",
    dataIndex: "occurred_at",
    key: "occurred_at",
    width: 160,
    render: (val: string) =>
      val ? (
        <span title={val}>{dayjs(val).fromNow()}</span>
      ) : (
        "—"
      ),
    sorter: (a: AssetEvent, b: AssetEvent) =>
      new Date(a.occurred_at || a.created_at).getTime() -
      new Date(b.occurred_at || b.created_at).getTime(),
  },
];

// ─── Component ───

export default function EventsPage() {
  const [assetId, setAssetId] = useState("");
  const [events, setEvents] = useState<AssetEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [eventTypeFilter, setEventTypeFilter] = useState<string | undefined>(undefined);

  const fetchEvents = useCallback(async () => {
    if (!assetId.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const res = await assetsApi.listEvents(assetId.trim(), {
        event_type: eventTypeFilter,
        limit: 100,
      });
      setEvents(res.items);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to fetch events";
      setError(msg);
      setEvents([]);
    } finally {
      setLoading(false);
    }
  }, [assetId, eventTypeFilter]);

  useEffect(() => {
    if (assetId.trim()) {
      fetchEvents();
    }
  }, [fetchEvents, assetId, eventTypeFilter]);

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>事件流总览</Title>
      <Text type="secondary" style={{ marginBottom: 16, display: "block" }}>
        查看资产事件流，支持按 Asset ID 和事件类型筛选。
      </Text>

      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Input
            prefix={<SearchOutlined />}
            placeholder="输入 Asset ID"
            value={assetId}
            onChange={(e) => setAssetId(e.target.value)}
            onPressEnter={fetchEvents}
            style={{ width: 360 }}
            allowClear
          />
          <Select
            placeholder="事件类型"
            value={eventTypeFilter}
            onChange={setEventTypeFilter}
            allowClear
            style={{ width: 200 }}
            options={[
              { value: "asset_created", label: "asset_created" },
              { value: "asset_updated", label: "asset_updated" },
              { value: "asset_lifecycle_changed", label: "lifecycle_changed" },
              { value: "tag_upserted", label: "tag_upserted" },
              { value: "tag_deleted", label: "tag_deleted" },
              { value: "algo_*", label: "algo_* (所有算法)" },
            ]}
          />
          <ReloadOutlined
            onClick={fetchEvents}
            style={{ cursor: "pointer", fontSize: 16, color: "#1890ff" }}
            title="刷新"
          />
        </Space>
      </Card>

      {error && (
        <Card size="small" style={{ marginBottom: 16, borderColor: "#ff4d4f" }}>
          <Text type="danger">{error}</Text>
        </Card>
      )}

      {loading ? (
        <div style={{ textAlign: "center", padding: 48 }}>
          <Spin size="large" />
        </div>
      ) : events.length === 0 && assetId.trim() ? (
        <Empty description="暂无事件" />
      ) : (
        <Table
          dataSource={events}
          columns={columns}
          rowKey="event_id"
          size="small"
          pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
          scroll={{ x: 1200 }}
        />
      )}
    </div>
  );
}
