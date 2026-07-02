import { Progress, Tooltip } from "antd";
import { formatDuration } from "../../lib/workflow-utils";

function parseProgress(progress?: string): number | undefined {
	if (!progress) return undefined;
	const [done, total] = progress.split("/").map((value) => Number(value));
	if (!Number.isFinite(done) || !Number.isFinite(total) || total <= 0) {
		return undefined;
	}
	return Math.min(100, Math.round((done / total) * 100));
}

function diffSeconds(from?: string, to?: string): number | undefined {
	if (!from) return undefined;
	const start = new Date(from).getTime();
	const end = to ? new Date(to).getTime() : Date.now();
	if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) {
		return undefined;
	}
	return Math.floor((end - start) / 1000);
}

function humanizeSeconds(seconds?: number): string {
	if (seconds === undefined) return "-";
	const hours = Math.floor(seconds / 3600);
	const minutes = Math.floor((seconds % 3600) / 60);
	const secs = seconds % 60;
	if (hours > 0) return `${hours}h ${minutes}m ${secs}s`;
	if (minutes > 0) return `${minutes}m ${secs}s`;
	return `${secs}s`;
}

/**
 * 区分「排队等待」与「实际执行」两段时间。
 * - createdAt：进入队列的时间（提交时刻）
 * - startedAt：真正开始执行（首个 Pod 起跑）的时间
 * - finishedAt：结束时间
 * 主显示采用 startedAt→finishedAt 的执行耗时；createdAt→startedAt 的排队等待
 * 通过 Tooltip 透出，避免把排队时间误算进「耗时」。
 */
export function DurationPanel({
	phase,
	createdAt,
	startedAt,
	finishedAt,
	progress,
}: {
	phase?: string;
	createdAt?: string;
	startedAt?: string;
	finishedAt?: string;
	progress?: string;
}) {
	const percent = phase === "Running" ? parseProgress(progress) : undefined;

	// 尚未开始执行（仍在排队）：主显示排队等待时长，并标注「排队中」。
	if (!startedAt) {
		const queued = diffSeconds(createdAt, finishedAt);
		const label = phase === "Pending" ? "排队中" : "等待中";
		return (
			<Tooltip title={`${label}：尚未开始执行`}>
				<span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
					<span>{humanizeSeconds(queued)}</span>
					<span style={{ color: "#9ca3af", fontSize: 11 }}>{label}</span>
				</span>
			</Tooltip>
		);
	}

	const execDuration = formatDuration(startedAt, finishedAt);
	const queueSeconds = diffSeconds(createdAt, startedAt);
	const tooltip =
		queueSeconds !== undefined
			? `排队等待 ${humanizeSeconds(queueSeconds)} · 实际执行 ${execDuration}`
			: `实际执行 ${execDuration}`;

	return (
		<Tooltip title={tooltip}>
			<span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
				{percent !== undefined && (
					<Progress
						percent={percent}
						size="small"
						showInfo={false}
						style={{ width: 72 }}
					/>
				)}
				<span>{execDuration}</span>
				{queueSeconds !== undefined && queueSeconds >= 5 ? (
					<span style={{ color: "#9ca3af", fontSize: 11 }}>
						+排队 {humanizeSeconds(queueSeconds)}
					</span>
				) : null}
			</span>
		</Tooltip>
	);
}
