// ─── EventsPage — General event stream overview ───
// P2-FE-2: Shows recent asset_events in a timeline/table view.
// Fetches from /api/v1/assets/:id/events for a given asset.

import { ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	Empty,
	Modal,
	message,
	Space,
	Spin,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { assetsApi } from "../api/assets";
import type { AssetEvent } from "../api/types";
import { isCanonicalAssetId } from "../lib/assetId";
import { navigateToAssetDetail } from "../lib/assets/assetWorkbenchNavigation";

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

function formatJSON(v: unknown): string {
	if (v === undefined || v === null) return "";
	try {
		return JSON.stringify(v, null, 2);
	} catch {
		return String(v);
	}
}

// ─── Table columns factory (needs navigate for return-url tracking) ───

function buildEventColumns(
	navigate: ReturnType<typeof useNavigate>,
	onViewPayload: (event: AssetEvent) => void,
) {
	return [
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
				<a
					href={`/assets/${val}`}
					style={{ fontFamily: "monospace", fontSize: 12 }}
					onClick={(e) => {
						if (
							e.ctrlKey ||
							e.metaKey ||
							e.shiftKey ||
							e.altKey ||
							e.button !== 0
						)
							return;
						e.preventDefault();
						navigateToAssetDetail(navigate, val);
					}}
				>
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
				const color =
					val === "published" ? "green" : val === "pending" ? "orange" : "red";
				return <Tag color={color}>{val || "—"}</Tag>;
			},
		},
		{
			title: "Payload",
			dataIndex: "event_payload",
			key: "event_payload",
			ellipsis: true,
			render: (
				_val: Record<string, unknown> | undefined,
				record: AssetEvent,
			) => {
				const txt = record.event_payload
					? JSON.stringify(record.event_payload)
					: "";
				return (
					<Space size={6}>
						<Text
							type="secondary"
							style={{ fontSize: 11, fontFamily: "monospace" }}
						>
							{txt ? `${txt.slice(0, 100)}${txt.length > 100 ? "…" : ""}` : "—"}
						</Text>
						{txt ? (
							<Button
								size="small"
								type="link"
								onClick={() => onViewPayload(record)}
							>
								查看
							</Button>
						) : null}
					</Space>
				);
			},
		},
		{
			title: "时间",
			dataIndex: "occurred_at",
			key: "occurred_at",
			width: 160,
			render: (val: string) =>
				val ? <span title={val}>{dayjs(val).fromNow()}</span> : "—",
			sorter: (a: AssetEvent, b: AssetEvent) =>
				new Date(a.occurred_at || a.created_at).getTime() -
				new Date(b.occurred_at || b.created_at).getTime(),
		},
	];
}

// ─── Component ───

export default function EventsPage() {
	const navigate = useNavigate();
	const [assetId, setAssetId] = useState("");
	const [events, setEvents] = useState<AssetEvent[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [eventTypeFilter, setEventTypeFilter] = useState<string | undefined>(
		undefined,
	);
	const [startTime, setStartTime] = useState("");
	const [endTime, setEndTime] = useState("");
	const [payloadOpen, setPayloadOpen] = useState(false);
	const [payloadEvent, setPayloadEvent] = useState<AssetEvent | null>(null);
	const [msg, msgCtx] = message.useMessage();

	const columns = useMemo(
		() =>
			buildEventColumns(navigate, (ev) => {
				setPayloadEvent(ev);
				setPayloadOpen(true);
			}),
		[navigate],
	);

	const fetchEvents = useCallback(async () => {
		const id = assetId.trim();
		if (!id) {
			setEvents([]);
			setError(null);
			return;
		}
		if (!isCanonicalAssetId(id)) {
			setEvents([]);
			setError(null);
			return;
		}
		setLoading(true);
		setError(null);
		try {
			const startISO = startTime ? new Date(startTime).toISOString() : undefined;
			const endISO = endTime ? new Date(endTime).toISOString() : undefined;
			const res = await assetsApi.listEvents(id, {
				event_type: eventTypeFilter,
				start_time: startISO,
				end_time: endISO,
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
	}, [assetId, eventTypeFilter, startTime, endTime]);

	useEffect(() => {
		const id = assetId.trim();
		if (!id) {
			setEvents([]);
			setError(null);
			return;
		}
		if (!isCanonicalAssetId(id)) {
			setEvents([]);
			setError(null);
			return;
		}
		fetchEvents();
	}, [fetchEvents, assetId]);

	return (
		<div style={{ padding: 24 }}>
			{msgCtx}
			<Title level={3}>事件流总览</Title>
			<Text type="secondary" style={{ marginBottom: 16, display: "block" }}>
				查看资产事件流，支持按 Asset ID 和事件类型筛选。
			</Text>

			<Card size="small" style={{ marginBottom: 16 }}>
				<Space wrap>
					<div
						style={{
							display: "flex",
							alignItems: "center",
							gap: 8,
							width: 360,
							padding: "0 10px",
							borderRadius: 6,
							border: `1px solid ${
								assetId.trim() !== "" && !isCanonicalAssetId(assetId)
									? "#fa541c"
									: "#d9d9d9"
							}`,
							background: "#fff",
							height: 32,
						}}
					>
						<SearchOutlined />
						<input
							id="events-asset-id"
							name="asset_id"
							aria-label="Asset ID"
							placeholder="输入 8 位 Asset ID"
							value={assetId}
							onChange={(e) => setAssetId(e.target.value)}
							onKeyDown={(e) => {
								if (e.key === "Enter") fetchEvents();
							}}
							style={{
								flex: 1,
								border: "none",
								outline: "none",
								fontSize: 14,
							}}
						/>
					</div>
					<select
						id="events-event-type"
						name="event_type"
						aria-label="事件类型"
						value={eventTypeFilter ?? ""}
						onChange={(e) => {
							const v = e.target.value;
							setEventTypeFilter(v ? v : undefined);
						}}
						style={{
							width: 200,
							height: 32,
							border: "1px solid #d9d9d9",
							borderRadius: 6,
							padding: "0 8px",
							background: "#fff",
						}}
					>
						<option value="">所有事件类型</option>
						<option value="asset_created">asset_created</option>
						<option value="asset_updated">asset_updated</option>
						<option value="asset_lifecycle_changed">lifecycle_changed</option>
						<option value="tag_upserted">tag_upserted</option>
						<option value="tag_deleted">tag_deleted</option>
						<option value="algo_*">algo_* (所有算法)</option>
					</select>
					<ReloadOutlined
						onClick={fetchEvents}
						style={{ cursor: "pointer", fontSize: 16, color: "#1890ff" }}
						title="刷新"
					/>
				</Space>
				<Space wrap style={{ marginTop: 12 }}>
					<input
						aria-label="开始时间"
						type="datetime-local"
						value={startTime}
						onChange={(e) => setStartTime(e.target.value)}
						style={{
							height: 32,
							border: "1px solid #d9d9d9",
							borderRadius: 6,
							padding: "0 8px",
						}}
					/>
					<input
						aria-label="结束时间"
						type="datetime-local"
						value={endTime}
						onChange={(e) => setEndTime(e.target.value)}
						style={{
							height: 32,
							border: "1px solid #d9d9d9",
							borderRadius: 6,
							padding: "0 8px",
						}}
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
			) : events.length === 0 &&
				assetId.trim() &&
				!isCanonicalAssetId(assetId) ? (
				<Empty description="请输入完整有效的 8 位 Asset ID 后再查询" />
			) : events.length === 0 && assetId.trim() ? (
				<Empty description="暂无事件" />
			) : (
				<Table
					dataSource={events}
					columns={columns}
					rowKey="event_id"
					size="small"
					pagination={{
						pageSize: 50,
						showSizeChanger: true,
						showTotal: (t) => `共 ${t} 条`,
					}}
					scroll={{ x: 1200 }}
				/>
			)}

			<Modal
				title="Event payload"
				open={payloadOpen}
				onCancel={() => setPayloadOpen(false)}
				footer={[
					<Button
						key="copy"
						onClick={async () => {
							const text = formatJSON(payloadEvent?.event_payload);
							if (!text) return;
							try {
								await navigator.clipboard.writeText(text);
								msg.success("已复制 payload");
							} catch {
								msg.error("复制失败（浏览器权限限制）");
							}
						}}
						disabled={!payloadEvent?.event_payload}
					>
						复制
					</Button>,
					<Button
						key="close"
						type="primary"
						onClick={() => setPayloadOpen(false)}
					>
						关闭
					</Button>,
				]}
				width={760}
			>
				<div
					style={{
						display: "flex",
						gap: 12,
						marginBottom: 8,
						flexWrap: "wrap",
					}}
				>
					<Tag>{payloadEvent?.event_type || "—"}</Tag>
					<Text
						type="secondary"
						style={{ fontFamily: "monospace", fontSize: 12 }}
					>
						seq={payloadEvent?.event_seq ?? "—"}
					</Text>
					<Text
						type="secondary"
						style={{ fontFamily: "monospace", fontSize: 12 }}
					>
						asset_id={payloadEvent?.asset_id ?? "—"}
					</Text>
				</div>
				<pre
					style={{
						margin: 0,
						maxHeight: 520,
						overflow: "auto",
						padding: 12,
						borderRadius: 8,
						background: "#0b1020",
						color: "#e6edf3",
						fontSize: 12,
						lineHeight: 1.5,
					}}
				>
					{formatJSON(payloadEvent?.event_payload) || "—"}
				</pre>
			</Modal>
		</div>
	);
}
