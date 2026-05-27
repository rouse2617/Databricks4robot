import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import type { PipelineNodeData } from "./types";

export function PipelineStepNode({ data, selected }: NodeProps<Node<PipelineNodeData>>) {
	return (
		<div className={`pipeline-node ${selected ? "selected" : ""}`}>
			<Handle type="target" position={Position.Left} className="node-handle" />
			<div className="node-header">
				<span className="node-status-dot" />
				<span>{data.label}</span>
			</div>
			<div className="node-body">
				<div className="node-info">{data.image}</div>
			</div>
			<Handle type="source" position={Position.Right} className="node-handle" />
		</div>
	);
}
