// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkflowDetailPage, {
	getWorkflowLogEmptyState,
} from "./WorkflowDetailPage";

const mockUseWorkflowDetail = vi.fn();

vi.mock("./useWorkflowDetail", () => ({
	useWorkflowDetail: (...args: unknown[]) => mockUseWorkflowDetail(...args),
}));

vi.mock("./WorkflowDagView", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		WorkflowDagView: () => <div data-testid="mock-dag-view" />,
	};
});

function renderWorkflowDetail() {
	return render(
		<MemoryRouter initialEntries={["/pipeline/executions/wf-asset"]}>
			<Routes>
				<Route
					path="/pipeline/executions/:name"
					element={<WorkflowDetailPage />}
				/>
			</Routes>
		</MemoryRouter>,
	);
}

function mockWorkflowDetailState(
	overrides: Partial<ReturnType<typeof mockUseWorkflowDetail>> = {},
) {
	mockUseWorkflowDetail.mockReturnValue({
		workflow: {
			name: "wf-asset",
			status: "Succeeded",
			nodes: [],
			createdAt: "2026-06-03T00:00:00Z",
			labels: {
				"asset-ids": "asset-a,asset-b",
				"template-name": "asset-pipeline",
				"template-version": "3",
			},
		},
		loading: false,
		loadError: null,
		selectedNode: null,
		selectNode: vi.fn(),
		loadWorkflow: vi.fn(),
		logState: {
			logs: "",
			loading: false,
			error: null,
			search: "",
			following: false,
			followError: null,
		},
		runEventState: {
			run: {
				id: "run-asset-1",
				templateId: "tpl-v3-id",
				templateName: "asset-pipeline",
				templateVersion: 3,
				triggerSource: "asset_run",
				assetIds: ["asset-a", "asset-b"],
			},
			items: [],
			total: 0,
			loading: false,
			error: null,
		},
		runEventFilters: {},
		setRunEventFilters: vi.fn(),
		loadRunEvents: vi.fn(),
		assetNodeState: {
			items: [],
			total: 0,
			loading: false,
			error: null,
			summary: {
				assetCount: 2,
				nodeCount: 0,
				statuses: {},
				costSource: "unavailable",
			},
		},
		costSummaryState: {
			item: null,
			loading: false,
			error: null,
		},
		setLogSearch: vi.fn(),
		startFollowLogs: vi.fn(),
		stopFollowLogs: vi.fn(),
		downloadLogs: vi.fn(),
		...overrides,
	});
}

describe("WorkflowDetailPage", () => {
	beforeEach(() => {
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
		vi.clearAllMocks();
		mockWorkflowDetailState();
	});

	afterEach(cleanup);

	it("renders linked input asset chips from workflow labels", () => {
		renderWorkflowDetail();

		expect(screen.getByText("asset-pipeline")).toBeInTheDocument();
		expect(screen.getByText(/快照 tplv3id · 资产运行/)).toBeInTheDocument();
		expect(screen.getByText("asset-a")).toBeInTheDocument();
		expect(screen.getByText("asset-b")).toBeInTheDocument();
		expect(screen.getByRole("link", { name: "asset-a" })).toHaveAttribute(
			"href",
			"/assets/asset-a",
		);
		expect(screen.getByRole("link", { name: "asset-b" })).toHaveAttribute(
			"href",
			"/assets/asset-b",
		);
	});

	it("uses product-facing copy when run events are unavailable", () => {
		mockWorkflowDetailState({
			runEventState: {
				items: [],
				total: 0,
				loading: false,
				error: "pipeline run not found",
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("事件暂不可用")).toBeInTheDocument();
		expect(
			screen.getByText(/这是历史工作流或外部提交的工作流/),
		).toBeInTheDocument();
	});

	it("shows waiting resource snapshot copy for pending cost rows", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Running",
				nodes: [
					{
						id: "step-1",
						name: "wf-asset.step-1",
						displayName: "step-1",
						type: "Pod",
						phase: "Pending",
					},
				],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"asset-ids": "asset-a",
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
			runEventState: {
				run: { id: "run-1" },
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			assetNodeState: {
				items: [
					{
						id: "row-1",
						runId: "run-1",
						assetId: "asset-a",
						pipelineNodeId: "step-1",
						displayName: "step-1",
						status: "Pending",
						costSource: "not_available",
						updatedAt: "2026-06-03T00:00:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
				summary: {
					assetCount: 1,
					nodeCount: 1,
					statuses: { Pending: 1 },
					costSource: "not_available",
				},
			},
			costSummaryState: {
				item: {
					runId: "run-1",
					costSource: "not_available",
					nodeSummaries: [
						{
							nodeId: "step-1",
							displayName: "step-1",
							status: "Pending",
							podCount: 0,
							costSource: "not_available",
						},
					],
					assetNodeSummaries: [],
					generatedAt: "2026-06-03T00:00:00Z",
				},
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getAllByText("等待资源快照").length).toBeGreaterThan(0);
		expect(
			screen.getByText(
				/节点还在排队或运行中，资源耗时生成后会自动补齐估算成本/,
			),
		).toBeInTheDocument();
		expect(screen.queryByText("暂无计费配置")).not.toBeInTheDocument();
	});

	it("uses live dag status when asset-node ledger rows lag behind", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Succeeded",
				nodes: [
					{
						id: "step-1",
						name: "wf-asset.step-1",
						displayName: "step-1",
						type: "Pod",
						phase: "Succeeded",
						startedAt: "2026-06-03T00:00:00Z",
						finishedAt: "2026-06-03T00:00:10Z",
					},
				],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"asset-ids": "asset-a",
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
			runEventState: {
				run: { id: "run-1" },
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			assetNodeState: {
				items: [
					{
						id: "row-1",
						runId: "run-1",
						assetId: "asset-a",
						pipelineNodeId: "step-1",
						displayName: "step-1",
						status: "Pending",
						costSource: "not_available",
						updatedAt: "2026-06-03T00:00:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
				summary: {
					assetCount: 1,
					nodeCount: 1,
					statuses: { Pending: 1 },
					costSource: "not_available",
				},
			},
			costSummaryState: {
				item: {
					runId: "run-1",
					costSource: "not_available",
					nodeSummaries: [
						{
							nodeId: "step-1",
							displayName: "step-1",
							status: "Pending",
							podCount: 0,
							costSource: "not_available",
						},
					],
					assetNodeSummaries: [],
					generatedAt: "2026-06-03T00:00:00Z",
				},
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getAllByText("Succeeded").length).toBeGreaterThan(0);
		expect(screen.getAllByText("暂无成本数据").length).toBeGreaterThan(0);
		expect(screen.queryByText("等待资源快照")).not.toBeInTheDocument();
		expect(
			screen.queryByText(
				/节点还在排队或运行中，资源耗时生成后会自动补齐估算成本/,
			),
		).not.toBeInTheDocument();
	});

	it("renders explicit failure summary for failed nodes", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-fail",
				status: "Failed",
				nodes: [
					{
						id: "step-1",
						name: "wf-fail.step-1",
						displayName: "step-1",
						type: "Pod",
						phase: "Failed",
						message: "ImagePullBackOff",
					},
				],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"template-name": "asset-pipeline",
				},
			},
			runEventState: {
				run: { id: "run-fail" },
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("检测到失败节点")).toBeInTheDocument();
		expect(screen.getByText(/wf-fail\.step-1/)).toBeInTheDocument();
	});

	it("shows logs action for failed asset nodes even when logRef is missing", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Failed",
				nodes: [],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
			runEventState: {
				run: { id: "run-1" },
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			assetNodeState: {
				items: [
					{
						id: "row-1",
						runId: "run-1",
						assetId: "asset-a",
						pipelineNodeId: "step-1",
						displayName: "step-1",
						status: "Failed",
						costSource: "not_available",
						updatedAt: "2026-06-03T00:00:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
				summary: {
					assetCount: 1,
					nodeCount: 1,
					statuses: { Failed: 1 },
					costSource: "not_available",
				},
			},
		});

		renderWorkflowDetail();

		expect(screen.getByRole("button", { name: "日志" })).toBeInTheDocument();
	});

	it("shows workflow node count and quiet missing cost copy", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Succeeded",
				nodes: [
					{
						id: "step-1",
						name: "wf-asset.step-1",
						displayName: "step-1",
						type: "Pod",
						phase: "Succeeded",
					},
					{
						id: "step-2",
						name: "wf-asset.step-2",
						displayName: "step-2",
						type: "Pod",
						phase: "Succeeded",
					},
				],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"asset-ids": "asset-a",
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
			runEventState: {
				run: { id: "run-1" },
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			assetNodeState: {
				items: [
					{
						id: "row-1",
						runId: "run-1",
						assetId: "asset-a",
						pipelineNodeId: "step-1",
						displayName: "step-1",
						status: "Succeeded",
						costSource: "not_available",
						updatedAt: "2026-06-03T00:00:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
				summary: {
					assetCount: 1,
					nodeCount: 1,
					statuses: { Succeeded: 1 },
					costSource: "not_available",
				},
			},
			costSummaryState: {
				item: {
					runId: "run-1",
					costSource: "not_available",
					nodeSummaries: [
						{
							nodeId: "step-1",
							displayName: "step-1",
							status: "Succeeded",
							podCount: 1,
							costSource: "not_available",
						},
					],
					assetNodeSummaries: [],
					generatedAt: "2026-06-03T00:00:00Z",
				},
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("节点 2")).toBeInTheDocument();
		expect(screen.getAllByText("暂无成本数据").length).toBeGreaterThan(0);
		expect(
			screen.getByText(/本次运行已有节点结果，但没有生成成本快照/),
		).toBeInTheDocument();
		expect(screen.queryByText("暂无计费配置")).not.toBeInTheDocument();
	});

	it("returns waiting log copy for pending nodes", () => {
		expect(
			getWorkflowLogEmptyState({
				phase: "Pending",
				message: "0/3 nodes are available: Insufficient cpu.",
			}),
		).toEqual({
			message: "等待日志输出",
			description:
				"节点还在调度、排队或启动中；开始运行后会逐步输出日志。当前状态：0/3 nodes are available: Insufficient cpu.",
		});
	});

	it("returns quiet log copy for completed nodes", () => {
		expect(
			getWorkflowLogEmptyState({
				phase: "Succeeded",
			}),
		).toEqual({
			message: "暂无日志输出",
			description:
				"这个步骤没有返回可展示的日志内容；可继续查看 Pod 事件、节点消息和运行环境。",
		});
	});
});
