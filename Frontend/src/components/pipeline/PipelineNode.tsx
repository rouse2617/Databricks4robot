import {
	Handle,
	type NodeProps,
	Position,
	SelectType,
} from "@ant-design/pro-flow";
import type { PipelineNodeData } from "./types";

export function PipelineStepNode({
	data,
	selected,
}: NodeProps<PipelineNodeData>) {
	const selectType = selected
		? (data.selectType as SelectType | undefined) || SelectType.SELECT
		: SelectType.DEFAULT;
	return (
		<div
			className={`pipeline-node select-${selectType} ${selected ? "selected" : ""}`}
		>
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
