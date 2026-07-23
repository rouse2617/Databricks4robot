import {
	ColumnWidthOutlined,
	DeleteOutlined,
	ExportOutlined,
	ImportOutlined,
	PlayCircleOutlined,
	SaveOutlined,
	SwapOutlined,
} from "@ant-design/icons";
import {
	applyEdgeChanges,
	type Connection,
	type EdgeChange,
	useEdgesState,
	useNodesState,
} from "@xyflow/react";
import {
	Alert,
	App,
	Button,
	Collapse,
	Input,
	Menu,
	Modal,
	Segmented,
	Select,
	Space,
	Spin,
	Tag,
	Tooltip,
	Typography,
} from "antd";

const { TextArea } = Input;

import {
	type DragEvent,
	type FocusEvent,
	type MutableRefObject,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { assetsApi } from "../../api/assets";
import { BATCH_ASSET_THRESHOLD } from "../../api/deployPipelineRun";
import {
	type ElasticQuota,
	type ExecutionTarget,
	listElasticQuotas,
	savePipeline,
} from "../../api/pipelineApi";
import ErrorBoundary from "../../components/ErrorBoundary";
import AssetPicker, {
	type AssetPickerHandle,
} from "../../components/pipeline/AssetPicker";
import { ComponentPalette } from "../../components/pipeline/ComponentPalette";
import { NodeConfigPanel } from "../../components/pipeline/NodeConfigPanel";
import { PipelineEmptyState } from "../../components/pipeline/PipelineEmptyState";
import {
	matchPoolEq,
	poolAvailability,
	poolAvailabilityLabel,
	poolFreeSummary,
} from "../../components/pipeline/poolUsage";
import type {
	Pipeline,
	PipelineNodeData,
	RegisteredComponent,
} from "../../components/pipeline/types";
import { usePipelineComponents } from "../../hooks/usePipelineComponents";
import { usePipelineKeyboardShortcuts } from "../../hooks/usePipelineKeyboardShortcuts";
import { computeDagreLayout } from "../../lib/autoLayout";
import { MAX_BATCH_ASSET_COUNT } from "../../lib/batchAssetLimits";
import {
	DATA_EDGE_STYLE,
	DEPENDENCY_EDGE_STYLE,
	dependencyEdgeData,
	isDependencyEdge,
} from "../../lib/pipeline-design/edge-format";
import {
	defaultOutputPorts,
	normalizePorts,
} from "../../lib/pipeline-design/port-normalizer";
import {
	fromTranspilerPipeline,
	toTranspilerPipeline,
} from "../../lib/pipelineContract";
import {
	getPipelineExample,
	PIPELINE_EXAMPLES,
	type PipelineExample,
} from "../../lib/pipelineExamples";
import {
	validatePipelineForRun,
	validatePipelineForSave,
} from "../../lib/pipelineValidation";
import {
	PRIORITY_TIER_OPTIONS,
	priorityBadge,
} from "../../lib/workflowPriority";
import {
	apiToRegistered,
	createPipelineNode,
	defaultDeployWorkflowName,
	extractAssetSizeBytes,
	extractNodeAssetIds,
	extractPipelineAssetIds,
	formatBytes,
	type PipelineFlowEdge,
	type PipelineFlowNode,
	parseAssetIds,
	parseAssetName,
	parseAssetType,
	releaseToRegistered,
} from "../../pages/pipeline/pipelinePageHelpers";
import { DesignerFlowSurface } from "./components/DesignerFlowSurface";
import { useXyflowCanvasEngine } from "./engine/useXyflowCanvasEngine";
import {
	initialDesignerState,
	useDesignerReducer,
} from "./hooks/useDesignerReducer";
import { usePipelineDeploy } from "./hooks/usePipelineDeploy";
import { usePipelineTemplateLoader } from "./hooks/usePipelineTemplateLoader";

import "../../styles/pipeline.css";

const DEFAULT_TARGET_PORT = "input";
const DEFAULT_SOURCE_PORT = "output";
const DEFAULT_NODE_WIDTH = 180;
const DEFAULT_NODE_X_GAP = 80;
const DEFAULT_NODE_Y_GAP = 160;
const DEFAULT_NODE_LEFT_PADDING = 80;
const DEFAULT_NODE_TOP_PADDING = 120;
type EdgeConnectionMode = "data" | "dependency";

function keepAllRegisteredComponents(components: RegisteredComponent[]) {
	return components;
}

function defaultNodeScreenPosition(
	bounds: DOMRect | undefined,
	nodeCount: number,
) {
	const stepX = DEFAULT_NODE_WIDTH + DEFAULT_NODE_X_GAP;
	const columnCount = bounds
		? Math.max(
				1,
				Math.floor((bounds.width - DEFAULT_NODE_LEFT_PADDING) / stepX),
			)
		: 3;
	const column = nodeCount % columnCount;
	const row = Math.floor(nodeCount / columnCount);
	if (!bounds) {
		return {
			x: 360 + column * stepX,
			y: 240 + row * DEFAULT_NODE_Y_GAP,
		};
	}
	return {
		x: bounds.left + DEFAULT_NODE_LEFT_PADDING + column * stepX,
		y: bounds.top + DEFAULT_NODE_TOP_PADDING + row * DEFAULT_NODE_Y_GAP,
	};
}

function findDuplicateTargetInput(edges: PipelineFlowEdge[]) {
	const seen = new Map<string, PipelineFlowEdge>();
	for (const edge of edges) {
		if (isDependencyEdge(edge)) continue;
		if (!edge.target) continue;
		const targetPort = edge.targetHandle || DEFAULT_TARGET_PORT;
		const key = `${edge.target}.${targetPort}`;
		const existing = seen.get(key);
		if (existing && existing.id !== edge.id) {
			return { key, existing, incoming: edge };
		}
		seen.set(key, edge);
	}
	return null;
}

function nodeDataDeclaresOutput(
	data: PipelineNodeData | undefined,
	portName: string,
) {
	if (!data) return false;
	const safe = portName.replace(/\./g, "-");
	// Mirror canvas-to-dsl: an unset outputPorts still yields the default
	// "output" port, so reuse the same normalization the deploy validator sees.
	return normalizePorts(data.outputPorts, defaultOutputPorts).some(
		(port) => port.name === portName || port.name.replace(/\./g, "-") === safe,
	);
}

function nodeDataWritesOutputPath(
	data: PipelineNodeData | undefined,
	outputName: string,
) {
	if (!data) return false;
	const path = `/tmp/outputs/${outputName}`;
	if (data.source?.includes(path)) return true;
	if (data.command?.some((part) => part.includes(path))) return true;
	if (
		data.args?.some(
			(arg) => arg.value?.includes(path) || arg.name?.includes(path),
		)
	)
		return true;
	return false;
}

function replaceAppendedValue(previous: string, next: string) {
	if (!previous || next === previous) return next;
	if (next.startsWith(previous) && next.length > previous.length) {
		return next.slice(previous.length);
	}
	return next;
}

function AssetRunSummary({
	assetIds,
	onClear,
}: {
	assetIds: string[];
	onClear: () => void;
}) {
	if (assetIds.length === 0) {
		return (
			<Alert
				type="warning"
				showIcon
				message="无资产运行"
				description="本次运行不会注入资产环境变量，适合调试不依赖资产输入的流水线。"
			/>
		);
	}

	const visibleIds = assetIds.slice(0, 8);
	const hiddenCount = Math.max(assetIds.length - visibleIds.length, 0);

	return (
		<Alert
			type={assetIds.length >= 2 ? "info" : "success"}
			showIcon
			message={
				assetIds.length >= BATCH_ASSET_THRESHOLD
					? `将创建批量任务，共 ${assetIds.length} 个子任务`
					: `将处理 ${assetIds.length} 个资产`
			}
			description={
				<div style={{ display: "grid", gap: 8 }}>
					{assetIds.length >= BATCH_ASSET_THRESHOLD ? (
						<Typography.Text type="secondary" style={{ fontSize: 12 }}>
							提交后跳转批次详情，可在「执行记录 → 批量任务」查看进度；单次最多{" "}
							{MAX_BATCH_ASSET_COUNT.toLocaleString()} 个资产。
						</Typography.Text>
					) : null}
					<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
						{visibleIds.map((assetId) => (
							<Tag key={assetId} color="blue" style={{ marginInlineEnd: 0 }}>
								{assetId}
							</Tag>
						))}
						{hiddenCount > 0 ? <Tag>+{hiddenCount}</Tag> : null}
					</div>
					<Button size="small" onClick={onClear}>
						转为无资产运行
					</Button>
				</div>
			}
		/>
	);
}

function AssetRunContextBanner({
	assetIds,
	isCanvasEmpty,
	onClear,
}: {
	assetIds: string[];
	isCanvasEmpty: boolean;
	onClear: () => void;
}) {
	if (assetIds.length === 0) return null;

	const visibleIds = assetIds.slice(0, 4);
	const hiddenCount = Math.max(assetIds.length - visibleIds.length, 0);
	const message = `已选择 ${assetIds.length} 个资产`;
	const description = isCanvasEmpty
		? "资产已就绪，请添加组件后部署。"
		: "部署时会把这些资产注入本次运行。";

	return (
		<Alert
			className="pipeline-asset-context"
			type="info"
			showIcon
			message={message}
			description={
				<div className="pipeline-asset-context__body">
					<span>{description}</span>
					<div className="pipeline-asset-context__assets">
						{visibleIds.map((assetId) => (
							<Tag key={assetId} color="blue" style={{ marginInlineEnd: 0 }}>
								{assetId}
							</Tag>
						))}
						{hiddenCount > 0 ? <Tag>+{hiddenCount}</Tag> : null}
					</div>
					<Button size="small" onClick={onClear}>
						转为无资产运行
					</Button>
				</div>
			}
		/>
	);
}

export interface PipelineDesignerCanvasProps {
	onDirtyChange?: (dirty: boolean) => void;
}

function createCanvasSnapshot(
	name: string,
	nodes: PipelineFlowNode[],
	edges: PipelineFlowEdge[],
): string {
	return JSON.stringify({
		name,
		nodes: nodes.map((node) => ({
			id: node.id,
			type: node.type,
			position: node.position,
			data: node.data,
		})),
		edges: edges.map((edge) => ({
			id: edge.id,
			source: edge.source,
			target: edge.target,
			sourceHandle: edge.sourceHandle ?? null,
			targetHandle: edge.targetHandle ?? null,
			type: edge.type,
			data: edge.data,
		})),
	});
}

export function confirmLeaveWithUnsavedChanges(
	modal: ReturnType<typeof App.useApp>["modal"],
): Promise<boolean> {
	return new Promise((resolve) => {
		modal.confirm({
			title: "离开当前编辑？",
			content: "存在未保存的变更，离开后这些修改会丢失。",
			okText: "离开",
			okType: "danger",
			cancelText: "继续编辑",
			onOk: () => resolve(true),
			onCancel: () => resolve(false),
		});
	});
}

export function PipelineDesignerCanvas({
	onDirtyChange,
}: PipelineDesignerCanvasProps) {
	return <PipelineDesignerCanvasInner onDirtyChange={onDirtyChange} />;
}

function PipelineDesignerCanvasInner({
	onDirtyChange,
}: PipelineDesignerCanvasProps) {
	const navigate = useNavigate();
	const [searchParams, setSearchParams] = useSearchParams();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const { message: messageApi, modal } = App.useApp();
	const queryAssetIds = useMemo(
		() => parseAssetIds(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [state, dispatch] = useDesignerReducer({
		...initialDesignerState,
		deploy: {
			...initialDesignerState.deploy,
			selectedAssetIds: queryAssetIds,
		},
	});
	const [nodes, setNodes, onNodesChange] = useNodesState<PipelineFlowNode>([]);
	const [edges, setEdges] = useEdgesState<PipelineFlowEdge>([]);
	const [edgeMode, setEdgeMode] = useState<EdgeConnectionMode>("data");
	const engine = useXyflowCanvasEngine(nodes, edges, setNodes, setEdges);
	const workflowNameReplaceRef = useRef(false);
	const {
		components: registeredComponents,
		loading: componentsLoading,
		error: componentsError,
		reload: reloadComponents,
	} = usePipelineComponents(
		apiToRegistered,
		keepAllRegisteredComponents,
		releaseToRegistered,
	);
	const cleanCanvasSnapshotRef = useRef<string | null>(null);
	const templateId = useMemo(
		() => searchParams.get("templateId") || null,
		[searchParams],
	);
	const {
		pipelineName,
		selectedNodeId,
		editingNodeId,
		contextMenu,
		jsonOutput,
		importModalOpen,
		importText,
	} = state.canvas;
	const {
		templateLoading,
		templateVersions,
		selectedTemplateVersionId,
		loadedTemplateScope,
	} = state.template;
	const {
		deployDialog,
		selectedAssetIds,
		assetPickerResetKey,
		executionTargets,
		selectedTargetId,
	} = state.deploy;
	const {
		selectedNodeAsset,
		selectedNodeAssetLoading,
		selectedNodeAssetError,
	} = state.assetContext;
	const selectedNode = useMemo(
		() => nodes.find((node) => node.id === selectedNodeId) ?? null,
		[nodes, selectedNodeId],
	);
	const contextMenuNode = useMemo(
		() =>
			contextMenu.nodeId
				? (nodes.find((node) => node.id === contextMenu.nodeId) ?? null)
				: null,
		[contextMenu.nodeId, nodes],
	);
	const readOnlyMode =
		searchParams.get("readonly") === "1" || loadedTemplateScope === "prod";
	const assetPickerRef = useRef<AssetPickerHandle>(null);
	const handleFlowError = useCallback((code: string, flowMessage: string) => {
		if (code === "002") return;
		console.warn(`[React Flow]: ${flowMessage}`);
	}, []);
	const markCanvasClean = useCallback(
		(
			nextName: string,
			nextNodes: PipelineFlowNode[],
			nextEdges: PipelineFlowEdge[],
		) => {
			cleanCanvasSnapshotRef.current = createCanvasSnapshot(
				nextName,
				nextNodes,
				nextEdges,
			);
			onDirtyChange?.(false);
		},
		[onDirtyChange],
	);
	const selectedExecutionTarget = useMemo(
		() =>
			executionTargets.find((target) => target.id === selectedTargetId) ?? null,
		[executionTargets, selectedTargetId],
	);
	// Per-dispatch Argo priority override for the designer deploy dialog;
	// undefined = inherit the selected pool's default.
	const [deployPriorityOverride, setDeployPriorityOverride] = useState<
		number | undefined
	>(undefined);
	// CYB-3486: fetch each target cluster's ElasticQuotas so the pool picker can
	// show live usage. The designer's deploy dropdown previously showed only
	// "name · namespace" — no signal on which pool actually has room.
	const [poolEqs, setPoolEqs] = useState<Record<string, ElasticQuota[]>>({});
	useEffect(() => {
		const clusterKeys = Array.from(
			new Set(executionTargets.map((t) => t.clusterId ?? "")),
		);
		let cancelled = false;
		Promise.all(
			clusterKeys.map((k) =>
				listElasticQuotas(k)
					.then((eqs) => [k, eqs] as const)
					.catch(() => [k, [] as ElasticQuota[]] as const),
			),
		).then((pairs) => {
			if (!cancelled) setPoolEqs(Object.fromEntries(pairs));
		});
		return () => {
			cancelled = true;
		};
	}, [executionTargets]);
	const hasDataEdges = useMemo(
		() => edges.some((edge) => !isDependencyEdge(edge)),
		[edges],
	);

	const onEdgesChange = useCallback(
		(changes: EdgeChange[]) => {
			setEdges((current) => {
				const next = applyEdgeChanges(changes, current);
				const duplicate = findDuplicateTargetInput(next);
				if (duplicate) {
					messageApi.warning(
						`输入端口 ${duplicate.key} 已有连线，请在 join 节点配置不同输入端口后再连接。`,
					);
					return current;
				}
				return next;
			});
		},
		[messageApi, setEdges],
	);
	const onConnect = useCallback(
		(connection: Connection) => {
			if (!connection.source || !connection.target) return;
			if (edgeMode === "dependency") {
				const nextEdge: PipelineFlowEdge = {
					id: `${connection.source}->${connection.target}:dependency`,
					source: connection.source,
					target: connection.target,
					animated: true,
					style: DEPENDENCY_EDGE_STYLE,
					data: dependencyEdgeData(),
				};
				setEdges((current) => {
					const withoutSameEdge = current.filter(
						(edge) => edge.id !== nextEdge.id,
					);
					return [...withoutSameEdge, nextEdge];
				});
				return;
			}
			const sourceHandle = connection.sourceHandle || DEFAULT_SOURCE_PORT;
			const targetHandle = connection.targetHandle || DEFAULT_TARGET_PORT;
			const nextEdge: PipelineFlowEdge = {
				id: `${connection.source}:${sourceHandle}->${connection.target}:${targetHandle}`,
				source: connection.source,
				target: connection.target,
				sourceHandle,
				targetHandle,
				style: DATA_EDGE_STYLE,
			};
			const sourceData = nodes.find(
				(node) => node.id === connection.source,
			)?.data;
			const willWarnContract =
				nodeDataDeclaresOutput(sourceData, sourceHandle) &&
				!nodeDataWritesOutputPath(sourceData, sourceHandle);
			const prospective = [
				...edges.filter((edge) => edge.id !== nextEdge.id),
				nextEdge,
			];
			const willBeDuplicate = Boolean(findDuplicateTargetInput(prospective));
			setEdges((current) => {
				const withoutSameEdge = current.filter(
					(edge) => edge.id !== nextEdge.id,
				);
				const next = [...withoutSameEdge, nextEdge];
				const duplicate = findDuplicateTargetInput(next);
				if (duplicate) {
					messageApi.warning(
						`输入端口 ${duplicate.key} 已有连线，请在 join 节点配置不同输入端口后再连接。`,
					);
					return current;
				}
				return next;
			});
			if (!willBeDuplicate && willWarnContract) {
				messageApi.warning(
					`数据连线已创建，但 ${connection.source} 的脚本未写入 /tmp/outputs/${sourceHandle}；若只需控制先后顺序请改用「顺序」连线，否则部署会失败。`,
				);
			}
		},
		[edgeMode, messageApi, setEdges, nodes, edges],
	);
	const convertDataEdgesToDependencies = useCallback(() => {
		let convertedCount = 0;
		setEdges((current) =>
			current.map((edge) => {
				if (isDependencyEdge(edge)) return edge;
				convertedCount += 1;
				return {
					...edge,
					sourceHandle: undefined,
					targetHandle: undefined,
					animated: true,
					style: DEPENDENCY_EDGE_STYLE,
					data: dependencyEdgeData(edge.data),
				};
			}),
		);
		if (convertedCount > 0) {
			messageApi.success(`已将 ${convertedCount} 条连线转为顺序依赖`);
		}
	}, [messageApi, setEdges]);
	const canvasSnapshot = useMemo(
		() => createCanvasSnapshot(pipelineName, nodes, edges),
		[pipelineName, nodes, edges],
	);
	const hasUnsavedChanges =
		!readOnlyMode &&
		cleanCanvasSnapshotRef.current !== null &&
		cleanCanvasSnapshotRef.current !== canvasSnapshot;

	useEffect(() => {
		if (cleanCanvasSnapshotRef.current === null) {
			cleanCanvasSnapshotRef.current = canvasSnapshot;
		}
	}, [canvasSnapshot]);

	useEffect(() => {
		onDirtyChange?.(hasUnsavedChanges);
	}, [hasUnsavedChanges, onDirtyChange]);

	useEffect(() => {
		if (!hasUnsavedChanges) return;
		const handleBeforeUnload = (event: BeforeUnloadEvent) => {
			event.preventDefault();
			event.returnValue = "";
		};
		window.addEventListener("beforeunload", handleBeforeUnload);
		return () => window.removeEventListener("beforeunload", handleBeforeUnload);
	}, [hasUnsavedChanges]);

	useEffect(() => {
		dispatch({ type: "deploy/setSelectedAssetIds", assetIds: queryAssetIds });
	}, [queryAssetIds, dispatch]);

	const updateSelectedAssetIds = useCallback(
		(nextIds: string[]) => {
			dispatch({ type: "deploy/setSelectedAssetIds", assetIds: nextIds });
			const nextParams = new URLSearchParams(searchParams);
			if (nextIds.length > 0) {
				nextParams.set("asset_ids", nextIds.join(","));
			} else {
				nextParams.delete("asset_ids");
			}
			setSearchParams(nextParams, { replace: true });
		},
		[searchParams, setSearchParams, dispatch],
	);

	const loadPipelineToCanvas = useCallback(
		(pipeline: Pipeline, displayName?: string) => {
			const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
			setNodes(n);
			setEdges(e);
			dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
			// CYB-3486: 基线快照的名字必须和最终落到 state 的名字一致。模板的展示名
			// (template.name)常与内嵌 pipeline.name 不同(后者甚至为空),若在 clean
			// 之后再单独 setPipelineName,基线 name 与当前 name 不符 → "未保存"误报。
			// 因此把权威展示名显式传进来,让 setPipelineName 与 markCanvasClean 用同一个值。
			const nextName = displayName || pipeline.name || pipelineName;
			if (nextName) {
				dispatch({ type: "canvas/setPipelineName", name: nextName });
			}
			dispatch({ type: "canvas/selectNode", nodeId: null });
			engine.deselectAll();
			dispatch({ type: "canvas/setJsonOutput", json: null });
			markCanvasClean(nextName, n, e);
		},
		[engine, markCanvasClean, pipelineName, setEdges, setNodes, dispatch],
	);

	const loadPipelineFromSessionStorage = useCallback(() => {
		const raw = sessionStorage.getItem("pipeline-edit");
		if (!raw) return;
		sessionStorage.removeItem("pipeline-edit");
		try {
			loadPipelineToCanvas(JSON.parse(raw));
		} catch {
			/* ignore */
		}
	}, [loadPipelineToCanvas]);

	usePipelineTemplateLoader({
		templateId,
		dispatch,
		messageApi,
		loadPipelineToCanvas,
		loadPipelineFromSessionStorage,
	});

	const handleTemplateVersionChange = useCallback(
		async (versionId: string) => {
			const version = templateVersions.find((item) => item.id === versionId);
			if (!version) return;
			if (hasUnsavedChanges) {
				const confirmed = await confirmLeaveWithUnsavedChanges(modal);
				if (!confirmed) return;
			}
			dispatch({ type: "template/setSelectedVersionId", versionId });
			// 展示名随基线一起在 loadPipelineToCanvas 内落定,避免二次 setPipelineName。
			loadPipelineToCanvas(version.pipeline, version.name);
			navigate(
				`/pipeline?templateId=${encodeURIComponent(versionId)}&tab=design`,
				{
					replace: true,
				},
			);
		},
		[
			hasUnsavedChanges,
			loadPipelineToCanvas,
			modal,
			navigate,
			templateVersions,
			dispatch,
		],
	);

	const componentById = useMemo(() => {
		const map = new Map<string, RegisteredComponent>();
		for (const component of registeredComponents) {
			map.set(component.id, component);
		}
		return map;
	}, [registeredComponents]);

	const onDragStart = useCallback((e: DragEvent, comp: RegisteredComponent) => {
		e.dataTransfer.setData("application/databrew-component-id", comp.id);
		e.dataTransfer.setData("text/plain", comp.id);
		e.dataTransfer.effectAllowed = "copy";
	}, []);

	const onDragOver = useCallback((event: DragEvent) => {
		event.preventDefault();
		event.dataTransfer.dropEffect = "copy";
	}, []);

	const addComponentToCanvas = useCallback(
		(comp: RegisteredComponent, clientPosition?: { x: number; y: number }) => {
			if (readOnlyMode) return;
			const bounds = wrapperRef.current?.getBoundingClientRect();
			const screenPosition =
				clientPosition ?? defaultNodeScreenPosition(bounds, nodes.length);
			const position = engine.screenToFlowPosition(screenPosition);
			const newNode = createPipelineNode(comp, position.x, position.y);
			engine.addNode(newNode);
			engine.selectElements([newNode.id]);
			dispatch({ type: "canvas/selectNode", nodeId: newNode.id });
			dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
		},
		[engine, nodes.length, readOnlyMode, dispatch],
	);

	const onDrop = useCallback(
		(event: DragEvent) => {
			event.preventDefault();
			event.stopPropagation();
			const componentId =
				event.dataTransfer.getData("application/databrew-component-id") ||
				event.dataTransfer.getData("text/plain");
			if (!componentId) return;
			const comp = componentById.get(componentId);
			if (!comp) return;
			addComponentToCanvas(comp, {
				x: event.clientX,
				y: event.clientY,
			});
		},
		[addComponentToCanvas, componentById],
	);

	const selectNodeWithEdges = useCallback(
		(node: PipelineFlowNode) => {
			const connectedEdgeIds = edges
				.filter((edge) => edge.source === node.id || edge.target === node.id)
				.map((edge) => edge.id);
			dispatch({ type: "canvas/selectNode", nodeId: node.id });
			engine.selectElements([node.id, ...connectedEdgeIds]);
		},
		[edges, engine, dispatch],
	);

	const onNodeClick = useCallback(
		(event: React.MouseEvent, node: PipelineFlowNode) => {
			dispatch({ type: "canvas/closeContextMenu" });
			selectNodeWithEdges(node);
			if (event.detail > 1) {
				dispatch({ type: "canvas/setEditingNodeId", nodeId: node.id });
			}
		},
		[selectNodeWithEdges, dispatch],
	);
	const onPaneClick = useCallback(() => {
		dispatch({ type: "canvas/selectNode", nodeId: null });
		dispatch({ type: "canvas/closeContextMenu" });
		dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
		engine.deselectAll();
	}, [engine, dispatch]);

	const onPaneContextMenu = useCallback(
		(event: React.MouseEvent | MouseEvent) => {
			event.preventDefault();
			dispatch({ type: "canvas/selectNode", nodeId: null });
			dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
			engine.deselectAll();
			dispatch({
				type: "canvas/setContextMenu",
				menu: {
					open: true,
					x: "clientX" in event ? event.clientX : 0,
					y: "clientY" in event ? event.clientY : 0,
					nodeId: null,
				},
			});
		},
		[engine, dispatch],
	);

	const onNodeContextMenu = useCallback(
		(event: React.MouseEvent, node: PipelineFlowNode) => {
			event.preventDefault();
			event.stopPropagation();
			selectNodeWithEdges(node);
			dispatch({
				type: "canvas/setContextMenu",
				menu: {
					open: true,
					x: event.clientX,
					y: event.clientY,
					nodeId: node.id,
				},
			});
		},
		[selectNodeWithEdges, dispatch],
	);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			engine.updateNodeData(id, data);
		},
		[engine],
	);

	const closeNodeConfig = useCallback(() => {
		dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
	}, [dispatch]);

	const saveNodeConfig = useCallback(
		(id: string, data: Partial<PipelineNodeData>) => {
			updateNodeData(id, data as Record<string, unknown>);
			closeNodeConfig();
		},
		[closeNodeConfig, updateNodeData],
	);

	const editingNode = useMemo(
		() => nodes.find((node) => node.id === editingNodeId) ?? null,
		[nodes, editingNodeId],
	);

	const handleContextMenuClick = useCallback(
		({ key }: { key: string }) => {
			const menuNode = contextMenuNode;
			dispatch({ type: "canvas/closeContextMenu" });

			if (menuNode) {
				if (key === "configure") {
					selectNodeWithEdges(menuNode);
					dispatch({
						type: "canvas/setEditingNodeId",
						nodeId: menuNode.id,
					});
					return;
				}
				if (key === "copy") {
					engine.selectElements([menuNode.id]);
					window.setTimeout(() => {
						void engine
							.copySelection()
							.catch(() => messageApi.warning("浏览器未允许读取剪贴板"));
					}, 0);
					return;
				}
				if (key === "delete") {
					engine.selectElements([menuNode.id]);
					window.setTimeout(() => {
						engine.deleteSelection();
						if (selectedNodeId === menuNode.id) {
							dispatch({ type: "canvas/selectNode", nodeId: null });
						}
						if (editingNodeId === menuNode.id) {
							dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
						}
					}, 0);
					return;
				}
			}

			if (key === "paste") {
				void engine
					.paste()
					.catch(() => messageApi.warning("浏览器未允许读取剪贴板"));
			}
			if (key === "selectAll") {
				engine.selectAll();
			}
			if (key === "zoomIn") {
				engine.zoomIn();
			}
			if (key === "zoomOut") {
				engine.zoomOut();
			}
			if (key === "fitView") {
				engine.fitView();
			}
		},
		[
			contextMenuNode,
			editingNodeId,
			engine,
			messageApi,
			selectNodeWithEdges,
			selectedNodeId,
			dispatch,
		],
	);

	const buildPipelineJSON = useCallback(
		(): Pipeline => toTranspilerPipeline(nodes, edges, { name: pipelineName }),
		[nodes, edges, pipelineName],
	);
	const assertPipelineSavable = useCallback(
		(pipeline: Pipeline, actionLabel: string) => {
			const validation = validatePipelineForSave(pipeline);
			if (validation.valid) return true;
			messageApi.error(`${actionLabel}失败: ${validation.errors[0]}`);
			return false;
		},
		[messageApi],
	);
	const assertPipelineRunnable = useCallback(
		(pipeline: Pipeline, actionLabel: string) => {
			const validation = validatePipelineForRun(pipeline);
			if (validation.valid) {
				if (validation.warnings[0]) {
					messageApi.warning(validation.warnings[0]);
				}
				return true;
			}
			messageApi.error(`${actionLabel}失败: ${validation.errors[0]}`);
			return false;
		},
		[messageApi],
	);

	const pipelineAssetIds = useMemo(
		() => extractPipelineAssetIds(buildPipelineJSON()),
		[buildPipelineJSON],
	);

	const selectedNodeAssetId = useMemo(() => {
		const nodeAssetIds = extractNodeAssetIds(selectedNode);
		if (nodeAssetIds.length > 0) {
			return nodeAssetIds[0];
		}
		if (pipelineAssetIds.length > 0) {
			return pipelineAssetIds[0];
		}
		return selectedAssetIds[0] ?? "";
	}, [pipelineAssetIds, selectedAssetIds, selectedNode]);

	useEffect(() => {
		if (!selectedNodeAssetId) {
			dispatch({ type: "assetContext/reset" });
			return;
		}

		let alive = true;
		dispatch({ type: "assetContext/setLoading", loading: true });
		dispatch({ type: "assetContext/setError", error: null });

		assetsApi
			.get(selectedNodeAssetId)
			.then((asset) => {
				if (!alive) return;
				dispatch({ type: "assetContext/setAsset", asset });
			})
			.catch(() => {
				if (!alive) return;
				dispatch({ type: "assetContext/setAsset", asset: null });
				dispatch({
					type: "assetContext/setError",
					error: "加载关联资产失败",
				});
			})
			.finally(() => {
				if (!alive) return;
				dispatch({ type: "assetContext/setLoading", loading: false });
			});

		return () => {
			alive = false;
		};
	}, [selectedNodeAssetId, dispatch]);

	const selectedNodeAssetInfo = useMemo(() => {
		if (!selectedNodeAsset) {
			return null;
		}

		const raw = selectedNodeAsset as unknown as Record<string, unknown>;
		return {
			name: parseAssetName(raw, selectedNodeAsset.asset_id),
			type: parseAssetType(raw, selectedNodeAsset.type ?? "—"),
			size: formatBytes(extractAssetSizeBytes(raw)),
		};
	}, [selectedNodeAsset]);

	const exportPipeline = useCallback(() => {
		dispatch({
			type: "canvas/setJsonOutput",
			json: JSON.stringify(buildPipelineJSON(), null, 2),
		});
	}, [buildPipelineJSON, dispatch]);

	const importPipeline = useCallback(() => {
		dispatch({ type: "canvas/setImportText", text: "" });
		dispatch({ type: "canvas/setImportModalOpen", open: true });
	}, [dispatch]);

	const applyImportedPipeline = useCallback(() => {
		const text = importText.trim();
		if (!text) {
			messageApi.warning("请粘贴 Pipeline JSON");
			return;
		}
		try {
			const pipeline: Pipeline = JSON.parse(text);
			const { nodes: importedNodes, edges: importedEdges } =
				fromTranspilerPipeline(pipeline);
			const nextName = pipeline.name || pipelineName;
			setNodes(importedNodes);
			setEdges(importedEdges);
			dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
			if (pipeline.name) {
				dispatch({ type: "canvas/setPipelineName", name: pipeline.name });
			}
			dispatch({ type: "canvas/resetImport" });
			markCanvasClean(nextName, importedNodes, importedEdges);
			messageApi.success("导入成功");
		} catch {
			messageApi.error("无效的 JSON");
		}
	}, [
		importText,
		markCanvasClean,
		messageApi,
		pipelineName,
		setNodes,
		setEdges,
		dispatch,
	]);

	const applyExampleToCanvas = useCallback(
		(example: PipelineExample) => {
			const { nodes: exampleNodes, edges: exampleEdges } =
				fromTranspilerPipeline(example.pipeline);
			const positionedNodes = exampleNodes.map((node) => ({
				...node,
				position: example.layout[node.id] ?? node.position,
			}));
			setNodes(positionedNodes);
			setEdges(exampleEdges);
			dispatch({
				type: "canvas/setPipelineName",
				name: example.pipeline.name,
			});
			dispatch({ type: "template/clearVersions" });
			dispatch({ type: "canvas/selectNode", nodeId: null });
			dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
			engine.deselectAll();
			dispatch({ type: "canvas/setJsonOutput", json: null });
			markCanvasClean(example.pipeline.name, positionedNodes, exampleEdges);
			messageApi.success(`已载入示例: ${example.label}`);
		},
		[engine, markCanvasClean, messageApi, setNodes, setEdges, dispatch],
	);

	const loadExample = useCallback(
		(exampleKey: string) => {
			const example = getPipelineExample(exampleKey);
			if (!example) return;
			const load = () => applyExampleToCanvas(example);
			if (nodes.length > 0 || edges.length > 0) {
				modal.confirm({
					title: "载入示例",
					content: "将覆盖当前画布。请确认当前修改已保存或不再需要。",
					okText: "载入",
					cancelText: "取消",
					onOk: load,
				});
				return;
			}
			load();
		},
		[applyExampleToCanvas, edges.length, modal.confirm, nodes.length],
	);

	const clearCanvas = useCallback(() => {
		modal.confirm({
			title: "清空画布",
			content: "将删除当前所有节点与连线，此操作不可撤销。",
			okText: "清空",
			okType: "danger",
			cancelText: "取消",
			onOk: () => {
				setNodes([]);
				setEdges([]);
				dispatch({ type: "canvas/selectNode", nodeId: null });
				dispatch({ type: "canvas/setEditingNodeId", nodeId: null });
				engine.deselectAll();
				dispatch({ type: "canvas/setJsonOutput", json: null });
				markCanvasClean(pipelineName, [], []);
			},
		});
	}, [
		engine,
		markCanvasClean,
		modal.confirm,
		pipelineName,
		setEdges,
		setNodes,
		dispatch,
	]);

	const handleSave = useCallback(async () => {
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineSavable(pipeline, "保存")) return;
			const saved = await savePipeline(pipelineName, pipeline);
			dispatch({
				type: "template/setSelectedVersionId",
				versionId: saved.id,
			});
			dispatch({
				type: "template/setVersions",
				versions: [
					saved,
					...templateVersions.filter((item) => item.id !== saved.id),
				].sort((a, b) => b.version - a.version),
			});
			markCanvasClean(pipelineName, nodes, edges);
			navigate(
				`/pipeline?templateId=${encodeURIComponent(saved.id)}&tab=design`,
				{
					replace: true,
				},
			);
			messageApi.success(`已保存为 v${saved.version}，可在「流水线」页签管理`);
		} catch (err) {
			messageApi.error(`保存失败: ${String(err)}`);
		}
	}, [
		pipelineName,
		buildPipelineJSON,
		navigate,
		assertPipelineSavable,
		messageApi,
		markCanvasClean,
		nodes,
		templateVersions,
		edges,
		dispatch,
	]);

	const canDeploy = nodes.length > 0 && !readOnlyMode;

	const openDeployDialog = useCallback(() => {
		if (nodes.length === 0) {
			messageApi.warning("请先从左侧拖入至少一个组件到画布");
			return;
		}
		if (!assertPipelineRunnable(buildPipelineJSON(), "部署")) {
			return;
		}
		dispatch({
			type: "deploy/openDialog",
			name: defaultDeployWorkflowName(),
		});
	}, [
		nodes.length,
		buildPipelineJSON,
		assertPipelineRunnable,
		messageApi,
		dispatch,
	]);

	const closeDeployDialog = useCallback(() => {
		dispatch({ type: "deploy/resetDialog" });
	}, [dispatch]);

	const { handleDeploy, handlePreviewDeploy } = usePipelineDeploy({
		assetPickerRef,
		buildPipelineJSON,
		assertPipelineRunnable,
		deployDialog,
		pipelineName,
		selectedAssetIds,
		selectedTargetId,
		priorityOverride: deployPriorityOverride,
		dispatch,
		closeDeployDialog,
		messageApi,
		navigate,
	});

	usePipelineKeyboardShortcuts({
		onSave: readOnlyMode ? () => undefined : handleSave,
		onDeploy: readOnlyMode ? () => undefined : openDeployDialog,
		onClear: readOnlyMode ? () => undefined : clearCanvas,
		onCloseModal: () => {
			if (deployDialog.open) closeDeployDialog();
			if (importModalOpen) {
				dispatch({ type: "canvas/resetImport" });
			}
		},
		canDeploy,
		modalOpen: deployDialog.open || importModalOpen,
	});

	const deployDialogAssetId = useMemo(() => {
		if (!deployDialog.result) return "";
		return (
			extractPipelineAssetIds(deployDialog.result.pipelineJSON ?? {})[0] ?? ""
		);
	}, [deployDialog.result]);

	const isCanvasEmpty = nodes.length === 0;
	const currentTemplateLabel = pipelineName || "未命名流水线";
	const deployDisabledReason = readOnlyMode
		? "只读模式：正式版模板不可部署"
		: canDeploy
			? "保存并部署为 Argo Workflow (⌘/Ctrl+D)"
			: "请先从左侧拖入至少一个组件到画布，再保存或部署";
	const markReplaceOnNextEdit = (
		event: FocusEvent<HTMLInputElement>,
		replaceRef: MutableRefObject<boolean>,
	) => {
		replaceRef.current = true;
		event.target.select();
	};

	return (
		<div className="pipeline-page">
			{/* Header */}
			<div className="pipeline-toolbar">
				<div className="pipeline-toolbar__left">
					<Typography.Title level={5} className="pipeline-toolbar__title">
						流水线设计
					</Typography.Title>
					<div className="pipeline-toolbar__name">
						<span className="pipeline-toolbar__name-label">名称</span>
						<Input
							id="pipeline-name-input"
							name="pipelineName"
							value={pipelineName}
							onChange={(e) => {
								dispatch({
									type: "canvas/setPipelineName",
									name: e.target.value,
								});
							}}
							maxLength={48}
							placeholder="输入流水线名称"
							aria-label="流水线名称"
							autoComplete="off"
							className="pipeline-toolbar__name-input"
							size="small"
						/>
					</div>
					{templateVersions.length > 0 ? (
						<Select
							size="small"
							className="pipeline-toolbar__version-select"
							value={selectedTemplateVersionId ?? undefined}
							onChange={handleTemplateVersionChange}
							options={templateVersions.map((version) => ({
								value: version.id,
								label: `v${version.version}`,
							}))}
							aria-label="流水线模板版本"
						/>
					) : null}
				</div>
				<div className="pipeline-toolbar__actions">
					{readOnlyMode ? (
						<Alert
							type="info"
							showIcon
							message="只读模式"
							description="正在查看正式版（prod）流水线定义，不可修改或部署。"
							style={{ padding: "4px 12px", fontSize: 12 }}
						/>
					) : (
						<>
							<Select
								size="small"
								placeholder="载入示例"
								style={{ width: 180 }}
								value={undefined}
								onChange={loadExample}
								options={PIPELINE_EXAMPLES.map((example) => ({
									value: example.key,
									label: example.label,
									title: example.description,
								}))}
								aria-label="载入标准流水线示例"
							/>
							<Tooltip title="数据连线会传递 /tmp/outputs 文件；顺序连线只控制节点先后。">
								<div className="pipeline-toolbar__edge-mode">
									<span className="pipeline-toolbar__edge-mode-label">
										新增连线
									</span>
									<Segmented<EdgeConnectionMode>
										size="small"
										value={edgeMode}
										onChange={setEdgeMode}
										options={[
											{ label: "数据", value: "data" },
											{ label: "顺序", value: "dependency" },
										]}
										aria-label="连线模式"
									/>
								</div>
							</Tooltip>
							{edgeMode === "dependency" && hasDataEdges ? (
								<Tooltip title="把当前画布已有的数据连线改成只控制先后顺序的依赖线。">
									<Button
										size="small"
										icon={<SwapOutlined />}
										onClick={convertDataEdgesToDependencies}
									>
										已有线转顺序
									</Button>
								</Tooltip>
							) : null}
							<Tooltip title="一键拓扑布局 (Dagre)">
								<Button
									size="small"
									icon={<ColumnWidthOutlined />}
									onClick={() => {
										const layout = computeDagreLayout(nodes, edges, "LR");
										setNodes((nds) =>
											nds.map((n) => ({
												...n,
												position: layout[n.id] ?? n.position,
											})),
										);
									}}
								>
									自动布局
								</Button>
							</Tooltip>
							<Tooltip title={deployDisabledReason}>
								{canDeploy ? (
									// 启用态直接渲染 Button：多余的 <span> 包裹会吞掉首次点击，
									// 导致需要点两次才能打开部署弹窗。
									<Button
										size="small"
										type="primary"
										className="pipeline-toolbar__deploy"
										icon={<PlayCircleOutlined />}
										onClick={openDeployDialog}
									>
										部署
									</Button>
								) : (
									// 禁用态必须用 <span> 包裹，否则 disabled Button 不派发
									// 鼠标事件，Tooltip 无法显示禁用原因。
									<span>
										<Button
											size="small"
											type="primary"
											className="pipeline-toolbar__deploy"
											icon={<PlayCircleOutlined />}
											disabled
										>
											部署
										</Button>
									</span>
								)}
							</Tooltip>
							<Tooltip title="保存 (⌘/Ctrl+S)">
								<Button
									size="small"
									type="primary"
									ghost
									icon={<SaveOutlined />}
									onClick={handleSave}
								>
									保存
								</Button>
							</Tooltip>
							<div className="pipeline-toolbar__secondary-actions">
								<Button
									size="small"
									icon={<ExportOutlined />}
									onClick={exportPipeline}
								>
									导出
								</Button>
								<Button
									size="small"
									icon={<ImportOutlined />}
									onClick={importPipeline}
								>
									导入
								</Button>
							</div>
							<Button
								size="small"
								danger
								icon={<DeleteOutlined />}
								onClick={clearCanvas}
							>
								清空
							</Button>
						</>
					)}
				</div>
			</div>

			{/* Body */}
			<div className="pipeline-body">
				<ComponentPalette
					components={registeredComponents}
					onDragStart={onDragStart}
					onAddComponent={addComponentToCanvas}
					loading={componentsLoading}
					error={componentsError}
					onRetry={reloadComponents}
					disabled={readOnlyMode}
				/>
				<div
					className={[
						"canvas-wrapper",
						isCanvasEmpty ? "canvas-wrapper--empty" : "",
					]
						.filter(Boolean)
						.join(" ")}
					ref={wrapperRef}
					style={{ flex: 1, height: "100%", position: "relative" }}
					onDrop={readOnlyMode ? undefined : onDrop}
					onDragOver={readOnlyMode ? undefined : onDragOver}
					role="application"
					aria-label="流水线画布"
				>
					<AssetRunContextBanner
						assetIds={selectedAssetIds}
						isCanvasEmpty={isCanvasEmpty}
						onClear={() => updateSelectedAssetIds([])}
					/>
					{templateLoading ? (
						<div className="pipeline-canvas-loading" aria-busy="true">
							<Spin />
							<span className="pipeline-loading-text">正在加载模板...</span>
						</div>
					) : null}
					{isCanvasEmpty && !templateLoading ? (
						<PipelineEmptyState
							variant="canvas"
							title="添加组件开始设计"
							description="从左侧组件栏点击或拖拽步骤，连接节点后保存。"
							hint="保存后可在「流水线」页签打开、运行或继续编辑。"
						/>
					) : null}
					<ErrorBoundary
						title="画布渲染错误"
						subTitle="流水线画布出现异常，可重试或刷新页面"
					>
						<DesignerFlowSurface
							nodes={nodes}
							edges={edges}
							onNodesChange={onNodesChange}
							onEdgesChange={onEdgesChange}
							onConnect={onConnect}
							readOnlyMode={readOnlyMode}
							onDrop={onDrop}
							onDragOver={onDragOver}
							onNodeClick={onNodeClick}
							onNodeContextMenu={onNodeContextMenu}
							onPaneClick={onPaneClick}
							onPaneContextMenu={onPaneContextMenu}
							onError={handleFlowError}
						/>
					</ErrorBoundary>
					{contextMenu.open && (
						<div
							className="pipeline-context-menu"
							style={{
								left: contextMenu.x,
								top: contextMenu.y,
							}}
						>
							<Menu
								selectable={false}
								onClick={handleContextMenuClick}
								items={
									contextMenuNode
										? [
												...(readOnlyMode
													? []
													: [{ key: "configure", label: "配置节点" }]),
												{ key: "copy", label: "复制节点" },
												{ type: "divider" },
												...(readOnlyMode
													? []
													: [
															{
																key: "delete",
																label: "删除节点",
																danger: true,
															},
														]),
											]
										: [
												{ key: "paste", label: "粘贴" },
												{ key: "selectAll", label: "选择全部" },
												{ type: "divider" },
												{ key: "zoomIn", label: "放大" },
												{ key: "zoomOut", label: "缩小" },
												{ key: "fitView", label: "适应画布" },
											]
								}
							/>
						</div>
					)}
				</div>
				<aside
					className={[
						"config-panel",
						selectedNode ? "config-panel--with-node" : "config-panel--empty",
					]
						.filter(Boolean)
						.join(" ")}
				>
					{selectedNode ? (
						<div className="config-panel__node">
							<div>
								<div className="config-panel-header">
									<div className="config-panel-header__info">
										<span className="config-panel-label">
											{selectedNode.data?.label || selectedNode.id}
										</span>
										<span className="config-panel-type">
											{selectedNode.data?.image || "未设置镜像"}
										</span>
									</div>
									{readOnlyMode ? null : (
										<Button
											size="small"
											type="primary"
											onClick={() =>
												dispatch({
													type: "canvas/setEditingNodeId",
													nodeId: selectedNode.id,
												})
											}
										>
											配置节点
										</Button>
									)}
								</div>
								<div className="config-content">
									<div className="config-section-title">关联资产</div>
									{selectedNodeAssetLoading ? (
										<div
											style={{
												fontSize: 12,
												color: "#64748b",
											}}
										>
											{selectedNodeAssetId
												? "正在加载关联资产..."
												: "未关联资产"}
										</div>
									) : selectedNodeAssetError ? (
										<div
											style={{
												fontSize: 12,
												color: "#dc2626",
											}}
										>
											{selectedNodeAssetError}
										</div>
									) : selectedNodeAssetInfo ? (
										<div
											style={{
												display: "grid",
												gap: 6,
											}}
										>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													名称
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.name}
												</Typography.Text>
											</div>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													类型
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.type}
												</Typography.Text>
											</div>
											<div className="config-field">
												<span
													style={{
														fontSize: 10,
														color: "var(--color-text-secondary, #64748b)",
														textTransform: "uppercase",
														letterSpacing: "0.8px",
													}}
												>
													大小
												</span>
												<Typography.Text>
													{selectedNodeAssetInfo.size}
												</Typography.Text>
											</div>
										</div>
									) : (
										<div
											style={{
												fontSize: 12,
												color: "#64748b",
											}}
										>
											未选择关联资产
										</div>
									)}
								</div>
							</div>
						</div>
					) : (
						<PipelineEmptyState
							variant="config"
							title="节点配置"
							description="选中画布节点后，在此查看关联资产与节点详情。"
							hint="已保存流水线请到「流水线」页签管理。"
						/>
					)}
				</aside>
				{editingNode && !readOnlyMode && (
					<NodeConfigPanel
						open={Boolean(editingNode)}
						node={editingNode}
						onCancel={closeNodeConfig}
						onSave={saveNodeConfig}
					/>
				)}
			</div>

			{/* JSON Output */}
			{jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

			{/* Deploy Dialog */}
			<Modal
				open={importModalOpen}
				title="导入流水线"
				okText="导入"
				cancelText="取消"
				onOk={applyImportedPipeline}
				onCancel={() => dispatch({ type: "canvas/resetImport" })}
				destroyOnHidden
			>
				<p style={{ marginTop: 0, color: "#64748b", fontSize: 13 }}>
					粘贴 Pipeline
					JSON。导入成功后会覆盖当前画布内容，请先确认当前修改已保存。
				</p>
				<TextArea
					value={importText}
					onChange={(event) => {
						dispatch({
							type: "canvas/setImportText",
							text: event.target.value,
						});
					}}
					placeholder='{"name":"my-pipeline","nodes":[],"edges":[]}'
					autoSize={{ minRows: 10, maxRows: 18 }}
					style={{
						fontFamily: '"SF Mono", "Fira Code", monospace',
						fontSize: 12,
					}}
				/>
			</Modal>

			<Modal
				title="部署流水线"
				open={deployDialog.open}
				onCancel={closeDeployDialog}
				footer={null}
				width={780}
				destroyOnHidden
			>
				{!deployDialog.deploying &&
					!deployDialog.done &&
					deployDialog.mode === "edit" && (
						<>
							<Typography.Text
								type="secondary"
								style={{ fontSize: 12, display: "block", marginBottom: 4 }}
							>
								当前模板：{currentTemplateLabel}
							</Typography.Text>
							<Typography.Paragraph
								type="secondary"
								style={{ fontSize: 12, marginBottom: 16 }}
							>
								选择资源池和资产后，将流水线转换为 Argo Workflow 并提交到
								Kubernetes
								集群。选择多个资产时，会为每个资产各下发一条执行记录。
							</Typography.Paragraph>
							<div style={{ marginBottom: 16 }}>
								<AssetRunSummary
									assetIds={selectedAssetIds}
									onClear={() => updateSelectedAssetIds([])}
								/>
							</div>
							{!canDeploy && (
								<Alert
									type="warning"
									showIcon
									message="画布为空"
									description="请先从左侧拖入至少一个组件，或导入带节点的 Pipeline JSON。"
									style={{ marginBottom: 16 }}
								/>
							)}
							<div
								style={{
									display: "flex",
									flexDirection: "column",
									gap: 12,
									marginBottom: 16,
								}}
							>
								<label
									htmlFor="pp-workflow-name"
									style={{
										display: "flex",
										flexDirection: "column",
										gap: 4,
										fontSize: 10,
										textTransform: "uppercase",
										letterSpacing: "0.8px",
										color: "#64748b",
									}}
								>
									工作流名称
									<Input
										id="pp-workflow-name"
										value={deployDialog.name}
										onChange={(e) => {
											const next = workflowNameReplaceRef.current
												? replaceAppendedValue(
														deployDialog.name,
														e.target.value,
													)
												: e.target.value;
											workflowNameReplaceRef.current = false;
											dispatch({
												type: "deploy/setDialog",
												dialog: { name: next },
											});
										}}
										onFocus={(e) =>
											markReplaceOnNextEdit(e, workflowNameReplaceRef)
										}
										onBlur={() => {
											workflowNameReplaceRef.current = false;
										}}
										maxLength={48}
										placeholder="留空则使用当前流水线名称"
										size="small"
									/>
								</label>
								<label
									htmlFor="pp-execution-target"
									style={{
										display: "flex",
										flexDirection: "column",
										gap: 4,
										fontSize: 10,
										textTransform: "uppercase",
										letterSpacing: "0.8px",
										color: "#64748b",
									}}
								>
									资源池
									<Select
										id="pp-execution-target"
										value={selectedTargetId}
										onChange={(targetId) =>
											dispatch({
												type: "deploy/setSelectedTargetId",
												targetId,
											})
										}
										size="small"
										options={(executionTargets.length > 0
											? executionTargets
											: [
													{
														id: "default",
														name: "Default Argo target",
														cluster: "default",
														namespace: "default",
														status: "unavailable",
														isDefault: true,
														argoServerConfigured: false,
													} satisfies ExecutionTarget,
												]
										).map((target) => {
											const eq = matchPoolEq(target, poolEqs);
											const usage = eq
												? ` · ${poolAvailabilityLabel(poolAvailability(eq))} · ${poolFreeSummary(eq)}`
												: "";
											return {
												value: target.id,
												label: `${target.isDefault ? "默认目标" : target.name} · ${target.namespace}${usage}`,
												disabled: target.status !== "available",
											};
										})}
									/>
								</label>
								<label
									htmlFor="pp-priority-override"
									style={{
										display: "flex",
										flexDirection: "column",
										gap: 4,
										fontSize: 10,
										textTransform: "uppercase",
										letterSpacing: "0.8px",
										color: "#64748b",
									}}
								>
									优先级
									<div
										style={{ display: "flex", alignItems: "center", gap: 8 }}
									>
										<Select
											id="pp-priority-override"
											size="small"
											style={{ flex: 1 }}
											value={deployPriorityOverride ?? "inherit"}
											onChange={(v) =>
												setDeployPriorityOverride(
													v === "inherit" ? undefined : (v as number),
												)
											}
											options={[
												{ value: "inherit", label: "跟随池子默认" },
												...PRIORITY_TIER_OPTIONS,
											]}
										/>
										{(() => {
											const badge = priorityBadge(
												deployPriorityOverride ??
													selectedExecutionTarget?.resourceDefaults?.priority,
											);
											return <Tag color={badge.color}>{badge.label}</Tag>;
										})()}
									</div>
								</label>
								<div
									style={{
										fontSize: 10,
										color: "#64748b",
										display: "flex",
										gap: 8,
									}}
								>
									<span>{nodes.length} 个步骤</span>
									<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
									<span>{edges.length} 条连线</span>
								</div>
								<AssetPicker
									ref={assetPickerRef}
									selectedIds={selectedAssetIds}
									onSelectionChange={updateSelectedAssetIds}
									maxHeight={180}
									resetKey={assetPickerResetKey}
								/>
							</div>
							<div
								style={{
									display: "flex",
									gap: 8,
									justifyContent: "flex-end",
									borderTop: "1px solid var(--color-border, #e2e8f0)",
									paddingTop: 14,
								}}
							>
								<Button onClick={closeDeployDialog}>取消</Button>
								<Button onClick={handlePreviewDeploy} disabled={!canDeploy}>
									预览
								</Button>
								<Button
									type="primary"
									onClick={handleDeploy}
									disabled={!canDeploy}
								>
									{selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
								</Button>
							</div>
						</>
					)}
				{!deployDialog.deploying &&
					!deployDialog.done &&
					deployDialog.mode === "preview" && (
						<>
							<Typography.Text
								type="secondary"
								style={{ fontSize: 12, display: "block", marginBottom: 4 }}
							>
								当前模板：{currentTemplateLabel}
							</Typography.Text>
							<Typography.Paragraph
								type="secondary"
								style={{ fontSize: 12, marginBottom: 12 }}
							>
								先确认运行摘要；需要排查 Argo 配置时再展开原始 YAML。
							</Typography.Paragraph>
							{deployDialog.previewLoading ? (
								<div style={{ textAlign: "center", padding: 20 }}>
									<Typography.Text type="secondary">
										正在生成预览内容...
									</Typography.Text>
								</div>
							) : deployDialog.previewError ? (
								<Alert
									type="error"
									showIcon
									message="预览失败"
									description={deployDialog.previewError}
									style={{ marginBottom: 16 }}
								/>
							) : (
								<Space direction="vertical" size={12} style={{ width: "100%" }}>
									<div
										style={{
											display: "grid",
											gridTemplateColumns:
												"repeat(auto-fit, minmax(150px, 1fr))",
											gap: 8,
										}}
									>
										{[
											["模板", currentTemplateLabel],
											["步骤", `${nodes.length} 个`],
											["连线", `${edges.length} 条`],
											[
												"资产",
												selectedAssetIds.length > 0
													? `${selectedAssetIds.length} 个`
													: "无资产运行",
											],
											[
												"目标",
												selectedExecutionTarget
													? `${selectedExecutionTarget.namespace}`
													: "默认目标",
											],
										].map(([label, value]) => (
											<div
												key={label}
												style={{
													border: "1px solid #e2e8f0",
													borderRadius: 8,
													padding: "10px 12px",
													background: "#f8fafc",
												}}
											>
												<Typography.Text
													type="secondary"
													style={{ display: "block", fontSize: 12 }}
												>
													{label}
												</Typography.Text>
												<Typography.Text strong>{value}</Typography.Text>
											</div>
										))}
									</div>
									{selectedExecutionTarget ? (
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											资源池：{selectedExecutionTarget.name}
										</Typography.Text>
									) : null}
									<Collapse
										size="small"
										items={[
											{
												key: "manifest",
												label: "原始 Argo YAML",
												children: (
													<pre
														style={{
															margin: 0,
															padding: 12,
															background: "#0f172a",
															color: "#e2e8f0",
															borderRadius: 8,
															overflow: "auto",
															maxHeight: 300,
															fontSize: 12,
															fontFamily:
																'"SF Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
															lineHeight: 1.5,
															whiteSpace: "pre",
														}}
													>
														{deployDialog.previewManifest || "（暂无内容）"}
													</pre>
												),
											},
										]}
									/>
								</Space>
							)}
							<div
								style={{
									display: "flex",
									gap: 8,
									justifyContent: "flex-end",
									borderTop: "1px solid var(--color-border, #e2e8f0)",
									paddingTop: 14,
									marginTop: 12,
								}}
							>
								<Button
									onClick={() =>
										dispatch({
											type: "deploy/setDialog",
											dialog: { mode: "edit" },
										})
									}
								>
									返回编辑
								</Button>
								<Button
									type="primary"
									onClick={handleDeploy}
									disabled={!canDeploy}
								>
									{selectedAssetIds.length > 0 ? "运行资产" : "无资产运行"}
								</Button>
							</div>
						</>
					)}
				{deployDialog.deploying && (
					<div style={{ textAlign: "center", padding: 20 }}>
						<Typography.Text type="secondary">
							正在部署流水线...
						</Typography.Text>
					</div>
				)}
				{deployDialog.done && (
					<div style={{ textAlign: "center", padding: 20 }}>
						{deployDialog.error ? (
							<>
								<Typography.Text type="danger" strong>
									部署失败
								</Typography.Text>
								<pre
									style={{
										fontSize: 11,
										color: "#ef4444",
										marginTop: 8,
										whiteSpace: "pre-wrap",
									}}
								>
									{deployDialog.error}
								</pre>
							</>
						) : deployDialog.result ? (
							<>
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>
									当前模板：{currentTemplateLabel}
								</Typography.Text>
								<br />
								<Typography.Text type="success" strong>
									{(deployDialog.results?.length ?? 1) > 1
										? `已下发 ${deployDialog.results?.length ?? 1} 个任务`
										: "部署成功"}
								</Typography.Text>
								{(deployDialog.results?.length ?? 0) > 1 ? (
									<div
										style={{
											marginTop: 12,
											textAlign: "left",
											maxHeight: 180,
											overflowY: "auto",
										}}
									>
										{(deployDialog.results ?? []).map((item) => (
											<p
												key={item.id}
												style={{
													fontFamily: '"SF Mono",monospace',
													fontSize: 12,
													color: "#64748b",
													margin: "4px 0",
												}}
											>
												{item.workflowName}
											</p>
										))}
									</div>
								) : (
									<p
										style={{
											fontFamily: '"SF Mono",monospace',
											fontSize: 12,
											color: "#64748b",
											marginTop: 8,
										}}
									>
										{deployDialog.result.workflowName}
									</p>
								)}
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>
									{deployDialog.result.assetCount
										? `${deployDialog.result.assetCount} 个资产`
										: "无资产"}
									{deployDialog.result.executionTarget
										? ` · ${deployDialog.result.executionTarget.cluster}/${deployDialog.result.executionTarget.namespace}`
										: ""}
								</Typography.Text>
								{deployDialogAssetId ? (
									<Button
										type="link"
										onClick={() => navigate(`/assets/${deployDialogAssetId}`)}
									>
										查看关联资产
									</Button>
								) : null}
								<div
									style={{
										fontSize: 10,
										color: "#64748b",
										display: "flex",
										gap: 8,
										justifyContent: "center",
										marginTop: 8,
									}}
								>
									<span>{deployDialog.result.nodeCount} 个步骤</span>
									<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
									<span>
										{new Date(deployDialog.result.createdAt).toLocaleString()}
									</span>
								</div>
								<div
									style={{
										marginTop: 16,
										display: "flex",
										gap: 8,
										justifyContent: "center",
									}}
								>
									<Button
										type="primary"
										onClick={() => {
											closeDeployDialog();
											navigate(
												deployDialog.result?.id
													? `/runs/${encodeURIComponent(deployDialog.result.id)}`
													: `/pipeline/executions/${encodeURIComponent(
															deployDialog.result?.workflowName ?? "",
														)}`,
											);
										}}
									>
										查看运行
									</Button>
									<Button
										onClick={() => {
											closeDeployDialog();
											navigate("/runs");
										}}
									>
										查看记录
									</Button>
									<Button onClick={closeDeployDialog}>关闭</Button>
								</div>
							</>
						) : null}
					</div>
				)}
			</Modal>
		</div>
	);
}
