import { describe, expect, it } from "vitest";
import type { Pipeline } from "../components/pipeline/types";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "./pipelineContract";
import { validatePipelineForRun } from "./pipelineValidation";

function basePipeline(overrides: Partial<Pipeline> = {}): Pipeline {
	return {
		name: "test-pipeline",
		nodes: [],
		edges: [],
		...overrides,
	};
}

describe("pipeline validation", () => {
	it("rejects multiple upstream edges targeting the same input parameter", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "left",
						component: {
							name: "left",
							image: "busybox",
							command: ["sh", "-c"],
							args: [
								{ name: "script", value: "echo left > /tmp/outputs/output" },
							],
						},
						outputs: [{ name: "output", type: "string" }],
					},
					{
						id: "right",
						component: {
							name: "right",
							image: "busybox",
							command: ["sh", "-c"],
							args: [
								{ name: "script", value: "echo right > /tmp/outputs/output" },
							],
						},
						outputs: [{ name: "output", type: "string" }],
					},
					{
						id: "join",
						component: { name: "join", image: "busybox" },
						inputs: [{ name: "input", type: "string" }],
					},
				],
				edges: [
					{ source: "left.output", target: "join.input" },
					{ source: "right.output", target: "join.input" },
				],
			}),
		);

		expect(result.valid).toBe(false);
		expect(result.errors[0]).toContain("join.input");
	});

	it("allows fan-in when each edge targets a distinct input", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "left",
						component: {
							name: "left",
							image: "busybox",
							command: ["sh", "-c"],
							args: [
								{ name: "script", value: "echo left > /tmp/outputs/output" },
							],
						},
						outputs: [{ name: "output", type: "string" }],
					},
					{
						id: "right",
						component: {
							name: "right",
							image: "busybox",
							command: ["sh", "-c"],
							args: [
								{ name: "script", value: "echo right > /tmp/outputs/output" },
							],
						},
						outputs: [{ name: "output", type: "string" }],
					},
					{
						id: "join",
						component: { name: "join", image: "busybox" },
						inputs: [
							{ name: "left", type: "string" },
							{ name: "right", type: "string" },
						],
					},
				],
				edges: [
					{ source: "left.output", target: "join.left" },
					{ source: "right.output", target: "join.right" },
				],
			}),
		);

		expect(result.valid).toBe(true);
	});

	it("warns when consumed output ports lack matching /tmp/outputs file writes", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "producer",
						component: {
							name: "producer",
							image: "busybox",
							command: ["sh", "-c"],
							args: [{ name: "script", value: "echo missing file" }],
						},
						outputs: [{ name: "output", type: "string" }],
					},
					{
						id: "consumer",
						component: { name: "consumer", image: "busybox" },
						inputs: [{ name: "input", type: "string" }],
					},
				],
				edges: [{ source: "producer.output", target: "consumer.input" }],
			}),
		);

		expect(result.valid).toBe(true);
		expect(result.warnings[0]).toContain("/tmp/outputs/output");
	});
});

describe("pipeline shell argument normalization", () => {
	it("keeps only the script body when command is sh -c and args include sh -c", () => {
		const args = normalizeShellCommandArgs(
			["sh", "-c"],
			normalizeComponentArgs(["sh", "-c", "echo ok"]),
		);

		expect(args).toEqual([{ name: "echo ok", value: "echo ok" }]);
	});
});
