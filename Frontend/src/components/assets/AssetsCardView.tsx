// ─── AssetsCardView — Wide card list view for data exploration ───
// Renders one full-width card per asset with thumbnail, all tags, lineage, storage.
// Validates: Requirements R1

import {
	Pagination,
	Select,
	Skeleton,
	Spin,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useRef, useState } from "react";
import type { Asset } from "../../api/types";
import { formatDurationSeconds } from "../../lib/assetPresentation";
import type { FetchStatus } from "../../lib/assets/assetsDiscoveryTypes";
import ResultsEmptyState from "./ResultsEmptyState";

const { Text } = Typography;

// ─── State colors & gradients ───

const SORT_OPTIONS: { value: string; label: string }[] = [
	{ value: "-updated_at", label: "更新时间 ↓" },
	{ value: "updated_at", label: "更新时间 ↑" },
	{ value: "-duration_ms", label: "时长 ↓" },
	{ value: "duration_ms", label: "时长 ↑" },
];

const STATE_COLORS: Record<string, string> = {
	ready: "#34d399",
	error: "#f87171",
	delivered: "#fbbf24",
	processed: "#a78bfa",
	raw: "#94a3b8",
	archived: "#9ca3af",
	ingested: "#60a5fa",
};

const THUMB_GRADIENTS: Record<string, [string, string]> = {
	ready: ["#d1fae5", "#a7f3d0"],
	delivered: ["#fef3c7", "#fde68a"],
	error: ["#fee2e2", "#fecaca"],
	processed: ["#e0e7ff", "#c7d2fe"],
	raw: ["#f1f5f9", "#e2e8f0"],
	archived: ["#f9fafb", "#e5e7eb"],
	ingested: ["#dbeafe", "#bfdbfe"],
};

const COLOR_BAR: Record<string, string> = {
	ready: "linear-gradient(180deg, #6ee7b7, #34d399)",
	delivered: "linear-gradient(180deg, #fde68a, #fbbf24)",
	error: "linear-gradient(180deg, #fca5a5, #f87171)",
	processed: "linear-gradient(180deg, #a5b4fc, #818cf8)",
	raw: "linear-gradient(180deg, #cbd5e1, #94a3b8)",
	archived: "linear-gradient(180deg, #e5e7eb, #9ca3af)",
	ingested: "linear-gradient(180deg, #93c5fd, #60a5fa)",
};

const TAG_COLORS: Record<string, string> = {
	high: "red",
	urgent: "red",
	excellent: "green",
	good: "green",
	acceptable: "blue",
	poor: "red",
	low: "gray",
};

function tagColor(key: string, value: string): string {
	if (key === "priority" || key === "quality")
		return TAG_COLORS[value] ?? "default";
	return "default";
}

// ─── Thumbnail placeholder ───

const DEFAULT_THUMB_SVG = `data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="260" height="160" viewBox="0 0 260 160" fill="none"><rect width="260" height="160" fill="%23f3f4f6"/><path d="M106.667 60H153.333C157.016 60 160 62.9838 160 66.6667V93.3333C160 97.0162 157.016 100 153.333 100H106.667C102.984 100 100 97.0162 100 93.3333V66.6667C100 62.9838 102.984 60 106.667 60ZM110.667 93.3333H149.333L138.835 77.2917L128.536 90.0469L120.354 78.4729L110.667 93.3333ZM116.667 73.3333C118.508 73.3333 120 71.841 120 70C120 68.1591 118.508 66.6667 116.667 66.6667C114.826 66.6667 113.333 68.1591 113.333 70C113.333 71.841 114.826 73.3333 116.667 73.3333Z" fill="%23d1d5db"/></svg>`;

function ThumbPlaceholder({ asset }: { asset: Asset }) {
	const state = asset.lifecycle_state ?? "ready";
	const [showFallback, setShowFallback] = useState(false);

	const imageSrc = asset.thumb_uri || DEFAULT_THUMB_SVG;

	if (!showFallback) {
		return (
			<div
				style={{
					width: 130,
					flexShrink: 0,
					position: "relative",
					background: "#f3f4f6",
				}}
			>
				<img
					src={imageSrc}
					alt={asset.asset_id}
					style={{
						width: "100%",
						height: "100%",
						minHeight: 80,
						objectFit: "cover",
					}}
					onError={() => setShowFallback(true)}
				/>
			</div>
		);
	}

	const [from, to] = THUMB_GRADIENTS[state] ?? THUMB_GRADIENTS.ready;
	return (
		<div
			style={{
				width: 130,
				flexShrink: 0,
				position: "relative",
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
				background: `linear-gradient(135deg, ${from} 0%, ${to} 100%)`,
				fontWeight: 700,
				fontFamily: "monospace",
				minHeight: 80,
				color:
					state === "ready"
						? "#065f46"
						: state === "error"
							? "#991b1b"
							: "#374151",
			}}
		>
			<span
				style={{
					fontSize: 26,
					fontWeight: 900,
					opacity: 0.13,
					position: "absolute",
					zIndex: 0,
				}}
			>
				{(asset.asset_type ?? asset.type ?? "?").slice(0, 3).toUpperCase()}
			</span>
			<span
				style={{
					fontSize: 13,
					fontWeight: 600,
					color: "inherit",
					position: "relative",
					zIndex: 1,
				}}
			>
				{asset.asset_id.slice(0, 4)}
			</span>
		</div>
	);
}

const MAX_TAG_BADGES = 4;

// ─── Main component ───

export interface AssetsCardViewProps {
	items: Asset[];
	total: number;
	totalApprox: boolean;
	fetchStatus: FetchStatus;
	error?: string | null;
	activeFilterCount?: number;
	hasActiveQueryText?: boolean;
	sort: string;
	page: number;
	pageSize: number;
	selectedIds: Set<string>;
	onSelectRow?: (id: string) => void;
	onRowClick: (id: string) => void;
	onMcapClick?: (mcapFileId: string) => void;
	onSortChange?: (sort: string) => void;
	onPageChange?: (page: number, pageSize: number) => void;
	onClearFilters?: () => void;
	onRetry?: () => void;
}

export default function AssetsCardView({
	items,
	total,
	totalApprox,
	fetchStatus,
	error,
	activeFilterCount = 0,
	hasActiveQueryText = false,
	sort,
	page,
	pageSize,
	selectedIds,
	onSelectRow,
	onRowClick,
	onMcapClick,
	onSortChange,
	onPageChange,
	onClearFilters,
	onRetry,
}: AssetsCardViewProps) {
	const emptyRef = useRef<HTMLDivElement>(null);

	// Keyboard: Enter to open asset detail
	useEffect(() => {
		const el = emptyRef.current;
		if (!el) return;
		const handler = (e: KeyboardEvent) => {
			const cardEl = (e.target as HTMLElement)?.closest?.(".assets-card-row");
			if (!cardEl || e.key !== "Enter") return;
			const aid = cardEl.getAttribute("data-asset-id");
			if (aid) onRowClick(aid);
		};
		el.addEventListener("keydown", handler as EventListener);
		return () => el.removeEventListener("keydown", handler as EventListener);
	}, [onRowClick]);

	// Error
	if (fetchStatus === "error") {
		return (
			<ResultsEmptyState variant="error" error={error} onRetry={onRetry} />
		);
	}
	if (fetchStatus === "success" && items.length === 0) {
		const variant =
			activeFilterCount > 0 || hasActiveQueryText ? "no_results" : "initial";
		return (
			<ResultsEmptyState
				variant={variant}
				activeFilterCount={activeFilterCount + (hasActiveQueryText ? 1 : 0)}
				onClearFilters={onClearFilters}
			/>
		);
	}

	// Initial loading with no data yet
	if (fetchStatus === "loading" && items.length === 0) {
		return (
			<div style={{ flex: 1, minWidth: 0 }}>
				<Skeleton active paragraph={{ rows: 6 }} />
			</div>
		);
	}

	return (
		<Spin spinning={fetchStatus === "loading"} tip="加载中...">
			<div ref={emptyRef} style={{ flex: 1, minWidth: 0 }}>
				{/* Stats toolbar */}
				<div
					style={{
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
						marginBottom: 10,
						fontSize: 12,
						color: "#6b7280",
						gap: 12,
					}}
				>
					<Text type="secondary" style={{ fontSize: 13 }}>
						共 {total}
						{totalApprox ? "+" : ""} 条
					</Text>
					{onSortChange && (
						<Select
							size="small"
							value={sort}
							onChange={onSortChange}
							options={SORT_OPTIONS}
							style={{ width: 140 }}
						/>
					)}
				</div>

				{/* Cards */}
				<div
					style={{
						display: "flex",
						flexDirection: "column",
						gap: 8,
					}}
				>
					{items.map((asset) => {
						const state = asset.lifecycle_state ?? "ready";
						const stateColor = STATE_COLORS[state] ?? "#94a3b8";
						const algoStats: Record<string, string> = {};
						let algoOkCount = 0;
						for (const [key, value] of Object.entries(
							asset.algo_results ?? {},
						)) {
							if (key.endsWith(":status")) {
								const algoName = key.split("@")[0];
								algoStats[algoName] = String(value ?? "");
								if (value === "ok") algoOkCount++;
							}
						}
						const totalAlgos = Object.keys(algoStats).length;
						const tagEntries = Object.entries(asset.tags ?? {}).slice(
							0,
							MAX_TAG_BADGES,
						);
						const isSelected = selectedIds.has(asset.asset_id);

						return (
							/* biome-ignore lint/a11y/useSemanticElements: card row needs nested interactive controls (checkbox and mcap button). */
							<div
								key={asset.asset_id}
								className={`assets-card-row ${isSelected ? "assets-card-row-selected" : ""}`}
								data-asset-id={asset.asset_id}
								tabIndex={0}
								role="button"
								aria-pressed={isSelected}
								onClick={() => onRowClick(asset.asset_id)}
								onKeyDown={(e) => {
									if (e.key === "Enter" || e.key === " ") {
										e.preventDefault();
										e.stopPropagation();
										onRowClick(asset.asset_id);
									}
								}}
								style={{
									borderRadius: 10,
									overflow: "hidden",
									background: "#fff",
									border: isSelected
										? "2px solid #6366f1"
										: "1px solid #e5e7eb",
									cursor: "pointer",
									display: "flex",
									flexDirection: "row",
									transition: "box-shadow 0.15s, border-color 0.15s",
								}}
								onMouseEnter={(e) => {
									e.currentTarget.style.boxShadow =
										"0 2px 12px rgba(0,0,0,0.08)";
								}}
								onMouseLeave={(e) => {
									e.currentTarget.style.boxShadow = "";
								}}
							>
								{/* Left color bar */}
								<div
									style={{
										width: 5,
										flexShrink: 0,
										alignSelf: "stretch",
										background: COLOR_BAR[state] ?? COLOR_BAR.ready,
									}}
								/>

								{/* Thumbnail */}
								<ThumbPlaceholder asset={asset} />

								{/* Body */}
								<div
									style={{
										flex: 1,
										padding: "8px 14px",
										display: "flex",
										flexDirection: "column",
										gap: 4,
										minWidth: 0,
									}}
								>
									{/* Row 1: ID + state + owner + actions */}
									<div
										style={{
											display: "flex",
											alignItems: "center",
											justifyContent: "space-between",
											gap: 8,
										}}
									>
										<div
											style={{ display: "flex", alignItems: "center", gap: 8 }}
										>
											<Tooltip title="点击打开详情">
												<Text
													code
													style={{
														fontSize: 13,
														fontWeight: 700,
														color: "#111827",
														cursor: "pointer",
													}}
												>
													{asset.asset_id}
												</Text>
											</Tooltip>
											<span
												style={{
													width: 7,
													height: 7,
													borderRadius: "50%",
													background: stateColor,
													flexShrink: 0,
												}}
											/>
											<Text
												style={{
													fontSize: 10,
													background: "#f3f4f6",
													padding: "1px 6px",
													borderRadius: 4,
													color: "#6b7280",
												}}
											>
												{state}
											</Text>
											{asset.retention_tier && (
												<Text
													style={{
														fontSize: 9,
														fontWeight: 600,
														padding: "1px 5px",
														borderRadius: 3,
														background:
															asset.retention_tier === "hot"
																? "#fef2f2"
																: asset.retention_tier === "warm"
																	? "#fef3c7"
																	: "#f0fdf4",
														color:
															asset.retention_tier === "hot"
																? "#dc2626"
																: asset.retention_tier === "warm"
																	? "#d97706"
																	: "#16a34a",
														border:
															asset.retention_tier === "hot"
																? "1px solid #fecaca"
																: asset.retention_tier === "warm"
																	? "1px solid #fde68a"
																	: "1px solid #bbf7d0",
													}}
												>
													{asset.retention_tier}
												</Text>
											)}
										</div>

										<div
											style={{
												display: "flex",
												alignItems: "center",
												gap: 8,
												fontSize: 11,
												color: "#9ca3af",
											}}
										>
											<Text style={{ fontSize: 11, color: "#374151" }}>
												{asset.owner}
											</Text>
											<Text style={{ fontSize: 10, color: "#9ca3af" }}>
												{asset.updated_at
													? dayjs(asset.updated_at).format("MM-DD HH:mm")
													: "—"}
											</Text>

											{onSelectRow && (
												<input
													type="checkbox"
													checked={isSelected}
													style={{ marginLeft: 4, accentColor: "#6366f1" }}
													onClick={(e) => e.stopPropagation()}
													onChange={() => onSelectRow(asset.asset_id)}
												/>
											)}
										</div>
									</div>

									{/* Row 2: MCAP + duration + env + algo ratio */}
									<div
										style={{
											display: "flex",
											alignItems: "center",
											gap: 8,
											fontSize: 11,
											color: "#6b7280",
											flexWrap: "wrap",
										}}
									>
										<Tooltip title={`MCAP: ${asset.mcap_file_id}`}>
											<button
												type="button"
												className="link-like-button"
												style={{
													fontSize: 11,
													color: "#6366f1",
													background: "#f3f4f6",
													padding: "2px 6px",
													borderRadius: 4,
													border: "1px solid #e5e7eb",
													fontFamily: "monospace",
													cursor: "pointer",
												}}
												onClick={(e) => {
													e.stopPropagation();
													onMcapClick?.(asset.mcap_file_id);
												}}
											>
												📁 {asset.mcap_file_id.slice(0, 8)}…
											</button>
										</Tooltip>
										<span style={{ color: "#d1d5db" }}>|</span>
										<Text style={{ fontSize: 11, color: "#6b7280" }}>
											⏱ {formatDurationSeconds(asset)}
										</Text>
										<span style={{ color: "#d1d5db" }}>|</span>
										<Text style={{ fontSize: 11, color: "#6b7280" }}>
											📍 {asset.env ?? "—"}
										</Text>
										<span style={{ color: "#d1d5db" }}>|</span>
										<Text style={{ fontSize: 11, color: "#6b7280" }}>
											🔍 算法{" "}
											{totalAlgos > 0 ? `${algoOkCount}/${totalAlgos}` : "—"}
										</Text>
										{asset.delivery_count > 0 && (
											<>
												<span style={{ color: "#d1d5db" }}>|</span>
												<Text style={{ fontSize: 11, color: "#6b7280" }}>
													📦 {asset.delivery_count} 次交付
												</Text>
											</>
										)}
									</div>

									{/* Row 3: Key tag badges (capped) */}
									{tagEntries.length > 0 && (
										<div
											style={{
												display: "flex",
												flexWrap: "wrap",
												gap: 3,
												alignItems: "center",
											}}
										>
											{tagEntries.map(([key, value]) => (
												<Tag
													key={key}
													color={tagColor(key, value)}
													style={{
														fontSize: 10,
														lineHeight: "18px",
														margin: 0,
														padding: "0 6px",
													}}
												>
													{key}: {value}
												</Tag>
											))}
										</div>
									)}
								</div>
							</div>
						);
					})}
				</div>

				{/* Pagination */}
				<Pagination
					current={page}
					total={total}
					pageSize={pageSize}
					showSizeChanger
					showQuickJumper={total > 200}
					pageSizeOptions={["20", "50", "100", "200"]}
					showTotal={(t) => `共 ${t}${totalApprox ? "+" : ""} 条`}
					onChange={(p, ps) => onPageChange?.(p, ps)}
					style={{ marginTop: 16, textAlign: "center" }}
				/>
			</div>
		</Spin>
	);
}
