import { describe, expect, it, vi } from "vitest";
import {
	batchJobDetailLocationState,
	goBackFromBatchJobDetail,
	goToPipelineBatchJobList,
	PIPELINE_BATCH_LIST_URL,
	resolveBatchJobDetailBackTarget,
	resolveWorkflowDetailBackTarget,
	workflowDetailLocationState,
} from "./pipelineNavigation";

describe("pipelineNavigation", () => {
	it("returns batch detail when opened from a batch subtask", () => {
		expect(
			resolveWorkflowDetailBackTarget(
				workflowDetailLocationState("job-123"),
			),
		).toBe("/pipeline/batch/job-123");
	});

	it("returns single execution list by default", () => {
		expect(resolveWorkflowDetailBackTarget(null)).toBe(
			"/pipeline?tab=executions",
		);
	});

	it("defines batch list url with executionView=batch", () => {
		expect(PIPELINE_BATCH_LIST_URL).toContain("executionView=batch");
	});

	it("resolves batch detail back target from location state", () => {
		expect(
			resolveBatchJobDetailBackTarget(
				batchJobDetailLocationState("/pipeline?tab=pipelines"),
			),
		).toBe("/pipeline?tab=pipelines");
		expect(resolveBatchJobDetailBackTarget(null)).toBe(
			PIPELINE_BATCH_LIST_URL,
		);
	});

	it("navigates to batch list with executions tab and batch view", () => {
		const navigate = vi.fn();
		const scrollTo = vi
			.spyOn(window, "scrollTo")
			.mockImplementation(() => undefined);
		goToPipelineBatchJobList(navigate);
		expect(navigate).toHaveBeenCalledWith({
			pathname: "/pipeline",
			search: "?tab=executions&executionView=batch",
		});
		expect(scrollTo).toHaveBeenCalledWith(0, 0);
		scrollTo.mockRestore();
	});

	it("navigates back from batch detail using returnTo state", () => {
		const navigate = vi.fn();
		const scrollTo = vi
			.spyOn(window, "scrollTo")
			.mockImplementation(() => undefined);
		goBackFromBatchJobDetail(
			navigate,
			batchJobDetailLocationState(PIPELINE_BATCH_LIST_URL),
		);
		expect(navigate).toHaveBeenCalledWith({
			pathname: "/pipeline",
			search: "?tab=executions&executionView=batch",
		});
		expect(scrollTo).toHaveBeenCalledWith(0, 0);
		scrollTo.mockRestore();
	});
});
