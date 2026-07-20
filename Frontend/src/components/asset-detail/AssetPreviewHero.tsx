// ─── AssetPreviewHero — Detail page preview hero with summary panel ───
// Placed above tabs on the asset detail page. Left: preview placeholder.
// Right: asset summary metadata. Stable layout for future video player.
// Validates: Requirements R10

import {
	ExpandOutlined,
	FileImageOutlined,
	SearchOutlined,
} from "@ant-design/icons";
import {
	Badge,
	Button,
	Card,
	Descriptions,
	Divider,
	Grid,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useNavigate } from "react-router-dom";
import type { Asset } from "../../api/types";
import {
	formatDurationSeconds,
	getAssetStateColor,
	getLifecycleState,
} from "../../lib/assetPresentation";
import type {
	PreviewAvailability,
	PreviewManifest,
} from "../../lib/assets/assetsDiscoveryTypes";
import { countStatuses, parseAlgoEntries } from "../assets/AlgoSummaryCell";
import PreviewPlayer from "../assets/PreviewPlayer";

const { Text } = Typography;
const { useBreakpoint } = Grid;

// ─── Props ───

export interface AssetPreviewHeroProps {
	asset: Asset;
	previewManifest: PreviewManifest | null;
	/** CYB-3294: jump to the 算法处理 tab (e.g. clicking the algo status summary). */
	onJumpToAlgo?: () => void;
}

// ─── Color Maps ───

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
	{ status: "default" | "processing" | "success" | "error"; text: string }
> = {
	missing: { status: "default", text: "暂无预览" },
	processing: { status: "processing", text: "Processing" },
	ready: { status: "success", text: "Ready" },
	failed: { status: "error", text: "Failed" },
};

// ─── Algo Summary for Hero ───

const ALGO_STATUS_ORDER = ["ok", "failed", "running", "pending", "blocked"];

function AlgoSummaryInline({
	asset,
	onJumpToAlgo,
}: {
	asset: Asset;
	onJumpToAlgo?: () => void;
}) {
	const entries = parseAlgoEntries(asset.algo_results);
	if (entries.length === 0) {
		return <Text type="secondary">—</Text>;
	}

	const counts = countStatuses(entries);
	const parts: { label: string; count: number; color?: string }[] = [];
	for (const status of ALGO_STATUS_ORDER) {
		if (counts[status]) {
			parts.push({
				label: status,
				count: counts[status],
				color: status === "failed" ? "#ff4d4f" : undefined,
			});
		}
	}
	for (const [status, count] of Object.entries(counts)) {
		if (!ALGO_STATUS_ORDER.includes(status)) {
			parts.push({ label: status, count });
		}
	}

	const content = (
		<span style={{ fontSize: 13 }}>
			{parts.map((p, i) => (
				<span key={p.label}>
					{i > 0 && " / "}
					<span style={p.color ? { color: p.color } : undefined}>
						{p.count} {p.label}
					</span>
				</span>
			))}
		</span>
	);

	// CYB-3294: clicking the algo summary jumps to the 算法处理 tab for triage.
	if (onJumpToAlgo) {
		return (
			<Tooltip title="查看算法处理">
				<button
					type="button"
					onClick={onJumpToAlgo}
					style={{
						cursor: "pointer",
						border: "none",
						background: "none",
						padding: 0,
						margin: 0,
						font: "inherit",
						color: "inherit",
						textAlign: "inherit",
					}}
				>
					{content}
				</button>
			</Tooltip>
		);
	}
	return content;
}

// ─── Left Panel: Preview Media Placeholder ───

function PreviewMediaPanel({
	manifest,
	asset,
	isNarrow,
}: {
	manifest: PreviewManifest | null;
	asset: Asset;
	isNarrow: boolean;
}) {
	const navigate = useNavigate();
	const availability = manifest?.availability ?? "missing";
	const badge = AVAILABILITY_BADGE[availability];
	if (manifest?.mode === "mcap") {
		return (
			<div
				style={{
					flex: isNarrow ? "1 1 100%" : "0 0 60%",
					display: "flex",
					flexDirection: "column",
					justifyContent: "center",
					alignItems: "stretch",
					minHeight: 200,
				}}
			>
				<PreviewPlayer manifest={manifest} />
				<div
					style={{
						marginTop: 8,
						display: "flex",
						alignItems: "center",
						justifyContent: "center",
						gap: 12,
					}}
				>
					<Badge
						status={badge.status}
						text={
							<Text type="secondary" style={{ fontSize: 12 }}>
								已就绪
							</Text>
						}
					/>
					<Button
						size="small"
						type="link"
						icon={<ExpandOutlined />}
						style={{ padding: 0, fontSize: 12 }}
						onClick={() =>
							navigate(`/preview?asset=${encodeURIComponent(asset.asset_id)}`)
						}
					>
						完整预览
					</Button>
				</div>
			</div>
		);
	}

	if (
		manifest?.mode === "thumbnail" &&
		typeof manifest.thumbnailUrl === "string" &&
		manifest.thumbnailUrl.length > 0
	) {
		return (
			<div
				style={{
					flex: isNarrow ? "1 1 100%" : "0 0 60%",
					display: "flex",
					flexDirection: "column",
					justifyContent: "center",
					alignItems: "center",
					minHeight: 200,
					padding: 8,
				}}
			>
				<img
					src={manifest.thumbnailUrl}
					alt=""
					style={{
						maxWidth: "100%",
						maxHeight: 320,
						objectFit: "contain",
						borderRadius: 4,
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
				flex: isNarrow ? "1 1 100%" : "0 0 60%",
				display: "flex",
				flexDirection: "column",
				justifyContent: "center",
				alignItems: "center",
				padding: 24,
				minHeight: 200,
			}}
		>
			<FileImageOutlined
				style={{ fontSize: 48, color: "#bfbfbf", marginBottom: 12 }}
			/>
			<Text type="secondary" style={{ fontSize: 14, marginBottom: 8 }}>
				暂无预览
			</Text>
			<Badge
				status={badge.status}
				text={
					<Text type="secondary" style={{ fontSize: 12 }}>
						{badge.text}
					</Text>
				}
			/>
			<Text type="secondary" style={{ fontSize: 11, marginTop: 4 }}>
				暂无缩略图或预览文件
			</Text>

			<Divider style={{ margin: "16px 0 12px" }} />

			<div style={{ display: "flex", gap: 8 }}>
				<Tooltip title="Coming in Phase 3">
					<Button size="small" icon={<SearchOutlined />} disabled>
						查找相似
					</Button>
				</Tooltip>
				<Button
					size="small"
					icon={<ExpandOutlined />}
					onClick={() =>
						navigate(`/preview?asset=${encodeURIComponent(asset.asset_id)}`)
					}
				>
					完整预览
				</Button>
			</div>
		</div>
	);
}

// ─── Right Panel: Asset Summary ───

function AssetSummaryPanel({
	asset,
	isNarrow,
	onJumpToAlgo,
}: {
	asset: Asset;
	isNarrow: boolean;
	onJumpToAlgo?: () => void;
}) {
	const priority = asset.tags?.priority;
	const quality = asset.tags?.quality;
	const fileCount = asset.files ? Object.keys(asset.files).length : 0;

	return (
		<div
			style={{
				flex: isNarrow ? "1 1 100%" : "0 0 40%",
				paddingLeft: isNarrow ? 0 : 16,
				paddingTop: isNarrow ? 16 : 0,
				display: "flex",
				flexDirection: "column",
				justifyContent: "flex-start",
				minHeight: 200,
			}}
		>
			{/* Status + Priority/Quality Tags */}
			<div
				style={{ marginBottom: 12, display: "flex", gap: 6, flexWrap: "wrap" }}
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

			{/* Summary Descriptions */}
			<Descriptions
				column={1}
				size="small"
				styles={{
					label: { fontSize: 12, color: "#8c8c8c", width: 80 },
					content: {
						fontSize: 12,
						minWidth: 0,
						overflowWrap: "anywhere",
						wordBreak: "break-word",
					},
				}}
			>
				<Descriptions.Item label="Owner">
					{asset.owner || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Reviewer">
					{asset.reviewer || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Duration">
					{formatDurationSeconds(asset)}
				</Descriptions.Item>
				<Descriptions.Item label="Env">{asset.env ?? "—"}</Descriptions.Item>
				<Descriptions.Item label="Algo">
					<AlgoSummaryInline asset={asset} onJumpToAlgo={onJumpToAlgo} />
				</Descriptions.Item>
				<Descriptions.Item label="已完成交付">
					{asset.delivery_count ?? 0}
				</Descriptions.Item>
				<Descriptions.Item label="Files">{fileCount}</Descriptions.Item>
			</Descriptions>
		</div>
	);
}

// ─── Main Component ───

export default function AssetPreviewHero({
	asset,
	previewManifest,
	onJumpToAlgo,
}: AssetPreviewHeroProps) {
	const screens = useBreakpoint();
	const isNarrow =
		typeof window !== "undefined" &&
		window.innerWidth < 768 &&
		screens.md !== true;
	return (
		<Card
			size="small"
			styles={{ body: { padding: 16 } }}
			style={{ marginBottom: 16 }}
		>
			<div
				style={{
					display: "flex",
					flexDirection: isNarrow ? "column" : "row",
					minHeight: 200,
					gap: 0,
				}}
			>
				<PreviewMediaPanel
					manifest={previewManifest}
					asset={asset}
					isNarrow={isNarrow}
				/>
				<AssetSummaryPanel
					asset={asset}
					isNarrow={isNarrow}
					onJumpToAlgo={onJumpToAlgo}
				/>
			</div>
		</Card>
	);
}
