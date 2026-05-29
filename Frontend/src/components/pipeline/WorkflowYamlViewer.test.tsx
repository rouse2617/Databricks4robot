// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { WorkflowYamlViewer } from "./WorkflowYamlViewer";
import type { WorkflowNodeStatus } from "../../api/workflowApi";

Object.defineProperty(window, "matchMedia", {
	writable: true,
	value: vi.fn().mockImplementation((query: string) => ({
		matches: false, media: query, onchange: null,
		addListener: vi.fn(), removeListener: vi.fn(),
		addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
	})),
});

describe("WorkflowYamlViewer", () => {
	const baseNode: WorkflowNodeStatus = {
		id: "node-1",
		name: "test-node",
		displayName: "数据预处理",
		phase: "Succeeded",
		type: "Pod",
	};

	it("renders nothing when nodeData is null", () => {
		const { container } = render(<WorkflowYamlViewer nodeData={null} onClose={vi.fn()} />);
		expect(container.querySelector(".ant-modal")).toBeFalsy();
	});

	it("renders modal title with displayName", () => {
		render(<WorkflowYamlViewer nodeData={baseNode} onClose={vi.fn()} />);
		expect(screen.getByText("数据预处理 YAML")).toBeTruthy();
	});

	it("falls back to name in title", () => {
		const node = { id: "n1", name: "my-node", phase: "Running", type: "Pod" };
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		expect(screen.getByText("my-node YAML")).toBeTruthy();
	});

	it("falls back to 'Node' when no name", () => {
		const node = { id: "n1", name: "", phase: "Pending", type: "Pod" };
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		expect(screen.getByText(/Node YAML/)).toBeTruthy();
	});

	it("renders JSON key names", () => {
		const node = { id: "n1", name: "test", phase: "Succeeded", type: "Pod", extra: "value" };
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		expect(screen.getByText(/"extra"/)).toBeTruthy();
	});

	it("renders nested objects", () => {
		const node = {
			id: "n1", name: "test", phase: "Succeeded", type: "Pod",
			inputs: { lr: 0.001 },
		};
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		expect(screen.getByText(/"lr"/)).toBeTruthy();
	});

	it("renders arrays", () => {
		const node = {
			id: "n1", name: "test", phase: "Succeeded", type: "Pod",
			children: ["c1", "c2"],
		};
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		expect(screen.getByText(/"c1"/)).toBeTruthy();
	});

	it("renders empty arrays", () => {
		const node = { id: "n1", name: "test", phase: "Succeeded", type: "Pod", items: [] };
		render(<WorkflowYamlViewer nodeData={node} onClose={vi.fn()} />);
		const brackets = screen.getAllByText("[");
		expect(brackets.length).toBeGreaterThanOrEqual(1);
	});
});
