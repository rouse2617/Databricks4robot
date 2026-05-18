import { Alert, Button, Modal, Tooltip, Typography } from "antd";
import { useEffect, useState } from "react";
import { adminApi, type SearchAuditResult } from "../../api/admin";
import {
	type SearchSyncProgressResponse,
	type SearchSyncStatusResponse,
	searchApi,
} from "../../api/search";

const { Text } = Typography;

/**
 * Search index sync status and PG↔ES audit (settings page; formerly on assets discovery).
 */
export default function SearchSyncStatusAlert() {
	const [status, setStatus] = useState<SearchSyncStatusResponse | null>(null);
	const [progress, setProgress] = useState<SearchSyncProgressResponse | null>(
		null,
	);
	const [loadError, setLoadError] = useState(false);
	const [auditOpen, setAuditOpen] = useState(false);
	const [auditLoading, setAuditLoading] = useState(false);
	const [auditError, setAuditError] = useState<string | null>(null);
	const [audit, setAudit] = useState<SearchAuditResult | null>(null);

	useEffect(() => {
		let cancelled = false;

		const pull = async () => {
			try {
				const [s, p] = await Promise.all([
					searchApi.fetchSyncStatus(),
					searchApi.fetchSyncProgress(),
				]);
				if (!cancelled) {
					setStatus(s);
					setProgress(p);
					setLoadError(false);
				}
			} catch {
				if (!cancelled) setLoadError(true);
			}
		};

		void pull();
		const id = window.setInterval(() => void pull(), 60_000);
		return () => {
			cancelled = true;
			window.clearInterval(id);
		};
	}, []);

	if (loadError && status == null) {
		return (
			<Alert
				type="warning"
				showIcon
				style={{ marginBottom: 12 }}
				message="无法获取搜索索引状态"
				description="请检查网络或登录态；关键词搜索仍可能可用。"
			/>
		);
	}

	if (status == null) return null;

	const pgMaxEventSeq = progress?.pg_max_event_seq ?? 0;
	const outboxPublishedMaxSeq = progress?.outbox_published_max_seq ?? 0;
	const seqLag = progress?.seq_lag ?? 0;
	const esAppliedMinSeq = progress?.es_applied_min_seq ?? 0;
	const consumerLag = progress?.consumer_lag ?? 0;

	const progressDescription = progress ? (
		<div style={{ marginTop: 8 }}>
			<div>
				<Text strong>PG / ES：</Text>
				<Text>
					{progress.postgres_assets_total.toLocaleString()} /{" "}
					{progress.elasticsearch_docs_total.toLocaleString()}
				</Text>
			</div>
			<div>
				<Text strong>同步比例：</Text>
				<Text>{(progress.pg_es_sync_ratio * 100).toFixed(2)}%</Text>
				<Text type="secondary">
					{" "}
					（Gap {progress.pg_es_gap.toLocaleString()}）
				</Text>
			</div>
			<div>
				<Text strong>Outbox：</Text>
				<Tooltip
					title={
						<div style={{ maxWidth: 360 }}>
							<p style={{ margin: "0 0 8px 0" }}>
								<code>pending</code> 总数里包含「写入后约 safety lag
								秒内还不能被 relay 认领」的事件，这是预期行为，不是积压卡死。
							</p>
							<p style={{ margin: 0 }}>
								若「可认领」长期很大或最老 pending 年龄持续飙高，才需要排查
								relay / ES / PG。
							</p>
						</div>
					}
				>
					<Text style={{ borderBottom: "1px dashed rgba(0,0,0,0.25)" }}>
						可认领 {progress.outbox_pending_claimable.toLocaleString()}
					</Text>
				</Tooltip>
				<Text type="secondary">
					{" "}
					· 写入缓冲约{" "}
					{Math.max(
						0,
						progress.outbox_pending_events - progress.outbox_pending_claimable,
					).toLocaleString()}{" "}
					条（relay lag {progress.outbox_relay_safety_lag_sec}s） · 投递中{" "}
					{progress.outbox_processing_events.toLocaleString()}
				</Text>
				<div style={{ marginTop: 4 }}>
					<Text type="secondary" className="text-xs">
						pending 合计 {progress.outbox_pending_events.toLocaleString()} ·
						最老 pending {Math.round(progress.oldest_pending_age_sec)}s
					</Text>
				</div>
			</div>
			<div style={{ marginTop: 6 }}>
				<Text strong>PG→MQ 水位：</Text>
				<Text>
					PG {pgMaxEventSeq.toLocaleString()} · MQ{" "}
					{outboxPublishedMaxSeq.toLocaleString()}
				</Text>
				<Text type="secondary"> （seq lag {seqLag.toLocaleString()}）</Text>
			</div>
			<div>
				<Text strong>MQ→ES 水位：</Text>
				<Text>
					ES min {esAppliedMinSeq.toLocaleString()} · consumer lag{" "}
					{consumerLag.toLocaleString()}
				</Text>
				{esAppliedMinSeq === 0 ? (
					<Text type="secondary">
						{" "}
						（checkpoint 预热中：分片未报齐前会保持 0）
					</Text>
				) : null}
			</div>
		</div>
	) : null;

	const runAudit = async () => {
		setAuditOpen(true);
		setAuditLoading(true);
		setAuditError(null);
		try {
			const r = await adminApi.auditSearch();
			setAudit(r);
		} catch (e) {
			setAudit(null);
			setAuditError(e instanceof Error ? e.message : "对账失败");
		} finally {
			setAuditLoading(false);
		}
	};

	const auditAction = (
		<Button size="small" onClick={() => void runAudit()} loading={auditLoading}>
			PG↔ES 对账
		</Button>
	);

	if (!status.elasticsearch_ok) {
		return (
			<>
				<Alert
					type="error"
					showIcon
					style={{ marginBottom: 12 }}
					message="Elasticsearch 未连接"
					description="关键词搜索不可用，可切换到「结构化」模式使用 PostgreSQL 列表。"
					action={auditAction}
				/>
				<AuditModal
					open={auditOpen}
					onClose={() => setAuditOpen(false)}
					loading={auditLoading}
					error={auditError}
					audit={audit}
				/>
			</>
		);
	}

	if (status.search_index_mode === "outbox_es_subscriber") {
		return (
			<>
				<Alert
					type="success"
					showIcon
					style={{ marginBottom: 12 }}
					message="搜索索引：Outbox 自动同步"
					description={
						<>
							<div>
								资产事件通过 PG Outbox → ES Subscriber 增量写入搜索索引。
							</div>
							{progressDescription}
						</>
					}
					action={auditAction}
				/>
				<AuditModal
					open={auditOpen}
					onClose={() => setAuditOpen(false)}
					loading={auditLoading}
					error={auditError}
					audit={audit}
				/>
			</>
		);
	}

	if (status.search_index_mode === "local_reconcile") {
		return (
			<>
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 12 }}
					message="搜索索引：开发环境定时对齐"
					description={
						<>
							<div>
								未启用 Outbox 订阅时由后台任务定期从 PostgreSQL 写入 ES（约每 30
								秒）。
							</div>
							{progressDescription}
						</>
					}
					action={auditAction}
				/>
				<AuditModal
					open={auditOpen}
					onClose={() => setAuditOpen(false)}
					loading={auditLoading}
					error={auditError}
					audit={audit}
				/>
			</>
		);
	}

	return (
		<>
			<Alert
				type="warning"
				showIcon
				style={{ marginBottom: 12 }}
				message="搜索索引：无自动同步"
				description={
					<>
						<div>
							当前环境未启用自动建索引，关键词结果可能过时；需通过管理接口或运维脚本重建索引。
						</div>
						{progressDescription}
					</>
				}
				action={auditAction}
			/>
			<AuditModal
				open={auditOpen}
				onClose={() => setAuditOpen(false)}
				loading={auditLoading}
				error={auditError}
				audit={audit}
			/>
		</>
	);
}

function AuditModal(props: {
	open: boolean;
	onClose: () => void;
	loading: boolean;
	error: string | null;
	audit: SearchAuditResult | null;
}) {
	const { open, onClose, loading, error, audit } = props;
	return (
		<Modal
			open={open}
			onCancel={onClose}
			onOk={onClose}
			title="PG↔ES 对账"
			okText="关闭"
			cancelButtonProps={{ style: { display: "none" } }}
		>
			{loading && <Text type="secondary">对账中…</Text>}
			{!loading && error && <Text type="danger">{error}</Text>}
			{!loading && !error && audit && (
				<div style={{ display: "grid", gap: 8 }}>
					<div>
						<Text strong>PG 活跃资产：</Text> <Text>{audit.pg_assets}</Text>
					</div>
					<div>
						<Text strong>ES 文档：</Text>{" "}
						<Text>{audit.elasticsearch_docs}</Text>
					</div>
					<div>
						<Text strong>PG→ES 缺失：</Text>{" "}
						<Text>{audit.missing_in_elasticsearch}</Text>
					</div>
					<div>
						<Text strong>ES 孤儿：</Text>{" "}
						<Text>{audit.orphan_in_elasticsearch}</Text>
					</div>
					<div>
						<Text strong>一致率：</Text>{" "}
						<Text>
							{(audit.consistency * 100).toFixed(3)}%（目标{" "}
							{(audit.target * 100).toFixed(2)}%）
						</Text>
					</div>
					{(audit.sample_missing_ids?.length ?? 0) > 0 && (
						<div>
							<Text strong>缺失样例：</Text>
							<pre
								style={{
									margin: "6px 0 0 0",
									maxHeight: 140,
									overflow: "auto",
								}}
							>
								{(audit.sample_missing_ids ?? []).join("\n")}
							</pre>
						</div>
					)}
					{(audit.sample_orphan_ids?.length ?? 0) > 0 && (
						<div>
							<Text strong>孤儿样例：</Text>
							<pre
								style={{
									margin: "6px 0 0 0",
									maxHeight: 140,
									overflow: "auto",
								}}
							>
								{(audit.sample_orphan_ids ?? []).join("\n")}
							</pre>
						</div>
					)}
					<Text type="secondary">耗时 {audit.duration_ms} ms</Text>
				</div>
			)}
		</Modal>
	);
}
