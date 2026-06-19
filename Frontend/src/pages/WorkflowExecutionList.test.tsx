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
import type { WorkflowSummary } from "../api/workflowApi";
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
const mockListPipelines = vi.fn();
const mockGetPipelineRunWatcherStatus = vi.fn();

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

vi.mock("../api/pipelineApi", () => ({
	getPipelineRunWatcherStatus: (...args: unknown[]) =>
		mockGetPipelineRunWatcherStatus(...args),
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
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

vi.mock("antd", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		message: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
	};
});

const allWorkflows: WorkflowSummary[] = [
	{
		name: "successful-run",
		status: "Succeeded",
		nodeCount: 2,
		createdAt: "2026-06-02T01:00:00Z",
		finishedAt: "2026-06-02T01:00:20Z",
	},
	{
		name: "failed-run",
		status: "Failed",
		nodeCount: 1,
		createdAt: "2026-06-02T02:00:00Z",
		finishedAt: "2026-06-02T02:00:10Z",
	},
];

// Ledger runs are the source of truth for the executions list. Live Argo
// workflows are only fetched when a label filter is active (perf optimization),
// so status filtering is asserted against ledger Runs.
const ledgerRuns = [
	{
		id: "run-success",
		pipelineName: "successful-run",
		workflowName: "successful-run",
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
		mockListWorkflows.mockImplementation((params = {}) => {
			const status = (params as { status?: string }).status;
			return Promise.resolve({
				items: status
					? allWorkflows.filter((item) => item.status === status)
					: allWorkflows,
			});
		});
		mockDeleteWorkflow.mockResolvedValue({ message: "deleted" });
		mockDeleteRun.mockResolvedValue(undefined);
		mockRetryRun.mockResolvedValue({ message: "retry submitted" });
		mockResubmitRun.mockResolvedValue({ message: "resubmit submitted" });
		mockStopRun.mockResolvedValue({ message: "stop submitted" });
		mockSuspendRun.mockResolvedValue({ message: "suspend submitted" });
		mockResumeRun.mockResolvedValue({ message: "resume submitted" });
		mockTerminateRun.mockResolvedValue({ message: "terminate submitted" });
		mockListRuns.mockResolvedValue({
			items: [
				{
					id: "run-1",
					pipelineName: "successful-run",
					workflowName: "successful-run",
					status: "Succeeded",
					nodeCount: 2,
					totalEstimatedCost: 1.25,
					createdAt: "2026-06-02T01:00:00Z",
				},
			],
			total: 1,
		});
		mockListPipelines.mockResolvedValue({ items: [] });
		mockGetPipelineRunWatcherStatus.mockResolvedValue({
			id: "default",
			activeScanLimit: 100,
			lastSyncedRunCount: 2,
			consecutiveFailures: 0,
			totalScans: 5,
			totalErrors: 0,
			scanLagSeconds: 8,
			healthy: true,
			stale: false,
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
		expect(screen.getAllByText("—").length).toBeGreaterThan(0);
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

	it("keeps ledger records visible when live workflow listing is unavailable", async () => {
		mockListWorkflows.mockRejectedValue(new Error("argo unavailable"));
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
	});

	it("does not append live-only Argo workflows as product execution rows", async () => {
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
			expect(mockListWorkflows).toHaveBeenCalledWith(
				expect.objectContaining({ label: ["team=external"] }),
			);
		});
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

		fireEvent.change(screen.getByPlaceholderText("按名称搜索"), {
			target: { value: "bbb" },
		});
		fireEvent.click(screen.getByRole("button", { name: "search" }));

		await waitFor(() => {
			expect(screen.getByText("pipeline-bbb")).toBeInTheDocument();
		});
		expect(screen.queryByText("pipeline-aaa")).not.toBeInTheDocument();
		expect(screen.getByText(/共 1 条/)).toBeInTheDocument();
	});
});
