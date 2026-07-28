// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import { WorkflowExecutionList } from "./WorkflowExecutionList";

const mockListWorkflows = vi.fn();
const mockDeleteWorkflow = vi.fn();
const mockListRuns = vi.fn();
const mockDeleteRun = vi.fn();
const mockRetryRun = vi.fn();
const mockResubmitRun = vi.fn();
const mockStopRun = vi.fn();
const mockSuspendRun = vi.fn();
const mockResumeRun = vi.fn();
const mockTerminateRun = vi.fn();
const mockListExecutionTargets = vi.fn();

vi.mock("../api/workflowApi", () => ({
	listWorkflows: (...args: unknown[]) => mockListWorkflows(...args),
	deleteWorkflow: (...args: unknown[]) => mockDeleteWorkflow(...args),
	resubmitWorkflow: vi.fn(),
	retryWorkflow: vi.fn(),
	stopWorkflow: vi.fn(),
	suspendWorkflow: vi.fn(),
	resumeWorkflow: vi.fn(),
	terminateWorkflow: vi.fn(),
}));

vi.mock("../api/runApi", () => ({
	listRuns: (...args: unknown[]) => mockListRuns(...args),
	deleteRun: (...args: unknown[]) => mockDeleteRun(...args),
	retryRun: (...args: unknown[]) => mockRetryRun(...args),
	resubmitRun: (...args: unknown[]) => mockResubmitRun(...args),
	stopRun: (...args: unknown[]) => mockStopRun(...args),
	suspendRun: (...args: unknown[]) => mockSuspendRun(...args),
	resumeRun: (...args: unknown[]) => mockResumeRun(...args),
	terminateRun: (...args: unknown[]) => mockTerminateRun(...args),
}));

vi.mock("../api/pipelineApi", async (importOriginal) => {
	const actual = await importOriginal<typeof import("../api/pipelineApi")>();
	return {
		...actual,
		// CYB-3486: 列表挂载时拉资源池做 executionTargetId→池名映射。
		listExecutionTargets: (...args: unknown[]) =>
			mockListExecutionTargets(...args),
	};
});

vi.mock("antd", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		message: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
	};
});

// Ledger runs are the source of truth for the executions list. Runtime Argo
// workflow listing is not part of the product list data path.
const ledgerRuns = [
	{
		id: "run-success",
		pipelineName: "successful-run",
		workflowName: "successful-run",
		templateName: "daily-ingest-template",
		status: "Succeeded",
		nodeCount: 2,
		totalEstimatedCost: 1.25,
		createdAt: "2026-06-02T01:00:00Z",
		finishedAt: "2026-06-02T01:00:20Z",
	},
	{
		id: "run-failed",
		pipelineName: "failed-run",
		workflowName: "failed-run",
		status: "Failed",
		nodeCount: 1,
		totalEstimatedCost: 0.5,
		createdAt: "2026-06-02T02:00:00Z",
		finishedAt: "2026-06-02T02:00:10Z",
	},
];

function renderList(initialEntry = "/pipeline?tab=executions") {
	return render(
		<MemoryRouter initialEntries={[initialEntry]}>
			<WorkflowExecutionList />
		</MemoryRouter>,
	);
}

function renderBatchList() {
	return render(
		<MemoryRouter initialEntries={["/pipeline/batch/batch-1"]}>
			<WorkflowExecutionList batchJobId="batch-1" embedded />
		</MemoryRouter>,
	);
}

function getColumnHeader(label: string): HTMLElement {
	const header = screen
		.getAllByText(label)
		.map((node) => node.closest("th"))
		.find((node): node is HTMLElement => Boolean(node));
	expect(header).toBeTruthy();
	return header;
}

function expectTableOrder(names: string[]) {
	const body = document.querySelector(
		".pipeline-execution-table .ant-table-tbody",
	);
	expect(body).toBeTruthy();
	const text = body?.textContent ?? "";
	let previousIndex = -1;
	for (const name of names) {
		const index = text.indexOf(name);
		expect(index).toBeGreaterThan(previousIndex);
		previousIndex = index;
	}
}

beforeAll(() => {
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
});

describe("WorkflowExecutionList", () => {
	beforeEach(() => {
		mockDeleteWorkflow.mockResolvedValue({ message: "deleted" });
		mockDeleteRun.mockResolvedValue(undefined);
		mockRetryRun.mockResolvedValue({ message: "retry submitted" });
		mockResubmitRun.mockResolvedValue({ message: "resubmit submitted" });
		mockStopRun.mockResolvedValue({ message: "stop submitted" });
		mockSuspendRun.mockResolvedValue({ message: "suspend submitted" });
		mockResumeRun.mockResolvedValue({ message: "resume submitted" });
		mockTerminateRun.mockResolvedValue({ message: "terminate submitted" });
		mockListExecutionTargets.mockResolvedValue([]);
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-1",
					pipelineName: "successful-run",
					workflowName: "successful-run",
					templateName: "daily-ingest-template",
					status: "Succeeded",
					nodeCount: 2,
					totalEstimatedCost: 1.25,
					createdAt: "2026-06-02T01:00:00Z",
				},
			],
			total: 1,
		});
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it("applies status filters from the URL", async () => {
		mockListRuns.mockResolvedValue({ items: ledgerRuns, total: 2 });
		renderList("/pipeline?tab=executions&status=Failed");

		await waitFor(() => {
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
		expect(screen.queryByText("successful-run")).not.toBeInTheDocument();
	});

	it("clears status filters when reset is clicked", async () => {
		mockListRuns.mockResolvedValue({ items: ledgerRuns, total: 2 });
		renderList("/pipeline?tab=executions&status=Failed");

		await waitFor(() => {
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
		expect(screen.queryByText("successful-run")).not.toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "重 置" }));

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
	});

	it("merges run-level estimated cost from Runs and shows a quiet empty state otherwise", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		expect(mockListRuns).toHaveBeenCalledWith(
			expect.objectContaining({
				view: "summary",
				excludeBatch: true,
				page: 1,
				pageSize: 20,
			}),
		);
		expect(screen.getAllByText("总成本").length).toBeGreaterThan(0);
		expect(screen.getByText("$1.25")).toBeInTheDocument();
		expect(screen.getByText(/daily-ingest-template/)).toBeInTheDocument();
		expect(screen.getAllByText("—").length).toBeGreaterThan(0);
	});

	it("passes name searches to the Run list API for cross-page template lookup", async () => {
		mockListRuns.mockResolvedValue({ items: ledgerRuns, total: 2 });
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		fireEvent.change(screen.getByPlaceholderText(/搜索名称/), {
			target: { value: "daily-ingest-template" },
		});
		fireEvent.click(screen.getByRole("button", { name: "search" }));

		await waitFor(() => {
			expect(mockListRuns).toHaveBeenLastCalledWith(
				expect.objectContaining({
					view: "summary",
					excludeBatch: true,
					q: "daily-ingest-template",
				}),
			);
		});
	});

	it("shows ledger-only runs with estimated cost after live Argo workflow TTL cleanup", async () => {
		mockListWorkflows.mockResolvedValue({ items: [] });
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-ledger-1",
					pipelineName: "ttl-cleaned-pipeline",
					workflowName: "ttl-cleaned-workflow",
					status: "Succeeded",
					nodeCount: 5,
					totalEstimatedCost: 0.009,
					createdAt: "2026-06-03T19:16:51Z",
					finishedAt: "2026-06-03T19:18:09Z",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("ttl-cleaned-workflow")).toBeInTheDocument();
		});
		expect(screen.getByText("$0.0090")).toBeInTheDocument();
		expect(screen.getByText("ID: runledge")).toBeInTheDocument();
	});

	it("loads ledger records without querying runtime workflow list", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-ledger-2",
					pipelineName: "ledger-pipeline",
					workflowName: "ledger-workflow",
					status: "Succeeded",
					nodeCount: 2,
					totalEstimatedCost: 1.5,
					createdAt: "2026-06-03T10:00:00Z",
					finishedAt: "2026-06-03T10:01:00Z",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("ledger-workflow")).toBeInTheDocument();
		});
		expect(screen.queryByText("服务不可用")).not.toBeInTheDocument();
		expect(screen.getByText("$1.50")).toBeInTheDocument();
		expect(mockListWorkflows).not.toHaveBeenCalled();
	});

	// CYB-4334: a run with neither workflowName nor pipelineName must not show
	// the raw run UUID as its primary label (that both is unreadable and
	// duplicates the ID line). It should fall back to the template name.
	it("shows template name (not the raw UUID) when a run has no workflow/pipeline name", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "e47b50ca-2b31-433f-a472-e3370e409418",
					templateName: "nameless-template",
					status: "Succeeded",
					nodeCount: 1,
					totalEstimatedCost: 0.02,
					createdAt: "2026-06-04T01:00:00Z",
					finishedAt: "2026-06-04T01:05:48Z",
				},
			],
			total: 1,
		});

		renderList();

		// Primary label falls back to the template name, and the raw UUID is
		// never rendered as standalone text anywhere in the row.
		await waitFor(() => {
			expect(screen.getByText("nameless-template")).toBeInTheDocument();
		});
		expect(
			screen.queryByText("e47b50ca-2b31-433f-a472-e3370e409418"),
		).not.toBeInTheDocument();
		// The ID line still shows the asset-style short id.
		expect(screen.getByText("ID: e47b50ca")).toBeInTheDocument();
	});

	it("shows a placeholder label when a run has neither name nor template", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "f573e4cb-dff7-4485-adf9-39f16af0e009",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-04T02:00:00Z",
					finishedAt: "2026-06-04T02:17:32Z",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("未命名运行")).toBeInTheDocument();
		});
		expect(
			screen.queryByText("f573e4cb-dff7-4485-adf9-39f16af0e009"),
		).not.toBeInTheDocument();
		expect(screen.getByText("ID: f573e4cb")).toBeInTheDocument();
	});

	it("sorts execution metric columns", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-short",
					pipelineName: "short-run",
					workflowName: "short-run",
					status: "Succeeded",
					nodeCount: 1,
					totalEstimatedCost: 0.2,
					createdAt: "2026-06-02T01:00:00Z",
					finishedAt: "2026-06-02T01:00:10Z",
				},
				{
					id: "run-middle",
					pipelineName: "middle-run",
					workflowName: "middle-run",
					status: "Succeeded",
					nodeCount: 1,
					totalEstimatedCost: 0.4,
					createdAt: "2026-06-02T02:00:00Z",
					finishedAt: "2026-06-02T02:02:00Z",
				},
				{
					id: "run-long",
					pipelineName: "long-run",
					workflowName: "long-run",
					status: "Succeeded",
					nodeCount: 1,
					totalEstimatedCost: 1.2,
					createdAt: "2026-06-02T03:00:00Z",
					finishedAt: "2026-06-02T03:05:00Z",
				},
			],
			total: 3,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("short-run")).toBeInTheDocument();
			expect(screen.getByText("middle-run")).toBeInTheDocument();
			expect(screen.getByText("long-run")).toBeInTheDocument();
		});

		for (const label of ["耗时", "总成本", "创建时间", "完成时间"]) {
			expect(getColumnHeader(label)).toHaveClass(
				"ant-table-column-has-sorters",
			);
		}

		const durationHeader = getColumnHeader("耗时");
		fireEvent.click(durationHeader);
		await waitFor(() =>
			expectTableOrder(["short-run", "middle-run", "long-run"]),
		);
		fireEvent.click(durationHeader);
		await waitFor(() =>
			expectTableOrder(["long-run", "middle-run", "short-run"]),
		);

		fireEvent.click(getColumnHeader("完成时间"));
		await waitFor(() =>
			expectTableOrder(["short-run", "middle-run", "long-run"]),
		);
	});

	it("does not query Argo workflows for product label filters", async () => {
		mockListWorkflows.mockResolvedValue({
			items: [
				{
					name: "external-live-workflow",
					status: "Running",
					nodeCount: 1,
					createdAt: "2026-06-03T10:00:00Z",
					labels: { team: "external" },
				},
			],
		});
		mockListRuns.mockResolvedValue({ items: [], total: 0 });

		renderList("/pipeline?tab=executions&label=team%3Dexternal");

		await waitFor(() => {
			expect(mockListRuns).toHaveBeenCalledWith(
				expect.objectContaining({ view: "summary", excludeBatch: true }),
			);
		});
		expect(mockListWorkflows).not.toHaveBeenCalled();
		expect(
			screen.queryByText("external-live-workflow"),
		).not.toBeInTheDocument();
		expect(
			screen.getByText("暂无执行记录，部署流水线后将自动生成"),
		).toBeInTheDocument();
	});

	it("filters batch-scoped runs by workflow name", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-a",
					pipelineName: "batch-a",
					workflowName: "pipeline-aaa",
					status: "Succeeded",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
				},
				{
					id: "run-b",
					pipelineName: "batch-b",
					workflowName: "pipeline-bbb",
					status: "Succeeded",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
				},
			],
			total: 2,
		});

		renderBatchList();

		await waitFor(() => {
			expect(screen.getByText("pipeline-aaa")).toBeInTheDocument();
			expect(screen.getByText("pipeline-bbb")).toBeInTheDocument();
		});

		fireEvent.change(screen.getByPlaceholderText(/搜索名称/), {
			target: { value: "bbb" },
		});
		fireEvent.click(screen.getByRole("button", { name: "search" }));

		await waitFor(() => {
			expect(screen.getByText("pipeline-bbb")).toBeInTheDocument();
		});
		expect(screen.queryByText("pipeline-aaa")).not.toBeInTheDocument();
		expect(screen.getByText(/共 1 条/)).toBeInTheDocument();
	});

	// CYB-3486: 产品不再暴露"命名空间",列表列改为"资源池",按 executionTargetId
	// 反查池名;池已删除时显示占位而非泄露命名空间。
	it("renders a 资源池 column that resolves executionTargetId to the pool name", async () => {
		mockListExecutionTargets.mockResolvedValue([
			{
				id: "tgt-low",
				name: "交付集群-低配额池",
				cluster: "delivery-clust",
				namespace: "cyber-delivery-prod",
				argoServerConfigured: false,
				status: "available",
				isDefault: false,
			},
		]);
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-pooled",
					pipelineName: "pooled-run",
					workflowName: "pooled-run",
					status: "Succeeded",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					executionTargetId: "tgt-low",
					argoNamespace: "cyber-delivery-prod",
				},
				{
					id: "run-orphan",
					pipelineName: "orphan-run",
					workflowName: "orphan-run",
					status: "Succeeded",
					nodeCount: 1,
					createdAt: "2026-06-02T02:00:00Z",
					executionTargetId: "tgt-gone",
					argoNamespace: "cyber-delivery-prod",
				},
			],
			total: 2,
		});

		renderList();

		// 列头改成"资源池",不再有"命名空间"。
		await waitFor(() => {
			expect(getColumnHeader("资源池")).toBeTruthy();
		});
		expect(screen.queryByText("命名空间")).not.toBeInTheDocument();

		// 现存资源池 → 展示池名;两行都渲染出来。
		await waitFor(() => {
			expect(screen.getByText("交付集群-低配额池")).toBeInTheDocument();
		});
		expect(screen.getByText("orphan-run")).toBeInTheDocument();
		// 池已加载(map 非空)时,不再把原始命名空间当作可见文本泄露出来。
		expect(screen.queryByText("cyber-delivery-prod")).not.toBeInTheDocument();
	});

	// runtime_missing 只是运行期的临时诊断（同步断流、ledger 未更新）。它应该在
	// 运行中的任务上展示，但在明确终态（成功/失败/异常/取消/过期）上被抑制，且不能
	// 因为"非活动即终态"的反向判断误伤未知/新增状态，也不能遮住真实的失败原因。
	it("shows the runtime_missing tag while the run is still active", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-active-missing",
					pipelineName: "active-run",
					workflowName: "active-run",
					status: "Running",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					blockingReason: "runtime_missing",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("active-run")).toBeInTheDocument();
		});
		expect(screen.getByText("Runtime 不可用")).toBeInTheDocument();
	});

	it("hides a stale runtime_missing tag once the run reaches a terminal status", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-terminal-missing",
					pipelineName: "terminal-run",
					workflowName: "terminal-run",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					finishedAt: "2026-06-02T01:01:00Z",
					blockingReason: "runtime_missing",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("terminal-run")).toBeInTheDocument();
		});
		expect(screen.queryByText("Runtime 不可用")).not.toBeInTheDocument();
	});

	it("keeps the concrete failure reason on terminal runs", async () => {
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-terminal-failed",
					pipelineName: "unschedulable-run",
					workflowName: "unschedulable-run",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					finishedAt: "2026-06-02T01:01:00Z",
					failureReason: "unschedulable",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("unschedulable-run")).toBeInTheDocument();
		});
		expect(screen.getByText("调度失败")).toBeInTheDocument();
	});

	it("does not hide runtime_missing for unknown (non-terminal) statuses", async () => {
		// 未知/未来新增的状态既不在活动集合也不在终态集合，旧逻辑用
		// !isActiveWorkflowStatus 会把它当作终态而误隐藏诊断，这里守住它。
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-unknown",
					pipelineName: "unknown-status-run",
					workflowName: "unknown-status-run",
					status: "Terminating",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					blockingReason: "runtime_missing",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("unknown-status-run")).toBeInTheDocument();
		});
		expect(screen.getByText("Runtime 不可用")).toBeInTheDocument();
	});

	it("prefers the terminal failure reason over a stale blocking reason", async () => {
		// 兼容旧数据/异常响应：终态同时带残留 blockingReason=runtime_missing 与真实
		// failureReason 时，应展示真实失败原因，而不是先选中 runtime_missing 再隐藏。
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-both-reasons",
					pipelineName: "both-reasons-run",
					workflowName: "both-reasons-run",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-02T01:00:00Z",
					finishedAt: "2026-06-02T01:01:00Z",
					blockingReason: "runtime_missing",
					failureReason: "runtime_config_projection_failed",
				},
			],
			total: 1,
		});

		renderList();

		await waitFor(() => {
			expect(screen.getByText("both-reasons-run")).toBeInTheDocument();
		});
		expect(screen.getByText("运行配置投影失败")).toBeInTheDocument();
		expect(screen.queryByText("Runtime 不可用")).not.toBeInTheDocument();
	});
});
