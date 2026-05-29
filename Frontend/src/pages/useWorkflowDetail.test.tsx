// @vitest-environment jsdom

import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../api/pipelineClient";

const mockGetWorkflow = vi.fn();

vi.mock("../api/workflowApi", () => ({
	getWorkflow: (...args: unknown[]) => mockGetWorkflow(...args),
	getWorkflowLogs: vi.fn(),
	getWorkflowLogStreamUrl: vi.fn(() => "/api/v1/workflows/wf/logs/stream"),
}));

import { useWorkflowDetail } from "./useWorkflowDetail";

describe("useWorkflowDetail", () => {
	beforeEach(() => {
		vi.clearAllMocks();
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
});
