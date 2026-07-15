import {
	DeleteOutlined,
	DownOutlined,
	EditOutlined,
	EyeOutlined,
	PlusOutlined,
	ReloadOutlined,
	RightOutlined,
	SyncOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Descriptions,
	Empty,
	Form,
	Grid,
	Input,
	Modal,
	message,
	Popconfirm,
	Segmented,
	Select,
	Space,
	Spin,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { InputRef } from "antd/es/input";
import type { ColumnsType } from "antd/es/table";
import {
	type ReactNode,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
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
	formatCommitDisplay,
	formatComponentImage,
	normalizeGitCommit,
} from "../lib/pipelineComponentDisplay";
import {
	normalizeComponentArgs,
	normalizeShellCommandArgs,
} from "../lib/pipelineContract";

const { useBreakpoint } = Grid;

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

type ReleaseKindFilter = "all" | "prod" | "dev";
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
		{ label: "生产", value: "prod" },
		{ label: "开发", value: "dev" },
	];

const SEARCH_MODE_OPTIONS: Array<{ label: string; value: SearchMode }> = [
	{ label: "全部字段", value: "smart" },
	{ label: "按名称", value: "task" },
	{ label: "按 commit", value: "commit" },
	{ label: "按版本号", value: "version" },
];

const LIBRARY_TYPE_OPTIONS: Array<{
	label: ReactNode;
	value: LibraryTypeFilter;
}> = [
	{ label: "全部", value: "all" },
	{
		label: (
			<Tooltip title="平台 CI 同步生成的任务镜像版本">
				<span>版本库组件</span>
			</Tooltip>
		),
		value: "release",
	},
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

const releaseTagEmptyHint = (release: PipelineComponentReleaseAPI): string => {
	switch (release.sourceRefType) {
		case "branch":
			return "该版本由分支构建，未打 Git tag";
		case "commit":
			return "该版本由 commit 构建，未打 Git tag";
		default:
			return "该版本未关联 Git tag";
	}
};

function ReleaseTagCell({
	release,
}: {
	release?: PipelineComponentReleaseAPI;
}) {
	const tag = releaseTagText(release);
	if (tag) {
		return (
			<Typography.Text code ellipsis={{ tooltip: tag }}>
				{tag}
			</Typography.Text>
		);
	}
	if (!release) {
		return <Typography.Text type="secondary">-</Typography.Text>;
	}
	return (
		<Tooltip title={releaseTagEmptyHint(release)}>
			<Typography.Text type="secondary">未打 Tag</Typography.Text>
		</Tooltip>
	);
}

const releaseEnvironmentMeta = (
	release: PipelineComponentReleaseAPI,
): { label: string; tooltip: string } => {
	const isProd = isOnlineRelease(release);
	return {
		label: isProd ? "生产" : "开发",
		tooltip: isProd ? "环境：prod" : "环境：dev",
	};
};

const releaseKindLabel = (release: PipelineComponentReleaseAPI): string =>
	releaseEnvironmentMeta(release).label;

const releaseKindColor = (release: PipelineComponentReleaseAPI): string =>
	isOnlineRelease(release) ? "green" : "default";

const componentKey = (value?: string): string =>
	(value || "").trim().toLowerCase();

const releaseDisplayName = (release: PipelineComponentReleaseAPI): string =>
	release.displayName || release.taskName || release.componentId;

const compareTextAsc = (a?: string, b?: string): number =>
	(a || "").localeCompare(b || "", undefined, { sensitivity: "base" });

const compareDateAsc = (a?: string, b?: string): number =>
	new Date(a || 0).getTime() - new Date(b || 0).getTime();

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

const CATEGORY_DEFS: Array<{ pattern: RegExp; id: string; color: string }> = [
	{ pattern: /^cyberpipe-/, id: "cyberpipe", color: "#1677ff" },
	{ pattern: /^cyb2178-/, id: "cyb2178", color: "#722ed1" },
	{ pattern: /^hand-track-/, id: "hand-track", color: "#fa8c16" },
	{ pattern: /^(smoke-|output-writer-)/, id: "tools", color: "#52c41a" },
];

function deriveCategory(name: string): { id: string; color: string } | null {
	if (!name) return null;
	for (const def of CATEGORY_DEFS) {
		if (def.pattern.test(name)) return { id: def.id, color: def.color };
	}
	return null;
}

const releaseImageUid = (release?: PipelineComponentReleaseAPI): string =>
	release?.imageUid ||
	shortImageUid(release?.imageDigest || release?.runtimeImage || release?.id);

const metadataStringValue = (
	metadata: Record<string, unknown> | undefined,
	...keys: string[]
): string => {
	for (const key of keys) {
		const value = metadata?.[key];
		if (typeof value === "string" && value.trim()) {
			return value.trim();
		}
	}
	return "";
};

const releaseTaskPath = (release: PipelineComponentReleaseAPI): string =>
	release.taskPath ||
	metadataStringValue(
		release.technicalMetadata,
		"taskDir",
		"taskPath",
		"task_path",
	);

const imageTagFromReference = (image?: string): string => {
	const value = (image || "").trim();
	if (!value || value.includes("@sha256:")) return "";
	const lastSlash = value.lastIndexOf("/");
	const lastColon = value.lastIndexOf(":");
	return lastColon > lastSlash && lastColon < value.length - 1
		? value.slice(lastColon + 1)
		: "";
};

const releaseImageTag = (release?: PipelineComponentReleaseAPI): string =>
	release?.imageTag || imageTagFromReference(release?.runtimeImage);

const releaseImageReference = (
	release?: PipelineComponentReleaseAPI,
): string => {
	if (!release) return "";
	if (release.runtimeImage) return release.runtimeImage;
	if (release.imageRepo && release.imageTag) {
		return `${release.imageRepo}:${release.imageTag}`;
	}
	return release.imageRepo || "";
};

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
): { label: string; color: string; tooltip: string } => {
	const env = releaseEnvironmentMeta(release);
	return {
		label: env.label,
		color: env.label === "生产" ? "green" : "default",
		tooltip: env.tooltip,
	};
};

const isPlainCommitLabel = (value?: string): boolean => {
	const normalized = normalizeGitCommit(value);
	return /^[0-9a-f]{7,40}$/i.test(normalized);
};

const releaseShortCommit = (release: PipelineComponentReleaseAPI): string => {
	const commit =
		normalizeGitCommit(release.sourceCommit) ||
		(isPlainCommitLabel(release.releaseLabel)
			? normalizeGitCommit(release.releaseLabel)
			: "");
	return commit ? commit.slice(0, 7) : "";
};

const releaseVersionLabel = (
	release: PipelineComponentReleaseAPI,
	overrideLabel?: string,
): string => {
	const explicit = (overrideLabel || release.releaseLabel || "").trim();
	if (explicit && !isPlainCommitLabel(explicit)) {
		return shortTechnicalValue(explicit);
	}

	const shortCommit = releaseShortCommit(release);
	if (release.sourceRefType === "tag") {
		return (
			shortTechnicalValue(normalizedReleaseRef(release) || explicit) || "-"
		);
	}
	if (release.sourceRefType === "branch") {
		const ref = normalizedReleaseRef(release) || "branch";
		return shortTechnicalValue(shortCommit ? `${ref}-${shortCommit}` : ref);
	}
	if (release.sourceRefType === "pr") {
		const ref = normalizedReleaseRef(release) || "pr";
		return shortTechnicalValue(shortCommit ? `${ref}-${shortCommit}` : ref);
	}
	if (shortCommit) return `commit-${shortCommit}`;
	return shortTechnicalValue(explicit) || "-";
};

const releaseBuildTrigger = (release: PipelineComponentReleaseAPI): string =>
	metadataStringValue(
		release.technicalMetadata,
		"cloudBuildTrigger",
		"trigger",
		"triggerName",
	);

const releaseVersionTooltip = (release: PipelineComponentReleaseAPI): string =>
	[
		`版本: ${release.releaseLabel || "-"}`,
		`来源: ${sourceRefTypeText(release)} ${release.sourceRef || "-"}`,
		`Commit: ${formatCommitDisplay(release.sourceCommit)}`,
		release.buildId ? `Build ID: ${release.buildId}` : "",
		releaseBuildTrigger(release)
			? `Trigger: ${releaseBuildTrigger(release)}`
			: "",
	]
		.filter(Boolean)
		.join("\n");

const buildVersionToggleLabel = (
	releaseCount: number,
	expanded: boolean,
): string => {
	if (expanded) return "收起构建版本";
	if (releaseCount > 1) return `查看 ${releaseCount} 个构建版本`;
	return "展开构建版本";
};

const componentSourceSummary = (record: ComponentLibraryRow): string => {
	if (record.primaryRelease || record.releases.length > 0) {
		const count = record.releases.length;
		if (count === 0) return "来自版本库 · 暂无构建版本";
		return `来自版本库 · ${count} 个构建版本`;
	}
	return "手工组件 · 无构建版本";
};

function ReleaseVersionChip({
	release,
	onClick,
	label,
}: {
	release: PipelineComponentReleaseAPI;
	onClick: () => void;
	label?: string;
}) {
	const badge = releaseRefBadge(release);
	return (
		<Tooltip title={releaseVersionTooltip(release)}>
			<Space size={6} align="center">
				<Tag
					color={badge.color}
					style={{ cursor: "pointer", fontFamily: "monospace", margin: 0 }}
					onClick={onClick}
				>
					{releaseVersionLabel(release, label)}
				</Tag>
				<Tooltip title={badge.tooltip}>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{badge.label}
					</Typography.Text>
				</Tooltip>
			</Space>
		</Tooltip>
	);
}

function ReleaseExpandedBuildCell({
	release,
	isPrimary,
	onClick,
}: {
	release: PipelineComponentReleaseAPI;
	isPrimary?: boolean;
	onClick: () => void;
}) {
	const badge = releaseRefBadge(release);
	return (
		<Space size={6} wrap align="center">
			<Tooltip title={releaseVersionTooltip(release)}>
				<Tag
					color={badge.color}
					className="component-library-release-grid__build-tag"
					style={{ cursor: "pointer", margin: 0 }}
					onClick={onClick}
				>
					{releaseVersionLabel(release)}
				</Tag>
			</Tooltip>
			{isPrimary ? (
				<Tag color="processing" style={{ margin: 0 }}>
					默认执行
				</Tag>
			) : null}
			<Tooltip title={badge.tooltip}>
				<Typography.Text type="secondary" style={{ fontSize: 11 }}>
					{badge.label}
				</Typography.Text>
			</Tooltip>
		</Space>
	);
}

const copyableCode = (value?: string, display?: string) => (
	<Tooltip title={value || "-"}>
		<Typography.Text
			code
			copyable={value ? { text: value } : false}
			className="component-library-release-grid__mono"
			ellipsis={{ tooltip: value || display }}
		>
			{display || shortTechnicalValue(value) || "-"}
		</Typography.Text>
	</Tooltip>
);

function ReleaseExpandedTable({
	releases,
	primaryReleaseId,
	onViewRelease,
}: {
	releases: PipelineComponentReleaseAPI[];
	primaryReleaseId?: string;
	onViewRelease: (release: PipelineComponentReleaseAPI) => void;
}) {
	const columns: ColumnsType<PipelineComponentReleaseAPI> = [
		{
			title: "版本名称",
			key: "releaseLabel",
			sorter: (a, b) => compareTextAsc(a.releaseLabel, b.releaseLabel),
			render: (_, release) => (
				<ReleaseExpandedBuildCell
					release={release}
					isPrimary={release.id === primaryReleaseId}
					onClick={() => onViewRelease(release)}
				/>
			),
		},
		{
			title: "Commit",
			key: "commit",
			sorter: (a, b) =>
				compareTextAsc(
					normalizeGitCommit(a.sourceCommit),
					normalizeGitCommit(b.sourceCommit),
				),
			render: (_, release) => {
				const commit = normalizeGitCommit(release.sourceCommit);
				return commit ? (
					copyableCode(commit, formatCommitDisplay(release.sourceCommit))
				) : (
					<Typography.Text type="secondary">-</Typography.Text>
				);
			},
		},
		{
			title: "Tag",
			key: "tag",
			sorter: (a, b) => compareTextAsc(releaseTagText(a), releaseTagText(b)),
			render: (_, release) => <ReleaseTagCell release={release} />,
		},
		{
			title: "构建时间",
			key: "updatedAt",
			width: 160,
			sorter: (a, b) => compareDateAsc(a.updatedAt, b.updatedAt),
			defaultSortOrder: "descend",
			render: (_, release) => formatDateTime(release.updatedAt),
		},
		{
			title: "镜像",
			key: "imageUid",
			sorter: (a, b) => compareTextAsc(releaseImageUid(a), releaseImageUid(b)),
			render: (_, release) => {
				const imageTag = releaseImageTag(release);
				const imageUid = releaseImageUid(release);
				const imageRef = releaseImageReference(release);
				return (
					<Tooltip
						title={[
							imageRef ? `完整镜像: ${imageRef}` : "",
							imageTag ? `Tag: ${imageTag}` : "",
							imageUid ? `Digest 短码: ${imageUid}` : "",
						]
							.filter(Boolean)
							.join("\n")}
					>
						<Space direction="vertical" size={0} style={{ maxWidth: 180 }}>
							<Typography.Text
								code
								copyable={imageRef ? { text: imageRef } : false}
								className="component-library-release-grid__mono"
							>
								{imageTag || imageUid || "-"}
							</Typography.Text>
							{imageTag && imageUid ? (
								<Typography.Text type="secondary" style={{ fontSize: 11 }}>
									Digest {imageUid}
								</Typography.Text>
							) : null}
						</Space>
					</Tooltip>
				);
			},
		},
		{
			title: "操作",
			key: "actions",
			width: 108,
			render: (_, release) => (
				<Button
					type="link"
					size="small"
					icon={<EyeOutlined />}
					onClick={() => onViewRelease(release)}
				>
					查看构建详情
				</Button>
			),
		},
	];

	return (
		<Table
			size="small"
			rowKey="id"
			columns={columns}
			dataSource={releases}
			pagination={false}
			scroll={{ x: 960 }}
			className="component-library-release-table"
		/>
	);
}

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
		releaseTaskPath(release),
		releaseBuildTrigger(release),
		releaseImageUid(release),
		release.imageDigest,
		release.runtimeImage,
	]
		.filter(Boolean)
		.join(" ")
		.toLowerCase();

const releaseIdentityText = (release: PipelineComponentReleaseAPI): string =>
	[
		release.componentId,
		release.taskName,
		release.displayName,
		releaseTaskPath(release),
	]
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
	if (kind === "prod") return isOnlineRelease(release);
	return !isOnlineRelease(release);
};

const releaseMatchesSearch = (
	release: PipelineComponentReleaseAPI,
	query: string,
	mode: SearchMode,
): boolean => {
	if (!query) return true;
	if (mode === "commit") {
		const normalized = normalizeGitCommit(release.sourceCommit);
		return (
			normalized.toLowerCase().includes(query) ||
			(release.sourceCommit || "").toLowerCase().includes(query)
		);
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
					<Tooltip title={releaseVersionTooltip(release)}>
						<Tag color={releaseRefBadge(release).color}>
							{releaseVersionLabel(release)}
						</Tag>
					</Tooltip>
					<Tooltip title={releaseRefBadge(release).tooltip}>
						<Tag color={releaseKindColor(release)}>
							{releaseKindLabel(release)}
						</Tag>
					</Tooltip>
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
				<Descriptions.Item label="镜像 Tag">
					{releaseImageTag(release) || "-"}
				</Descriptions.Item>
				<Descriptions.Item label="镜像 Digest 短码">
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
					{releaseTaskPath(release) || "-"}
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
					<Typography.Text
						copyable={{
							text: normalizeGitCommit(release.sourceCommit) || "",
						}}
					>
						{formatCommitDisplay(release.sourceCommit)}
					</Typography.Text>
				</Descriptions.Item>
				<Descriptions.Item label="Build ID">
					{copyableCode(release.buildId, shortTechnicalValue(release.buildId))}
				</Descriptions.Item>
				<Descriptions.Item label="Build Trigger">
					{releaseBuildTrigger(release) || "-"}
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
							<Tooltip
								title={
									fields.length <= 1
										? `每个步骤至少需要一个${showOutputPath ? "输出" : "输入"}端口`
										: ""
								}
							>
								{/* Wrap in span so the tooltip still fires on the disabled button.
								    CYB-3094: forbid removing the last port — the pipeline stack
								    assumes every node has ≥1 input and ≥1 output port, so allowing
								    zero would silently re-inject defaults and mismatch the canvas. */}
								<span>
									<Button
										type="text"
										danger
										icon={<DeleteOutlined />}
										aria-label={`移除${title}`}
										disabled={fields.length <= 1}
										onClick={() => remove(field.name)}
									/>
								</span>
							</Tooltip>
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
	const screens = useBreakpoint();
	const isNarrow =
		typeof window !== "undefined" &&
		window.innerWidth < 768 &&
		screens.md !== true;
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
	const libraryRowsReady = !loading && !releaseLoading;
	const displayedLibraryRows = libraryRowsReady ? libraryRows : [];

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

	const getRowKey = (record: ComponentLibraryRow): string =>
		record.legacyComponent?.id || record.key;

	const toggleRowExpanded = (rowKey: string) => {
		setExpandedRowKeys((prev) =>
			prev.includes(rowKey)
				? prev.filter((key) => key !== rowKey)
				: [...prev, rowKey],
		);
	};

	const renderLibraryActions = (
		record: ComponentLibraryRow,
		variant: "table" | "mobile" = "table",
	) => {
		const component = record.legacyComponent;
		const linkedToReleaseLibrary = Boolean(record.primaryRelease && component);
		const showComponentActions =
			Boolean(component) &&
			!record.primaryRelease &&
			libraryTypeFilter !== "release";
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
			<Space
				size={variant === "mobile" ? 6 : 4}
				wrap
				className={
					variant === "mobile"
						? "component-library-mobile-card__actions"
						: "component-library-row-actions"
				}
				style={{
					whiteSpace: variant === "mobile" ? "normal" : "nowrap",
					justifyContent: variant === "mobile" ? "flex-start" : "flex-end",
				}}
			>
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
						{variant === "mobile" ? "查看版本" : "查看当前版本"}
					</Button>
				) : null}
				{linkedToReleaseLibrary && record.releases.length === 0 ? (
					<Tooltip title="该任务已接入版本库，等待 CI 同步镜像构建版本">
						<Typography.Text type="secondary" style={{ fontSize: 12 }}>
							暂无可用版本
						</Typography.Text>
					</Tooltip>
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
	};

	const renderMobileLibraryCard = (record: ComponentLibraryRow) => {
		const rowKey = getRowKey(record);
		const expanded = activeExpandedRowKeys.includes(rowKey);
		const canExpand = record.releases.length > 0;
		const imageTag = record.primaryRelease
			? releaseImageTag(record.primaryRelease)
			: "";
		const imageRef = record.primaryRelease
			? releaseImageReference(record.primaryRelease)
			: "";
		return (
			<li className="component-library-mobile-card" key={rowKey}>
				<div className="component-library-mobile-card__header">
					{canExpand ? (
						<Button
							type="text"
							size="small"
							className="component-library-row-expand"
							icon={expanded ? <DownOutlined /> : <RightOutlined />}
							aria-label={expanded ? "收起镜像构建版本" : "展开镜像构建版本"}
							aria-expanded={expanded}
							onClick={() => toggleRowExpanded(rowKey)}
						/>
					) : (
						<span className="component-library-row-expand component-library-row-expand--placeholder" />
					)}
					<Tooltip title={record.name}>
						<span className="component-library-mobile-card__title">
							{record.name}
						</span>
					</Tooltip>
				</div>
				<div className="component-library-mobile-card__meta">
					<Typography.Text type="secondary">
						{componentSourceSummary(record)}
					</Typography.Text>
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
				</div>
				<Tooltip
					title={[
						imageRef ? `完整镜像: ${imageRef}` : "",
						imageTag ? `Tag: ${imageTag}` : "",
						record.imageUid ? `Digest 短码: ${record.imageUid}` : "",
					]
						.filter(Boolean)
						.join("\n")}
				>
					<div className="component-library-mobile-card__image">
						<Typography.Text
							type="secondary"
							copyable={
								imageRef
									? { text: imageRef, tooltips: false }
									: record.imageUid
										? { text: record.imageUid, tooltips: false }
										: false
							}
						>
							{record.primaryRelease
								? `镜像 Tag：${imageTag || "-"}`
								: `镜像摘要：${record.imageUid || "-"}`}
						</Typography.Text>
						{record.primaryRelease && record.imageUid ? (
							<Typography.Text type="secondary">
								Digest：{record.imageUid}
							</Typography.Text>
						) : null}
					</div>
				</Tooltip>
				{renderLibraryActions(record, "mobile")}
				{canExpand && expanded ? (
					<div className="component-library-mobile-card__versions">
						{record.releases.map((release) => (
							<div
								key={release.id}
								className="component-library-mobile-card__version"
							>
								<ReleaseVersionChip
									release={release}
									onClick={() => openReleaseView(release)}
								/>
								<Typography.Text type="secondary">
									{releaseImageTag(release) || releaseImageUid(release) || "-"}
								</Typography.Text>
							</div>
						))}
					</div>
				) : null}
			</li>
		);
	};

	const libraryColumns: ColumnsType<ComponentLibraryRow> = [
		{
			title: "组件信息",
			key: "name",
			width: isNarrow ? 300 : 320,
			sorter: (a, b) => compareTextAsc(a.name, b.name),
			render: (_, record) => {
				const rowKey = getRowKey(record);
				const expanded = activeExpandedRowKeys.includes(rowKey);
				const canExpand = record.releases.length > 0;
				const linkedToReleaseLibrary = Boolean(
					record.primaryRelease && record.legacyComponent,
				);
				return (
					<div className="component-library-row-head">
						<div className="component-library-row-head__title">
							{canExpand ? (
								<Button
									type="text"
									size="small"
									className="component-library-row-expand"
									icon={expanded ? <DownOutlined /> : <RightOutlined />}
									aria-label={
										expanded ? "收起镜像构建版本" : "展开镜像构建版本"
									}
									aria-expanded={expanded}
									onClick={(event) => {
										event.stopPropagation();
										toggleRowExpanded(rowKey);
									}}
								/>
							) : (
								<span className="component-library-row-expand component-library-row-expand--placeholder" />
							)}
							<Typography.Text
								strong
								ellipsis={{ tooltip: record.name }}
								className="component-library-row-name"
							>
								{record.name}
							</Typography.Text>
						</div>
						<div className="component-library-row-head__meta">
							<Typography.Text type="secondary">
								{componentSourceSummary(record)}
							</Typography.Text>
							{linkedToReleaseLibrary && record.releases.length === 0 ? (
								<Typography.Text type="secondary">
									· 已关联版本库，暂无构建版本
								</Typography.Text>
							) : null}
							{canExpand ? (
								<Button
									type="link"
									size="small"
									className="component-library-row-inline-action component-library-row-inline-action--muted"
									onClick={() => toggleRowExpanded(rowKey)}
								>
									{buildVersionToggleLabel(record.releases.length, expanded)}
								</Button>
							) : null}
						</div>
						{record.primaryRelease ? (
							<Tooltip
								title={[
									releaseImageReference(record.primaryRelease)
										? `完整镜像: ${releaseImageReference(record.primaryRelease)}`
										: "",
									releaseImageTag(record.primaryRelease)
										? `Tag: ${releaseImageTag(record.primaryRelease)}`
										: "",
									record.imageUid ? `Digest 短码: ${record.imageUid}` : "",
								]
									.filter(Boolean)
									.join("\n")}
							>
								<div className="component-library-row-image-id">
									<Typography.Text
										type="secondary"
										copyable={
											releaseImageReference(record.primaryRelease)
												? {
														text: releaseImageReference(record.primaryRelease),
														tooltips: false,
													}
												: false
										}
									>
										镜像 Tag：{releaseImageTag(record.primaryRelease) || "-"}
									</Typography.Text>
									{record.imageUid ? (
										<Typography.Text type="secondary">
											Digest：{record.imageUid}
										</Typography.Text>
									) : null}
								</div>
							</Tooltip>
						) : (
							<Typography.Text
								type="secondary"
								copyable={
									record.imageUid
										? { text: record.imageUid, tooltips: false }
										: false
								}
								className="component-library-row-image-id"
							>
								镜像摘要：{record.imageUid || "-"}
							</Typography.Text>
						)}
					</div>
				);
			},
		},
		...(!isNarrow
			? [
					{
						title: "默认执行版本",
						key: "versions",
						width: 240,
						render: (_: unknown, record: ComponentLibraryRow) => {
							const tag = releaseTagText(record.primaryRelease);
							return (
								<Space direction="vertical" size={4} style={{ maxWidth: 240 }}>
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
									{record.releases.length === 0 && record.legacyComponent ? (
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											默认镜像：
											<Typography.Text code>
												{record.legacyComponent.tag || "latest"}
											</Typography.Text>
										</Typography.Text>
									) : null}
									{record.releases.length === 0 && !record.legacyComponent
										? "-"
										: null}
									{tag ? (
										<Typography.Text type="secondary" style={{ fontSize: 12 }}>
											Tag：{tag}
										</Typography.Text>
									) : null}
								</Space>
							);
						},
					},
					{
						title: "最近更新时间",
						key: "updatedAt",
						width: 160,
						sorter: (a: ComponentLibraryRow, b: ComponentLibraryRow) =>
							compareDateAsc(a.updatedAt, b.updatedAt),
						defaultSortOrder: "descend" as const,
						render: (_: unknown, record: ComponentLibraryRow) =>
							formatDateTime(record.updatedAt),
					},
				]
			: []),
		{
			title: "操作",
			key: "actions",
			width: isNarrow ? 150 : 180,
			fixed: isNarrow ? undefined : "right",
			className: "component-library-row-actions-cell",
			render: (_, record) => renderLibraryActions(record),
		},
	];

	return (
		<div
			className="component-library-page"
			style={{ display: "flex", flexDirection: "column", gap: 16 }}
		>
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
						<div
							style={{
								minWidth: isNarrow ? 0 : 280,
								flex: isNarrow ? "1 1 100%" : "0 1 340px",
							}}
						>
							<Input.Search
								id="component-manager-search"
								allowClear
								placeholder="搜索名称、commit、版本号"
								value={search}
								onChange={(event) => setSearch(event.target.value)}
								style={{ width: "100%" }}
							/>
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>
								可按任务名称、Commit、版本号搜索；先定位任务，再查看对应镜像版本。
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

			{libraryRowsReady && libraryRows.length === 0 ? (
				<Empty description="暂无组件；可以新建 legacy 组件，或同步平台生成的任务版本" />
			) : isNarrow ? (
				<ul className="component-library-mobile-list" aria-label="组件列表">
					{libraryRowsReady ? (
						displayedLibraryRows.map(renderMobileLibraryCard)
					) : (
						<div className="component-library-mobile-list__loading">
							<Spin />
						</div>
					)}
				</ul>
			) : (
				<Table
					className="component-library-table"
					rowKey={(record) => record.legacyComponent?.id || record.key}
					loading={loading || releaseLoading}
					columns={libraryColumns}
					dataSource={displayedLibraryRows}
					scroll={{ x: isNarrow ? 480 : 1100 }}
					tableLayout="fixed"
					onRow={(record) => {
						const cat = deriveCategory(record.name);
						return cat
							? { className: `component-cat component-cat--${cat.id}` }
							: {};
					}}
					expandable={{
						showExpandColumn: false,
						expandedRowKeys: activeExpandedRowKeys,
						onExpandedRowsChange: (keys) =>
							setExpandedRowKeys(keys.map(String)),
						rowExpandable: (record) => record.releases.length > 0,
						expandedRowRender: (record) => (
							<div className="component-library-expanded-panel">
								<Typography.Text
									strong
									className="component-library-expanded-panel__title"
								>
									镜像构建版本
								</Typography.Text>
								<Typography.Text
									type="secondary"
									className="component-library-expanded-panel__subtitle"
								>
									{record.name} · 共 {record.releases.length} 个镜像版本
								</Typography.Text>
								<ReleaseExpandedTable
									releases={record.releases}
									primaryReleaseId={record.primaryRelease?.id}
									onViewRelease={openReleaseView}
								/>
							</div>
						),
					}}
					pagination={{ pageSize: 12, showSizeChanger: true }}
				/>
			)}

			{releaseModalOpen && activeRelease ? (
				<Modal
					open
					title="镜像构建详情"
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
