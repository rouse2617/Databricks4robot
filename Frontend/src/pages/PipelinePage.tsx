import {
	DeleteOutlined,
	ExportOutlined,
	ImportOutlined,
	PlayCircleOutlined,
	SaveOutlined,
} from "@ant-design/icons";
import {
	FlowEditor,
	FlowEditorProvider,
	type Node,
	useFlowEditor,
} from "@ant-design/pro-flow";
import {
	Alert,
	App,
	Button,
	Collapse,
	Input,
	Menu,
	Modal,
	Select,
	Space,
	Spin,
	Tabs,
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
import { assetsApi } from "../api/assets";
import {
	BATCH_ASSET_THRESHOLD,
	deployPipelineForAssets,
} from "../api/deployPipelineRun";
import {
	type Deployment,
	type ExecutionTarget,
	getPipeline,
	listExecutionTargets,
	listPipelineVersions,
	type PipelineTemplate,
	previewDeploy,
	savePipeline,
} from "../api/pipelineApi";
import type { Asset } from "../api/types";
import ErrorBoundary from "../components/ErrorBoundary";
import AssetPicker from "../components/pipeline/AssetPicker";
import { ComponentPalette } from "../components/pipeline/ComponentPalette";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { NodeConfigPanel } from "../components/pipeline/NodeConfigPanel";
import { PipelineEmptyState } from "../components/pipeline/PipelineEmptyState";
import { PipelineStepNode } from "../components/pipeline/PipelineNode";
import type {
	Pipeline,
	PipelineNodeData,
	RegisteredComponent,
} from "../components/pipeline/types";
import { usePipelineComponents } from "../hooks/usePipelineComponents";
import { usePipelineKeyboardShortcuts } from "../hooks/usePipelineKeyboardShortcuts";
import { MAX_BATCH_ASSET_COUNT } from "../lib/batchAssetLimits";
import {
	fromTranspilerPipeline,
	toTranspilerPipeline,
} from "../lib/pipelineContract";
import {
	getPipelineExample,
	PIPELINE_EXAMPLES,
	type PipelineExample,
} from "../lib/pipelineExamples";
import { validatePipelineForRun } from "../lib/pipelineValidation";
import { ComponentManager } from "./ComponentManager";
import { ExecutionRecordsPanel } from "./ExecutionRecordsPanel";
import {
	apiToRegistered,
	createPipelineNode,
	dedupeComponentsByName,
	extractAssetSizeBytes,
	extractNodeAssetIds,
	extractPipelineAssetIds,
	formatBytes,
	type PipelineFlowEdge,
	type PipelineFlowNode,
	parseAssetIds,
	parseAssetName,
	parseAssetType,
	toRecord,
} from "./pipeline/pipelinePageHelpers";

import "../styles/pipeline.css";

const PIPELINE_NODE_TYPES = { pipelineStep: PipelineStepNode };
const PIPELINE_EDGE_TYPES = {};

type DeployMode = "edit" | "preview";

type CanvasMenuState = {
	open: boolean;
	x: number;
	y: number;
	node: PipelineFlowNode | null;
};

const DEFAULT_TARGET_PORT = "input";

function findDuplicateTargetInput(edges: PipelineFlowEdge[]) {
	const seen = new Map<string, PipelineFlowEdge>();
	for (const edge of edges) {
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

interface PipelineCanvasProps {
	onDirtyChange?: (dirty: boolean) => void;
}

function createCanvasSnapshot(
	name: string,
	nodes: PipelineFlowNode[],
	edges: PipelineFlowEdge[],
): string {
	return JSON.stringify({ name, nodes, edges });
}

function confirmLeaveWithUnsavedChanges(
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

function PipelineCanvas({ onDirtyChange }: PipelineCanvasProps) {
	const navigate = useNavigate();
	const [searchParams, setSearchParams] = useSearchParams();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const editor = useFlowEditor();
	const editorRef = useRef(editor);
	const { message: messageApi, modal } = App.useApp();
	const queryAssetIds = useMemo(
		() => parseAssetIds(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [nodes, setNodes] = useState<PipelineFlowNode[]>([]);
	const [edges, setEdges] = useState<PipelineFlowEdge[]>([]);
	const [pipelineName, setPipelineName] = useState("my-pipeline");
	const pipelineNameReplaceRef = useRef(false);
	const workflowNameReplaceRef = useRef(false);
	const [selectedNode, setSelectedNode] = useState<PipelineFlowNode | null>(
		null,
	);
	const [contextMenu, setContextMenu] = useState<CanvasMenuState>({
		open: false,
		x: 0,
		y: 0,
		node: null,
	});
	const [editingNodeId, setEditingNodeId] = useState<string | null>(null);
	const {
		components: registeredComponents,
		loading: componentsLoading,
		error: componentsError,
		reload: reloadComponents,
	} = usePipelineComponents(apiToRegistered, dedupeComponentsByName);
	const [templateLoading, setTemplateLoading] = useState(false);
	const [templateVersions, setTemplateVersions] = useState<PipelineTemplate[]>(
		[],
	);
	const [selectedTemplateVersionId, setSelectedTemplateVersionId] = useState<
		string | null
	>(null);
	const cleanCanvasSnapshotRef = useRef<string | null>(null);
	const [loadedTemplateScope, setLoadedTemplateScope] = useState<string | null>(
		null,
	);
	const templateId = useMemo(
		() => searchParams.get("templateId") || null,
		[searchParams],
	);
	const readOnlyMode =
		searchParams.get("readonly") === "1" || loadedTemplateScope === "prod";
	const [deployDialog, setDeployDialog] = useState<{
		open: boolean;
		deploying: boolean;
		done: boolean;
		name: string;
		mode: DeployMode;
		result?: Deployment;
		results?: Deployment[];
		error?: string;
		previewManifest?: string;
		previewLoading?: boolean;
		previewError?: string;
	}>({
		open: false,
		deploying: false,
		done: false,
		name: "",
		mode: "edit",
	});
	const [jsonOutput, setJsonOutput] = useState<string | null>(null);
	const [importModalOpen, setImportModalOpen] = useState(false);
	const [importText, setImportText] = useState("");
	const importTextRef = useRef<string>("");
	// Asset selection for deploy modal
	const [selectedAssetIds, setSelectedAssetIds] =
		useState<string[]>(queryAssetIds);
	const [assetPickerResetKey, setAssetPickerResetKey] = useState(0);
	const [executionTargets, setExecutionTargets] = useState<ExecutionTarget[]>(
		[],
	);
	const [selectedTargetId, setSelectedTargetId] = useState("default");
	const [selectedNodeAsset, setSelectedNodeAsset] = useState<Asset | null>(
		null,
	);
	const [selectedNodeAssetLoading, setSelectedNodeAssetLoading] =
		useState(false);
	const [selectedNodeAssetError, setSelectedNodeAssetError] = useState<
		string | null
	>(null);
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

	const flattenNodes = useMemo(() => toRecord(nodes), [nodes]);
	const flattenEdges = useMemo(() => toRecord(edges), [edges]);
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

	const applyCanvasEdges = useCallback(
		(nextEdges: PipelineFlowEdge[]) => {
			const duplicate = findDuplicateTargetInput(nextEdges);
			if (duplicate) {
				messageApi.warning(
					`输入端口 ${duplicate.key} 已有连线，请在 join 节点配置不同输入端口后再连接。`,
				);
				return;
			}
			setEdges(nextEdges);
		},
		[messageApi],
	);

	useEffect(() => {
		editorRef.current = editor;
	}, [editor]);

	useEffect(() => {
		setSelectedAssetIds(queryAssetIds);
	}, [queryAssetIds]);

	const updateSelectedAssetIds = useCallback(
		(nextIds: string[]) => {
			setSelectedAssetIds(nextIds);
			const nextParams = new URLSearchParams(searchParams);
			if (nextIds.length > 0) {
				nextParams.set("asset_ids", nextIds.join(","));
			} else {
				nextParams.delete("asset_ids");
			}
			setSearchParams(nextParams, { replace: true });
		},
		[searchParams, setSearchParams],
	);

	useEffect(() => {
		let alive = true;
		listExecutionTargets()
			.then((targets) => {
				if (!alive) return;
				setExecutionTargets(targets);
				const defaultTarget =
					targets.find((target) => target.isDefault) ?? targets[0];
				if (defaultTarget) setSelectedTargetId(defaultTarget.id);
			})
			.catch(() => {
				if (!alive) return;
				setExecutionTargets([]);
			});
		return () => {
			alive = false;
		};
	}, []);

	useEffect(() => {
		setSelectedNode((prev: PipelineFlowNode | null) => {
			if (!prev) return null;
			return nodes.find((node) => node.id === prev.id) ?? null;
		});
	}, [nodes]);

	const loadPipelineToCanvas = useCallback(
		(pipeline: Pipeline) => {
			const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
			setNodes(n);
			setEdges(e);
			setEditingNodeId(null);
			const nextName = pipeline.name || pipelineName;
			if (pipeline.name) setPipelineName(pipeline.name);
			setSelectedNode(null);
			editorRef.current.deselectAll();
			setJsonOutput(null);
			markCanvasClean(nextName, n, e);
		},
		[markCanvasClean, pipelineName],
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

	useEffect(() => {
		if (!templateId) {
			setTemplateVersions((prev) => (prev.length > 0 ? [] : prev));
			setSelectedTemplateVersionId((prev) => (prev !== null ? null : prev));
			setLoadedTemplateScope(null);
			loadPipelineFromSessionStorage();
			return;
		}

		let cancelled = false;
		setTemplateLoading(true);
		Promise.all([getPipeline(templateId), listPipelineVersions(templateId)])
			.then(([template, versions]) => {
				if (cancelled) return;
				loadPipelineToCanvas(template.pipeline);
				setPipelineName(template.name);
				setTemplateVersions(versions);
				setSelectedTemplateVersionId(template.id);
				setLoadedTemplateScope(template.scope ?? null);
			})
			.catch((err) => {
				if (cancelled) return;
				messageApi.error(`模板加载失败: ${String(err)}`);
				setTemplateVersions([]);
				setSelectedTemplateVersionId(null);
				loadPipelineFromSessionStorage();
			})
			.finally(() => {
				if (!cancelled) setTemplateLoading(false);
			});

		return () => {
			cancelled = true;
		};
	}, [
		loadPipelineFromSessionStorage,
		loadPipelineToCanvas,
		messageApi,
		templateId,
	]);

	const handleTemplateVersionChange = useCallback(
		async (versionId: string) => {
			const version = templateVersions.find((item) => item.id === versionId);
			if (!version) return;
			if (hasUnsavedChanges) {
				const confirmed = await confirmLeaveWithUnsavedChanges(modal);
				if (!confirmed) return;
			}
			setSelectedTemplateVersionId(versionId);
			loadPipelineToCanvas(version.pipeline);
			setPipelineName(version.name);
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
				clientPosition ??
				(bounds
					? {
							x: bounds.left + bounds.width / 2 + nodes.length * 24,
							y: bounds.top + 140 + nodes.length * 24,
						}
					: { x: 360 + nodes.length * 24, y: 240 + nodes.length * 24 });
			const position = editor.screenToFlowPosition(screenPosition);
			const newNode = createPipelineNode(comp, position.x, position.y);
			editor.addNode(newNode);
			editor.selectElements([newNode.id]);
			setSelectedNode(newNode);
			setEditingNodeId(null);
		},
		[editor, nodes.length, readOnlyMode],
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
			setSelectedNode(node);
			editor.selectElements([node.id, ...connectedEdgeIds]);
		},
		[edges, editor],
	);

	const onNodeClick = useCallback(
		(event: React.MouseEvent, node: Node) => {
			setContextMenu((prev) => ({ ...prev, open: false }));
			const pipelineNode = node as PipelineFlowNode;
			selectNodeWithEdges(pipelineNode);
			if (event.detail > 1) {
				setEditingNodeId(pipelineNode.id);
			}
		},
		[selectNodeWithEdges],
	);
	const onPaneClick = useCallback(() => {
		setSelectedNode(null);
		setContextMenu((prev) => ({ ...prev, open: false }));
		setEditingNodeId(null);
		editor.deselectAll();
	}, [editor]);

	const onPaneContextMenu = useCallback(
		(event: React.MouseEvent) => {
			event.preventDefault();
			setSelectedNode(null);
			setEditingNodeId(null);
			editor.deselectAll();
			setContextMenu({
				open: true,
				x: event.clientX,
				y: event.clientY,
				node: null,
			});
		},
		[editor],
	);

	const onNodeContextMenu = useCallback(
		(event: React.MouseEvent, node: Node) => {
			event.preventDefault();
			event.stopPropagation();
			const pipelineNode = node as PipelineFlowNode;
			selectNodeWithEdges(pipelineNode);
			setContextMenu({
				open: true,
				x: event.clientX,
				y: event.clientY,
				node: pipelineNode,
			});
		},
		[selectNodeWithEdges],
	);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			editor.updateNodeData(id, data);
			setSelectedNode((prev: PipelineFlowNode | null) =>
				prev?.id === id ? { ...prev, data: { ...prev.data, ...data } } : prev,
			);
		},
		[editor],
	);

	const closeNodeConfig = useCallback(() => {
		setEditingNodeId(null);
	}, []);

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
			const menuNode = contextMenu.node;
			setContextMenu((prev) => ({ ...prev, open: false }));

			if (menuNode) {
				if (key === "configure") {
					selectNodeWithEdges(menuNode);
					setEditingNodeId(menuNode.id);
					return;
				}
				if (key === "copy") {
					editor.selectElements([menuNode.id]);
					window.setTimeout(() => {
						void editor
							.copySelection()
							.catch(() => messageApi.warning("浏览器未允许读取剪贴板"));
					}, 0);
					return;
				}
				if (key === "delete") {
					editor.selectElements([menuNode.id]);
					window.setTimeout(() => {
						editor.deleteSelection();
						setSelectedNode((prev: PipelineFlowNode | null) =>
							prev?.id === menuNode.id ? null : prev,
						);
						setEditingNodeId((prev) => (prev === menuNode.id ? null : prev));
					}, 0);
					return;
				}
			}

			if (key === "paste") {
				void editor
					.paste()
					.catch(() => messageApi.warning("浏览器未允许读取剪贴板"));
			}
			if (key === "selectAll") {
				editor.selectAll();
			}
			if (key === "zoomIn") {
				editor.reactflow?.zoomIn();
			}
			if (key === "zoomOut") {
				editor.reactflow?.zoomOut();
			}
			if (key === "fitView") {
				editor.reactflow?.fitView();
			}
		},
		[contextMenu.node, editor, messageApi, selectNodeWithEdges],
	);

	const buildPipelineJSON = useCallback(
		(): Pipeline => toTranspilerPipeline(nodes, edges, { name: pipelineName }),
		[nodes, edges, pipelineName],
	);
	const assertPipelineRunnable = useCallback(
		(pipeline: Pipeline, actionLabel: string) => {
			const validation = validatePipelineForRun(pipeline);
			if (validation.valid) return true;
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
			setSelectedNodeAsset(null);
			setSelectedNodeAssetError(null);
			setSelectedNodeAssetLoading(false);
			return;
		}

		let alive = true;
		setSelectedNodeAssetLoading(true);
		setSelectedNodeAssetError(null);

		assetsApi
			.get(selectedNodeAssetId)
			.then((asset) => {
				if (!alive) return;
				setSelectedNodeAsset(asset);
			})
			.catch(() => {
				if (!alive) return;
				setSelectedNodeAsset(null);
				setSelectedNodeAssetError("加载关联资产失败");
			})
			.finally(() => {
				if (!alive) return;
				setSelectedNodeAssetLoading(false);
			});

		return () => {
			alive = false;
		};
	}, [selectedNodeAssetId]);

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
		setJsonOutput(JSON.stringify(buildPipelineJSON(), null, 2));
	}, [buildPipelineJSON]);

	const importPipeline = useCallback(() => {
		setImportText("");
		importTextRef.current = "";
		setImportModalOpen(true);
	}, []);

	const applyImportedPipeline = useCallback(() => {
		const text =
			(
				document.querySelector(
					".ant-modal textarea",
				) as HTMLTextAreaElement | null
			)?.value?.trim() || importTextRef.current.trim();
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
			setEditingNodeId(null);
			if (pipeline.name) {
				setPipelineName(pipeline.name);
			}
			setJsonOutput(null);
			setImportModalOpen(false);
			setImportText("");
			markCanvasClean(nextName, importedNodes, importedEdges);
			messageApi.success("导入成功");
		} catch {
			messageApi.error("无效的 JSON");
		}
	}, [markCanvasClean, messageApi, pipelineName]);

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
			setPipelineName(example.pipeline.name);
			setSelectedTemplateVersionId(null);
			setTemplateVersions([]);
			setSelectedNode(null);
			setEditingNodeId(null);
			editor.deselectAll();
			setJsonOutput(null);
			markCanvasClean(example.pipeline.name, positionedNodes, exampleEdges);
			messageApi.success(`已载入示例: ${example.label}`);
		},
		[editor, markCanvasClean, messageApi],
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
				setSelectedNode(null);
				setEditingNodeId(null);
				editor.deselectAll();
				setJsonOutput(null);
				markCanvasClean(pipelineName, [], []);
			},
		});
	}, [editor, markCanvasClean, modal.confirm, pipelineName]);

	const handleSave = useCallback(async () => {
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineRunnable(pipeline, "保存")) return;
			const saved = await savePipeline(pipelineName, pipeline);
			setSelectedTemplateVersionId(saved.id);
			setTemplateVersions((prev) => {
				const withoutSaved = prev.filter((item) => item.id !== saved.id);
				return [saved, ...withoutSaved].sort((a, b) => b.version - a.version);
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
		assertPipelineRunnable,
		messageApi,
		markCanvasClean,
		nodes,
		edges,
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
		setDeployDialog({
			open: true,
			deploying: false,
			done: false,
			mode: "edit",
			name: pipelineName,
		});
		setAssetPickerResetKey((key) => key + 1);
	}, [
		pipelineName,
		nodes.length,
		buildPipelineJSON,
		assertPipelineRunnable,
		messageApi,
	]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
			mode: "edit",
		});
		setAssetPickerResetKey((key) => key + 1);
	}, []);

	usePipelineKeyboardShortcuts({
		onSave: readOnlyMode ? () => undefined : handleSave,
		onDeploy: readOnlyMode ? () => undefined : openDeployDialog,
		onClear: readOnlyMode ? () => undefined : clearCanvas,
		onCloseModal: () => {
			if (deployDialog.open) closeDeployDialog();
			if (importModalOpen) {
				setImportModalOpen(false);
				setImportText("");
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

	const handleDeploy = useCallback(async () => {
		const assetCount = selectedAssetIds.length;
		if (assetCount >= BATCH_ASSET_THRESHOLD) {
			const confirmed = await new Promise<boolean>((resolve) => {
				Modal.confirm({
					title: "确认创建批量任务",
					content: (
						<div>
							<p>
								将为 {assetCount} 个资产各创建 1 条子任务，共 {assetCount}{" "}
								条执行记录。
							</p>
							<p style={{ marginBottom: 0, color: "#64748b", fontSize: 12 }}>
								提交后可在「执行记录 →
								批量任务」查看进度，并支持暂停或重试失败项。
							</p>
						</div>
					),
					okText: "确认运行",
					cancelText: "取消",
					onOk: () => resolve(true),
					onCancel: () => resolve(false),
				});
			});
			if (!confirmed) return;
		}

		setDeployDialog((prev) => ({ ...prev, deploying: true, done: false }));
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineRunnable(pipeline, "运行")) {
				setDeployDialog((prev) => ({ ...prev, deploying: false }));
				return;
			}
			const name = deployDialog.name || pipelineName;
			const saved = await savePipeline(name, pipeline);
			const result = await deployPipelineForAssets(saved.id, selectedAssetIds, {
				targetId: selectedTargetId,
				batchName: `${name}-${Date.now()}`,
			});
			if (result.mode === "batch") {
				messageApi.success(
					`已创建批量任务，共 ${result.batchJob.totalCount} 个子任务`,
				);
				closeDeployDialog();
				navigate(`/pipeline/batch/${result.batchJob.id}`);
				return;
			}
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				result: result.runs[0],
				results: result.runs,
			}));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				error: String(err),
			}));
		}
	}, [
		buildPipelineJSON,
		assertPipelineRunnable,
		deployDialog.name,
		pipelineName,
		selectedAssetIds,
		selectedTargetId,
		closeDeployDialog,
		messageApi,
		navigate,
	]);

	const handlePreviewDeploy = useCallback(async () => {
		setDeployDialog((prev) => ({
			...prev,
			previewLoading: true,
			previewError: undefined,
			previewManifest: undefined,
			mode: "preview",
		}));
		try {
			const pipeline = buildPipelineJSON();
			if (!assertPipelineRunnable(pipeline, "预览")) {
				setDeployDialog((prev) => ({
					...prev,
					previewLoading: false,
					mode: "edit",
				}));
				return;
			}
			const { manifest } = await previewDeploy(pipeline);
			setDeployDialog((prev) => ({
				...prev,
				previewLoading: false,
				previewManifest: manifest,
			}));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				previewLoading: false,
				previewError: String(err),
			}));
		}
	}, [buildPipelineJSON, assertPipelineRunnable]);

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
				</div>

				<div className="pipeline-toolbar__name">
					<span className="pipeline-toolbar__name-label">名称</span>
					<Input
						id="pipeline-name-input"
						name="pipelineName"
						value={pipelineName}
						onChange={(e) => {
							const next = pipelineNameReplaceRef.current
								? replaceAppendedValue(pipelineName, e.target.value)
								: e.target.value;
							pipelineNameReplaceRef.current = false;
							setPipelineName(next);
						}}
						onFocus={(e) => markReplaceOnNextEdit(e, pipelineNameReplaceRef)}
						onBlur={() => {
							pipelineNameReplaceRef.current = false;
						}}
						maxLength={48}
						placeholder="输入流水线名称"
						aria-label="流水线名称"
						autoComplete="off"
						className="pipeline-toolbar__name-input"
						size="small"
					/>
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
							<Tooltip title={deployDisabledReason}>
								<span>
									<Button
										size="small"
										type="primary"
										className="pipeline-toolbar__deploy"
										icon={<PlayCircleOutlined />}
										onClick={openDeployDialog}
										disabled={!canDeploy}
									>
										部署
									</Button>
								</span>
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
						<FlowEditor
							nodeTypes={PIPELINE_NODE_TYPES}
							flattenNodes={flattenNodes}
							flattenEdges={flattenEdges}
							onFlattenNodesChange={(nextNodes: Record<string, unknown>) =>
								setNodes(Object.values(nextNodes) as PipelineFlowNode[])
							}
							onFlattenEdgesChange={(nextEdges: Record<string, unknown>) =>
								applyCanvasEdges(Object.values(nextEdges) as PipelineFlowEdge[])
							}
							contextMenuEnabled={false}
							flowProps={{
								edgeTypes: PIPELINE_EDGE_TYPES,
								nodesDraggable: !readOnlyMode,
								nodesConnectable: !readOnlyMode,
								elementsSelectable: true,
								onDrop: readOnlyMode ? undefined : onDrop,
								onDragOver: readOnlyMode ? undefined : onDragOver,
								onNodeClick,
								onNodeContextMenu,
								onPaneClick,
								onPaneContextMenu,
								onError: handleFlowError,
								onlyRenderVisibleElements: true,
								minZoom: 0.2,
								maxZoom: 2,
								proOptions: { hideAttribution: true },
							}}
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
									contextMenu.node
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
											onClick={() => setEditingNodeId(selectedNode.id)}
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
				onCancel={() => {
					setImportModalOpen(false);
					setImportText("");
				}}
				destroyOnHidden
			>
				<p style={{ marginTop: 0, color: "#64748b", fontSize: 13 }}>
					粘贴 Pipeline
					JSON。导入成功后会覆盖当前画布内容，请先确认当前修改已保存。
				</p>
				<TextArea
					value={importText}
					onChange={(event) => {
						setImportText(event.target.value);
						importTextRef.current = event.target.value;
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
								选择执行目标和资产后，将流水线转换为 Argo Workflow 并提交到
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
										onChange={(e) =>
											setDeployDialog((prev) => ({
												...prev,
												name: workflowNameReplaceRef.current
													? replaceAppendedValue(prev.name, e.target.value)
													: e.target.value,
											}))
										}
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
									执行目标
									<Select
										id="pp-execution-target"
										value={selectedTargetId}
										onChange={setSelectedTargetId}
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
										).map((target) => ({
											value: target.id,
											label: `${target.isDefault ? "默认目标" : target.name} · ${target.namespace}`,
											disabled: target.status !== "available",
										}))}
									/>
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
											执行目标：{selectedExecutionTarget.cluster}/
											{selectedExecutionTarget.namespace}
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
										setDeployDialog((prev) => ({ ...prev, mode: "edit" }))
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
												`/pipeline/executions/${deployDialog.result?.workflowName}`,
											);
										}}
									>
										查看 Workflow
									</Button>
									<Button
										onClick={() => {
											closeDeployDialog();
											navigate("/pipeline?tab=executions");
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

type PipelineTab = "design" | "pipelines" | "executions" | "components";

function resolvePipelineTab(raw: string | null): PipelineTab {
	switch (raw) {
		case "templates":
		case "pipelines":
			return "pipelines";
		case "executions":
			return "executions";
		case "components":
			return "components";
		default:
			return "design";
	}
}

export default function PipelinePage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const { modal } = App.useApp();
	const [designDirty, setDesignDirty] = useState(false);
	const activeTab = useMemo(
		() => resolvePipelineTab(searchParams.get("tab")),
		[searchParams],
	);
	const onTabChange = useCallback(
		async (nextTab: string) => {
			if (nextTab !== activeTab && designDirty) {
				const confirmed = await confirmLeaveWithUnsavedChanges(modal);
				if (!confirmed) return;
				setDesignDirty(false);
			}
			const next = new URLSearchParams(searchParams);
			if (nextTab === "design") {
				next.delete("tab");
			} else {
				next.set("tab", nextTab);
				next.delete("templateId");
				next.delete("readonly");
				next.delete("asset_ids");
			}
			setSearchParams(next, { replace: true });
		},
		[activeTab, designDirty, modal, searchParams, setSearchParams],
	);

	const tabLabel = useCallback((title: string, subtitle: string) => {
		return (
			<div className="pipeline-tab-label">
				<span className="pipeline-tab-label__title">{title}</span>
				<span className="pipeline-tab-label__subtitle">{subtitle}</span>
			</div>
		);
	}, []);

	return (
		<div className="pipeline-tabs-page">
			<Tabs
				activeKey={activeTab}
				onChange={onTabChange}
				destroyOnHidden={false}
				animated={{ inkBar: true, tabPane: true }}
				items={[
					{
						key: "design",
						label: tabLabel("设计", "编辑流水线画布"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--canvas">
								<ErrorBoundary
									title="流水线设计错误"
									subTitle="设计画布加载失败，可重试或刷新页面"
								>
									<FlowEditorProvider>
										<PipelineCanvas onDirtyChange={setDesignDirty} />
									</FlowEditorProvider>
								</ErrorBoundary>
							</div>
						),
					},
					{
						key: "pipelines",
						label: tabLabel("流水线", "管理已保存的流水线"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--management">
								<DeployPanel variant="full" />
							</div>
						),
					},
					{
						key: "executions",
						label: tabLabel("执行记录", "查看和管理流水线运行"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel pipeline-tab-content--executions">
								<ExecutionRecordsPanel active={activeTab === "executions"} />
							</div>
						),
					},
					{
						key: "components",
						label: tabLabel("组件", "管理可复用的步骤定义"),
						children: (
							<div className="pipeline-tab-content pipeline-tab-content--panel">
								<ComponentManager />
							</div>
						),
					},
				]}
			/>
		</div>
	);
}
