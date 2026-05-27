import type { Node as ReactFlowNode, NodeTypes } from "@xyflow/react";
import {
	Background,
	BackgroundVariant,
	Controls,
	type Edge,
	MiniMap,
	ReactFlow,
	ReactFlowProvider,
	addEdge,
	useEdgesState,
	useNodesState,
	useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
	Button,
	Input,
	Modal,
	message,
	Spin,
	Tabs,
	Typography,
} from "antd";
import { useCallback, useEffect, useRef, useState } from "react";
import { ComponentPalette } from "../components/pipeline/ComponentPalette";
import { ComponentManager } from "../components/pipeline/ComponentManager";
import { DeployPanel } from "../components/pipeline/DeployPanel";
import { NodeConfigPanel } from "../components/pipeline/NodeConfigPanel";
import { PipelineStepNode } from "../components/pipeline/PipelineNode";
import type { RegisteredComponent } from "../components/pipeline/types";
import type { PipelineNodeData } from "../components/pipeline/types";
import type { Pipeline } from "../components/pipeline/types";
import "../styles/pipeline.css";
import * as api from "../api/pipelineApi";

const { Text } = Typography;

const STORAGE_KEY = "databrew-components";

function loadComponents(): RegisteredComponent[] {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : getDefaultComponents();
	} catch {
		return getDefaultComponents();
	}
}

function saveComponents(comps: RegisteredComponent[]) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(comps));
}

function getDefaultComponents(): RegisteredComponent[] {
	return [
		{
			id: "c1",
			name: "BusyBox",
			image: "busybox:latest",
			command: ["sh", "-c"],
			args: [{ name: "script", value: "echo hello" }],
			cpu: "",
			memory: "",
			disk: "",
		},
		{
			id: "c2",
			name: "Python",
			image: "python:3.12-slim",
			command: ["python", "-c"],
			args: [{ name: "script", value: 'print("hello")' }],
			cpu: "",
			memory: "",
			disk: "",
		},
		{
			id: "c3",
			name: "Alpine",
			image: "alpine:latest",
			command: ["sh", "-c"],
			args: [{ name: "script", value: "echo hello" }],
			cpu: "",
			memory: "",
			disk: "",
		},
	];
}

const nodeTypes: NodeTypes = { pipelineStep: PipelineStepNode };

let nodeCounter = 0;

function createPipelineNode(
	comp: RegisteredComponent,
	x: number,
	y: number,
): ReactFlowNode<PipelineNodeData> {
	nodeCounter++;
	const id = `step-${nodeCounter}`;
	return {
		id,
		type: "pipelineStep",
		position: { x, y },
		data: {
			label: comp.name,
			image: comp.image,
			command: comp.command,
			args: comp.args || [],
			cpu: comp.cpu,
			memory: comp.memory,
			disk: comp.disk,
		},
	};
}

function PipelineCanvas() {
	const wrapperRef = useRef<HTMLDivElement>(null);
	const [nodes, setNodes, onNodesChange] = useNodesState<ReactFlowNode<PipelineNodeData>>([]);
	const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
	const reactFlow = useReactFlow();
	const [pipelineName, setPipelineName] = useState("my-pipeline");
	const [selectedNode, setSelectedNode] = useState<ReactFlowNode<PipelineNodeData> | null>(null);
	const [jsonOutput, setJsonOutput] = useState<string | null>(null);
	const [registeredComponents] = useState<RegisteredComponent[]>(
		loadComponents,
	);
	const [deployDialog, setDeployDialog] = useState<{
		open: boolean;
		deploying: boolean;
		done: boolean;
		name: string;
		result?: api.Deployment;
		error?: string;
	}>({ open: false, deploying: false, done: false, name: "" });

	useEffect(() => {
		saveComponents(registeredComponents);
	}, [registeredComponents]);

	const onConnect = useCallback(
		(connection: any) => setEdges((eds) => addEdge(connection, eds)),
		[setEdges],
	);

	const onDragStart = useCallback((e: React.DragEvent, comp: RegisteredComponent) => {
		e.dataTransfer.setData("application/reactflow", JSON.stringify(comp));
		e.dataTransfer.effectAllowed = "move";
	}, []);

	const onDragOver = useCallback((event: React.DragEvent) => {
		event.preventDefault();
		event.dataTransfer.dropEffect = "move";
	}, []);

	const onDrop = useCallback(
		(event: React.DragEvent) => {
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
		(_: React.MouseEvent, node: ReactFlowNode) => setSelectedNode(node as ReactFlowNode<PipelineNodeData>),
		[],
	);
	const onPaneClick = useCallback(() => setSelectedNode(null), []);

	const updateNodeData = useCallback(
		(id: string, data: Record<string, unknown>) => {
			setNodes((nds) =>
				nds.map((n) =>
					n.id === id ? { ...n, data: { ...n.data, ...data } } : n,
				),
			);
			setSelectedNode((prev) =>
				prev?.id === id ? { ...prev, data: { ...prev.data, ...data } } : prev,
			);
		},
		[setNodes],
	);

	const exportPipeline = useCallback(() => {
		const pipeline: Pipeline = {
			name: pipelineName,
			version: "1",
			nodes: nodes.map((n) => ({
				id: n.id,
				component: {
					name: n.data.label || "",
					image: n.data.image || "",
					command: n.data.command || [],
					args: n.data.args || [],
					resources:
						n.data.cpu || n.data.memory || n.data.disk
							? {
									cpu: n.data.cpu,
									memory: n.data.memory,
									disk: n.data.disk,
								}
							: undefined,
				},
				outputs: [],
			})),
			edges: edges.map((e) => ({ source: e.source, target: e.target })),
		};
		setJsonOutput(JSON.stringify(pipeline, null, 2));
	}, [nodes, edges, pipelineName]);

	const importPipeline = useCallback(() => {
		const text = prompt("粘贴 Pipeline JSON:");
		if (!text) return;
		try {
			const pipeline: Pipeline = JSON.parse(text);
			setNodes(
				pipeline.nodes.map((pn, i) => ({
					id: pn.id,
					type: "pipelineStep" as const,
					position: { x: 100 + i * 50, y: 100 + i * 80 },
					data: {
						label: pn.component.name,
						image: pn.component.image,
						command: pn.component.command || [],
						args: pn.component.args || [],
						cpu: pn.component.resources?.cpu || "",
						memory: pn.component.resources?.memory || "",
						disk: pn.component.resources?.disk || "",
					},
				})),
			);
			setEdges(
				pipeline.edges.map((pe, i) => ({
					id: `e-${i}`,
					source: pe.source,
					target: pe.target,
				})),
			);
			setJsonOutput(null);
		} catch {
			message.error("无效的 JSON");
		}
	}, [setNodes, setEdges]);

	const clearCanvas = useCallback(() => {
		setNodes([]);
		setEdges([]);
		setSelectedNode(null);
		setJsonOutput(null);
	}, [setNodes, setEdges]);

	const openDeployDialog = useCallback(() => {
		setDeployDialog({ open: true, deploying: false, done: false, name: pipelineName });
	}, [pipelineName]);

	const closeDeployDialog = useCallback(() => {
		setDeployDialog({
			open: false,
			deploying: false,
			done: false,
			name: "",
			result: undefined,
			error: undefined,
		});
	}, []);

	const handleDeploy = useCallback(async () => {
		setDeployDialog((prev) => ({ ...prev, deploying: true, done: false, error: undefined }));
		try {
			const pipeline: Pipeline = {
				name: pipelineName,
				version: "1",
				nodes: nodes.map((n) => ({
					id: n.id,
					component: {
						name: n.data.label || "",
						image: n.data.image || "",
						command: n.data.command || [],
						args: n.data.args || [],
						resources:
							n.data.cpu || n.data.memory || n.data.disk
								? {
										cpu: n.data.cpu,
										memory: n.data.memory,
										disk: n.data.disk,
									}
								: undefined,
					},
					outputs: [],
				})),
				edges: edges.map((e) => ({ source: e.source, target: e.target })),
			};
			const result = await api.deploy(pipeline, deployDialog.name || pipelineName);
			setDeployDialog((prev) => ({ ...prev, deploying: false, done: true, result }));
		} catch (err) {
			setDeployDialog((prev) => ({
				...prev,
				deploying: false,
				done: true,
				error: String(err),
			}));
		}
	}, [nodes, edges, pipelineName, deployDialog.name]);

	const handleSaveTemplate = useCallback(async () => {
		const name = prompt("流水线模板名称:", pipelineName);
		if (!name) return;
		const pipeline: Pipeline = {
			name: pipelineName,
			version: "1",
			nodes: nodes.map((n) => ({
				id: n.id,
				component: {
					name: n.data.label || "",
					image: n.data.image || "",
					command: n.data.command || [],
					args: n.data.args || [],
				},
				outputs: [],
			})),
			edges: edges.map((e) => ({ source: e.source, target: e.target })),
		};
		try {
			await api.savePipeline(name, pipeline);
			message.success("模板已保存");
		} catch (err) {
			message.error("保存失败: " + String(err));
		}
	}, [nodes, edges, pipelineName]);

	return (
		<>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 8,
					padding: "8px 0",
					flexShrink: 0,
					flexWrap: "wrap",
				}}
			>
				<Input
					style={{ width: 200, fontFamily: "monospace", fontSize: 12 }}
					value={pipelineName}
					onChange={(e) => setPipelineName(e.target.value)}
					placeholder="pipeline-name"
				/>
				<Button type="primary" size="small" onClick={openDeployDialog}>
					部署
				</Button>
				<Button size="small" onClick={handleSaveTemplate}>
					保存
				</Button>
				<Button size="small" onClick={exportPipeline}>
					导出
				</Button>
				<Button size="small" onClick={importPipeline}>
					导入
				</Button>
				<Button size="small" danger onClick={clearCanvas}>
					清空
				</Button>
			</div>
			<div className="pipeline-body">
				<ComponentPalette components={registeredComponents} onDragStart={onDragStart} />
				<div className="canvas-wrapper" ref={wrapperRef}>
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
							color="#cbd5e1"
						/>
						<Controls />
						<MiniMap />
					</ReactFlow>
				</div>
				<aside className="config-panel">
					{selectedNode ? (
						<NodeConfigPanel node={selectedNode} onUpdate={updateNodeData} />
					) : (
						<div className="config-empty">选择节点进行配置</div>
					)}
				</aside>
			</div>

			{jsonOutput && <pre className="json-output">{jsonOutput}</pre>}

			{/* Deploy Dialog */}
			<Modal
				open={deployDialog.open}
				onCancel={closeDeployDialog}
				footer={null}
				title="部署流水线"
				width={480}
			>
				{!deployDialog.deploying && !deployDialog.done && (
					<>
						<Text
							type="secondary"
							style={{ fontSize: 12, display: "block", marginBottom: 14 }}
						>
							将编译流水线并提交到 Kubernetes 集群。
						</Text>
						<div className="deploy-dialog-fields">
							<label>
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
								/>
							</label>
						</div>
						<div className="deploy-dialog-actions">
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
					<div className="deploy-progress">
						<Spin />
						<Text type="secondary">正在部署流水线...</Text>
					</div>
				)}
				{deployDialog.done && (
					<div className="deploy-result">
						{deployDialog.error ? (
							<>
								<div className="dep-empty" style={{ color: "#dc2626" }}>
									部署失败
								</div>
								<Text
									type="secondary"
									style={{
										fontSize: 12,
										display: "block",
										marginTop: 8,
										fontFamily: "monospace",
										whiteSpace: "pre-wrap",
									}}
								>
									{deployDialog.error}
								</Text>
							</>
						) : deployDialog.result ? (
							<>
								<div style={{ color: "#16a34a", fontWeight: 600, fontSize: 16 }}>
									部署成功
								</div>
								<Text
									code
									style={{ display: "block", marginTop: 8, fontSize: 13 }}
								>
									{deployDialog.result.workflowName}
								</Text>
								<div className="dep-card-meta" style={{ justifyContent: "center", marginTop: 8 }}>
									<span>{deployDialog.result.nodes} 个节点</span>
									<span className="dot">•</span>
									<span>{new Date(deployDialog.result.createdAt).toLocaleString()}</span>
								</div>
							</>
						) : null}
						<div style={{ marginTop: 16 }}>
							<Button onClick={closeDeployDialog}>关闭</Button>
						</div>
					</div>
				)}
			</Modal>
		</>
	);
}

export default function PipelinePage() {
	const [tab, setTab] = useState("pipeline");
	const [registeredComponents, setRegisteredComponents] = useState<RegisteredComponent[]>(
		loadComponents,
	);

	useEffect(() => {
		saveComponents(registeredComponents);
	}, [registeredComponents]);

	const tabItems = [
		{
			key: "pipeline",
			label: "流水线设计",
			children: (
				<div style={{ height: "calc(100vh - 220px)" }}>
					<ReactFlowProvider>
						<PipelineCanvas />
					</ReactFlowProvider>
				</div>
			),
		},
		{
			key: "registry",
			label: "组件注册表",
			children: (
				<ComponentManager
					components={registeredComponents}
					onChange={setRegisteredComponents}
				/>
			),
		},
		{
			key: "deploy",
			label: "部署管理",
			children: <DeployPanel />,
		},
	];

	return (
		<div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
			<Typography.Title level={4} style={{ margin: "0 0 12px 0", flexShrink: 0 }}>
				流水线设计器
			</Typography.Title>
			<Tabs
				activeKey={tab}
				onChange={setTab}
				items={tabItems}
				style={{ flex: 1, display: "flex", flexDirection: "column" }}
				tabBarStyle={{ marginBottom: 8 }}
			/>
		</div>
	);
}
