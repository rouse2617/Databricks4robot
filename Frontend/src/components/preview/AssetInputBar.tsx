import { SearchOutlined } from "@ant-design/icons";
import { Button, Input, Tooltip, Typography } from "antd";
import { useEffect, useState } from "react";

const { Text } = Typography;

interface AssetInputBarProps {
	onLoad: (assetId: string) => void;
	loading: boolean;
	/** Current range (start/end in seconds), null if not set */
	range?: { startSec: number; endSec: number } | null;
	onRangeChange?: (range: { startSec: number; endSec: number } | null) => void;
	/** Absolute start timestamp (nanoseconds) — used to convert Unix timestamps */
	startTimestampNs?: number;
}

function fmtRange(sec: number, startNs: number): string {
	if (startNs > 0) {
		const ms = Math.floor(startNs / 1e6 + sec * 1000);
		if (ms > 1000000000000) return `${Math.floor(ms / 1000)}`;
		return `${ms}`;
	}
	const h = Math.floor(sec / 3600);
	const m = Math.floor((sec % 3600) / 60);
	const s = Math.floor(sec % 60);
	if (h > 0)
		return `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
	return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

/**
 * Parse user input — supports:
 * - "MM:SS" or "HH:MM:SS" → relative seconds
 * - bare number < 100000 → relative seconds
 * - bare number ≥ 100000 → Unix timestamp ms/s → relative seconds (via startNs)
 */
function parseTimeInput(text: string, startNs: number): number | null {
	const t = text.trim();
	if (!t) return null;
	// HH:MM:SS or MM:SS
	const parts = t.split(":").map(Number);
	if (parts.length === 3 && parts.every((n) => Number.isFinite(n)))
		return parts[0] * 3600 + parts[1] * 60 + parts[2];
	if (parts.length === 2 && parts.every((n) => Number.isFinite(n)))
		return parts[0] * 60 + parts[1];
	// bare number
	const n = parseFloat(t);
	if (!Number.isFinite(n) || n < 0) return null;
	// If number looks like a Unix timestamp (seconds since epoch), convert to relative
	if (n > 1000000000 && startNs > 0) return (n * 1e9 - startNs) / 1e9;
	if (n > 1000000000000 && startNs > 0) return (n * 1e6 - startNs) / 1e9;
	return n; // relative seconds
}

export default function AssetInputBar({
	onLoad,
	loading,
	range,
	onRangeChange,
	startTimestampNs,
}: AssetInputBarProps) {
	const [assetID, setAssetID] = useState("");
	const [startText, setStartText] = useState("");
	const [endText, setEndText] = useState("");
	const [startEdit, setStartEdit] = useState(false);
	const [endEdit, setEndEdit] = useState(false);

	// Sync text when range changes externally
	useEffect(() => {
		if (range && !startEdit)
			setStartText(fmtRange(range.startSec, startTimestampNs || 0));
		if (range && !endEdit)
			setEndText(fmtRange(range.endSec, startTimestampNs || 0));
	}, [
		range?.startSec,
		range?.endSec,
		startEdit,
		endEdit,
		startTimestampNs,
		range,
	]);

	const handleLoad = () => {
		const id = assetID.trim();
		if (!id) return;
		onLoad(id);
	};

	const submitStart = () => {
		setStartEdit(false);
		const raw = parseTimeInput(startText, startTimestampNs || 0);
		if (raw === null || !range) return;
		const sec = Math.max(0, raw);
		const end = range.endSec;
		onRangeChange?.({
			startSec: Math.min(sec, end),
			endSec: Math.max(sec, end),
		});
	};

	const submitEnd = () => {
		setEndEdit(false);
		const raw = parseTimeInput(endText, startTimestampNs || 0);
		if (raw === null || !range) return;
		const sec = Math.max(0, raw);
		const start = range.startSec;
		onRangeChange?.({
			startSec: Math.min(start, sec),
			endSec: Math.max(start, sec),
		});
	};

	return (
		<div
			style={{
				display: "flex",
				alignItems: "center",
				gap: 10,
				padding: "var(--space-3) var(--space-6)",
				borderBottom: "1px solid var(--gray-200)",
				background: "var(--gray-50)",
				flexWrap: "wrap",
			}}
		>
			<Text
				strong
				style={{
					fontSize: "var(--font-size-xl)",
					whiteSpace: "nowrap",
					color: "var(--gray-900)",
				}}
			>
				视频预览
			</Text>

			<Input
				placeholder="资产 ID"
				value={assetID}
				onChange={(e) => setAssetID(e.target.value)}
				onPressEnter={handleLoad}
				allowClear
				style={{
					maxWidth: 280,
					fontFamily: "var(--font-mono)",
					fontSize: "var(--font-size-sm)",
				}}
				size="small"
				prefix={<SearchOutlined style={{ color: "var(--gray-400)" }} />}
			/>
			<Button
				type="primary"
				onClick={handleLoad}
				loading={loading}
				size="small"
			>
				加载
			</Button>

			{/* Range start */}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 2,
					borderLeft: "1px solid var(--gray-200)",
					paddingLeft: 10,
				}}
			>
				<Text type="secondary" style={{ fontSize: "var(--font-size-xs)" }}>
					区间
				</Text>
				{startEdit ? (
					<Input
						size="small"
						value={startText}
						onChange={(e) => setStartText(e.target.value)}
						onPressEnter={submitStart}
						onBlur={submitStart}
						autoFocus
						style={{
							width: 180,
							fontFamily: "var(--font-mono)",
							fontSize: "var(--font-size-sm)",
						}}
					/>
				) : (
					<Text
						onClick={() => {
							if (range) {
								setStartText(fmtRange(range.startSec, startTimestampNs || 0));
								setStartEdit(true);
							} else {
								onRangeChange?.({ startSec: 0, endSec: 10 });
							}
						}}
						style={{
							cursor: "pointer",
							fontFamily: "var(--font-mono)",
							fontSize: "var(--font-size-sm)",
							color: range ? "var(--color-primary)" : "var(--gray-400)",
							borderBottom: range ? "1px dashed var(--color-primary)" : "none",
							minWidth: 80,
							textAlign: "center",
						}}
					>
						{range ? fmtRange(range.startSec, startTimestampNs || 0) : "—"}
					</Text>
				)}
				<Text type="secondary" style={{ fontSize: "var(--font-size-xs)" }}>
					~
				</Text>
				{/* Range end */}
				{endEdit ? (
					<Input
						size="small"
						value={endText}
						onChange={(e) => setEndText(e.target.value)}
						onPressEnter={submitEnd}
						onBlur={submitEnd}
						autoFocus
						style={{
							width: 180,
							fontFamily: "var(--font-mono)",
							fontSize: "var(--font-size-sm)",
						}}
					/>
				) : (
					<Text
						onClick={() => {
							if (range) {
								setEndText(fmtRange(range.endSec, startTimestampNs || 0));
								setEndEdit(true);
							} else {
								onRangeChange?.({ startSec: 0, endSec: 10 });
							}
						}}
						style={{
							cursor: "pointer",
							fontFamily: "var(--font-mono)",
							fontSize: "var(--font-size-sm)",
							color: range ? "var(--color-primary)" : "var(--gray-400)",
							borderBottom: range ? "1px dashed var(--color-primary)" : "none",
							minWidth: 80,
							textAlign: "center",
						}}
					>
						{range ? fmtRange(range.endSec, startTimestampNs || 0) : "—"}
					</Text>
				)}

				{/* Clear range */}
				{range && (
					<Tooltip title="清除区间">
						<Button
							type="text"
							size="small"
							onClick={() => onRangeChange?.(null)}
							style={{
								color: "var(--gray-400)",
								width: 20,
								height: 20,
								fontSize: 12,
								marginLeft: 2,
							}}
						>
							×
						</Button>
					</Tooltip>
				)}
			</div>
		</div>
	);
}
