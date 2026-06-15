import {
	DeleteOutlined,
	EditOutlined,
	EyeOutlined,
	PlusOutlined,
	ReloadOutlined,
	SyncOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Descriptions,
	Empty,
	Form,
	Input,
	Modal,
	message,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { InputRef } from "antd/es/input";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
	createComponent,
	deleteComponent,
	listComponentReleases,
	listComponents,
	type PipelineComponentAPI,
	type PipelineComponentPayload,
	type PipelineComponentReleaseAPI,
	type PipelineComponentReleasePayload,
	type PipelineComponentType,
	type PortDef,
	syncComponentReleases,
	updateComponent,
} from "../api/pipelineComponentApi";
import { toAssetStyleId } from "../lib/idDisplay";
import {
	dedupePipelineComponentsByName,
	formatComponentImage,
} from "../lib/pipelineComponentDisplay";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "../lib/pipelineContract";
import { withSelectAllColumn } from "../lib/tableSelection";

type EnvRow = { name?: string; value?: string };
type PortRow = {
	name?: string;
	type?: string;
	desc?: string;
	default_value?: string;
};

type ModalMode = "create" | "edit" | "view";

type SyncReleaseBody =
	| { items?: PipelineComponentReleasePayload[] }
	| PipelineComponentReleasePayload[];

interface ComponentLibraryRow {
	key: string;
	name: string;
	componentId: string;
	legacyComponent?: PipelineComponentAPI;
	releases: PipelineComponentReleaseAPI[];
	primaryRelease?: PipelineComponentReleaseAPI;
	updatedAt?: string;
	searchText: string;
}

interface ComponentFormValues {
	name: string;
	type: PipelineComponentType;
	image: string;
	tag?: string;
	description?: string;
	command?: string;
	args?: string;
	cpu?: string;
	memory?: string;
	disk?: string;
	gpu?: string;
	computeTier?: string;
	envRows?: EnvRow[];
	inputPorts?: PortRow[];
	outputPorts?: PortRow[];
}

const TYPE_OPTIONS: Array<{ label: string; value: PipelineComponentType }> = [
	{ label: "Container", value: "container" },
	{ label: "Script", value: "script" },
	{ label: "Resource", value: "resource" },
	{ label: "Suspend", value: "suspend" },
];

const TYPE_COLORS: Record<PipelineComponentType, string> = {
	container: "blue",
	script: "purple",
	resource: "green",
	suspend: "orange",
};

const DEFAULT_INPUT_PORTS: PortDef[] = [{ name: "input", type: "asset" }];
const DEFAULT_OUTPUT_PORTS: PortDef[] = [{ name: "output", type: "asset" }];

const formatDateTime = (value?: string): string =>
	value ? new Date(value).toLocaleString() : "-";

const shortTechnicalValue = (value?: string): string => {
	if (!value) return "-";
	if (value.length <= 18) return value;
	return `${value.slice(0, 10)}...${value.slice(-8)}`;
};

const releaseStatusColor = (status?: string): string => {
	switch (status) {
		case "ready":
			return "green";
		case "pending":
			return "blue";
		case "deprecated":
			return "default";
		case "failed":
			return "red";
		default:
			return "default";
	}
};

const validationStatusColor = (status?: string): string =>
	status === "passed" ? "green" : "red";

const componentKey = (value?: string): string =>
	(value || "").trim().toLowerCase();

const releaseDisplayName = (release: PipelineComponentReleaseAPI): string =>
	release.displayName || release.taskName || release.componentId;

const compareDateDesc = (a?: string, b?: string): number =>
	new Date(b || 0).getTime() - new Date(a || 0).getTime();

const splitInputItems = (value?: string): string[] =>
	(value || "")
		.split(/[\n,]/)
		.map((v) => v.trim())
		.filter(Boolean);

const joinInputItems = (value?: string[]): string =>
	(value || [])
		.map((item) => item.trim())
		.filter(Boolean)
		.join(", ");

const resourceString = (
	resources: Record<string, unknown> | undefined,
	key: string,
): string => {
	const value = resources?.[key];
	return typeof value === "string" ? value : "";
};

function normalizePortRows(
	rows: PortRow[] | undefined,
	fallback: PortDef[],
): PortDef[] {
	if (!rows || rows.length === 0) return fallback;
	const seen = new Set<string>();
	const next: PortDef[] = [];
	for (const row of rows) {
		const name = row.name?.trim();
		if (!name || seen.has(name)) continue;
		seen.add(name);
		next.push({
			name,
			type: row.type?.trim() || "string",
			...(row.desc?.trim() ? { desc: row.desc.trim() } : {}),
			...(row.default_value?.trim()
				? { default_value: row.default_value.trim() }
				: {}),
		});
	}
	return next.length > 0 ? next : fallback;
}

function toFormValues(component?: PipelineComponentAPI): ComponentFormValues {
	if (!component) {
		return {
			name: "",
			type: "container",
			image: "",
			tag: "latest",
			description: "",
			command: "",
			args: "",
			cpu: "",
			memory: "",
			disk: "",
			gpu: "",
			computeTier: "",
			envRows: [],
			inputPorts: DEFAULT_INPUT_PORTS,
			outputPorts: DEFAULT_OUTPUT_PORTS,
		};
	}
	const resources = component.resources || {};

	return {
		name: component.name,
		type: component.type || "container",
		image: component.image,
		tag: component.tag || "latest",
		description: component.description || "",
		command: joinInputItems(component.command || []),
		args: joinInputItems(component.args || []),
		cpu: resourceString(resources, "cpu"),
		memory: resourceString(resources, "memory"),
		disk: resourceString(resources, "disk"),
		gpu: resourceString(resources, "gpu"),
		computeTier: resourceString(resources, "computeTier"),
		envRows: Object.entries(component.env || {}).map(([name, value]) => ({
			name,
			value,
		})),
		inputPorts: normalizePortRows(component.inputPorts, DEFAULT_INPUT_PORTS),
		outputPorts: normalizePortRows(component.outputPorts, DEFAULT_OUTPUT_PORTS),
	};
}

function toPayload(
	values: ComponentFormValues,
	componentSource?: string,
): PipelineComponentPayload {
	const env: Record<string, string> = {};
	for (const row of values.envRows || []) {
		const name = row.name?.trim();
		if (name) env[name] = row.value || "";
	}

	const command = splitInputItems(values.command);
	const args = normalizeShellCommandArgs(
		command,
		normalizeComponentArgs(splitInputItems(values.args)),
	)
		.map((arg) => arg.value || arg.name)
		.filter(Boolean);

	return {
		name: values.name.trim(),
		type: values.type,
		image: values.image.trim(),
		tag: values.tag?.trim() || "latest",
		source: componentSource || "custom",
		description: values.description?.trim() || "",
		command,
		args,
		env,
		inputPorts: normalizePortRows(values.inputPorts, DEFAULT_INPUT_PORTS),
		outputPorts: normalizePortRows(values.outputPorts, DEFAULT_OUTPUT_PORTS),
		resources: {
			type: values.type,
			command,
			args,
			env,
			...(values.cpu?.trim() ? { cpu: values.cpu.trim() } : {}),
			...(values.memory?.trim() ? { memory: values.memory.trim() } : {}),
			...(values.disk?.trim() ? { disk: values.disk.trim() } : {}),
			...(values.gpu?.trim() ? { gpu: values.gpu.trim() } : {}),
			...(values.computeTier?.trim()
				? { computeTier: values.computeTier.trim() }
				: {}),
		},
	};
}

function ComponentDetail({ component }: { component: PipelineComponentAPI }) {
	const envEntries = Object.entries(component.env || {});
	const imageText = formatComponentImage(component.image, component.tag);
	const resources = component.resources || {};

	return (
		<Space direction="vertical" size={16} style={{ width: "100%" }}>
			<Descriptions bordered column={2} size="small">
				<Descriptions.Item label="名称" span={2}>
					<Typography.Text strong>{component.name}</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="ID" span={2}>
					<Typography.Text copyable={{ text: component.id }}>
						{toAssetStyleId(component.id)}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="类型">
					<Tag color={TYPE_COLORS[component.type] || "default"}>
						{component.type || "container"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="来源">
					<Tag color={component.source === "system" ? "gold" : "default"}>
						{component.source || "-"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="镜像" span={2}>
					<Typography.Text copyable={{ text: imageText }}>
						{imageText || "-"}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="描述" span={2}>
					{component.description || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="创建时间">
					{formatDateTime(component.createdAt)}
				</Descriptions.Item>
				<Descriptions.Item label="更新时间">
					{formatDateTime(component.updatedAt)}
				</Descriptions.Item>
			</Descriptions>

			<Descriptions bordered column={1} size="small">
				<Descriptions.Item label="命令">
					{component.command?.length ? component.command.join(", ") : "-"}
				</Descriptions.Item>
				<Descriptions.Item label="参数">
					{component.args?.length ? component.args.join(", ") : "-"}
				</Descriptions.Item>
				<Descriptions.Item label="环境变量">
					{envEntries.length > 0 ? (
						<Space direction="vertical" size={4}>
							{envEntries.map(([name, value]) => (
								<Typography.Text key={name} code>
									{name}={value}
								</Typography.Text>
							))}
						</Space>
					) : (
						"-"
					)}
				</Descriptions.Item>
				<Descriptions.Item label="输入端口">
					{component.inputPorts?.length ? (
						<Space wrap size={4}>
							{component.inputPorts.map((port) => (
								<Tag key={port.name}>
									{port.name}:{port.type}
								</Tag>
							))}
						</Space>
					) : (
						"-"
					)}
				</Descriptions.Item>
				<Descriptions.Item label="输出端口">
					{component.outputPorts?.length ? (
						<Space direction="vertical" size={4}>
							{component.outputPorts.map((port) => (
								<Typography.Text key={port.name} code>
									{port.name}:{port.type} /tmp/outputs/{port.name}
								</Typography.Text>
							))}
						</Space>
					) : (
						"-"
					)}
				</Descriptions.Item>
				<Descriptions.Item label="资源">
					<Space wrap size={4}>
						<Tag>CPU: {resourceString(resources, "cpu") || "-"}</Tag>
						<Tag>内存: {resourceString(resources, "memory") || "-"}</Tag>
						<Tag>磁盘: {resourceString(resources, "disk") || "-"}</Tag>
						<Tag>GPU: {resourceString(resources, "gpu") || "-"}</Tag>
						<Tag>
							计算档位: {resourceString(resources, "computeTier") || "-"}
						</Tag>
					</Space>
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

function ReleaseDetail({ release }: { release: PipelineComponentReleaseAPI }) {
	return (
		<Space direction="vertical" size={16} style={{ width: "100%" }}>
			<Descriptions bordered column={2} size="small">
				<Descriptions.Item label="任务" span={2}>
					<Typography.Text strong>
						{release.displayName || release.taskName || release.componentId}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="版本">
					<Tag color={release.channel === "prod" ? "green" : "blue"}>
						{release.releaseLabel}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="可选择">
					<Tag color={release.selectable ? "green" : "red"}>
						{release.selectable ? "是" : "否"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="状态">
					<Tag color={releaseStatusColor(release.status)}>{release.status}</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="校验">
					<Tag color={validationStatusColor(release.validationStatus)}>
						{release.validationStatus}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="组件 ID" span={2}>
					<Typography.Text copyable>{release.componentId}</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="镜像" span={2}>
					<Typography.Text copyable={{ text: release.runtimeImage }}>
						{release.runtimeImage || "-"}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="命令" span={2}>
					{release.runtimeSnapshot?.command?.join(" ") || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="资源" span={2}>
					<Space wrap>
						{Object.entries(release.runtimeSnapshot?.resources || {}).map(
							([key, value]) => (
								<Tag key={key}>
									{key}: {String(value)}
								</Tag>
							),
						)}
					</Space>
				</Descriptions.Item>
			</Descriptions>

			<Descriptions bordered column={1} size="small" title="技术详情">
				<Descriptions.Item label="Release ID">
					<Typography.Text copyable>{release.id}</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="任务路径">
					{release.taskPath || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="Source Repo">
					{release.sourceRepo || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="Source Ref">
					{release.sourceRef || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="Source Commit">
					<Typography.Text copyable={{ text: release.sourceCommit || "" }}>
						{release.sourceCommit || "-"}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="Image Digest">
					<Typography.Text copyable={{ text: release.imageDigest || "" }}>
						{release.imageDigest || "-"}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="校验错误">
					{release.validationErrors?.length ? (
						<Space direction="vertical" size={4}>
							{release.validationErrors.map((item) => (
								<Typography.Text key={item} type="danger">
									{item}
								</Typography.Text>
							))}
						</Space>
					) : (
						"-"
					)}
				</Descriptions.Item>
			</Descriptions>
		</Space>
	);
}

function PortFormList({
	name,
	title,
	emptyPort,
	showOutputPath = false,
}: {
	name: "inputPorts" | "outputPorts";
	title: string;
	emptyPort: PortRow;
	showOutputPath?: boolean;
}) {
	return (
		<Form.List name={name}>
			{(fields, { add, remove }) => (
				<div
					style={{
						border: "1px solid #e2e8f0",
						borderRadius: 8,
						padding: 12,
						background: "#fff",
						minWidth: 0,
					}}
				>
					<div
						style={{
							display: "flex",
							alignItems: "center",
							justifyContent: "space-between",
							gap: 12,
							marginBottom: 12,
						}}
					>
						<div>
							<Typography.Text strong>{title}</Typography.Text>
							<Typography.Text
								type="secondary"
								style={{ display: "block", fontSize: 12, marginTop: 2 }}
							>
								{showOutputPath
									? "这个步骤产出的结果，可连接给后续步骤使用。"
									: "这个步骤运行前需要接收的数据。"}
							</Typography.Text>
						</div>
						<Button
							size="small"
							icon={<PlusOutlined />}
							onClick={() => add(emptyPort)}
						>
							添加端口
						</Button>
					</div>
					{fields.map(({ key, ...field }) => (
						<div
							key={key}
							style={{
								display: "grid",
								gridTemplateColumns:
									"minmax(120px, 1fr) minmax(96px, 140px) minmax(160px, 1.4fr) 32px",
								gap: 8,
								alignItems: "center",
								marginBottom: 8,
								maxWidth: "100%",
								minWidth: 0,
							}}
						>
							<Form.Item
								{...field}
								name={[field.name, "name"]}
								rules={[{ required: true, message: "请输入端口名" }]}
								style={{ marginBottom: 0, minWidth: 0 }}
							>
								<Input placeholder={showOutputPath ? "output" : "input"} />
							</Form.Item>
							<Form.Item
								{...field}
								name={[field.name, "type"]}
								rules={[{ required: true, message: "请输入类型" }]}
								style={{ marginBottom: 0, minWidth: 0 }}
							>
								<Input placeholder="asset" />
							</Form.Item>
							<Form.Item
								{...field}
								name={[field.name, "desc"]}
								style={{ marginBottom: 0, minWidth: 0 }}
							>
								<Input
									placeholder={
										showOutputPath ? "结果说明，可选" : "端口说明，可选"
									}
								/>
							</Form.Item>
							<Button
								type="text"
								danger
								icon={<DeleteOutlined />}
								aria-label={`移除${title}`}
								onClick={() => remove(field.name)}
							/>
						</div>
					))}
					{showOutputPath ? (
						<Typography.Text type="secondary" style={{ fontSize: 12 }}>
							要把结果传给后续步骤时，请把结果写入对应文件。例如输出名为{" "}
							<Typography.Text code>output</Typography.Text>
							，就写入{" "}
							<Typography.Text code>/tmp/outputs/output</Typography.Text>。
						</Typography.Text>
					) : null}
				</div>
			)}
		</Form.List>
	);
}

export function ComponentManager() {
	const [items, setItems] = useState<PipelineComponentAPI[]>([]);
	const [releases, setReleases] = useState<PipelineComponentReleaseAPI[]>([]);
	const [loading, setLoading] = useState(true);
	const [releaseLoading, setReleaseLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [syncing, setSyncing] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [releaseError, setReleaseError] = useState<string | null>(null);
	const [search, setSearch] = useState("");
	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<ModalMode | null>(null);
	const [releaseModalOpen, setReleaseModalOpen] = useState(false);
	const [syncModalOpen, setSyncModalOpen] = useState(false);
	const [activeComponent, setActiveComponent] =
		useState<PipelineComponentAPI | null>(null);
	const [activeRelease, setActiveRelease] =
		useState<PipelineComponentReleaseAPI | null>(null);
	const [syncText, setSyncText] = useState("");
	const [selectedComponentIds, setSelectedComponentIds] = useState<string[]>(
		[],
	);
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
	const [bulkDeleteConfirmOpen, setBulkDeleteConfirmOpen] = useState(false);
	const [bulkDeleting, setBulkDeleting] = useState(false);
	const [form] = Form.useForm<ComponentFormValues>();
	const [messageApi, contextHolder] = message.useMessage();
	const nameInputRef = useRef<InputRef>(null);

	const refresh = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const res = await listComponents();
			const nextItems = dedupePipelineComponentsByName(res.items || []);
			setItems(nextItems);
			setSelectedComponentIds((prev) =>
				prev.filter((id) =>
					nextItems.some((item) => item.id === id && item.source !== "system"),
				),
			);
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			setError(detail);
		} finally {
			setLoading(false);
		}
	}, []);

	const refreshReleases = useCallback(async () => {
		setReleaseLoading(true);
		setReleaseError(null);
		try {
			const res = await listComponentReleases({ selectable: true });
			setReleases(res.items || []);
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			setReleaseError(detail);
		} finally {
			setReleaseLoading(false);
		}
	}, []);

	useEffect(() => {
		refresh();
		refreshReleases();
	}, [refresh, refreshReleases]);

	const libraryRows = useMemo(() => {
		const groups = new Map<string, PipelineComponentReleaseAPI[]>();
		for (const release of releases) {
			const key = componentKey(release.componentId || release.taskName);
			if (!key) continue;
			const group = groups.get(key) || [];
			group.push(release);
			groups.set(key, group);
		}
		for (const group of groups.values()) {
			group.sort((a, b) => compareDateDesc(a.updatedAt, b.updatedAt));
		}

		const rows: ComponentLibraryRow[] = items.map((component) => {
			const key = componentKey(component.name || component.id);
			const groupedReleases = groups.get(key) || [];
			groups.delete(key);
			const primaryRelease =
				groupedReleases.find((release) => release.selectable) ||
				groupedReleases[0];
			const updatedAt = primaryRelease?.updatedAt || component.updatedAt;
			const searchText = [
				component.name,
				component.id,
				component.image,
				component.tag,
				component.description,
				component.source,
				...groupedReleases.flatMap((release) => [
					release.componentId,
					release.taskName,
					release.displayName,
					release.releaseLabel,
					release.channel,
					release.runtimeImage,
					release.sourceCommit,
				]),
			]
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			return {
				key: `component:${component.id}`,
				name: component.name,
				componentId: component.name,
				legacyComponent: component,
				releases: groupedReleases,
				primaryRelease,
				updatedAt,
				searchText,
			};
		});

		for (const [key, groupedReleases] of groups) {
			const primaryRelease =
				groupedReleases.find((release) => release.selectable) ||
				groupedReleases[0];
			const name = primaryRelease ? releaseDisplayName(primaryRelease) : key;
			const searchText = groupedReleases
				.flatMap((release) => [
					release.componentId,
					release.taskName,
					release.displayName,
					release.releaseLabel,
					release.channel,
					release.runtimeImage,
					release.sourceCommit,
				])
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			rows.push({
				key: `release:${key}`,
				name,
				componentId: primaryRelease?.componentId || key,
				releases: groupedReleases,
				primaryRelease,
				updatedAt: primaryRelease?.updatedAt,
				searchText,
			});
		}

		const q = search.trim().toLowerCase();
		const filtered = q
			? rows.filter((row) => row.searchText.includes(q))
			: rows;
		return filtered.sort((a, b) => {
			if (a.primaryRelease && !b.primaryRelease) return -1;
			if (!a.primaryRelease && b.primaryRelease) return 1;
			return compareDateDesc(a.updatedAt, b.updatedAt);
		});
	}, [items, releases, search]);

	const isCreateMode = modalMode === "create";
	const isEditMode = modalMode === "edit";
	const isViewMode = modalMode === "view";
	const modalFormKey = `${modalMode || "closed"}:${activeComponent?.id || "new"}`;
	const modalFormInitialValues = useMemo(
		() => toFormValues(activeComponent || undefined),
		[activeComponent],
	);

	useEffect(() => {
		if (!modalOpen || isViewMode) return;
		form.resetFields();
		form.setFieldsValue(modalFormInitialValues);
	}, [form, isViewMode, modalFormInitialValues, modalOpen]);

	useEffect(() => {
		if (!modalOpen || !isCreateMode) return;
		// 等 Modal 进场动画完成后再聚焦
		const id = window.setTimeout(() => {
			nameInputRef.current?.focus();
		}, 200);

		return () => window.clearTimeout(id);
	}, [isCreateMode, modalOpen]);

	const openCreate = () => {
		setActiveComponent(null);
		setModalMode("create");
		setModalOpen(true);
	};

	const openEdit = (component: PipelineComponentAPI) => {
		setActiveComponent(component);
		setModalMode("edit");
		setModalOpen(true);
	};

	const openView = (component: PipelineComponentAPI) => {
		setActiveComponent(component);
		setModalMode("view");
		form.setFieldsValue(toFormValues(component));
		setModalOpen(true);
	};

	const openReleaseView = (release: PipelineComponentReleaseAPI) => {
		setActiveRelease(release);
		setReleaseModalOpen(true);
	};

	const closeModal = () => {
		setModalOpen(false);
		setModalMode(null);
		setActiveComponent(null);
		form.resetFields();
	};

	const closeReleaseModal = () => {
		setReleaseModalOpen(false);
		setActiveRelease(null);
	};

	const openSyncModal = () => {
		setSyncText(
			JSON.stringify(
				{
					items: [
						{
							componentId: "hand-detect-yolov26m",
							taskName: "hand-detect-yolov26m",
							taskPath: "tasks/hand_detect_yolov26m",
							releaseLabel: "main-abc1234",
							runtimeImage:
								"us-central1-docker.pkg.dev/project/video-proc-images/hand-detect-yolov26m@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							runtimeSnapshot: {
								command: ["python", "src/main.py"],
								inputPorts: [{ name: "input", type: "asset" }],
								outputPorts: [{ name: "output", type: "asset" }],
								resources: { cpu: "14000m", memory: "55Gi", gpu: "1" },
							},
						},
					],
				},
				null,
				2,
			),
		);
		setSyncModalOpen(true);
	};

	const handleSyncReleases = async () => {
		let parsed: SyncReleaseBody;
		try {
			parsed = JSON.parse(syncText) as SyncReleaseBody;
		} catch {
			messageApi.error("请输入合法 JSON");
			return;
		}
		const items = Array.isArray(parsed) ? parsed : parsed.items;
		if (!Array.isArray(items)) {
			messageApi.error("JSON 需要是数组，或包含 items 数组");
			return;
		}
		setSyncing(true);
		try {
			const res = await syncComponentReleases(items);
			const failed = (res.items || []).filter(
				(item) => item.validationStatus !== "passed",
			).length;
			messageApi.success(
				failed > 0
					? `已同步 ${res.items.length} 个版本，其中 ${failed} 个不可选`
					: `已同步 ${res.items.length} 个版本`,
			);
			setSyncModalOpen(false);
			await refreshReleases();
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			messageApi.error(detail);
		} finally {
			setSyncing(false);
		}
	};

	const handleSave = async () => {
		if (isViewMode) {
			closeModal();
			return;
		}
		let values: ComponentFormValues;
		try {
			values = await form.validateFields();
		} catch {
			return;
		}
		setSaving(true);
		try {
			const payload = toPayload(values, activeComponent?.source);
			if (activeComponent) {
				await updateComponent(activeComponent.id, payload);
				messageApi.success("组件已更新");
			} else {
				await createComponent(payload);
				messageApi.success("组件已创建");
			}
			closeModal();
			await refresh();
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			messageApi.error(detail);
		} finally {
			setSaving(false);
		}
	};

	const handleDelete = async (component: PipelineComponentAPI) => {
		try {
			await deleteComponent(component.id);
			messageApi.success("组件已删除");
			setConfirmDeleteId(null);
			await refresh();
		} catch (err) {
			const detail = err instanceof Error ? err.message : String(err);
			messageApi.error(detail);
		}
	};

	const handleBulkDelete = async () => {
		if (selectedComponentIds.length === 0) return;
		setBulkDeleting(true);
		try {
			const results = await Promise.allSettled(
				selectedComponentIds.map((id) => deleteComponent(id)),
			);
			const failedCount = results.filter(
				(result) => result.status === "rejected",
			).length;
			const deletedCount = results.length - failedCount;
			if (deletedCount > 0) messageApi.success(`已删除 ${deletedCount} 个组件`);
			if (failedCount > 0) messageApi.error(`${failedCount} 个组件删除失败`);
			setSelectedComponentIds([]);
			setBulkDeleteConfirmOpen(false);
			await refresh();
		} finally {
			setBulkDeleting(false);
		}
	};

	const libraryColumns: ColumnsType<ComponentLibraryRow> = [
		{
			title: "组件 / 任务",
			key: "name",
			width: 260,
			render: (_, record) => (
				<Space direction="vertical" size={0}>
					<Typography.Text strong ellipsis={{ tooltip: record.name }}>
						{record.name}
					</Typography.Text>
					<Space size={4} wrap>
						{record.primaryRelease ? <Tag color="green">版本库</Tag> : null}
						{record.legacyComponent ? (
							<Tag
								color={
									record.legacyComponent.source === "system"
										? "gold"
										: "default"
								}
							>
								旧组件
							</Tag>
						) : null}
						{record.componentId !== record.name ? (
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>
								{record.componentId}
							</Typography.Text>
						) : null}
					</Space>
					{record.legacyComponent ? (
						<Typography.Text
							type="secondary"
							copyable={{ text: record.legacyComponent.id }}
							style={{ fontSize: 12 }}
						>
							ID: {toAssetStyleId(record.legacyComponent.id)}
						</Typography.Text>
					) : null}
				</Space>
			),
		},
		{
			title: "版本",
			key: "versions",
			width: 260,
			render: (_, record) => (
				<Space wrap size={4}>
					{record.releases.slice(0, 4).map((release) => (
						<Tag
							key={release.id}
							color={release.selectable ? "green" : "red"}
							style={{ cursor: "pointer" }}
							onClick={() => openReleaseView(release)}
						>
							{release.releaseLabel}
						</Tag>
					))}
					{record.releases.length > 4 ? (
						<Tag>+{record.releases.length - 4}</Tag>
					) : null}
					{record.legacyComponent ? (
						<Tag color="default">
							旧版:{record.legacyComponent.tag || "latest"}
						</Tag>
					) : null}
					{record.releases.length === 0 && !record.legacyComponent ? "-" : null}
				</Space>
			),
		},
		{
			title: "状态",
			key: "status",
			width: 160,
			render: (_, record) => {
				const release = record.primaryRelease;
				return (
					<Space direction="vertical" size={0}>
						{release ? (
							<>
								<Tag color={releaseStatusColor(release.status)}>
									{release.status}
								</Tag>
								<Tag color={validationStatusColor(release.validationStatus)}>
									{release.validationStatus}
								</Tag>
							</>
						) : (
							<Tag color="default">仅旧组件</Tag>
						)}
					</Space>
				);
			},
		},
		{
			title: "运行镜像",
			key: "runtime",
			width: 260,
			render: (_, record) => {
				const release = record.primaryRelease;
				const imageText = release
					? release.runtimeImage
					: record.legacyComponent
						? formatComponentImage(
								record.legacyComponent.image,
								record.legacyComponent.tag,
							)
						: "";
				return (
					<Tooltip title={imageText || "-"}>
						<Typography.Text code ellipsis style={{ maxWidth: 240 }}>
							{release
								? shortTechnicalValue(release.imageDigest || imageText)
								: imageText || "-"}
						</Typography.Text>
					</Tooltip>
				);
			},
		},
		{
			title: "来源",
			key: "source",
			width: 210,
			render: (_, record) => (
				<Space direction="vertical" size={0}>
					<Typography.Text>
						{record.primaryRelease?.sourceCommit
							? shortTechnicalValue(record.primaryRelease.sourceCommit)
							: record.primaryRelease
								? "平台生成"
								: record.legacyComponent?.source || "-"}
					</Typography.Text>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{record.primaryRelease?.taskPath ||
							record.legacyComponent?.description ||
							"-"}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: "更新时间",
			key: "updatedAt",
			width: 160,
			render: (_, record) => formatDateTime(record.updatedAt),
		},
		{
			title: "操作",
			key: "actions",
			width: 260,
			render: (_, record) => {
				const component = record.legacyComponent;
				const isSystemComponent = component?.source === "system";
				const deleteButton = component ? (
					<Button
						type="link"
						size="small"
						danger
						disabled={isSystemComponent}
						icon={<DeleteOutlined />}
						aria-label="删除组件"
						onClick={() => {
							if (isSystemComponent) return;
							setConfirmDeleteId(component.id);
							setBulkDeleteConfirmOpen(false);
						}}
					>
						删除
					</Button>
				) : null;
				return (
					<Space size={4} style={{ whiteSpace: "nowrap" }}>
						{record.primaryRelease ? (
							<Button
								type="link"
								size="small"
								icon={<EyeOutlined />}
								onClick={() =>
									openReleaseView(
										record.primaryRelease as PipelineComponentReleaseAPI,
									)
								}
							>
								版本详情
							</Button>
						) : null}
						{component ? (
							<Button
								type="link"
								size="small"
								icon={<EyeOutlined />}
								aria-label="查看组件"
								onClick={() => openView(component)}
							>
								组件
							</Button>
						) : null}
						{component ? (
							<Button
								type="link"
								size="small"
								icon={<EditOutlined />}
								aria-label="编辑组件"
								onClick={() => openEdit(component)}
							>
								编辑
							</Button>
						) : null}
						{component && deleteButton ? (
							isSystemComponent ? (
								<Tooltip title="系统来源组件禁止删除">{deleteButton}</Tooltip>
							) : confirmDeleteId === component.id ? (
								<Popconfirm
									title="删除组件"
									description={`确认删除 ${component.name}？此操作无法撤销。`}
									okText="确认删除"
									cancelText="取消"
									open
									onCancel={() => setConfirmDeleteId(null)}
									onConfirm={() => handleDelete(component)}
									destroyOnHidden
								>
									{deleteButton}
								</Popconfirm>
							) : (
								deleteButton
							)
						) : null}
					</Space>
				);
			},
		},
	];

	return (
		<div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
			{contextHolder}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					gap: 12,
					flexWrap: "wrap",
				}}
			>
				<div>
					<Typography.Title level={2} style={{ margin: 0 }}>
						组件库
					</Typography.Title>
					<Typography.Text type="secondary">
						统一管理手工组件和平台生成的任务版本
					</Typography.Text>
				</div>
				<Space wrap>
					{selectedComponentIds.length > 0 ? (
						bulkDeleteConfirmOpen ? (
							<Popconfirm
								title={`删除选中的 ${selectedComponentIds.length} 个组件？`}
								description="删除后不可恢复。系统组件不可选择。"
								okText="确认删除"
								cancelText="取消"
								open
								onCancel={() => setBulkDeleteConfirmOpen(false)}
								onConfirm={handleBulkDelete}
								destroyOnHidden
							>
								<Button danger icon={<DeleteOutlined />} loading={bulkDeleting}>
									批量删除（{selectedComponentIds.length}）
								</Button>
							</Popconfirm>
						) : (
							<Button
								danger
								icon={<DeleteOutlined />}
								loading={bulkDeleting}
								onClick={() => {
									setBulkDeleteConfirmOpen(true);
									setConfirmDeleteId(null);
								}}
							>
								批量删除（{selectedComponentIds.length}）
							</Button>
						)
					) : null}
					<Input.Search
						id="component-manager-search"
						allowClear
						placeholder="搜索组件、任务、版本或镜像"
						value={search}
						onChange={(event) => setSearch(event.target.value)}
						style={{ width: 280 }}
					/>
					<Button
						icon={<ReloadOutlined />}
						onClick={() => {
							refresh();
							refreshReleases();
						}}
						loading={loading || releaseLoading}
					/>
					<Button icon={<SyncOutlined />} onClick={openSyncModal}>
						同步版本
					</Button>
					<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
						新建组件
					</Button>
				</Space>
			</div>

			{error ? (
				<Alert
					type="error"
					showIcon
					message="组件列表加载失败"
					description={error}
					action={
						<Button size="small" onClick={refresh} loading={loading}>
							重试
						</Button>
					}
				/>
			) : null}

			{releaseError ? (
				<Alert
					type="error"
					showIcon
					message="组件版本加载失败"
					description={releaseError}
					action={
						<Button
							size="small"
							onClick={refreshReleases}
							loading={releaseLoading}
						>
							重试
						</Button>
					}
				/>
			) : null}

			{!loading && !releaseLoading && libraryRows.length === 0 ? (
				<Empty description="暂无组件；可以新建 legacy 组件，或同步平台生成的任务版本" />
			) : (
				<Table
					rowKey={(record) => record.legacyComponent?.id || record.key}
					loading={loading || releaseLoading}
					columns={libraryColumns}
					dataSource={libraryRows}
					rowSelection={withSelectAllColumn<ComponentLibraryRow>({
						selectedRowKeys: selectedComponentIds,
						onChange: (_keys, rows) =>
							setSelectedComponentIds(
								rows
									.map((row) => row.legacyComponent?.id)
									.filter(Boolean) as string[],
							),
						getCheckboxProps: (record) => ({
							disabled:
								!record.legacyComponent ||
								record.legacyComponent.source === "system",
							name: record.name,
						}),
					})}
					expandable={{
						rowExpandable: (record) => record.releases.length > 0,
						expandedRowRender: (record) => (
							<Table
								size="small"
								rowKey="id"
								pagination={false}
								columns={[
									{
										title: "版本",
										dataIndex: "releaseLabel",
										render: (value: string, release) => (
											<Tag
												color={release.selectable ? "green" : "red"}
												style={{ cursor: "pointer" }}
												onClick={() => openReleaseView(release)}
											>
												{value}
											</Tag>
										),
									},
									{
										title: "状态",
										render: (_, release) => (
											<Space size={4}>
												<Tag color={releaseStatusColor(release.status)}>
													{release.status}
												</Tag>
												<Tag
													color={validationStatusColor(
														release.validationStatus,
													)}
												>
													{release.validationStatus}
												</Tag>
											</Space>
										),
									},
									{
										title: "镜像",
										render: (_, release) => (
											<Tooltip title={release.runtimeImage}>
												<Typography.Text code>
													{shortTechnicalValue(
														release.imageDigest || release.runtimeImage,
													)}
												</Typography.Text>
											</Tooltip>
										),
									},
									{
										title: "来源",
										render: (_, release) =>
											release.sourceCommit
												? shortTechnicalValue(release.sourceCommit)
												: release.taskPath || "-",
									},
									{
										title: "操作",
										width: 90,
										render: (_, release) => (
											<Button
												type="link"
												size="small"
												icon={<EyeOutlined />}
												onClick={() => openReleaseView(release)}
											>
												详情
											</Button>
										),
									},
								]}
								dataSource={record.releases}
							/>
						),
					}}
					pagination={{ pageSize: 12, showSizeChanger: true }}
				/>
			)}

			{releaseModalOpen && activeRelease ? (
				<Modal
					open
					title="组件版本详情"
					width={820}
					footer={[
						<Button key="close" onClick={closeReleaseModal}>
							关闭
						</Button>,
					]}
					onCancel={closeReleaseModal}
					destroyOnHidden
				>
					<ReleaseDetail release={activeRelease} />
				</Modal>
			) : null}

			{syncModalOpen ? (
				<Modal
					open
					title="同步组件版本"
					okText="同步"
					cancelText="关闭"
					confirmLoading={syncing}
					onOk={handleSyncReleases}
					onCancel={() => setSyncModalOpen(false)}
					width={840}
					destroyOnHidden
				>
					<Alert
						type="info"
						showIcon
						message="这里接收平台或 CI 生成的 release manifest。普通算法用户不需要填写这些字段。"
						style={{ marginBottom: 12 }}
					/>
					<Input.TextArea
						value={syncText}
						onChange={(event) => setSyncText(event.target.value)}
						rows={18}
						style={{ fontFamily: "monospace" }}
					/>
				</Modal>
			) : null}

			{modalOpen ? (
				<Modal
					open
					title={isViewMode ? "查看组件" : isEditMode ? "编辑组件" : "新建组件"}
					okText={isEditMode ? "保存" : "创建"}
					cancelText="关闭"
					confirmLoading={saving}
					onOk={handleSave}
					onCancel={closeModal}
					width={760}
					destroyOnHidden
					footer={
						isViewMode
							? [
									<Button key="close" onClick={closeModal}>
										关闭
									</Button>,
								]
							: [
									<Button key="close" onClick={closeModal} disabled={saving}>
										关闭
									</Button>,
									<Button
										key="submit"
										type="primary"
										loading={saving}
										onClick={handleSave}
									>
										{isEditMode ? "保存" : "创建"}
									</Button>,
								]
					}
				>
					{isViewMode && activeComponent ? (
						<ComponentDetail component={activeComponent} />
					) : (
						<Form
							key={modalFormKey}
							form={form}
							layout="vertical"
							initialValues={modalFormInitialValues}
							style={{ marginTop: 16 }}
						>
							<div
								style={{
									display: "grid",
									gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
									gap: 12,
								}}
							>
								<Form.Item
									name="name"
									label="名称"
									rules={[{ required: true, message: "请输入组件名称" }]}
								>
									<Input
										placeholder="normalize-mcap"
										ref={nameInputRef}
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item
									name="type"
									label="类型"
									rules={[{ required: true, message: "请选择组件类型" }]}
								>
									<Select options={TYPE_OPTIONS} />
								</Form.Item>
							</div>

							<div
								style={{
									display: "grid",
									gridTemplateColumns: "minmax(260px, 1fr) 160px",
									gap: 12,
								}}
							>
								<Form.Item
									name="image"
									label="镜像"
									rules={[{ required: true, message: "请输入镜像" }]}
								>
									<Input
										placeholder="registry.example.com/databrew/worker"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item name="tag" label="标签">
									<Input
										placeholder="latest"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
							</div>

							<Form.Item name="description" label="描述">
								<Input.TextArea rows={3} maxLength={500} showCount />
							</Form.Item>

							<Form.Item
								name="command"
								label="命令"
								extra="例如 sh, -c；如果使用 shell 执行脚本，参数只填写脚本本体。"
							>
								<Input.TextArea
									rows={2}
									placeholder="例如: python, /app/main.py"
								/>
							</Form.Item>

							<Form.Item
								name="args"
								label="参数"
								extra="连接下游输出时，脚本需要写入 /tmp/outputs/output；多个输出端口分别写入对应文件名。"
							>
								<Input.TextArea
									rows={2}
									placeholder="例如: --input, {{inputs.asset}}"
								/>
							</Form.Item>

							<div
								style={{
									display: "grid",
									gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
									gap: 12,
								}}
							>
								<Form.Item name="cpu" label="CPU" extra="例如 500m、4">
									<Input
										data-testid="component-resource-cpu"
										placeholder="500m"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item name="memory" label="内存" extra="例如 256Mi、16Gi">
									<Input
										data-testid="component-resource-memory"
										placeholder="256Mi"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item name="disk" label="磁盘" extra="临时存储，例如 20Gi">
									<Input
										data-testid="component-resource-disk"
										placeholder="20Gi"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item name="gpu" label="GPU" extra="数量，例如 1、2">
									<Input
										data-testid="component-resource-gpu"
										placeholder="1"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
								<Form.Item
									name="computeTier"
									label="计算档位"
									extra="用于后续调度、成本和配额策略"
								>
									<Input
										data-testid="component-resource-compute-tier"
										placeholder="gpu-l4"
										autoComplete="off"
										onFocus={(e) => e.target.select()}
									/>
								</Form.Item>
							</div>

							<div
								style={{
									display: "grid",
									gridTemplateColumns: "1fr",
									gap: 12,
									marginBottom: 16,
								}}
							>
								<PortFormList
									name="inputPorts"
									title="输入数据"
									emptyPort={{ name: "input", type: "asset" }}
								/>
								<PortFormList
									name="outputPorts"
									title="输出结果"
									emptyPort={{ name: "output", type: "asset" }}
									showOutputPath
								/>
							</div>

							<Form.List name="envRows">
								{(fields, { add, remove }) => (
									<div>
										<div
											style={{
												display: "flex",
												alignItems: "center",
												justifyContent: "space-between",
												marginBottom: 8,
											}}
										>
											<Typography.Text>环境变量</Typography.Text>
											<Button size="small" onClick={() => add({})}>
												添加变量
											</Button>
										</div>
										{fields.map(({ key, ...field }, index) => (
											<Space
												key={key}
												align="baseline"
												style={{ display: "flex", marginBottom: 8 }}
											>
												<span style={{ width: 20, color: "rgba(0,0,0,0.45)" }}>
													{index + 1}.
												</span>
												<Form.Item {...field} name={[field.name, "name"]}>
													<Input placeholder="KEY" style={{ width: 220 }} />
												</Form.Item>
												<Form.Item {...field} name={[field.name, "value"]}>
													<Input placeholder="value" style={{ width: 320 }} />
												</Form.Item>
												<Button
													type="text"
													danger
													icon={<DeleteOutlined />}
													aria-label="移除环境变量"
													onClick={() => remove(field.name)}
												/>
											</Space>
										))}
									</div>
								)}
							</Form.List>
						</Form>
					)}
				</Modal>
			) : null}
		</div>
	);
}
