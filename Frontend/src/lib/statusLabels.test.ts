import { describe, expect, it } from "vitest";
import {
	formatBatchJobStatus,
	formatWorkflowPhaseLabel,
	resolveStatusTagColor,
} from "./statusLabels";

describe("statusLabels", () => {
	it("maps workflow phases to Chinese", () => {
		expect(formatWorkflowPhaseLabel("Succeeded")).toBe("成功");
		expect(formatWorkflowPhaseLabel("Running")).toBe("运行中");
		expect(formatWorkflowPhaseLabel("failed")).toBe("失败");
	});

	it("maps batch job statuses", () => {
		expect(formatBatchJobStatus("completed")).toBe("已完成");
		expect(formatBatchJobStatus("running")).toBe("运行中");
	});

	it("resolves tag colors for batch and workflow statuses", () => {
		expect(resolveStatusTagColor("completed")).toBe("success");
		expect(resolveStatusTagColor("Succeeded")).toBe("success");
		expect(resolveStatusTagColor("Running")).toBe("blue");
	});
});
