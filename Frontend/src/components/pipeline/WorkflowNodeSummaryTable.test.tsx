// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { WorkflowNodeStatus } from "../../api/workflowApi";
import {
	buildWorkflowNodeSummaryRows,
	WorkflowNodeSummaryTable,
} from "./WorkflowNodeSummaryTable";

Object.defineProperty(window, "matchMedia", {
	writable: true,
	value: vi.fn().mockImplementation((query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: vi.fn(),
		removeListener: vi.fn(),
		addEventListener: vi.fn(),
		removeEventListener: vi.fn(),
		dispatchEvent: vi.fn(),
	})),
});

const nodes: WorkflowNodeStatus[] = [
	{
		id: "node-a",
		name: "workflow.step-a",
		displayName: "训练节点",
		templateName: "train",
		type: "Pod",
		phase: "Succeeded",
		podName: "workflow-step-a-123",
		startedAt: "2026-06-02T10:00:00Z",
		finishedAt: "2026-06-02T10:05:00Z",
		estimatedCostUsd: 1.25,
	},
	{
		id: "node-b",
		name: "workflow.step-b",
		displayName: "上传节点",
		templateName: "upload",
		type: "Pod",
		phase: "Failed",
		podName: "workflow-step-b-123",
		startedAt: "2026-06-02T10:05:00Z",
		finishedAt: "2026-06-02T10:05:30Z",
	},
];

describe("WorkflowNodeSummaryTable", () => {
	it("derives duration and cost summary rows", () => {
		const rows = buildWorkflowNodeSummaryRows(nodes);
		expect(rows).toHaveLength(2);
		expect(rows[0].durationSeconds).toBe(300);
		expect(rows[0].estimatedCostUsd).toBe(1.25);
		expect(rows[1].estimatedCostUsd).toBeNull();
	});

	it("does not keep increasing terminal node duration when finishedAt is missing", () => {
		const rows = buildWorkflowNodeSummaryRows([
			{
				id: "node-c",
				name: "workflow.step-c",
				displayName: "终态缺结束时间",
				type: "Pod",
				phase: "Failed",
				startedAt: "2026-06-02T10:00:00Z",
			},
			{
				id: "node-d",
				name: "workflow.step-d",
				displayName: "终态估算耗时",
				type: "Pod",
				phase: "Succeeded",
				startedAt: "2026-06-02T10:00:00Z",
				estimatedDuration: 42,
			},
		]);

		expect(rows[0].durationSeconds).toBeNull();
		expect(rows[1].durationSeconds).toBe(42);
	});

	it("renders node duration, cost, and pending cost state", () => {
		render(
			<WorkflowNodeSummaryTable
				nodes={nodes}
				onInspectNode={vi.fn()}
				onOpenLogs={vi.fn()}
				onOpenRuntime={vi.fn()}
			/>,
		);

		expect(screen.getByText("节点耗时 / 花费")).toBeTruthy();
		expect(screen.getByText("训练节点")).toBeTruthy();
		expect(screen.getByText("5m 0s")).toBeTruthy();
		expect(screen.getAllByText("$1.25").length).toBeGreaterThan(0);
		expect(screen.getAllByText("—").length).toBeGreaterThan(0);
	});

	it("opens existing node surfaces from table actions", () => {
		const onInspectNode = vi.fn();
		const onOpenLogs = vi.fn();
		const onOpenRuntime = vi.fn();
		render(
			<WorkflowNodeSummaryTable
				nodes={nodes}
				onInspectNode={onInspectNode}
				onOpenLogs={onOpenLogs}
				onOpenRuntime={onOpenRuntime}
			/>,
		);

		fireEvent.click(screen.getByRole("button", { name: "查看节点 训练节点" }));
		expect(onInspectNode).toHaveBeenCalledWith(nodes[0]);

		fireEvent.click(screen.getByRole("button", { name: "查看日志 训练节点" }));
		expect(onOpenLogs).toHaveBeenCalledWith(nodes[0]);

		fireEvent.click(
			screen.getByRole("button", { name: "查看运行资源 训练节点" }),
		);
		expect(onOpenRuntime).toHaveBeenCalledWith(nodes[0]);
	});
});
