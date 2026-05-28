import {
	Background,
	BackgroundVariant,
	Controls,
	MarkerType,
	Position,
	ReactFlow,
	ReactFlowProvider,
	type Edge as RFEdge,
	type Node as RFNode,
	useEdgesState,
	useNodesState,
	useReactFlow,
} from "@xyflow/react";
import dagre from "dagre";
import { type MouseEvent, useCallback, useEffect, useMemo } from "react";
import "@xyflow/react/dist/style.css";
import "./WorkflowDagView.css";
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

export function countDisplayableWorkflowNodes(
	rawNodes: WorkflowNodeStatus[],
): number {
	return rawNodes.filter(isDisplayableNode).length;
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
			draggable: false,
			connectable: false,
			selectable: true,
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
	emptyMessage?: string;
}

function FitViewOnGraphChange({ graphKey }: { graphKey: string }): null {
	const { fitView } = useReactFlow();

	useEffect(() => {
		if (!graphKey) return;
		const frame = requestAnimationFrame(() => {
			void fitView({ padding: 0.3, duration: 200 });
		});
		return () => cancelAnimationFrame(frame);
	}, [graphKey, fitView]);

	return null;
}

function WorkflowDagViewInner({
	nodes: rawNodes,
	workflowEdges,
	selectedNodeId,
	onNodeSelect,
	emptyMessage,
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

	const graphKey = useMemo(() => {
		const displayableNodes = rawNodes.filter(isDisplayableNode);
		if (displayableNodes.length === 0) return "";
		const nodeIds = displayableNodes.map((node) => node.id).sort().join("|");
		const edgeIds = (workflowEdges ?? [])
			.map((edge) => `${edge.source}->${edge.target}`)
			.sort()
			.join("|");
		return `${nodeIds}::${edgeIds}`;
	}, [rawNodes, workflowEdges]);

	const showEmptyState = countDisplayableWorkflowNodes(rawNodes) === 0;

	return (
		<div
			className="workflow-dag-view"
			style={{
				flex: 1,
				width: "100%",
				height: "100%",
				minHeight: 0,
				minWidth: 0,
				position: "relative",
			}}
		>
			<ReactFlow
				nodes={nodes}
				edges={edges}
				onNodesChange={onNodesChange}
				onEdgesChange={onEdgesChange}
				onNodeClick={onNodeClick}
				onPaneClick={onPaneClick}
				nodesDraggable={false}
				nodesConnectable={false}
				elementsSelectable
				minZoom={0.1}
				style={{ width: "100%", height: "100%" }}
			>
				<FitViewOnGraphChange graphKey={graphKey} />
				<Background variant={BackgroundVariant.Dots} gap={24} color="#cbd5e1" />
				<Controls showInteractive={false} />
			</ReactFlow>
			{showEmptyState ? (
				<div
					style={{
						position: "absolute",
						inset: 0,
						display: "flex",
						alignItems: "center",
						justifyContent: "center",
						padding: 24,
						pointerEvents: "none",
					}}
				>
					<div
						style={{
							maxWidth: 560,
							textAlign: "center",
							color: "#475569",
							fontSize: 14,
							lineHeight: 1.6,
							background: "rgba(255,255,255,0.92)",
							border: "1px solid #e2e8f0",
							borderRadius: 8,
							padding: "20px 24px",
							boxShadow: "0 8px 24px rgba(15,23,42,0.08)",
						}}
					>
						<div style={{ fontWeight: 600, marginBottom: 8, color: "#0f172a" }}>
							暂无可展示的 DAG 节点
						</div>
						<div>
							{emptyMessage ||
								"工作流可能在启动前失败，或所有步骤仍处于隐藏/省略状态。"}
						</div>
					</div>
				</div>
			) : null}
		</div>
	);
}

export function WorkflowDagView(props: WorkflowDagViewProps): React.JSX.Element {
	return (
		<ReactFlowProvider>
			<WorkflowDagViewInner {...props} />
		</ReactFlowProvider>
	);
}
