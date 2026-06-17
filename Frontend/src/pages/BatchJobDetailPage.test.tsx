// @vitest-environment jsdom

import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";

const mockMessage = vi.hoisted(() => ({
	success: vi.fn(),
	error: vi.fn(),
	warning: vi.fn(),
	info: vi.fn(),
}));

const mockGetBatchJob = vi.fn();
const mockGetBatchNodeSummary = vi.fn();
const mockListBatchNodeFailures = vi.fn();
const mockPauseBatchJob = vi.fn();
const mockRerunBatchJob = vi.fn();
const mockResumeBatchJob = vi.fn();
const mockContinueFullBatchJob = vi.fn();
const mockListPipelines = vi.fn();
const mockListPipelineVersions = vi.fn();

vi.mock("antd", async () => {
	const actual = await vi.importActual<typeof import("antd")>("antd");
	const AppMock = ({ children }: { children: ReactNode }) => children;
	(AppMock as typeof actual.App).useApp = () => ({ message: mockMessage });
	return {
		...actual,
		App: AppMock,
	};
});

vi.mock("../api/batchJobApi", async (importOriginal) => {
	const actual = await importOriginal<typeof import("../api/batchJobApi")>();
	return {
		...actual,
		getBatchJob: (...args: unknown[]) => mockGetBatchJob(...args),
		getBatchNodeSummary: (...args: unknown[]) =>
			mockGetBatchNodeSummary(...args),
		listBatchNodeFailures: (...args: unknown[]) =>
			mockListBatchNodeFailures(...args),
		pauseBatchJob: (...args: unknown[]) => mockPauseBatchJob(...args),
		rerunBatchJob: (...args: unknown[]) => mockRerunBatchJob(...args),
		resumeBatchJob: (...args: unknown[]) => mockResumeBatchJob(...args),
		continueFullBatchJob: (...args: unknown[]) =>
			mockContinueFullBatchJob(...args),
	};
});

vi.mock("../api/pipelineApi", async (importOriginal) => {
	const actual = await importOriginal<typeof import("../api/pipelineApi")>();
	return {
		...actual,
		listPipelines: (...args: unknown[]) => mockListPipelines(...args),
		listPipelineVersions: (...args: unknown[]) =>
			mockListPipelineVersions(...args),
	};
});

vi.mock("./WorkflowExecutionList", () => ({
	WorkflowExecutionList: () => <div data-testid="workflow-execution-list" />,
}));

import BatchJobDetailPage, { formatRerunFeedback } from "./BatchJobDetailPage";

function renderBatchJobDetail() {
	return render(
		<MemoryRouter initialEntries={["/pipeline/batch/batch-1"]}>
			<Routes>
				<Route path="/pipeline/batch/:id" element={<BatchJobDetailPage />} />
			</Routes>
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

describe("BatchJobDetailPage", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		let refreshCount = 0;
		mockGetBatchJob.mockImplementation(async () => {
			refreshCount += 1;
			return {
				id: "batch-1",
				name: "Batch 1",
				templateId: "tmpl-1",
				templateVersion: 3,
				totalCount: 2,
				completedCount: 0,
				failedCount: 1,
				pilotCount: 0,
				pilotPhase: "none",
				status: "running",
				createdAt: "2026-06-17T00:00:00Z",
				updatedAt: `2026-06-17T00:00:0${refreshCount}Z`,
			};
		});
		mockGetBatchNodeSummary.mockResolvedValue({
			batchJobId: "batch-1",
			templateId: "tmpl-1",
			templateVersion: 3,
			subtasks: {
				total: 2,
				completed: 0,
				failed: 1,
				running: 1,
				pending: 0,
				paused: false,
			},
			nodes: [],
			dataCoverage: {
				runsWithNodeRows: 2,
				runsTotal: 2,
				complete: true,
			},
			generatedAt: "2026-06-17T00:00:00Z",
		});
		mockListBatchNodeFailures.mockResolvedValue({
			items: [],
			total: 0,
			page: 1,
			pageSize: 20,
		});
		mockPauseBatchJob.mockResolvedValue({ status: "paused" });
		mockRerunBatchJob.mockResolvedValue({
			status: "succeeded",
			dryRun: false,
			matchedCount: 1,
			retriedCount: 1,
			templateId: "tmpl-1",
			templateVersion: 3,
			skipped: [],
		});
		mockResumeBatchJob.mockResolvedValue(undefined);
		mockContinueFullBatchJob.mockResolvedValue(undefined);
		mockListPipelines.mockResolvedValue({
			items: [
				{
					id: "tmpl-1",
					name: "Template 1",
					version: 3,
					pipeline: { name: "Template 1", version: "3", nodes: [], edges: [] },
					nodeCount: 0,
					createdAt: "2026-06-17T00:00:00Z",
				},
			],
			total: 1,
			page: 1,
			pageSize: 200,
		});
		mockListPipelineVersions.mockResolvedValue([
			{
				id: "tmpl-1",
				name: "Template 1",
				version: 3,
				pipeline: { name: "Template 1", version: "3", nodes: [], edges: [] },
				nodeCount: 0,
				createdAt: "2026-06-17T00:00:00Z",
			},
		]);
	});

	afterEach(() => {
		cleanup();
	});

	it("maps failed rerun to error feedback", () => {
		expect(
			formatRerunFeedback({
				status: "failed",
				matchedCount: 2,
			}),
		).toEqual({
			level: "error",
			text: "重跑失败，没有可重新提交的子任务",
		});
	});

	it("maps partial rerun to warning feedback", () => {
		expect(
			formatRerunFeedback({
				status: "partial_success",
				matchedCount: 2,
				retriedCount: 1,
				skipped: [{ itemId: "item-2", reason: "duplicate" }],
			}),
		).toEqual({
			level: "warning",
			text: "重跑部分提交成功：1 / 2",
		});
	});

	it("maps successful rerun to success feedback", () => {
		expect(
			formatRerunFeedback({
				status: "succeeded",
				matchedCount: 1,
				retriedCount: 1,
				skipped: [],
			}),
		).toEqual({
			level: "success",
			text: "重跑已提交",
		});
	});

	it("keeps a single polling interval while a running batch refreshes", async () => {
		const setIntervalSpy = vi.spyOn(window, "setInterval");

		renderBatchJobDetail();

		expect(await screen.findByText("Batch 1")).toBeInTheDocument();
		await waitFor(() => expect(mockGetBatchJob).toHaveBeenCalledTimes(1));
		expect(
			setIntervalSpy.mock.calls.filter(([, delay]) => delay === 5_000),
		).toHaveLength(1);

		const pollOnce = setIntervalSpy.mock.calls.find(
			([, delay]) => delay === 5_000,
		)?.[0];
		expect(typeof pollOnce).toBe("function");
		await act(async () => {
			(pollOnce as TimerHandler as () => void)();
			await Promise.resolve();
		});

		await waitFor(() => expect(mockGetBatchJob).toHaveBeenCalledTimes(2));
		expect(
			setIntervalSpy.mock.calls.filter(([, delay]) => delay === 5_000),
		).toHaveLength(1);
	});

	it("does not refetch template metadata on silent polling when template is unchanged", async () => {
		const setIntervalSpy = vi.spyOn(window, "setInterval");

		renderBatchJobDetail();

		expect(await screen.findByText("Batch 1")).toBeInTheDocument();
		await waitFor(() => expect(mockGetBatchJob).toHaveBeenCalledTimes(1));
		expect(mockListPipelines).toHaveBeenCalledTimes(1);
		expect(mockListPipelineVersions).toHaveBeenCalledTimes(1);

		const pollOnce = setIntervalSpy.mock.calls.find(
			([, delay]) => delay === 5_000,
		)?.[0];
		expect(typeof pollOnce).toBe("function");
		await act(async () => {
			(pollOnce as TimerHandler as () => void)();
			await Promise.resolve();
		});

		await waitFor(() => expect(mockGetBatchJob).toHaveBeenCalledTimes(2));
		expect(mockGetBatchNodeSummary).toHaveBeenCalledTimes(2);
		expect(mockListPipelines).toHaveBeenCalledTimes(1);
		expect(mockListPipelineVersions).toHaveBeenCalledTimes(1);
	});
});
