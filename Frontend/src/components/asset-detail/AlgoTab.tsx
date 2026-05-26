import {
	CheckCircleOutlined,
	ClockCircleOutlined,
	CloseCircleOutlined,
	LockOutlined,
	PlayCircleOutlined,
	ReloadOutlined,
	RocketOutlined,
} from "@ant-design/icons";
import {
	Button,
	Card,
	Input,
	Modal,
	message,
	Popconfirm,
	Select,
	Table,
	Tag,
	Timeline,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { type AlgoRegistryItem, algoRegistryApi } from "../../api/algoRegistry";
import { assetsApi } from "../../api/assets";
import type { AlgoEvent, AlgoStatus } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";
import RunIdLink from "./RunIdLink";

const { Text } = Typography;

const algoStatusConfig: Record<
	string,
	{ color: string; icon: React.ReactNode; label: string }
> = {
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
	onJumpToPreviewTime?: (eventTimeIso: string) => void;
}

export default function AlgoTab({
	assetId,
	algoList,
	events,
	eventsLoading = false,
	hasMoreEvents = false,
	onLoadMoreEvents,
	onRefresh,
	onJumpToPreviewTime,
}: Props) {
	const [actionLoading, setActionLoading] = useState<string | null>(null);
	const [startModalOpen, setStartModalOpen] = useState(false);
	const [algoKey, setAlgoKey] = useState("");
	const [method, setMethod] = useState("");
	const [runId, setRunId] = useState("");
	const [starting, setStarting] = useState(false);
	const [registry, setRegistry] = useState<AlgoRegistryItem[]>([]);

	useEffect(() => {
		algoRegistryApi
			.list()
			.then(setRegistry)
			.catch(() => {});
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
			const trimmedRunId = runId.trim();
			await assetsApi.startAlgo(assetId, algoKey, {
				method,
				...(trimmedRunId ? { run_id: trimmedRunId } : {}),
			});
			message.success(`算法 ${algoKey} 已启动`);
			setStartModalOpen(false);
			setAlgoKey("");
			setMethod("");
			setRunId("");
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
							return (
								<Tag color={cfg.color} icon={cfg.icon}>
									{cfg.label}
								</Tag>
							);
						},
					},
					{
						title: "来自 run",
						key: "run_id",
						width: 140,
						render: (_: unknown, r: AlgoInfo) => (
							<RunIdLink runId={r.run_id} showLabel={false} />
						),
					},
					{
						title: "开始时间",
						dataIndex: "started_at",
						width: 160,
						render: (v: string) =>
							v ? dayjs(v).format("MM-DD HH:mm:ss") : "—",
					},
					{
						title: "耗时",
						key: "duration",
						width: 80,
						render: (_: unknown, r: AlgoInfo) => {
							if (!r.started_at || !r.finished_at) return "—";
							const sec = dayjs(r.finished_at).diff(
								dayjs(r.started_at),
								"second",
							);
							return `${sec}s`;
						},
					},
					{
						title: "产物 / 原因",
						key: "output",
						render: (_: unknown, r: AlgoInfo) => {
							if (r.status === "failed" && r.reason) {
								return (
									<Text type="danger" className="text-xs">
										{r.reason}
									</Text>
								);
							}
							if (r.output_uri) {
								return (
									<Text className="text-xs font-mono">
										{r.output_uri.slice(-40)}
									</Text>
								);
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
								e.new_status === "ok"
									? "green"
									: e.new_status === "failed"
										? "red"
										: e.new_status === "running"
											? "blue"
											: "gray",
							children: (
								<div>
									<Text strong className="text-xs">
										{e.algo_key}
									</Text>
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
									{onJumpToPreviewTime && (
										<Button
											size="small"
											type="link"
											style={{ paddingInline: 6 }}
											onClick={() => onJumpToPreviewTime(e.created_at)}
										>
											定位预览
										</Button>
									)}
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
					setRunId("");
				}}
				onOk={handleStartAlgo}
				confirmLoading={starting}
				okButtonProps={{ disabled: !algoKey || !method }}
				okText="启动"
				cancelText="取消"
			>
				<div className="mb-3">
					<label htmlFor="algo-key-select" className="block text-sm mb-1">
						算法 Key
					</label>
					<Select
						id="algo-key-select"
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
				<div className="mb-3">
					<label htmlFor="algo-method-input" className="block text-sm mb-1">
						Method
					</label>
					<Input
						id="algo-method-input"
						placeholder="输入 method（如 default, gpu, cpu）"
						value={method}
						onChange={(e) => setMethod(e.target.value)}
					/>
				</div>
				<div>
					<label htmlFor="algo-run-id-input" className="block text-sm mb-1">
						Run ID{" "}
						<Text type="secondary" className="text-xs">
							（可选，关联已有运行记录）
						</Text>
					</label>
					<Input
						id="algo-run-id-input"
						placeholder="输入 16 位 run_id"
						value={runId}
						onChange={(e) => setRunId(e.target.value)}
						maxLength={16}
						allowClear
					/>
				</div>
			</Modal>
		</div>
	);
}
