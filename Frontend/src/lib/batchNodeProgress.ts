import type { PipelineRunNodeProgress } from "../api/pipelineApi";
import { formatWorkflowPhaseLabel } from "./statusLabels";

export function formatPipelineRunNodeProgress(
	progress?: PipelineRunNodeProgress | null,
): { text: string; tooltip?: string } {
	if (!progress) {
		return { text: "—" };
	}
	if (progress.label) {
		return { text: progress.label };
	}
	if (progress.focusNodeName && progress.focusStatus) {
		let text = `${progress.focusNodeName} · ${formatWorkflowPhaseLabel(progress.focusStatus)}`;
		if (progress.parallelRunning && progress.parallelRunning > 0) {
			text += ` (+${progress.parallelRunning})`;
		}
		return { text, tooltip: progress.message || undefined };
	}
	return { text: "—" };
}
