const WORKFLOW_EXECUTION_PATH = "/pipeline/executions/";

export function buildWorkflowExecutionUrl(
	workflowName: string,
	runId?: string,
): string {
	const trimmedRunId = runId?.trim();
	const encodedName = encodeURIComponent(workflowName);
	if (!trimmedRunId) {
		return `${WORKFLOW_EXECUTION_PATH}${encodedName}`;
	}
	const params = new URLSearchParams({ runId: trimmedRunId });
	return `${WORKFLOW_EXECUTION_PATH}${encodedName}?${params.toString()}`;
}
