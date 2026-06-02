import {
	CheckCircleOutlined,
	ClockCircleOutlined,
	CloseCircleOutlined,
	ExclamationCircleOutlined,
	PlayCircleOutlined,
} from "@ant-design/icons";
import { createElement, type ReactNode } from "react";

export const WORKFLOW_PHASES = [
	"Running",
	"Succeeded",
	"Failed",
	"Error",
	"Pending",
	"Suspended",
	"Expired",
] as const;

export const WORKFLOW_PHASE_LABELS: Record<
	(typeof WORKFLOW_PHASES)[number],
	string
> = {
	Running: "运行中",
	Succeeded: "成功",
	Failed: "失败",
	Error: "异常",
	Pending: "等待中",
	Suspended: "已暂停",
	Expired: "已过期",
};

/** Always show these summary cards; hide other phases when count is 0. */
export const WORKFLOW_SUMMARY_ALWAYS_VISIBLE = new Set<
	(typeof WORKFLOW_PHASES)[number]
>(["Running", "Succeeded", "Failed", "Error"]);

export type WorkflowPhase = (typeof WORKFLOW_PHASES)[number] | string;

export const PHASE_COLORS: Record<string, string> = {
	Succeeded: "#16a34a",
	Running: "#2563eb",
	Pending: "#d97706",
	Failed: "#dc2626",
	Error: "#dc2626",
	Skipped: "#6b7280",
	Suspended: "#7c3aed",
	Expired: "#6b7280",
};

export const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "blue",
	Pending: "warning",
	Failed: "error",
	Error: "error",
	Skipped: "default",
	Suspended: "purple",
	Expired: "default",
};

export const STATUS_ACCENT_COLORS: Record<string, string> = {
	Succeeded: "#52c41a",
	Running: "#1677ff",
	Pending: "#faad14",
	Failed: "#ff4d4f",
	Error: "#cf1322",
	Suspended: "#722ed1",
	Expired: "#6b7280",
};

export const STATUS_ICONS: Record<string, ReactNode> = {
	Succeeded: createElement(CheckCircleOutlined),
	Running: createElement(PlayCircleOutlined),
	Pending: createElement(ClockCircleOutlined),
	Failed: createElement(CloseCircleOutlined),
	Error: createElement(ExclamationCircleOutlined),
	Suspended: createElement(ClockCircleOutlined),
	Expired: createElement(ClockCircleOutlined),
};
