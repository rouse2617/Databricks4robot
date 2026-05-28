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
	resubmitWorkflow,
	resumeWorkflow,
	retryWorkflow,
	stopWorkflow,
	suspendWorkflow,
	terminateWorkflow,
	type WorkflowDetail,
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

type WorkflowLike = Pick<WorkflowSummary | WorkflowDetail, "name" | "status">;

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

export function isWorkflowOperationEnabled(
	operation: WorkflowOperationDefinition,
	workflow?: WorkflowLike | null,
): boolean {
	if (!workflow?.status) return false;
	return (
		operation.phases.includes("*") || operation.phases.includes(workflow.status)
	);
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
