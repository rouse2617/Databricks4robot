// @vitest-environment jsdom

import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../api/pipelineClient";

const mockGetWorkflow = vi.fn();
const mockGetWorkflowLogs = vi.fn();
const mockListPipelineRuns = vi.fn(() =>
	Promise.resolve({ items: [], total: 0 }),
);
const mockGetPipelineRun = vi.fn();
const mockGetPipelineRunByWorkflowName = vi.fn();
const mockGetPipelineRunCostSummary = vi.fn();
const mockListPipelineRunAssetNodes = vi.fn();
const mockListPipelineRunEvents = vi.fn();

class MockEventSource extends EventTarget {
	static instances: MockEventSource[] = [];
	onopen: (() => void) | null = null;
	onerror: (() => void) | null = null;
	url: string;
	closed = false;

	constructor(url: string) {
		super();
		this.url = url;
		MockEventSource.instances.push(this);
	}

	close() {
		this.closed = true;
	}
}

vi.mock("../api/workflowApi", () => ({
	getWorkflow: (...args: unknown[]) => mockGetWorkflow(...args),
	getWorkflowLogs: (...args: unknown[]) => mockGetWorkflowLogs(...args),
	getWorkflowLogStreamUrl: vi.fn(() => "/api/v1/workflows/wf/logs/stream"),
}));

vi.mock("../api/pipelineApi", () => ({
	getPipelineRun: (...args: unknown[]) => mockGetPipelineRun(...args),
	getPipelineRunCostSummary: (...args: unknown[]) =>
		mockGetPipelineRunCostSummary(...args),
	listPipelineRunAssetNodes: (...args: unknown[]) =>
		mockListPipelineRunAssetNodes(...args),
	listPipelineRunEvents: (...args: unknown[]) =>
		mockListPipelineRunEvents(...args),
	listPipelineRuns: (...args: unknown[]) => mockListPipelineRuns(...args),
	getPipelineRunByWorkflowName: (...args: unknown[]) =>
		mockGetPipelineRunByWorkflowName(...args),
}));

import { useWorkflowDetail } from "./useWorkflowDetail";

describe("useWorkflowDetail", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		MockEventSource.instances = [];
		vi.stubGlobal("EventSource", MockEventSource);
		mockGetPipelineRun.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "not found"),
		);
		mockGetPipelineRunByWorkflowName.mockResolvedValue(null);
		mockGetPipelineRunCostSummary.mockResolvedValue(null);
		mockListPipelineRunAssetNodes.mockResolvedValue({
			items: [],
			summary: null,
		});
		mockListPipelineRunEvents.mockResolvedValue({
			items: [],
			nextCursor: undefined,
		});
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("maps 404 to not_found load error", async () => {
		mockGetWorkflow.mockRejectedValue(
			new ApiError(404, "WORKFLOW_NOT_FOUND", "workflow not found"),
		);

		const { result } = renderHook(() => useWorkflowDetail("wf-missing"));

		await waitFor(() => expect(result.current.loading).toBe(false));

		expect(result.current.workflow).toBeNull();
		expect(result.current.loadError).toEqual({
			kind: "not_found",
			message: "workflow not found",
		});
	});

	it("maps non-404 API errors to error load error", async () => {
		mockGetWorkflow.mockRejectedValue(
			new ApiError(500, "INTERNAL", "argo unavailable"),
		);

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() => expect(result.current.loading).toBe(false));

		expect(result.current.workflow).toBeNull();
		expect(result.current.loadError).toEqual({
			kind: "error",
			message: "argo unavailable",
		});
	});

	it("loads bounded log metadata for selected node", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Succeeded",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [
				{
					id: "node-1",
					name: "node-1",
					displayName: "node-1",
					phase: "Succeeded",
				},
			],
		});
		mockGetWorkflowLogs.mockResolvedValue({
			workflowName: "wf-1",
			nodeId: "node-1",
			podName: "pod-1",
			container: "main",
			source: "argo-live",
			logs: "hello\n",
			lineCount: 1,
			truncated: false,
			nextCursor: null,
			truncation: {
				bounded: true,
				tailLines: 200,
				maxTailLines: 2000,
				limitBytes: 262144,
				maxLimitBytes: 2097152,
				bytesTruncated: false,
			},
			pagination: {
				available: false,
				nextCursor: null,
				reason:
					"live Argo logs do not provide stable historical cursor pagination",
			},
			window: {
				mode: "tail",
				tailLines: 200,
				limitBytes: 262144,
				scope: "bounded-live-window",
			},
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() => expect(result.current.workflow).not.toBeNull());
		const node = result.current.workflow?.nodes[0];
		expect(node).toBeDefined();
		act(() => {
			result.current.selectNode(node ?? null);
		});

		await waitFor(() =>
			expect(result.current.logState.content).toBe("hello\n"),
		);

		expect(result.current.logState.response?.pagination?.available).toBe(false);
		expect(result.current.logState.response?.window?.scope).toBe(
			"bounded-live-window",
		);
	});

	it("appends workflow log stream line events", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [
				{
					id: "node-1",
					name: "node-1",
					displayName: "node-1",
					phase: "Running",
				},
			],
		});
		mockGetWorkflowLogs.mockResolvedValue({
			workflowName: "wf-1",
			nodeId: "node-1",
			podName: "pod-1",
			container: "main",
			source: "argo-live",
			logs: "",
			lineCount: 0,
			truncated: false,
			truncation: {
				bounded: true,
				tailLines: 200,
				maxTailLines: 2000,
				limitBytes: 262144,
				maxLimitBytes: 2097152,
			},
			pagination: { available: false, nextCursor: null },
			window: { mode: "tail", scope: "bounded-live-window" },
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() => expect(result.current.workflow).not.toBeNull());
		const node = result.current.workflow?.nodes[0];
		expect(node).toBeDefined();
		act(() => {
			result.current.selectNode(node ?? null);
		});
		await waitFor(() => expect(result.current.logState.content).toBe(""));

		act(() => {
			result.current.startFollowLogs();
		});
		const source = MockEventSource.instances[0];
		act(() => {
			source.onopen?.();
			source.dispatchEvent(
				new MessageEvent("log", { data: JSON.stringify({ line: "hello" }) }),
			);
		});

		await waitFor(() =>
			expect(result.current.logState.content).toContain("hello\n"),
		);
		expect(result.current.logState.followStatus).toBe("connected");
	});

	it("keeps a single polling interval while an active workflow refreshes", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		const setIntervalSpy = vi.spyOn(window, "setInterval");

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() =>
			expect(result.current.workflow?.status).toBe("Running"),
		);
		expect(
			setIntervalSpy.mock.calls.filter(([, delay]) => delay === 8_000),
		).toHaveLength(1);

		await act(async () => {
			const pollOnce = setIntervalSpy.mock.calls.find(
				([, delay]) => delay === 8_000,
			)?.[0];
			expect(typeof pollOnce).toBe("function");
			(pollOnce as TimerHandler as () => void)();
			await Promise.resolve();
		});

		await waitFor(() => expect(mockGetWorkflow).toHaveBeenCalledTimes(2));
		expect(
			setIntervalSpy.mock.calls.filter(([, delay]) => delay === 8_000),
		).toHaveLength(1);
	});

	it("skips pipeline run lookups for external workflows", async () => {
		const consoleErrorSpy = vi
			.spyOn(console, "error")
			.mockImplementation(() => undefined);
		mockGetWorkflow.mockResolvedValue({
			name: "wf-external",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			labels: {
				"workflows.argoproj.io/completed": "false",
				"workflows.argoproj.io/phase": "Running",
			},
			nodes: [],
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-external"));

		await waitFor(() =>
			expect(result.current.workflow?.name).toBe("wf-external"),
		);
		await waitFor(() =>
			expect(result.current.runEventState.error).toBe(
				"未找到关联的 DataBrew pipeline run",
			),
		);
		expect(mockGetPipelineRunByWorkflowName).not.toHaveBeenCalled();
		expect(mockGetPipelineRun).not.toHaveBeenCalled();
		expect(consoleErrorSpy).not.toHaveBeenCalled();
	});
});
