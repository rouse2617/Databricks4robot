import {
	Alert,
	App,
	Button,
	Card,
	DatePicker,
	Form,
	Input,
	Modal,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import type { Dayjs } from "dayjs";
import { useCallback, useEffect, useState } from "react";

import {
	AVAILABLE_SCOPES,
	type ApiKey,
	apiKeysApi,
} from "../api/apiKeys";
import { useAuth } from "../hooks/useAuth";

function fmt(ts?: string | null): string {
	if (!ts) return "—";
	const d = new Date(ts);
	return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString();
}

export default function ApiKeysManager() {
	const { user } = useAuth();
	const isAdmin = user?.role === "admin";
	const { message } = App.useApp();

	const [rows, setRows] = useState<ApiKey[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const [createOpen, setCreateOpen] = useState(false);
	const [creating, setCreating] = useState(false);
	const [form] = Form.useForm();
	const [newKey, setNewKey] = useState<string | null>(null);

	const load = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			setRows(await apiKeysApi.list());
		} catch (e) {
			setError(
				(e as { response?: { status?: number } })?.response?.status === 403
					? "需要管理员权限(ADMIN_EMAILS)才能管理 API Key。"
					: `加载失败:${String(e)}`,
			);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		if (isAdmin) void load();
	}, [isAdmin, load]);

	const onCreate = async () => {
		const v = await form.validateFields();
		setCreating(true);
		try {
			const resp = await apiKeysApi.create({
				name: v.name,
				owner: v.owner ?? "",
				scopes: v.scopes,
				expiresAt: (v.expiresAt as Dayjs | undefined)?.toISOString() ?? null,
			});
			setCreateOpen(false);
			form.resetFields();
			setNewKey(resp.key); // show once
			message.success("API Key 已创建");
			void load();
		} catch (e) {
			message.error(`创建失败:${String(e)}`);
		} finally {
			setCreating(false);
		}
	};

	const onRevoke = async (id: string) => {
		try {
			await apiKeysApi.revoke(id);
			message.success("已吊销");
			void load();
		} catch (e) {
			message.error(`吊销失败:${String(e)}`);
		}
	};

	if (!isAdmin) {
		return (
			<Card title="API Keys" size="small">
				<Alert
					type="info"
					showIcon
					message="仅管理员可见"
					description="API Key 管理需要管理员权限(邮箱在 ADMIN_EMAILS 白名单内)。"
				/>
			</Card>
		);
	}

	return (
		<Card
			title="API Keys(SDK / API 调用方)"
			size="small"
			extra={
				<Space>
					<Button onClick={load} loading={loading}>
						刷新
					</Button>
					<Button type="primary" onClick={() => setCreateOpen(true)}>
						创建密钥
					</Button>
				</Space>
			}
		>
			{error ? (
				<Alert
					type="error"
					showIcon
					message={error}
					style={{ marginBottom: 12 }}
				/>
			) : null}

			<Table<ApiKey>
				rowKey="id"
				size="small"
				loading={loading}
				dataSource={rows}
				pagination={{ pageSize: 10, hideOnSinglePage: true }}
				columns={[
					{ title: "名称", dataIndex: "name", render: (v) => v || "—" },
					{ title: "归属", dataIndex: "owner", render: (v) => v || "—" },
					{ title: "前缀", dataIndex: "keyPrefix", render: (v) => <code>{v}</code> },
					{
						title: "Scopes",
						dataIndex: "scopes",
						render: (s: string[]) =>
							(s ?? []).map((x) => <Tag key={x}>{x}</Tag>),
					},
					{
						title: "状态",
						dataIndex: "status",
						render: (s: string) => (
							<Tag color={s === "active" ? "green" : "red"}>{s}</Tag>
						),
					},
					{ title: "最近使用", dataIndex: "lastUsedAt", render: fmt },
					{ title: "创建者", dataIndex: "createdBy", render: (v) => v || "—" },
					{ title: "创建时间", dataIndex: "createdAt", render: fmt },
					{
						title: "操作",
						key: "actions",
						render: (_, row) =>
							row.status === "active" ? (
								<Popconfirm
									title="吊销后立即失效,不可恢复。确认?"
									okText="吊销"
									okButtonProps={{ danger: true }}
									onConfirm={() => onRevoke(row.id)}
								>
									<Button danger size="small">
										吊销
									</Button>
								</Popconfirm>
							) : (
								<Typography.Text type="secondary">已吊销</Typography.Text>
							),
					},
				]}
			/>

			{/* Create modal */}
			<Modal
				title="创建 API Key"
				open={createOpen}
				onOk={onCreate}
				confirmLoading={creating}
				onCancel={() => setCreateOpen(false)}
				okText="创建"
				cancelText="取消"
			>
				<Form form={form} layout="vertical" requiredMark="optional">
					<Form.Item
						name="name"
						label="名称"
						rules={[{ required: true, message: "给这把 key 起个名字" }]}
					>
						<Input placeholder="如:partner-x / grace-sync" />
					</Form.Item>
					<Form.Item name="owner" label="归属(可选)">
						<Input placeholder="调用方 / 团队" />
					</Form.Item>
					<Form.Item
						name="scopes"
						label="Scopes(最小权限)"
						rules={[{ required: true, message: "至少选一个 scope" }]}
					>
						<Select
							mode="multiple"
							placeholder="选择权限"
							options={AVAILABLE_SCOPES.map((s) => ({ label: s, value: s }))}
						/>
					</Form.Item>
					<Form.Item name="expiresAt" label="过期时间(可选)">
						<DatePicker showTime style={{ width: "100%" }} />
					</Form.Item>
				</Form>
			</Modal>

			{/* One-time key reveal */}
			<Modal
				title="密钥已创建 — 只显示这一次"
				open={newKey !== null}
				onOk={() => setNewKey(null)}
				onCancel={() => setNewKey(null)}
				okText="我已保存"
				cancelButtonProps={{ style: { display: "none" } }}
			>
				<Alert
					type="warning"
					showIcon
					style={{ marginBottom: 12 }}
					message="请立即复制并妥善保存,关闭后无法再次查看(库内只存哈希)。"
				/>
				<Typography.Paragraph
					copyable={{ text: newKey ?? "" }}
					code
					style={{ wordBreak: "break-all" }}
				>
					{newKey}
				</Typography.Paragraph>
				<Typography.Text type="secondary">
					调用方使用:{" "}
					<code>Authorization: Bearer {"<key>"}</code>
				</Typography.Text>
			</Modal>
		</Card>
	);
}
