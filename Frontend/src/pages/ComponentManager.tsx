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
import {
	dedupePipelineComponentsByName,
	formatComponentImage,
} from "../lib/pipelineComponentDisplay";

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
	const args = splitInputItems(values.args);

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
					<Typography.Text strong>{name}</Typography.Text>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{record.id}
					</Typography.Text>
				</Space>
			),
		},
		{
			title: "类型",
			dataIndex: "type",
			key: "type",
			width: 120,
			render: (type: PipelineComponentType) => (
				<Tag color={TYPE_COLORS[type] || "default"}>{type || "container"}</Tag>
			),
		},
		{
			title: "镜像",
			key: "image",
			width: 240,
			render: (_, record) => {
				const imageText = formatComponentImage(record.image, record.tag);
				return (
					<Tooltip title={imageText}>
						<div
							style={{
								maxWidth: 220,
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
			width: 220,
			ellipsis: true,
			render: (value: string) => value || "-",
		},
		{
			title: "来源",
			dataIndex: "source",
			key: "source",
			width: 110,
			render: (source: string) => (
				<Tag color={source === "system" ? "gold" : "default"}>{source}</Tag>
			),
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 170,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "更新时间",
			dataIndex: "updatedAt",
			key: "updatedAt",
			width: 170,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "操作",
			key: "actions",
			width: 170,
			render: (_, record) => {
				const isSystemComponent = record.source === "system";
				return (
					<Space>
						<Button
							type="text"
							icon={<EyeOutlined />}
							aria-label="查看组件"
							onClick={() => openView(record)}
						/>
						<Button
							type="text"
							icon={<EditOutlined />}
							aria-label="编辑组件"
							onClick={() => openEdit(record)}
						/>
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
									type="text"
									danger
									disabled={isSystemComponent}
									icon={<DeleteOutlined />}
									aria-label="删除组件"
								/>
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
					scroll={{ x: 900 }}
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
				destroyOnClose
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
								disabled={isViewMode}
								ref={nameInputRef}
							/>
						</Form.Item>
						<Form.Item
							name="type"
							label="类型"
							rules={[{ required: true, message: "请选择组件类型" }]}
						>
							<Select options={TYPE_OPTIONS} disabled={isViewMode} />
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
								disabled={isViewMode}
							/>
						</Form.Item>
						<Form.Item name="tag" label="标签">
							<Input placeholder="latest" disabled={isViewMode} />
						</Form.Item>
					</div>

					<Form.Item name="description" label="描述">
						<Input.TextArea
							rows={3}
							maxLength={500}
							showCount
							disabled={isViewMode}
						/>
					</Form.Item>

					<Form.Item name="command" label="命令">
						<Input.TextArea
							rows={2}
							placeholder="例如: python, /app/main.py"
							disabled={isViewMode}
						/>
					</Form.Item>

					<Form.Item name="args" label="参数">
						<Input.TextArea
							rows={2}
							placeholder="例如: --input, {{inputs.asset}}"
							disabled={isViewMode}
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
									{!isViewMode ? (
										<Button size="small" onClick={() => add({})}>
											添加变量
										</Button>
									) : null}
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
											<Input
												placeholder="KEY"
												style={{ width: 220 }}
												disabled={isViewMode}
											/>
										</Form.Item>
										<Form.Item {...field} name={[field.name, "value"]}>
											<Input
												placeholder="value"
												style={{ width: 320 }}
												disabled={isViewMode}
											/>
										</Form.Item>
										{!isViewMode ? (
											<Button
												type="text"
												danger
												icon={<DeleteOutlined />}
												aria-label="移除环境变量"
												onClick={() => remove(field.name)}
											/>
										) : null}
									</Space>
								))}
							</div>
						)}
					</Form.List>
				</Form>
			</Modal>
		</div>
	);
}
