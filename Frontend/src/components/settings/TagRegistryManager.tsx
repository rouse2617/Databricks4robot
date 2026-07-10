import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Form,
	Input,
	InputNumber,
	Modal,
	message,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	type TagDefRequest,
	type TagRegistryEntry,
	tagRegistryAdminApi,
} from "../../api/tagRegistry";
import { extractApiErrorMessage } from "../../lib/apiError";

function statusFrom(err: unknown): number | undefined {
	return (err as { response?: { status?: number } })?.response?.status;
}

export default function TagRegistryManager() {
	const [items, setItems] = useState<TagRegistryEntry[]>([]);
	const [loading, setLoading] = useState(false);
	const [denied, setDenied] = useState(false);
	const [modalOpen, setModalOpen] = useState(false);
	const [editingKey, setEditingKey] = useState<string | null>(null);
	const [saving, setSaving] = useState(false);
	const [form] = Form.useForm<TagDefRequest>();
	const [msg, msgCtx] = message.useMessage();

	const type = Form.useWatch("type", form);

	const load = useCallback(async () => {
		setLoading(true);
		try {
			const list = await tagRegistryAdminApi.list();
			setItems(list);
			setDenied(false);
		} catch (err) {
			if (statusFrom(err) === 401) {
				setDenied(true);
			} else {
				msg.error(extractApiErrorMessage(err, "加载标签注册表失败"));
			}
		} finally {
			setLoading(false);
		}
	}, [msg]);

	useEffect(() => {
		load();
	}, [load]);

	const openCreate = () => {
		setEditingKey(null);
		form.resetFields();
		form.setFieldsValue({ type: "string", propagation: "none" });
		setModalOpen(true);
	};

	const openEdit = (entry: TagRegistryEntry) => {
		setEditingKey(entry.key);
		form.setFieldsValue({
			key: entry.key,
			description: entry.description,
			type: entry.type,
			values: entry.values ?? [],
			max_length: entry.max_length ?? 0,
			propagation: entry.propagation ?? "none",
		});
		setModalOpen(true);
	};

	const submit = async () => {
		let body: TagDefRequest;
		try {
			body = await form.validateFields();
		} catch {
			return; // form shows field errors
		}
		setSaving(true);
		try {
			if (editingKey) {
				await tagRegistryAdminApi.update(editingKey, body);
				msg.success(`已更新标签 ${editingKey}`);
			} else {
				await tagRegistryAdminApi.create(body);
				msg.success(`已注册标签 ${body.key}`);
			}
			setModalOpen(false);
			await load();
		} catch (err) {
			if (statusFrom(err) === 409) {
				msg.error(`标签 ${body.key} 已存在`);
			} else {
				msg.error(extractApiErrorMessage(err, "保存标签失败"));
			}
		} finally {
			setSaving(false);
		}
	};

	const remove = async (key: string) => {
		try {
			await tagRegistryAdminApi.remove(key);
			msg.success(`已删除标签 ${key}`);
			await load();
		} catch (err) {
			msg.error(extractApiErrorMessage(err, "删除标签失败"));
		}
	};

	if (denied) {
		return (
			<Alert
				type="info"
				showIcon
				message="需要管理员权限"
				description="标签注册表管理仅对管理员开放（ADMIN_EMAILS 或 admin token）。未注册的自定义标签仍可在资产详情页自由添加（开放词汇）。"
			/>
		);
	}

	const columns = [
		{
			title: "Key",
			dataIndex: "key",
			key: "key",
			render: (k: string) => <strong>{k}</strong>,
		},
		{ title: "描述", dataIndex: "description", key: "description" },
		{
			title: "类型",
			dataIndex: "type",
			key: "type",
			render: (t: string) => (
				<Tag color={t === "enum" ? "blue" : "default"}>{t}</Tag>
			),
		},
		{
			title: "允许值 / 长度",
			key: "constraint",
			render: (_: unknown, r: TagRegistryEntry) =>
				r.type === "enum" ? (
					<Space wrap size={[4, 4]}>
						{(r.values ?? []).map((v) => (
							<Tag key={v}>{v}</Tag>
						))}
					</Space>
				) : (
					<span>{r.max_length ? `≤ ${r.max_length}` : "无限制"}</span>
				),
		},
		{
			title: "传播",
			dataIndex: "propagation",
			key: "propagation",
			render: (p?: string) =>
				p === "descendants" ? <Tag color="purple">子孙</Tag> : "—",
		},
		{
			title: "操作",
			key: "actions",
			width: 140,
			render: (_: unknown, r: TagRegistryEntry) => (
				<Space>
					<Button
						size="small"
						icon={<EditOutlined />}
						onClick={() => openEdit(r)}
					>
						编辑
					</Button>
					<Popconfirm
						title={`删除标签 ${r.key}？`}
						okText="删除"
						cancelText="取消"
						okButtonProps={{ danger: true }}
						onConfirm={() => remove(r.key)}
					>
						<Button size="small" danger icon={<DeleteOutlined />} />
					</Popconfirm>
				</Space>
			),
		},
	];

	return (
		<div>
			{msgCtx}
			<div
				style={{
					display: "flex",
					justifyContent: "space-between",
					alignItems: "center",
					marginBottom: 12,
				}}
			>
				<span style={{ color: "var(--color-text-secondary)" }}>
					受管标签（enum 允许值、string
					长度、传播策略）即时生效，无需重启。未注册 key 仍走开放词汇。
				</span>
				<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
					新增标签
				</Button>
			</div>
			<Table
				rowKey="key"
				size="small"
				loading={loading}
				columns={columns}
				dataSource={items}
				pagination={false}
			/>

			<Modal
				title={editingKey ? `编辑标签：${editingKey}` : "新增标签"}
				open={modalOpen}
				onOk={submit}
				onCancel={() => setModalOpen(false)}
				confirmLoading={saving}
				okText="保存"
				cancelText="取消"
				destroyOnClose
			>
				<Form form={form} layout="vertical" preserve={false}>
					<Form.Item
						name="key"
						label="标签 Key"
						rules={[
							{ required: true, message: "请输入标签 Key" },
							{
								pattern: /^[A-Za-z0-9_.]+$/,
								message: "仅允许字母、数字、下划线、点",
							},
						]}
					>
						<Input placeholder="如 severity" disabled={!!editingKey} />
					</Form.Item>
					<Form.Item name="description" label="描述">
						<Input placeholder="可选" />
					</Form.Item>
					<Form.Item
						name="type"
						label="类型"
						rules={[{ required: true, message: "请选择类型" }]}
					>
						<Select
							options={[
								{ label: "枚举 (enum)", value: "enum" },
								{ label: "字符串 (string)", value: "string" },
							]}
						/>
					</Form.Item>
					{type === "enum" ? (
						<Form.Item
							name="values"
							label="允许值"
							rules={[{ required: true, message: "枚举类型需至少一个允许值" }]}
						>
							<Select
								mode="tags"
								placeholder="输入后回车添加，如 critical / high / low"
								tokenSeparators={[",", " "]}
							/>
						</Form.Item>
					) : (
						<Form.Item name="max_length" label="最大长度（0 = 无限制）">
							<InputNumber min={0} style={{ width: "100%" }} />
						</Form.Item>
					)}
					<Form.Item name="propagation" label="传播策略">
						<Select
							options={[
								{ label: "不传播 (none)", value: "none" },
								{ label: "传播到子孙资产 (descendants)", value: "descendants" },
							]}
						/>
					</Form.Item>
				</Form>
			</Modal>
		</div>
	);
}
