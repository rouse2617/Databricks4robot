import { Checkbox, Descriptions, Divider, Segmented, Typography } from "antd";
import { memo, useMemo, useState } from "react";
import TimeSeriesChart, { generateMockData } from "./TimeSeriesChart";
import Viewport3D from "./Viewport3D";

const { Text } = Typography;

interface SidebarChannel {
	topic: string;
	label: string;
}

interface SidebarPanelProps {
	channels: SidebarChannel[];
	activeTopics: string[];
	onToggle: (topic: string) => void;
	metadata: {
		durationMs: number;
		fileSize?: number;
		channelCount: number;
		assetId: string;
	};
}

function fmtSize(bytes?: number): string {
	if (!bytes) return "-";
	const gb = bytes / 1e9;
	if (gb > 1) return `${gb.toFixed(1)} GB`;
	const mb = bytes / 1e6;
	if (mb > 1) return `${mb.toFixed(0)} MB`;
	return `${(bytes / 1e3).toFixed(0)} KB`;
}

function fmtDuration(ms: number): string {
	if (!ms) return "-";
	const s = Math.round(ms / 1000);
	const m = Math.floor(s / 60);
	const sec = s % 60;
	return m > 0 ? `${m}m ${sec}s` : `${sec}s`;
}

function SidebarPanel({
	channels,
	activeTopics,
	onToggle,
	metadata,
}: SidebarPanelProps) {
	const [vizMode, setVizMode] = useState("3d");

	// Mock chart data is static — generate once, not on every render.
	const combinedData = useMemo<[number[], number[], number[]]>(() => {
		const mockData = generateMockData(500, 0.3);
		const mockData2 = generateMockData(500, 0.8);
		return [mockData[0], mockData[1], mockData2[1]];
	}, []);

	return (
		<div
			style={{
				padding: 16,
				display: "flex",
				flexDirection: "column",
				gap: 16,
				overflow: "auto",
			}}
		>
			{/* File info section */}
			<div>
				<Text
					strong
					style={{
						fontSize: "var(--font-size-base)",
						color: "var(--gray-800)",
					}}
				>
					MCAP 文件信息
				</Text>
				<Descriptions
					column={1}
					size="small"
					style={{ marginTop: 8 }}
					styles={{
						label: {
							fontSize: "var(--font-size-sm)",
							color: "var(--gray-500)",
							paddingBottom: 4,
						},
						content: {
							fontSize: "var(--font-size-sm)",
							fontFamily: "var(--font-mono)",
							color: "var(--gray-700)",
						},
					}}
				>
					<Descriptions.Item label="资产 ID">
						{metadata.assetId.slice(0, 16)}…
					</Descriptions.Item>
					<Descriptions.Item label="时长">
						{fmtDuration(metadata.durationMs)}
					</Descriptions.Item>
					<Descriptions.Item label="大小">
						{fmtSize(metadata.fileSize)}
					</Descriptions.Item>
					<Descriptions.Item label="通道数">
						{metadata.channelCount}
					</Descriptions.Item>
				</Descriptions>
			</div>

			<Divider style={{ margin: 0 }} />

			{/* Channel list */}
			<div>
				<Text
					strong
					style={{
						fontSize: "var(--font-size-base)",
						color: "var(--gray-800)",
					}}
				>
					通道列表
				</Text>
				<div
					style={{
						marginTop: 8,
						display: "flex",
						flexDirection: "column",
						gap: 4,
					}}
				>
					{channels.map((ch) => (
						<button
							key={ch.topic}
							type="button"
							style={{
								display: "flex",
								alignItems: "center",
								gap: 8,
								padding: "6px 8px",
								borderRadius: "var(--radius-sm)",
								background: activeTopics.includes(ch.topic)
									? "var(--color-primary-light)"
									: "transparent",
								cursor: "pointer",
								transition: "background 150ms ease",
								border: "none",
								width: "100%",
								textAlign: "left",
							}}
							onClick={() => onToggle(ch.topic)}
						>
							<Checkbox checked={activeTopics.includes(ch.topic)} />
							<span
								style={{
									fontSize: "var(--font-size-sm)",
									color: "var(--gray-700)",
								}}
							>
								📷 {ch.label}
							</span>
							<span
								style={{
									marginLeft: "auto",
									fontSize: "var(--font-size-xs)",
									color: "var(--gray-400)",
									fontFamily: "var(--font-mono)",
								}}
							>
								{ch.topic.split("/").filter(Boolean).pop()}
							</span>
						</button>
					))}
				</div>
			</div>

			<Divider style={{ margin: 0 }} />

			{/* Visualization / Data panels */}
			<div>
				<Segmented
					value={vizMode}
					onChange={(v) => setVizMode(v as string)}
					options={[
						{ value: "3d", label: "3D" },
						{ value: "chart", label: "图表" },
					]}
					size="small"
					block
					style={{ marginBottom: 8 }}
				/>
				{vizMode === "3d" ? (
					<div
						style={{
							height: 300,
							borderRadius: "var(--radius-md)",
							overflow: "hidden",
						}}
					>
						<Viewport3D />
					</div>
				) : vizMode === "chart" ? (
					<TimeSeriesChart
						title="IMU 加速度"
						data={combinedData}
						series={["X轴", "Y轴"]}
						height={280}
					/>
				) : null}
			</div>
		</div>
	);
}

// Memoized: props are stabilized by the parent (metadata via useMemo, callbacks
// via useCallback), so the sidebar — and its 3D viewport / charts — no longer
// re-renders on every currentTime tick.
export default memo(SidebarPanel);
