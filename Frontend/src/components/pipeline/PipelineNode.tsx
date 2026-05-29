import {
	Handle,
	type NodeProps,
	Position,
	SelectType,
} from "@ant-design/pro-flow";
import { memo } from "react";
import type { PipelineNodeData } from "./types";

function PipelineStepNodeInner({
	data,
	selected,
}: NodeProps<PipelineNodeData>) {
	const selectType = selected
		? (data.selectType as SelectType | undefined) || SelectType.SELECT
		: SelectType.DEFAULT;
	const label = data.label || "未命名步骤";
	return (
		<div
			className={`pipeline-node select-${selectType} ${selected ? "selected" : ""}`}
			role="option"
			tabIndex={0}
			aria-label={`流水线节点 ${label}`}
			aria-selected={selected}
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

export const PipelineStepNode = memo(PipelineStepNodeInner);
