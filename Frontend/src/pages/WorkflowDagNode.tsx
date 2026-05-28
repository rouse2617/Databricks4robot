import {
	Handle,
	Position,
	type Node,
	type NodeProps,
} from "@xyflow/react";
import { Tag, Tooltip } from "antd";
import dayjs from "dayjs";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import {
	PHASE_COLORS,
	STATUS_ICONS,
	WORKFLOW_PHASE_LABELS,
} from "../lib/constants";
import { getWorkflowNodeDisplayText } from "../lib/workflowNodeDisplay";
import "./WorkflowDagNode.css";

const PROGRESS_RING_SIZE = 18;
const PROGRESS_RING_STROKE = 2.5;

export interface WorkflowDagNodeData extends Record<string, unknown> {
	workflowNode: WorkflowNodeStatus;
	selected: boolean;
	dimmed: boolean;
	progressPercent: number | null;
}

function getProgressRingColor(phase: string): string {
	if (phase === "Succeeded") return PHASE_COLORS.Succeeded;
	if (phase === "Running") return PHASE_COLORS.Running;
	return "#94a3b8";
}

function getNodeRelativeTime(node: WorkflowNodeStatus): string | null {
	const startedAt = node.startedAt;
	if (!startedAt) return null;
	const started = dayjs(startedAt);
	if (!started.isValid()) return null;
	if (node.finishedAt) {
		const finished = dayjs(node.finishedAt);
		if (finished.isValid()) {
			return finished.fromNow();
		}
	}
	return started.fromNow();
}

function ProgressRing({ percent, color }: { percent: number; color: string }) {
	const size = PROGRESS_RING_SIZE;
	const center = size / 2;
	const radius = center - PROGRESS_RING_STROKE / 2;
	const circumference = 2 * Math.PI * radius;
	const dashOffset = circumference - (percent / 100) * circumference;

	return (
		<svg
			className="workflow-dag-node__progress-ring"
			width={size}
			height={size}
			viewBox={`0 0 ${size} ${size}`}
			aria-hidden="true"
		>
			<circle
				cx={center}
				cy={center}
				r={radius}
				fill="none"
				stroke="#e2e8f0"
				strokeWidth={PROGRESS_RING_STROKE}
			/>
			<circle
				cx={center}
				cy={center}
				r={radius}
				fill="none"
				stroke={color}
				strokeWidth={PROGRESS_RING_STROKE}
				strokeDasharray={`${circumference} ${circumference}`}
				strokeDashoffset={dashOffset}
				transform={`rotate(-90 ${center} ${center})`}
				strokeLinecap="round"
			/>
		</svg>
	);
}

export function WorkflowDagNode({
	data,
}: NodeProps<Node<WorkflowDagNodeData>>): React.JSX.Element {
	const { workflowNode, selected, dimmed, progressPercent } = data;
	const displayText = getWorkflowNodeDisplayText(workflowNode);
	const phase = workflowNode.phase;
	const accent = PHASE_COLORS[phase] || "#64748b";
	const phaseLabel =
		WORKFLOW_PHASE_LABELS[phase as keyof typeof WORKFLOW_PHASE_LABELS] ||
		phase;
	const relTime = getNodeRelativeTime(workflowNode);
	const isRunning = phase === "Running";

	return (
		<div
			className={[
				"workflow-dag-node",
				selected ? "workflow-dag-node--selected" : "",
				dimmed ? "workflow-dag-node--dimmed" : "",
				isRunning ? "workflow-dag-node--running" : "",
			]
				.filter(Boolean)
				.join(" ")}
			style={{ "--workflow-dag-accent": accent } as React.CSSProperties}
		>
			<Handle
				type="target"
				position={Position.Left}
				className="workflow-dag-node__handle workflow-dag-node__handle--target"
			/>
			<Handle
				type="source"
				position={Position.Right}
				className="workflow-dag-node__handle workflow-dag-node__handle--source"
			/>

			<div className="workflow-dag-node__accent" aria-hidden="true" />

			<div className="workflow-dag-node__body">
				<div className="workflow-dag-node__header">
					<div className="workflow-dag-node__title" title={displayText}>
						{displayText}
					</div>
					{progressPercent !== null ? (
						<ProgressRing
							percent={progressPercent}
							color={getProgressRingColor(phase)}
						/>
					) : null}
				</div>

				<div className="workflow-dag-node__meta">
					<Tag
						className="workflow-dag-node__phase-tag"
						icon={STATUS_ICONS[phase]}
						color={phase === "Pending" ? "default" : undefined}
						style={
							phase !== "Pending"
								? {
										color: accent,
										background: `${accent}14`,
										borderColor: `${accent}33`,
									}
								: undefined
						}
					>
						{phaseLabel}
					</Tag>
					{relTime ? (
						<Tooltip
							title={
								workflowNode.startedAt
									? dayjs(workflowNode.startedAt).format(
											"YYYY-MM-DD HH:mm:ss",
										)
									: ""
							}
						>
							<span className="workflow-dag-node__time">{relTime}</span>
						</Tooltip>
					) : null}
				</div>
			</div>
		</div>
	);
}
