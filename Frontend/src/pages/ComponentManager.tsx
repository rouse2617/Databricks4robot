import {
	DeleteOutlined,
	EditOutlined,
	EyeOutlined,
	PlusOutlined,
	ReloadOutlined,
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
	listComponents,
	type PipelineComponentAPI,
	type PipelineComponentPayload,
	type PipelineComponentType,
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

type EnvRow = { name?: string; value?: string };

type ModalMode = "create" | "edit" | "view";

interface ComponentFormValues {
	name: string;
	type: PipelineComponentType;
	image: string;
	tag?: string;
	description?: string;
	command?: string;
	args?: string;
	envRows?: EnvRow[];
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

const formatDateTime = (value?: string): string =>
	value ? new Date(value).toLocaleString() : "-";

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
			envRows: [],
		};
	}

	return {
		name: component.name,
		type: component.type || "container",
		image: component.image,
		tag: component.tag || "latest",
		description: component.description || "",
		command: joinInputItems(component.command || []),
		args: joinInputItems(component.args || []),
		envRows: Object.entries(component.env || {}).map(([name, value]) => ({
			name,
			value,
		})),
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
		inputPorts: [{ name: "input", type: "asset" }],
		outputPorts: [{ name: "output", type: "asset" }],
		resources: {
			type: values.type,
			command,
			args,
			env,
		},
	};
}

function ComponentDetail({ component }: { component: PipelineComponentAPI }) {
	const envEntries = Object.entries(component.env || {});
	const imageText = formatComponentImage(component.image, component.tag);

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
			</Descriptions>
		</Space>
	);
}

export function ComponentManager() {
	const [items, setItems] = useState<PipelineComponentAPI[]>([]);
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [search, setSearch] = useState("");
	const [modalOpen, setModalOpen] = useState(false);
	const [modalMode, setModalMode] = useState<ModalMode | null>(null);
	const [activeComponent, setActiveComponent] =
		useState<PipelineComponentAPI | null>(null);
	const [selectedComponentIds, setSelectedComponentIds] = useState<string[]>(
		[],
	);
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

	useEffect(() => {
		refresh();
	}, [refresh]);

	const filteredItems = useMemo(() => {
		const q = search.trim().toLowerCase();
		if (!q) return items;
		return items.filter((item) =>
			[item.name, item.image, item.description, item.type]
				.filter(Boolean)
				.some((value) => value.toLowerCase().includes(q)),
		);
	}, [items, search]);

	const isCreateMode = modalMode === "create";
	const isEditMode = modalMode === "edit";
	const isViewMode = modalMode === "view";

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
		form.setFieldsValue(toFormValues());
		setModalOpen(true);
	};

	const openEdit = (component: PipelineComponentAPI) => {
		setActiveComponent(component);
		setModalMode("edit");
		form.setFieldsValue(toFormValues(component));
		setModalOpen(true);
	};

	const openView = (component: PipelineComponentAPI) => {
		setActiveComponent(component);
		setModalMode("view");
		form.setFieldsValue(toFormValues(component));
		setModalOpen(true);
	};

	const closeModal = () => {
		setModalOpen(false);
		setModalMode(null);
		setActiveComponent(null);
	};

	const handleSave = async () => {
		if (isViewMode) {
			closeModal();
			return;
		}
		const values = await form.validateFields();
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
			await refresh();
		} finally {
			setBulkDeleting(false);
		}
	};

	const columns: ColumnsType<PipelineComponentAPI> = [
		{
			title: "名称",
			dataIndex: "name",
			key: "name",
			width: 220,
			render: (name: string, record) => (
				<Space direction="vertical" size={0}>
					<Typography.Text strong ellipsis={{ tooltip: name }}>
						{name}
					</Typography.Text>
					<Typography.Text
						type="secondary"
						copyable={{ text: record.id }}
						style={{ fontSize: 12 }}
					>
						ID: {toAssetStyleId(record.id)}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: "类型",
			dataIndex: "type",
			key: "type",
			width: 92,
			render: (type: PipelineComponentType) => (
				<Tag color={TYPE_COLORS[type] || "default"}>{type || "container"}</Tag>
			),
		},
		{
			title: "镜像",
			key: "image",
			width: 180,
			render: (_, record) => {
				const imageText = formatComponentImage(record.image, record.tag);
				return (
					<Tooltip title={imageText}>
						<div
							style={{
								maxWidth: 160,
								overflow: "hidden",
								textOverflow: "ellipsis",
								whiteSpace: "nowrap",
							}}
						>
							{imageText}
						</div>
					</Tooltip>
				);
			},
		},
		{
			title: "描述",
			dataIndex: "description",
			key: "description",
			width: 260,
			ellipsis: true,
			render: (value: string) => (
				<Typography.Text ellipsis={{ tooltip: value || "-" }}>
					{value || "-"}
				</Typography.Text>
			),
		},
		{
			title: "来源",
			dataIndex: "source",
			key: "source",
			width: 88,
			render: (source: string) => (
				<Tag color={source === "system" ? "gold" : "default"}>{source}</Tag>
			),
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 150,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "更新时间",
			dataIndex: "updatedAt",
			key: "updatedAt",
			width: 150,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "操作",
			key: "actions",
			width: 210,
			render: (_, record) => {
				const isSystemComponent = record.source === "system";
				return (
					<Space size={4} style={{ whiteSpace: "nowrap" }}>
						<Button
							type="link"
							size="small"
							icon={<EyeOutlined />}
							aria-label="查看组件"
							onClick={() => openView(record)}
						>
							查看
						</Button>
						<Button
							type="link"
							size="small"
							icon={<EditOutlined />}
							aria-label="编辑组件"
							onClick={() => openEdit(record)}
						>
							编辑
						</Button>
						<Tooltip
							title={isSystemComponent ? "系统来源组件禁止删除" : "删除组件"}
						>
							<Popconfirm
								title="删除组件"
								description={`确认删除 ${record.name}？此操作无法撤销。`}
								okText="确认删除"
								cancelText="取消"
								disabled={isSystemComponent}
								onConfirm={() => handleDelete(record)}
							>
								<Button
									type="link"
									size="small"
									danger
									disabled={isSystemComponent}
									icon={<DeleteOutlined />}
									aria-label="删除组件"
								>
									删除
								</Button>
							</Popconfirm>
						</Tooltip>
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
						步骤组件
					</Typography.Title>
					<Typography.Text type="secondary">
						管理可复用的流水线步骤定义
					</Typography.Text>
				</div>
				<Space wrap>
					{selectedComponentIds.length > 0 ? (
						<Popconfirm
							title={`删除选中的 ${selectedComponentIds.length} 个组件？`}
							description="删除后不可恢复。系统组件不可选择。"
							okText="确认删除"
							cancelText="取消"
							onConfirm={handleBulkDelete}
						>
							<Button danger icon={<DeleteOutlined />} loading={bulkDeleting}>
								批量删除（{selectedComponentIds.length}）
							</Button>
						</Popconfirm>
					) : null}
					<Input.Search
						id="component-manager-search"
						allowClear
						placeholder="搜索名称、镜像或描述"
						value={search}
						onChange={(event) => setSearch(event.target.value)}
						style={{ width: 280 }}
					/>
					<Button
						icon={<ReloadOutlined />}
						onClick={refresh}
						loading={loading}
					/>
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

			{!loading && filteredItems.length === 0 ? (
				<Empty description="暂无组件，点击「新建组件」创建第一个步骤定义" />
			) : (
				<Table
					rowKey="id"
					loading={loading}
					columns={columns}
					dataSource={filteredItems}
					rowSelection={{
						selectedRowKeys: selectedComponentIds,
						onChange: (keys) => setSelectedComponentIds(keys as string[]),
						getCheckboxProps: (record) => ({
							disabled: record.source === "system",
							name: record.name,
						}),
					}}
					pagination={{ pageSize: 12, showSizeChanger: true }}
				/>
			)}

			<Modal
				open={modalOpen}
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
						: undefined
				}
			>
				{isViewMode && activeComponent ? (
					<ComponentDetail component={activeComponent} />
				) : (
					<Form
						form={form}
						layout="vertical"
						style={{ marginTop: 16 }}
						preserve={false}
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
									{fields.map((field, index) => (
										<Space
											key={field.key}
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
		</div>
	);
}
