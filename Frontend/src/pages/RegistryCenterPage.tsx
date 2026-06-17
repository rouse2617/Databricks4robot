import {
	ApartmentOutlined,
	BranchesOutlined,
	EditOutlined,
	EyeOutlined,
	FileAddOutlined,
	PlusOutlined,
	ReloadOutlined,
	StopOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Col,
	Descriptions,
	Drawer,
	Form,
	Input,
	Modal,
	message,
	Popconfirm,
	Row,
	Select,
	Space,
	Table,
	Tabs,
	Tag,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
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
	content?: string;
}

interface SelectedVersionContent {
	configName: string;
	version: ConfigVersionRecord;
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

function mapConfigVersion(version: PipelineConfigVersion): ConfigVersionRecord {
	return {
		version: versionLabel(version.version),
		versionNumber: version.version,
		lifecycle: version.status,
		updatedAt: formatConfigTime(version.createdAt),
		author: version.author || "—",
		summary: version.summary || "—",
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

export default function RegistryCenterPage() {
	const [msg, msgCtx] = message.useMessage();
	const [configRecords, setConfigRecords] = useState<UserConfigRecord[]>([]);
	const [configQuery, setConfigQuery] = useState("");
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
	const [configForm] = Form.useForm<ConfigFormValues>();
	const [versionForm] = Form.useForm<VersionFormValues>();
	const [algos, setAlgos] = useState<AlgoRegistryItem[]>([]);
	const [tags, setTags] = useState<TagRegistryItem[]>([]);
	const [metrics, setMetrics] = useState<MetricRegistryItem[]>([]);
	const [states, setStates] = useState<string[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [configsLoading, setConfigsLoading] = useState(false);
	const [savingConfig, setSavingConfig] = useState(false);
	const [savingVersion, setSavingVersion] = useState(false);
	const [deprecatingConfigId, setDeprecatingConfigId] = useState<string | null>(
		null,
	);
	const [loadingVersionKey, setLoadingVersionKey] = useState<string | null>(
		null,
	);

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
		if (!query) return configRecords;
		return configRecords.filter((cfg) =>
			[
				cfg.name,
				cfg.owner,
				cfg.description,
				cfg.fileType,
				cfg.lifecycle,
				cfg.latestVersion,
				...cfg.tags,
			]
				.join(" ")
				.toLowerCase()
				.includes(query),
		);
	}, [configRecords, configQuery]);

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
			} catch {
				msg.error(editingConfig ? "配置更新失败" : "配置创建失败");
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
				msg.success("配置已废弃");
			} catch {
				msg.error("配置废弃失败");
			} finally {
				setDeprecatingConfigId(null);
			}
		})();
	};

	const handleOpenVersionContent = (
		config: UserConfigRecord,
		version: ConfigVersionRecord,
	) => {
		void (async () => {
			const loadingKey = `${config.id}:${version.versionNumber}`;
			setLoadingVersionKey(loadingKey);
			try {
				const detail = await pipelineConfigApi.getVersion(
					config.id,
					version.versionNumber,
				);
				setSelectedVersionContent({
					configName: config.name,
					version: mapConfigVersion(detail),
				});
			} catch {
				msg.error("读取版本内容失败");
			} finally {
				setLoadingVersionKey((current) =>
					current === loadingKey ? null : current,
				);
			}
		})();
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
			render: (_, row) => (
				<div>
					<Space size={6}>
						<div style={{ fontWeight: 600 }}>{row.name}</div>
						<Tag>{row.fileType}</Tag>
					</Space>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{row.description}
					</Text>
				</div>
			),
		},
		{
			title: "拥有者",
			dataIndex: "owner",
			width: 110,
			render: (v) => <Tag color="blue">{v}</Tag>,
		},
		{
			title: "状态",
			dataIndex: "lifecycle",
			width: 120,
			render: (v: UserConfigRecord["lifecycle"]) => {
				const color =
					v === "ready" ? "green" : v === "draft" ? "gold" : "default";
				return <Tag color={color}>{v}</Tag>;
			},
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
			render: (v: string[]) => (
				<Space wrap size={4}>
					{v.map((tag) => (
						<Tag key={tag}>{tag}</Tag>
					))}
				</Space>
			),
		},
		{
			title: "操作",
			key: "actions",
			width: 260,
			render: (_, record) => (
				<Space size={4} wrap>
					<Button
						size="small"
						icon={<EyeOutlined />}
						onClick={() => setSelectedConfigId(record.id)}
					>
						详情
					</Button>
					<Button
						size="small"
						icon={<EditOutlined />}
						onClick={() => openEditConfig(record)}
					>
						编辑
					</Button>
					<Button
						size="small"
						icon={<FileAddOutlined />}
						onClick={() => openCreateVersion(record)}
					>
						新版本
					</Button>
					<Popconfirm
						title="废弃配置"
						description="废弃后不会物理删除，历史引用仍可追溯。"
						okText="废弃"
						cancelText="取消"
						onConfirm={() => handleDeprecateConfig(record)}
					>
						<Button
							size="small"
							danger
							icon={<StopOutlined />}
							loading={deprecatingConfigId === record.id}
							disabled={record.lifecycle === "deprecated"}
						>
							废弃
						</Button>
					</Popconfirm>
				</Space>
			),
		},
	];

	const createVersionCols = (
		config: UserConfigRecord,
	): ColumnsType<ConfigVersionRecord> => [
		{
			title: "版本",
			dataIndex: "version",
			width: 100,
			render: (v) => <Text code>{v}</Text>,
		},
		{
			title: "状态",
			dataIndex: "lifecycle",
			width: 120,
			render: (v: ConfigVersionRecord["lifecycle"]) => {
				const color =
					v === "ready" ? "green" : v === "draft" ? "gold" : "default";
				return <Tag color={color}>{v}</Tag>;
			},
		},
		{ title: "更新人", dataIndex: "author", width: 120 },
		{ title: "更新时间", dataIndex: "updatedAt", width: 180 },
		{ title: "变更说明", dataIndex: "summary" },
		{
			title: "文件",
			key: "content",
			width: 210,
			render: (_, version) => (
				<Space size={4}>
					<Button
						size="small"
						icon={<EyeOutlined />}
						loading={
							loadingVersionKey === `${config.id}:${version.versionNumber}`
						}
						onClick={() => handleOpenVersionContent(config, version)}
					>
						内容
					</Button>
					<Button
						size="small"
						icon={<EditOutlined />}
						onClick={() => openCreateVersion(config, version)}
					>
						编辑为新版本
					</Button>
				</Space>
			),
		},
	];

	const configStats = useMemo(
		() => ({
			total: configRecords.length,
			ready: configRecords.filter((cfg) => cfg.lifecycle === "ready").length,
			draft: configRecords.filter((cfg) => cfg.lifecycle === "draft").length,
			deprecated: configRecords.filter((cfg) => cfg.lifecycle === "deprecated")
				.length,
			versions: configRecords.reduce((sum, cfg) => sum + cfg.versionCount, 0),
		}),
		[configRecords],
	);

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
								<Card size="small" bordered style={{ background: "#f8fafc" }}>
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
								<Card size="small" bordered style={{ background: "#f8fafc" }}>
									<div style={{ fontSize: 12, color: "#64748b" }}>配置总数</div>
									<div style={{ fontSize: 20, fontWeight: 700 }}>
										{configStats.total}
									</div>
									<div style={{ fontSize: 12, color: "#64748b" }}>
										用户注册的文件记录
									</div>
								</Card>
							</Col>
							<Col xs={12} md={6}>
								<Card size="small" bordered style={{ background: "#f8fafc" }}>
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
								<Card size="small" bordered style={{ background: "#f8fafc" }}>
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
									label: `配置库 (${configStats.total})`,
									children: (
										<Space
											direction="vertical"
											size={12}
											style={{ width: "100%" }}
										>
											<Alert
												type="success"
												showIcon
												message="用户配置文件库"
												description="这里注册的是用户自己维护的文件，不是组件子对象。每个配置可以有多个版本，deploy 时只选择当前用户可用的 ready 版本。"
											/>
											<Row gutter={[12, 12]} align="middle">
												<Col flex="auto">
													<Space wrap>
														<Tag color="green">Ready {configStats.ready}</Tag>
														<Tag color="gold">Draft {configStats.draft}</Tag>
														<Tag>Deprecated {configStats.deprecated}</Tag>
														<Tag color="geekblue">
															Versions {configStats.versions}
														</Tag>
														<Tag color="blue">Own / Shared deploy filter</Tag>
													</Space>
												</Col>
												<Col>
													<Space>
														<Input.Search
															allowClear
															placeholder="搜索名称 / owner / tag"
															value={configQuery}
															onChange={(event) =>
																setConfigQuery(event.target.value)
															}
															style={{ width: 260 }}
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
											<Table
												rowKey="id"
												pagination={false}
												size="small"
												loading={configsLoading}
												columns={configCols}
												dataSource={filteredConfigs}
												expandable={{
													expandedRowRender: (record) => (
														<Table
															rowKey="version"
															pagination={false}
															size="small"
															columns={createVersionCols(record)}
															dataSource={record.versions}
														/>
													),
												}}
											/>
											<Paragraph type="secondary" style={{ marginBottom: 0 }}>
												建议的运行语义是：配置先由用户注册为独立记录，版本变更独立留痕；deploy
												时只显示当前用户自己可用的 ready 版本。
											</Paragraph>
										</Space>
									),
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
												<Table
													rowKey="key"
													pagination={false}
													size="small"
													columns={metricCols}
													dataSource={metrics}
												/>
											</Card>

											<Card size="small" title={`Tags (${tags.length})`}>
												<Table
													rowKey="key"
													pagination={false}
													size="small"
													columns={tagCols}
													dataSource={tags}
												/>
											</Card>

											<Card size="small" title={`Algos (${algos.length})`}>
												<Table
													rowKey="key"
													pagination={false}
													size="small"
													columns={algoCols}
													dataSource={algos}
												/>
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
				title={selectedConfig ? selectedConfig.name : "配置详情"}
				width={720}
				open={Boolean(selectedConfig)}
				onClose={() => setSelectedConfigId(null)}
				extra={
					selectedConfig ? (
						<Space>
							<Button
								icon={<EditOutlined />}
								onClick={() => openEditConfig(selectedConfig)}
							>
								编辑
							</Button>
							<Button
								type="primary"
								icon={<FileAddOutlined />}
								onClick={() => openCreateVersion(selectedConfig)}
							>
								新增版本
							</Button>
						</Space>
					) : null
				}
			>
				{selectedConfig ? (
					<Space direction="vertical" size={16} style={{ width: "100%" }}>
						<Descriptions column={2} size="small" bordered>
							<Descriptions.Item label="配置 ID" span={2}>
								<Text code>{selectedConfig.id}</Text>
							</Descriptions.Item>
							<Descriptions.Item label="文件名">
								<Text code>{selectedConfig.name}</Text>
							</Descriptions.Item>
							<Descriptions.Item label="文件类型">
								<Tag>{selectedConfig.fileType}</Tag>
							</Descriptions.Item>
							<Descriptions.Item label="拥有者">
								<Tag color="blue">{selectedConfig.owner}</Tag>
							</Descriptions.Item>
							<Descriptions.Item label="状态">
								<Tag
									color={
										selectedConfig.lifecycle === "ready"
											? "green"
											: selectedConfig.lifecycle === "draft"
												? "gold"
												: "default"
									}
								>
									{selectedConfig.lifecycle}
								</Tag>
							</Descriptions.Item>
							<Descriptions.Item label="当前版本">
								<Tag color="geekblue">{selectedConfig.latestVersion}</Tag>
							</Descriptions.Item>
							<Descriptions.Item label="更新时间">
								{selectedConfig.updatedAt}
							</Descriptions.Item>
							<Descriptions.Item label="描述" span={2}>
								{selectedConfig.description}
							</Descriptions.Item>
							<Descriptions.Item label="标签" span={2}>
								<Space wrap size={4}>
									{selectedConfig.tags.map((tag) => (
										<Tag key={tag}>{tag}</Tag>
									))}
								</Space>
							</Descriptions.Item>
						</Descriptions>
						<Card size="small" title="版本历史">
							<Table
								rowKey="version"
								pagination={false}
								size="small"
								columns={createVersionCols(selectedConfig)}
								dataSource={selectedConfig.versions}
							/>
						</Card>
					</Space>
				) : null}
			</Drawer>

			<Modal
				title={editingConfig ? "编辑配置" : "新建配置"}
				open={configModalOpen}
				onCancel={() => {
					setConfigModalOpen(false);
					setEditingConfig(null);
					configForm.resetFields();
				}}
				onOk={() => configForm.submit()}
				okText={editingConfig ? "保存" : "创建"}
				confirmLoading={savingConfig}
				cancelText="取消"
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
						rules={[{ required: true, message: "请输入配置名称" }]}
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
								{ label: "deprecated", value: "deprecated" },
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
							/>
						</Form.Item>
					) : null}
				</Form>
			</Modal>

			<Modal
				title={
					versionTarget
						? versionSource
							? `基于 ${versionSource.version} 编辑为新版本：${versionTarget.name}`
							: `新增版本：${versionTarget.name}`
						: "新增版本"
				}
				open={Boolean(versionTarget)}
				onCancel={() => {
					setVersionTarget(null);
					setVersionSource(null);
					versionForm.resetFields();
				}}
				onOk={() => versionForm.submit()}
				okText="创建版本"
				confirmLoading={savingVersion}
				cancelText="取消"
				destroyOnHidden
			>
				<Form
					form={versionForm}
					layout="vertical"
					requiredMark={false}
					onFinish={handleCreateVersion}
				>
					<Form.Item
						name="summary"
						label="变更说明"
						rules={[{ required: true, message: "请输入变更说明" }]}
					>
						<Input.TextArea rows={3} placeholder="说明这次版本改了什么" />
					</Form.Item>
					<Form.Item
						name="lifecycle"
						label="版本状态"
						rules={[{ required: true, message: "请选择版本状态" }]}
					>
						<Select
							options={[
								{ label: "draft", value: "draft" },
								{ label: "ready", value: "ready" },
							]}
						/>
					</Form.Item>
					<Form.Item
						name="content"
						label="文件内容"
						rules={[{ required: true, message: "请输入配置文件内容" }]}
					>
						<Input.TextArea
							rows={12}
							placeholder="复制或编辑完整配置文件内容"
							style={{ fontFamily: "monospace" }}
						/>
					</Form.Item>
				</Form>
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
								color={
									selectedVersionContent.version.lifecycle === "ready"
										? "green"
										: selectedVersionContent.version.lifecycle === "draft"
											? "gold"
											: "default"
								}
							>
								{selectedVersionContent.version.lifecycle}
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
