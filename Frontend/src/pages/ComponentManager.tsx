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
	Segmented,
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
	type ComponentReleaseIngestSource,
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
import {
	dedupePipelineComponentsByName,
	formatComponentImage,
} from "../lib/pipelineComponentDisplay";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "../lib/pipelineContract";

type EnvRow = { name?: string; value?: string };
type PortRow = {
	name?: string;
	type?: string;
	desc?: string;
	default_value?: string;
};

type ModalMode = "create" | "edit" | "view";

type SyncReleaseBody =
	| {
			source?: ComponentReleaseIngestSource;
			items?: PipelineComponentReleasePayload[];
	  }
	| PipelineComponentReleasePayload[];

interface ComponentLibraryRow {
	key: string;
	name: string;
	imageUid: string;
	componentId: string;
	legacyComponent?: PipelineComponentAPI;
	releases: PipelineComponentReleaseAPI[];
	primaryRelease?: PipelineComponentReleaseAPI;
	updatedAt?: string;
	searchText: string;
}

type ReleaseKindFilter = "all" | "tag" | "commit" | "branch" | "pr";
type SearchMode = "smart" | "task" | "commit" | "version";
type LibraryTypeFilter = "all" | "release" | "legacy";

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

const RELEASE_KIND_OPTIONS: Array<{ label: string; value: ReleaseKindFilter }> =
	[
		{ label: "全部标记", value: "all" },
		{ label: "线上版本", value: "tag" },
		{ label: "测试版本", value: "commit" },
		{ label: "分支版本", value: "branch" },
		{ label: "PR 预览", value: "pr" },
	];

const SEARCH_MODE_OPTIONS: Array<{ label: string; value: SearchMode }> = [
	{ label: "全部字段", value: "smart" },
	{ label: "按名称", value: "task" },
	{ label: "按 commit", value: "commit" },
	{ label: "按版本号", value: "version" },
];

const LIBRARY_TYPE_OPTIONS: Array<{
	label: string;
	value: LibraryTypeFilter;
}> = [
	{ label: "全部", value: "all" },
	{ label: "版本库", value: "release" },
	{ label: "手工组件", value: "legacy" },
];

const formatDateTime = (value?: string): string =>
	value ? new Date(value).toLocaleString() : "-";

const shortTechnicalValue = (value?: string): string => {
	if (!value) return "-";
	if (value.length <= 18) return value;
	return `${value.slice(0, 10)}…${value.slice(-8)}`;
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

const normalizedReleaseRef = (release: PipelineComponentReleaseAPI): string =>
	(release.sourceRef || "")
		.replace(/^refs\/heads\//, "")
		.replace(/^refs\/tags\//, "");

const isMainRelease = (release: PipelineComponentReleaseAPI): boolean => {
	const ref = normalizedReleaseRef(release);
	return ref === "main" || release.releaseLabel?.startsWith("main-");
};

const isOnlineRelease = (release: PipelineComponentReleaseAPI): boolean =>
	release.sourceRefType === "tag" || isMainRelease(release);

const releaseTagText = (release?: PipelineComponentReleaseAPI): string => {
	if (!release || release.sourceRefType !== "tag") return "";
	return normalizedReleaseRef(release) || release.releaseLabel || "";
};

const releaseKindLabel = (release: PipelineComponentReleaseAPI): string => {
	if (isOnlineRelease(release)) return "线上版本";
	switch (release.sourceRefType) {
		case "commit":
			return "测试版本";
		case "pr":
			return "PR 预览";
		case "branch":
			return "分支版本";
		default:
			return "开发版本";
	}
};

const releaseKindColor = (release: PipelineComponentReleaseAPI): string => {
	if (isOnlineRelease(release)) return "green";
	switch (release.sourceRefType) {
		case "commit":
			return "blue";
		case "pr":
			return "purple";
		case "branch":
			return "cyan";
		default:
			return "default";
	}
};

const componentKey = (value?: string): string =>
	(value || "").trim().toLowerCase();

const releaseDisplayName = (release: PipelineComponentReleaseAPI): string =>
	release.displayName || release.taskName || release.componentId;

const compareDateDesc = (a?: string, b?: string): number =>
	new Date(b || 0).getTime() - new Date(a || 0).getTime();

const shortImageUid = (identity?: string): string => {
	const value = (identity || "").trim().toLowerCase();
	if (!value) return "";
	let hash = 0x811c9dc5;
	for (let i = 0; i < value.length; i += 1) {
		hash ^= value.charCodeAt(i);
		hash = Math.imul(hash, 0x01000193);
	}
	return (hash >>> 0).toString(16).padStart(8, "0");
};

const releaseImageUid = (release?: PipelineComponentReleaseAPI): string =>
	release?.imageUid ||
	shortImageUid(release?.imageDigest || release?.runtimeImage || release?.id);

const legacyImageUid = (component?: PipelineComponentAPI): string =>
	component
		? shortImageUid(formatComponentImage(component.image, component.tag))
		: "";

const sourceRefTypeText = (release: PipelineComponentReleaseAPI): string => {
	switch (release.sourceRefType) {
		case "tag":
			return "Git tag";
		case "commit":
			return "Git commit";
		case "branch":
			return "Git branch";
		case "pr":
			return "Git PR";
		default:
			return "Git";
	}
};

const releaseRefBadge = (
	release: PipelineComponentReleaseAPI,
): { label: string; color: string } => {
	if (isOnlineRelease(release)) {
		return { label: "线上", color: "green" };
	}
	if (release.sourceRefType === "pr") {
		return { label: "PR", color: "purple" };
	}
	const ref = normalizedReleaseRef(release);
	if (ref) {
		return { label: shortTechnicalValue(ref), color: "cyan" };
	}
	if (release.sourceRefType === "commit") {
		return { label: "commit", color: "blue" };
	}
	return { label: "dev", color: "default" };
};

function ReleaseVersionChip({
	release,
	onClick,
}: {
	release: PipelineComponentReleaseAPI;
	onClick: () => void;
}) {
	const badge = releaseRefBadge(release);
	const commitText = shortTechnicalValue(
		release.sourceCommit || release.releaseLabel,
	);
	return (
		<Tooltip title={release.sourceCommit || release.releaseLabel}>
			<Space size={4}>
				<Tag
					color={badge.color}
					style={{ cursor: "pointer", fontFamily: "monospace" }}
					onClick={onClick}
				>
					{commitText}
				</Tag>
				<Tag color={badge.color}>{badge.label}</Tag>
			</Space>
		</Tooltip>
	);
}

const copyableCode = (value?: string, display?: string) => (
	<Tooltip title={value || "-"}>
		<Typography.Text
			code
			copyable={value ? { text: value } : false}
			style={{ maxWidth: "100%", display: "inline-block" }}
		>
			{display || shortTechnicalValue(value) || "-"}
		</Typography.Text>
	</Tooltip>
);

const releaseSearchText = (release: PipelineComponentReleaseAPI): string =>
	[
		release.componentId,
		release.taskName,
		release.displayName,
		release.releaseLabel,
		release.channel,
		release.sourceRefType,
		release.sourceRepo,
		release.sourceRef,
		release.sourceCommit,
		release.buildId,
		release.imageTag,
		releaseImageUid(release),
		release.imageDigest,
		release.runtimeImage,
	]
		.filter(Boolean)
		.join(" ")
		.toLowerCase();

const releaseIdentityText = (release: PipelineComponentReleaseAPI): string =>
	[release.componentId, release.taskName, release.displayName, release.taskPath]
		.filter(Boolean)
		.join(" ")
		.toLowerCase();

const isCommitSearch = (query: string): boolean =>
	query.length >= 7 && query.length <= 64 && /^[0-9a-f]+$/i.test(query);

const releaseMatchesKind = (
	release: PipelineComponentReleaseAPI,
	kind: ReleaseKindFilter,
): boolean => {
	if (kind === "all") return true;
	if (kind === "tag") return isOnlineRelease(release);
	if (kind === "commit") return release.sourceRefType === "commit";
	if (kind === "branch") {
		return release.sourceRefType === "branch" && !isMainRelease(release);
	}
	return release.sourceRefType === "pr";
};

const releaseMatchesSearch = (
	release: PipelineComponentReleaseAPI,
	query: string,
	mode: SearchMode,
): boolean => {
	if (!query) return true;
	if (mode === "commit") {
		return (release.sourceCommit || "").toLowerCase().includes(query);
	}
	if (mode === "version") {
		return (release.releaseLabel || "").toLowerCase().includes(query);
	}
	return releaseSearchText(release).includes(query);
};

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
				<Descriptions.Item label="组件 ID" span={2}>
					<Typography.Text copyable={{ text: component.id }}>
						{component.id}
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
					<Tag color={releaseRefBadge(release).color}>
						{release.releaseLabel}
					</Tag>
					<Tag color={releaseKindColor(release)}>
						{releaseKindLabel(release)}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="状态" span={1}>
					<Tag color={releaseStatusColor(release.status)}>{release.status}</Tag>
					<Tag color={validationStatusColor(release.validationStatus)}>
						{release.validationStatus}
					</Tag>
					<Tag color={release.selectable ? "green" : "red"}>
						{release.selectable ? "可选择" : "不可选"}
					</Tag>
				</Descriptions.Item>
				<Descriptions.Item label="镜像ID">
					{copyableCode(releaseImageUid(release), releaseImageUid(release))}
				</Descriptions.Item>
				<Descriptions.Item label="来源类型">
					{sourceRefTypeText(release)}
				</Descriptions.Item>
				<Descriptions.Item label="镜像" span={2}>
					<div style={{ maxWidth: "100%" }}>
						{copyableCode(
							release.runtimeImage,
							shortTechnicalValue(release.runtimeImage),
						)}
					</div>
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
				<Descriptions.Item label="Component ID">
					<Typography.Text copyable>{release.componentId}</Typography.Text>
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
				<Descriptions.Item label="Source Ref Type">
					{release.sourceRefType || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="Source Commit">
					<Typography.Text copyable={{ text: release.sourceCommit || "" }}>
						{release.sourceCommit || "-"}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="Image Digest">
					{copyableCode(
						release.imageDigest,
						shortTechnicalValue(release.imageDigest),
					)}
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
	const [searchMode, setSearchMode] = useState<SearchMode>("smart");
	const [releaseKindFilter, setReleaseKindFilter] =
		useState<ReleaseKindFilter>("all");
	const [libraryTypeFilter, setLibraryTypeFilter] =
		useState<LibraryTypeFilter>("all");
	const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);
	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<ModalMode | null>(null);
	const [releaseModalOpen, setReleaseModalOpen] = useState(false);
	const [syncModalOpen, setSyncModalOpen] = useState(false);
	const [activeComponent, setActiveComponent] =
		useState<PipelineComponentAPI | null>(null);
	const [activeRelease, setActiveRelease] =
		useState<PipelineComponentReleaseAPI | null>(null);
	const [syncText, setSyncText] = useState("");
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
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
		const q = search.trim().toLowerCase();
		const commitQuery = isCommitSearch(q);
		const filterReleases = (
			groupedReleases: PipelineComponentReleaseAPI[],
			identityText: string,
		): PipelineComponentReleaseAPI[] => {
			const base = groupedReleases.filter((release) =>
				releaseMatchesKind(release, releaseKindFilter),
			);
			if (!q) return base;
			if (searchMode === "task") {
				return identityText.includes(q) ? base : [];
			}
			if (searchMode === "smart" && !commitQuery && identityText.includes(q)) {
				return base;
			}
			return base.filter((release) =>
				releaseMatchesSearch(release, q, searchMode),
			);
		};

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
			const identityText = [
				component.name,
				component.id,
				component.description,
				component.source,
			]
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			const visibleReleases = filterReleases(groupedReleases, identityText);
			const primaryRelease =
				visibleReleases.find((release) => release.selectable) ||
				visibleReleases[0] ||
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
				releaseImageUid(primaryRelease) || legacyImageUid(component),
				...visibleReleases.map(releaseSearchText),
			]
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			return {
				key: `component:${component.id}`,
				name: component.name,
				imageUid: releaseImageUid(primaryRelease) || legacyImageUid(component),
				componentId: component.name,
				legacyComponent: component,
				releases: visibleReleases,
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
			const identityText = groupedReleases.map(releaseIdentityText).join(" ");
			const visibleReleases = filterReleases(groupedReleases, identityText);
			if (visibleReleases.length === 0) {
				continue;
			}
			const searchText = visibleReleases
				.map(releaseSearchText)
				.filter(Boolean)
				.join(" ")
				.toLowerCase();
			const visiblePrimaryRelease =
				visibleReleases.find((release) => release.selectable) ||
				visibleReleases[0];
			rows.push({
				key: `release:${key}`,
				name,
				imageUid: releaseImageUid(visiblePrimaryRelease),
				componentId: visiblePrimaryRelease?.componentId || key,
				releases: visibleReleases,
				primaryRelease: visiblePrimaryRelease,
				updatedAt: visiblePrimaryRelease?.updatedAt,
				searchText,
			});
		}

		const filtered = rows.filter((row) => {
			if (row.releases.length === 0 && !row.legacyComponent) return false;
			if (libraryTypeFilter === "release" && row.releases.length === 0) {
				return false;
			}
			if (
				libraryTypeFilter === "legacy" &&
				(!row.legacyComponent || row.releases.length > 0)
			) {
				return false;
			}
			if (releaseKindFilter !== "all" && row.releases.length === 0) {
				return false;
			}
			if (!q) return true;
			return row.searchText.includes(q) || row.releases.length > 0;
		});
		return filtered.sort((a, b) => {
			if (a.primaryRelease && !b.primaryRelease) return -1;
			if (!a.primaryRelease && b.primaryRelease) return 1;
			return compareDateDesc(a.updatedAt, b.updatedAt);
		});
	}, [
		items,
		libraryTypeFilter,
		releaseKindFilter,
		releases,
		search,
		searchMode,
	]);

	const activeExpandedRowKeys = useMemo(() => {
		if (!search.trim()) return expandedRowKeys;
		return libraryRows
			.filter((row) => row.releases.length > 0)
			.map((row) => row.legacyComponent?.id || row.key);
	}, [expandedRowKeys, libraryRows, search]);

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
					source: {
						provider: "cloud-build",
						repo: "CyberOrigin2077/automated-processing-gcloud",
						ref: "refs/heads/main",
						commit: "abc123",
						buildId: "cloud-build-id",
						trigger: "hand-detect-yolov26m-build-trigger",
					},
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
		const source = Array.isArray(parsed) ? undefined : parsed.source;
		const items = Array.isArray(parsed) ? parsed : parsed.items;
		if (!Array.isArray(items)) {
			messageApi.error("JSON 需要是数组，或包含 items 数组");
			return;
		}
		setSyncing(true);
		try {
			const res = await syncComponentReleases(items, source);
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

	const libraryColumns: ColumnsType<ComponentLibraryRow> = [
		{
			title: "任务 / 镜像",
			key: "name",
			width: 240,
			render: (_, record) => (
				<Space direction="vertical" size={0}>
					<Typography.Text strong ellipsis={{ tooltip: record.name }}>
						{record.name}
					</Typography.Text>
					<Space size={4} wrap>
						<Tag color={record.primaryRelease ? "green" : "default"}>
							{record.primaryRelease ? "版本库" : "手工组件"}
						</Tag>
					</Space>
					<Typography.Text
						type="secondary"
						copyable={{ text: record.imageUid }}
						style={{ fontSize: 12, fontFamily: "monospace" }}
					>
						镜像ID {record.imageUid || "-"}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: "构建版本",
			key: "versions",
			width: 240,
			render: (_, record) => (
				<Space wrap size={4}>
					{record.primaryRelease ? (
						<ReleaseVersionChip
							release={record.primaryRelease}
							onClick={() =>
								openReleaseView(
									record.primaryRelease as PipelineComponentReleaseAPI,
								)
							}
						/>
					) : null}
					{record.releases.length > 1 ? (
						<Tag>{record.releases.length} 个版本</Tag>
					) : null}
					{record.releases.length === 0 && record.legacyComponent ? (
						<Typography.Text code>
							{record.legacyComponent.tag || "latest"}
						</Typography.Text>
					) : null}
					{record.releases.length === 0 && !record.legacyComponent ? "-" : null}
				</Space>
			),
		},
		{
			title: "Tag",
			key: "tag",
			width: 160,
			render: (_, record) => {
				const tag = releaseTagText(record.primaryRelease);
				return tag ? (
					<Typography.Text code ellipsis={{ tooltip: tag }}>
						{tag}
					</Typography.Text>
				) : (
					<Typography.Text type="secondary">-</Typography.Text>
				);
			},
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
				const showComponentActions =
					Boolean(component) && libraryTypeFilter !== "release";
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
						{record.primaryRelease && showComponentActions ? (
							<Typography.Text type="secondary">|</Typography.Text>
						) : null}
						{showComponentActions && component ? (
							<Button
								type="link"
								size="small"
								icon={<EyeOutlined />}
								aria-label="查看组件定义"
								onClick={() => openView(component)}
							>
								组件定义
							</Button>
						) : null}
						{showComponentActions && component ? (
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
						{showComponentActions && component && deleteButton ? (
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
			<div>
				<div>
					<Typography.Title level={2} style={{ margin: 0 }}>
						组件库
					</Typography.Title>
					<Typography.Text type="secondary">
						统一管理手工组件和平台生成的任务版本
					</Typography.Text>
				</div>
				<div
					style={{
						display: "flex",
						alignItems: "flex-start",
						justifyContent: "space-between",
						gap: 12,
						flexWrap: "wrap",
						marginTop: 16,
					}}
				>
					<div
						style={{
							display: "flex",
							alignItems: "flex-start",
							gap: 8,
							flexWrap: "wrap",
							flex: "1 1 620px",
							minWidth: 0,
						}}
					>
						<div style={{ minWidth: 280, flex: "0 1 340px" }}>
							<Input.Search
								id="component-manager-search"
								allowClear
								placeholder="搜索名称、commit、版本号"
								value={search}
								onChange={(event) => setSearch(event.target.value)}
								style={{ width: "100%" }}
							/>
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>
								先按名称定位 task，再切到按 commit 精确到镜像版本
							</Typography.Text>
						</div>
						<Select<SearchMode>
							value={searchMode}
							options={SEARCH_MODE_OPTIONS}
							onChange={setSearchMode}
							style={{ width: 128 }}
						/>
						<Select<ReleaseKindFilter>
							value={releaseKindFilter}
							options={RELEASE_KIND_OPTIONS}
							onChange={setReleaseKindFilter}
							style={{ width: 128 }}
						/>
						<Segmented<LibraryTypeFilter>
							value={libraryTypeFilter}
							options={LIBRARY_TYPE_OPTIONS}
							onChange={setLibraryTypeFilter}
						/>
					</div>
					<Space wrap style={{ justifyContent: "flex-end" }}>
						<Tooltip title="刷新">
							<Button
								icon={<ReloadOutlined />}
								onClick={() => {
									refresh();
									refreshReleases();
								}}
								loading={loading || releaseLoading}
							/>
						</Tooltip>
						<Button icon={<SyncOutlined />} onClick={openSyncModal}>
							同步版本
						</Button>
						<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
							新建组件
						</Button>
					</Space>
				</div>
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
					expandable={{
						expandedRowKeys: activeExpandedRowKeys,
						onExpandedRowsChange: (keys) =>
							setExpandedRowKeys(keys.map(String)),
						rowExpandable: (record) => record.releases.length > 0,
						expandedRowRender: (record) => (
							<div
								style={{
									background: "#f8fafc",
									borderTop: "1px solid #eef2f7",
									borderBottom: "1px solid #eef2f7",
									margin: "-16px",
									padding: "8px 0",
								}}
							>
								<div
									style={{
										display: "grid",
										gridTemplateColumns:
											"44px 240px minmax(260px, 320px) minmax(140px, 180px) minmax(120px, 160px) 160px",
										alignItems: "center",
										columnGap: 16,
										padding: "8px 0",
										color: "rgba(15, 23, 42, 0.65)",
										fontSize: 12,
										fontWeight: 600,
									}}
								>
									<div />
									<div>全部构建版本</div>
									<div>Commit</div>
									<div>Tag</div>
									<div>镜像ID</div>
									<div>操作</div>
								</div>
								{record.releases.map((release) => (
									<div
										key={release.id}
										style={{
											display: "grid",
											gridTemplateColumns:
												"44px 240px minmax(260px, 320px) minmax(140px, 180px) minmax(120px, 160px) 160px",
											alignItems: "center",
											columnGap: 16,
											padding: "8px 0",
											borderTop: "1px solid #eef2f7",
										}}
									>
										<div />
										<div>
											<ReleaseVersionChip
												release={release}
												onClick={() => openReleaseView(release)}
											/>
										</div>
										<div>
											{releaseTagText(release) ? (
												<Typography.Text
													code
													ellipsis={{ tooltip: releaseTagText(release) }}
												>
													{releaseTagText(release)}
												</Typography.Text>
											) : (
												<Typography.Text type="secondary">-</Typography.Text>
											)}
										</div>
										<div>
											<Typography.Text
												code
												copyable={{ text: releaseImageUid(release) }}
											>
												{releaseImageUid(release)}
											</Typography.Text>
										</div>
										<div>
											<Button
												type="link"
												size="small"
												icon={<EyeOutlined />}
												onClick={() => openReleaseView(release)}
											>
												版本详情
											</Button>
										</div>
									</div>
								))}
							</div>
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
					styles={{
						body: { maxHeight: "68vh", overflowY: "auto", paddingRight: 8 },
					}}
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
					styles={{
						body: { maxHeight: "68vh", overflowY: "auto", paddingRight: 8 },
					}}
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
									gridTemplateColumns:
										"minmax(280px, 1fr) minmax(180px, 220px)",
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
