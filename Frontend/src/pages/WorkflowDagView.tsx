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
import dagre from "dagre";
import { type MouseEvent, useCallback, useEffect } from "react";
import type { WorkflowDagEdge, WorkflowNodeStatus } from "../api/workflowApi";

const DISPLAYABLE_NODE_TYPES = new Set(["pod", "template"]);
const DAG_NODE_WIDTH = 220;
const DAG_NODE_HEIGHT = 86;

const PHASE_COLORS: Record<string, string> = {
	Running: "#2563eb",
	Succeeded: "#16a34a",
	Failed: "#dc2626",
	Pending: "#6b7280",
	Error: "#dc2626",
	Skipped: "#9ca3af",
	Suspended: "#7c3aed",
};

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

	const nodes = displayableNodes.map((node) => {
		const position = graph.node(node.id);
		const isSelected = node.id === selectedNodeId;
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
						}}
					>
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
						</div>
					</div>
				),
			},
			style: {
				width: DAG_NODE_WIDTH,
				padding: 10,
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

	useEffect(() => {
		const { nodes: nextNodes, edges: nextEdges } = buildDagElements(
			rawNodes,
			workflowEdges,
			selectedNodeId,
		);
		setNodes(nextNodes);
		setEdges(nextEdges);
	}, [rawNodes, workflowEdges, selectedNodeId, setNodes, setEdges]);

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
			<Background variant={BackgroundVariant.Dots} gap={24} color="#cbd5e1" />
			<Controls />
		</ReactFlow>
	);
}
