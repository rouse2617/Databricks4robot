import {
	CopyOutlined,
	LinkOutlined,
	PlayCircleOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Descriptions,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { assetsApi } from "../../api/assets";
import type { Asset, McapFile } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";
import {
	formatDurationSeconds,
	getAssetStateColor,
	getLifecycleState,
} from "../../lib/assetPresentation";
import { navigateToAssetDetail } from "../../lib/assets/assetWorkbenchNavigation";

const { Text } = Typography;

function formatBytes(bytes: number): string {
	if (!bytes || bytes === 0) return "—";
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024)
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function formatNs(ns?: number): string {
	if (!ns) return "—";
	return dayjs(ns / 1e6).format("YYYY-MM-DD HH:mm:ss");
}

function algoSummary(algoResults: Record<string, string> | undefined): string {
	if (!algoResults) return "—";
	const counts: Record<string, number> = {};
	for (const [key, value] of Object.entries(algoResults)) {
		if (key.endsWith(":status")) {
			counts[value] = (counts[value] ?? 0) + 1;
		}
	}
	if (Object.keys(counts).length === 0) return "—";
	return Object.entries(counts)
		.map(([status, count]) => `${count} ${status}`)
		.join(" / ");
}

interface McapDetailContentProps {
	mcapFile: McapFile;
	onNavigateAway?: () => void;
}

export function McapDetailActions({ mcapFile }: { mcapFile: McapFile }) {
	const foxgloveUrl = mcapFile.gcs_path
		? `foxglove://open?ds=remote-file&ds.url=${encodeURIComponent(mcapFile.gcs_path)}`
		: null;

	if (!foxgloveUrl) return null;
	return (
		<Tooltip title="在 Foxglove Studio 中打开">
			<Button
				icon={<PlayCircleOutlined />}
				href={foxgloveUrl}
				target="_blank"
				rel="noopener noreferrer"
				size="small"
			>
				Foxglove
			</Button>
		</Tooltip>
	);
}

export default function McapDetailContent({
	mcapFile,
	onNavigateAway,
}: McapDetailContentProps) {
	const navigate = useNavigate();
	const [assets, setAssets] = useState<Asset[]>([]);
	const [assetsLoading, setAssetsLoading] = useState(false);
	const [assetsError, setAssetsError] = useState<string | null>(null);

	const loadRelatedAssets = useCallback(async (fileId: string) => {
		setAssetsLoading(true);
		setAssetsError(null);
		try {
			const data = await assetsApi.list({
				mcap_file_id: fileId,
				page: 1,
				page_size: 100,
			});
			setAssets(data.items ?? []);
		} catch (err) {
			setAssets([]);
			setAssetsError(extractApiErrorMessage(err, "加载关联资产失败"));
		} finally {
			setAssetsLoading(false);
		}
	}, []);

	useEffect(() => {
		setAssets([]);
		void loadRelatedAssets(mcapFile.mcap_file_id);
	}, [mcapFile.mcap_file_id, loadRelatedAssets]);

	const assetColumns: ColumnsType<Asset> = [
		{
			title: "Asset ID",
			dataIndex: "asset_id",
			width: 140,
			render: (id: string) => (
				<button
					type="button"
					className="link-like-button"
					onClick={(event) => {
						event.stopPropagation();
						onNavigateAway?.();
						navigateToAssetDetail(navigate, id);
					}}
				>
					<span className="font-mono text-xs">{id?.slice(0, 12)}…</span>
				</button>
			),
		},
		{
			title: "生命周期",
			width: 90,
			render: (_: unknown, asset: Asset) => (
				<Tag color={getAssetStateColor(asset)}>
					{getLifecycleState(asset) || "—"}
				</Tag>
			),
		},
		{
			title: "时长 (s)",
			width: 90,
			render: (_: unknown, asset: Asset) => formatDurationSeconds(asset),
		},
		{
			title: "环境",
			dataIndex: "env",
			width: 90,
			render: (value: string) => value || "—",
		},
		{
			title: "算法摘要",
			key: "algo_summary",
			width: 140,
			render: (_: unknown, record: Asset) => algoSummary(record.algo_results),
		},
		{
			title: "更新时间",
			dataIndex: "updated_at",
			width: 130,
			render: (value: string) =>
				value ? dayjs(value).format("MM-DD HH:mm") : "—",
		},
	];

	return (
		<>
			<Descriptions
				column={2}
				size="small"
				bordered
				styles={{ label: { width: 140 } }}
			>
				<Descriptions.Item label="MCAP File ID" span={2}>
					<Text copyable className="font-mono text-xs">
						{mcapFile.mcap_file_id}
					</Text>
				</Descriptions.Item>
				<Descriptions.Item label="GCS Path" span={2}>
					<Space size={4}>
						<Text
							ellipsis={{ tooltip: mcapFile.gcs_path }}
							style={{ maxWidth: 560, fontSize: 12 }}
						>
							{mcapFile.gcs_path || "—"}
						</Text>
						{mcapFile.gcs_path && (
							<Tooltip title="复制路径">
								<Button
									type="text"
									size="small"
									icon={<CopyOutlined />}
									onClick={() =>
										navigator.clipboard.writeText(mcapFile.gcs_path)
									}
								/>
							</Tooltip>
						)}
					</Space>
				</Descriptions.Item>
				<Descriptions.Item label="大小">
					{formatBytes(mcapFile.size_bytes)}
				</Descriptions.Item>
				<Descriptions.Item label="MD5">
					<Text className="font-mono text-xs">
						{mcapFile.raw_hash_md5 || "—"}
					</Text>
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag
						color={
							mcapFile.ingest_state === "summarized"
								? "success"
								: mcapFile.ingest_state === "failed"
									? "error"
									: "default"
						}
					>
						{mcapFile.ingest_state || "—"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="Owner">
					{mcapFile.owner || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Channels">
					{mcapFile.channel_count ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Chunks">
					{mcapFile.chunk_count ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="开始时间">
					{formatNs(mcapFile.start_timestamp_ns)}
				</Descriptions.Item>
				<Descriptions.Item label="结束时间">
					{formatNs(mcapFile.end_timestamp_ns)}
				</Descriptions.Item>
				<Descriptions.Item label="创建时间">
					{mcapFile.created_at
						? dayjs(mcapFile.created_at).format("YYYY-MM-DD HH:mm:ss")
						: "—"}
				</Descriptions.Item>
				<Descriptions.Item label="更新时间">
					{mcapFile.updated_at
						? dayjs(mcapFile.updated_at).format("YYYY-MM-DD HH:mm:ss")
						: "—"}
				</Descriptions.Item>
				<Descriptions.Item label="版本">{mcapFile.version}</Descriptions.Item>
				<Descriptions.Item label="Process State">
					{mcapFile.process_state
						? JSON.stringify(mcapFile.process_state)
						: "—"}
				</Descriptions.Item>
			</Descriptions>

			<div style={{ marginTop: 24 }}>
				<Text strong style={{ fontSize: 14 }}>
					<LinkOutlined style={{ marginRight: 6 }} />
					关联资产 ({assets.length})
				</Text>
				{assetsError && (
					<Alert
						type="error"
						showIcon
						style={{ marginTop: 8 }}
						message="加载关联资产失败"
						description={assetsError}
						action={
							<Button
								size="small"
								type="primary"
								onClick={() => loadRelatedAssets(mcapFile.mcap_file_id)}
							>
								重试
							</Button>
						}
					/>
				)}
				<Table
					rowKey="asset_id"
					columns={assetColumns}
					dataSource={assets}
					loading={assetsLoading}
					size="small"
					pagination={false}
					scroll={{ x: 600 }}
					style={{ marginTop: 8 }}
					locale={{
						emptyText: assetsError ? "关联资产加载失败" : "暂无关联资产",
					}}
					onRow={(record) => ({
						style: { cursor: "pointer" },
						onClick: () => {
							onNavigateAway?.();
							navigateToAssetDetail(navigate, record.asset_id);
						},
					})}
				/>
			</div>
		</>
	);
}
