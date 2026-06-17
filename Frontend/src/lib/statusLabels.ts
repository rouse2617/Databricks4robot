import { STATUS_COLORS, WORKFLOW_PHASE_LABELS } from "./constants";
import {
	formatBusinessStatusLabel,
	resolveBusinessStatusTagColor,
} from "./productVocabulary";

export { formatBusinessStatusLabel, resolveBusinessStatusTagColor };

const BATCH_JOB_STATUS_LABELS: Record<string, string> = {
	running: "运行中",
	paused: "已暂停",
	completed: "已完成",
	failed: "失败",
};

const LOWERCASE_PHASE_LABELS: Record<string, string> = {
	running: "运行中",
	succeeded: "成功",
	failed: "失败",
	error: "异常",
	pending: "等待中",
	suspended: "已暂停",
	expired: "已过期",
	skipped: "已跳过",
	omitted: "已省略",
	cancelled: "已取消",
	canceled: "已取消",
};

/** Human-readable workflow / run phase label (Chinese). */
export function formatWorkflowPhaseLabel(phase: string | undefined): string {
	if (!phase) return "—";
	if (phase in WORKFLOW_PHASE_LABELS) {
		return WORKFLOW_PHASE_LABELS[phase as keyof typeof WORKFLOW_PHASE_LABELS];
	}
	const lower = phase.toLowerCase();
	if (lower in BATCH_JOB_STATUS_LABELS) {
		return BATCH_JOB_STATUS_LABELS[lower];
	}
	if (lower in LOWERCASE_PHASE_LABELS) {
		return LOWERCASE_PHASE_LABELS[lower];
	}
	return phase;
}

export function formatBatchJobStatus(status: string): string {
	return BATCH_JOB_STATUS_LABELS[status] ?? formatWorkflowPhaseLabel(status);
}

/** Ant Design Tag color for workflow phase or batch job status. */
export function resolveStatusTagColor(status: string): string {
	if (status in BATCH_JOB_STATUS_LABELS) {
		const map: Record<string, string> = {
			running: "processing",
			paused: "warning",
			completed: "success",
			failed: "error",
		};
		return map[status] ?? "default";
	}
	const pascal = status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
	return STATUS_COLORS[status] ?? STATUS_COLORS[pascal] ?? "default";
}
