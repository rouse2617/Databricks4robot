import { CopyOutlined, LinkOutlined } from "@ant-design/icons";
import { Button, Descriptions, Popover, Spin, Tag, Typography, message } from "antd";
import { useCallback, useState } from "react";
import { algoRunsApi, type AlgoRun } from "../../api/algoRuns";
import { formatDateTime } from "../../lib/dateTime";
import { formatRunIdShort, isRegisteredRunId } from "../../lib/runId";

const { Text } = Typography;

const statusColor: Record<string, string> = {
	pending: "default",
	running: "processing",
	ok: "success",
	failed: "error",
	cancelled: "warning",
};

interface RunIdLinkProps {
	runId?: string | null;
	/** Prefix label in table cells, e.g. empty in provenance line */
	showLabel?: boolean;
}

export default function RunIdLink({
	runId,
	showLabel = false,
}: RunIdLinkProps) {
	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [run, setRun] = useState<AlgoRun | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);

	const id = runId?.trim() ?? "";
	if (!id) {
		return <Text type="secondary">—</Text>;
	}

	if (!isRegisteredRunId(id)) {
		return (
			<Text code className="text-xs">
				{id}
			</Text>
		);
	}

	const fetchRun = useCallback(async () => {
		setLoading(true);
		setLoadError(null);
		try {
			const row = await algoRunsApi.get(id);
			setRun(row);
		} catch {
			setRun(null);
			setLoadError("未找到已登记的 run，或暂无权限访问");
		} finally {
			setLoading(false);
		}
	}, [id]);

	const handleOpenChange = (next: boolean) => {
		setOpen(next);
		if (next && !run && !loadError) {
			void fetchRun();
		}
	};

	const copyId = () => {
		void navigator.clipboard.writeText(id);
		message.success("已复制 run_id");
	};

	const popoverBody = loading ? (
		<div className="py-4 px-6">
			<Spin size="small" />
		</div>
	) : loadError ? (
		<Text type="secondary" className="text-xs">
			{loadError}
		</Text>
	) : run ? (
		<Descriptions column={1} size="small" className="max-w-xs">
			<Descriptions.Item label="算法">
				{run.algo_name}@{run.algo_version}
			</Descriptions.Item>
			<Descriptions.Item label="状态">
				<Tag color={statusColor[run.status] ?? "default"}>{run.status}</Tag>
			</Descriptions.Item>
			<Descriptions.Item label="触发方">{run.triggered_by}</Descriptions.Item>
			{run.started_at ? (
				<Descriptions.Item label="开始">
					{formatDateTime(run.started_at)}
				</Descriptions.Item>
			) : null}
			{run.finished_at ? (
				<Descriptions.Item label="结束">
					{formatDateTime(run.finished_at)}
				</Descriptions.Item>
			) : null}
			{run.assets_processed != null ? (
				<Descriptions.Item label="处理统计">
					{run.assets_succeeded ?? 0} 成功 / {run.assets_failed ?? 0} 失败 /{" "}
					{run.assets_processed} 总计
				</Descriptions.Item>
			) : null}
			<Descriptions.Item label="run_id">
				<Text code className="text-xs break-all">
					{id}
				</Text>
			</Descriptions.Item>
		</Descriptions>
	) : (
		<Text type="secondary" className="text-xs">
			加载中…
		</Text>
	);

	return (
		<span className="inline-flex items-center gap-1 flex-wrap">
			{showLabel ? (
				<Text type="secondary" className="text-xs">
					来自 run
				</Text>
			) : null}
			<Popover
				open={open}
				onOpenChange={handleOpenChange}
				trigger="click"
				placement="right"
				title={
					<span className="inline-flex items-center gap-1">
						<LinkOutlined />
						算法运行
					</span>
				}
				content={
					<div>
						{popoverBody}
						<div className="mt-2 text-right">
							<Button
								type="text"
								size="small"
								icon={<CopyOutlined />}
								onClick={copyId}
							>
								复制 run_id
							</Button>
						</div>
					</div>
				}
			>
				<Button
					type="link"
					size="small"
					className="p-0 h-auto font-mono text-xs inline-flex items-center"
					aria-label={`查看 run ${id}`}
				>
					{formatRunIdShort(id)}
				</Button>
			</Popover>
		</span>
	);
}
