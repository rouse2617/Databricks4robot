// @vitest-environment jsdom

import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { PipelineTemplate } from "../../../api/pipelineApi";
import type { Pipeline } from "../../../components/pipeline/types";
import { usePipelineTemplateLoader } from "./usePipelineTemplateLoader";

const mockGetPipeline = vi.fn();
const mockListPipelineVersions = vi.fn();

vi.mock("../../../api/pipelineApi", () => ({
	getPipeline: (...args: unknown[]) => mockGetPipeline(...args),
	listPipelineVersions: (...args: unknown[]) =>
		mockListPipelineVersions(...args),
}));

const emptyPipeline: Pipeline = { name: "", nodes: [], edges: [] };

function template(overrides: Partial<PipelineTemplate> = {}): PipelineTemplate {
	return {
		id: "tpl-1",
		name: "每日巡检",
		version: 1,
		scope: "dev",
		pipeline: emptyPipeline,
		nodeCount: 0,
		createdAt: "2026-07-01T00:00:00Z",
		...overrides,
	};
}

describe("usePipelineTemplateLoader", () => {
	afterEach(() => {
		vi.clearAllMocks();
	});

	// CYB-3486 回归:模板展示名(template.name)常与内嵌 pipeline.name 不同(后者甚至
	// 为空)。若在 markCanvasClean 之后再单独 setPipelineName,基线 name 与当前 name
	// 不符,刚打开模板未做任何修改就被判为 dirty → 离开时"未保存"误报。
	it("passes the display name into loadPipelineToCanvas and does not re-set the name after (no false dirty)", async () => {
		const tpl = template();
		mockGetPipeline.mockResolvedValue(tpl);
		mockListPipelineVersions.mockResolvedValue([]);
		const dispatch = vi.fn();
		const loadPipelineToCanvas = vi.fn();

		renderHook(() =>
			usePipelineTemplateLoader({
				templateId: "tpl-1",
				dispatch,
				messageApi: { error: vi.fn() },
				loadPipelineToCanvas,
				loadPipelineFromSessionStorage: vi.fn(),
			}),
		);

		await waitFor(() => {
			expect(loadPipelineToCanvas).toHaveBeenCalledWith(
				tpl.pipeline,
				"每日巡检",
			);
		});

		// 名字由 loadPipelineToCanvas 内部随基线一起落定;loader 不得再单独派发。
		expect(dispatch).not.toHaveBeenCalledWith({
			type: "canvas/setPipelineName",
			name: "每日巡检",
		});
	});

	it("loads from session storage and clears versions when there is no templateId", () => {
		const dispatch = vi.fn();
		const loadPipelineFromSessionStorage = vi.fn();

		renderHook(() =>
			usePipelineTemplateLoader({
				templateId: null,
				dispatch,
				messageApi: { error: vi.fn() },
				loadPipelineToCanvas: vi.fn(),
				loadPipelineFromSessionStorage,
			}),
		);

		expect(dispatch).toHaveBeenCalledWith({ type: "template/clearVersions" });
		expect(loadPipelineFromSessionStorage).toHaveBeenCalledTimes(1);
		expect(mockGetPipeline).not.toHaveBeenCalled();
	});
});
