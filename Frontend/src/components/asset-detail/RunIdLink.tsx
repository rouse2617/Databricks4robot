import { CopyOutlined, LinkOutlined } from "@ant-design/icons";
import {
	Button,
	Descriptions,
	message,
	Popover,
	Spin,
	Tag,
	Typography,
} from "antd";
import { useCallback, useState } from "react";
import { getPipelineRun, type PipelineRun } from "../../api/pipelineApi";
import { formatDateTime } from "../../lib/dateTime";
import {
	formatBusinessStatusLabel,
	resolveBusinessStatusTagColor,
} from "../../lib/productVocabulary";
import { formatRunIdShort, isRegisteredRunId } from "../../lib/runId";

const { Text } = Typography;

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
	const [pipelineRun, setPipelineRun] = useState<PipelineRun | null>(null);
	const [loadError, setLoadError] = useState<string | null>(null);

	const id = runId?.trim() ?? "";
	const fetchRun = useCallback(async () => {
		setLoading(true);
		setLoadError(null);
		try {
			const row = await getPipelineRun(id);
			setPipelineRun(row);
		} catch {
			setPipelineRun(null);
			setLoadError("未找到运行记录，或暂无权限访问");
		} finally {
			setLoading(false);
		}
	}, [id]);

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

	const handleOpenChange = (next: boolean) => {
		setOpen(next);
		if (next && !pipelineRun) {
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
	) : pipelineRun ? (
		<Descriptions column={1} size="small" className="max-w-xs">
			<Descriptions.Item label="流水线">
				{pipelineRun.pipelineName || "—"}
			</Descriptions.Item>
			<Descriptions.Item label="状态">
				<Tag color={resolveBusinessStatusTagColor(pipelineRun.status)}>
					{formatBusinessStatusLabel(pipelineRun.status)}
				</Tag>
			</Descriptions.Item>
			{pipelineRun.startedAt ? (
				<Descriptions.Item label="开始">
					{formatDateTime(pipelineRun.startedAt)}
				</Descriptions.Item>
			) : null}
			{pipelineRun.finishedAt ? (
				<Descriptions.Item label="结束">
					{formatDateTime(pipelineRun.finishedAt)}
				</Descriptions.Item>
			) : null}
			{pipelineRun.assetCount != null ? (
				<Descriptions.Item label="处理资产">
					{pipelineRun.assetCount}
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
						运行
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
