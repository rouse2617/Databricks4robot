import { useState, useCallback, useRef, useEffect, type DragEvent } from "react";
import {
	ReactFlow,
	addEdge,
	useNodesState,
	useEdgesState,
	type Node,
	type Edge,
	type Connection,
	type NodeTypes,
	Background,
	Controls,
	MiniMap,
	BackgroundVariant,
	ReactFlowProvider,
	useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Button, Input, message, Modal, Typography, Collapse } from "antd";
import {
	PlayCircleOutlined,
	SaveOutlined,
	ExportOutlined,
	ImportOutlined,
	DeleteOutlined,
} from "@ant-design/icons";
import { ComponentPalette } from "../components/pipeline/ComponentPalette";
import { NodeConfigPanel } from "../components/pipeline/NodeConfigPanel";
import { ComponentManager } from "../components/pipeline/ComponentManager";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { PipelineStepNode } from "../components/pipeline/PipelineNode";
import type {
	RegisteredComponent,
	Pipeline,
	PipelineNodeData,
} from "../components/pipeline/types";
import { savePipeline, deployTemplate, type Deployment } from "../api/pipelineApi";
import {
	listComponents,
	createComponent,
	updateComponent,
	deleteComponent,
	type PipelineComponentAPI,
} from "../api/pipelineComponentApi";
import { toTranspilerPipeline, fromTranspilerPipeline } from "../lib/pipelineContract";
import { useNavigate } from "react-router-dom";
import AssetPicker from "../components/pipeline/AssetPicker";

import "../styles/pipeline.css";

const nodeTypes: NodeTypes = { pipelineStep: PipelineStepNode };

const STORAGE_KEY = "databrew-components";

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

/** Map backend PipelineComponentAPI → frontend RegisteredComponent. */
function apiToRegistered(api: PipelineComponentAPI): RegisteredComponent {
	const resources = api.resources ?? {};
	return {
		id: api.id,
		name: api.name,
		image: api.tag ? `${api.image}:${api.tag}` : api.image,
		command: (resources.command as string[]) ?? ["sh", "-c"],
		args: (resources.args as {name: string; value?: string; from?: string}[]) ?? [],
		cpu: (resources.cpu as string) ?? "",
		memory: (resources.memory as string) ?? "",
		disk: (resources.disk as string) ?? "",
	};
}

/** Map frontend RegisteredComponent → backend PipelineComponentAPI shape. */
function registeredToApi(comp: RegisteredComponent): Omit<PipelineComponentAPI, "createdAt" | "updatedAt"> {
	const idx = comp.image.lastIndexOf(":");
	const image = idx > 0 ? comp.image.slice(0, idx) : comp.image;
	const tag = idx > 0 ? comp.image.slice(idx + 1) : "latest";
	return {
		id: comp.id,
		name: comp.name,
		description: "",
		image,
		tag,
		source: "manual",
		inputPorts: [{ name: "input", type: "string" }],
		outputPorts: [{ name: "output", type: "string" }],
		resources: {
			command: comp.command,
			args: comp.args,
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
): Node<PipelineNodeData> {
	nodeCounter++;
	return {
		id: `step-${nodeCounter}`,
		type: "pipelineStep",
		position: { x, y },
		data: {
			label: comp.name,
			image: comp.image,
			command: comp.command,
			args: comp.args || [],
			cpu: comp.cpu || "",
			memory: comp.memory || "",
			disk: comp.disk || "",
		},
	};
}

function PipelineCanvas() {
	const navigate = useNavigate();
	const wrapperRef = useRef<HTMLDivElement>(null);
	const [nodes, setNodes, onNodesChange] = useNodesState<Node<PipelineNodeData>>([]);
	const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
	const reactFlow = useReactFlow();
	const [pipelineName, setPipelineName] = useState("my-pipeline");
	const [selectedNode, setSelectedNode] = useState<Node<PipelineNodeData> | null>(null);
	const [registeredComponents, setRegisteredComponents] = useState<RegisteredComponent[]>(loadComponents);
	const [view, setView] = useState<"pipeline" | "components" | "deploy">("pipeline");
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
	const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);

	useEffect(() => {
		saveComponents(registeredComponents);
	}, [registeredComponents]);

	// Load registered components from API on startup, fallback to localStorage
	useEffect(() => {
		listComponents()
			.then((res) => {
				const mapped = (res.items ?? []).map(apiToRegistered);
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
			} catch {
				// API failed — still keep changes in localStorage via existing effect
			}
		},
		[],
	);

	const handleComponentDelete = useCallback(async (id: string) => {
		try {
			await deleteComponent(id);
		} catch {
			// API failed — still keep changes in localStorage
		}
	}, []);

	const loadPipelineToCanvas = useCallback((pipeline: Pipeline) => {
		const { nodes: n, edges: e } = fromTranspilerPipeline(pipeline);
		setNodes(n);
		setEdges(e);
		if (pipeline.name) setPipelineName(pipeline.name);
		setView("pipeline");
		setSelectedNode(null);
		setJsonOutput(null);
	}, [setNodes, setEdges]);

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

	const onConnect = useCallback(
		(connection: Connection) => setEdges((eds) => addEdge(connection, eds)),
		[setEdges],
	);

	const onDragStart = useCallback(
		(e: DragEvent, comp: RegisteredComponent) => {
			e.dataTransfer.setData("application/reactflow", JSON.stringify(comp));
			e.dataTransfer.effectAllowed = "move";
		},
		[],
	);

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
				const position = reactFlow.screenToFlowPosition({
					x: event.clientX,
					y: event.clientY,
				});
				const newNode = createPipelineNode(comp, position.x, position.y);
				setNodes((nds) => nds.concat(newNode));
			} catch {
				/* ignore */
			}
		},
		[reactFlow, setNodes],
	);

	const onNodeClick = useCallback(
		(_: React.MouseEvent, node: Node) => setSelectedNode(node as Node<PipelineNodeData>),
		[],
	);
	const onPaneClick = useCallback(() => setSelectedNode(null), []);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			setNodes((nds) =>
				nds.map((n) => (n.id === id ? { ...n, data: { ...n.data, ...data } } : n)),
			);
			setSelectedNode((prev) =>
				prev?.id === id ? { ...prev, data: { ...prev.data, ...data } } : prev,
			);
		},
		[setNodes],
	);

	const buildPipelineJSON = useCallback(
		(): Pipeline => toTranspilerPipeline(nodes, edges, { name: pipelineName }),
		[nodes, edges, pipelineName],
	);

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
			if (pipeline.name) {
				setPipelineName(pipeline.name);
			}
			setJsonOutput(null);
		} catch {
			message.error("无效的 JSON");
		}
	}, [setNodes, setEdges]);

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
				setJsonOutput(null);
			},
		});
	}, [setNodes, setEdges]);

	const handleSave = useCallback(async () => {
		try {
			const result = await savePipeline(pipelineName, buildPipelineJSON());
			message.success(`已保存: ${result.name}`);
		} catch (err) {
			message.error("保存失败: " + String(err));
		}
	}, [pipelineName, buildPipelineJSON]);

	const openDeployDialog = useCallback(() => {
		setDeployDialog({
			open: true,
			deploying: false,
			done: false,
			name: pipelineName,
		});
		setSelectedAssetIds([]);
	}, [pipelineName]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
		});
	}, []);

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
								borderBottom: view === tab ? "2px solid #2563eb" : "2px solid transparent",
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
							<Button
								size="small"
								type="primary"
								icon={<PlayCircleOutlined />}
								onClick={openDeployDialog}
							>
								部署
							</Button>
							<Button
								size="small"
								icon={<SaveOutlined />}
								onClick={handleSave}
							>
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
							<ReactFlow
								nodes={nodes}
								edges={edges}
								onNodesChange={onNodesChange}
								onEdgesChange={onEdgesChange}
								onConnect={onConnect}
								onDrop={onDrop}
								onDragOver={onDragOver}
								onNodeClick={onNodeClick}
								onPaneClick={onPaneClick}
								nodeTypes={nodeTypes}
								fitView
							>
								<Background
									variant={BackgroundVariant.Dots}
									gap={24}
									color="#d4c9bc"
								/>
								<Controls />
								<MiniMap />
							</ReactFlow>
						</div>
						<aside className="config-panel">
							{selectedNode ? (
								<NodeConfigPanel node={selectedNode} onUpdate={updateNodeData} />
							) : (
								<div className="config-empty">选择一个节点进行配置</div>
							)}
						</aside>
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
						<div
							style={{
								display: "flex",
								flexDirection: "column",
								gap: 12,
								marginBottom: 16,
							}}
						>
							<label
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
								disabled={nodes.length === 0}
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
										{new Date(
											deployDialog.result.createdAt,
										).toLocaleString()}
									</span>
								</div>
								<div style={{ marginTop: 16, display: "flex", gap: 8, justifyContent: "center" }}>
									<Button
										type="primary"
										onClick={() => {
											closeDeployDialog();
											navigate("/workflows/" + deployDialog.result!.workflowName);
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
									<Button
										onClick={closeDeployDialog}
									>
										关闭
									</Button>
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
		<ReactFlowProvider>
			<PipelineCanvas />
		</ReactFlowProvider>
	);
}
