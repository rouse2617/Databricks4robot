import {
	ArrowLeftOutlined,
	CheckCircleOutlined,
	ClockCircleOutlined,
	CloseCircleOutlined,
	ExportOutlined,
	LockOutlined,
	PlayCircleOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Button,
	Card,
	Descriptions,
	message,
	Spin,
	Tag,
	Timeline,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { algoRunsApi, type AlgoRun } from "../api/algoRuns";
import { formatDateTime } from "../lib/dateTime";

const { Title, Text } = Typography;

const STATUS_COLOR: Record<string, string> = {
	pending: "default",
	running: "processing",
	ok: "success",
	failed: "error",
	cancelled: "warning",
};

const STATUS_LABEL: Record<string, string> = {
	pending: "待处理",
	running: "运行中",
	ok: "成功",
	failed: "失败",
	cancelled: "已取消",
};

const STATUS_ICON: Record<string, React.ReactNode> = {
	pending: <ClockCircleOutlined />,
	running: <PlayCircleOutlined />,
	ok: <CheckCircleOutlined />,
	failed: <CloseCircleOutlined />,
	cancelled: <LockOutlined />,
};

function formatDuration(startedAt?: string, finishedAt?: string): string {
	if (!startedAt) return "—";
	const end = finishedAt ? dayjs(finishedAt) : dayjs();
	const sec = end.diff(dayjs(startedAt), "second");
	if (sec < 60) return `${sec}s`;
	const min = Math.floor(sec / 60);
	return `${min}m ${sec % 60}s`;
}

export default function AlgoRunDetailPage() {
	const { run_id } = useParams<{ run_id: string }>();
	const navigate = useNavigate();
	const [msg, msgCtx] = message.useMessage();

	const [run, setRun] = useState<AlgoRun | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const msgRef = useRef(msg);
	msgRef.current = msg;

	const fetchRun = useCallback(async () => {
		if (!run_id) return;
		setLoading(true);
		setError(null);
		try {
			const data = await algoRunsApi.get(run_id);
			setRun(data);
		} catch {
			setError("加载运行记录失败");
		} finally {
			setLoading(false);
		}
	}, [run_id]);

	useEffect(() => {
		fetchRun();
	}, [fetchRun]);

	if (loading) {
		return (
			<div style={{ textAlign: "center", padding: 80 }}>
				<Spin size="large" />
			</div>
		);
	}

	if (error || !run) {
		return (
			<div style={{ textAlign: "center", padding: 80 }}>
				<Typography.Text type="secondary">
					{error ?? "运行记录未找到"}
				</Typography.Text>
				<div style={{ marginTop: 16 }}>
					<Button onClick={() => navigate("/algo-runs")}>返回列表</Button>
				</div>
			</div>
		);
	}

	// Build status timeline from run metadata
	const timelineItems = [];
	if (run.created_at) {
		timelineItems.push({
			color: "gray",
			children: (
				<div>
					<Text strong className="text-xs">创建</Text>
					<Text type="secondary" className="text-xs ml-2">
						{formatDateTime(run.created_at)}
					</Text>
				</div>
			),
		});
	}
	if (run.started_at) {
		timelineItems.push({
			color: "blue",
			children: (
				<div>
					<Text strong className="text-xs">开始运行</Text>
					<Text type="secondary" className="text-xs ml-2">
						{formatDateTime(run.started_at)}
					</Text>
				</div>
			),
		});
	}
	if (run.finished_at) {
		timelineItems.push({
			color: run.status === "ok" ? "green" : run.status === "failed" ? "red" : "gray",
			children: (
				<div>
					<Text strong className="text-xs">
						{run.status === "ok" ? "完成" : run.status === "failed" ? "失败" : "结束"}
					</Text>
					<Text type="secondary" className="text-xs ml-2">
						{formatDateTime(run.finished_at)}
					</Text>
				</div>
			),
		});
	}

	return (
		<div>
			{msgCtx}

			{/* Header */}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 16,
				}}
			>
				<Button
					icon={<ArrowLeftOutlined />}
					type="text"
					onClick={() => navigate("/algo-runs")}
				/>
				<Title level={4} style={{ margin: 0 }}>
					算法运行详情
				</Title>
				<Tag
					color={STATUS_COLOR[run.status] ?? "default"}
					icon={STATUS_ICON[run.status]}
				>
					{STATUS_LABEL[run.status] ?? run.status}
				</Tag>
				<Button
					icon={<ReloadOutlined />}
					size="small"
					onClick={fetchRun}
				>
					刷新
				</Button>
				{run.external_url && (
					<Button
						icon={<ExportOutlined />}
						size="small"
						href={run.external_url}
						target="_blank"
						rel="noopener noreferrer"
					>
						外部链接
					</Button>
				)}
			</div>

			{/* Basic info */}
			<Descriptions
				bordered
				column={2}
				size="small"
				style={{ marginBottom: 24 }}
			>
				<Descriptions.Item label="Run ID">
					<Text code className="text-xs">{run.run_id}</Text>
				</Descriptions.Item>
				<Descriptions.Item label="算法">
					<code className="text-xs">{run.algo_name}@{run.algo_version}</code>
				</Descriptions.Item>
				<Descriptions.Item label="算法类型">
					{run.algo_kind}
				</Descriptions.Item>
				<Descriptions.Item label="触发方">
					{run.triggered_by}
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag
						color={STATUS_COLOR[run.status] ?? "default"}
						icon={STATUS_ICON[run.status]}
					>
						{STATUS_LABEL[run.status] ?? run.status}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="耗时">
					{formatDuration(run.started_at, run.finished_at)}
				</Descriptions.Item>
				<Descriptions.Item label="开始时间">
					{formatDateTime(run.started_at)}
				</Descriptions.Item>
				<Descriptions.Item label="结束时间">
					{formatDateTime(run.finished_at)}
				</Descriptions.Item>
				<Descriptions.Item label="处理资产">
					{run.assets_processed ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="成功/失败">
					<span>
						<Text type="success">{run.assets_succeeded ?? 0}</Text>
						{" / "}
						<Text type="danger">{run.assets_failed ?? 0}</Text>
					</span>
				</Descriptions.Item>
				<Descriptions.Item label="创建时间" span={2}>
					{formatDateTime(run.created_at)}
				</Descriptions.Item>
			</Descriptions>

			{/* Status timeline */}
			{timelineItems.length > 0 && (
				<Card title="状态时间线" size="small" style={{ marginBottom: 24 }}>
					<Timeline items={timelineItems} />
				</Card>
			)}

			{/* Asset processing summary */}
			{run.assets_processed != null && run.assets_processed > 0 && (
				<Card title="资产处理统计" size="small">
					<div style={{ display: "flex", gap: 24, flexWrap: "wrap" }}>
						<div>
							<Text type="secondary">总计</Text>
							<div style={{ fontSize: 24, fontWeight: 600 }}>
								{run.assets_processed}
							</div>
						</div>
						<div>
							<Text type="secondary">成功</Text>
							<div style={{ fontSize: 24, fontWeight: 600, color: "#52c41a" }}>
								{run.assets_succeeded ?? 0}
							</div>
						</div>
						<div>
							<Text type="secondary">失败</Text>
							<div style={{ fontSize: 24, fontWeight: 600, color: "#ff4d4f" }}>
								{run.assets_failed ?? 0}
							</div>
						</div>
					</div>
				</Card>
			)}
		</div>
	);
}
