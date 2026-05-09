import { Button, Card, Empty, Tag, Timeline, Typography } from "antd";
import dayjs from "dayjs";
import { Link } from "react-router-dom";
import type { AssetEvent } from "../../api/types";

const { Text } = Typography;

interface Props {
	/** When set, shows a shortcut to the global events page scoped to this asset. */
	assetId?: string;
	events: AssetEvent[];
	loading?: boolean;
	hasMore?: boolean;
	onLoadMore?: () => void;
}

function payloadString(v: unknown): string {
	if (
		typeof v === "string" ||
		typeof v === "number" ||
		typeof v === "boolean"
	) {
		return String(v);
	}
	return "";
}

function summarizeEvent(event: AssetEvent): string {
	const payload = event.event_payload ?? {};
	switch (event.event_type) {
		case "asset_created":
			return payloadString(payload.segment_locator) || "资产已创建";
		case "asset_updated":
			return payloadString(payload.owner) || "资产元数据更新";
		case "asset_lifecycle_changed": {
			const prev =
				payloadString(payload.prev_lifecycle_state) ||
				payloadString(payload.prev_status) ||
				"—";
			const next =
				payloadString(payload.new_lifecycle_state) ||
				payloadString(payload.new_status) ||
				"—";
			return `${prev} → ${next}`;
		}
		case "tag_upserted": {
			const key = payloadString(payload.tag_key);
			const value = payloadString(payload.tag_value);
			return key && value ? `${key} = ${value}` : key || "标签已更新";
		}
		case "tag_deleted": {
			const key = payloadString(payload.tag_key);
			return key ? `删除 ${key}` : "标签已删除";
		}
		case "algo_started":
		case "algo_finished":
		case "algo_failed":
		case "algo_reset":
		case "algo_unblocked": {
			const algoKey = payloadString(payload.algo_key);
			const prev = payloadString(payload.prev_status);
			const next = payloadString(payload.new_status);
			if (algoKey && (prev || next)) {
				return `${algoKey}: ${prev || "—"} → ${next || "—"}`;
			}
			return algoKey || "算法事件";
		}
		default:
			return (
				payloadString(payload.tag_key) ||
				payloadString(payload.algo_key) ||
				"事件详情见 payload"
			);
	}
}

export default function AssetEventsTab({
	assetId,
	events,
	loading = false,
	hasMore = false,
	onLoadMore,
}: Props) {
	if (events.length === 0) {
		return (
			<Card size="small" title="全部事件">
				<Empty description="暂无事件" image={Empty.PRESENTED_IMAGE_SIMPLE}>
					{assetId ? (
						<Link to={`/events?asset_id=${encodeURIComponent(assetId)}`}>
							在事件流页查看（表格视图）
						</Link>
					) : null}
				</Empty>
			</Card>
		);
	}

	return (
		<Card
			size="small"
			title="全部事件"
			extra={
				assetId ? (
					<Link to={`/events?asset_id=${encodeURIComponent(assetId)}`}>
						事件流页打开
					</Link>
				) : null
			}
		>
			<Timeline
				items={events.map((event) => ({
					color: event.event_type.includes("failed")
						? "red"
						: event.event_type.includes("deleted")
							? "orange"
							: event.event_type.includes("created") ||
									event.event_type.includes("upserted")
								? "green"
								: "blue",
					children: (
						<div>
							<div className="flex items-center gap-2">
								<Tag>{event.event_type}</Tag>
								<Text className="text-xs">{summarizeEvent(event)}</Text>
							</div>
							<div>
								<Text type="secondary" className="text-xs">
									seq={event.event_seq} ·{" "}
									{dayjs(event.created_at).format("MM-DD HH:mm:ss")}
								</Text>
							</div>
						</div>
					),
				}))}
			/>
			{hasMore && (
				<div className="mt-3">
					<Button size="small" onClick={onLoadMore} loading={loading}>
						加载更多
					</Button>
				</div>
			)}
		</Card>
	);
}
