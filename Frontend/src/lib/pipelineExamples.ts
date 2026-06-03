import type { Pipeline } from "../components/pipeline/types";

export type PipelineExample = {
	key: string;
	label: string;
	description: string;
	pipeline: Pipeline;
	layout: Record<string, { x: number; y: number }>;
};

const shell = ["sh", "-c"];

function script(value: string) {
	return [{ name: "script", value }];
}

export const PIPELINE_EXAMPLES: PipelineExample[] = [
	{
		key: "sequential",
		label: "顺序处理链路",
		description: "读取资产、转换、汇总，适合验证基础依赖和输出传递。",
		layout: {
			"read-asset": { x: 120, y: 120 },
			transform: { x: 360, y: 120 },
			summarize: { x: 600, y: 120 },
		},
		pipeline: {
			name: "example-sequential-pipeline",
			version: "1",
			nodes: [
				{
					id: "read-asset",
					component: {
						name: "read-asset",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 10; echo raw-asset > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "asset" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "transform",
					component: {
						name: "transform",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 12; echo transformed > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "summarize",
					component: {
						name: "summarize",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 10; echo done > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
			],
			edges: [
				{ source: "read-asset.output", target: "transform.input" },
				{ source: "transform.output", target: "summarize.input" },
			],
		},
	},
	{
		key: "fan-in",
		label: "Fan-in 汇聚链路",
		description: "两个分支写出不同结果，再汇聚到 join.left / join.right。",
		layout: {
			ingest: { x: 120, y: 180 },
			"extract-left": { x: 360, y: 80 },
			"extract-right": { x: 360, y: 280 },
			join: { x: 640, y: 180 },
		},
		pipeline: {
			name: "example-fan-in-join",
			version: "1",
			nodes: [
				{
					id: "ingest",
					component: {
						name: "ingest",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 10; echo asset > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "asset" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "extract-left",
					component: {
						name: "extract-left",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 14; echo left > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "extract-right",
					component: {
						name: "extract-right",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 16; echo right > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "join",
					component: {
						name: "join-results",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 10; echo joined > /tmp/outputs/output"),
					},
					inputs: [
						{ name: "left", type: "string" },
						{ name: "right", type: "string" },
					],
					outputs: [{ name: "output", type: "string" }],
				},
			],
			edges: [
				{ source: "ingest.output", target: "extract-left.input" },
				{ source: "ingest.output", target: "extract-right.input" },
				{ source: "extract-left.output", target: "join.left" },
				{ source: "extract-right.output", target: "join.right" },
			],
		},
	},
	{
		key: "observation",
		label: "10 秒观测链路",
		description: "四个 10s+ 节点，适合观察日志、事件、耗时和成本汇总。",
		layout: {
			prepare: { x: 120, y: 140 },
			analyze: { x: 360, y: 140 },
			"quality-check": { x: 600, y: 140 },
			publish: { x: 840, y: 140 },
		},
		pipeline: {
			name: "example-observation-10s",
			version: "1",
			nodes: [
				{
					id: "prepare",
					component: {
						name: "prepare",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 11; echo prepared > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "asset" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "analyze",
					component: {
						name: "analyze",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 13; echo analyzed > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "quality-check",
					component: {
						name: "quality-check",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 15; echo pass > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
				{
					id: "publish",
					component: {
						name: "publish",
						image: "busybox:latest",
						type: "container",
						source: "example",
						command: shell,
						args: script("sleep 12; echo published > /tmp/outputs/output"),
					},
					inputs: [{ name: "input", type: "string" }],
					outputs: [{ name: "output", type: "string" }],
				},
			],
			edges: [
				{ source: "prepare.output", target: "analyze.input" },
				{ source: "analyze.output", target: "quality-check.input" },
				{ source: "quality-check.output", target: "publish.input" },
			],
		},
	},
];

export function getPipelineExample(key: string) {
	return PIPELINE_EXAMPLES.find((example) => example.key === key) ?? null;
}
