// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkflowDetailPage from "./WorkflowDetailPage";

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

		expect(screen.getByRole("link", { name: "asset-a" })).toHaveAttribute(
			"href",
			"/assets/asset-a",
		);
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

	it("prefers DataBrew run status over active Argo workflow status", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Running",
				message: "",
				nodes: [],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
			runEventState: {
				run: {
					id: "run-1",
					workflowName: "wf-asset",
					pipelineName: "asset-pipeline",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
					finishedAt: "2026-06-03T00:10:00Z",
					message: "workflow shutdown with strategy: Stop",
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("失败")).toBeInTheDocument();
		expect(
			screen.getByText("workflow shutdown with strategy: Stop"),
		).toBeInTheDocument();
		expect(
			screen.queryByRole("button", { name: /停止/ }),
		).not.toBeInTheDocument();
		expect(
			screen.queryByRole("button", { name: /暂停/ }),
		).not.toBeInTheDocument();
		expect(
			screen.queryByRole("button", { name: /终止/ }),
		).not.toBeInTheDocument();
		expect(screen.getByRole("button", { name: /重提交/ })).toBeInTheDocument();
	});

	it("shows a not-found alert instead of an indefinite spinner", () => {
		mockWorkflowDetailState({
			workflow: null,
			loading: false,
			loadError: {
				kind: "not_found",
				message: "workflow not found",
			},
			runEventState: {
				items: [],
				total: 0,
				loading: true,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("未找到工作流")).toBeInTheDocument();
		expect(screen.getByText("workflow not found")).toBeInTheDocument();
		expect(screen.queryByText("正在加载执行记录…")).not.toBeInTheDocument();
	});

	it("shows workflow node count and explicit unavailable cost copy", () => {
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
		expect(screen.getByText("暂无估算成本")).toBeInTheDocument();
		expect(screen.getByText("暂无计费配置")).toBeInTheDocument();
	});
});
