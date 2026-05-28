import {
	CheckCircleOutlined,
	ClockCircleOutlined,
	CloseCircleOutlined,
	ExclamationCircleOutlined,
	LoadingOutlined,
} from "@ant-design/icons";
import { createElement, type ReactNode } from "react";

export const WORKFLOW_PHASES = [
	"Running",
	"Succeeded",
	"Failed",
	"Error",
	"Pending",
	"Suspended",
] as const;

export type WorkflowPhase = (typeof WORKFLOW_PHASES)[number] | string;

export const PHASE_COLORS: Record<string, string> = {
	Succeeded: "#16a34a",
	Running: "#2563eb",
	Pending: "#d97706",
	Failed: "#dc2626",
	Error: "#dc2626",
	Skipped: "#6b7280",
	Suspended: "#7c3aed",
};

export const STATUS_COLORS: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "warning",
	Failed: "error",
	Error: "error",
	Skipped: "default",
	Suspended: "purple",
};

export const STATUS_ACCENT_COLORS: Record<string, string> = {
	Succeeded: "#52c41a",
	Running: "#1677ff",
	Pending: "#faad14",
	Failed: "#ff4d4f",
	Error: "#cf1322",
	Suspended: "#722ed1",
};

export const STATUS_ICONS: Record<string, ReactNode> = {
	Succeeded: createElement(CheckCircleOutlined),
	Running: createElement(LoadingOutlined),
	Pending: createElement(ClockCircleOutlined),
	Failed: createElement(CloseCircleOutlined),
	Error: createElement(ExclamationCircleOutlined),
	Suspended: createElement(ClockCircleOutlined),
};
