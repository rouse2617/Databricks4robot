import {
	DeleteOutlined,
	PauseCircleFilled,
	PauseCircleOutlined,
	PlayCircleOutlined,
	RedoOutlined,
	ReloadOutlined,
	StopOutlined,
} from "@ant-design/icons";
import type { ButtonProps, MenuProps } from "antd";
import { createElement, type ReactNode } from "react";
import {
	deleteWorkflow,
	getWorkflow,
	resubmitWorkflow,
	resumeWorkflow,
	retryWorkflow,
	stopWorkflow,
	suspendWorkflow,
	terminateWorkflow,
	type WorkflowDetail,
	type WorkflowNodeStatus,
	type WorkflowOperationResponse,
	type WorkflowSummary,
} from "../api/workflowApi";

export type WorkflowOperationKey =
	| "stop"
	| "retry"
	| "resume"
	| "suspend"
	| "terminate"
	| "resubmit"
	| "delete";

type WorkflowLike = Pick<WorkflowSummary | WorkflowDetail, "name" | "status"> & {
	message?: string;
	nodes?: Array<Pick<WorkflowNodeStatus, "phase" | "type">>;
};

export interface WorkflowOperationDefinition {
	title: string;
	icon: ReactNode;
	phases: readonly string[];
	danger?: boolean;
	action: (workflowName: string) => Promise<WorkflowOperationResponse>;
}

export interface WorkflowOperationConfig extends ButtonProps {
	key: WorkflowOperationKey;
	title: string;
	disabled: boolean;
	run: () => Promise<WorkflowOperationResponse>;
}

export const WORKFLOW_OPERATIONS: Record<
	WorkflowOperationKey,
	WorkflowOperationDefinition
> = {
	stop: {
		title: "停止",
		icon: createElement(PauseCircleOutlined),
		phases: ["Running", "Pending"],
		action: stopWorkflow,
	},
	retry: {
		title: "重试",
		icon: createElement(ReloadOutlined),
		phases: ["Failed", "Error"],
		action: retryWorkflow,
	},
	resume: {
		title: "恢复",
		icon: createElement(PlayCircleOutlined),
		phases: ["Suspended"],
		action: resumeWorkflow,
	},
	suspend: {
		title: "暂停",
		icon: createElement(PauseCircleFilled),
		phases: ["Running"],
		action: suspendWorkflow,
	},
	terminate: {
		title: "终止",
		icon: createElement(StopOutlined),
		phases: ["Running", "Pending"],
		danger: true,
		action: terminateWorkflow,
	},
	resubmit: {
		title: "重提交",
		icon: createElement(RedoOutlined),
		phases: ["Succeeded", "Failed", "Error"],
		action: resubmitWorkflow,
	},
	delete: {
		title: "删除",
		icon: createElement(DeleteOutlined),
		phases: ["*"],
		danger: true,
		action: deleteWorkflow,
	},
};

export const WORKFLOW_OPERATION_ORDER: WorkflowOperationKey[] = [
	"stop",
	"retry",
	"resume",
	"suspend",
	"terminate",
	"resubmit",
	"delete",
];

export function isWorkflowStopped(workflow?: WorkflowLike | null): boolean {
	const message = workflow?.message?.trim().toLowerCase() ?? "";
	return message.includes("stopped");
}

function isRetryableTaskNode(
	node: Pick<WorkflowNodeStatus, "phase" | "type">,
): boolean {
	const type = (node.type || "").toLowerCase();
	if (
		type === "dag" ||
		type === "steps" ||
		type === "stepgroup" ||
		type === "retry" ||
		type === "skipped"
	) {
		return false;
	}
	return node.phase === "Failed" || node.phase === "Error";
}

export function workflowHasRetryableFailedNodes(
	workflow?: WorkflowLike | null,
): boolean {
	if (!workflow?.nodes?.length) return false;
	return workflow.nodes.some(isRetryableTaskNode);
}

export function isWorkflowRetryEnabled(workflow?: WorkflowLike | null): boolean {
	if (!workflow?.status) return false;
	if (!WORKFLOW_OPERATIONS.retry.phases.includes(workflow.status)) return false;
	if (isWorkflowStopped(workflow)) return false;
	if (workflow.nodes?.length) {
		return workflowHasRetryableFailedNodes(workflow);
	}
	return true;
}

export function workflowShowsRetryProgress(
	before: WorkflowLike,
	after: WorkflowLike,
): boolean {
	if (after.status === "Running" || after.status === "Pending") return true;
	if (before.status !== after.status) return true;
	if (!before.nodes?.length || !after.nodes?.length) return false;
	const countRetryable = (nodes: WorkflowLike["nodes"]) =>
		nodes?.filter(isRetryableTaskNode).length ?? 0;
	const failedBefore = before.nodes.filter(
		(node) =>
			isRetryableTaskNode(node) &&
			(node.phase === "Failed" || node.phase === "Error"),
	).length;
	const failedAfter = after.nodes.filter(
		(node) =>
			isRetryableTaskNode(node) &&
			(node.phase === "Failed" || node.phase === "Error"),
	).length;
	if (failedAfter < failedBefore) return true;
	return countRetryable(after.nodes) > countRetryable(before.nodes);
}

export function getWorkflowOperationConfirmText(
	key: WorkflowOperationKey,
): string | null {
	if (key === "retry") {
		return "将重试工作流中失败或出错的节点，不会从头重新执行。若工作流曾被手动停止，请改用「重提交」。";
	}
	if (key === "resubmit") {
		return "将基于当前工作流模板重新提交一次全新执行。";
	}
	return null;
}

export async function runWorkflowRetryWithFeedback(
	workflow: WorkflowDetail,
	run: () => Promise<WorkflowOperationResponse>,
): Promise<"success" | "no_progress"> {
	const before = workflow;
	await run();
	try {
		const after = await getWorkflow(workflow.name);
		return workflowShowsRetryProgress(before, after) ? "success" : "no_progress";
	} catch {
		return "success";
	}
}

export function isWorkflowOperationEnabled(
	operation: WorkflowOperationDefinition,
	workflow?: WorkflowLike | null,
): boolean {
	if (!workflow?.status) return false;
	if (
		!operation.phases.includes("*") &&
		!operation.phases.includes(workflow.status)
	) {
		return false;
	}
	if (operation === WORKFLOW_OPERATIONS.retry) {
		return isWorkflowRetryEnabled(workflow);
	}
	return true;
}

export function getWorkflowOperationConfigs(
	workflow?: WorkflowLike | null,
	keys = WORKFLOW_OPERATION_ORDER,
): WorkflowOperationConfig[] {
	return keys.map((key) => {
		const operation = WORKFLOW_OPERATIONS[key];
		const disabled = !isWorkflowOperationEnabled(operation, workflow);
		return {
			key,
			title: operation.title,
			children: operation.title,
			icon: operation.icon,
			danger: operation.danger,
			disabled,
			run: () => operation.action(workflow?.name ?? ""),
		};
	});
}

export function getAvailableWorkflowOperationConfigs(
	workflow?: WorkflowLike | null,
	keys = WORKFLOW_OPERATION_ORDER,
): WorkflowOperationConfig[] {
	return getWorkflowOperationConfigs(workflow, keys).filter(
		(operation) => !operation.disabled,
	);
}

export function getWorkflowOperationMenuItems(
	workflow?: WorkflowLike | null,
): NonNullable<MenuProps["items"]> {
	return getAvailableWorkflowOperationConfigs(workflow).map((operation) => ({
		key: operation.key,
		label: operation.title,
		icon: operation.icon,
		danger: operation.danger,
	}));
}
