import type { Node } from "@ant-design/pro-flow";
import { Input } from "antd";
import { useEffect, useState } from "react";
import type { PipelineNodeData } from "./types";

interface Props {
	node: Node<PipelineNodeData>;
	onUpdate: (id: string, data: Record<string, unknown>) => void;
}

export function NodeConfigPanel({ node, onUpdate }: Props) {
	const [commandText, setCommandText] = useState("");

	useEffect(() => {
		setCommandText(JSON.stringify(node.data.command ?? []));
	}, [node.data.command]);

	return (
		<>
			<div className="config-panel-header">
				<h3>节点配置</h3>
			</div>
			<div className="config-content">
				<div className="config-field">
					<label htmlFor="ncc-name">名称</label>
					<Input
						id="ncc-name"
						value={node.data.label ?? ""}
						onChange={(e) => onUpdate(node.id, { label: e.target.value })}
					/>
				</div>
				<div className="config-field">
					<label htmlFor="ncc-image">镜像</label>
					<Input
						id="ncc-image"
						value={node.data.image ?? ""}
						onChange={(e) => onUpdate(node.id, { image: e.target.value })}
					/>
				</div>
				<div className="config-field">
					<label htmlFor="ncc-command">命令 (JSON)</label>
					<Input
						id="ncc-command"
						value={commandText}
						onChange={(e) => setCommandText(e.target.value)}
						onBlur={() => {
							try {
								const parsed = JSON.parse(commandText);
								if (Array.isArray(parsed)) {
									onUpdate(node.id, { command: parsed });
								}
							} catch {
								/* ignore */
							}
						}}
					/>
				</div>
				<div className="config-section-title">资源配额</div>
				<div className="config-field">
					<label htmlFor="ncc-cpu">CPU</label>
					<Input
						id="ncc-cpu"
						value={node.data.cpu ?? ""}
						onChange={(e) => onUpdate(node.id, { cpu: e.target.value })}
						placeholder="500m"
					/>
				</div>
				<div className="config-field">
					<label htmlFor="ncc-memory">内存</label>
					<Input
						id="ncc-memory"
						value={node.data.memory ?? ""}
						onChange={(e) => onUpdate(node.id, { memory: e.target.value })}
						placeholder="256Mi"
					/>
				</div>
				<div className="config-field">
					<label htmlFor="ncc-disk">磁盘</label>
					<Input
						id="ncc-disk"
						value={node.data.disk ?? ""}
						onChange={(e) => onUpdate(node.id, { disk: e.target.value })}
						placeholder="1Gi"
					/>
				</div>
			</div>
		</>
	);
}
