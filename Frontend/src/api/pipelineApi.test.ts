import { afterEach, describe, expect, it, vi } from "vitest";
import { deployTemplate } from "./pipelineApi";

describe("pipelineApi.deployTemplate", () => {
	afterEach(() => {
		vi.restoreAllMocks();
	});

	it("creates a first-class pipeline run with explicit empty asset ids", async () => {
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
				{ headers: { "content-type": "application/json" }, status: 200 },
			),
		);

		await deployTemplate("tmpl-001", undefined, "default");

		expect(fetchMock).toHaveBeenCalledWith(
			"/api/v1/pipeline-runs/template/tmpl-001",
			expect.objectContaining({
				method: "POST",
				body: JSON.stringify({ asset_ids: [], target_id: "default" }),
			}),
		);
	});
});
