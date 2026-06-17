import { type Dispatch, useEffect, useRef } from "react";
import { getPipeline, listPipelineVersions } from "../../../api/pipelineApi";
import type { Pipeline } from "../../../components/pipeline/types";
import type { DesignerAction } from "./designerReducer";

type MessageApi = {
	error: (content: string) => void;
};

interface UsePipelineTemplateLoaderOptions {
	templateId: string | null;
	dispatch: Dispatch<DesignerAction>;
	messageApi: MessageApi;
	loadPipelineToCanvas: (pipeline: Pipeline) => void;
	loadPipelineFromSessionStorage: () => void;
}

export function usePipelineTemplateLoader({
	templateId,
	dispatch,
	messageApi,
	loadPipelineToCanvas,
	loadPipelineFromSessionStorage,
}: UsePipelineTemplateLoaderOptions) {
	const loadPipelineToCanvasRef = useRef(loadPipelineToCanvas);
	loadPipelineToCanvasRef.current = loadPipelineToCanvas;
	const loadPipelineFromSessionStorageRef = useRef(
		loadPipelineFromSessionStorage,
	);
	loadPipelineFromSessionStorageRef.current = loadPipelineFromSessionStorage;

	useEffect(() => {
		if (!templateId) {
			dispatch({ type: "template/clearVersions" });
			loadPipelineFromSessionStorageRef.current();
			return;
		}

		let cancelled = false;
		dispatch({ type: "template/setLoading", loading: true });
		Promise.all([getPipeline(templateId), listPipelineVersions(templateId)])
			.then(([template, versions]) => {
				if (cancelled) return;
				loadPipelineToCanvasRef.current(template.pipeline);
				dispatch({ type: "canvas/setPipelineName", name: template.name });
				dispatch({
					type: "template/setVersions",
					versions,
					selectedVersionId: template.id,
					loadedScope: template.scope ?? null,
				});
			})
			.catch((err) => {
				if (cancelled) return;
				messageApi.error(`模板加载失败: ${String(err)}`);
				dispatch({ type: "template/clearVersions" });
				loadPipelineFromSessionStorageRef.current();
			})
			.finally(() => {
				if (!cancelled) {
					dispatch({ type: "template/setLoading", loading: false });
				}
			});

		return () => {
			cancelled = true;
		};
	}, [dispatch, messageApi, templateId]);
}
