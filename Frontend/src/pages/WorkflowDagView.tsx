import {
	Background,
	BackgroundVariant,
	Controls,
	MarkerType,
	Position,
	ReactFlow,
	type Edge as RFEdge,
	type Node as RFNode,
	useEdgesState,
	useNodesState,
} from "@xyflow/react";
import { Input, Tooltip } from "antd";
import dagre from "dagre";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import { type MouseEvent, useCallback, useEffect, useState } from "react";
import type { WorkflowDagEdge, WorkflowNodeStatus } from "../api/workflowApi";

dayjs.extend(relativeTime);

const DISPLAYABLE_NODE_TYPES = new Set(["pod", "template"]);
const DAG_NODE_WIDTH = 220;
const DAG_NODE_HEIGHT = 86;
const PROGRESS_RING_SIZE = 20;
const PROGRESS_RING_STROKE = 3;

const PHASE_COLORS: Record<string, string> = {
	Running: "#2563eb",
	Succeeded: "#16a34a",
	Failed: "#dc2626",
	Pending: "#6b7280",
	Error: "#dc2626",
	Skipped: "#9ca3af",
	Suspended: "#7c3aed",
};

function getProgressPercent(progress: string | undefined): number | null {
	if (!progress) return null;
	const trimmed = progress.trim();
	if (!trimmed) return null;
	const parts = trimmed.split("/");
	if (parts.length !== 2) return null;
	const done = Number(parts[0]);
	const total = Number(parts[1]);
	if (!Number.isFinite(done) || !Number.isFinite(total) || total <= 0) return null;
	return Math.min(100, Math.max(0, Math.round((done / total) * 100)));
}

function getProgressRingColor(phase: string): string {
	if (phase === "Succeeded") return PHASE_COLORS.Succeeded;
	if (phase === "Running") return PHASE_COLORS.Running;
	return "#9ca3af";
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
			width={size}
			height={size}
			viewBox={`0 0 ${size} ${size}`}
			style={{ position: "absolute", right: 4, top: 4 }}
		>
			<circle
				cx={center}
				cy={center}
				r={radius}
				fill="none"
				stroke="rgba(255, 255, 255, 0.35)"
				strokeWidth={PROGRESS_RING_STROKE}
				opacity={0.8}
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

function isDisplayableNode(node: WorkflowNodeStatus): boolean {
	const type = (node.type ?? "").toLowerCase();
	const phase = node.phase;
	const parentPath = node.id.split(".").slice(0, -1).join(".");
	const isRootDagNode =
		node.type === "" &&
		!!node.name &&
		(node.name === node.id || (!!parentPath && node.name === parentPath));
	const hasMeaningfulPhase =
		phase === "Skipped" ||
		phase === "Omitted" ||
		DISPLAYABLE_NODE_TYPES.has(type);

	return !!hasMeaningfulPhase && !isRootDagNode;
}

function getNodeDisplayText(node: WorkflowNodeStatus): string {
	return node.displayName || node.templateName || node.name;
}

function getNearestVisibleAncestorId(
	node: WorkflowNodeStatus,
	nodesByName: Map<string, WorkflowNodeStatus>,
	displayableIds: Set<string>,
): string | null {
	let current: WorkflowNodeStatus | undefined = node;
	while (current) {
		if (displayableIds.has(current.id)) {
			return current.id;
		}
		const parts = current.name.split(".");
		if (parts.length <= 1) {
			break;
		}
		current = nodesByName.get(parts.slice(0, -1).join("."));
	}
	return null;
}

interface WorkflowDagNodeData extends Record<string, unknown> {
	workflowNode: WorkflowNodeStatus;
}

type WorkflowDagNode = RFNode<WorkflowDagNodeData>;

export function buildDagElements(
	rawNodes: WorkflowNodeStatus[],
	workflowEdges: WorkflowDagEdge[] | undefined,
	selectedNodeId: string | null,
	nodeSearch: string,
): {
	nodes: WorkflowDagNode[];
	edges: RFEdge[];
} {
	const displayableNodes = rawNodes.filter(isDisplayableNode);
	const displayableIds = new Set(displayableNodes.map((node) => node.id));
	const byId = new Map(rawNodes.map((node) => [node.id, node]));
	const byName = new Map(rawNodes.map((node) => [node.name, node]));

	const edges: RFEdge[] = [];
	const seen = new Set<string>();

	const addEdge = (source: string, target: string) => {
		if (!source || !target || source === target) return;
		const edgeId = `e-${source}-${target}`;
		if (seen.has(edgeId)) return;
		seen.add(edgeId);
		edges.push({
			id: edgeId,
			source,
			target,
			style: { stroke: "#94a3b8" },
			markerEnd: {
				type: MarkerType.ArrowClosed,
				color: "#94a3b8",
				width: 12,
				height: 12,
			},
		});
	};

	if (workflowEdges) {
		for (const edge of workflowEdges) {
			if (
				!displayableIds.has(edge.source) ||
				!displayableIds.has(edge.target)
			) {
				continue;
			}
			addEdge(edge.source, edge.target);
		}
	} else {
		for (const sourceNode of rawNodes) {
			const sourceId = getNearestVisibleAncestorId(
				sourceNode,
				byName,
				displayableIds,
			);
			for (const childId of sourceNode.children ?? []) {
				const childNode = byId.get(childId);
				if (!childNode || !displayableIds.has(childId) || !sourceId) continue;
				addEdge(sourceId, childId);
			}
		}

		for (const targetNode of displayableNodes) {
			const sourceId = getNearestVisibleAncestorId(
				targetNode,
				byName,
				displayableIds,
			);
			if (sourceId && sourceId !== targetNode.id) {
				addEdge(sourceId, targetNode.id);
			}
		}
	}

	const graph = new dagre.graphlib.Graph();
	graph.setGraph({
		rankdir: "TB",
		nodesep: 70,
		ranksep: 80,
		marginx: 20,
		marginy: 20,
	});
	graph.setDefaultEdgeLabel(() => ({}));

	for (const node of displayableNodes) {
		graph.setNode(node.id, { width: DAG_NODE_WIDTH, height: DAG_NODE_HEIGHT });
	}
	for (const edge of edges) {
		graph.setEdge(edge.source, edge.target);
	}
	dagre.layout(graph);

	const normalizedSearch = nodeSearch.trim().toLowerCase();
	const nodes = displayableNodes.map((node) => {
		const position = graph.node(node.id);
		const isSelected = node.id === selectedNodeId;
		const displayText = getNodeDisplayText(node).toLowerCase();
		const matchesSearch =
			normalizedSearch.length === 0 ||
			displayText.includes(normalizedSearch) ||
			node.name.toLowerCase().includes(normalizedSearch);
		const progressPercent = getProgressPercent(node.progress);
		return {
			id: node.id,
			type: "default",
			position: {
				x: (position?.x ?? 0) - DAG_NODE_WIDTH / 2,
				y: (position?.y ?? 0) - DAG_NODE_HEIGHT / 2,
			},
			data: {
				workflowNode: node,
				label: (
					<div
						style={{
							textAlign: "center",
							whiteSpace: "nowrap",
							overflow: "hidden",
							textOverflow: "ellipsis",
							lineHeight: 1.25,
							position: "relative",
						}}
					>
						{progressPercent !== null && (
							<ProgressRing
								percent={progressPercent}
								color={getProgressRingColor(node.phase)}
							/>
						)}
						<div
							style={{
								fontWeight: 700,
								overflow: "hidden",
								textOverflow: "ellipsis",
							}}
							title={getNodeDisplayText(node)}
						>
							{getNodeDisplayText(node)}
						</div>
						<div style={{ fontSize: 11, opacity: 0.85, marginTop: 4 }}>
							{node.phase}
							{(() => {
								const relTime = getNodeRelativeTime(node);
								return relTime ? (
									<>
										{" "}
										<Tooltip title={node.startedAt ? dayjs(node.startedAt).toLocaleString() : ""}>
											<span style={{ opacity: 0.75 }}>{relTime}</span>
										</Tooltip>
									</>
								) : null;
							})()}
						</div>
					</div>
				),
			},
			style: {
				width: DAG_NODE_WIDTH,
				padding: 10,
				opacity: matchesSearch ? 1 : 0.2,
				borderRadius: 8,
				border: isSelected ? "2px solid #111827" : "1px solid #d1d5db",
				background: PHASE_COLORS[node.phase] || "#9ca3af",
				color: "#ffffff",
				fontSize: 12,
				fontWeight: 600,
				minWidth: DAG_NODE_WIDTH,
			},
			sourcePosition: Position.Bottom,
			targetPosition: Position.Top,
			};
		});

	return { nodes, edges };
}

interface WorkflowDagViewProps {
	nodes: WorkflowNodeStatus[];
	workflowEdges?: WorkflowDagEdge[];
	selectedNodeId: string | null;
	onNodeSelect: (node: WorkflowNodeStatus | null) => void;
}

export function WorkflowDagView({
	nodes: rawNodes,
	workflowEdges,
	selectedNodeId,
	onNodeSelect,
}: WorkflowDagViewProps): React.JSX.Element {
	const [nodes, setNodes, onNodesChange] = useNodesState<WorkflowDagNode>([]);
	const [edges, setEdges, onEdgesChange] = useEdgesState<RFEdge>([]);
	const [nodeSearch, setNodeSearch] = useState("");

	useEffect(() => {
		const { nodes: nextNodes, edges: nextEdges } = buildDagElements(
			rawNodes,
			workflowEdges,
			selectedNodeId,
			nodeSearch,
		);
		setNodes(nextNodes);
		setEdges(nextEdges);
	}, [rawNodes, workflowEdges, selectedNodeId, nodeSearch, setNodes, setEdges]);

	const onNodeClick = useCallback(
		(_: MouseEvent, node: WorkflowDagNode) => {
			const workflowNode = node.data.workflowNode;
			if (workflowNode) {
				onNodeSelect(workflowNode);
			}
		},
		[onNodeSelect],
	);

	const onPaneClick = useCallback(() => {
		onNodeSelect(null);
	}, [onNodeSelect]);

	return (
		<div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
			<div style={{ padding: 8, borderBottom: "1px solid #e5e7eb" }}>
				<Input.Search
					placeholder="搜索节点..."
					allowClear
					value={nodeSearch}
					onChange={(event) => setNodeSearch(event.target.value)}
					style={{ width: 280, maxWidth: "100%" }}
				/>
			</div>
			<div style={{ flex: 1 }}>
				<ReactFlow
					nodes={nodes}
					edges={edges}
					onNodesChange={onNodesChange}
					onEdgesChange={onEdgesChange}
					onNodeClick={onNodeClick}
					onPaneClick={onPaneClick}
					fitView
					minZoom={0.1}
				>
					<Background
						variant={BackgroundVariant.Dots}
						gap={24}
						color="#cbd5e1"
					/>
					<Controls />
				</ReactFlow>
			</div>
		</div>
	);
}
