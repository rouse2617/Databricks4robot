import type { Edge, Node, XYPosition } from "@xyflow/react";

/** Adapter boundary so the designer can migrate canvas engines without rewriting orchestration. */
export interface CanvasEngineAdapter {
	addNode: (node: Node) => void;
	updateNodeData: (id: string, data: Record<string, unknown>) => void;
	selectElements: (ids: string[]) => void;
	deselectAll: () => void;
	selectAll: () => void;
	deleteSelection: () => void;
	copySelection: () => Promise<void>;
	paste: () => Promise<void>;
	screenToFlowPosition: (position: { x: number; y: number }) => XYPosition;
	zoomIn: () => void;
	zoomOut: () => void;
	fitView: () => void;
}

export type { Edge, Node, XYPosition };
