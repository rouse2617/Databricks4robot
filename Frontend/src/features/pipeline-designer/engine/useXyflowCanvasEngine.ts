import {
	type Edge,
	type Node,
	useReactFlow,
} from "@xyflow/react";
import { useCallback, useMemo, useRef, type Dispatch, type SetStateAction } from "react";
import { createPipelineNodeId } from "../model/node-id";
import type { CanvasEngineAdapter } from "./canvasEngineAdapter";

const CLIPBOARD_PREFIX = "databrew-pipeline-canvas:";

type ClipboardPayload = {
	nodes: Node[];
	edges: Edge[];
};

function parseClipboardPayload(text: string): ClipboardPayload | null {
	const raw = text.startsWith(CLIPBOARD_PREFIX)
		? text.slice(CLIPBOARD_PREFIX.length)
		: text;
	try {
		const parsed = JSON.parse(raw) as ClipboardPayload;
		if (!Array.isArray(parsed.nodes) || !Array.isArray(parsed.edges)) {
			return null;
		}
		return parsed;
	} catch {
		return null;
	}
}

export function useXyflowCanvasEngine<T extends Node>(
	nodes: T[],
	edges: Edge[],
	setNodes: Dispatch<SetStateAction<T[]>>,
	setEdges: Dispatch<SetStateAction<Edge[]>>,
): CanvasEngineAdapter {
	const { screenToFlowPosition, zoomIn, zoomOut, fitView } = useReactFlow();
	const clipboardRef = useRef<ClipboardPayload | null>(null);

	const addNode = useCallback(
		(node: Node) => {
			setNodes((current) => [...current, node as T]);
		},
		[setNodes],
	);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			setNodes((current) =>
				current.map((node) =>
					node.id === id
						? { ...node, data: { ...node.data, ...data } }
						: node,
				),
			);
		},
		[setNodes],
	);

	const selectElements = useCallback(
		(ids: string[]) => {
			const idSet = new Set(ids);
			setNodes((current) =>
				current.map((node) => ({ ...node, selected: idSet.has(node.id) })),
			);
			setEdges((current) =>
				current.map((edge) => ({ ...edge, selected: idSet.has(edge.id) })),
			);
		},
		[setEdges, setNodes],
	);

	const deselectAll = useCallback(() => {
		setNodes((current) =>
			current.map((node) => ({ ...node, selected: false })),
		);
		setEdges((current) =>
			current.map((edge) => ({ ...edge, selected: false })),
		);
	}, [setEdges, setNodes]);

	const selectAll = useCallback(() => {
		setNodes((current) => current.map((node) => ({ ...node, selected: true })));
		setEdges((current) => current.map((edge) => ({ ...edge, selected: true })));
	}, [setEdges, setNodes]);

	const deleteSelection = useCallback(() => {
		setNodes((currentNodes) => {
			const removedIds = new Set(
				currentNodes.filter((node) => node.selected).map((node) => node.id),
			);
			setEdges((currentEdges) =>
				currentEdges.filter(
					(edge) =>
						!edge.selected &&
						!removedIds.has(edge.source) &&
						!removedIds.has(edge.target),
				),
			);
			return currentNodes.filter((node) => !node.selected);
		});
	}, [setEdges, setNodes]);

	const copySelection = useCallback(async () => {
		const selectedNodes = nodes.filter((node) => node.selected);
		if (selectedNodes.length === 0) return;
		const selectedNodeIds = new Set(selectedNodes.map((node) => node.id));
		const selectedEdges = edges.filter(
			(edge) =>
				selectedNodeIds.has(edge.source) && selectedNodeIds.has(edge.target),
		);
		const payload: ClipboardPayload = {
			nodes: selectedNodes,
			edges: selectedEdges,
		};
		clipboardRef.current = payload;
		const serialized = `${CLIPBOARD_PREFIX}${JSON.stringify(payload)}`;
		if (navigator.clipboard?.writeText) {
			await navigator.clipboard.writeText(serialized);
		}
	}, [edges, nodes]);

	const paste = useCallback(async () => {
		let payload = clipboardRef.current;
		if (navigator.clipboard?.readText) {
			try {
				const fromClipboard = parseClipboardPayload(
					await navigator.clipboard.readText(),
				);
				if (fromClipboard) payload = fromClipboard;
			} catch {
				/* fall back to in-memory clipboard */
			}
		}
		if (!payload || payload.nodes.length === 0) return;

		const idMap = new Map<string, string>();
		for (const node of payload.nodes) {
			idMap.set(node.id, createPipelineNodeId());
		}

		const pastedNodes = payload.nodes.map((node) => ({
			...node,
			id: idMap.get(node.id) ?? createPipelineNodeId(),
			position: {
				x: node.position.x + 40,
				y: node.position.y + 40,
			},
			selected: true,
		}));

		const pastedEdges = payload.edges.map((edge) => {
			const source = idMap.get(edge.source) ?? edge.source;
			const target = idMap.get(edge.target) ?? edge.target;
			const handle = edge.targetHandle ?? "input";
			return {
				...edge,
				id: `${source}-${target}-${handle}-${createPipelineNodeId()}`,
				source,
				target,
				selected: false,
			};
		});

		deselectAll();
		setNodes((current) => [
			...current.map((node) => ({ ...node, selected: false })),
			...(pastedNodes as T[]),
		]);
		setEdges((current) => [...current, ...pastedEdges]);
		clipboardRef.current = { nodes: pastedNodes, edges: pastedEdges };
	}, [deselectAll, setEdges, setNodes]);

	const fitViewPadded = useCallback(() => fitView({ padding: 0.2 }), [fitView]);

	return useMemo(
		() => ({
			addNode,
			updateNodeData,
			selectElements,
			deselectAll,
			selectAll,
			deleteSelection,
			copySelection,
			paste,
			screenToFlowPosition,
			zoomIn,
			zoomOut,
			fitView: fitViewPadded,
		}),
		[
			addNode,
			copySelection,
			deleteSelection,
			deselectAll,
			fitViewPadded,
			paste,
			screenToFlowPosition,
			selectAll,
			selectElements,
			updateNodeData,
			zoomIn,
			zoomOut,
		],
	);
}
