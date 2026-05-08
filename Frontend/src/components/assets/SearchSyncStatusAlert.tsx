import { Alert, Button, Modal, Typography } from "antd";
import { useEffect, useState } from "react";
import { adminApi, type SearchAuditResult } from "../../api/admin";
import { type SearchSyncStatusResponse, searchApi } from "../../api/search";

const { Text } = Typography;

/**
 * Banner on the assets discovery page: ES availability and CDC vs dev reconcile path.
 */
export default function SearchSyncStatusAlert() {
	const [status, setStatus] = useState<SearchSyncStatusResponse | null>(null);
	const [loadError, setLoadError] = useState(false);
	const [auditOpen, setAuditOpen] = useState(false);
	const [auditLoading, setAuditLoading] = useState(false);
	const [auditError, setAuditError] = useState<string | null>(null);
	const [audit, setAudit] = useState<SearchAuditResult | null>(null);

	useEffect(() => {
		let cancelled = false;

		const pull = async () => {
			try {
				const s = await searchApi.fetchSyncStatus();
				if (!cancelled) {
					setStatus(s);
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

	if (status.search_index_mode === "cdc") {
		return (
			<>
				<Alert
					type="success"
					showIcon
					style={{ marginBottom: 12 }}
					message="搜索索引：CDC 同步"
					description="资产变更经 CDC 写入 Elasticsearch，通常为秒级～分钟级可在关键词搜索中看到。"
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
					description="未启用 CDC 时由后台任务定期从 PostgreSQL 写入 ES（约每 30 秒），新建资产后关键词搜索可能略有延迟。"
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
				description="当前环境未走 CDC 自动建索引，关键词结果可能过时；需通过管理接口或运维脚本重建索引。"
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
