import { describe, expect, it } from "vitest";
import type { Pipeline } from "../components/pipeline/types";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "./pipelineContract";
import {
	validatePipelineForRun,
	validatePipelineForSave,
} from "./pipelineValidation";

function basePipeline(overrides: Partial<Pipeline> = {}): Pipeline {
	return {
		name: "test-pipeline",
		nodes: [],
		edges: [],
		...overrides,
	};
}

describe("pipeline validation", () => {
	it("allows saving a draft container without runnable command", () => {
		const pipeline = basePipeline({
			nodes: [
				{
					id: "step-1",
					component: {
						name: "Pass Through",
						image: "busybox:latest",
						command: ["sh", "-c"],
						args: [],
					},
				},
			],
		});

		expect(validatePipelineForSave(pipeline).valid).toBe(true);
		expect(validatePipelineForRun(pipeline).valid).toBe(false);
	});

	it("rejects duplicate node IDs for save", () => {
		const result = validatePipelineForSave(
			basePipeline({
				nodes: [
					{
						id: "step-1",
						component: { name: "left", image: "busybox" },
					},
					{
						id: "step-1",
						component: { name: "right", image: "busybox" },
					},
				],
			}),
		);

		expect(result.valid).toBe(false);
		expect(result.errors[0]).toContain("重复");
	});

	it("rejects edges referencing missing nodes for save", () => {
		const result = validatePipelineForSave(
			basePipeline({
				nodes: [
					{
						id: "step-1",
						component: { name: "left", image: "busybox" },
					},
				],
				edges: [{ source: "step-1.output", target: "missing.input" }],
			}),
		);

		expect(result.valid).toBe(false);
		expect(result.errors[0]).toContain("missing");
	});

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
						component: {
							name: "join",
							image: "busybox",
							command: ["sh", "-c", "echo join"],
						},
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

	it("rejects container nodes without runnable command", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "step-1",
						component: {
							name: "Pass Through",
							image: "busybox:latest",
							command: ["sh", "-c"],
							args: [],
						},
					},
				],
			}),
		);

		expect(result.valid).toBe(false);
		expect(result.errors[0]).toContain("Pass Through");
	});

	it("allows image-only container nodes to rely on image entrypoints", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "step-1",
						component: {
							name: "Image Entrypoint",
							image: "registry.example.com/image-entrypoint:latest",
							type: "container",
							command: [],
							args: [],
						},
					},
				],
			}),
		);

		expect(result.valid).toBe(true);
	});

	it("rejects consumed output ports without matching /tmp/outputs file writes", () => {
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

		expect(result.valid).toBe(false);
		expect(result.errors[0]).toContain("/tmp/outputs/output");
	});

	it("warns instead of blocking when an opaque container output is consumed", () => {
		const result = validatePipelineForRun(
			basePipeline({
				nodes: [
					{
						id: "producer",
						component: {
							name: "producer",
							image: "registry.example.com/producer:latest",
							type: "container",
							command: ["python", "src/main.py"],
							args: [],
						},
						outputs: [{ name: "output", type: "asset" }],
					},
					{
						id: "consumer",
						component: {
							name: "consumer",
							image: "registry.example.com/consumer:latest",
							type: "container",
							command: ["python", "src/main.py"],
							args: [],
						},
						inputs: [{ name: "input", type: "asset" }],
					},
				],
				edges: [{ source: "producer.output", target: "consumer.input" }],
			}),
		);

		expect(result.valid).toBe(true);
		expect(result.errors).toEqual([]);
		expect(result.warnings[0]).toContain("无法静态确认");
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
