import { afterEach, describe, expect, it, vi } from "vitest";
import { createRunByTemplate, listRuns, retryRun } from "./runApi";

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
});
