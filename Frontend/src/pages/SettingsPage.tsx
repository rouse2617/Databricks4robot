import {
	CloudSyncOutlined,
	ExclamationCircleOutlined,
	SettingOutlined,
	ThunderboltOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Descriptions,
	Modal,
	Progress,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	adminApi,
	type ReindexJob,
	type SearchAuditResult,
} from "../api/admin";
import { type SearchSyncStatusResponse, searchApi } from "../api/search";
import BronzeSyncStatusAlert from "../components/assets/BronzeSyncStatusAlert";
import SearchSyncStatusAlert from "../components/assets/SearchSyncStatusAlert";
import { useAuth } from "../hooks/useAuth";
import { extractApiErrorMessage } from "../lib/apiError";

const { Title, Text } = Typography;
const SEARCH_ADMIN_NOTICE_STYLE = { marginBottom: 16, minHeight: 72 } as const;

type ReindexPhase = "idle" | "reindexing" | "paused" | "done" | "error";

const SETTINGS_REINDEX_STATE_KEY = "settings_reindex_state_v1";
const REINDEX_POLL_INTERVAL_MS = 2500;

type PersistedReindexState = {
	phase: ReindexPhase;
	runJob: ReindexJob | null;
	error: string | null;
	savedAt: number;
};

function loadPersistedState(): PersistedReindexState | null {
	if (typeof window === "undefined") return null;
	const storage = window.localStorage;
	if (!storage || typeof storage.getItem !== "function") return null;
	try {
		const raw = storage.getItem(SETTINGS_REINDEX_STATE_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as PersistedReindexState;
		// Expire persisted reindex state after 5 minutes to prevent
		// stale state from surviving browser restarts or tab switches.
		if (Date.now() - (parsed.savedAt ?? 0) > 5 * 60 * 1000) {
			storage.removeItem(SETTINGS_REINDEX_STATE_KEY);
			return null;
		}
		return parsed;
	} catch {
		return null;
	}
}

export default function SettingsPage() {
	const { isAuthenticated } = useAuth();
	const persisted = loadPersistedState();
	const [phase, setPhase] = useState<ReindexPhase>(persisted?.phase ?? "idle");
	const [runJob, setRunJob] = useState<ReindexJob | null>(
		persisted?.runJob ?? null,
	);
	const [error, setError] = useState<string | null>(persisted?.error ?? null);
	const [confirmOpen, setConfirmOpen] = useState(false);
	const [confirmLoading, setConfirmLoading] = useState(false);
	const [auditResult, setAuditResult] = useState<SearchAuditResult | null>(
		null,
	);
	const [auditError, setAuditError] = useState<string | null>(null);
	const [pollJobId, setPollJobId] = useState<string | null>(null);
	const [history, setHistory] = useState<ReindexJob[]>([]);
	const [historyLoading, setHistoryLoading] = useState(false);
	const [historyError, setHistoryError] = useState<string | null>(null);
	const [searchStatus, setSearchStatus] =
		useState<SearchSyncStatusResponse | null>(null);
	const [searchStatusLoaded, setSearchStatusLoaded] = useState(false);

	const adminSearchEnabled = searchStatus?.admin_search_enabled === true;

	const loadHistory = useCallback(async () => {
		if (!adminSearchEnabled) return;
		setHistoryLoading(true);
		setHistoryError(null);
		try {
			const items = await adminApi.listReindexJobs(20);
			setHistory(items);
			// If this tab is idle but the server has an active non-dry job (e.g. created by
			// another tab), adopt it so the user sees progress instead of starting again.
			if (phase === "idle") {
				const active = items.find(
					(j) =>
						!j.dry_run && (j.status === "queued" || j.status === "running"),
				);
				if (active) {
					setRunJob(active);
					setPhase("reindexing");
					setPollJobId(active.id);
				}
			}
		} catch (err) {
			setHistoryError(extractApiErrorMessage(err, "加载任务历史失败"));
		} finally {
			setHistoryLoading(false);
		}
	}, [adminSearchEnabled, phase]);

	useEffect(() => {
		let cancelled = false;
		searchApi
			.fetchSyncStatus()
			.then((status) => {
				if (cancelled) return;
				setSearchStatus(status);
			})
			.catch(() => {
				if (!cancelled) setSearchStatus(null);
			})
			.finally(() => {
				if (!cancelled) setSearchStatusLoaded(true);
			});
		return () => {
			cancelled = true;
		};
	}, []);

	useEffect(() => {
		if (adminSearchEnabled) loadHistory();
	}, [adminSearchEnabled, loadHistory]);

	useEffect(() => {
		if (!adminSearchEnabled) return;
		if (phase === "done" || phase === "error" || phase === "paused") {
			loadHistory();
		}
	}, [adminSearchEnabled, phase, loadHistory]);

	useEffect(() => {
		if (typeof window === "undefined") return;
		const storage = window.localStorage;
		if (!storage || typeof storage.setItem !== "function") return;
		const payload: PersistedReindexState = {
			phase,
			runJob,
			error,
			savedAt: Date.now(),
		};
		storage.setItem(SETTINGS_REINDEX_STATE_KEY, JSON.stringify(payload));
	}, [phase, runJob, error]);

	useEffect(() => {
		if (!pollJobId) return;
		let cancelled = false;
		const tick = async () => {
			try {
				const job = await adminApi.getReindexJob(pollJobId);
				if (cancelled) return;
				setRunJob(job);
				switch (job.status) {
					case "succeeded":
						setPhase("done");
						setPollJobId(null);
						break;
					case "paused":
						setPhase("paused");
						setPollJobId(null);
						break;
					case "failed":
						setPhase("error");
						setError(job.error || "重建索引失败");
						setPollJobId(null);
						break;
					default:
						setPhase("reindexing");
						break;
				}
			} catch (err) {
				if (!cancelled) {
					setError(extractApiErrorMessage(err, "读取任务状态失败"));
					setPhase("error");
					setPollJobId(null);
				}
			}
		};
		void tick();
		const timer = window.setInterval(
			() => void tick(),
			REINDEX_POLL_INTERVAL_MS,
		);
		return () => {
			cancelled = true;
			window.clearInterval(timer);
		};
	}, [pollJobId]);

	useEffect(() => {
		if (phase !== "reindexing") return;
		if (!runJob?.id) return;
		if (!pollJobId) {
			setPollJobId(runJob.id);
		}
	}, [phase, runJob, pollJobId]);

	const handleOpenConfirm = async () => {
		if (!adminSearchEnabled) return;
		setConfirmOpen(true);
		setAuditError(null);
		setAuditResult(null);
		try {
			const result = await adminApi.auditSearch();
			setAuditResult(result);
		} catch (err) {
			setAuditError(extractApiErrorMessage(err, "对账查询失败"));
		}
	};

	const handleCloseConfirm = () => {
		setConfirmOpen(false);
	};

	const handleReindexSubmit = async () => {
		if (!adminSearchEnabled) return;
		setConfirmLoading(true);
		setError(null);
		try {
			const job = await adminApi.createReindexJob(false);
			setRunJob(job);
			setPhase("reindexing");
			setPollJobId(job.id);
			setConfirmOpen(false);
		} catch (err) {
			setError(extractApiErrorMessage(err, "重建索引失败"));
			setPhase("error");
			setConfirmOpen(false);
		} finally {
			setConfirmLoading(false);
		}
	};

	const handleStop = async () => {
		if (!runJob?.id) return;
		try {
			const job = await adminApi.stopReindexJob(runJob.id);
			setRunJob(job);
			if (job.status === "paused") {
				setPhase("paused");
				setPollJobId(null);
				return;
			}
			setPhase("reindexing");
			setPollJobId(job.id);
		} catch (err) {
			setError(extractApiErrorMessage(err, "停止任务失败"));
			setPhase("error");
		}
	};

	const handleResume = async () => {
		if (!runJob?.id) return;
		try {
			const job = await adminApi.resumeReindexJob(runJob.id);
			setError(null);
			setRunJob(job);
			setPhase("reindexing");
			setPollJobId(job.id);
		} catch (err) {
			setError(extractApiErrorMessage(err, "续开任务失败"));
			setPhase("error");
		}
	};

	const handleReset = () => {
		setPhase("idle");
		setRunJob(null);
		setError(null);
		setPollJobId(null);
		setConfirmOpen(false);
	};

	return (
		<div>
			<Title level={4} style={{ margin: 0, marginBottom: 16 }}>
				<SettingOutlined style={{ marginRight: 8 }} />
				设置
			</Title>
			<Card title="当前会话" size="small" style={{ marginBottom: 16 }}>
				<Descriptions column={1} size="small">
					<Descriptions.Item label="认证方式">
						Cookie Session (HttpOnly)
					</Descriptions.Item>
					<Descriptions.Item label="状态">
						<code className="text-xs">
							{isAuthenticated ? "已登录" : "未登录"}
						</code>
					</Descriptions.Item>
					<Descriptions.Item label="API 地址">/api/v1</Descriptions.Item>
				</Descriptions>
			</Card>

			<Card
				title={
					<span>
						<CloudSyncOutlined style={{ marginRight: 8 }} />
						搜索索引状态
					</span>
				}
				size="small"
				style={{ marginBottom: 16 }}
				data-testid="search-sync-card"
			>
				<Text type="secondary" style={{ display: "block", marginBottom: 12 }}>
					服务端返回的 CDC / 定时对齐 / 手动模式说明，以及 PG 与 ES
					文档数的对账。若关键词结果明显滞后，可在下方「重建 ES
					索引」做全量修复。
				</Text>
				<SearchSyncStatusAlert adminSearchEnabled={adminSearchEnabled} />
				<BronzeSyncStatusAlert />
			</Card>

			{!searchStatusLoaded && (
				<Alert
					type="info"
					showIcon
					style={SEARCH_ADMIN_NOTICE_STYLE}
					message="正在加载搜索管理配置"
					description="正在确认当前环境是否开放 ES 重建与对账管理接口。"
				/>
			)}

			{searchStatusLoaded && !adminSearchEnabled && (
				<Alert
					type="info"
					showIcon
					style={SEARCH_ADMIN_NOTICE_STYLE}
					message="搜索管理工具未启用"
					description="当前环境未开放 ES 重建与对账管理接口；搜索状态仍可查看。"
				/>
			)}

			{adminSearchEnabled && (
				<>
					<Card
						title={
							<span>
								<ThunderboltOutlined style={{ marginRight: 8 }} />
								重建 ES 索引
							</span>
						}
						size="small"
						data-testid="reindex-card"
					>
						<Text
							type="secondary"
							style={{ display: "block", marginBottom: 12 }}
						>
							当 Elasticsearch 索引与 PostgreSQL
							数据不一致时，可手动触发全量重建。点击下方按钮会先做一次 PG↔ES
							对账，再二次确认是否执行。
						</Text>

						{error && (
							<Alert
								type="error"
								showIcon
								message={error}
								style={{ marginBottom: 12 }}
								closable
								onClose={() => setError(null)}
							/>
						)}

						{runJob &&
							(phase === "reindexing" ||
								phase === "paused" ||
								phase === "done" ||
								phase === "error") && (
								<div style={{ marginBottom: 12 }}>
									<Progress
										percent={Number(runJob.progress_pct.toFixed(1))}
										status={phase === "error" ? "exception" : "active"}
									/>
									<div style={{ marginTop: 8 }}>
										<Text type="secondary">
											状态：{runJob.status} · 扫描 {runJob.assets_scanned}/
											{runJob.total_assets || "?"}· 索引{" "}
											{runJob.documents_indexed} · 删除{" "}
											{runJob.documents_deleted} · 失败 {runJob.failed}
										</Text>
									</div>
								</div>
							)}

						{/* Actual reindex result */}
						{runJob && phase === "done" && (
							<Alert
								type="success"
								showIcon
								message="索引重建完成"
								description={
									<div>
										<p>总资产数：{runJob.total_assets}</p>
										<p>成功索引：{runJob.documents_indexed}</p>
										<p>删除文档：{runJob.documents_deleted}</p>
										<p>失败：{runJob.failed}</p>
									</div>
								}
								style={{ marginBottom: 12 }}
							/>
						)}

						<Space>
							{phase === "idle" && (
								<Button
									type="primary"
									danger
									onClick={handleOpenConfirm}
									data-testid="reindex-open-confirm-btn"
								>
									重建 ES 索引
								</Button>
							)}
							{phase === "reindexing" && (
								<Button onClick={handleStop} data-testid="reindex-stop-btn">
									停止任务
								</Button>
							)}
							{runJob &&
								(phase === "paused" ||
									(phase === "error" && runJob.status === "failed")) && (
									<Button
										type="primary"
										onClick={handleResume}
										data-testid="reindex-resume-btn"
									>
										断点续开
									</Button>
								)}
							{(phase === "done" ||
								phase === "error" ||
								phase === "paused") && (
								<Button onClick={handleReset}>重置</Button>
							)}
						</Space>

						{confirmOpen && (
							<Modal
								title={
									<span>
										<ExclamationCircleOutlined
											style={{
												color: "var(--color-warning, #faad14)",
												marginRight: 8,
											}}
										/>
										确认重建 ES 索引
									</span>
								}
								open={confirmOpen}
								okText="确认重建"
								cancelText="取消"
								okButtonProps={{
									danger: true,
									disabled: !auditResult,
								}}
								confirmLoading={confirmLoading}
								onOk={handleReindexSubmit}
								onCancel={handleCloseConfirm}
								footer={(_, { OkBtn }) => (
									<>
										<Button
											onClick={handleCloseConfirm}
											data-testid="reindex-cancel-btn"
										>
											取消
										</Button>
										<OkBtn />
									</>
								)}
								destroyOnHidden
							>
								<div>
									{auditError && (
										<Alert
											type="error"
											message={auditError}
											style={{ marginBottom: 12 }}
										/>
									)}
									{!auditResult && !auditError && (
										<p>正在查询 PG↔ES 对账数据...</p>
									)}
									{auditResult && (
										<Descriptions
											column={1}
											size="small"
											style={{ marginBottom: 12 }}
										>
											<Descriptions.Item label="PG 资产数">
												<strong>
													{auditResult.pg_assets.toLocaleString()}
												</strong>
											</Descriptions.Item>
											<Descriptions.Item label="ES 文档数">
												<strong>
													{auditResult.elasticsearch_docs.toLocaleString()}
												</strong>
											</Descriptions.Item>
											<Descriptions.Item label="缺失（PG 有 ES 无）">
												<Text type="warning">
													{auditResult.missing_in_elasticsearch.toLocaleString()}
												</Text>
											</Descriptions.Item>
											<Descriptions.Item label="孤儿（ES 有 PG 无）">
												<Text type="warning">
													{auditResult.orphan_in_elasticsearch.toLocaleString()}
												</Text>
											</Descriptions.Item>
											<Descriptions.Item label="一致性">
												{(auditResult.consistency * 100).toFixed(2)}%
											</Descriptions.Item>
										</Descriptions>
									)}
									<p>
										重建会从 PG 全量扫描资产并写入 ES，
										以异步任务执行，可在运行中停止并断点续开。
									</p>
									<p>确定继续？</p>
								</div>
							</Modal>
						)}
					</Card>

					<Card
						title={
							<span>
								<ThunderboltOutlined style={{ marginRight: 8 }} />
								重建任务历史
							</span>
						}
						size="small"
						style={{ marginTop: 16 }}
						data-testid="reindex-history-card"
						extra={
							<Button
								size="small"
								onClick={loadHistory}
								loading={historyLoading}
							>
								刷新
							</Button>
						}
					>
						{historyError && (
							<Alert
								type="error"
								message={historyError}
								style={{ marginBottom: 12 }}
								closable
							/>
						)}
						<Table<ReindexJob>
							rowKey="id"
							size="small"
							loading={historyLoading}
							dataSource={history}
							pagination={false}
							locale={{ emptyText: "暂无重建任务" }}
							columns={[
								{
									title: "任务 ID",
									dataIndex: "id",
									width: 220,
									render: (v: string) => (
										<Tooltip title={v}>
											<code style={{ fontSize: 12 }}>{v.slice(0, 12)}…</code>
										</Tooltip>
									),
								},
								{
									title: "类型",
									dataIndex: "dry_run",
									width: 80,
									render: (dry: boolean) =>
										dry ? (
											<Tag color="blue">Dry Run</Tag>
										) : (
											<Tag color="purple">重建</Tag>
										),
								},
								{
									title: "状态",
									dataIndex: "status",
									width: 100,
									render: (s: ReindexJob["status"]) => {
										const map: Record<ReindexJob["status"], string> = {
											queued: "default",
											running: "processing",
											paused: "warning",
											succeeded: "success",
											failed: "error",
										};
										return <Tag color={map[s]}>{s}</Tag>;
									},
								},
								{
									title: "进度",
									dataIndex: "progress_pct",
									width: 140,
									render: (p: number, row) => (
										<Tooltip
											title={`扫描 ${row.assets_scanned}/${row.total_assets} · 写入 ${row.documents_indexed} · 失败 ${row.failed}`}
										>
											<Progress
												percent={Math.round(p * 10) / 10}
												size="small"
											/>
										</Tooltip>
									),
								},
								{
									title: "创建时间",
									dataIndex: "created_at",
									width: 170,
									render: (v: string) => new Date(v).toLocaleString(),
								},
								{
									title: "更新时间",
									dataIndex: "updated_at",
									width: 170,
									render: (v: string) => new Date(v).toLocaleString(),
								},
								{
									title: "错误样本",
									dataIndex: "error_samples",
									render: (samples?: string[]) => {
										if (!samples || samples.length === 0)
											return <Text type="secondary">—</Text>;
										return (
											<Tooltip title={samples.join("\n")}>
												<Text type="danger">{samples.length} 条</Text>
											</Tooltip>
										);
									},
								},
							]}
						/>
					</Card>
				</>
			)}
		</div>
	);
}
