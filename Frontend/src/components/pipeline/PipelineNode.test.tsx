// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PipelineStepNode } from "./PipelineNode";
import type { PipelineNodeData } from "./types";

vi.mock("@xyflow/react", () => ({
	Handle: ({ type }: { type: string }) => (
		<div data-testid={`handle-${type}`} />
	),
	Position: { Left: "left", Right: "right" },
}));

describe("PipelineStepNode", () => {
	const baseData: PipelineNodeData = {
		label: "数据预处理",
		image: "python:3.11",
	};

	it("renders node label in aria-label", () => {
		render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={baseData}
				selected={false}
				isConnectable
			/>,
		);
		expect(screen.getByText("数据预处理")).toBeTruthy();
	});

	it("renders fallback label when missing", () => {
		render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={{}}
				selected={false}
				isConnectable
			/>,
		);
		expect(screen.getByText("未命名步骤")).toBeTruthy();
	});

	it("applies selected class when selected", () => {
		const { container } = render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={baseData}
				selected
				isConnectable
			/>,
		);
		expect(container.querySelector(".pipeline-node")?.className).toContain(
			"selected",
		);
	});

	it("has aria-selected attribute", () => {
		render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={baseData}
				selected
				isConnectable
			/>,
		);
		expect(screen.getByRole("option").getAttribute("aria-selected")).toBe(
			"true",
		);
	});

	it("renders handles", () => {
		render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={baseData}
				selected={false}
				isConnectable
			/>,
		);
		expect(screen.getByTestId("handle-target")).toBeTruthy();
		expect(screen.getByTestId("handle-source")).toBeTruthy();
	});

	it("renders runtime config badge when node has a binding", () => {
		render(
			<PipelineStepNode
				id="n1"
				type="pipelineStep"
				data={{
					...baseData,
					runtimeConfig: {
						mode: "saved",
						configId: "cfg-1",
						version: 1,
						fileName: "detector.yaml",
						mountPath: "/workspace/configs",
						targetFilename: "detector.yaml",
						displayName: "Detector Config",
					},
				}}
				selected={false}
				isConnectable
			/>,
		);
		expect(screen.getByText("配置 Detector Config")).toBeTruthy();
	});
});
