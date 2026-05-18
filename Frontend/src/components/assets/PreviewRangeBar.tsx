import { Typography } from "antd";

const { Text } = Typography;

export interface PreviewRange {
	startSec: number;
	endSec: number;
}

export function PreviewRangeBar({
	ranges,
	startSec,
	endSec,
	playheadSec,
}: {
	ranges: PreviewRange[];
	startSec: number;
	endSec: number;
	playheadSec: number;
}) {
	const total = Math.max(0.001, endSec - startSec);
	const clampedPlayhead = Math.min(Math.max(playheadSec, startSec), endSec);
	const markerLeft = ((clampedPlayhead - startSec) / total) * 100;

	return (
		<div style={{ width: "100%" }}>
			<div
				style={{
					position: "relative",
					height: 8,
					borderRadius: 6,
					background: "#f0f0f0",
					overflow: "hidden",
				}}
			>
				{ranges.map((r) => {
					const left = ((r.startSec - startSec) / total) * 100;
					const width = ((r.endSec - r.startSec) / total) * 100;
					return (
						<div
							key={`${r.startSec}-${r.endSec}`}
							style={{
								position: "absolute",
								left: `${Math.max(0, left)}%`,
								width: `${Math.max(0, width)}%`,
								top: 0,
								bottom: 0,
								background: "#69b1ff",
							}}
						/>
					);
				})}
				<div
					style={{
						position: "absolute",
						left: `${markerLeft}%`,
						width: 2,
						top: -2,
						bottom: -2,
						background: "#f5222d",
						transform: "translateX(-50%)",
					}}
				/>
			</div>
			<div
				style={{
					marginTop: 4,
					display: "flex",
					justifyContent: "space-between",
				}}
			>
				<Text type="secondary" style={{ fontSize: 11 }}>
					{startSec.toFixed(1)}s
				</Text>
				<Text type="secondary" style={{ fontSize: 11 }}>
					{endSec.toFixed(1)}s
				</Text>
			</div>
		</div>
	);
}
