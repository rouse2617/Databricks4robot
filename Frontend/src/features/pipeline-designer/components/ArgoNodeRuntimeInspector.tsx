import { Descriptions, Tag, Typography } from "antd";
import type { WorkflowNodeStatus } from "../../../api/workflowApi";
import {
	formatWorkflowPhaseLabel,
	resolveStatusTagColor,
} from "../../../lib/statusLabels";
import { toArgoNodeRuntimeView } from "../model/runtime-model";

export interface ArgoNodeRuntimeInspectorProps {
	node: WorkflowNodeStatus;
}

export function ArgoNodeRuntimeInspector({
	node,
}: ArgoNodeRuntimeInspectorProps) {
	const view = toArgoNodeRuntimeView(node);

	return (
		<Descriptions
			size="small"
			column={1}
			bordered
			title="Argo 节点运行时"
			styles={{ label: { width: 140 } }}
		>
			<Descriptions.Item label="Argo Node ID">
				<Typography.Text copyable={{ text: view.argoNodeId }}>
					{view.argoNodeId}
				</Typography.Text>
			</Descriptions.Item>
			<Descriptions.Item label="阶段">
				<Tag color={resolveStatusTagColor(view.phase)}>
					{formatWorkflowPhaseLabel(view.phase)}
				</Tag>
			</Descriptions.Item>
			<Descriptions.Item label="消息">{view.message || "—"}</Descriptions.Item>
			<Descriptions.Item label="Pod">{view.podName || "—"}</Descriptions.Item>
			<Descriptions.Item label="模板">
				{view.templateName || "—"}
			</Descriptions.Item>
			<Descriptions.Item label="开始时间">
				{view.startedAt || "—"}
			</Descriptions.Item>
			<Descriptions.Item label="结束时间">
				{view.finishedAt || "—"}
			</Descriptions.Item>
			<Descriptions.Item label="进度">{view.progress || "—"}</Descriptions.Item>
			<Descriptions.Item label="子节点">
				{view.children && view.children.length > 0 ? (
					<Typography.Text>{view.children.join(", ")}</Typography.Text>
				) : (
					"—"
				)}
			</Descriptions.Item>
		</Descriptions>
	);
}
