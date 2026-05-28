import type { KeyboardEvent } from "react";
import type { WorkflowNodeStatus } from "../api/workflowApi";
import { PHASE_COLORS } from "../lib/constants";
import { getWorkflowNodeDisplayText } from "../lib/workflowNodeDisplay";
import { filterDisplayableWorkflowNodes } from "./WorkflowDagView";
import "./WorkflowTimelineView.css";

function getStepSortKey(name: string): number {
	const match = name.match(/step-(\d+)/i);
	return match ? Number(match[1]) : Number.MAX_SAFE_INTEGER;
}

interface TimelineItem {
	node: WorkflowNodeStatus;
	start: number;
	end: number;
}

interface WorkflowTimelineViewProps {
	nodes: WorkflowNodeStatus[];
	selectedNodeId: string | null;
	onNodeSelect: (node: WorkflowNodeStatus | null) => void;
}

export function WorkflowTimelineView({
	nodes,
	selectedNodeId,
	onNodeSelect,
}: WorkflowTimelineViewProps): React.JSX.Element {
	const items: TimelineItem[] = filterDisplayableWorkflowNodes(nodes)
		.filter((node) => node.startedAt)
		.sort((a, b) => {
			const startA = new Date(a.startedAt as string).getTime();
			const startB = new Date(b.startedAt as string).getTime();
			if (startA !== startB) return startA - startB;
			return (
				getStepSortKey(getWorkflowNodeDisplayText(a)) -
				getStepSortKey(getWorkflowNodeDisplayText(b))
			);
		})
		.map((node) => ({
			node,
			start: new Date(node.startedAt as string).getTime(),
			end: node.finishedAt
				? new Date(node.finishedAt).getTime()
				: Date.now(),
		}));

	if (items.length === 0) {
		return (
			<div className="workflow-timeline-view">
				<div className="workflow-timeline-view__empty">
					暂无节点时间数据
				</div>
			</div>
		);
	}

	const globalStart = Math.min(...items.map((item) => item.start));
	const globalEnd = Math.max(...items.map((item) => item.end));
	const range = globalEnd - globalStart || 1;
	const ticks = [0, 0.25, 0.5, 0.75, 1];

	const handleRowKeyDown = (
		event: KeyboardEvent<HTMLButtonElement>,
		node: WorkflowNodeStatus,
	) => {
		if (event.key === "Enter" || event.key === " ") {
			event.preventDefault();
			onNodeSelect(node);
		}
	};

	const rangeSec = Math.max(1, Math.round(range / 1000));

	return (
		<div className="workflow-timeline-view">
			<div className="workflow-timeline-view__inner">
				<div className="workflow-timeline-view__header">
					<div className="workflow-timeline-view__header-label">
						<span>Step</span>
						<span className="workflow-timeline-view__duration">
							{rangeSec}s total
						</span>
					</div>
					<div className="workflow-timeline-view__header-track">
						{ticks.map((pct) => (
							<span
								key={pct}
								className="workflow-timeline-view__tick"
								style={{ left: `${pct * 100}%` }}
							>
								{new Date(globalStart + range * pct).toLocaleTimeString()}
							</span>
						))}
					</div>
				</div>

				<div className="workflow-timeline-view__rows">
					<div className="workflow-timeline-view__grid" aria-hidden="true">
						{ticks.map((pct) => (
							<span
								key={pct}
								className="workflow-timeline-view__grid-line"
								style={{ left: `${pct * 100}%` }}
							/>
						))}
					</div>
					{items.map((item) => {
						const { node } = item;
						const left = ((item.start - globalStart) / range) * 100;
						const width = ((item.end - item.start) / range) * 100;
						const durationSec = Math.max(
							1,
							Math.round((item.end - item.start) / 1000),
						);
						const isSelected = node.id === selectedNodeId;
						const barColor = PHASE_COLORS[node.phase] || "#64748b";

						return (
							<button
								type="button"
								key={node.id}
								className={`workflow-timeline-view__row${isSelected ? " workflow-timeline-view__row--selected" : ""}`}
								onClick={() => onNodeSelect(node)}
								onKeyDown={(event) => handleRowKeyDown(event, node)}
								aria-pressed={isSelected}
								title={`${getWorkflowNodeDisplayText(node)} · ${node.phase} · ${durationSec}s`}
							>
								<div className="workflow-timeline-view__label">
									{getWorkflowNodeDisplayText(node)}
								</div>
								<div className="workflow-timeline-view__track">
									<div
										className="workflow-timeline-view__bar"
										style={{
											left: `${left}%`,
											width: `${Math.max(width, 0.5)}%`,
											background: barColor,
										}}
									>
										{durationSec}s
									</div>
								</div>
							</button>
						);
					})}
				</div>
			</div>
		</div>
	);
}
