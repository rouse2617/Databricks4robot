import { Handle, type Node, type NodeProps, Position } from "@xyflow/react";
import { memo } from "react";
import type { PipelineNodeData, Port } from "./types";

const DEFAULT_INPUTS: Port[] = [{ name: "input", type: "asset" }];
const DEFAULT_OUTPUTS: Port[] = [{ name: "output", type: "asset" }];

function normalizePorts(ports: Port[] | undefined, fallback: Port[]) {
	return ports?.length ? ports.filter((port) => port.name?.trim()) : fallback;
}

function handleTop(index: number, total: number) {
	if (total <= 1) return "50%";
	return `${Math.round(((index + 1) / (total + 1)) * 100)}%`;
}

function PipelineStepNodeInner({
	data,
	selected,
}: NodeProps<Node<PipelineNodeData>>) {
	const label = data.label || "未命名步骤";
	const versionHint = data.componentVersionLabel
		? ` · ${data.componentVersionLabel}`
		: "";
	const runtimeConfigLabel =
		data.runtimeConfig?.displayName ||
		data.runtimeConfig?.fileName ||
		data.runtimeConfig?.targetFilename;
	const inputPorts = normalizePorts(data.inputPorts, DEFAULT_INPUTS);
	const outputPorts = normalizePorts(data.outputPorts, DEFAULT_OUTPUTS);
	return (
		<div
			className={`pipeline-node select-${selected ? "select" : "default"} ${selected ? "selected" : ""}`}
			role="option"
			tabIndex={0}
			aria-label={`流水线节点 ${label}`}
			aria-selected={selected}
		>
			{inputPorts.map((port, index) => (
				<Handle
					key={`in-${port.name}`}
					id={port.name}
					type="target"
					position={Position.Left}
					className="node-handle node-handle--input"
					style={{ top: handleTop(index, inputPorts.length) }}
				/>
			))}
			<div className="node-header">
				<span className="node-status-dot" />
				<span>
					{label}
					{versionHint}
				</span>
			</div>
			<div className="node-body">
				<div className="node-info">{data.image}</div>
				{runtimeConfigLabel ? (
					<div className="node-config-chip" title={runtimeConfigLabel}>
						配置 {runtimeConfigLabel}
					</div>
				) : null}
				<div className="node-ports">
					<div className="node-port-list">
						<span className="node-port-list__label">IN</span>
						{inputPorts.map((port) => (
							<span key={port.name} className="node-port-chip">
								{port.name}
							</span>
						))}
					</div>
					<div className="node-port-list node-port-list--right">
						<span className="node-port-list__label">OUT</span>
						{outputPorts.map((port) => (
							<span key={port.name} className="node-port-chip">
								{port.name}
							</span>
						))}
					</div>
				</div>
			</div>
			{outputPorts.map((port, index) => (
				<Handle
					key={`out-${port.name}`}
					id={port.name}
					type="source"
					position={Position.Right}
					className="node-handle node-handle--output"
					style={{ top: handleTop(index, outputPorts.length) }}
				/>
			))}
		</div>
	);
}

export const PipelineStepNode = memo(PipelineStepNodeInner);
