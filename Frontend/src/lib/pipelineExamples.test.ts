import { describe, expect, it } from "vitest";
import { PIPELINE_EXAMPLES } from "./pipelineExamples";
import { validatePipelineForRun } from "./pipelineValidation";

describe("pipeline examples", () => {
	it("ships runnable standard examples", () => {
		for (const example of PIPELINE_EXAMPLES) {
			const result = validatePipelineForRun(example.pipeline);
			expect(result.errors, example.key).toEqual([]);
			expect(result.valid, example.key).toBe(true);
		}
	});

	it("uses distinct target inputs for the fan-in example", () => {
		const fanIn = PIPELINE_EXAMPLES.find((example) => example.key === "fan-in");
		expect(fanIn).toBeTruthy();
		expect(fanIn?.pipeline.edges).toContainEqual({
			source: "extract-left.output",
			target: "join.left",
		});
		expect(fanIn?.pipeline.edges).toContainEqual({
			source: "extract-right.output",
			target: "join.right",
		});
	});

	it("keeps observation example nodes at ten seconds or longer", () => {
		const observation = PIPELINE_EXAMPLES.find(
			(example) => example.key === "observation",
		);
		expect(observation).toBeTruthy();
		for (const node of observation?.pipeline.nodes ?? []) {
			const script = node.component.args?.[0]?.value ?? "";
			expect(script).toMatch(/sleep 1[0-9]/);
			for (const output of node.outputs ?? []) {
				expect(script).toContain(`/tmp/outputs/${output.name}`);
			}
		}
	});
});
