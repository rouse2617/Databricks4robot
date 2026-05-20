// ─── AssetQuickPreviewPane — Right-side quick preview panel ───
// Shows asset summary, algo status, files, and actions when a row is selected.
// Validates: Requirements R7

import {
	CheckCircleOutlined,
	CloseCircleOutlined,
	CopyOutlined,
	EyeOutlined,
	FileImageOutlined,
	InboxOutlined,
	MenuFoldOutlined,
	MenuUnfoldOutlined,
	SearchOutlined,
} from "@ant-design/icons";
import {
	Badge,
	Button,
	Card,
	Descriptions,
	Divider,
	Input,
	message,
	Select,
	Skeleton,
	Spin,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useMemo } from "react";
import type { Asset } from "../../api/types";
import {
	formatDurationSeconds,
	getAssetStateColor,
	getLifecycleState,
} from "../../lib/assetPresentation";
import type {
	FetchStatus,
	PreviewAvailability,
	PreviewManifest,
} from "../../lib/assets/assetsDiscoveryTypes";
import { countStatuses, parseAlgoEntries } from "./AlgoSummaryCell";
import PreviewPlayer from "./PreviewPlayer";

const { Text, Title } = Typography;

// ─── Props ───

export interface AssetQuickPreviewPaneProps {
	activeAssetId: string | null;
	fetchStatus: FetchStatus;
	asset: Asset | null;
	previewManifest: PreviewManifest | null;
	collapsed: boolean;
	onCollapse: () => void;
	onOpenDetail: (assetId: string) => void;
	onMcapClick?: (mcapFileId: string) => void;
	onFindSimilar: (assetId: string) => void;
	onRetry?: () => void;
	previewTopic?: string | null;
	onTopicChange?: (topic: string | null) => void;
	previewSourceId?: string | null;
	onSourceChange?: (sourceId: string | null) => void;
	previewTimeSec?: number | null;
	onPreviewTimeChange?: (timeSec: number | null) => void;
}

// ─── Status Color Map ───

const PRIORITY_TAG_COLOR: Record<string, string> = {
	critical: "red",
	high: "orange",
	medium: "blue",
	low: "default",
};

const QUALITY_TAG_COLOR: Record<string, string> = {
	excellent: "green",
	good: "cyan",
	acceptable: "blue",
	poor: "orange",
	unusable: "red",
};

const AVAILABILITY_BADGE: Record<
	PreviewAvailability,
	{
		status: "default" | "success" | "processing" | "error" | "warning";
		text: string;
	}
> = {
	missing: { status: "default", text: "暂无预览" },
	processing: { status: "processing", text: "Processing" },
	ready: { status: "success", text: "Ready" },
	failed: { status: "error", text: "Failed" },
};

// ─── Algo Summary Helpers (reuses AlgoSummaryCell parsing) ───

const ALGO_STATUS_ORDER = ["ok", "failed", "running", "pending", "blocked"];

function AlgoSummaryBlock({ asset }: { asset: Asset }) {
	const entries = parseAlgoEntries(asset.algo_results);
	if (entries.length === 0) {
		return <Text type="secondary">无算法结果</Text>;
	}

	const counts = countStatuses(entries);

	const parts: { label: string; count: number; bg: string; fg: string }[] = [];
	for (const status of ALGO_STATUS_ORDER) {
		if (counts[status]) {
			const isOk = status === "ok";
			const isFailed = status === "failed";
			parts.push({
				label: status,
				count: counts[status],
				bg: isOk ? "#f0fdf4" : isFailed ? "#fef2f2" : "#f3f4f6",
				fg: isOk ? "#16a34a" : isFailed ? "#dc2626" : "#4b5563",
			});
		}
	}
	for (const [status, count] of Object.entries(counts)) {
		if (!ALGO_STATUS_ORDER.includes(status)) {
			parts.push({ label: status, count, bg: "#f3f4f6", fg: "#4b5563" });
		}
	}

	// Find most recent failure reason
	const failedEntry = entries.find((e) => e.status === "failed");
	const failureReasonKey = failedEntry
		? Object.keys(asset.algo_results).find(
				(k) => k.startsWith(failedEntry.key) && k.endsWith(":reason"),
			)
		: null;
	const failureReason = failureReasonKey
		? asset.algo_results[failureReasonKey]
		: null;

	return (
		<div>
			<div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
				{parts.map((p) => (
					<span
						key={p.label}
						style={{
							fontSize: 11,
							padding: "2px 8px",
							borderRadius: 4,
							background: p.bg,
							color: p.fg,
							fontWeight: 500,
						}}
					>
						{p.count} {p.label}
					</span>
				))}
			</div>
			{failureReason && (
				<Text
					type="secondary"
					style={{
						fontSize: 11,
						display: "block",
						marginTop: 6,
						background: "#fef2f2",
						padding: "4px 8px",
						borderRadius: 4,
						color: "#b91c1c",
					}}
				>
					失败原因: {failureReason}
				</Text>
			)}
		</div>
	);
}

// ─── Files Block ───

function FilesBlock({ files }: { files: Record<string, string> }) {
	const keys = Object.keys(files);
	if (keys.length === 0) {
		return <Text type="secondary">无文件</Text>;
	}
	return (
		<div>
			{keys.map((key) => (
				<div
					key={key}
					style={{
						display: "flex",
						justifyContent: "space-between",
						padding: "2px 0",
						fontSize: 12,
					}}
				>
					<Text style={{ fontSize: 12 }}>{key}</Text>
					{files[key] ? (
						<CheckCircleOutlined style={{ color: "#52c41a" }} />
					) : (
						<CloseCircleOutlined style={{ color: "#ff4d4f" }} />
					)}
				</div>
			))}
		</div>
	);
}

// ─── Empty State ───

function EmptyState() {
	return (
		<div
			style={{
				display: "flex",
				flexDirection: "column",
				alignItems: "center",
				justifyContent: "center",
				height: "100%",
				minHeight: 300,
				color: "#bfbfbf",
			}}
		>
			<InboxOutlined style={{ fontSize: 48, marginBottom: 12 }} />
			<Text type="secondary">点击行查看预览</Text>
		</div>
	);
}

// ─── Error State ───

function ErrorState({ onRetry }: { onRetry?: () => void }) {
	return (
		<div
			style={{
				display: "flex",
				flexDirection: "column",
				alignItems: "center",
				justifyContent: "center",
				height: "100%",
				minHeight: 300,
				color: "#ff4d4f",
			}}
		>
			<CloseCircleOutlined style={{ fontSize: 48, marginBottom: 12 }} />
			<Text type="danger" style={{ marginBottom: 12 }}>
				加载预览失败
			</Text>
			{onRetry && (
				<Button size="small" onClick={onRetry}>
					重试
				</Button>
			)}
		</div>
	);
}

// ─── Preview Media Placeholder ───

function PreviewMediaBlock({
	manifest,
}: {
	manifest: PreviewManifest | null;
	asset: Asset;
}) {
	const availability = manifest?.availability ?? "missing";
	const badge = AVAILABILITY_BADGE[availability];
	const hasMcap = manifest?.mode === "mcap";
	const hasThumbnail =
		manifest?.mode === "thumbnail" &&
		typeof manifest.thumbnailUrl === "string" &&
		manifest.thumbnailUrl.length > 0;
	if (hasMcap) {
		return (
			<div style={{ marginBottom: 12 }}>
				<PreviewPlayer manifest={manifest} compact />
			</div>
		);
	}

	if (hasThumbnail) {
		return (
			<div style={{ marginBottom: 12 }}>
				<img
					src={manifest.thumbnailUrl as string}
					alt=""
					style={{
						width: "100%",
						maxHeight: 260,
						objectFit: "contain",
						borderRadius: 4,
						background: "#f5f5f5",
					}}
				/>
				<div style={{ marginTop: 8, textAlign: "center" }}>
					<Badge
						status={badge.status}
						text={
							<Text type="secondary" style={{ fontSize: 12 }}>
								缩略图
							</Text>
						}
					/>
				</div>
			</div>
		);
	}

	return (
		<div
			style={{
				background: "#fafafa",
				border: "1px dashed #d9d9d9",
				borderRadius: 4,
				padding: "16px 12px",
				textAlign: "center",
				marginBottom: 12,
			}}
		>
			<FileImageOutlined
				style={{ fontSize: 24, color: "#d9d9d9", marginBottom: 8 }}
			/>
			<div style={{ marginBottom: 4 }}>
				<Text type="secondary" style={{ fontSize: 12 }}>
					暂无预览
				</Text>
			</div>
			<Badge
				status={badge.status}
				text={
					<Text type="secondary" style={{ fontSize: 11 }}>
						{badge.text}
					</Text>
				}
			/>
		</div>
	);
}

// ─── Loaded Content ───

function LoadedContent({
	asset,
	previewManifest,
	onOpenDetail,
	onMcapClick,
	onFindSimilar,
}: {
	asset: Asset;
	previewManifest: PreviewManifest | null;
	onOpenDetail: (assetId: string) => void;
	onMcapClick?: (mcapFileId: string) => void;
	onFindSimilar: (assetId: string) => void;
}) {
	const priority = asset.tags?.priority;
	const quality = asset.tags?.quality;

	return (
		<div style={{ padding: "0 4px" }}>
			{/* Block 1: Header */}
			<div style={{ marginBottom: 12 }}>
				<Title
					level={5}
					style={{ margin: 0, fontFamily: "monospace", fontSize: 14 }}
				>
					{asset.asset_id}
				</Title>
				<div
					style={{ marginTop: 6, display: "flex", gap: 4, flexWrap: "wrap" }}
				>
					<Tag color={getAssetStateColor(asset)}>
						{getLifecycleState(asset) || "—"}
					</Tag>
					{priority && (
						<Tag color={PRIORITY_TAG_COLOR[priority] ?? "default"}>
							{priority}
						</Tag>
					)}
					{quality && (
						<Tag color={QUALITY_TAG_COLOR[quality] ?? "default"}>{quality}</Tag>
					)}
				</div>
			</div>

			<Divider style={{ margin: "8px 0" }} />

			{/* Block 2: Preview Media */}
			<PreviewMediaBlock manifest={previewManifest} asset={asset} />

			<Divider style={{ margin: "8px 0" }} />

			{/* Block 3: Summary */}
			<Descriptions
				column={2}
				size="small"
				styles={{
					label: { fontSize: 11, color: "#8c8c8c", paddingBottom: 2 },
					content: { fontSize: 12, paddingBottom: 8 },
				}}
				layout="vertical"
			>
				<Descriptions.Item label="MCAP" span={2}>
					<div style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
						<button
							type="button"
							className="link-like-button font-mono text-xs"
							onClick={() => onMcapClick?.(asset.mcap_file_id)}
							aria-label={`MCAP ${asset.mcap_file_id}`}
						>
							{asset.mcap_file_id}
						</button>
						<Tooltip title="复制 MCAP ID">
							<button
								type="button"
								className="link-like-button"
								aria-label={`复制 MCAP ${asset.mcap_file_id}`}
								onClick={async () => {
									try {
										await navigator.clipboard.writeText(asset.mcap_file_id);
										message.success("已复制 MCAP ID");
									} catch {
										message.error("复制失败");
									}
								}}
							>
								<CopyOutlined style={{ fontSize: 12 }} />
							</button>
						</Tooltip>
					</div>
				</Descriptions.Item>
				<Descriptions.Item label="Env">{asset.env ?? "—"}</Descriptions.Item>
				<Descriptions.Item label="Scene">
					{asset.tags?.scene ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Duration">
					{formatDurationSeconds(asset)}
				</Descriptions.Item>
				<Descriptions.Item label="Owner">{asset.owner}</Descriptions.Item>
				<Descriptions.Item label="Reviewer">{asset.reviewer}</Descriptions.Item>
			</Descriptions>

			<Divider style={{ margin: "8px 0" }} />

			{/* Block 4: Algo Summary */}
			<div style={{ marginBottom: 8 }}>
				<Text
					strong
					style={{ fontSize: 12, display: "block", marginBottom: 4 }}
				>
					算法摘要
				</Text>
				<AlgoSummaryBlock asset={asset} />
			</div>

			<Divider style={{ margin: "8px 0" }} />

			{/* Block 5: Files */}
			<div style={{ marginBottom: 12 }}>
				<Text
					strong
					style={{ fontSize: 12, display: "block", marginBottom: 4 }}
				>
					文件
				</Text>
				<FilesBlock files={asset.files ?? {}} />
			</div>

			<Divider style={{ margin: "8px 0" }} />

			{/* Actions */}
			<div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
				<Button
					type="primary"
					size="small"
					icon={<EyeOutlined />}
					onClick={() => onOpenDetail(asset.asset_id)}
				>
					查看详情
				</Button>
				<Button
					size="small"
					icon={<CopyOutlined />}
					onClick={async () => {
						try {
							await navigator.clipboard.writeText(asset.asset_id);
							message.success("已复制 Asset ID");
						} catch {
							message.error("复制失败");
						}
					}}
				>
					复制 ID
				</Button>
				<Button
					size="small"
					icon={<SearchOutlined />}
					onClick={() => onFindSimilar(asset.asset_id)}
				>
					查找相似
				</Button>
			</div>
		</div>
	);
}

// ─── Main Component ───

export default function AssetQuickPreviewPane({
	activeAssetId,
	fetchStatus,
	asset,
	previewManifest,
	collapsed,
	onCollapse,
	onOpenDetail,
	onMcapClick,
	onFindSimilar,
	onRetry,
	previewTopic,
	onTopicChange,
	previewSourceId,
	onSourceChange,
	// previewTimeSec / onPreviewTimeChange are still part of the public prop surface
	// (callers / URL sync) but the native <video> control owns seeking now, so we
	// no longer plumb them down to PreviewPlayer.
}: AssetQuickPreviewPaneProps) {
	const canInputTopic = useMemo(
		() => activeAssetId && previewManifest?.mode === "mcap",
		[activeAssetId, previewManifest?.mode],
	);
	// Collapsed state: thin vertical bar with expand button
	if (collapsed) {
		return !activeAssetId ? null : (
			<div
				style={{
					width: 36,
					minHeight: 400,
					background: "#fafafa",
					borderLeft: "1px solid #f0f0f0",
					display: "flex",
					alignItems: "flex-start",
					justifyContent: "center",
					paddingTop: 12,
				}}
			>
				<Button
					type="text"
					size="small"
					icon={<MenuUnfoldOutlined />}
					onClick={onCollapse}
					title="展开预览"
				/>
			</div>
		);
	}

	// Determine content
	let content: React.ReactNode;
	if (!activeAssetId) {
		content = <EmptyState />;
	} else if (fetchStatus === "loading" && !asset) {
		content = <Skeleton active paragraph={{ rows: 4 }} />;
	} else if (fetchStatus === "error") {
		content = <ErrorState onRetry={onRetry} />;
	} else if (asset) {
		content = (
			<Spin spinning={fetchStatus === "loading"} tip="加载中...">
				<LoadedContent
					asset={asset}
					previewManifest={previewManifest}
					onOpenDetail={onOpenDetail}
					onMcapClick={onMcapClick}
					onFindSimilar={onFindSimilar}
				/>
			</Spin>
		);
	} else {
		content = <EmptyState />;
	}

	return (
		<Card
			size="small"
			style={{
				// Match parent sidebar width from AssetsPage (320 / collapsed 36)
				// to avoid clipping the header extra actions (e.g. collapse button).
				width: "100%",
				minWidth: 0,
				minHeight: 400,
				maxHeight: "calc(100vh - 140px)",
				overflow: "hidden",
			}}
			title={
				<div
					style={{
						display: "flex",
						flexDirection: "column",
						gap: 8,
						minWidth: 0,
						paddingRight: 4,
					}}
				>
					<Text strong style={{ fontSize: 13 }}>
						预览
					</Text>
					{(activeAssetId && previewManifest?.sources?.length) ||
					canInputTopic ? (
						<div
							style={{
								display: "flex",
								flexWrap: "wrap",
								gap: 8,
								alignItems: "center",
							}}
						>
							{activeAssetId && previewManifest?.sources?.length ? (
								<Select
									size="small"
									showSearch
									optionFilterProp="label"
									placeholder="预览源"
									value={
										previewSourceId ??
										previewManifest.activeSourceId ??
										undefined
									}
									onChange={(value) => onSourceChange?.(value ?? null)}
									options={previewManifest.sources.map((s) => ({
										value: s.id,
										label: s.label,
									}))}
									style={{ minWidth: 140, flex: "1 1 180px", maxWidth: "100%" }}
								/>
							) : null}
							{canInputTopic ? (
								<Input
									size="small"
									placeholder="预览 Topic（可选）"
									value={previewTopic ?? ""}
									onChange={(e) => {
										const next = e.target.value.trim();
										onTopicChange?.(next ? next : null);
									}}
									style={{ minWidth: 120, flex: "1 1 160px", maxWidth: "100%" }}
								/>
							) : null}
						</div>
					) : null}
				</div>
			}
			extra={
				<Tooltip title="收起预览侧栏">
					<Button
						type="default"
						size="small"
						icon={<MenuFoldOutlined />}
						onClick={onCollapse}
						aria-label="收起预览"
					>
						收起
					</Button>
				</Tooltip>
			}
			styles={{
				body: {
					padding: 12,
					overflowY: "auto",
					maxHeight: "calc(100vh - 210px)",
				},
			}}
		>
			{content}
		</Card>
	);
}
