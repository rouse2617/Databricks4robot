// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkflowDetailPage, {
	findPipelineNodeForWorkflowNode,
} from "./WorkflowDetailPage";

const mockUseWorkflowDetail = vi.fn();
const mockDeleteRun = vi.fn();
const mockRetryRun = vi.fn();
const mockResubmitRun = vi.fn();
const mockRerunRun = vi.fn();
const mockStopRun = vi.fn();
const mockSuspendRun = vi.fn();
const mockResumeRun = vi.fn();
const mockTerminateRun = vi.fn();
const mockWorkflowDagView = vi.fn(
	({
		nodes,
	}: {
		nodes: Array<{ displayName?: string; name?: string; phase: string }>;
	}) => (
		<div data-testid="mock-dag-view">
			{nodes
				.map((node) => `${node.displayName || node.name}:${node.phase}`)
				.join("|")}
		</div>
	),
);

vi.mock("./useWorkflowDetail", () => ({
	useWorkflowDetail: (...args: unknown[]) => mockUseWorkflowDetail(...args),
}));

vi.mock("../api/runApi", () => ({
	deleteRun: (...args: unknown[]) => mockDeleteRun(...args),
	retryRun: (...args: unknown[]) => mockRetryRun(...args),
	resubmitRun: (...args: unknown[]) => mockResubmitRun(...args),
	rerunRun: (...args: unknown[]) => mockRerunRun(...args),
	stopRun: (...args: unknown[]) => mockStopRun(...args),
	suspendRun: (...args: unknown[]) => mockSuspendRun(...args),
	resumeRun: (...args: unknown[]) => mockResumeRun(...args),
	terminateRun: (...args: unknown[]) => mockTerminateRun(...args),
}));

vi.mock("./WorkflowDagView", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		WorkflowDagView: (props: Parameters<typeof mockWorkflowDagView>[0]) =>
			mockWorkflowDagView(props),
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
				"asset-ids": "aa111111,bb222222",
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
		runMetadataState: {
			inputs: null,
			outputs: null,
			runtime: null,
			loading: false,
			error: null,
		},
		setLogSearch: vi.fn(),
		startFollowLogs: vi.fn(),
		stopFollowLogs: vi.fn(),
		userStopFollowLogs: vi.fn(),
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
		mockDeleteRun.mockResolvedValue(undefined);
		mockRetryRun.mockResolvedValue({ message: "retry submitted" });
		mockResubmitRun.mockResolvedValue({ message: "resubmit submitted" });
		mockRerunRun.mockResolvedValue({ message: "rerun submitted" });
		mockStopRun.mockResolvedValue({ message: "stop submitted" });
		mockSuspendRun.mockResolvedValue({ message: "suspend submitted" });
		mockResumeRun.mockResolvedValue({ message: "resume submitted" });
		mockTerminateRun.mockResolvedValue({ message: "terminate submitted" });
		mockWorkflowDetailState();
	});

	afterEach(cleanup);

	it("uses runId route params for run-centric detail URLs", () => {
		render(
			<MemoryRouter initialEntries={["/runs/run-123"]}>
				<Routes>
					<Route path="/runs/:runId" element={<WorkflowDetailPage />} />
				</Routes>
			</MemoryRouter>,
		);

		expect(mockUseWorkflowDetail).toHaveBeenCalledWith("run-123", {
			lookupMode: "runId",
		});
	});

	it("maps workflow nodes back to pipeline node snapshots by run node id", () => {
		const pipelineNode = findPipelineNodeForWorkflowNode(
			{
				name: "runtime-pipeline",
				nodes: [
					{
						id: "node-configured",
						component: { name: "configured probe", image: "busybox" },
						storageMounts: [
							{
								resourceId: "scratch-emptydir",
								mountPath: "/workspace/scratch",
							},
						],
					},
				],
				edges: [],
			},
			{
				id: "run-1",
				workflowName: "wf-asset",
				pipelineName: "runtime-pipeline",
				status: "Running",
				nodeCount: 1,
				createdAt: "2026-06-03T00:00:00Z",
				nodes: [
					{
						id: "run-node-1",
						runId: "run-1",
						pipelineNodeId: "node-configured",
						argoNodeId: "argo-node-1",
					},
				],
			},
			{
				id: "argo-node-1",
				name: "wf-asset.step-node-configured",
				displayName: "configured probe",
				type: "Pod",
				phase: "Running",
			},
		);

		expect(pipelineNode?.id).toBe("node-configured");
		expect(pipelineNode?.storageMounts?.[0]?.resourceId).toBe(
			"scratch-emptydir",
		);
	});

	it("renders linked input asset chips from workflow labels", () => {
		renderWorkflowDetail();

		expect(screen.getByRole("link", { name: "aa111111" })).toHaveAttribute(
			"href",
			"/assets/aa111111",
		);
		expect(screen.getByText("aa111111")).toBeInTheDocument();
		expect(screen.getByText("bb222222")).toBeInTheDocument();
		expect(screen.getByRole("link", { name: "bb222222" })).toHaveAttribute(
			"href",
			"/assets/bb222222",
		);
	});

	it("renders UUID asset ids as asset detail links", () => {
		// Grace videos are now first-class DataBrew assets addressed by UUID
		// asset_id (CYB-1215 / grace_video), so a UUID asset id is canonical and
		// links to its asset detail page. (Previously these were treated as
		// external ids with no link.)
		const videoID = "019dabf3-5685-769f-8ec3-3992767ebe65";
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Succeeded",
				nodes: [],
				createdAt: "2026-06-03T00:00:00Z",
				labels: {
					"asset-ids": videoID,
					"template-name": "asset-pipeline",
					"template-version": "3",
				},
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText(videoID)).toBeInTheDocument();
		const link = screen.getByRole("link", { name: videoID });
		expect(link).toHaveAttribute("href", `/assets/${videoID}`);
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

		// The unavailable-events copy lives under the "事件时间线" tab, whose content
		// only mounts once the tab is activated (CYB-1565 tabbed layout).
		fireEvent.click(screen.getByRole("tab", { name: /事件时间线/ }));
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

	it("renders Run inputs, outputs, and runtime metadata", () => {
		mockWorkflowDetailState({
			runEventState: {
				run: {
					id: "run-1",
					workflowName: "wf-asset",
					pipelineName: "asset-pipeline",
					status: "Succeeded",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			runMetadataState: {
				inputs: {
					runId: "run-1",
					total: 1,
					items: [
						{
							id: "input-config",
							runId: "run-1",
							nodeId: "node-1",
							type: "config",
							refId: "cfg-1",
							refVersion: "2",
							mountPath: "/mnt/parameters",
							targetFilename: "params.yaml",
							source: "runtime_config",
						},
					],
				},
				outputs: {
					runId: "run-1",
					total: 1,
					items: [
						{
							id: "output-logs",
							runId: "run-1",
							nodeId: "node-1",
							type: "logs",
							refId: "wf-asset/node-1",
							uri: "argo://wf-asset/node-1",
						},
					],
				},
				runtime: {
					runId: "run-1",
					runtime: {
						runtimeType: "argo",
						workflowName: "wf-asset",
						namespace: "video-proc-dev",
						uid: "uid-1",
						status: "Succeeded",
						executionTargetId: "target-gpu",
					},
				},
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		// Inputs/outputs metadata now live under the "运行上下文" tab; its content
		// only mounts once the tab is activated (CYB-1565 tabbed layout). The
		// panel shows the config input (refId@version) and the outputs table.
		fireEvent.click(screen.getByRole("tab", { name: /运行上下文/ }));
		expect(screen.getByText("cfg-1@2")).toBeInTheDocument();
		expect(screen.getByText("argo://wf-asset/node-1")).toBeInTheDocument();
		expect(screen.getByText("argo://wf-asset/node-1")).toBeInTheDocument();
		expect(screen.getByText("video-proc-dev")).toBeInTheDocument();
		expect(screen.getByText("target-gpu")).toBeInTheDocument();
	});

	it("overlays DAG node status from the DataBrew run node snapshot", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Running",
				message: "",
				nodes: [
					{
						id: "argo-node-1",
						name: "wf-asset.node-1",
						displayName: "node-1",
						type: "Pod",
						phase: "Pending",
						startedAt: "2026-06-03T00:09:00Z",
					},
				],
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
					message: "main: Error (exit code 1)",
					nodes: [
						{
							id: "run-node-1",
							runId: "run-1",
							pipelineNodeId: "node-1",
							argoNodeId: "argo-node-1",
							displayName: "node-1",
							phase: "Failed",
							message: "main: Error (exit code 1)",
							finishedAt: "2026-06-03T00:10:00Z",
							updatedAt: "2026-06-03T00:10:00Z",
						},
					],
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByTestId("mock-dag-view")).toHaveTextContent(
			"node-1:Failed",
		);
		expect(screen.getByText("main: Error (exit code 1)")).toBeInTheDocument();
	});

	it("overlays terminal run status on active unmapped DAG nodes", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Running",
				message: "",
				nodes: [
					{
						id: "argo-node-unknown",
						name: "wf-asset.unknown",
						displayName: "unknown",
						type: "Pod",
						phase: "Pending",
						createdAt: "2026-06-03T00:00:00Z",
					},
				],
				createdAt: "2026-06-03T00:09:00Z",
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
					createdAt: "2026-06-03T00:09:00Z",
					finishedAt: "2026-06-03T00:10:00Z",
					message: "workflow shutdown with strategy: Failed",
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(screen.getByTestId("mock-dag-view")).toHaveTextContent(
			"unknown:Failed",
		);
		expect(
			screen.getByText("workflow shutdown with strategy: Failed"),
		).toBeInTheDocument();
	});

	it("deletes the DataBrew execution record when a run id is available", async () => {
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
				run: {
					id: "run-1",
					workflowName: "wf-asset",
					pipelineName: "asset-pipeline",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		fireEvent.click(screen.getByRole("button", { name: /删除/ }));
		expect(await screen.findByText(/将从执行记录列表删除/)).toBeInTheDocument();
		const dialog = await screen.findByRole("dialog");
		fireEvent.click(within(dialog).getByRole("button", { name: /删\s*除/ }));

		await waitFor(() => expect(mockDeleteRun).toHaveBeenCalledWith("run-1"));
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

		expect(screen.getByText("未找到该工作流")).toBeInTheDocument();
		expect(screen.getByText("workflow not found")).toBeInTheDocument();
		expect(screen.queryByText("正在加载执行记录…")).not.toBeInTheDocument();
	});

	it("keeps Run ledger visible when the runtime workflow is missing", () => {
		mockWorkflowDetailState({
			workflow: null,
			loading: false,
			loadError: {
				kind: "not_found",
				message: "workflow not found",
			},
			runEventState: {
				run: {
					id: "run-ttl",
					workflowName: "wf-expired",
					pipelineName: "asset-pipeline",
					status: "Error",
					// A genuinely TTL-cleaned workflow existed in Argo and was
					// assigned a UID (persisted by DataBrew), distinguishing it from
					// a run that failed before any workflow was created (CYB-3082).
					argoWorkflowUid: "wf-expired-uid",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
					finishedAt: "2026-06-03T00:10:00Z",
					assetIds: ["aa111111"],
					message: "workflow not found",
				},
				items: [
					{
						id: "event-1",
						runId: "run-ttl",
						workflowName: "wf-expired",
						eventType: "run_failed",
						subjectType: "run",
						subjectId: "run-ttl",
						status: "Error",
						message: "workflow not found",
						sequence: 1,
						occurredAt: "2026-06-03T00:10:00Z",
						observedAt: "2026-06-03T00:10:00Z",
						createdAt: "2026-06-03T00:10:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
			},
			runMetadataState: {
				inputs: {
					runId: "run-ttl",
					total: 1,
					items: [
						{
							id: "input-asset",
							runId: "run-ttl",
							type: "asset",
							refId: "aa111111",
							source: "run",
						},
					],
				},
				outputs: { runId: "run-ttl", total: 0, items: [] },
				runtime: {
					runId: "run-ttl",
					runtime: {
						runtimeType: "argo",
						workflowName: "wf-expired",
						namespace: "video-proc-dev",
						status: "missing",
					},
				},
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(
			screen.getByText("底层 Runtime 已不可用，正在展示 DataBrew 历史账本"),
		).toBeInTheDocument();
		expect(screen.getByText("run-ttl")).toBeInTheDocument();
		expect(screen.getByText("Runtime 不可用")).toBeInTheDocument();
		expect(screen.getByText("运行上下文")).toBeInTheDocument();
		expect(screen.getAllByText("wf-expired").length).toBeGreaterThan(0);
		expect(screen.getByText("video-proc-dev")).toBeInTheDocument();
		expect(screen.queryByText("未找到工作流")).not.toBeInTheDocument();
	});

	it("explains runs that failed before any workflow was created", () => {
		// A run rejected before submission (e.g. by the resource guard) is
		// terminally failed but never obtained an Argo workflow UID. The banner
		// must not imply TTL cleanup — the workflow never existed (CYB-3082).
		mockWorkflowDetailState({
			workflow: null,
			loading: false,
			loadError: {
				kind: "not_found",
				message: "workflow not found",
			},
			runEventState: {
				run: {
					id: "run-rejected",
					workflowName: "youxin-all-batch-abc123",
					pipelineName: "youxin-all",
					status: "Failed",
					nodeCount: 0,
					createdAt: "2026-06-03T00:00:00Z",
					finishedAt: "2026-06-03T00:00:05Z",
					blockingReason: "resource_incompatible",
					blockingMessage:
						'执行目标 "Default Argo target"不支持该资源规格：节点 "head-track" 请求 cpu=14000m',
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(
			screen.getByText("该运行在提交到 Runtime 前失败，未创建底层 workflow"),
		).toBeInTheDocument();
		expect(screen.getByText(/该运行未生成底层 workflow/)).toBeInTheDocument();
		// Must NOT show the TTL-cleanup wording for a never-created workflow.
		expect(
			screen.queryByText("底层 Runtime 已不可用，正在展示 DataBrew 历史账本"),
		).not.toBeInTheDocument();
	});

	it("explains pending Runs that have not been submitted to runtime", () => {
		mockWorkflowDetailState({
			workflow: null,
			loading: false,
			loadError: {
				kind: "not_found",
				message: "Run has no runtime workflow reference",
			},
			runEventState: {
				run: {
					id: "run-pending",
					pipelineName: "asset-pipeline",
					status: "Pending",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
					noAssetRun: true,
					blockingReason: "runtime_not_submitted",
					blockingMessage: "等待提交到运行时",
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
		});

		renderWorkflowDetail();

		expect(
			screen.getByText("Run 已创建，等待提交到 Runtime"),
		).toBeInTheDocument();
		expect(screen.getByText(/DAG、Pod、实时日志暂不可用/)).toBeInTheDocument();
		expect(screen.getByText("run-pending")).toBeInTheDocument();
		expect(screen.getByText("无资产运行")).toBeInTheDocument();
		expect(screen.getByText("等待提交")).toBeInTheDocument();
		expect(screen.getByText("等待提交到运行时")).toBeInTheDocument();
		expect(screen.queryByText("未找到工作流")).not.toBeInTheDocument();
	});

	it("shows workflow node count without unavailable cost noise", () => {
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
		expect(screen.queryByText("暂无估算成本")).not.toBeInTheDocument();
		expect(screen.queryByText("暂无计费配置")).not.toBeInTheDocument();
	});

	it("uses pipeline component names for asset node rows", () => {
		mockWorkflowDetailState({
			workflow: {
				name: "wf-asset",
				status: "Running",
				nodes: [
					{
						id: "argo-node-detector",
						name: "wf-asset.step-node-detector",
						displayName: "step-node-detector",
						type: "Pod",
						phase: "Pending",
					},
				],
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
					status: "Running",
					nodeCount: 1,
					createdAt: "2026-06-03T00:00:00Z",
					pipelineJSON: {
						name: "asset-pipeline",
						nodes: [
							{
								id: "node-detector",
								component: {
									name: "hand-detect-yolov26m",
									image: "example.com/hand-detect:yolo",
								},
							},
						],
						edges: [],
					},
				},
				items: [],
				total: 0,
				loading: false,
				error: null,
			},
			assetNodeState: {
				items: [
					{
						id: "asset-node-1",
						runId: "run-1",
						assetId: "no-asset",
						pipelineNodeId: "step-node-detector",
						argoNodeId: "wf-asset.step-node-detector",
						displayName: "step-node-detector",
						status: "Pending",
						message:
							"InvalidImageName: failed to apply default image tag for dummy digest",
						costSource: "not_available",
						updatedAt: "2026-06-03T00:00:00Z",
					},
				],
				total: 1,
				loading: false,
				error: null,
				summary: {
					assetCount: 0,
					nodeCount: 1,
					statuses: { Pending: 1 },
					costSource: "not_available",
				},
			},
		});

		renderWorkflowDetail();

		expect(screen.getByText("hand-detect-yolov26m")).toBeInTheDocument();
		expect(screen.getByText("step-node-detector")).toBeInTheDocument();
		expect(screen.getByText(/InvalidImageName/)).toBeInTheDocument();
	});
});
