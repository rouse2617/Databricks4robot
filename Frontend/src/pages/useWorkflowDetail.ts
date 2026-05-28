import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
	getWorkflow,
	getWorkflowLogs,
	getWorkflowLogStreamUrl,
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
	const eventSourceRef = useRef<EventSource | null>(null);

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

	const stopNodeLogStream = useCallback(() => {
		if (eventSourceRef.current) {
			eventSourceRef.current.close();
			eventSourceRef.current = null;
		}
	}, []);

	const fallbackToLogsApi = useCallback(
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

	const loadNodeLogs = useCallback(
		async (nodeId: string) => {
			if (!name) return;
			stopNodeLogStream();
			setLogState((current) => ({
				...current,
				loading: true,
				error: null,
				content: "",
			}));

			if (typeof EventSource === "undefined") {
				await fallbackToLogsApi(nodeId);
				return;
			}

			const stream = new EventSource(getWorkflowLogStreamUrl(name, nodeId));
			eventSourceRef.current = stream;
			let receivedLine = false;

			stream.onmessage = (event) => {
				receivedLine = true;
				setLogState((current) => ({
					...current,
					loading: false,
					error: null,
					content: current.content
						? `${current.content}\n${event.data}`
						: event.data,
				}));
			};

			stream.onerror = () => {
				if (eventSourceRef.current !== stream) {
					return;
				}
				stopNodeLogStream();
				if (!receivedLine) {
					void fallbackToLogsApi(nodeId);
					return;
				}
				setLogState((current) => ({
					...current,
					loading: false,
				}));
			};
		},
		[fallbackToLogsApi, name, stopNodeLogStream],
	);

	const selectNode = useCallback(
		(node: WorkflowNodeStatus | null) => {
			if (!node || !name) {
				stopNodeLogStream();
				setSelectedNodeId(null);
				setLogState((current) => ({
					...EMPTY_LOG_STATE,
					search: current.search,
				}));
				return;
			}
			setSelectedNodeId(node.id);
			setLogState((current) => ({
				...current,
				content: "",
				loading: false,
				error: null,
			}));
		},
		[name, stopNodeLogStream],
	);

	const selectedNode = useMemo(() => {
		if (!workflow || !selectedNodeId) return null;
		return workflow.nodes.find((node) => node.id === selectedNodeId) ?? null;
	}, [workflow, selectedNodeId]);

	useEffect(() => {
		if (!selectedNodeId || !name) {
			stopNodeLogStream();
			return;
		}
		void loadNodeLogs(selectedNodeId);
		return () => {
			stopNodeLogStream();
		};
	}, [loadNodeLogs, name, selectedNodeId, stopNodeLogStream]);

	useEffect(() => {
		return () => {
			stopNodeLogStream();
		};
	}, [stopNodeLogStream]);

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
