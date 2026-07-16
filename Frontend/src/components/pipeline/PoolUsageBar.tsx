import { Typography } from "antd";

const { Text } = Typography;

export interface PoolUsageBarProps {
	// Optional dimension label rendered to the left of the bar (e.g. "CPU").
	label?: string;
	used: string;
	min: string;
	max: string;
	// 0–100 utilisation; clamped defensively.
	percent: number;
}

// PoolUsageBar is a compact used / min / max utilisation bar for one dimension
// of a Koordinator ElasticQuota. It mirrors the QuotaBar in PoolManager's
// admin-facing ElasticQuotaPanel so deployers see the same usage view inline in
// the Deploy flow (CYB-3486). Kept standalone (not imported from PoolManager)
// so the two panels can evolve independently.
export function PoolUsageBar({
	label,
	used,
	min,
	max,
	percent,
}: PoolUsageBarProps) {
	const pct = Math.max(0, Math.min(100, percent));
	const color = pct > 80 ? "#ef4444" : pct > 60 ? "#f59e0b" : "#22c55e";
	return (
		<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
			{label ? (
				<Text
					style={{
						fontSize: 11,
						minWidth: 30,
						color: "var(--gray-500)",
					}}
				>
					{label}
				</Text>
			) : null}
			<div
				style={{
					flex: 1,
					height: 8,
					background: "#e5e7eb",
					borderRadius: 4,
					overflow: "hidden",
				}}
			>
				<div
					style={{
						width: `${pct}%`,
						height: "100%",
						background: color,
						borderRadius: 4,
						transition: "width 0.3s",
					}}
				/>
			</div>
			<Text
				style={{
					fontSize: 11,
					fontFamily: "var(--font-mono)",
					color,
					minWidth: 96,
					textAlign: "right",
				}}
			>
				{used} / {min} / {max}
			</Text>
		</div>
	);
}
