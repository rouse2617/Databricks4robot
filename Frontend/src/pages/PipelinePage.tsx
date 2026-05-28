import {
	DeleteOutlined,
	ExportOutlined,
	ImportOutlined,
	PlayCircleOutlined,
	SaveOutlined,
} from "@ant-design/icons";
import {
	type Edge,
	FlowEditor,
	FlowEditorProvider,
	type Node,
	SelectType,
	useFlowEditor,
} from "@ant-design/pro-flow";
import {
	Alert,
	Button,
	Collapse,
	Input,
	Menu,
	Modal,
	message,
	Tooltip,
	Typography,
} from "antd";
import {
	type DragEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { assetsApi } from "../api/assets";
import {
	type Deployment,
	deployTemplate,
	savePipeline,
} from "../api/pipelineApi";
import {
	createComponent,
	deleteComponent,
	listComponents,
	type PipelineComponentAPI,
	type PipelineComponentPayload,
	type PipelineComponentType,
	updateComponent,
} from "../api/pipelineComponentApi";
import type { Asset } from "../api/types";
import AssetPicker from "../components/pipeline/AssetPicker";
import { ComponentManager } from "../components/pipeline/ComponentManager";
import { ComponentPalette } from "../components/pipeline/ComponentPalette";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { NodeConfigPanel } from "../components/pipeline/NodeConfigPanel";
import { PipelineStepNode } from "../components/pipeline/PipelineNode";
import type {
	Pipeline,
	PipelineNodeData,
	RegisteredComponent,
} from "../components/pipeline/types";
import {
	fromTranspilerPipeline,
	toTranspilerPipeline,
} from "../lib/pipelineContract";

import "../styles/pipeline.css";

const nodeTypes = { pipelineStep: PipelineStepNode };

const STORAGE_KEY = "databrew-components";

type PipelineFlowNode = Node<PipelineNodeData>;
type PipelineFlowEdge = Edge;

type CanvasMenuState = {
	open: boolean;
	x: number;
	y: number;
	node: PipelineFlowNode | null;
};

function toRecord<T extends { id: string }>(items: T[]): Record<string, T> {
	return items.reduce<Record<string, T>>((acc, item) => {
		acc[item.id] = item;
		return acc;
	}, {});
}

function dedupeComponentsByName(
	comps: RegisteredComponent[],
): RegisteredComponent[] {
	const seen = new Set<string>();
	return comps.filter((c) => {
		const key = c.name.trim().toLowerCase();
		if (!key || seen.has(key)) return false;
		seen.add(key);
		return true;
	});
}

function imageHasTag(image: string): boolean {
	if (image.includes("@")) return true;
	const lastSlash = image.lastIndexOf("/");
	const lastColon = image.lastIndexOf(":");
	return lastColon > lastSlash && lastColon < image.length - 1;
}

function formatImage(image: string, tag?: string): string {
	if (!tag || imageHasTag(image)) return image;
	return `${image}:${tag}`;
}

function normalizeComponentType(
	type: string | undefined,
): PipelineComponentType {
	const value = (type || "").trim();
	switch (value) {
		case "container":
		case "script":
		case "resource":
		case "suspend":
			return value;
		default:
			return "container";
	}
}

function loadComponents(): RegisteredComponent[] {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) return JSON.parse(raw);
	} catch {
		/* ignore */
	}
	return [];
}

function saveComponents(comps: RegisteredComponent[]) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(comps));
}

function uniqSorted(values: string[]): string[] {
	return Array.from(new Set(values.filter(Boolean))).sort();
}

function toStringArray(value: unknown): string[] {
	if (typeof value === "string") return [value].filter(Boolean);
	if (!Array.isArray(value)) return [];
	return value
		.map((item) => (typeof item === "string" ? item.trim() : ""))
		.filter((item) => item.length > 0);
}

function parseAssetIds(raw: string | null | undefined): string[] {
	if (!raw) return [];
	return raw
		.split(",")
		.map((item) => item.trim())
		.filter((item) => item.length > 0);
}

function extractNodeAssetIds(node: PipelineFlowNode | null): string[] {
	if (!node) return [];
	const data = node.data as Record<string, unknown>;
	const direct = uniqSorted([
		...toStringArray(data.asset_id),
		...toStringArray(data.assetId),
		...toStringArray(data.input_asset_id),
		...toStringArray(data._input_asset_id),
		...toStringArray(data.asset_ids),
		...toStringArray(data.assetIds),
		...toStringArray(data.input_asset_ids),
		...toStringArray(data._input_asset_ids),
	]);
	const fromAssetSelection = toStringArray(
		(data.assetSelection as Record<string, unknown> | undefined)?.assetIds,
	);
	return uniqSorted([...direct, ...fromAssetSelection]);
}

function extractPipelineAssetIds(
	pipeline: Pipeline | Record<string, unknown>,
): string[] {
	const contract = pipeline as Record<string, unknown>;
	return uniqSorted([
		...toStringArray(contract._input_asset_ids),
		...toStringArray(contract.input_asset_ids),
		...toStringArray(
			(contract.input as Record<string, unknown> | undefined)?.asset_ids,
		),
		...toStringArray(
			(contract.assetSelection as Record<string, unknown> | undefined)
				?.assetIds,
		),
	]);
}

function formatBytes(bytes: unknown): string {
	const n = typeof bytes === "string" ? Number(bytes) : bytes;
	if (typeof n !== "number" || !Number.isFinite(n) || n < 0) return "—";
	if (n === 0) return "0 B";
	const units = ["B", "KB", "MB", "GB", "TB", "PB"];
	let size = n;
	let i = 0;
	while (size >= 1024 && i < units.length - 1) {
		size /= 1024;
		i += 1;
	}
	return `${size.toFixed(i === 0 ? 0 : 2)} ${units[i]}`;
}

function toNumber(value: unknown): number | null {
	if (typeof value === "number") {
		return Number.isFinite(value) ? value : null;
	}
	if (typeof value === "string") {
		const parsed = Number(value);
		return Number.isFinite(parsed) ? parsed : null;
	}
	return null;
}

function extractAssetSizeBytes(raw: Record<string, unknown>): number | null {
	const directSize =
		toNumber(raw.size) ??
		toNumber(raw.size_bytes) ??
		toNumber((raw as { bytes?: unknown }).bytes);
	if (directSize !== null) return directSize;

	const directNested = [raw.metadata, raw.files_json];
	for (const nested of directNested) {
		if (nested && typeof nested === "object") {
			const candidate = nested as Record<string, unknown>;
			const candidateSize =
				toNumber(candidate.size) ??
				toNumber(candidate.size_bytes) ??
				toNumber(candidate.byte_size) ??
				toNumber(candidate.bytes);
			if (candidateSize !== null) return candidateSize;
		}
	}

	if (raw.files_json && typeof raw.files_json === "object") {
		for (const value of Object.values(
			raw.files_json as Record<string, unknown>,
		)) {
			if (!value || typeof value !== "object") continue;
			const fileMeta = value as Record<string, unknown>;
			const fromFile =
				toNumber(fileMeta.size) ??
				toNumber(fileMeta.size_bytes) ??
				toNumber(fileMeta.byte_size) ??
				toNumber(fileMeta.bytes);
			if (fromFile !== null) return fromFile;
		}
	}

	return null;
}

function parseAssetName(
	asset: Record<string, unknown>,
	fallbackId: string,
): string {
	if (typeof asset.name === "string" && asset.name.trim()) return asset.name;
	if (typeof asset.display_name === "string" && asset.display_name.trim()) {
		return asset.display_name;
	}
	return fallbackId || "—";
}

function parseAssetType(
	asset: Record<string, unknown>,
	fallback: string,
): string {
	if (typeof asset.asset_type === "string" && asset.asset_type.trim()) {
		return asset.asset_type;
	}
	if (typeof asset.type === "string" && asset.type.trim()) {
		return asset.type;
	}
	return fallback;
}

/** Map backend PipelineComponentAPI → frontend RegisteredComponent. */
function apiToRegistered(api: PipelineComponentAPI): RegisteredComponent {
	const resources = api.resources ?? {};
	const normalizedType = normalizeComponentType(api.type);
	const normalizedSource = (api.source || "custom").trim() || "custom";
	const legacyArgs = resources.args as
		| { name: string; value?: string; from?: string }[]
		| undefined;
	const envFromObject = api.env
		? Object.entries(api.env).map(([name, value]) => ({
				name,
				value: value || "",
			}))
		: [];
	const envFromResource =
		Array.isArray(resources.env) && resources.env.length > 0
			? (resources.env as { name?: string; value?: string }[]).map((item) => ({
					name: item.name || "",
					value: item.value || "",
				}))
			: typeof resources.env === "object" && resources.env
				? Object.entries(resources.env as Record<string, string>).map(
						([name, value]) => ({
							name,
							value: value || "",
						}),
					)
				: [];
	return {
		id: api.id,
		name: api.name,
		type: normalizedType,
		source: normalizedSource,
		image: formatImage(api.image, api.tag),
		command: api.command ?? ((resources.command as string[]) || ["sh", "-c"]),
		args:
			legacyArgs ??
			(api.args ?? []).map((value, index) => ({
				name: `arg${index + 1}`,
				value,
			})),
		env: envFromObject.length > 0 ? envFromObject : envFromResource,
		cpu: (resources.cpu as string) ?? "",
		memory: (resources.memory as string) ?? "",
		disk: (resources.disk as string) ?? "",
	};
}

/** Map frontend RegisteredComponent → backend PipelineComponentAPI shape. */
function registeredToApi(comp: RegisteredComponent): PipelineComponentPayload {
	const idx = comp.image.lastIndexOf(":");
	const image = idx > 0 ? comp.image.slice(0, idx) : comp.image;
	const tag = idx > 0 ? comp.image.slice(idx + 1) : "latest";
	const normalizedType = normalizeComponentType(comp.type);
	const normalizedSource = (comp.source || "").trim() || "custom";
	return {
		name: comp.name,
		type: normalizedType,
		description: "",
		image,
		tag,
		source: normalizedSource,
		command: comp.command,
		args: comp.args.map((arg) => arg.value || arg.name).filter(Boolean),
		inputPorts: [{ name: "input", type: "string" }],
		outputPorts: [{ name: "output", type: "string" }],
		resources: {
			command: comp.command,
			args: comp.args,
			type: normalizedType,
			env: (comp.env ?? []).reduce<Record<string, string>>((acc, item) => {
				const key = item.name?.trim();
				if (!key) return acc;
				acc[key] = item.value || "";
				return acc;
			}, {}),
			cpu: comp.cpu,
			memory: comp.memory,
			disk: comp.disk,
		},
		envVars: [],
	};
}

let nodeCounter = 0;

function createPipelineNode(
	comp: RegisteredComponent,
	x: number,
	y: number,
): PipelineFlowNode {
	nodeCounter++;
	return {
		id: `step-${nodeCounter}`,
		type: "pipelineStep",
		position: { x, y },
		data: {
			label: comp.name,
			type: comp.type || "container",
			source: comp.source || "",
			image: comp.image,
			command: comp.command,
			args: comp.args || [],
			env: comp.env || [],
			cpu: comp.cpu || "",
			memory: comp.memory || "",
			disk: comp.disk || "",
			selectType: SelectType.DEFAULT,
		},
	};
}

function PipelineCanvas() {
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const editor = useFlowEditor();
	const queryAssetIds = useMemo(
		() => parseAssetIds(searchParams.get("asset_ids")),
		[searchParams],
	);
	const [nodes, setNodes] = useState<PipelineFlowNode[]>([]);
	const [edges, setEdges] = useState<PipelineFlowEdge[]>([]);
	const [pipelineName, setPipelineName] = useState("my-pipeline");
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
	const [registeredComponents, setRegisteredComponents] =
		useState<RegisteredComponent[]>(loadComponents);
	const [view, setView] = useState<"pipeline" | "components" | "deploy">(
		"pipeline",
	);
	const [deployDialog, setDeployDialog] = useState<{
		open: boolean;
		deploying: boolean;
		done: boolean;
		name: string;
		result?: Deployment;
		error?: string;
	}>({ open: false, deploying: false, done: false, name: "" });
	const [jsonOutput, setJsonOutput] = useState<string | null>(null);
	// Asset selection for deploy modal
	const [selectedAssetIds, setSelectedAssetIds] =
		useState<string[]>(queryAssetIds);
	const [selectedNodeAsset, setSelectedNodeAsset] = useState<Asset | null>(
		null,
	);
	const [selectedNodeAssetLoading, setSelectedNodeAssetLoading] =
		useState(false);
	const [selectedNodeAssetError, setSelectedNodeAssetError] = useState<
		string | null
	>(null);

	const flattenNodes = useMemo(() => toRecord(nodes), [nodes]);
	const flattenEdges = useMemo(() => toRecord(edges), [edges]);

	useEffect(() => {
		setSelectedAssetIds(queryAssetIds);
	}, [queryAssetIds]);

	useEffect(() => {
		saveComponents(registeredComponents);
	}, [registeredComponents]);

	useEffect(() => {
		setSelectedNode((prev: PipelineFlowNode | null) => {
			if (!prev) return null;
			return nodes.find((node) => node.id === prev.id) ?? null;
		});
	}, [nodes]);

	// Load registered components from API on startup, fallback to localStorage
	useEffect(() => {
		listComponents()
			.then((res) => {
				const mapped = dedupeComponentsByName(
					(res.items ?? []).map(apiToRegistered),
				);
				if (mapped.length > 0) {
					setRegisteredComponents(mapped);
					saveComponents(mapped);
				}
			})
			.catch(() => {
				// API failed, keep localStorage data (already set in useState)
			});
	}, []);

	const handleComponentSave = useCallback(
		async (comp: RegisteredComponent, isNew: boolean) => {
			try {
				const apiData = registeredToApi(comp);
				if (isNew) {
					const created = await createComponent(apiData);
					comp.id = created.id; // Use server-assigned ID
				} else {
					await updateComponent(comp.id, apiData);
				}
			} catch (err) {
				const detail = err instanceof Error ? err.message : String(err);
				throw new Error(`组件保存到服务端失败: ${detail}`);
			}
		},
		[],
	);

	const handleComponentDelete = useCallback(async (id: string) => {
		try {
			await deleteComponent(id);
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			throw new Error(`组件从服务端删除失败: ${detail}`);
		}
	}, []);

	const loadPipelineToCanvas = useCallback(
		(pipeline: Pipeline) => {
			const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
			setNodes(n);
			setEdges(e);
			setEditingNodeId(null);
			if (pipeline.name) setPipelineName(pipeline.name);
			setView("pipeline");
			setSelectedNode(null);
			editor.deselectAll();
			setJsonOutput(null);
		},
		[editor],
	);

	useEffect(() => {
		const raw = sessionStorage.getItem("pipeline-edit");
		if (!raw) return;
		sessionStorage.removeItem("pipeline-edit");
		try {
			loadPipelineToCanvas(JSON.parse(raw));
		} catch {
			/* ignore */
		}
	}, [loadPipelineToCanvas]);

	const onDragStart = useCallback((e: DragEvent, comp: RegisteredComponent) => {
		e.dataTransfer.setData("application/reactflow", JSON.stringify(comp));
		e.dataTransfer.effectAllowed = "move";
	}, []);

	const onDragOver = useCallback((event: DragEvent) => {
		event.preventDefault();
		event.dataTransfer.dropEffect = "move";
	}, []);

	const onDrop = useCallback(
		(event: DragEvent) => {
			event.preventDefault();
			const raw = event.dataTransfer.getData("application/reactflow");
			if (!raw) return;
			try {
				const comp: RegisteredComponent = JSON.parse(raw);
				const bounds = wrapperRef.current?.getBoundingClientRect();
				if (!bounds) return;
				const position = editor.screenToFlowPosition({
					x: event.clientX,
					y: event.clientY,
				});
				const newNode = createPipelineNode(comp, position.x, position.y);
				editor.addNode(newNode);
				editor.selectElements([newNode.id]);
				setSelectedNode(newNode);
				setEditingNodeId(null);
			} catch {
				/* ignore */
			}
		},
		[editor],
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
							.catch(() => message.warning("浏览器未允许读取剪贴板"));
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
					.catch(() => message.warning("浏览器未允许读取剪贴板"));
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
		[contextMenu.node, editor, selectNodeWithEdges],
	);

	const buildPipelineJSON = useCallback(
		(): Pipeline => toTranspilerPipeline(nodes, edges, { name: pipelineName }),
		[nodes, edges, pipelineName],
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
		const text = prompt("粘贴 Pipeline JSON:");
		if (!text) return;
		try {
			const pipeline: Pipeline = JSON.parse(text);
			const { nodes: importedNodes, edges: importedEdges } =
				fromTranspilerPipeline(pipeline);
			setNodes(importedNodes);
			setEdges(importedEdges);
			setEditingNodeId(null);
			if (pipeline.name) {
				setPipelineName(pipeline.name);
			}
			setJsonOutput(null);
		} catch {
			message.error("无效的 JSON");
		}
	}, []);

	const clearCanvas = useCallback(() => {
		Modal.confirm({
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
			},
		});
	}, [editor]);

	const handleSave = useCallback(async () => {
		try {
			const result = await savePipeline(pipelineName, buildPipelineJSON());
			message.success(`已保存: ${result.name}`);
		} catch (err) {
			message.error(`保存失败: ${String(err)}`);
		}
	}, [pipelineName, buildPipelineJSON]);

	const canDeploy = nodes.length > 0;

	const openDeployDialog = useCallback(() => {
		if (nodes.length === 0) {
			message.warning("请先从左侧拖入至少一个组件到画布");
			return;
		}
		setDeployDialog({
			open: true,
			deploying: false,
			done: false,
			name: pipelineName,
		});
		setSelectedAssetIds([]);
	}, [pipelineName, nodes.length]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
		});
	}, []);

	const deployDialogAssetId = useMemo(() => {
		if (!deployDialog.result) return "";
		return (
			extractPipelineAssetIds(deployDialog.result.pipelineJSON ?? {})[0] ?? ""
		);
	}, [deployDialog.result]);

	const handleDeploy = useCallback(async () => {
		setDeployDialog((prev) => ({ ...prev, deploying: true, done: false }));
		try {
			const pipeline = buildPipelineJSON();
			const name = deployDialog.name || pipelineName;
			const saved = await savePipeline(name, pipeline);
			const result = await deployTemplate(
				saved.id,
				selectedAssetIds.length > 0 ? selectedAssetIds : undefined,
			);
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				result,
			}));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				error: String(err),
			}));
		}
	}, [buildPipelineJSON, deployDialog.name, pipelineName, selectedAssetIds]);

	return (
		<div
			style={{
				display: "flex",
				flexDirection: "column",
				height: "calc(100vh - 110px)",
				overflow: "hidden",
				position: "relative",
			}}
		>
			{/* Header */}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					padding: "0 16px",
					height: 48,
					borderBottom: "1px solid var(--color-border, #e2e8f0)",
					flexShrink: 0,
					background: "#fff",
				}}
			>
				<div
					style={{
						display: "flex",
						gap: 0,
						height: "100%",
						alignItems: "stretch",
					}}
				>
					{(["pipeline", "components", "deploy"] as const).map((tab) => (
						<button
							type="button"
							key={tab}
							onClick={() => {
								setView(tab);
								setJsonOutput(null);
							}}
							style={{
								padding: "0 14px",
								border: "none",
								background: "transparent",
								cursor: "pointer",
								fontSize: 12,
								letterSpacing: "0.5px",
								textTransform: "uppercase",
								color: view === tab ? "#2563eb" : "#94a3b8",
								borderBottom:
									view === tab ? "2px solid #2563eb" : "2px solid transparent",
								fontWeight: view === tab ? 600 : 400,
								transition: "color 0.15s",
							}}
						>
							{tab === "pipeline"
								? "画布"
								: tab === "components"
									? "组件"
									: "部署"}
						</button>
					))}
				</div>

				{view === "pipeline" && (
					<>
						<Input
							value={pipelineName}
							onChange={(e) => setPipelineName(e.target.value)}
							placeholder="pipeline-name"
							style={{
								width: 180,
								fontFamily: '"SF Mono",monospace',
								fontSize: 12,
							}}
							size="small"
						/>
						<div style={{ marginLeft: "auto", display: "flex", gap: 6 }}>
							<Tooltip
								title={
									canDeploy
										? "保存并部署为 Argo Workflow"
										: "请先从左侧拖入至少一个组件到画布"
								}
							>
								<span>
									<Button
										size="small"
										type="primary"
										icon={<PlayCircleOutlined />}
										onClick={openDeployDialog}
										disabled={!canDeploy}
									>
										部署
									</Button>
								</span>
							</Tooltip>
							<Button size="small" icon={<SaveOutlined />} onClick={handleSave}>
								保存
							</Button>
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
							<Button
								size="small"
								danger
								icon={<DeleteOutlined />}
								onClick={clearCanvas}
							>
								清空
							</Button>
						</div>
					</>
				)}
			</div>

			{/* Body */}
			<div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
				{view === "pipeline" ? (
					<>
						<ComponentPalette
							components={registeredComponents}
							onDragStart={onDragStart}
						/>
						<div
							className="canvas-wrapper"
							ref={wrapperRef}
							style={{ flex: 1, height: "100%", position: "relative" }}
						>
							<FlowEditor
								nodeTypes={nodeTypes}
								flattenNodes={flattenNodes}
								flattenEdges={flattenEdges}
								onFlattenNodesChange={(nextNodes: Record<string, unknown>) =>
									setNodes(Object.values(nextNodes) as PipelineFlowNode[])
								}
								onFlattenEdgesChange={(nextEdges: Record<string, unknown>) =>
									setEdges(Object.values(nextEdges) as PipelineFlowEdge[])
								}
								contextMenuEnabled={false}
								flowProps={{
									onDrop,
									onDragOver,
									onNodeClick,
									onNodeContextMenu,
									onPaneClick,
									onPaneContextMenu,
								}}
							/>
							{nodes.length === 0 && (
								<div className="canvas-empty" aria-live="polite">
									从左侧拖入组件 → 连接圆点 → 保存 / 部署
								</div>
							)}
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
														{ key: "configure", label: "配置节点" },
														{ key: "copy", label: "复制节点" },
														{ type: "divider" },
														{
															key: "delete",
															label: "删除节点",
															danger: true,
														},
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
						<aside className="config-panel">
							{selectedNode ? (
								<>
									<div className="config-panel-header">
										<span className="config-panel-label">{selectedNode.data?.label || selectedNode.id}</span>
										<span className="config-panel-type">双击进行配置</span>
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
								</>
							) : (
								<div className="config-empty">选择一个节点进行配置</div>
							)}
						</aside>
						{editingNode && (
							<NodeConfigPanel
								open={Boolean(editingNode)}
								node={editingNode}
								onCancel={closeNodeConfig}
								onSave={saveNodeConfig}
							/>
						)}
					</>
				) : view === "components" ? (
					<div className="registry-view">
						<ComponentManager
							components={registeredComponents}
							onChange={setRegisteredComponents}
							onSaveApi={handleComponentSave}
							onDeleteApi={handleComponentDelete}
						/>
					</div>
				) : (
					<DeployPanel onEditTemplate={loadPipelineToCanvas} />
				)}
			</div>

			{/* JSON Output */}
			{jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

			{/* Deploy Dialog */}
			<Modal
				title="部署流水线"
				open={deployDialog.open}
				onCancel={closeDeployDialog}
				footer={null}
				width={480}
			>
				{!deployDialog.deploying && !deployDialog.done && (
					<>
						<Typography.Paragraph
							type="secondary"
							style={{ fontSize: 12, marginBottom: 16 }}
						>
							将流水线转换为 Argo Workflow 并提交到 Kubernetes 集群。
						</Typography.Paragraph>
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
											name: e.target.value,
										}))
									}
									placeholder={pipelineName}
									size="small"
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
								<span>{nodes.length} 个节点</span>
								<span style={{ fontSize: 3, color: "#cbd5e1" }}>•</span>
								<span>{edges.length} 条连线</span>
							</div>
							<Collapse
								ghost
								size="small"
								items={[
									{
										key: "assets",
										label: (
											<span style={{ fontSize: 12, color: "#64748b" }}>
												高级：绑定资产（可选）
											</span>
										),
										children: (
											<AssetPicker
												selectedIds={selectedAssetIds}
												onSelectionChange={setSelectedAssetIds}
												maxHeight={180}
											/>
										),
									},
								]}
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
							<Button
								type="primary"
								onClick={handleDeploy}
								disabled={!canDeploy}
							>
								部署
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
								<Typography.Text type="success" strong>
									部署成功
								</Typography.Text>
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
									<span>{deployDialog.result.nodeCount} 个节点</span>
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
												`/workflows/${deployDialog.result?.workflowName}`,
											);
										}}
									>
										查看 Workflow
									</Button>
									<Button
										onClick={() => {
											setView("deploy");
											closeDeployDialog();
										}}
									>
										查看部署
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

export default function PipelinePage() {
	return (
		<FlowEditorProvider>
			<PipelineCanvas />
		</FlowEditorProvider>
	);
}
