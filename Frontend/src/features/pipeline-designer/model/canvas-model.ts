import type { Edge, Node } from "@xyflow/react";
import type { PipelineNodeData } from "../../../components/pipeline/types";

/** React Flow presentation model for the pipeline designer canvas. */
export type PipelineCanvasNode = Node<PipelineNodeData>;
export type PipelineCanvasEdge = Edge;

export type CanvasSelectionState = {
	selectedNodeId: string | null;
	editingNodeId: string | null;
};

export type CanvasMenuState = {
	open: boolean;
	x: number;
	y: number;
	nodeId: string | null;
};

export interface CanvasState {
	nodes: PipelineCanvasNode[];
	edges: PipelineCanvasEdge[];
	pipelineName: string;
	selection: CanvasSelectionState;
	contextMenu: CanvasMenuState;
	jsonOutput: string | null;
	importModalOpen: boolean;
	importText: string;
}
