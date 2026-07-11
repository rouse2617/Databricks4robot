import {
	DeleteOutlined,
	EditOutlined,
	PlusOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Button,
	Card,
	Form,
	Input,
	Modal,
	message,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	type ExecutionTarget,
	listExecutionTargets,
} from "../../api/pipelineApi";

const { Text } = Typography;

interface QuotaInfo {
	cpu: { used: string; hard: string };
	memory: { used: string; hard: string };
}

export default function PoolManager() {
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [quotas, setQuotas] = useState<Record<string, QuotaInfo>>({});
	const [loading, setLoading] = useState(false);
	const [modalOpen, setModalOpen] = useState(false);
	const [editTarget, setEditTarget] = useState<ExecutionTarget | null>(null);
	const [form] = Form.useForm();

	const fetchData = useCallback(async () => {
		setLoading(true);
		try {
			const [t, q] = await Promise.all([
				listExecutionTargets(),
				fetch("/api/v1/resource-quotas").then((r) => r.json()),
			]);
			setTargets(t);
			setQuotas(q.items || {});
		} catch {
			/* ignore */
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		fetchData();
	}, [fetchData]);

	const openCreate = () => {
		setEditTarget(null);
		form.resetFields();
		setModalOpen(true);
	};

	const openEdit = (target: ExecutionTarget) => {
		setEditTarget(target);
		form.setFieldsValue({
			name: target.name,
			namespace: target.namespace,
			cluster: target.cluster,
		});
		setModalOpen(true);
	};

	const handleSave = async () => {
		try {
			const values = await form.validateFields();
			const body = {
				name: values.name,
				namespace: values.namespace,
				cluster: values.cluster || "default",
				status: "available" as const,
			};

			if (editTarget) {
				await fetch(`/api/v1/execution-targets/${editTarget.id}`, {
					method: "PUT",
					headers: {
						"Content-Type": "application/json",
						"X-Databrew-Token": "dev-token",
					},
					body: JSON.stringify(body),
				});
				message.success("已更新");
			} else {
				await fetch("/api/v1/execution-targets", {
					method: "POST",
					headers: {
						"Content-Type": "application/json",
						"X-Databrew-Token": "dev-token",
					},
					body: JSON.stringify(body),
				});
				message.success("已创建");
			}
			setModalOpen(false);
			fetchData();
		} catch {
			message.error("操作失败");
		}
	};

	const cpuPct = (q?: QuotaInfo) => {
		if (!q) return 0;
		return Math.min(
			100,
			Math.round((parseFloat(q.cpu.used) / parseFloat(q.cpu.hard)) * 100),
		);
	};

	const columns = [
		{
			title: "名称",
			dataIndex: "name",
			key: "name",
			render: (name: string, r: ExecutionTarget) => (
				<Text strong>
					{name}
					{r.isDefault ? (
						<Tag color="blue" style={{ marginLeft: 6 }}>
							默认
						</Tag>
					) : null}
				</Text>
			),
		},
		{
			title: "命名空间",
			dataIndex: "namespace",
			key: "ns",
			width: 160,
			render: (ns: string) => (
				<Text style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>
					{ns}
				</Text>
			),
		},
		{
			title: "CPU",
			key: "cpu",
			width: 180,
			render: (_: unknown, r: ExecutionTarget) => {
				const q = quotas[r.namespace];
				const pct = cpuPct(q);
				if (!q) return <Text type="secondary">—</Text>;
				const color = pct > 80 ? "#ef4444" : pct > 60 ? "#f59e0b" : "#22c55e";
				return (
					<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
						<div
							style={{
								flex: 1,
								height: 8,
								background: "#e5e7eb",
								borderRadius: 4,
								overflow: "hidden",
							}}
						>
							<div
								style={{
									width: `${pct}%`,
									height: "100%",
									background: color,
									borderRadius: 4,
									transition: "width 0.3s",
								}}
							/>
						</div>
						<Text
							style={{
								fontSize: 11,
								fontFamily: "var(--font-mono)",
								color,
								minWidth: 50,
							}}
						>
							{q.cpu.used}/{q.cpu.hard}
						</Text>
					</div>
				);
			},
		},
		{
			title: "MEM",
			key: "mem",
			width: 200,
			render: (_: unknown, r: ExecutionTarget) => {
				const q = quotas[r.namespace];
				if (!q) return <Text type="secondary">—</Text>;
				const used = parseInt(q.memory.used, 10) || 0;
				const hard = parseInt(q.memory.hard, 10) || 1;
				const pct = Math.min(100, Math.round(used / hard));
				const color = pct > 80 ? "#ef4444" : pct > 60 ? "#f59e0b" : "#22c55e";
				return (
					<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
						<div
							style={{
								flex: 1,
								height: 8,
								background: "#e5e7eb",
								borderRadius: 4,
								overflow: "hidden",
							}}
						>
							<div
								style={{
									width: `${pct}%`,
									height: "100%",
									background: color,
									borderRadius: 4,
									transition: "width 0.3s",
								}}
							/>
						</div>
						<Text
							style={{
								fontSize: 11,
								fontFamily: "var(--font-mono)",
								color,
								minWidth: 60,
							}}
						>
							{q.memory.used}/{q.memory.hard}
						</Text>
					</div>
				);
			},
		},
		{
			title: "状态",
			dataIndex: "status",
			width: 80,
			render: (s: string) => (
				<Tag color={s === "available" ? "green" : "default"}>
					{s === "available" ? "可用" : "不可用"}
				</Tag>
			),
		},
		{
			key: "actions",
			width: 80,
			render: (_: unknown, r: ExecutionTarget) =>
				r.isDefault ? null : (
					<Space size={4}>
						<Tooltip title="编辑">
							<Button
								type="text"
								size="small"
								icon={<EditOutlined />}
								onClick={() => openEdit(r)}
							/>
						</Tooltip>
						<Tooltip title="删除">
							<Button
								type="text"
								size="small"
								danger
								icon={<DeleteOutlined />}
								onClick={() =>
									Modal.confirm({
										title: "删除资源池",
										content: `确认删除「${r.name}」？`,
										okText: "删除",
										okType: "danger",
										onOk: async () => {
											try {
												await fetch(`/api/v1/execution-targets/${r.id}`, {
													method: "DELETE",
													headers: { "X-Databrew-Token": "dev-token" },
												});
												message.success("已删除");
												fetchData();
											} catch {
												message.error("删除失败");
											}
										},
									})
								}
							/>
						</Tooltip>
					</Space>
				),
		},
	];

	return (
		<>
			<Card
				size="small"
				title="资源池管理"
				extra={
					<Space size={8}>
						<Button
							size="small"
							icon={<ReloadOutlined />}
							onClick={fetchData}
							loading={loading}
						>
							刷新
						</Button>
						<Button
							size="small"
							type="primary"
							icon={<PlusOutlined />}
							onClick={openCreate}
						>
							新建
						</Button>
					</Space>
				}
			>
				<Table
					size="small"
					rowKey="id"
					dataSource={targets}
					columns={columns}
					loading={loading}
					pagination={false}
					locale={{ emptyText: "暂无资源池" }}
				/>
				<div style={{ marginTop: 12, color: "var(--gray-400)", fontSize: 12 }}>
					<Typography.Text type="secondary">
						资源池对应 K8s 命名空间，部署流水线时选择资源池即提交 Workflow
						到对应命名空间
						{Object.keys(quotas).length > 0
							? "，受 ResourceQuota 约束"
							: "；当前命名空间未配置 ResourceQuota"}
						。
					</Typography.Text>
				</div>
			</Card>

			<Modal
				title={editTarget ? "编辑资源池" : "新建资源池"}
				open={modalOpen}
				onOk={handleSave}
				onCancel={() => setModalOpen(false)}
				okText={editTarget ? "保存" : "创建"}
				width={480}
			>
				<Form form={form} layout="vertical" style={{ marginTop: 16 }}>
					<Form.Item
						name="name"
						label="名称"
						rules={[{ required: true, message: "请输入名称" }]}
					>
						<Input placeholder="例如: 客户A生产池" />
					</Form.Item>
					<Form.Item
						name="namespace"
						label="K8s 命名空间"
						rules={[{ required: true, message: "请输入命名空间" }]}
					>
						<Input placeholder="例如: pool-customer-a" />
					</Form.Item>
					<Form.Item name="cluster" label="集群" initialValue="default">
						<Input placeholder="default" />
					</Form.Item>
				</Form>
			</Modal>
		</>
	);
}
