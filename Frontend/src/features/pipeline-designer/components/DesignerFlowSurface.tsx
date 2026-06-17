import {
	Background,
	BackgroundVariant,
	ReactFlow,
	type Edge,
	type Node,
	type OnEdgesChange,
	type OnNodesChange,
} from "@xyflow/react";
import type { DragEvent, MouseEvent as ReactMouseEvent } from "react";
import "@xyflow/react/dist/style.css";
import { PipelineStepNode } from "../../../components/pipeline/PipelineNode";
import type { PipelineNodeData } from "../../../components/pipeline/types";

const PIPELINE_NODE_TYPES = {
	pipelineStep: PipelineStepNode,
} as const;

export interface DesignerFlowSurfaceProps {
	nodes: Node<PipelineNodeData>[];
	edges: Edge[];
	onNodesChange: OnNodesChange<Node<PipelineNodeData>>;
	onEdgesChange: OnEdgesChange<Edge>;
	readOnlyMode: boolean;
	onDrop?: (event: DragEvent) => void;
	onDragOver?: (event: DragEvent) => void;
	onNodeClick: (event: ReactMouseEvent, node: Node<PipelineNodeData>) => void;
	onNodeContextMenu: (
		event: ReactMouseEvent,
		node: Node<PipelineNodeData>,
	) => void;
	onPaneClick: () => void;
	onPaneContextMenu: (event: ReactMouseEvent | MouseEvent) => void;
	onError?: (code: string, message: string) => void;
}

export function DesignerFlowSurface({
	nodes,
	edges,
	onNodesChange,
	onEdgesChange,
	readOnlyMode,
	onDrop,
	onDragOver,
	onNodeClick,
	onNodeContextMenu,
	onPaneClick,
	onPaneContextMenu,
	onError,
}: DesignerFlowSurfaceProps) {
	return (
		<div style={{ width: "100%", height: "100%" }}>
			<ReactFlow
				nodes={nodes}
				edges={edges}
				nodeTypes={PIPELINE_NODE_TYPES}
				onNodesChange={onNodesChange}
				onEdgesChange={onEdgesChange}
				nodesDraggable={!readOnlyMode}
				nodesConnectable={!readOnlyMode}
				elementsSelectable
				onDrop={readOnlyMode ? undefined : onDrop}
				onDragOver={readOnlyMode ? undefined : onDragOver}
				onNodeClick={onNodeClick}
				onNodeContextMenu={onNodeContextMenu}
				onPaneClick={onPaneClick}
				onPaneContextMenu={onPaneContextMenu}
				onError={onError}
				onlyRenderVisibleElements
				minZoom={0.2}
				maxZoom={2}
				proOptions={{ hideAttribution: true }}
				style={{ width: "100%", height: "100%" }}
			>
				<Background
					variant={BackgroundVariant.Lines}
					gap={20}
					color="#cbd5e1"
					lineWidth={1}
				/>
			</ReactFlow>
		</div>
	);
}
