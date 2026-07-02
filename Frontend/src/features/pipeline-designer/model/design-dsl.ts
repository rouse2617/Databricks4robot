import type {
	Argument,
	Pipeline,
	PipelineEdgeDef,
	PipelineNodeDef,
	Port,
} from "../../../components/pipeline/types";

/** Product-facing pipeline design document (persisted / saved). */
export type PipelineDesignDSL = Pipeline;

export type PipelineDesignNode = PipelineNodeDef;
export type PipelineDesignEdge = PipelineEdgeDef;

export interface ComponentRef {
	componentId?: string;
	releaseId?: string;
	componentVersionLabel?: string;
}

export interface PipelineDesignMeta {
	name: string;
	version?: string;
}

export type { Argument, Port };
