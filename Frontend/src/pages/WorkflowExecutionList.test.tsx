// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
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
const mockListDeployments = vi.fn();
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
	listDeployments: (...args: unknown[]) => mockListDeployments(...args),
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
			<LocationProbe />
		</MemoryRouter>,
	);
}

function LocationProbe() {
	const location = useLocation();
	return (
		<div data-testid="location">
			{location.pathname}
			{location.search}
		</div>
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
		mockListDeployments.mockResolvedValue([
			{
				id: "run-1",
				pipelineName: "successful-run",
				workflowName: "successful-run",
				status: "Succeeded",
				nodeCount: 2,
				templateVersion: 3,
				createdAt: "2026-06-02T01:00:00Z",
				finishedAt: "2026-06-02T01:00:20Z",
			},
		]);
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

	it("merges deployment metadata for display ids and template versions", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		expect(mockListDeployments).toHaveBeenCalled();
		await waitFor(() => {
			expect(screen.getByText("ID: run1")).toBeInTheDocument();
			expect(screen.getByText("模板 v3")).toBeInTheDocument();
		});
	});

	it("opens execution detail with durable run id when available", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		fireEvent.click(screen.getAllByRole("button", { name: "查看" })[0]);

		expect(screen.getByTestId("location")).toHaveTextContent(
			"/pipeline/executions/successful-run?runId=run-1",
		);
	});

	it("shows an empty state when live workflow listing has no records", async () => {
		mockListWorkflows.mockResolvedValue({ items: [] });
		mockListDeployments.mockResolvedValue([]);

		renderList();

		await waitFor(() => {
			expect(
				screen.getByText("暂无执行记录，部署流水线后将自动生成"),
			).toBeInTheDocument();
		});
	});

	it("shows a service error when live workflow listing is unavailable", async () => {
		mockListWorkflows.mockRejectedValue(new Error("argo unavailable"));

		renderList();

		await waitFor(() => {
			expect(screen.getByText("服务不可用")).toBeInTheDocument();
		});
		expect(screen.getByText("argo unavailable")).toBeInTheDocument();
	});
});
