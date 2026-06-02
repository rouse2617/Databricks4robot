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
	});

	afterEach(() => {
		cleanup();
		vi.clearAllMocks();
	});

	it("filters immediately when a status summary card is clicked", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});

		fireEvent.click(
			screen.getByRole("button", { name: "筛选 Failed 执行记录" }),
		);

		await waitFor(() => {
			expect(mockListWorkflows).toHaveBeenLastCalledWith(
				expect.objectContaining({ status: "Failed" }),
			);
			expect(screen.queryByText("successful-run")).not.toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
	});

	it("clears the status filter when the active status card is clicked again", async () => {
		renderList();

		await waitFor(() => {
			expect(screen.getByText("successful-run")).toBeInTheDocument();
		});

		const failedCard = screen.getByRole("button", {
			name: "筛选 Failed 执行记录",
		});
		fireEvent.click(failedCard);

		await waitFor(() => {
			expect(mockListWorkflows).toHaveBeenLastCalledWith(
				expect.objectContaining({ status: "Failed" }),
			);
		});

		fireEvent.click(failedCard);

		await waitFor(() => {
			expect(mockListWorkflows).toHaveBeenLastCalledWith(
				expect.objectContaining({ status: undefined }),
			);
			expect(screen.getByText("successful-run")).toBeInTheDocument();
			expect(screen.getByText("failed-run")).toBeInTheDocument();
		});
	});
});
