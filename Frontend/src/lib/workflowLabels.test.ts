import { describe, expect, it } from "vitest";
import {
	formatWorkflowLabelKey,
	getDisplayLabelEntries,
} from "./workflowLabels";

describe("workflowLabels", () => {
	it("hides noisy Argo system labels", () => {
		const entries = getDisplayLabelEntries({
			"workflows.argoproj.io/completed": "true",
			"workflows.argoproj.io/creator": "system",
			"workflows.argoproj.io/phase": "Failed",
			"workflows.argoproj.io/resubmitted-from-workflow": "wf-a",
		});
		expect(entries).toEqual([
			["workflows.argoproj.io/resubmitted-from-workflow", "wf-a"],
		]);
	});

	it("formats resubmit label key in Chinese", () => {
		expect(
			formatWorkflowLabelKey("workflows.argoproj.io/resubmitted-from-workflow"),
		).toBe("重提交自");
	});
});
