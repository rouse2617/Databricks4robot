import { Alert, Tooltip, Typography } from "antd";
import { useEffect, useState } from "react";
import {
	type BronzeSyncProgressResponse,
	lakehouseApi,
} from "../../api/lakehouse";

const { Text } = Typography;

// Alert thresholds match docs/review/outbox-watermark-alerts.md (Bronze section).
// Keep these in sync — they decide which Antd alert type we show.
// Scheduler runs hourly, so thresholds are sized to ~1.5 / ~3 ticks.
const STALE_WARN_SEC = 5_400; // 90 min — more than one scheduler tick (1 h)
const STALE_ERROR_SEC = 10_800; // 3 h — Bronze almost certainly broken
// Lag-event thresholds are loose by design — hourly schedule + bursty
// imports can legitimately accumulate hundreds of thousands of events
// between ticks. The authoritative health signal is stale_seconds above.
const LAG_WARN_EVENTS = 50_000;
const LAG_ERROR_EVENTS = 500_000;

function formatStale(sec: number): string {
	if (sec < 0) return "无数据";
	if (sec < 60) return `${Math.round(sec)}s`;
	if (sec < 3600) return `${Math.round(sec / 60)} 分钟`;
	if (sec < 86_400) return `${(sec / 3600).toFixed(1)} 小时`;
	return `${(sec / 86_400).toFixed(1)} 天`;
}

export default function BronzeSyncStatusAlert() {
	const [progress, setProgress] = useState<BronzeSyncProgressResponse | null>(
		null,
	);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		let cancelled = false;
		const pull = async () => {
			try {
				const p = await lakehouseApi.syncProgress();
				if (!cancelled) {
					setProgress(p);
					setError(null);
				}
			} catch (e) {
				if (!cancelled) setError(e instanceof Error ? e.message : "请求失败");
			}
		};
		void pull();
		const id = window.setInterval(() => void pull(), 60_000);
		return () => {
			cancelled = true;
			window.clearInterval(id);
		};
	}, []);

	if (error && progress == null) {
		return (
			<Alert
				type="warning"
				showIcon
				style={{ marginBottom: 12 }}
				message="无法获取湖仓同步状态"
				description="请检查 BigQuery / Postgres 连接。"
			/>
		);
	}
	if (progress == null) return null;

	const outbox_published_max_seq = progress.outbox_published_max_seq ?? 0;
	const bronze_max_event_seq = progress.bronze_max_event_seq ?? 0;
	const bronze_lag_events = progress.bronze_lag_events ?? 0;
	const bronze_stale_seconds = progress.bronze_stale_seconds ?? 0;

	const isEmpty = bronze_max_event_seq === 0;
	const alertType: "success" | "info" | "warning" | "error" = isEmpty
		? "info"
		: bronze_stale_seconds > STALE_ERROR_SEC ||
				bronze_lag_events > LAG_ERROR_EVENTS
			? "error"
			: bronze_stale_seconds > STALE_WARN_SEC ||
					bronze_lag_events > LAG_WARN_EVENTS
				? "warning"
				: "success";

	const title = isEmpty
		? "湖仓 Bronze：尚无数据"
		: alertType === "success"
			? "湖仓 Bronze：实时同步中"
			: alertType === "warning"
				? "湖仓 Bronze：同步延迟"
				: alertType === "error"
					? "湖仓 Bronze：同步异常"
					: "湖仓 Bronze";

	return (
		<Alert
			type={alertType}
			showIcon
			style={{ marginBottom: 12 }}
			message={title}
			description={
				<Tooltip
					title={
						<div style={{ maxWidth: 320 }}>
							Bronze 由 Cloud Run Job <code>bronze-incremental</code> 每小时
							增量写入；Bronze 允许 raw 重复，Silver 在 event_id 上去重。
						</div>
					}
				>
					<Text style={{ borderBottom: "1px dashed rgba(0,0,0,0.25)" }}>
						Bronze {bronze_max_event_seq.toLocaleString()} / PG{" "}
						{outbox_published_max_seq.toLocaleString()} · 滞后{" "}
						{bronze_lag_events.toLocaleString()} 事件 ·{" "}
						{formatStale(bronze_stale_seconds)}前同步
					</Text>
				</Tooltip>
			}
		/>
	);
}
