import { useCallback, useEffect, useMemo, useState } from "react";
import {
	getWorkflow,
	getWorkflowLogs,
	type WorkflowDetail,
	type WorkflowNodeStatus,
} from "../api/workflowApi";

interface WorkflowLogState {
	content: string | null;
	loading: boolean;
	error: string | null;
	search: string;
}

interface UseWorkflowDetailResult {
	workflow: WorkflowDetail | null;
	loading: boolean;
	selectedNode: WorkflowNodeStatus | null;
	selectNode: (node: WorkflowNodeStatus | null) => void;
	loadWorkflow: () => void;
	logState: WorkflowLogState;
	setLogSearch: (query: string) => void;
}

const EMPTY_LOG_STATE: WorkflowLogState = {
	content: null,
	loading: false,
	error: null,
	search: "",
};

function toErrorMessage(err: unknown): string {
	if (err instanceof Error) {
		return err.message;
	}
	return String(err);
}

export function useWorkflowDetail(name?: string): UseWorkflowDetailResult {
	const [workflow, setWorkflow] = useState<WorkflowDetail | null>(null);
	const [loading, setLoading] = useState(true);
	const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
	const [logState, setLogState] = useState<WorkflowLogState>(EMPTY_LOG_STATE);

	const loadWorkflow = useCallback(() => {
		if (!name) return;
		setLoading(true);
		getWorkflow(name)
			.then(setWorkflow)
			.catch((err) => {
				console.error(err);
				setWorkflow(null);
			})
			.finally(() => setLoading(false));
	}, [name]);

	useEffect(() => {
		loadWorkflow();
	}, [loadWorkflow]);

	const loadNodeLogs = useCallback(
		async (nodeId: string) => {
			if (!name) return;
			setLogState((current) => ({ ...current, loading: true, error: null }));
			try {
				const res = await getWorkflowLogs(name, nodeId);
				setLogState({
					content: res.logs || "",
					loading: false,
					error: null,
					search: "",
				});
			} catch (err) {
				setLogState({
					content: null,
					loading: false,
					error: toErrorMessage(err),
					search: "",
				});
			}
		},
		[name],
	);

	const selectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			if (!node || !name) {
				setSelectedNodeId(null);
				setLogState((current) => ({
					...EMPTY_LOG_STATE,
					search: current.search,
				}));
				return;
			}
			setSelectedNodeId(node.id);
			setLogState((current) => ({
				...EMPTY_LOG_STATE,
				search: current.search,
			}));
			void loadNodeLogs(node.id);
		},
		[loadNodeLogs, name],
	);

	const selectedNode = useMemo(() => {
		if (!workflow || !selectedNodeId) return null;
		return workflow.nodes.find((node) => node.id === selectedNodeId) ?? null;
	}, [workflow, selectedNodeId]);

	useEffect(() => {
		if (!workflow || !selectedNodeId) return;
		const selectedNodeExists = workflow.nodes.some(
			(node) => node.id === selectedNodeId,
		);
		if (!selectedNodeExists) {
			setSelectedNodeId(null);
			setLogState((current) => ({
				...EMPTY_LOG_STATE,
				search: current.search,
			}));
		}
	}, [selectedNodeId, workflow]);

	return {
		workflow,
		loading,
		selectedNode,
		loadWorkflow,
		selectNode,
		logState: {
			...logState,
		},
		setLogSearch: useCallback((query: string) => {
			setLogState((current) => ({ ...current, search: query }));
		}, []),
	};
}
