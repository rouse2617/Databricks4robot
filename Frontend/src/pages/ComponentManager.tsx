import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Form,
	Input,
	Modal,
	message,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
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

interface ComponentFormValues {
	name: string;
	type: PipelineComponentType;
	image: string;
	tag?: string;
	description?: string;
	command?: string[];
	args?: string[];
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

function toFormValues(
	component?: PipelineComponentAPI,
): ComponentFormValues {
	if (!component) {
		return {
			name: "",
			type: "container",
			image: "",
			tag: "latest",
			description: "",
			command: [],
			args: [],
			envRows: [],
		};
	}

	return {
		name: component.name,
		type: component.type || "container",
		image: component.image,
		tag: component.tag || "latest",
		description: component.description || "",
		command: component.command || [],
		args: component.args || [],
		envRows: Object.entries(component.env || {}).map(([name, value]) => ({
			name,
			value,
		})),
	};
}

function toPayload(values: ComponentFormValues): PipelineComponentPayload {
	const env: Record<string, string> = {};
	for (const row of values.envRows || []) {
		const name = row.name?.trim();
		if (name) env[name] = row.value || "";
	}

	const command = (values.command || []).map((v) => v.trim()).filter(Boolean);
	const args = (values.args || []).map((v) => v.trim()).filter(Boolean);

	return {
		name: values.name.trim(),
		type: values.type,
		image: values.image.trim(),
		tag: values.tag?.trim() || "latest",
		source: "custom",
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
	const [editing, setEditing] = useState<PipelineComponentAPI | null>(null);
	const [form] = Form.useForm<ComponentFormValues>();
	const [messageApi, contextHolder] = message.useMessage();

	const refresh = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const res = await listComponents();
			setItems(dedupePipelineComponentsByName(res.items || []));
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

	const openCreate = () => {
		setEditing(null);
		form.setFieldsValue(toFormValues());
		setModalOpen(true);
	};

	const openEdit = (component: PipelineComponentAPI) => {
		setEditing(component);
		form.setFieldsValue(toFormValues(component));
		setModalOpen(true);
	};

	const handleSave = async () => {
		const values = await form.validateFields();
		setSaving(true);
		try {
			const payload = toPayload(values);
			if (editing) {
				await updateComponent(editing.id, payload);
				messageApi.success("组件已更新");
			} else {
				await createComponent(payload);
				messageApi.success("组件已创建");
			}
			setModalOpen(false);
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
			ellipsis: true,
			render: (_, record) => formatComponentImage(record.image, record.tag),
		},
		{
			title: "描述",
			dataIndex: "description",
			key: "description",
			ellipsis: true,
			render: (value: string) => value || "-",
		},
		{
			title: "来源",
			dataIndex: "source",
			key: "source",
			width: 100,
			render: (source: string) => (
				<Tag color={source === "system" ? "gold" : "default"}>{source}</Tag>
			),
		},
		{
			title: "创建时间",
			dataIndex: "createdAt",
			key: "createdAt",
			width: 180,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "更新时间",
			dataIndex: "updatedAt",
			key: "updatedAt",
			width: 180,
			render: (value: string) =>
				value ? new Date(value).toLocaleString() : "-",
		},
		{
			title: "操作",
			key: "actions",
			width: 160,
			render: (_, record) => (
				<Space>
					<Button
						type="text"
						icon={<EditOutlined />}
						aria-label="编辑组件"
						onClick={() => openEdit(record)}
					/>
					<Popconfirm
						title="删除组件"
						description={`确认删除 ${record.name}？`}
						okText="删除"
						cancelText="取消"
						disabled={record.source === "system"}
						onConfirm={() => handleDelete(record)}
					>
						<Button
							type="text"
							danger
							disabled={record.source === "system"}
							icon={<DeleteOutlined />}
							aria-label="删除组件"
						/>
					</Popconfirm>
				</Space>
			),
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
				/>
			) : null}

			<Table
				rowKey="id"
				loading={loading}
				columns={columns}
				dataSource={filteredItems}
				pagination={{ pageSize: 12, showSizeChanger: true }}
				scroll={{ x: 900 }}
			/>

			<Modal
				open={modalOpen}
				title={editing ? "编辑组件" : "新建组件"}
				okText={editing ? "保存" : "创建"}
				cancelText="取消"
				confirmLoading={saving}
				onOk={handleSave}
				onCancel={() => setModalOpen(false)}
				width={760}
				destroyOnHidden
			>
				<Form
					form={form}
					layout="vertical"
					initialValues={toFormValues()}
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
							<Input placeholder="normalize-mcap" />
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
							<Input placeholder="registry.example.com/databrew/worker" />
						</Form.Item>
						<Form.Item name="tag" label="标签">
							<Input placeholder="latest" />
						</Form.Item>
					</div>

					<Form.Item name="description" label="描述">
						<Input.TextArea rows={3} maxLength={500} showCount />
					</Form.Item>

					<Form.Item name="command" label="命令">
						<Select
							mode="tags"
							tokenSeparators={[","]}
							placeholder="例如 python, /app/main.py"
						/>
					</Form.Item>

					<Form.Item name="args" label="参数">
						<Select
							mode="tags"
							tokenSeparators={[","]}
							placeholder="例如 --input, {{inputs.asset}}"
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
								{fields.map((field) => (
									<Space
										key={field.key}
										align="baseline"
										style={{ display: "flex", marginBottom: 8 }}
									>
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
			</Modal>
		.</div>
	);
}
