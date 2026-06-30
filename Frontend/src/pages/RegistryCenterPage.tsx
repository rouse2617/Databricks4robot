import {
	ApartmentOutlined,
	BranchesOutlined,
	CheckCircleOutlined,
	CopyOutlined,
	DiffOutlined,
	EditOutlined,
	EllipsisOutlined,
	EyeOutlined,
	FileAddOutlined,
	InboxOutlined,
	PlusOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import Editor, { type Monaco } from "@monaco-editor/react";
import {
	Alert,
	Button,
	Card,
	Col,
	Collapse,
	Descriptions,
	Drawer,
	Dropdown,
	Empty,
	Form,
	Grid,
	Input,
	Modal,
	message,
	Row,
	Segmented,
	Select,
	Space,
	Table,
	Tabs,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { AlgoRegistryItem } from "../api/algoRegistry";
import {
	type PipelineConfig,
	type PipelineConfigVersion,
	pipelineConfigApi,
} from "../api/pipelineConfigs";
import { type MetricRegistryItem, registryApi } from "../api/registry";
import type { TagRegistryItem } from "../api/tagRegistry";
import {
	ContentErrorState,
	ContentLoadingState,
} from "../components/common/PageContentState";
import PoolManager from "../components/pipeline/PoolManager";
import { COLUMN_LABELS } from "../lib/productVocabulary";

const { Title, Text, Paragraph } = Typography;

interface UserConfigRecord {
	id: string;
	name: string;
	owner: string;
	description: string;
	tags: string[];
	fileType: "yaml" | "json";
	lifecycle: "draft" | "ready" | "deprecated";
	latestVersion: string;
	currentVersion: number;
	versionCount: number;
	updatedAt: string;
	versions: ConfigVersionRecord[];
}

interface ConfigVersionRecord {
	version: string;
	versionNumber: number;
	lifecycle: "draft" | "ready" | "deprecated";
	updatedAt: string;
	author: string;
	summary: string;
	contentSha256: string;
	contentSizeBytes: number;
	content?: string;
}

interface SelectedVersionContent {
	configName: string;
	version: ConfigVersionRecord;
}

interface SelectedVersionCompare {
	configName: string;
	left: ConfigVersionRecord;
	right: ConfigVersionRecord;
	rows: VersionDiffRow[];
}

interface VersionDiffRow {
	key: string;
	kind: "equal" | "added" | "removed" | "changed";
	leftLine?: number;
	rightLine?: number;
	leftText?: string;
	rightText?: string;
}

interface ConfigFormValues {
	name: string;
	description: string;
	tags?: string[];
	lifecycle: UserConfigRecord["lifecycle"];
	versionSummary?: string;
	content?: string;
}

interface VersionFormValues {
	summary: string;
	lifecycle: ConfigVersionRecord["lifecycle"];
	content: string;
}

interface CompareFormValues {
	leftVersion?: number;
	rightVersion?: number;
}

type ConfigShelf = "active" | "archived";

function versionLabel(version: number) {
	return `v${version}`;
}

function formatConfigTime(value?: string) {
	if (!value) return "—";
	return dayjs(value).format("YYYY-MM-DD HH:mm");
}

function inferConfigFileType(name: string): UserConfigRecord["fileType"] {
	return name.toLowerCase().endsWith(".json") ? "json" : "yaml";
}

function shortHash(value?: string) {
	return value ? value.slice(0, 8) : "—";
}

function versionDescriptor(version: ConfigVersionRecord) {
	return `${version.version} · ${version.summary || "无说明"} · ${version.updatedAt} · ${shortHash(version.contentSha256)}`;
}

function lifecycleTagColor(
	lifecycle: UserConfigRecord["lifecycle"] | ConfigVersionRecord["lifecycle"],
) {
	if (lifecycle === "ready") return "green";
	if (lifecycle === "draft") return "gold";
	return "default";
}

function lifecycleDisplay(
	lifecycle: UserConfigRecord["lifecycle"] | ConfigVersionRecord["lifecycle"],
) {
	return lifecycle === "deprecated" ? "archived" : lifecycle;
}

function compactUserName(value?: string) {
	if (!value) return "—";
	const [name] = value.split("@");
	return name || value;
}

function tagColor(value: string) {
	const normalized = value.trim().toLowerCase();
	if (normalized.includes("prod") || normalized.includes("ready"))
		return "green";
	if (normalized.includes("test") || normalized.includes("smoke"))
		return "blue";
	if (normalized.includes("ui") || normalized.includes("probe"))
		return "purple";
	if (normalized.includes("old") || normalized.includes("archive"))
		return "default";
	return "geekblue";
}

const VALID_CONFIG_EXTENSIONS = [".yaml", ".yml", ".json"];
const CONFIG_NAME_PATTERN =
	/^[a-z0-9][a-z0-9-]*(\.[a-z0-9][a-z0-9-]*)*\.(yaml|yml|json)$/;
const CONFIG_NAME_MAX_LENGTH = 96;

function looksLikeTypoConfigName(name?: string) {
	const lower = (name || "").trim().toLowerCase();
	if (lower.endsWith(".ymal"))
		return "文件后缀看起来像 .ymal，通常应为 .yaml 或 .yml。";
	if (/\.(yam|yml|yaml|ym|josn|jso|yml\.yaml)$/.test(lower))
		return "文件后缀可能拼写有误，有效后缀：.yaml / .yml / .json";
	return null;
}

function configNameExtra(name?: string): string {
	const trimmed = (name || "").trim();
	if (!trimmed)
		return "推荐格式：<组件>-<环境>.yaml，例如 head-track-detector.yaml、hand-detect-dev.yaml、node-a-config.yaml";

	const typo = looksLikeTypoConfigName(trimmed);
	if (typo) return `⚠ ${typo}`;

	const hasExtension = VALID_CONFIG_EXTENSIONS.some((ext) =>
		trimmed.toLowerCase().endsWith(ext),
	);
	if (!hasExtension) return "⚠ 需要文件扩展名 .yaml / .yml / .json";

	if (/[A-Z]/.test(trimmed)) return "⚠ 建议使用全小写字母";

	if (trimmed.length > CONFIG_NAME_MAX_LENGTH)
		return `⚠ 名称不能超过 ${CONFIG_NAME_MAX_LENGTH} 字符`;

	return "✓ 格式正确";
}

function isArchivedConfig(config: UserConfigRecord) {
	return config.lifecycle === "deprecated";
}

function mapConfigVersion(version: PipelineConfigVersion): ConfigVersionRecord {
	return {
		version: versionLabel(version.version),
		versionNumber: version.version,
		lifecycle: version.status,
		updatedAt: formatConfigTime(version.createdAt),
		author: version.author || "—",
		summary: version.summary || "—",
		contentSha256: version.contentSha256,
		contentSizeBytes: version.contentSizeBytes,
		content: version.content,
	};
}

function mapConfig(config: PipelineConfig): UserConfigRecord {
	return {
		id: config.id,
		name: config.name,
		owner: config.owner,
		description: config.description || "—",
		tags: config.tags ?? [],
		fileType: config.fileType,
		lifecycle: config.lifecycle,
		latestVersion: versionLabel(config.currentVersion),
		currentVersion: config.currentVersion,
		versionCount: config.versionCount,
		updatedAt: formatConfigTime(config.updatedAt),
		versions: (config.versions ?? []).map(mapConfigVersion),
	};
}

async function fetchConfigRecords(): Promise<UserConfigRecord[]> {
	const list = await pipelineConfigApi.list();
	const details = await Promise.all(
		list.items.map(async (config) => {
			try {
				return await pipelineConfigApi.get(config.id);
			} catch {
				return config;
			}
		}),
	);
	return details.map(mapConfig);
}

function splitConfigContent(content?: string) {
	if (!content) return [""];
	return content.split(/\r?\n/);
}

function buildIndexDiffRows(left: string[], right: string[]): VersionDiffRow[] {
	const length = Math.max(left.length, right.length);
	return Array.from({ length }, (_, index) => {
		const hasLeft = index < left.length;
		const hasRight = index < right.length;
		const leftText = hasLeft ? left[index] : undefined;
		const rightText = hasRight ? right[index] : undefined;
		const kind =
			hasLeft && hasRight
				? leftText === rightText
					? "equal"
					: "changed"
				: hasLeft
					? "removed"
					: "added";
		return {
			key: `fallback-${index}`,
			kind,
			leftLine: hasLeft ? index + 1 : undefined,
			rightLine: hasRight ? index + 1 : undefined,
			leftText,
			rightText,
		};
	});
}

function buildVersionDiffRows(leftContent?: string, rightContent?: string) {
	const left = splitConfigContent(leftContent);
	const right = splitConfigContent(rightContent);
	if (left.length * right.length > 250_000) {
		return buildIndexDiffRows(left, right);
	}

	const dp = Array.from({ length: left.length + 1 }, () =>
		Array(right.length + 1).fill(0),
	);
	for (let i = left.length - 1; i >= 0; i -= 1) {
		for (let j = right.length - 1; j >= 0; j -= 1) {
			dp[i][j] =
				left[i] === right[j]
					? dp[i + 1][j + 1] + 1
					: Math.max(dp[i + 1][j], dp[i][j + 1]);
		}
	}

	const ops: VersionDiffRow[] = [];
	let i = 0;
	let j = 0;
	while (i < left.length && j < right.length) {
		if (left[i] === right[j]) {
			ops.push({
				key: `equal-${i}-${j}`,
				kind: "equal",
				leftLine: i + 1,
				rightLine: j + 1,
				leftText: left[i],
				rightText: right[j],
			});
			i += 1;
			j += 1;
		} else if (dp[i + 1][j] >= dp[i][j + 1]) {
			ops.push({
				key: `removed-${i}`,
				kind: "removed",
				leftLine: i + 1,
				leftText: left[i],
			});
			i += 1;
		} else {
			ops.push({
				key: `added-${j}`,
				kind: "added",
				rightLine: j + 1,
				rightText: right[j],
			});
			j += 1;
		}
	}
	while (i < left.length) {
		ops.push({
			key: `removed-${i}`,
			kind: "removed",
			leftLine: i + 1,
			leftText: left[i],
		});
		i += 1;
	}
	while (j < right.length) {
		ops.push({
			key: `added-${j}`,
			kind: "added",
			rightLine: j + 1,
			rightText: right[j],
		});
		j += 1;
	}

	const rows: VersionDiffRow[] = [];
	for (let index = 0; index < ops.length; index += 1) {
		const current = ops[index];
		const next = ops[index + 1];
		if (current.kind === "removed" && next?.kind === "added") {
			rows.push({
				key: `changed-${current.leftLine}-${next.rightLine}`,
				kind: "changed",
				leftLine: current.leftLine,
				rightLine: next.rightLine,
				leftText: current.leftText,
				rightText: next.rightText,
			});
			index += 1;
		} else {
			rows.push(current);
		}
	}
	return rows;
}

function diffCellBackground(
	kind: VersionDiffRow["kind"],
	side: "left" | "right",
) {
	if (kind === "equal") return "#ffffff";
	if (kind === "changed") return "#fff3cd";
	if (kind === "removed" && side === "left") return "#fddcd7";
	if (kind === "added" && side === "right") return "#dafbe1";
	return "#f8fafc";
}

function UserTag({ value }: { value?: string }) {
	const display = compactUserName(value);
	return (
		<Tooltip title={value || "—"}>
			<Tag color="blue" style={{ maxWidth: 132, marginInlineEnd: 0 }}>
				<Text
					style={{
						maxWidth: 108,
						display: "inline-block",
						verticalAlign: "bottom",
					}}
					ellipsis={{ tooltip: value }}
				>
					{display}
				</Text>
			</Tag>
		</Tooltip>
	);
}

function ConfigTagList({ tags }: { tags: string[] }) {
	if (!tags?.length) return <Text type="secondary">—</Text>;
	return (
		<Space wrap size={4}>
			{tags.map((tag) => (
				<Tag key={tag} color={tagColor(tag)}>
					{tag}
				</Tag>
			))}
		</Space>
	);
}

function ConfigTitleCell({ row }: { row: UserConfigRecord }) {
	const typo = looksLikeTypoConfigName(row.name);
	return (
		<div style={{ minWidth: 0 }}>
			<Space size={6} style={{ maxWidth: "100%" }}>
				<Text strong ellipsis={{ tooltip: row.name }} style={{ maxWidth: 260 }}>
					{row.name}
				</Text>
				<Tag>{row.fileType}</Tag>
				{typo ? (
					<Tooltip title="文件后缀可能写错，通常应为 .yaml 或 .yml">
						<Tag color="gold" style={{ marginInlineEnd: 0 }}>
							后缀
						</Tag>
					</Tooltip>
				) : null}
			</Space>
			<Text
				style={{
					display: "block",
					color: "#475569",
					fontSize: 12,
					lineHeight: 1.45,
					maxWidth: 420,
				}}
				ellipsis={{ tooltip: row.description }}
			>
				{row.description}
			</Text>
		</div>
	);
}

function IconActionButton({
	title,
	icon,
	onClick,
	disabled,
	danger,
	loading,
}: {
	title: string;
	icon: React.ReactNode;
	onClick?: () => void;
	disabled?: boolean;
	danger?: boolean;
	loading?: boolean;
}) {
	return (
		<Tooltip title={title}>
			<span>
				<Button
					size="small"
					type="text"
					aria-label={title}
					icon={icon}
					onClick={onClick}
					disabled={disabled}
					danger={danger}
					loading={loading}
				/>
			</span>
		</Tooltip>
	);
}

function TableScrollBoundary({ children }: { children: React.ReactNode }) {
	return (
		<div
			style={{
				width: "100%",
				maxWidth: "100%",
				minWidth: 0,
				overflowX: "auto",
			}}
		>
			{children}
		</div>
	);
}

export default function RegistryCenterPage() {
	const [msg, msgCtx] = message.useMessage();
	const [configRecords, setConfigRecords] = useState<UserConfigRecord[]>([]);
	const [configQuery, setConfigQuery] = useState("");
	const [configShelf, setConfigShelf] = useState<ConfigShelf>("active");
	const [selectedConfigId, setSelectedConfigId] = useState<string | null>(null);
	const [editingConfig, setEditingConfig] = useState<UserConfigRecord | null>(
		null,
	);
	const [configModalOpen, setConfigModalOpen] = useState(false);
	const [versionTarget, setVersionTarget] = useState<UserConfigRecord | null>(
		null,
	);
	const [versionSource, setVersionSource] =
		useState<ConfigVersionRecord | null>(null);
	const [selectedVersionContent, setSelectedVersionContent] =
		useState<SelectedVersionContent | null>(null);
	const [compareTarget, setCompareTarget] = useState<UserConfigRecord | null>(
		null,
	);
	const [selectedVersionCompare, setSelectedVersionCompare] =
		useState<SelectedVersionCompare | null>(null);
	const [configForm] = Form.useForm<ConfigFormValues>();
	const [versionForm] = Form.useForm<VersionFormValues>();
	const [compareForm] = Form.useForm<CompareFormValues>();
	const [algos, setAlgos] = useState<AlgoRegistryItem[]>([]);
	const [tags, setTags] = useState<TagRegistryItem[]>([]);
	const [metrics, setMetrics] = useState<MetricRegistryItem[]>([]);
	const [states, setStates] = useState<string[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [configsLoading, setConfigsLoading] = useState(false);
	const [savingConfig, setSavingConfig] = useState(false);
	const [savingVersion, setSavingVersion] = useState(false);
	const [, setDeprecatingConfigId] = useState<string | null>(null);
	const [compareLoading, setCompareLoading] = useState(false);
	const [diffPreviewRows, setDiffPreviewRows] = useState<VersionDiffRow[]>([]);
	const [diffBaseVersionNumber, setDiffBaseVersionNumber] = useState<
		number | null
	>(null);
	const [diffBaseContent, setDiffBaseContent] = useState<string>("");
	const diffTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	const [editVersionState, setEditVersionState] = useState<{
		visible: boolean;
		configId: string;
		versionNumber: number;
		content: string;
		summary: string;
		saving: boolean;
	}>({
		visible: false,
		configId: "",
		versionNumber: 0,
		content: "",
		summary: "",
		saving: false,
	});

	// When edit modal opens, load version content from API
	useEffect(() => {
		if (
			editVersionState.visible &&
			editVersionState.configId &&
			editVersionState.versionNumber
		) {
			pipelineConfigApi
				.getVersion(editVersionState.configId, editVersionState.versionNumber)
				.then((detail) => {
					setEditVersionState((prev) => ({
						...prev,
						content: detail.content || "",
					}));
				})
				.catch(() => {
					msg.error("读取版本内容失败");
				});
		}
	}, [
		editVersionState.visible,
		editVersionState.configId,
		editVersionState.versionNumber,
		msg.error,
	]);

	const versionContentValue = Form.useWatch("content", versionForm);

	const loadDiffBaseVersion = useCallback(
		async (configId: string, versionNumber: number) => {
			try {
				const detail = await pipelineConfigApi.getVersion(
					configId,
					versionNumber,
				);
				setDiffBaseContent(detail.content || "");
			} catch {
				setDiffBaseContent("");
			}
		},
		[],
	);

	// When modal opens or versionSource changes, default to source version
	useEffect(() => {
		if (versionTarget && versionSource) {
			const vn = versionSource.versionNumber;
			setDiffBaseVersionNumber(vn);
			setDiffBaseContent(versionSource.content || "");
		}
	}, [versionTarget, versionSource]);

	// Debounced diff computation
	useEffect(() => {
		if (diffTimerRef.current) clearTimeout(diffTimerRef.current);
		diffTimerRef.current = setTimeout(() => {
			if (diffBaseContent && versionContentValue) {
				setDiffPreviewRows(
					buildVersionDiffRows(diffBaseContent, versionContentValue),
				);
			} else {
				setDiffPreviewRows([]);
			}
		}, 400);
		return () => {
			if (diffTimerRef.current) clearTimeout(diffTimerRef.current);
		};
	}, [versionContentValue, diffBaseContent]);
	const configNameValue = Form.useWatch("name", configForm);
	const screens = Grid.useBreakpoint();
	const isNarrow = !screens.md;

	const refreshConfigs = async () => {
		setConfigsLoading(true);
		try {
			const records = await fetchConfigRecords();
			setConfigRecords(records);
			if (
				selectedConfigId &&
				!records.some((record) => record.id === selectedConfigId)
			) {
				setSelectedConfigId(null);
			}
		} catch {
			msg.error("加载配置库失败");
		} finally {
			setConfigsLoading(false);
		}
	};

	useEffect(() => {
		let cancelled = false;
		(async () => {
			try {
				setLoading(true);
				const [a, t, m, s, configs] = await Promise.all([
					registryApi.listAlgos(),
					registryApi.listTags(),
					registryApi.listMetrics(),
					registryApi.listLifecycleStates(),
					fetchConfigRecords(),
				]);
				if (cancelled) return;
				setAlgos(a);
				setTags(t);
				setMetrics(m);
				setStates(s);
				setConfigRecords(configs);
			} catch {
				if (!cancelled) setError("加载注册中心失败");
			} finally {
				if (!cancelled) setLoading(false);
			}
		})();
		return () => {
			cancelled = true;
		};
	}, []);

	const selectedConfig = useMemo(
		() => configRecords.find((cfg) => cfg.id === selectedConfigId) ?? null,
		[configRecords, selectedConfigId],
	);

	const filteredConfigs = useMemo(() => {
		const query = configQuery.trim().toLowerCase();
		const scopedRecords = configRecords.filter((cfg) =>
			configShelf === "archived"
				? isArchivedConfig(cfg)
				: !isArchivedConfig(cfg),
		);
		if (!query) return scopedRecords;
		return scopedRecords.filter((cfg) =>
			[
				cfg.name,
				cfg.owner,
				cfg.description,
				cfg.fileType,
				cfg.lifecycle,
				lifecycleDisplay(cfg.lifecycle),
				cfg.latestVersion,
				...cfg.tags,
				...cfg.versions.map((v) => v.summary),
				...cfg.versions.map((v) => v.version),
			]
				.join(" ")
				.toLowerCase()
				.includes(query),
		);
	}, [configRecords, configQuery, configShelf]);

	const availableConfigTags = useMemo(
		() => Array.from(new Set(configRecords.flatMap((cfg) => cfg.tags))).sort(),
		[configRecords],
	);

	const openCreateConfig = () => {
		setEditingConfig(null);
		configForm.resetFields();
		configForm.setFieldsValue({
			lifecycle: "draft",
			tags: [],
			versionSummary: "首次注册",
			content: "",
		});
		setConfigModalOpen(true);
	};

	const openEditConfig = (record: UserConfigRecord) => {
		setEditingConfig(record);
		configForm.setFieldsValue({
			name: record.name,
			description: record.description,
			tags: record.tags,
			lifecycle: record.lifecycle,
		});
		setConfigModalOpen(true);
	};

	const handleSaveConfig = (values: ConfigFormValues) => {
		void (async () => {
			setSavingConfig(true);
			try {
				if (editingConfig) {
					const updated = await pipelineConfigApi.update(editingConfig.id, {
						name: values.name,
						description: values.description,
						tags: values.tags ?? [],
						fileType: inferConfigFileType(values.name),
						lifecycle: values.lifecycle,
					});
					setSelectedConfigId(updated.id);
					msg.success("配置已更新");
				} else {
					const created = await pipelineConfigApi.create({
						name: values.name,
						description: values.description,
						tags: values.tags ?? [],
						fileType: inferConfigFileType(values.name),
						lifecycle: values.lifecycle,
						content: values.content || "",
						summary: values.versionSummary || "首次注册",
					});
					setSelectedConfigId(created.id);
					msg.success("配置已创建");
				}
				await refreshConfigs();
				setConfigModalOpen(false);
				setEditingConfig(null);
				configForm.resetFields();
			} catch (err: any) {
				const detail: string = err?.response?.data?.message || "";
				if (detail.toLowerCase().includes("already exists")) {
					msg.error("配置名称已存在，请换一个名称");
				} else {
					msg.error(editingConfig ? "配置更新失败" : "配置创建失败");
				}
			} finally {
				setSavingConfig(false);
			}
		})();
	};

	const openCreateVersion = (
		record: UserConfigRecord,
		sourceVersion?: ConfigVersionRecord,
	) => {
		void (async () => {
			setVersionTarget(record);
			versionForm.resetFields();
			const latestVersion =
				sourceVersion ??
				record.versions.find(
					(version) => version.version === record.latestVersion,
				);
			if (!latestVersion) {
				setVersionSource(null);
				versionForm.setFieldsValue({
					lifecycle: "draft",
					summary: "",
					content: "",
				});
				return;
			}
			try {
				const detail = await pipelineConfigApi.getVersion(
					record.id,
					latestVersion.versionNumber,
				);
				const source = mapConfigVersion(detail);
				setVersionSource(source);
				versionForm.setFieldsValue({
					lifecycle: "draft",
					summary: `基于 ${source.version} 编辑`,
					content: detail.content ?? "",
				});
			} catch {
				setVersionSource(latestVersion);
				versionForm.setFieldsValue({
					lifecycle: "draft",
					summary: `基于 ${latestVersion.version} 编辑`,
					content: latestVersion.content ?? "",
				});
				msg.error("读取源版本内容失败");
			}
		})();
	};

	const handleCreateVersion = (values: VersionFormValues) => {
		if (!versionTarget) return;
		void (async () => {
			setSavingVersion(true);
			try {
				await pipelineConfigApi.createVersion(versionTarget.id, {
					status: values.lifecycle,
					content: values.content,
					summary: values.summary,
				});
				setSelectedConfigId(versionTarget.id);
				await refreshConfigs();
				setVersionTarget(null);
				setVersionSource(null);
				versionForm.resetFields();
				msg.success("新版本已创建");
			} catch {
				msg.error("新版本创建失败");
			} finally {
				setSavingVersion(false);
			}
		})();
	};

	const handleDeprecateConfig = (record: UserConfigRecord) => {
		void (async () => {
			setDeprecatingConfigId(record.id);
			try {
				await pipelineConfigApi.deprecate(record.id);
				await refreshConfigs();
				msg.success("配置已移入归档区");
			} catch {
				msg.error("配置归档失败");
			} finally {
				setDeprecatingConfigId(null);
			}
		})();
	};

	const handleOpenEditVersion = (
		config: UserConfigRecord,
		version: ConfigVersionRecord,
	) => {
		setEditVersionState({
			visible: true,
			configId: config.id,
			versionNumber: version.versionNumber,
			content: "",
			summary: version.summary === "—" ? "" : version.summary || "",
			saving: false,
		});
	};

	const handleOpenVersionCompare = (
		config: UserConfigRecord,
		version?: ConfigVersionRecord,
	) => {
		setCompareTarget(config);
		setSelectedVersionCompare(null);
		const versions = [...config.versions].sort(
			(a, b) => b.versionNumber - a.versionNumber,
		);
		const rightVersion =
			config.currentVersion ||
			versions[0]?.versionNumber ||
			version?.versionNumber;
		const leftVersion =
			version?.versionNumber && version.versionNumber !== rightVersion
				? version.versionNumber
				: versions.find((item) => item.versionNumber !== rightVersion)
						?.versionNumber;
		compareForm.setFieldsValue({ leftVersion, rightVersion });
		if (leftVersion && rightVersion) {
			void loadVersionCompare(config, leftVersion, rightVersion);
		}
	};

	const loadVersionCompare = async (
		config: UserConfigRecord,
		leftVersion: number,
		rightVersion: number,
	) => {
		setCompareLoading(true);
		try {
			const [leftDetail, rightDetail] = await Promise.all([
				pipelineConfigApi.getVersion(config.id, leftVersion),
				pipelineConfigApi.getVersion(config.id, rightVersion),
			]);
			const left = mapConfigVersion(leftDetail);
			const right = mapConfigVersion(rightDetail);
			setSelectedVersionCompare({
				configName: config.name,
				left,
				right,
				rows: buildVersionDiffRows(left.content, right.content),
			});
		} catch {
			msg.error("读取对比版本失败");
		} finally {
			setCompareLoading(false);
		}
	};

	const handleCompareSubmit = async () => {
		if (!compareTarget) return;
		const values = await compareForm.validateFields();
		if (!values.leftVersion || !values.rightVersion) return;
		if (values.leftVersion === values.rightVersion) {
			msg.warning("请选择两个不同版本");
			return;
		}
		await loadVersionCompare(
			compareTarget,
			values.leftVersion,
			values.rightVersion,
		);
	};

	const algoCols: ColumnsType<AlgoRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{ title: "名称", dataIndex: "name" },
		{ title: "版本", dataIndex: "version" },
		{
			title: COLUMN_LABELS.dependsOn,
			dataIndex: "depends_on",
			render: (v: string[]) => v?.join(", ") || "—",
		},
	];
	const tagCols: ColumnsType<TagRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{ title: "类型", dataIndex: "type" },
		{
			title: "说明",
			dataIndex: "description",
			render: (v?: string) => v || "—",
		},
	];
	const metricCols: ColumnsType<MetricRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{
			title: "展示名",
			dataIndex: "display_name",
			render: (v?: string) => v || "—",
		},
		{ title: "类型", dataIndex: "metric_type" },
		{
			title: COLUMN_LABELS.queryable,
			dataIndex: "queryable",
			render: (v?: boolean) =>
				v ? <Tag color="green">是</Tag> : <Tag>否</Tag>,
		},
	];

	const configCols: ColumnsType<UserConfigRecord> = [
		{
			title: "配置",
			dataIndex: "name",
			render: (_, row) => <ConfigTitleCell row={row} />,
		},
		{
			title: "拥有者",
			dataIndex: "owner",
			width: 150,
			render: (v) => <UserTag value={v} />,
		},
		{
			title: "状态",
			dataIndex: "lifecycle",
			width: 120,
			render: (v: UserConfigRecord["lifecycle"]) => (
				<Tag color={lifecycleTagColor(v)}>{lifecycleDisplay(v)}</Tag>
			),
		},
		{
			title: "当前版本",
			dataIndex: "latestVersion",
			width: 120,
			render: (v) => <Tag color="geekblue">{v}</Tag>,
		},
		{
			title: "版本数",
			dataIndex: "versions",
			width: 110,
			render: (_versions: ConfigVersionRecord[], row) => (
				<Space size={6}>
					<BranchesOutlined />
					<Text>{row.versionCount}</Text>
				</Space>
			),
		},
		{
			title: "标签",
			dataIndex: "tags",
			render: (v: string[]) => <ConfigTagList tags={v} />,
		},
		{
			title: "操作",
			key: "actions",
			width: 250,
			align: "right",
			render: (_, record) => {
				const archived = isArchivedConfig(record);
				return (
					<Space size={2} wrap>
						{archived ? null : (
							<Button
								size="small"
								type="primary"
								icon={<FileAddOutlined />}
								onClick={(e) => {
									e.stopPropagation();
									openCreateVersion(record);
								}}
							>
								新建版本
							</Button>
						)}
						<IconActionButton
							title="查看详情"
							icon={<EyeOutlined />}
							onClick={() => setSelectedConfigId(record.id)}
						/>
						<Dropdown
							menu={{
								items: [
									...(archived
										? []
										: [
												{
													key: "edit",
													icon: <EditOutlined />,
													label: "编辑属性",
													onClick: () => openEditConfig(record),
												},
											]),
									...(record.lifecycle === "draft" && !archived
										? [
												{
													key: "ready",
													icon: <CheckCircleOutlined />,
													label: "发布为 Ready",
													onClick: async () => {
														try {
															await pipelineConfigApi.updateVersionStatus(
																record.id,
																record.currentVersion,
																"ready",
															);
															msg.success("已发布为 Ready");
															refreshConfigs();
														} catch (err) {
															msg.error(`发布失败: ${err}`);
														}
													},
												},
											]
										: []),
									...(record.versions.length >= 2
										? [
												{
													key: "diff",
													icon: <DiffOutlined />,
													label: "对比版本",
													onClick: () => handleOpenVersionCompare(record),
												},
											]
										: []),
									...(!archived
										? [
												{ key: "divider", type: "divider" },
												{
													key: "archive",
													icon: <InboxOutlined />,
													label: "归档配置",
													danger: true,
													onClick: () => handleDeprecateConfig(record),
												},
											]
										: []),
								].filter(Boolean) as any,
							}}
						>
							<Button size="small" icon={<EllipsisOutlined />}>
								更多
							</Button>
						</Dropdown>
					</Space>
				);
			},
		},
	];

	const ContentPreviewTab = ({ config: cfg }: { config: UserConfigRecord }) => {
		const [content, setContent] = useState<string | null>(null);
		const [loading, setLoading] = useState(false);
		useEffect(() => {
			void (async () => {
				setLoading(true);
				try {
					const detail = await pipelineConfigApi.getVersion(
						cfg.id,
						cfg.currentVersion,
					);
					setContent(detail.content || "");
				} catch {
					setContent(null);
				} finally {
					setLoading(false);
				}
			})();
		}, [cfg.id, cfg.currentVersion]);
		if (loading) return <ContentLoadingState title="加载内容…" />;
		if (content === null)
			return (
				<ContentErrorState title="加载失败" onRetry={() => setLoading(true)} />
			);
		if (!content)
			return (
				<Empty
					description="版本内容为空"
					image={Empty.PRESENTED_IMAGE_SIMPLE}
				/>
			);
		return (
			<div>
				<div
					style={{
						marginBottom: 8,
						display: "flex",
						justifyContent: "flex-end",
						gap: 8,
					}}
				>
					<Button
						size="small"
						icon={<CopyOutlined />}
						onClick={() => {
							navigator.clipboard.writeText(content);
							msg.success("已复制");
						}}
					>
						复制
					</Button>
				</div>
				<Input.TextArea
					rows={20}
					value={content}
					readOnly
					style={{
						fontFamily: "monospace",
						fontSize: 12,
						background: "#f8fafc",
					}}
				/>
			</div>
		);
	};

	const VersionHistoryTab = ({ config: cfg }: { config: UserConfigRecord }) => {
		const [loadingVersionKey, setLocalLoading] = useState<string | null>(null);
		const handleViewContent = async (version: ConfigVersionRecord) => {
			const key = `${cfg.id}:${version.versionNumber}`;
			setLocalLoading(key);
			try {
				const detail = await pipelineConfigApi.getVersion(
					cfg.id,
					version.versionNumber,
				);
				setSelectedVersionContent({
					configName: cfg.name,
					version: mapConfigVersion(detail),
				});
			} catch {
				msg.error("读取失败");
			} finally {
				setLocalLoading(null);
			}
		};
		const handlePublishReady = async (version: ConfigVersionRecord) => {
			try {
				await pipelineConfigApi.updateVersionStatus(
					cfg.id,
					version.versionNumber,
					"ready",
				);
				msg.success("已发布为 Ready");
				refreshConfigs();
			} catch (err) {
				msg.error(`发布失败: ${err}`);
			}
		};
		return (
			<Space direction="vertical" size={8} style={{ width: "100%" }}>
				{cfg.versions.length === 0 ? (
					<Empty description="暂无版本" image={Empty.PRESENTED_IMAGE_SIMPLE} />
				) : (
					[...cfg.versions]
						.sort((a, b) => b.versionNumber - a.versionNumber)
						.map((version) => {
							const isCurrent = version.versionNumber === cfg.currentVersion;
							const isDraft = version.lifecycle === "draft";
							return (
								<Card
									key={version.versionNumber}
									size="small"
									style={{
										borderLeft: isCurrent
											? "3px solid #1677ff"
											: "1px solid #d9d9d9",
										background: isCurrent ? "#f0f5ff" : undefined,
									}}
								>
									<div
										style={{
											display: "flex",
											justifyContent: "space-between",
											alignItems: "center",
											marginBottom: 4,
										}}
									>
										<Space size={6}>
											<Text code strong style={{ fontSize: 14 }}>
												{version.version}
											</Text>
											{isCurrent ? (
												<Tag color="blue" style={{ marginRight: 0 }}>
													当前版本
												</Tag>
											) : null}
											<Tag
												color={lifecycleTagColor(version.lifecycle)}
												style={{ marginRight: 0 }}
											>
												{lifecycleDisplay(version.lifecycle)}
											</Tag>
										</Space>
									</div>
									<div
										style={{ fontSize: 12, color: "#64748b", marginBottom: 6 }}
									>
										{version.author} · {version.updatedAt}
									</div>
									<Text style={{ color: "#475569", fontSize: 13 }}>
										{version.summary}
									</Text>
									<div style={{ marginTop: 8 }}>
										<Space size={4}>
											<Button
												size="small"
												type="link"
												icon={<EyeOutlined />}
												loading={
													loadingVersionKey ===
													`${cfg.id}:${version.versionNumber}`
												}
												onClick={() => handleViewContent(version)}
											>
												查看
											</Button>
											{cfg.versions.length >= 2 ? (
												<Button
													size="small"
													type="link"
													icon={<DiffOutlined />}
													onClick={() => handleOpenVersionCompare(cfg, version)}
												>
													对比
												</Button>
											) : null}
											<Button
												size="small"
												type="link"
												icon={<FileAddOutlined />}
												onClick={() => openCreateVersion(cfg, version)}
											>
												基于此版本创建
											</Button>
											{isDraft && !isArchivedConfig(cfg) ? (
												<>
													<Button
														size="small"
														type="link"
														icon={<CheckCircleOutlined />}
														style={{ color: "#52c41a" }}
														onClick={() => handlePublishReady(version)}
													>
														发布为 Ready
													</Button>
													<Button
														size="small"
														type="link"
														icon={<EditOutlined />}
														style={{ color: "#faad14" }}
														onClick={() => handleOpenEditVersion(cfg, version)}
													>
														编辑
													</Button>
												</>
											) : null}
										</Space>
									</div>
								</Card>
							);
						})
				)}
			</Space>
		);
	};

	const configStats = useMemo(
		() => ({
			total: configRecords.length,
			active: configRecords.filter((cfg) => !isArchivedConfig(cfg)).length,
			ready: configRecords.filter((cfg) => cfg.lifecycle === "ready").length,
			draft: configRecords.filter((cfg) => cfg.lifecycle === "draft").length,
			archived: configRecords.filter(isArchivedConfig).length,
			versions: configRecords.reduce((sum, cfg) => sum + cfg.versionCount, 0),
		}),
		[configRecords],
	);

	const emptyConfigText =
		configShelf === "archived" ? "暂无归档配置" : "暂无工作区配置";

	return (
		<div>
			{msgCtx}
			<Title level={4} style={{ marginTop: 0, marginBottom: 8 }}>
				注册中心
			</Title>
			<Alert
				type="info"
				showIcon
				style={{ marginBottom: 16 }}
				message="注册中心现在分成两层：用户配置库 + 只读平台字典"
				description="配置中心是用户自己注册和维护的独立文件库，支持多版本管理；Algo / Tag / Metric / Lifecycle 继续作为平台字典，只提供口径和校验。"
			/>

			{error ? (
				<ContentErrorState
					title="注册表加载失败"
					description={error}
					onRetry={() => window.location.reload()}
				/>
			) : loading ? (
				<ContentLoadingState title="正在加载注册表…" />
			) : (
				<Space direction="vertical" size={16} style={{ width: "100%" }}>
					<Card
						size="small"
						title={
							<Space size={8}>
								<ApartmentOutlined />
								<span>治理总览</span>
							</Space>
						}
						extra={
							<Button
								icon={<ReloadOutlined />}
								onClick={() => window.location.reload()}
							>
								刷新
							</Button>
						}
					>
						<Row gutter={[12, 12]}>
							<Col xs={12} md={6}>
								<Card
									size="small"
									variant="outlined"
									style={{ background: "#f8fafc" }}
								>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										只读注册表
									</div>
									<div style={{ fontSize: 20, fontWeight: 700 }}>
										{algos.length +
											tags.length +
											metrics.length +
											states.length}
									</div>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										平台字典，不做编辑
									</div>
								</Card>
							</Col>
							<Col xs={12} md={6}>
								<Card
									size="small"
									variant="outlined"
									style={{ background: "#f8fafc" }}
								>
									<div style={{ fontSize: 12, color: "#64748b" }}>配置总数</div>
									<div style={{ fontSize: 20, fontWeight: 700 }}>
										{configStats.total}
									</div>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										工作区 {configStats.active} / 归档 {configStats.archived}
									</div>
								</Card>
							</Col>
							<Col xs={12} md={6}>
								<Card
									size="small"
									variant="outlined"
									style={{ background: "#f8fafc" }}
								>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										可部署配置
									</div>
									<div style={{ fontSize: 20, fontWeight: 700 }}>
										{configStats.ready}
									</div>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										ready 状态可被选择
									</div>
								</Card>
							</Col>
							<Col xs={12} md={6}>
								<Card
									size="small"
									variant="outlined"
									style={{ background: "#f8fafc" }}
								>
									<div style={{ fontSize: 12, color: "#64748b" }}>版本总数</div>
									<div style={{ fontSize: 20, fontWeight: 700 }}>
										{configStats.versions}
									</div>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										配置可回滚和审计
									</div>
								</Card>
							</Col>
						</Row>
					</Card>

					<Card size="small">
						<Tabs
							items={[
								{
									key: "configs",
									label: `配置库 (${configStats.active})`,
									children: (
										<Space
											direction="vertical"
											size={12}
											style={{ width: "100%", minWidth: 0 }}
										>
											<Alert
												type="success"
												showIcon
												message="用户配置文件库"
												description="这里注册的是用户自己维护的文件，不是组件子对象。工作区只显示可维护配置；归档区保留历史追溯，不参与 deploy 选择。"
											/>
											<Row gutter={[12, 12]} align="middle">
												<Col xs={24} lg={12} style={{ minWidth: 0 }}>
													<Space wrap>
														<Tag color="green">可用 {configStats.ready}</Tag>
														<Tag color="gold">草稿 {configStats.draft}</Tag>
														<Tag>已归档 {configStats.archived}</Tag>
														<Tag color="geekblue">
															版本 {configStats.versions}
														</Tag>
														<Tag color="blue">部署可见</Tag>
													</Space>
												</Col>
												<Col xs={24} lg={12} style={{ minWidth: 0 }}>
													<Space
														wrap
														style={{
															width: "100%",
															justifyContent: isNarrow
																? "flex-start"
																: "flex-end",
														}}
													>
														<Segmented
															value={configShelf}
															options={[
																{
																	label: `工作区 ${configStats.active}`,
																	value: "active",
																},
																{
																	label: (
																		<Space size={4}>
																			<InboxOutlined />
																			<span>归档区 {configStats.archived}</span>
																		</Space>
																	),
																	value: "archived",
																},
															]}
															onChange={(value) =>
																setConfigShelf(value as ConfigShelf)
															}
														/>
														<Input.Search
															allowClear
															placeholder={
																configShelf === "archived"
																	? "搜索归档配置"
																	: "搜索名称 / owner / tag / 版本号 / 说明"
															}
															value={configQuery}
															onChange={(event) =>
																setConfigQuery(event.target.value)
															}
															style={{
																width: isNarrow ? "100%" : 260,
																maxWidth: "100%",
															}}
														/>
														<Button
															type="primary"
															icon={<PlusOutlined />}
															onClick={openCreateConfig}
														>
															新建配置
														</Button>
													</Space>
												</Col>
											</Row>
											<TableScrollBoundary>
												<Table
													rowKey="id"
													pagination={false}
													size="small"
													loading={configsLoading}
													columns={configCols}
													dataSource={filteredConfigs}
													scroll={{ x: 960 }}
													onRow={(record) => ({
														onClick: () => setSelectedConfigId(record.id),
														style: { cursor: "pointer" },
													})}
													locale={{ emptyText: emptyConfigText }}
												/>
											</TableScrollBoundary>
											<Paragraph type="secondary" style={{ marginBottom: 0 }}>
												建议的运行语义是：配置先由用户注册为独立记录，版本变更独立留痕；deploy
												时只显示当前用户自己可用的 ready
												版本，归档配置只用于查看、内容追溯和版本对比。
											</Paragraph>
										</Space>
									),
								},
								{
									key: "pools",
									label: `资源池`,
									children: <PoolManager />,
								},
								{
									key: "registry",
									label: "平台字典",
									children: (
										<Space
											direction="vertical"
											size={16}
											style={{ width: "100%" }}
										>
											<Alert
												type="info"
												showIcon
												message="这部分继续只读"
												description="Algo / Tag / Metric / Lifecycle 仍是平台字典来源，用来约束校验、搜索和 UI 口径，不承担配置编辑职责。"
											/>

											<Card size="small" title={`Lifecycle (${states.length})`}>
												<Space wrap>
													{states.map((s) => (
														<Tag key={s} color="blue">
															{s}
														</Tag>
													))}
												</Space>
											</Card>

											<Card size="small" title={`Metrics (${metrics.length})`}>
												<TableScrollBoundary>
													<Table
														rowKey="key"
														pagination={false}
														size="small"
														columns={metricCols}
														dataSource={metrics}
														scroll={{ x: 680 }}
													/>
												</TableScrollBoundary>
											</Card>

											<Card size="small" title={`Tags (${tags.length})`}>
												<TableScrollBoundary>
													<Table
														rowKey="key"
														pagination={false}
														size="small"
														columns={tagCols}
														dataSource={tags}
														scroll={{ x: 680 }}
													/>
												</TableScrollBoundary>
											</Card>

											<Card size="small" title={`Algos (${algos.length})`}>
												<TableScrollBoundary>
													<Table
														rowKey="key"
														pagination={false}
														size="small"
														columns={algoCols}
														dataSource={algos}
														scroll={{ x: 760 }}
													/>
												</TableScrollBoundary>
											</Card>
										</Space>
									),
								},
							]}
						/>
					</Card>
				</Space>
			)}
			<Drawer
				title={
					selectedConfig ? (
						<Space size={4} style={{ lineHeight: 1.4 }}>
							<Text strong style={{ fontSize: 16 }}>
								{selectedConfig.name}
							</Text>
							<Tag>{selectedConfig.fileType}</Tag>
							<Tag color={lifecycleTagColor(selectedConfig.lifecycle)}>
								{lifecycleDisplay(selectedConfig.lifecycle)}
							</Tag>
							<Text type="secondary" style={{ fontSize: 13 }}>
								当前 {selectedConfig.latestVersion} ·{" "}
								{selectedConfig.versionCount} 个版本
							</Text>
						</Space>
					) : (
						"配置详情"
					)
				}
				width={720}
				open={Boolean(selectedConfig)}
				onClose={() => setSelectedConfigId(null)}
				maskStyle={{ backgroundColor: "rgba(0,0,0,0.08)" }}
				extra={
					selectedConfig ? (
						<Space>
							<Button
								icon={<DiffOutlined />}
								disabled={selectedConfig.versions.length < 2}
								onClick={() => handleOpenVersionCompare(selectedConfig)}
							>
								对比版本
							</Button>
							{isArchivedConfig(selectedConfig) ? null : (
								<Button
									type="primary"
									icon={<FileAddOutlined />}
									onClick={() => openCreateVersion(selectedConfig)}
								>
									基于当前版本新建
								</Button>
							)}
						</Space>
					) : null
				}
			>
				{selectedConfig ? (
					<Tabs
						items={[
							{
								key: "info",
								label: "属性",
								children: (
									<Space
										direction="vertical"
										size={12}
										style={{ width: "100%" }}
									>
										<Card
											size="small"
											variant="outlined"
											style={{ background: "#fafafa" }}
										>
											<Space
												direction="vertical"
												size={8}
												style={{ width: "100%" }}
											>
												<Row gutter={[16, 8]}>
													<Col span={12}>
														<Text type="secondary" style={{ fontSize: 12 }}>
															状态
														</Text>
														<div>
															<Tag
																color={lifecycleTagColor(
																	selectedConfig.lifecycle,
																)}
															>
																{lifecycleDisplay(selectedConfig.lifecycle)}
															</Tag>
														</div>
													</Col>
													<Col span={12}>
														<Text type="secondary" style={{ fontSize: 12 }}>
															当前版本
														</Text>
														<div style={{ fontWeight: 600 }}>
															{selectedConfig.latestVersion}
														</div>
													</Col>
													<Col span={12}>
														<Text type="secondary" style={{ fontSize: 12 }}>
															Owner
														</Text>
														<div>
															<UserTag value={selectedConfig.owner} />
														</div>
													</Col>
													<Col span={12}>
														<Text type="secondary" style={{ fontSize: 12 }}>
															最近更新
														</Text>
														<div style={{ fontSize: 13 }}>
															{selectedConfig.updatedAt}
														</div>
													</Col>
												</Row>
												{selectedConfig.lifecycle === "draft" ? (
													<Alert
														type="warning"
														showIcon
														message="当前版本是 Draft，不能用于部署"
														style={{ marginBottom: 0 }}
													/>
												) : selectedConfig.lifecycle === "deprecated" ? (
													<Alert
														type="error"
														showIcon
														message="该配置已归档，不参与部署选择"
														style={{ marginBottom: 0 }}
													/>
												) : (
													<Alert
														type="success"
														showIcon
														message={`当前部署可用版本：${selectedConfig.latestVersion} ${lifecycleDisplay(selectedConfig.lifecycle)}`}
														style={{ marginBottom: 0 }}
													/>
												)}
											</Space>
										</Card>
										<Collapse
											ghost
											items={[
												{
													key: "more",
													label: "更多信息",
													children: (
														<Descriptions column={1} size="small">
															<Descriptions.Item label="配置 ID">
																<Space size={4}>
																	<Text code style={{ fontSize: 12 }}>
																		{selectedConfig.id}
																	</Text>
																	<Button
																		type="link"
																		size="small"
																		icon={<CopyOutlined />}
																		onClick={() => {
																			navigator.clipboard.writeText(
																				selectedConfig.id,
																			);
																			msg.success("已复制");
																		}}
																	/>
																</Space>
															</Descriptions.Item>
															<Descriptions.Item label="文件类型">
																<Tag>{selectedConfig.fileType}</Tag>
															</Descriptions.Item>
															<Descriptions.Item label="描述">
																{selectedConfig.description}
															</Descriptions.Item>
															<Descriptions.Item label="标签">
																<ConfigTagList tags={selectedConfig.tags} />
															</Descriptions.Item>
														</Descriptions>
													),
												},
											]}
										/>
									</Space>
								),
							},
							{
								key: "content",
								label: "内容",
								children: <ContentPreviewTab config={selectedConfig} />,
							},
							{
								key: "history",
								label: "版本历史",
								children: <VersionHistoryTab config={selectedConfig} />,
							},
							{
								key: "references",
								label: "引用关系",
								children: (
									<Card size="small">
										<Empty
											description="暂未接入引用数据"
											image={Empty.PRESENTED_IMAGE_SIMPLE}
										/>
									</Card>
								),
							},
						]}
					/>
				) : null}
			</Drawer>

			<Modal
				title={editingConfig ? "编辑配置属性" : "新建配置"}
				open={configModalOpen}
				onCancel={() => {
					setConfigModalOpen(false);
					setEditingConfig(null);
					configForm.resetFields();
				}}
				onOk={() => configForm.submit()}
				okText={editingConfig ? "保存属性" : "创建"}
				confirmLoading={savingConfig}
				cancelText="取消"
				style={{ top: 20 }}
				bodyStyle={{
					maxHeight: "calc(80vh - 120px)",
					overflowY: "auto",
					paddingTop: 12,
				}}
				destroyOnHidden
			>
				<Form
					form={configForm}
					layout="vertical"
					requiredMark={false}
					onFinish={handleSaveConfig}
				>
					<Form.Item
						name="name"
						label="配置名称"
						rules={[
							{ required: true, message: "请输入配置名称" },
							{
								pattern: CONFIG_NAME_PATTERN,
								message:
									"仅小写字母、数字、连字符、点号，以 .yaml/.yml/.json 结尾",
							},
							{
								max: CONFIG_NAME_MAX_LENGTH,
								message: `不能超过 ${CONFIG_NAME_MAX_LENGTH} 字符`,
							},
						]}
						extra={configNameExtra(configNameValue)}
					>
						<Input placeholder="example.yaml" />
					</Form.Item>
					<Form.Item
						name="description"
						label="描述"
						rules={[{ required: true, message: "请输入配置描述" }]}
					>
						<Input.TextArea rows={3} placeholder="说明配置用途和适用场景" />
					</Form.Item>
					<Form.Item name="tags" label="标签">
						<Select
							mode="tags"
							placeholder="输入标签后回车"
							options={availableConfigTags.map((tag) => ({
								label: tag,
								value: tag,
							}))}
						/>
					</Form.Item>
					<Form.Item
						name="lifecycle"
						label="状态"
						rules={[{ required: true, message: "请选择状态" }]}
					>
						<Select
							options={[
								{ label: "draft", value: "draft" },
								{ label: "ready", value: "ready" },
								{ label: "archived", value: "deprecated" },
							]}
						/>
					</Form.Item>
					{!editingConfig ? (
						<Form.Item name="versionSummary" label="初始版本说明">
							<Input placeholder="首次注册" />
						</Form.Item>
					) : null}
					{!editingConfig ? (
						<Form.Item
							name="content"
							label="文件内容"
							rules={[{ required: true, message: "请输入配置文件内容" }]}
						>
							<Input.TextArea
								rows={10}
								placeholder={"key: value\n# yaml 或 json 文件内容"}
								style={{ fontFamily: "monospace" }}
								spellCheck={false}
							/>
						</Form.Item>
					) : null}
				</Form>
			</Modal>

			<Modal
				title={
					versionTarget ? `创建新版本：${versionTarget.name}` : "创建新版本"
				}
				open={Boolean(versionTarget)}
				width={1200}
				onCancel={() => {
					setVersionTarget(null);
					setVersionSource(null);
					versionForm.resetFields();
					setDiffPreviewRows([]);
					setDiffBaseContent("");
				}}
				onOk={() => versionForm.submit()}
				okText="创建版本"
				confirmLoading={savingVersion}
				cancelText="取消"
				style={{ top: 20 }}
				bodyStyle={{ padding: "20px 24px" }}
				destroyOnHidden
				footer={(_, { OkBtn, CancelBtn }) => (
					<div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
						<CancelBtn />
						<OkBtn />
					</div>
				)}
			>
				{versionTarget && versionSource ? (
					<Form form={versionForm} onFinish={handleCreateVersion}>
						<div>
							<div
								style={{
									display: "flex",
									alignItems: "center",
									gap: 8,
									marginBottom: 16,
									padding: "6px 12px",
									background: "#f6f8fa",
									borderRadius: 6,
									border: "1px solid #d0d7de",
								}}
							>
								<Tag color="blue" style={{ margin: 0 }}>
									Base {versionSource.version}
								</Tag>
								<Text type="secondary">→</Text>
								<Tag color="green" style={{ margin: 0 }}>
									Next v{(versionTarget.currentVersion || 0) + 1}
								</Tag>
								<Text type="secondary" style={{ fontSize: 12, flex: 1 }}>
									{versionSource.summary} · {versionSource.updatedAt}
								</Text>
								<Tag
									color={diffPreviewRows.length > 0 ? "blue" : "default"}
									style={{ margin: 0, fontSize: 11 }}
								>
									+{diffPreviewRows.filter((r) => r.kind === "added").length} −
									{diffPreviewRows.filter((r) => r.kind === "removed").length}
								</Tag>
								<Form.Item name="lifecycle" noStyle>
									<Select
										size="small"
										style={{ width: 100 }}
										options={[
											{ label: "Draft", value: "draft" },
											{ label: "Ready", value: "ready" },
										]}
									/>
								</Form.Item>
							</div>

							<div style={{ marginBottom: 16 }}>
								<Text
									strong
									style={{ fontSize: 13, display: "block", marginBottom: 6 }}
								>
									变更说明
								</Text>
								<Form.Item
									name="summary"
									rules={[{ required: true, message: "请输入变更说明" }]}
									style={{ marginBottom: 0 }}
								>
									<Input.TextArea
										rows={2}
										placeholder={`基于 ${versionSource.version} 做了什么修改`}
									/>
								</Form.Item>
							</div>
							<Row gutter={16}>
								<Col span={16}>
									<div
										style={{ marginBottom: 8, fontWeight: 600, fontSize: 13 }}
									>
										YAML 编辑器
									</div>
									<Form.Item
										name="content"
										rules={[{ required: true, message: "请输入配置内容" }]}
										style={{ marginBottom: 0 }}
									>
										<Editor
											height="420px"
											defaultLanguage="yaml"
											theme="vs"
											options={{
												minimap: { enabled: false },
												lineNumbers: "on",
												tabSize: 2,
												renderWhitespace: "selection",
												scrollBeyondLastLine: false,
												fontSize: 12,
												fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
											}}
											beforeMount={(monaco: Monaco) => {
												monaco.languages.json.jsonDefaults.setDiagnosticsOptions(
													{
														validate: true,
														allowComments: true,
													},
												);
											}}
										/>
									</Form.Item>
								</Col>
								<Col span={8}>
									<div
										style={{
											marginBottom: 8,
											display: "flex",
											alignItems: "center",
											gap: 8,
										}}
									>
										<Text strong style={{ fontSize: 13 }}>
											变更预览
										</Text>
										<Select
											size="small"
											style={{ width: 90 }}
											value={
												diffBaseVersionNumber ?? versionSource.versionNumber
											}
											onChange={async (vn: number) => {
												setDiffBaseVersionNumber(vn);
												await loadDiffBaseVersion(versionTarget.id, vn);
											}}
											options={versionTarget.versions.map((v) => ({
												label: v.version,
												value: v.versionNumber,
											}))}
										/>
									</div>
									{diffPreviewRows.length > 0 ? (
										<div
											style={{
												border: "1px solid #e2e8f0",
												borderRadius: 6,
												overflow: "hidden",
											}}
										>
											<div
												style={{
													display: "grid",
													gridTemplateColumns:
														"40px minmax(0, 1fr) 40px minmax(0, 1fr)",
													background: "#f8fafc",
													borderBottom: "1px solid #e2e8f0",
													fontWeight: 600,
													fontSize: 11,
												}}
											>
												<div style={{ padding: "4px 6px" }}></div>
												<div style={{ padding: "4px 6px" }}>
													{versionSource.version}
												</div>
												<div style={{ padding: "4px 6px" }}></div>
												<div style={{ padding: "4px 6px" }}>当前</div>
											</div>
											<div style={{ maxHeight: 378, overflow: "auto" }}>
												{diffPreviewRows.map((row) => (
													<div
														key={row.key}
														style={{
															display: "grid",
															gridTemplateColumns:
																"40px minmax(0, 1fr) 40px minmax(0, 1fr)",
															borderBottom: "1px solid #f0f2f5",
															fontSize: 11,
														}}
													>
														<div
															style={{
																padding: "1px 4px",
																color: "#94a3b8",
																background: "#f8fafc",
																textAlign: "right",
																fontFamily: "monospace",
															}}
														>
															{row.leftLine ?? ""}
														</div>
														<pre
															style={{
																margin: 0,
																padding: "1px 4px",
																whiteSpace: "pre-wrap",
																wordBreak: "break-word",
																fontFamily: "monospace",
																background: diffCellBackground(
																	row.kind,
																	"left",
																),
															}}
														>
															{row.leftText ?? ""}
														</pre>
														<div
															style={{
																padding: "1px 4px",
																color: "#94a3b8",
																background: "#f8fafc",
																textAlign: "right",
																fontFamily: "monospace",
															}}
														>
															{row.rightLine ?? ""}
														</div>
														<pre
															style={{
																margin: 0,
																padding: "1px 4px",
																whiteSpace: "pre-wrap",
																wordBreak: "break-word",
																fontFamily: "monospace",
																background: diffCellBackground(
																	row.kind,
																	"right",
																),
															}}
														>
															{row.rightText ?? ""}
														</pre>
													</div>
												))}
											</div>
										</div>
									) : (
										<div
											style={{
												height: 420,
												display: "flex",
												alignItems: "center",
												justifyContent: "center",
												border: "1px dashed #d9d9d9",
												borderRadius: 6,
											}}
										>
											<Text type="secondary">
												编辑左侧 YAML 后，此处实时显示差异
											</Text>
										</div>
									)}
								</Col>
							</Row>

							{/* DESCRIPTION_MOVED */}
						</div>
					</Form>
				) : (
					<Alert
						type="info"
						showIcon
						message="请先在配置列表中选择一个配置，再点击「新建版本」"
					/>
				)}
			</Modal>

			<Modal
				title={`编辑版本 v${editVersionState.versionNumber}`}
				open={editVersionState.visible}
				width={700}
				okText="保存"
				cancelText="取消"
				confirmLoading={editVersionState.saving}
				onCancel={() =>
					setEditVersionState((prev) => ({
						...prev,
						visible: false,
						saving: false,
					}))
				}
				onOk={async () => {
					setEditVersionState((prev) => ({ ...prev, saving: true }));
					try {
						await pipelineConfigApi.updateVersionContent(
							editVersionState.configId,
							editVersionState.versionNumber,
							editVersionState.content,
							editVersionState.summary,
						);
						msg.success("版本已更新");
						setEditVersionState((prev) => ({
							...prev,
							visible: false,
							saving: false,
						}));
						refreshConfigs();
					} catch (err) {
						msg.error(`更新失败: ${err}`);
						setEditVersionState((prev) => ({ ...prev, saving: false }));
					}
				}}
			>
				<div key={editVersionState.versionNumber}>
					<div style={{ marginBottom: 6, fontWeight: 500 }}>变更说明</div>
					<Input.TextArea
						rows={2}
						value={editVersionState.summary}
						onChange={(e) =>
							setEditVersionState((prev) => ({
								...prev,
								summary: e.target.value,
							}))
						}
						style={{ marginBottom: 14 }}
					/>
					<div style={{ marginBottom: 6, fontWeight: 500 }}>文件内容</div>
					<Input.TextArea
						rows={12}
						value={editVersionState.content}
						onChange={(e) =>
							setEditVersionState((prev) => ({
								...prev,
								content: e.target.value,
							}))
						}
					/>
				</div>
			</Modal>

			<Modal
				title={
					compareTarget ? `配置版本对比：${compareTarget.name}` : "配置版本对比"
				}
				open={Boolean(compareTarget)}
				onCancel={() => {
					setCompareTarget(null);
					setSelectedVersionCompare(null);
					compareForm.resetFields();
				}}
				footer={[
					<Button
						key="close"
						onClick={() => {
							setCompareTarget(null);
							setSelectedVersionCompare(null);
							compareForm.resetFields();
						}}
					>
						关闭
					</Button>,
				]}
				width={1120}
				destroyOnHidden
			>
				{compareTarget ? (
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						<Form
							form={compareForm}
							layout="inline"
							requiredMark={false}
							onFinish={handleCompareSubmit}
						>
							<Form.Item
								name="leftVersion"
								label="基准版本"
								rules={[{ required: true, message: "请选择基准版本" }]}
							>
								<Select
									style={{ width: 360 }}
									options={compareTarget.versions.map((version) => ({
										value: version.versionNumber,
										label: versionDescriptor(version),
									}))}
								/>
							</Form.Item>
							<Form.Item
								name="rightVersion"
								label="目标版本"
								rules={[{ required: true, message: "请选择目标版本" }]}
							>
								<Select
									style={{ width: 360 }}
									options={compareTarget.versions.map((version) => ({
										value: version.versionNumber,
										label: versionDescriptor(version),
									}))}
								/>
							</Form.Item>
							<Form.Item>
								<Button
									type="primary"
									icon={<DiffOutlined />}
									loading={compareLoading}
									onClick={() => compareForm.submit()}
								>
									对比
								</Button>
							</Form.Item>
						</Form>

						{selectedVersionCompare ? (
							<Space direction="vertical" size={8} style={{ width: "100%" }}>
								<Space wrap>
									<Tag color="red">
										基准 {selectedVersionCompare.left.version}
									</Tag>
									<Text type="secondary">
										{selectedVersionCompare.left.summary} ·{" "}
										{selectedVersionCompare.left.updatedAt} ·{" "}
										{shortHash(selectedVersionCompare.left.contentSha256)}
									</Text>
								</Space>
								<Space wrap>
									<Tag color="green">
										目标 {selectedVersionCompare.right.version}
									</Tag>
									<Text type="secondary">
										{selectedVersionCompare.right.summary} ·{" "}
										{selectedVersionCompare.right.updatedAt} ·{" "}
										{shortHash(selectedVersionCompare.right.contentSha256)}
									</Text>
								</Space>
								<Text type="secondary">
									变更行{" "}
									{
										selectedVersionCompare.rows.filter(
											(row) => row.kind !== "equal",
										).length
									}{" "}
									/ 共 {selectedVersionCompare.rows.length} 行
								</Text>
								<div
									style={{
										border: "1px solid #e2e8f0",
										borderRadius: 6,
										overflow: "hidden",
									}}
								>
									<div
										style={{
											display: "grid",
											gridTemplateColumns:
												"64px minmax(0, 1fr) 64px minmax(0, 1fr)",
											background: "#f8fafc",
											borderBottom: "1px solid #e2e8f0",
											fontWeight: 600,
										}}
									>
										<div style={{ padding: "8px 10px" }}>行</div>
										<div style={{ padding: "8px 10px" }}>
											{selectedVersionCompare.left.version}
										</div>
										<div style={{ padding: "8px 10px" }}>行</div>
										<div style={{ padding: "8px 10px" }}>
											{selectedVersionCompare.right.version}
										</div>
									</div>
									<div style={{ maxHeight: 460, overflow: "auto" }}>
										{selectedVersionCompare.rows.map((row) => (
											<div
												key={row.key}
												style={{
													display: "grid",
													gridTemplateColumns:
														"64px minmax(0, 1fr) 64px minmax(0, 1fr)",
													borderBottom: "1px solid #eef2f7",
												}}
											>
												<div
													style={{
														padding: "4px 8px",
														color: "#64748b",
														background: "#f8fafc",
														textAlign: "right",
														fontFamily: "monospace",
													}}
												>
													{row.leftLine ?? ""}
												</div>
												<pre
													style={{
														margin: 0,
														padding: "4px 8px",
														whiteSpace: "pre-wrap",
														wordBreak: "break-word",
														fontFamily: "monospace",
														background: diffCellBackground(row.kind, "left"),
													}}
												>
													{row.leftText ?? ""}
												</pre>
												<div
													style={{
														padding: "4px 8px",
														color: "#64748b",
														background: "#f8fafc",
														textAlign: "right",
														fontFamily: "monospace",
													}}
												>
													{row.rightLine ?? ""}
												</div>
												<pre
													style={{
														margin: 0,
														padding: "4px 8px",
														whiteSpace: "pre-wrap",
														wordBreak: "break-word",
														fontFamily: "monospace",
														background: diffCellBackground(row.kind, "right"),
													}}
												>
													{row.rightText ?? ""}
												</pre>
											</div>
										))}
									</div>
								</div>
							</Space>
						) : (
							<Alert type="info" showIcon message="请选择两个版本后查看差异" />
						)}
					</Space>
				) : null}
			</Modal>

			<Modal
				title={
					selectedVersionContent
						? `${selectedVersionContent.configName} @ ${selectedVersionContent.version.version}`
						: "版本文件内容"
				}
				open={Boolean(selectedVersionContent)}
				onCancel={() => setSelectedVersionContent(null)}
				footer={[
					<Button key="close" onClick={() => setSelectedVersionContent(null)}>
						关闭
					</Button>,
				]}
				width={760}
			>
				{selectedVersionContent ? (
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						<Space wrap>
							<Tag color="geekblue">
								{selectedVersionContent.version.version}
							</Tag>
							<Tag
								color={lifecycleTagColor(
									selectedVersionContent.version.lifecycle,
								)}
							>
								{lifecycleDisplay(selectedVersionContent.version.lifecycle)}
							</Tag>
							<Text type="secondary">
								{selectedVersionContent.version.updatedAt} by{" "}
								{selectedVersionContent.version.author}
							</Text>
						</Space>
						<pre
							style={{
								margin: 0,
								padding: 12,
								background: "#0f172a",
								color: "#e2e8f0",
								borderRadius: 6,
								maxHeight: 420,
								overflow: "auto",
								fontSize: 12,
								lineHeight: 1.6,
							}}
						>
							{selectedVersionContent.version.content}
						</pre>
					</Space>
				) : null}
			</Modal>
		</div>
	);
}
