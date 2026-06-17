import { describe, expect, it } from "vitest";
import type { RegisteredComponent } from "../../components/pipeline/types";
import {
	dedupeComponents,
	defaultDeployWorkflowName,
} from "./pipelinePageHelpers";

function releaseComponent(
	overrides: Partial<RegisteredComponent> & {
		componentId: string;
		releaseId: string;
	},
): RegisteredComponent {
	return {
		id: overrides.releaseId,
		componentId: overrides.componentId,
		releaseId: overrides.releaseId,
		name: overrides.name ?? "smoke",
		image: "busybox:latest",
		command: ["sh", "-c", "echo ok"],
		args: [],
		cpu: "",
		memory: "",
		disk: "",
		source: "component-release",
		releaseLabel: overrides.releaseLabel ?? "v1",
		...overrides,
	};
}

describe("dedupeComponents", () => {
	it("keeps one release per component id", () => {
		const comps = dedupeComponents([
			releaseComponent({
				componentId: "comp-1",
				releaseId: "rel-1",
				releaseLabel: "v1",
			}),
			releaseComponent({
				componentId: "comp-1",
				releaseId: "rel-2",
				releaseLabel: "v2",
			}),
		]);

		expect(comps).toHaveLength(1);
		expect(comps[0]?.releaseId).toBe("rel-2");
	});
});

describe("defaultDeployWorkflowName", () => {
	it("uses pipeline timestamp prefix", () => {
		expect(defaultDeployWorkflowName(1_700_000_000_000)).toBe(
			"pipeline-1700000000000",
		);
	});
});
