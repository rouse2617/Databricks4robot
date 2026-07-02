import { describe, expect, it } from "vitest";
import {
	getWorkflowOperationConfirmText,
	isWorkflowRetryEnabled,
	isWorkflowStopped,
	workflowHasRetryableFailedNodes,
	workflowShowsRetryProgress,
} from "./workflow-operations";

describe("workflow retry eligibility", () => {
	it("detects stopped workflows", () => {
		expect(
			isWorkflowStopped({ name: "wf", status: "Failed", message: "Stopped" }),
		).toBe(true);
	});

	it("disables retry for stopped failed workflows", () => {
		expect(
			isWorkflowRetryEnabled({
				name: "wf",
				status: "Failed",
				message: "Stopped",
				nodes: [{ phase: "Pending", type: "Pod" }],
			}),
		).toBe(false);
	});

	it("enables retry when failed pod nodes exist", () => {
		expect(
			isWorkflowRetryEnabled({
				name: "wf",
				status: "Failed",
				nodes: [{ phase: "Failed", type: "Pod" }],
			}),
		).toBe(true);
	});

	it("ignores dag container nodes when checking retryable failures", () => {
		expect(
			workflowHasRetryableFailedNodes({
				name: "wf",
				status: "Failed",
				nodes: [{ phase: "Failed", type: "DAG" }],
			}),
		).toBe(false);
	});

	it("detects retry progress when workflow becomes running", () => {
		expect(
			workflowShowsRetryProgress(
				{ name: "wf", status: "Failed" },
				{ name: "wf", status: "Running" },
			),
		).toBe(true);
	});
});

describe("getWorkflowOperationConfirmText", () => {
	it("differentiates retry and resubmit copy", () => {
		expect(getWorkflowOperationConfirmText("retry")).toContain("失败");
		expect(getWorkflowOperationConfirmText("resubmit")).toContain("全新执行");
		expect(getWorkflowOperationConfirmText("rerun")).toContain(
			"产品级全量重跑",
		);
	});
});
