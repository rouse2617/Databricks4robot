import { Input } from "antd";
import type { Node } from "@xyflow/react";
import type { PipelineNodeData } from "./types";

interface Props {
	node: Node<PipelineNodeData>;
	onUpdate: (id: string, data: Record<string, unknown>) => void;
}

export function NodeConfigPanel({ node, onUpdate }: Props) {
	return (
		<>
			<div className="config-panel-header">
				<h3>节点配置</h3>
			</div>
			<div className="config-content">
				<div className="config-field">
					<label>名称</label>
					<Input
						value={node.data.label ?? ""}
						onChange={(e) => onUpdate(node.id, { label: e.target.value })}
					/>
				</div>
				<div className="config-field">
					<label>镜像</label>
					<Input
						value={node.data.image ?? ""}
						onChange={(e) => onUpdate(node.id, { image: e.target.value })}
					/>
				</div>
				<div className="config-field">
					<label>命令 (JSON)</label>
					<Input
						value={JSON.stringify(node.data.command ?? [])}
						onChange={(e) => {
							try {
								onUpdate(node.id, { command: JSON.parse(e.target.value) });
							} catch {
								/* ignore */
							}
						}}
					/>
				</div>
				<div className="config-section-title">资源配额</div>
				<div className="config-field">
					<label>CPU</label>
					<Input
						value={node.data.cpu ?? ""}
						onChange={(e) => onUpdate(node.id, { cpu: e.target.value })}
						placeholder="500m"
					/>
				</div>
				<div className="config-field">
					<label>内存</label>
					<Input
						value={node.data.memory ?? ""}
						onChange={(e) => onUpdate(node.id, { memory: e.target.value })}
						placeholder="256Mi"
					/>
				</div>
				<div className="config-field">
					<label>磁盘</label>
					<Input
						value={node.data.disk ?? ""}
						onChange={(e) => onUpdate(node.id, { disk: e.target.value })}
						placeholder="1Gi"
					/>
				</div>
			</div>
		</>
	);
}
