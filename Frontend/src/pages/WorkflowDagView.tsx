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
import { Input } from "antd";
import dagre from "dagre";
import {
	type MouseEvent,
	useCallback,
	useEffect,
	useMemo,
	useState,
} from "react";
import "@xyflow/react/dist/style.css";
import "./WorkflowDagView.css";
import type { WorkflowDagEdge, WorkflowNodeStatus } from "../api/workflowApi";
import { getWorkflowNodeDisplayText } from "../lib/workflowNodeDisplay";
import { WorkflowDagNode, type WorkflowDagNodeData } from "./WorkflowDagNode";

const DISPLAYABLE_NODE_TYPES = new Set(["pod", "template"]);
const DAG_NODE_WIDTH = 240;
const DAG_NODE_HEIGHT = 92;
const DAG_RANK_DIR = "LR" as const;
const DAG_NODE_GAP = 72;
const DAG_FIT_MIN_ZOOM = 0.72;
const DAG_FIT_MAX_ZOOM = 1;

const nodeTypes = { workflowStep: WorkflowDagNode };

function getProgressPercent(progress: string | undefined): number | null {
	if (!progress) return null;
	const trimmed = progress.trim();
	if (!trimmed) return null;
	const parts = trimmed.split("/");
	if (parts.length !== 2) return null;
	const done = Number(parts[0]);
	const total = Number(parts[1]);
	if (!Number.isFinite(done) || !Number.isFinite(total) || total <= 0)
		return null;
	return Math.min(100, Math.max(0, Math.round((done / total) * 100)));
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

export function countDisplayableWorkflowNodes(
	rawNodes: WorkflowNodeStatus[],
): number {
	return rawNodes.filter(isDisplayableNode).length;
}

export function filterDisplayableWorkflowNodes(
	rawNodes: WorkflowNodeStatus[],
): WorkflowNodeStatus[] {
	return rawNodes.filter(isDisplayableNode);
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

interface WorkflowDagNodeType extends RFNode<WorkflowDagNodeData> {
	type: "workflowStep";
}

export function buildDagElements(
	rawNodes: WorkflowNodeStatus[],
	workflowEdges: WorkflowDagEdge[] | undefined,
	selectedNodeId: string | null,
	nodeSearch: string,
): {
	nodes: WorkflowDagNodeType[];
	edges: RFEdge[];
} {
	const displayableNodes = rawNodes.filter(isDisplayableNode);
	const normalizedSearch = nodeSearch.trim().toLowerCase();
	const visibleNodes = displayableNodes.filter((node) => {
		if (normalizedSearch.length === 0) return true;
		const displayText = getWorkflowNodeDisplayText(node).toLowerCase();
		return (
			displayText.includes(normalizedSearch) ||
			node.name.toLowerCase().includes(normalizedSearch)
		);
	});
	const visibleIds = new Set(visibleNodes.map((node) => node.id));
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
			type: "smoothstep",
			style: { stroke: "#64748b", strokeWidth: 1.5 },
			markerEnd: {
				type: MarkerType.ArrowClosed,
				color: "#64748b",
				width: 14,
				height: 14,
			},
		});
	};

	if (workflowEdges) {
		for (const edge of workflowEdges) {
			if (!visibleIds.has(edge.source) || !visibleIds.has(edge.target)) {
				continue;
			}
			addEdge(edge.source, edge.target);
		}
	} else {
		for (const sourceNode of rawNodes) {
			const sourceId = getNearestVisibleAncestorId(
				sourceNode,
				byName,
				visibleIds,
			);
			for (const childId of sourceNode.children ?? []) {
				const childNode = byId.get(childId);
				if (!childNode || !visibleIds.has(childId) || !sourceId) continue;
				addEdge(sourceId, childId);
			}
		}

		for (const targetNode of visibleNodes) {
			const sourceId = getNearestVisibleAncestorId(
				targetNode,
				byName,
				visibleIds,
			);
			if (sourceId && sourceId !== targetNode.id) {
				addEdge(sourceId, targetNode.id);
			}
		}
	}

	const graph = new dagre.graphlib.Graph();
	graph.setGraph({
		rankdir: DAG_RANK_DIR,
		nodesep: DAG_NODE_GAP,
		ranksep: 80,
		marginx: 20,
		marginy: 20,
	});
	graph.setDefaultEdgeLabel(() => ({}));

	for (const node of visibleNodes) {
		graph.setNode(node.id, { width: DAG_NODE_WIDTH, height: DAG_NODE_HEIGHT });
	}
	for (const edge of edges) {
		graph.setEdge(edge.source, edge.target);
	}
	dagre.layout(graph);
	const nodes = visibleNodes.map((node, index) => {
		const dagreNode = graph.node(node.id);
		const hasLayout =
			dagreNode &&
			typeof dagreNode.x === "number" &&
			typeof dagreNode.y === "number";
		const layoutX = hasLayout
			? dagreNode.x
			: index * (DAG_NODE_WIDTH + DAG_NODE_GAP);
		const layoutY = hasLayout ? dagreNode.y : 0;
		const isSelected = node.id === selectedNodeId;
		const displayText = getWorkflowNodeDisplayText(node).toLowerCase();
		const matchesSearch =
			normalizedSearch.length === 0 ||
			displayText.includes(normalizedSearch) ||
			node.name.toLowerCase().includes(normalizedSearch);
		const dimmed = !matchesSearch;
		const progressPercent = getProgressPercent(node.progress);
		return {
			id: node.id,
			type: "workflowStep" as const,
			width: DAG_NODE_WIDTH,
			height: DAG_NODE_HEIGHT,
			draggable: false,
			connectable: false,
			selectable: true,
			position: {
				x: layoutX - DAG_NODE_WIDTH / 2,
				y: layoutY - DAG_NODE_HEIGHT / 2,
			},
			data: {
				workflowNode: node,
				selected: isSelected,
				dimmed,
				progressPercent,
			},
			sourcePosition: Position.Right,
			targetPosition: Position.Left,
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
	workflowStatus?: string;
}

function FitViewOnGraphChange({ graphKey }: { graphKey: string }): null {
	const { fitView } = useReactFlow();

	useEffect(() => {
		if (!graphKey) return;
		const frame = requestAnimationFrame(() => {
			void fitView({
				padding: 0.18,
				minZoom: DAG_FIT_MIN_ZOOM,
				maxZoom: DAG_FIT_MAX_ZOOM,
				duration: 200,
			});
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
	workflowStatus,
}: WorkflowDagViewProps): React.JSX.Element {
	const [nodes, setNodes, onNodesChange] = useNodesState<WorkflowDagNodeType>(
		[],
	);
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
		(_: MouseEvent, node: WorkflowDagNodeType) => {
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
		const nodeIds = displayableNodes
			.map((node) => node.id)
			.sort()
			.join("|");
		const edgeIds = (workflowEdges ?? [])
			.map((edge) => `${edge.source}->${edge.target}`)
			.sort()
			.join("|");
		return `${nodeIds}::${edgeIds}::${nodeSearch}`;
	}, [rawNodes, workflowEdges, nodeSearch]);

	const displayableCount = countDisplayableWorkflowNodes(rawNodes);
	const showEmptyState = displayableCount === 0;
	const failedNodes = rawNodes.filter((node) =>
		["Failed", "Error"].includes(node.phase),
	);
	const isFailedWorkflow = ["Failed", "Error"].includes(workflowStatus ?? "");

	return (
		<div className="workflow-dag-view">
			<div className="workflow-dag-view__toolbar">
				<Input.Search
					placeholder="搜索节点..."
					allowClear
					value={nodeSearch}
					onChange={(event) => setNodeSearch(event.target.value)}
					className="workflow-dag-view__search"
				/>
				<span className="workflow-dag-view__meta">
					{displayableCount} 个步骤
				</span>
			</div>
			<div className="workflow-dag-view__canvas">
				<ReactFlow
					nodes={nodes}
					edges={edges}
					nodeTypes={nodeTypes}
					onNodesChange={onNodesChange}
					onEdgesChange={onEdgesChange}
					onNodeClick={onNodeClick}
					onPaneClick={onPaneClick}
					nodesDraggable={false}
					nodesConnectable={false}
					elementsSelectable
					panOnDrag
					panOnScroll
					zoomOnScroll
					minZoom={0.35}
					maxZoom={1.5}
					proOptions={{ hideAttribution: true }}
				>
					<FitViewOnGraphChange graphKey={graphKey} />
					<Background
						variant={BackgroundVariant.Lines}
						gap={20}
						color="#cbd5e1"
						lineWidth={1}
					/>
					<Controls showInteractive={false} position="bottom-right" />
				</ReactFlow>
				{showEmptyState ? (
					<div className="workflow-dag-view__empty">
						<div
							className={[
								"workflow-dag-view__empty-card",
								isFailedWorkflow ? "workflow-dag-view__empty-card--error" : "",
							]
								.filter(Boolean)
								.join(" ")}
						>
							<div className="workflow-dag-view__empty-title">
								{isFailedWorkflow
									? "工作流失败，暂无可展示 DAG"
									: "暂无可展示的 DAG 节点"}
							</div>
							<div>
								{emptyMessage ||
									"工作流可能在启动前失败，或所有步骤仍处于隐藏/省略状态。"}
							</div>
							{failedNodes.length > 0 ? (
								<div className="workflow-dag-view__empty-actions">
									{failedNodes.slice(0, 3).map((node) => (
										<button
											key={node.id}
											type="button"
											onClick={() => onNodeSelect(node)}
										>
											查看失败节点：{getWorkflowNodeDisplayText(node)}
										</button>
									))}
								</div>
							) : null}
						</div>
					</div>
				) : null}
			</div>
		</div>
	);
}

export function WorkflowDagView(
	props: WorkflowDagViewProps,
): React.JSX.Element {
	return (
		<ReactFlowProvider>
			<WorkflowDagViewInner {...props} />
		</ReactFlowProvider>
	);
}
