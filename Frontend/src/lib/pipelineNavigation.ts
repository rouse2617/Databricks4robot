import type { NavigateFunction } from "react-router-dom";

export const PIPELINE_RUNS_URL = "/runs";
export const PIPELINE_EXECUTIONS_URL = PIPELINE_RUNS_URL;
export const PIPELINE_BATCH_LIST_URL = "/runs?executionView=batch";

export type WorkflowDetailLocationState = {
	fromBatchJobId?: string;
};

export type BatchJobDetailLocationState = {
	returnTo?: string;
};

export function resolveWorkflowDetailBackTarget(state: unknown): string {
	const batchId = (state as WorkflowDetailLocationState | null)?.fromBatchJobId;
	if (batchId) {
		return `/pipeline/batch/${batchId}`;
	}
	return PIPELINE_RUNS_URL;
}

export function workflowDetailLocationState(
	batchJobId?: string,
): WorkflowDetailLocationState | undefined {
	if (!batchJobId) return undefined;
	return { fromBatchJobId: batchJobId };
}

export function batchJobDetailLocationState(
	returnTo: string = PIPELINE_BATCH_LIST_URL,
): BatchJobDetailLocationState {
	return { returnTo };
}

export function resolveBatchJobDetailBackTarget(state: unknown): string {
	const returnTo = (state as BatchJobDetailLocationState | null)?.returnTo;
	return returnTo ?? PIPELINE_BATCH_LIST_URL;
}

export function goToPipelineBatchJobList(navigate: NavigateFunction): void {
	navigate({
		pathname: "/runs",
		search: "?executionView=batch",
	});
	if (typeof window !== "undefined") {
		window.scrollTo(0, 0);
	}
}

export function goBackFromBatchJobDetail(
	navigate: NavigateFunction,
	state: unknown,
): void {
	const target = resolveBatchJobDetailBackTarget(state);
	if (target.includes("?")) {
		const [pathname, search] = target.split("?");
		navigate({ pathname, search: `?${search}` });
	} else {
		navigate(target);
	}
	if (typeof window !== "undefined") {
		window.scrollTo(0, 0);
	}
}
