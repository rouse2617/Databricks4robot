import { describe, expect, it } from "vitest";
import type { PipelineTemplate } from "../../api/pipelineApi";
import {
	dedupeTemplatesByName,
	prepareDeployments,
	sortByCreatedDesc,
} from "./deployPanelUtils";

const template = (
	overrides: Partial<PipelineTemplate> = {},
): PipelineTemplate => ({
	id: "id-1",
	name: "my-pipeline",
	version: 1,
	pipeline: { name: "my-pipeline", version: "1", nodes: [], edges: [] },
	nodeCount: 1,
	createdAt: "2026-05-27T12:00:00Z",
	...overrides,
});

describe("deployPanelUtils", () => {
	it("sorts items by createdAt descending", () => {
		const sorted = sortByCreatedDesc([
			template({ id: "a", createdAt: "2026-05-27T10:00:00Z" }),
			template({ id: "b", createdAt: "2026-05-28T10:00:00Z" }),
		]);
		expect(sorted.map((item) => item.id)).toEqual(["b", "a"]);
	});

	it("dedupes templates by name keeping the newest", () => {
		const deduped = dedupeTemplatesByName([
			template({ id: "old", version: 1, createdAt: "2026-05-27T10:00:00Z" }),
			template({ id: "new", version: 2, createdAt: "2026-05-28T10:00:00Z" }),
			template({
				id: "other",
				name: "other-pipeline",
				createdAt: "2026-05-29T10:00:00Z",
			}),
		]);
		expect(deduped).toHaveLength(2);
		expect(deduped.find((item) => item.name === "my-pipeline")?.id).toBe("new");
	});

	it("prepares deployments in reverse chronological order", () => {
		const sorted = prepareDeployments([
			{
				id: "d1",
				pipelineName: "a",
				workflowName: "wf-a",
				status: "Failed",
				nodeCount: 1,
				createdAt: "2026-05-27T10:00:00Z",
			},
			{
				id: "d2",
				pipelineName: "b",
				workflowName: "wf-b",
				status: "Succeeded",
				nodeCount: 2,
				createdAt: "2026-05-28T10:00:00Z",
			},
		]);
		expect(sorted.map((item) => item.id)).toEqual(["d2", "d1"]);
	});
});
