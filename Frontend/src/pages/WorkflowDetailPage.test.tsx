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

		expect(screen.getByText("asset-pipeline")).toBeInTheDocument();
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
});
