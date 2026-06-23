export function runDetailPath(
	runId?: string | null,
	workflowName?: string | null,
): string {
	if (runId) {
		return `/runs/${encodeURIComponent(runId)}`;
	}
	if (workflowName) {
		return `/pipeline/executions/${encodeURIComponent(workflowName)}`;
	}
	return "/pipeline?tab=executions";
}
