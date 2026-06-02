// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { WorkflowDetail, WorkflowNodeStatus } from "../../api/workflowApi";
import { WorkflowNodeDetailPanel } from "./WorkflowNodeDetailPanel";

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

vi.mock("dayjs", () => {
	const dayjs = (value?: string) => ({
		isValid: () => !!value,
		format: () => (value ? value.slice(0, 19).replace("T", " ") : ""),
		fromNow: () => "a few seconds ago",
	});
	dayjs.extend = () => dayjs;
	return { default: dayjs };
});

const baseNode: WorkflowNodeStatus = {
	id: "wf-node-1",
	name: "train-step",
	displayName: "训练步骤",
	phase: "Succeeded",
	type: "Pod",
	startedAt: "2026-01-15T10:00:00Z",
	finishedAt: "2026-01-15T10:05:00Z",
	hostNodeName: "gke-node-1234",
	progress: "2/2",
};

const baseWorkflow: WorkflowDetail = {
	name: "ml-training-pipeline",
	namespace: "default",
	phase: "Succeeded",
	startedAt: "2026-01-15T10:00:00Z",
	finishedAt: "2026-01-15T10:30:00Z",
	progress: "5/5",
};

describe("WorkflowNodeDetailPanel", () => {
	it("returns null when node is null", () => {
		const { container } = render(
			<WorkflowNodeDetailPanel
				node={null}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(container.innerHTML).toBe("");
	});

	it("returns null when workflow is null", () => {
		const { container } = render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={null}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(container.innerHTML).toBe("");
	});

	it("renders drawer with summary tab", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(screen.getByText("概览")).toBeTruthy();
		expect(screen.getByRole("tab", { name: /日志/ })).toBeTruthy();
		expect(screen.getByRole("tab", { name: /运行环境/ })).toBeTruthy();
		expect(screen.getByText("输入/输出")).toBeTruthy();
		expect(screen.getAllByRole("tab")).toHaveLength(4);
	});

	it("shows node phase tag", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(screen.getByText("Succeeded")).toBeTruthy();
	});

	it("shows resolved pod name when provided by the API", () => {
		render(
			<WorkflowNodeDetailPanel
				node={{
					...baseNode,
					podName: "wf-step-emit-123",
				}}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(screen.getByText("wf-step-emit-123")).toBeTruthy();
	});

	it("shows retry button when canRetryWorkflow", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
				onRetryWorkflow={vi.fn()}
				canRetryWorkflow
			/>,
		);
		expect(screen.getByText("重试工作流")).toBeTruthy();
	});

	it("renders containers section in runtime tab", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByRole("tab", { name: /运行环境/ }));
		expect(screen.getByText("暂无容器详情")).toBeTruthy();
	});

	it("renders containers table when runtime data present", () => {
		const node = {
			...baseNode,
			containers: [
				{
					name: "main",
					image: "python:3.11",
					command: ["python"],
					args: ["train.py"],
				},
			],
		};
		render(
			<WorkflowNodeDetailPanel
				node={node}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByRole("tab", { name: /运行环境/ }));
		expect(screen.getByText("main")).toBeTruthy();
		expect(screen.getByText("python:3.11")).toBeTruthy();
	});

	it("renders inputs in i/o tab", () => {
		const node: WorkflowNodeStatus = {
			...baseNode,
			inputs: {
				parameters: [{ name: "lr", value: "0.01" }],
				artifacts: [{ name: "ds", path: "gs://b/data" }],
			},
		};
		render(
			<WorkflowNodeDetailPanel
				node={node}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByText("输入/输出"));
		expect(screen.getByText("lr")).toBeTruthy();
	});

	it("renders string exit code in i/o tab", () => {
		const node: WorkflowNodeStatus = {
			...baseNode,
			outputs: {
				exitCode: "0",
			},
		};
		render(
			<WorkflowNodeDetailPanel
				node={node}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByText("输入/输出"));
		expect(screen.getByText("0")).toBeTruthy();
	});

	it("renders memoization info", () => {
		const node: WorkflowNodeStatus = {
			...baseNode,
			memoizationStatus: { hit: true, key: "abc", cacheName: "c1" },
		};
		render(
			<WorkflowNodeDetailPanel
				node={node}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		expect(screen.getByText(/命中=是/)).toBeTruthy();
	});

	it("renders runtime diagnostics placeholders", () => {
		render(
			<WorkflowNodeDetailPanel
				node={{
					...baseNode,
					cluster: "gke-dev",
					namespace: "cyber-databrew-dev",
					serviceAccountName: "argo-workflow",
					podIp: "10.1.2.3",
					podConditions: [{ type: "Ready", status: "True" }],
					podEvents: [
						{
							type: "Warning",
							reason: "BackOff",
							message: "Back-off restarting failed container",
							count: 2,
						},
					],
				}}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByRole("tab", { name: /运行环境/ }));
		expect(screen.getByText("gke-dev")).toBeTruthy();
		expect(screen.getByText("cyber-databrew-dev")).toBeTruthy();
		expect(screen.getByText("BackOff")).toBeTruthy();
	});

	it("renders monitoring and billing shells", () => {
		const node: WorkflowNodeStatus = {
			...baseNode,
			metrics: {
				cpuCores: 0.5,
				cpuLimitCores: 1,
				memoryBytes: 512 * 1024 * 1024,
				memoryLimitBytes: 1024 * 1024 * 1024,
			},
			cost: {
				totalCostUsd: 0.12,
				cpuCostUsd: 0.05,
				memoryCostUsd: 0.04,
				provider: "opencost",
				window: "1h",
			},
		};
		render(
			<WorkflowNodeDetailPanel
				node={node}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);

		fireEvent.click(screen.getByRole("tab", { name: /运行环境/ }));
		expect(screen.getByText("监控快照")).toBeTruthy();
		expect(screen.getAllByText("用量 50%").length).toBeGreaterThan(0);

		expect(screen.getByText("计费快照")).toBeTruthy();
		expect(screen.getByText("$0.1200")).toBeTruthy();
	});

	it("renders disabled debug terminal shell before backend exec is available", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
			/>,
		);
		fireEvent.click(screen.getByRole("tab", { name: /运行环境/ }));
		expect(screen.getByText("Exec 接口待接入")).toBeTruthy();
		expect(screen.getByText("等待后端 WebSocket exec 能力接入")).toBeTruthy();
		expect(screen.getByRole("button", { name: "pwd" })).toBeDisabled();
	});

	it("opens the requested compact tab", () => {
		render(
			<WorkflowNodeDetailPanel
				node={baseNode}
				workflow={baseWorkflow}
				open
				onClose={vi.fn()}
				onShowLogs={vi.fn()}
				activeTab="logs"
			/>,
		);
		expect(screen.getByText("查看该步骤日志")).toBeTruthy();
		expect(screen.getByRole("button", { name: "打开日志查看器" })).toBeTruthy();
	});
});
