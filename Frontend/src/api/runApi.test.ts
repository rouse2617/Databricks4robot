import { afterEach, describe, expect, it, vi } from "vitest";
import {
	createRunByTemplate,
	listRunChildren,
	listRuns,
	retryRun,
} from "./runApi";

describe("runApi", () => {
	afterEach(() => {
		vi.restoreAllMocks();
	});

	it("lists product runs through /runs", async () => {
		const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
			new Response(JSON.stringify({ items: [], total: 0 }), {
				headers: { "content-type": "application/json" },
				status: 200,
			}),
		);

		await listRuns({
			view: "summary",
			excludeBatch: true,
			status: "Running",
			page: 2,
			pageSize: 50,
		});

		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/runs?view=summary&excludeBatch=true&status=Running&page=2&pageSize=50",
			expect.objectContaining({ method: "GET" }),
		);
	});

	it("creates runs by template through /runs/template", async () => {
		const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
			new Response(
				JSON.stringify({
					id: "run-1",
					pipelineName: "pipeline",
					workflowName: "wf-1",
					status: "Running",
					nodeCount: 1,
					createdAt: "2026-06-02T00:00:00Z",
				}),
				{ headers: { "content-type": "application/json" }, status: 201 },
			),
		);

		await createRunByTemplate("tmpl-001", undefined, "gpu-l4", 3);

		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/runs/template/tmpl-001",
			expect.objectContaining({
				method: "POST",
				body: JSON.stringify({
					asset_ids: [],
					target_id: "gpu-l4",
					version: 3,
				}),
			}),
		);
	});

	it("retries a run in place through /runs/:id/retry", async () => {
		const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
			new Response(
				JSON.stringify({
					id: "run-1",
					workflowName: "wf-1",
					status: "Failed",
					nodeCount: 1,
					createdAt: "2026-06-02T00:00:00Z",
				}),
				{ headers: { "content-type": "application/json" }, status: 200 },
			),
		);

		await retryRun("run-1");

		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/runs/run-1/retry",
			expect.objectContaining({ method: "POST" }),
		);
	});

	it("reads enriched run children through /runs/:id/children", async () => {
		const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
			new Response(
				JSON.stringify({
					runId: "batch-1",
					items: [{ id: "run-1", status: "Running" }],
					relations: [
						{
							id: "batch-1:run-1:batch_child",
							parentRunId: "batch-1",
							childRunId: "run-1",
							relationType: "batch_child",
							source: "pipeline_runs.batch_job_id",
						},
					],
					summary: {
						total: 1,
						statuses: { Running: 1 },
						aggregateStatus: "Running",
						activeCount: 1,
						terminalCount: 0,
						succeededCount: 0,
						failedCount: 0,
						cancelledCount: 0,
						pendingCount: 0,
						runningCount: 1,
						suspendedCount: 0,
					},
					total: 1,
				}),
				{ headers: { "content-type": "application/json" }, status: 200 },
			),
		);

		const result = await listRunChildren("batch-1");

		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/runs/batch-1/children",
			expect.objectContaining({ method: "GET" }),
		);
		expect(result.summary.aggregateStatus).toBe("Running");
		expect(result.relations?.[0]?.relationType).toBe("batch_child");
	});
});
