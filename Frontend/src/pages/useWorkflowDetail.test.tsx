// @vitest-environment jsdom

import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../api/pipelineClient";

const mockGetWorkflow = vi.fn();
const mockGetWorkflowLogs = vi.fn();
const mockGetWorkflowLogStreamUrl = vi.fn(
	() => "/api/v1/workflows/wf/logs/stream",
);
const mockGetRun = vi.fn();
const mockGetRunByWorkflowName = vi.fn();
const mockGetRunCostSummary = vi.fn();
const mockGetRunRuntime = vi.fn();
const mockListRunAssetNodes = vi.fn();
const mockListRunEvents = vi.fn();
const mockListRunInputs = vi.fn();
const mockListRunOutputs = vi.fn();

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
	getWorkflowLogStreamUrl: (...args: unknown[]) =>
		mockGetWorkflowLogStreamUrl(...args),
}));

vi.mock("../api/runApi", () => ({
	getRun: (...args: unknown[]) => mockGetRun(...args),
	getRunByWorkflowName: (...args: unknown[]) =>
		mockGetRunByWorkflowName(...args),
	getRunCostSummary: (...args: unknown[]) => mockGetRunCostSummary(...args),
	getRunRuntime: (...args: unknown[]) => mockGetRunRuntime(...args),
	listRunAssetNodes: (...args: unknown[]) => mockListRunAssetNodes(...args),
	listRunEvents: (...args: unknown[]) => mockListRunEvents(...args),
	listRunInputs: (...args: unknown[]) => mockListRunInputs(...args),
	listRunOutputs: (...args: unknown[]) => mockListRunOutputs(...args),
}));

import { useWorkflowDetail } from "./useWorkflowDetail";

describe("useWorkflowDetail", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		MockEventSource.instances = [];
		vi.stubGlobal("EventSource", MockEventSource);
		mockGetRun.mockRejectedValue(new ApiError(404, "NOT_FOUND", "not found"));
		mockGetRunByWorkflowName.mockResolvedValue(null);
		mockGetRunCostSummary.mockResolvedValue(null);
		mockListRunAssetNodes.mockResolvedValue({
			items: [],
			summary: null,
		});
		mockListRunEvents.mockResolvedValue({
			items: [],
			nextCursor: undefined,
		});
		mockListRunInputs.mockResolvedValue({
			runId: "run-1",
			items: [],
			total: 0,
		});
		mockListRunOutputs.mockResolvedValue({
			runId: "run-1",
			items: [],
			total: 0,
		});
		mockGetRunRuntime.mockResolvedValue({
			runId: "run-1",
			runtime: { runtimeType: "argo" },
		});
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
	});

	it("maps 404 to not_found load error", async () => {
		const consoleErrorSpy = vi
			.spyOn(console, "error")
			.mockImplementation(() => undefined);
		mockGetRunByWorkflowName.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "not found"),
		);
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
		expect(consoleErrorSpy).not.toHaveBeenCalled();
	});

	it("loads DataBrew run ledger when the Argo workflow is gone", async () => {
		mockGetWorkflow.mockRejectedValue(
			new ApiError(404, "WORKFLOW_NOT_FOUND", "workflow not found"),
		);
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-expired",
			pipelineName: "pipeline",
			status: "Succeeded",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
			finishedAt: "2026-06-03T00:10:00Z",
		});
		mockListRunEvents.mockResolvedValue({
			items: [
				{
					id: "event-1",
					runId: "run-1",
					eventType: "run_completed",
					subjectType: "run",
					subjectId: "run-1",
					occurredAt: "2026-06-03T00:10:00Z",
				},
			],
			nextCursor: undefined,
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-expired"));

		await waitFor(() => expect(result.current.loading).toBe(false));
		await waitFor(() =>
			expect(result.current.runEventState.run?.status).toBe("Succeeded"),
		);

		expect(result.current.workflow).toBeNull();
		expect(result.current.loadError?.kind).toBe("not_found");
		expect(mockGetRunByWorkflowName).toHaveBeenCalledWith("wf-expired");
		expect(mockListRunEvents).toHaveBeenCalledWith("run-1", {
			limit: 100,
			cursor: undefined,
		});
	});

	it("fetches run ledger data exactly once per mount, not twice", async () => {
		// Regression test: loadWorkflow's post-getWorkflow refreshDetailData()
		// and loadRunEvents' own mount effect used to both independently
		// trigger the same run-ledger fetch (CYB-3068).
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Succeeded",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Succeeded",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
			finishedAt: "2026-06-03T00:10:00Z",
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() => expect(result.current.loading).toBe(false));
		await waitFor(() =>
			expect(result.current.runEventState.run?.status).toBe("Succeeded"),
		);

		expect(mockGetWorkflow).toHaveBeenCalledTimes(1);
		expect(mockGetRunByWorkflowName).toHaveBeenCalledTimes(1);
		expect(mockListRunEvents).toHaveBeenCalledTimes(1);
		expect(mockListRunAssetNodes).toHaveBeenCalledTimes(1);
		expect(mockGetRunCostSummary).toHaveBeenCalledTimes(1);
		expect(mockListRunInputs).toHaveBeenCalledTimes(1);
		expect(mockListRunOutputs).toHaveBeenCalledTimes(1);
		expect(mockGetRunRuntime).toHaveBeenCalledTimes(1);
	});

	it("loads run-id detail and resolves runtime workflow for terminal runs", async () => {
		mockGetRun.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-expired",
			pipelineName: "pipeline",
			status: "Succeeded",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
			finishedAt: "2026-06-03T00:10:00Z",
		});
		mockListRunEvents.mockResolvedValue({
			items: [
				{
					id: "event-1",
					runId: "run-1",
					eventType: "run_completed",
					subjectType: "run",
					subjectId: "run-1",
					occurredAt: "2026-06-03T00:10:00Z",
				},
			],
			nextCursor: undefined,
		});
		mockGetWorkflow.mockResolvedValue({
			name: "wf-expired",
			status: "Succeeded",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		mockListRunInputs.mockResolvedValue({
			runId: "run-1",
			total: 1,
			items: [
				{
					id: "input-config",
					runId: "run-1",
					type: "config",
					refId: "cfg-1",
				},
			],
		});
		mockListRunOutputs.mockResolvedValue({
			runId: "run-1",
			total: 1,
			items: [
				{
					id: "output-logs",
					runId: "run-1",
					type: "logs",
					refId: "wf-expired",
				},
			],
		});
		mockGetRunRuntime.mockResolvedValue({
			runId: "run-1",
			runtime: {
				runtimeType: "argo",
				workflowName: "wf-expired",
				namespace: "video-proc-dev",
			},
		});

		const { result } = renderHook(() =>
			useWorkflowDetail("run-1", { lookupMode: "runId" }),
		);

		await waitFor(() => expect(result.current.loading).toBe(false));
		await waitFor(() =>
			expect(result.current.runEventState.run?.id).toBe("run-1"),
		);

		expect(result.current.workflow?.name).toBe("wf-expired");
		expect(result.current.loadError).toBeNull();
		expect(mockGetRun).toHaveBeenCalledWith("run-1");
		expect(mockGetWorkflow).toHaveBeenCalledWith("wf-expired");
		expect(mockGetRunByWorkflowName).not.toHaveBeenCalled();
		expect(mockListRunEvents).toHaveBeenCalledWith("run-1", {
			limit: 100,
			cursor: undefined,
		});
		expect(mockListRunInputs).toHaveBeenCalledWith("run-1");
		expect(mockListRunOutputs).toHaveBeenCalledWith("run-1");
		expect(mockGetRunRuntime).toHaveBeenCalledWith("run-1");
		expect(result.current.runMetadataState.inputs?.items[0]?.refId).toBe(
			"cfg-1",
		);
		expect(result.current.runMetadataState.outputs?.items[0]?.type).toBe(
			"logs",
		);
		expect(result.current.runMetadataState.runtime?.runtime.namespace).toBe(
			"video-proc-dev",
		);
		expect(result.current.runMetadataState.error).toBeNull();
	});

	it("keeps Run detail usable when metadata endpoints fail", async () => {
		mockGetRun.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Succeeded",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
		});
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Succeeded",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		mockListRunEvents.mockResolvedValue({
			items: [
				{
					id: "event-1",
					runId: "run-1",
					eventType: "run_completed",
					subjectType: "run",
					subjectId: "run-1",
					occurredAt: "2026-06-03T00:10:00Z",
				},
			],
			nextCursor: undefined,
		});
		mockListRunInputs.mockRejectedValue(new Error("inputs unavailable"));

		const { result } = renderHook(() =>
			useWorkflowDetail("run-1", { lookupMode: "runId" }),
		);

		await waitFor(() => expect(result.current.loading).toBe(false));
		await waitFor(() =>
			expect(result.current.runEventState.items.length).toBe(1),
		);

		expect(result.current.workflow?.name).toBe("wf-1");
		expect(result.current.loadError).toBeNull();
		expect(result.current.runMetadataState.error).toContain(
			"inputs unavailable",
		);
	});

	it("uses resolved workflowName for logs and log stream in run-id mode", async () => {
		mockGetRun.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Running",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
		});
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
			logs: "hello\n",
			lineCount: 1,
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

		const { result } = renderHook(() =>
			useWorkflowDetail("run-1", { lookupMode: "runId" }),
		);

		await waitFor(() => expect(result.current.workflow?.name).toBe("wf-1"));
		const node = result.current.workflow?.nodes[0];
		act(() => {
			result.current.selectNode(node ?? null);
		});
		await waitFor(() =>
			expect(mockGetWorkflowLogs).toHaveBeenCalledWith("wf-1", "node-1"),
		);

		act(() => {
			result.current.startFollowLogs();
		});
		expect(mockGetWorkflowLogStreamUrl).toHaveBeenCalledWith("wf-1", "node-1");
	});

	it("keeps polling ledger data when workflow is missing but run is still active", async () => {
		vi.useFakeTimers();
		mockGetWorkflow.mockRejectedValue(
			new ApiError(404, "WORKFLOW_NOT_FOUND", "workflow not found"),
		);
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-pending",
			workflowName: "wf-pending",
			pipelineName: "pipeline",
			status: "Pending",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
		});
		mockListRunEvents.mockResolvedValue({
			items: [],
			nextCursor: undefined,
		});
		const setIntervalSpy = vi.spyOn(window, "setInterval");

		renderHook(() => useWorkflowDetail("wf-pending"));

		await act(async () => {
			await Promise.resolve();
		});
		expect(mockGetRunByWorkflowName).toHaveBeenCalledWith("wf-pending");
		expect(
			setIntervalSpy.mock.calls.filter(([, delay]) => delay === 8_000),
		).toHaveLength(1);

		const pollOnce = setIntervalSpy.mock.calls.find(
			([, delay]) => delay === 8_000,
		)?.[0];
		expect(typeof pollOnce).toBe("function");

		await act(async () => {
			(pollOnce as TimerHandler as () => void)();
			await Promise.resolve();
		});

		expect(mockGetWorkflow).toHaveBeenCalledTimes(2);
		// One mount + one poll tick = 2 ledger fetches, not 4: loadWorkflow and
		// loadRunEvents both used to independently trigger the same ledger load
		// on every cycle (see CYB-3068); skipNextRunEventsLoadRef now dedupes it.
		expect(mockGetRunByWorkflowName).toHaveBeenCalledTimes(2);
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

	it("suppresses console output for expected 404 errors while loading run details", async () => {
		const consoleErrorSpy = vi
			.spyOn(console, "error")
			.mockImplementation(() => undefined);
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Pending",
			nodeCount: 1,
			createdAt: "2026-06-03T00:01:00Z",
		});
		mockListRunEvents.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "run events not found"),
		);
		mockListRunAssetNodes.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "run assets not found"),
		);
		mockGetRunCostSummary.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "run cost not found"),
		);

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() =>
			expect(result.current.runEventState.error).toContain(
				"run events not found",
			),
		);

		expect(consoleErrorSpy).not.toHaveBeenCalled();
		expect(result.current.assetNodeState.error).toContain(
			"run events not found",
		);
		expect(result.current.costSummaryState.error).toContain(
			"run events not found",
		);
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

	it("moves stalled workflow log streams out of the connecting state", async () => {
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

		vi.useFakeTimers();
		act(() => {
			result.current.startFollowLogs();
		});
		expect(result.current.logState.followStatus).toBe("connecting");

		act(() => {
			vi.advanceTimersByTime(5_000);
		});

		expect(result.current.logState.followStatus).toBe("connected");
		expect(result.current.logState.followMessage).toBe(
			"实时日志已连接，等待日志事件",
		);
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

	it("does not keep polling stale Argo status after a terminal run with no newer active nodes", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [],
		});
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Failed",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
			message: "workflow shutdown with strategy: Stop",
		});
		const setIntervalSpy = vi.spyOn(window, "setInterval");
		const clearIntervalSpy = vi.spyOn(window, "clearInterval");

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() =>
			expect(result.current.runEventState.run?.status).toBe("Failed"),
		);

		expect(result.current.workflow?.status).toBe("Running");
		const workflowPolls = setIntervalSpy.mock.calls.filter(
			([, delay]) => delay === 8_000,
		);
		if (workflowPolls.length > 0) {
			expect(clearIntervalSpy).toHaveBeenCalledWith(workflowPolls[0][0]);
		}
	});

	it("keeps polling while an active workflow retry node is newer than the terminal run snapshot", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-1",
			status: "Running",
			createdAt: "2026-06-03T00:00:00Z",
			nodes: [
				{
					id: "node-1",
					name: "wf-1.node-1",
					displayName: "node-1",
					phase: "Pending",
					startedAt: "2026-06-03T00:11:00Z",
				},
			],
		});
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-1",
			workflowName: "wf-1",
			pipelineName: "pipeline",
			status: "Failed",
			nodeCount: 1,
			createdAt: "2026-06-03T00:00:00Z",
			finishedAt: "2026-06-03T00:10:00Z",
			message: "previous attempt failed",
		});
		const setIntervalSpy = vi.spyOn(window, "setInterval");

		const { result } = renderHook(() => useWorkflowDetail("wf-1"));

		await waitFor(() =>
			expect(result.current.runEventState.run?.status).toBe("Failed"),
		);
		await waitFor(() =>
			expect(
				setIntervalSpy.mock.calls.filter(([, delay]) => delay === 8_000),
			).toHaveLength(1),
		);
	});

	it("treats workflows as external only after Run lookup misses", async () => {
		const consoleErrorSpy = vi
			.spyOn(console, "error")
			.mockImplementation(() => undefined);
		mockGetRunByWorkflowName.mockRejectedValue(
			new ApiError(404, "NOT_FOUND", "not found"),
		);
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
				"未找到关联的 DataBrew Run",
			),
		);
		expect(mockGetRunByWorkflowName).toHaveBeenCalledWith("wf-external");
		expect(mockGetRun).toHaveBeenCalledWith("wf-external");
		expect(consoleErrorSpy).not.toHaveBeenCalled();
	});

	it("loads Run data for Argo-labeled resubmitted workflows", async () => {
		mockGetWorkflow.mockResolvedValue({
			name: "wf-resubmitted",
			status: "Succeeded",
			createdAt: "2026-06-03T00:00:00Z",
			labels: {
				"workflows.argoproj.io/completed": "true",
				"workflows.argoproj.io/phase": "Succeeded",
				"workflows.argoproj.io/resubmitted-from-workflow": "wf-source",
			},
			nodes: [],
		});
		mockGetRunByWorkflowName.mockResolvedValue({
			id: "run-resubmitted",
			workflowName: "wf-resubmitted",
			pipelineJSON: { nodes: [] },
		});
		mockListRunEvents.mockResolvedValue({
			items: [
				{
					id: "event-1",
					runId: "run-resubmitted",
					eventType: "run_resubmitted",
					subjectType: "run",
					subjectId: "run-resubmitted",
					occurredAt: "2026-06-03T00:00:00Z",
				},
			],
		});

		const { result } = renderHook(() => useWorkflowDetail("wf-resubmitted"));

		await waitFor(() =>
			expect(result.current.runEventState.run?.id).toBe("run-resubmitted"),
		);

		expect(mockGetRunByWorkflowName).toHaveBeenCalledWith("wf-resubmitted");
		expect(result.current.runEventState.error).toBeNull();
		expect(result.current.runEventState.items).toHaveLength(1);
	});
});
