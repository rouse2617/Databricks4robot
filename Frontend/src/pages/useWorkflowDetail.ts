import { useCallback, useEffect, useMemo, useState } from "react";
import { ApiError } from "../api/pipelineClient";
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

export type WorkflowLoadErrorKind = "not_found" | "error";

export interface WorkflowLoadError {
	kind: WorkflowLoadErrorKind;
	message: string;
}

interface UseWorkflowDetailResult {
	workflow: WorkflowDetail | null;
	loading: boolean;
	loadError: WorkflowLoadError | null;
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

const ACTIVE_WORKFLOW_STATUSES = new Set(["Running", "Pending"]);
const WORKFLOW_POLL_INTERVAL_MS = 8_000;

function toErrorMessage(err: unknown): string {
	if (err instanceof Error) {
		return err.message;
	}
	return String(err);
}

function toLoadError(err: unknown): WorkflowLoadError {
	if (err instanceof ApiError && err.status === 404) {
		return { kind: "not_found", message: err.message };
	}
	return { kind: "error", message: toErrorMessage(err) };
}

export function useWorkflowDetail(name?: string): UseWorkflowDetailResult {
	const [workflow, setWorkflow] = useState<WorkflowDetail | null>(null);
	const [loading, setLoading] = useState(true);
	const [loadError, setLoadError] = useState<WorkflowLoadError | null>(null);
	const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
	const [logState, setLogState] = useState<WorkflowLogState>(EMPTY_LOG_STATE);

	const loadWorkflow = useCallback(() => {
		if (!name) return;
		setLoading(true);
		setLoadError(null);
		getWorkflow(name)
			.then((detail) => {
				setWorkflow(detail);
				setLoadError(null);
			})
			.catch((err) => {
				console.error(err);
				setWorkflow(null);
				setLoadError(toLoadError(err));
			})
			.finally(() => setLoading(false));
	}, [name]);

	useEffect(() => {
		loadWorkflow();
	}, [loadWorkflow]);

	useEffect(() => {
		if (!name || !workflow || !ACTIVE_WORKFLOW_STATUSES.has(workflow.status)) {
			return;
		}

		const timer = window.setInterval(() => {
			getWorkflow(name)
				.then((detail) => {
					setWorkflow(detail);
					setLoadError(null);
				})
				.catch((err) => {
					console.error(err);
				});
		}, WORKFLOW_POLL_INTERVAL_MS);

		return () => window.clearInterval(timer);
	}, [name, workflow?.status, workflow]);

	const loadNodeLogs = useCallback(
		async (nodeId: string) => {
			if (!name) return;
			setLogState((current) => ({
				...current,
				loading: true,
				error: null,
			}));
			try {
				const res = await getWorkflowLogs(name, nodeId);
				setLogState((current) => ({
					content: res.logs || "",
					loading: false,
					error: null,
					search: current.search,
				}));
			} catch (err) {
				setLogState((current) => ({
					content: null,
					loading: false,
					error: toErrorMessage(err),
					search: current.search,
				}));
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
			if (node.id !== selectedNodeId) {
				setLogState((current) => ({
					...current,
					content: "",
					loading: false,
					error: null,
				}));
			}
		},
		[name, selectedNodeId],
	);

	const selectedNode = useMemo(() => {
		if (!workflow || !selectedNodeId) return null;
		return workflow.nodes.find((node) => node.id === selectedNodeId) ?? null;
	}, [workflow, selectedNodeId]);

	useEffect(() => {
		if (!selectedNodeId || !name) {
			return;
		}
		void loadNodeLogs(selectedNodeId);
	}, [loadNodeLogs, name, selectedNodeId]);

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
		loadError,
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
