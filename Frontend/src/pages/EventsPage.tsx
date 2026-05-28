// ─── EventsPage — General event stream overview ───
// Default view: loads recent events from GET /api/v1/events (last 24h).
// Per-asset mode: triggered by ?asset_id= or entering a valid 8-char Asset ID.

import { ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import {
	Alert,
	Radio,
	Button,
	Card,
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
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
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
			width: 100,
			ellipsis: true,
			render: (val: string) =>
				val ? (
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
				) : (
					"—"
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

function initialAssetIdFromSearch(searchParams: URLSearchParams): string {
	const raw = searchParams.get("asset_id")?.trim() ?? "";
	return raw && isCanonicalAssetId(raw) ? raw : "";
}

export default function EventsPage() {
	const navigate = useNavigate();
	const [searchParams, setSearchParams] = useSearchParams();
	const [assetId, setAssetId] = useState(() =>
		initialAssetIdFromSearch(searchParams),
	);
const [events, setEvents] = useState<AssetEvent[]>([]);
	const [loading, setLoading] = useState(false);
	const [loadingMore, setLoadingMore] = useState(false);
	const [realtimeMode, setRealtimeMode] = useState(false);
	const [streamConnected, setStreamConnected] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [eventTypeFilter] = useState<string | undefined>(undefined);
	const [payloadOpen, setPayloadOpen] = useState(false);
	const [payloadEvent, setPayloadEvent] = useState<AssetEvent | null>(null);
	const [nextCursor, setNextCursor] = useState<number | undefined>(undefined);
	const [msgApi, msgCtx] = message.useMessage();
	const fetchIdRef = useRef(0);

	const isPerAsset = assetId.trim() !== "" && isCanonicalAssetId(assetId);

	const columns = useMemo(
		() =>
			buildEventColumns(navigate, (ev) => {
				setPayloadEvent(ev);
				setPayloadOpen(true);
			}),
		[navigate],
	);

	const fetchEvents = useCallback(async () => {
		const fetchId = ++fetchIdRef.current;
		setLoading(true);
		setError(null);
		setNextCursor(undefined);
		setEvents([]);
		try {
			if (isPerAsset) {
				const res = await assetsApi.listEvents(assetId.trim(), {
					event_type: eventTypeFilter,
					limit: 100,
				});
				if (fetchId !== fetchIdRef.current) return;
				setEvents(res.items);
				setNextCursor(res.next_cursor);
			} else {
				const res = await assetsApi.listGlobalEvents({
					event_type: eventTypeFilter,
					limit: 100,
				});
				if (fetchId !== fetchIdRef.current) return;
				setEvents(res.items);
				setNextCursor(res.next_cursor);
			}
		} catch (err: unknown) {
			if (fetchId !== fetchIdRef.current) return;
			let m = "Failed to fetch events";
			if (err instanceof Error) {
				m = err.message;
				if (
					isPerAsset &&
					(m.includes("404") || m.includes("status code 404"))
				) {
					m = `Asset ${assetId.trim()} 不存在或暂无事件`;
				}
			}
			setError(m);
			setEvents([]);
		} finally {
		if (fetchId === fetchIdRef.current) {
				setLoading(false);
			}
		}
	}, [assetId, isPerAsset, eventTypeFilter]);

	const closeEventStream = useCallback(() => {
		streamCleanupRef.current?.();
		streamCleanupRef.current = null;
		setStreamConnected(false);
	}, []);

	const streamCleanupRef = useRef<(() => void) | null>(null);

	const loadMore = useCallback(async () => {
		if (nextCursor == null || loadingMore) return;
		setLoadingMore(true);
		try {
			const res = await assetsApi.listGlobalEvents({
				event_type: eventTypeFilter,
				cursor: nextCursor,
				limit: 100,
			});
			setEvents((prev) => [...prev, ...res.items]);
			setNextCursor(res.next_cursor);
		} catch (err: unknown) {
			const m = err instanceof Error ? err.message : "Failed to load more";
			msgApi.error(m);
		} finally {
			setLoadingMore(false);
		}
	}, [nextCursor, loadingMore, eventTypeFilter, msgApi]);

	useEffect(() => {
		if (!isPerAsset) {
			setRealtimeMode(false);
			return;
		}
		if (!realtimeMode) {
			closeEventStream();
			return;
		}
		const trimmed = assetId.trim();
		if (!isCanonicalAssetId(trimmed)) return;

		const cleanup = assetsApi.streamForAsset(trimmed, {
			onOpen: () => {
				setError(null);
				setStreamConnected(true);
			},
			onEvent: (evt) => {
				setEvents((prev) => {
					if (
						prev.some(
							(row) =>
								row.event_seq === evt.event_seq &&
								row.event_id === evt.event_id,
						)
					)
						return prev;
					return [evt, ...prev];
				});
			},
			onError: () => {
				setStreamConnected(false);
			},
		});
		streamCleanupRef.current = cleanup;
		setStreamConnected(false);
		return () => {
			cleanup();
			streamCleanupRef.current = null;
			setStreamConnected(false);
		};
	}, [assetId, isPerAsset, realtimeMode, closeEventStream]);

	useEffect(() => {
		const trimmed = assetId.trim();
		if (trimmed && !isCanonicalAssetId(trimmed)) {
			setLoading(false);
			return;
		}
		if (realtimeMode) {
			return;
		}
		setEvents([]);
		setNextCursor(undefined);
		setError(null);
		fetchEvents();
	}, [fetchEvents, realtimeMode]);

	useEffect(() => {
		if (!searchParams.has("asset_id")) return;
		setAssetId(initialAssetIdFromSearch(searchParams));
	}, [searchParams]);

	const onAssetIdInput = (value: string) => {
		setAssetId(value);
		if (realtimeMode) {
			setRealtimeMode(false);
			closeEventStream();
		}
		setLoading(true);
		setEvents([]);
		setNextCursor(undefined);
		setError(null);
		const t = value.trim();
		if (!t) {
			setLoading(true);
			setEvents([]);
			setNextCursor(undefined);
			setError(null);
		}
		if (t && isCanonicalAssetId(t)) {
			setSearchParams({ asset_id: t }, { replace: true });
		} else if (!t) {
			setSearchParams({}, { replace: true });
		}
	};

	return (
		<div style={{ padding: 24 }}>
			{msgCtx}
			<Title level={3}>事件流总览</Title>
			<Text type="secondary" style={{ marginBottom: 16, display: "block" }}>
				默认展示最近 24 小时全局事件。输入 8 位 Asset ID
				后可切换为单资产视图（支持 cursor 翻页）。
			</Text>

			{error && (
				<Alert
					type="error"
					showIcon
					closable
					style={{ marginBottom: 16 }}
					message={error}
				/>
			)}

			<Card size="small" style={{ marginBottom: 16 }}>
				<Space wrap>
					<div
						style={{
							display: "flex",
							alignItems: "center",
							gap: 8,
							width: 240,
							padding: "0 10px",
							borderRadius: 6,
							border: `1px solid ${
								assetId.trim() !== "" && !isCanonicalAssetId(assetId)
									? "#fa541c"
									: "var(--color-border, #d9d9d9)"
							}`,
							background: "#fff",
						}}
					>
						<SearchOutlined style={{ color: "#aaa" }} />
						<input
							type="text"
							placeholder="输入 Asset ID 筛选"
							value={assetId}
							onChange={(e) => onAssetIdInput(e.target.value)}
							style={{
								border: "none",
								outline: "none",
								flex: 1,
								fontSize: 14,
								fontFamily: "monospace",
								padding: "8px 0",
							}}
						/>
					</div>
				{isPerAsset && (
					<Radio.Group
						value={realtimeMode ? "realtime" : "manual"}
						onChange={(e) => setRealtimeMode(e.target.value === "realtime")}
						optionType="button"
						size="small"
						options={[
							{ label: "手动刷新", value: "manual" },
							{ label: "实时模式", value: "realtime" },
						]}
					/>
				)}
				{isPerAsset && realtimeMode ? (
					<Tag color={streamConnected ? "green" : "red"}>
						{streamConnected ? "🟢 实时" : "🔴 断开"}
					</Tag>
				) : null}
				{isPerAsset && (
					<Button
						icon={<ReloadOutlined />}
						onClick={() => onAssetIdInput("")}
						size="small"
					>
						清除资产
					</Button>
				)}
				<Button
					icon={<ReloadOutlined />}
					onClick={fetchEvents}
					loading={loading}
					size="small"
					disabled={realtimeMode}
				>
					刷新
				</Button>
				</Space>
			</Card>

			<Spin spinning={loading}>
				{loading && !assetId.trim() ? null : (
					<Table
						rowKey="event_seq"
						columns={columns}
						dataSource={events}
						size="small"
						pagination={false}
						locale={{ emptyText: loading ? " " : "暂无事件" }}
					/>
				)}
			</Spin>

			{!isPerAsset && nextCursor != null && (
				<div style={{ textAlign: "center", marginTop: 16 }}>
					<Button loading={loadingMore} onClick={loadMore} size="small">
						加载更多
					</Button>
				</div>
			)}

			<Modal
				open={payloadOpen}
				title="事件详情"
				onCancel={() => setPayloadOpen(false)}
				footer={null}
				width={640}
			>
				{payloadEvent && (
					<div>
						<p>
							<Tag color={eventTypeColor(payloadEvent.event_type)}>
								{payloadEvent.event_type}
							</Tag>
							<Text style={{ marginLeft: 8, fontFamily: "monospace" }}>
								seq={payloadEvent.event_seq}
							</Text>
						</p>
						<pre
							style={{
								background: "#f5f5f5",
								padding: 12,
								borderRadius: 6,
								maxHeight: 400,
								overflow: "auto",
								fontSize: 12,
							}}
						>
							{formatJSON(payloadEvent.event_payload)}
						</pre>
					</div>
				)}
			</Modal>
		</div>
	);
}
