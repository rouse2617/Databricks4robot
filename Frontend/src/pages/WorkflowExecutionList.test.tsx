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
const mockListPipelineRuns = vi.fn();
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
	listPipelineRuns: (...args: unknown[]) => mockListPipelineRuns(...args),
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
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

function renderList(initialEntry = "/pipeline?tab=executions") {
	return render(
		<MemoryRouter initialEntries={[initialEntry]}>
			<WorkflowExecutionList />
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
		mockListPipelineRuns.mockResolvedValue([
			{
				id: "run-1",
				pipelineName: "successful-run",
				workflowName: "successful-run",
				status: "Succeeded",
				nodeCount: 2,
				totalEstimatedCost: 1.25,
				createdAt: "2026-06-02T01:00:00Z",
			},
		]);
		mockListPipelines.mockResolvedValue([]);
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
		renderList("/pipeline?tab=executions&status=Failed");

		await waitFor(() => {
			expect(mockListWorkflows).toHaveBeenLastCalledWith(
				expect.objectContaining({ status: "Failed" }),
			);
			expect(screen.queryByText("successful-run")).not.toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
	});

	it("clears status filters when reset is clicked", async () => {
		renderList("/pipeline?tab=executions&status=Failed");

		await waitFor(() => {
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});

		fireEvent.click(screen.getByRole("button", { name: "重 置" }));

		await waitFor(() => {
			expect(mockListWorkflows).toHaveBeenLastCalledWith(
				expect.objectContaining({ status: undefined }),
			);
			expect(screen.getByText("successful-run")).toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
	});

	it("merges run-level estimated cost from pipeline runs and shows a quiet empty state otherwise", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		expect(mockListPipelineRuns).toHaveBeenCalled();
		expect(screen.getAllByText("总成本").length).toBeGreaterThan(0);
		expect(screen.getByText("$1.25")).toBeInTheDocument();
		expect(screen.getAllByText("—").length).toBeGreaterThan(0);
	});

	it("shows ledger-only runs with estimated cost after live Argo workflow TTL cleanup", async () => {
		mockListWorkflows.mockResolvedValue({ items: [] });
		mockListPipelineRuns.mockResolvedValue([
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
		]);

		renderList();

		await waitFor(() => {
			expect(screen.getByText("ttl-cleaned-workflow")).toBeInTheDocument();
		});
		expect(screen.getByText("$0.0090")).toBeInTheDocument();
		expect(screen.getByText("ID: runledge")).toBeInTheDocument();
	});

	it("keeps ledger records visible when live workflow listing is unavailable", async () => {
		mockListWorkflows.mockRejectedValue(new Error("argo unavailable"));
		mockListPipelineRuns.mockResolvedValue([
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
		]);

		renderList();

		await waitFor(() => {
			expect(screen.getByText("ledger-workflow")).toBeInTheDocument();
		});
		expect(screen.queryByText("服务不可用")).not.toBeInTheDocument();
		expect(screen.getByText("$1.50")).toBeInTheDocument();
	});
});
