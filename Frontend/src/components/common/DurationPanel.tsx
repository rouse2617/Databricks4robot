import { Progress } from "antd";
import { formatDuration } from "../../lib/workflow-utils";

function parseProgress(progress?: string): number | undefined {
	if (!progress) return undefined;
	const [done, total] = progress.split("/").map((value) => Number(value));
	if (!Number.isFinite(done) || !Number.isFinite(total) || total <= 0) {
		return undefined;
	}
	return Math.min(100, Math.round((done / total) * 100));
}

export function DurationPanel({
	phase,
	startedAt,
	finishedAt,
	progress,
}: {
	phase?: string;
	startedAt?: string;
	finishedAt?: string;
	progress?: string;
}) {
	const duration = formatDuration(startedAt, finishedAt);
	const percent = phase === "Running" ? parseProgress(progress) : undefined;

	return (
		<span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
			{percent !== undefined && (
				<Progress
					percent={percent}
					size="small"
					showInfo={false}
					style={{ width: 72 }}
				/>
			)}
			<span>{duration}</span>
		</span>
	);
}
