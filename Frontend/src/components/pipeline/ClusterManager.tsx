import {
	DeleteOutlined,
	EditOutlined,
	PlusOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Button,
	Card,
	Checkbox,
	Form,
	Input,
	InputNumber,
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
	type Cluster,
	createCluster,
	deleteCluster,
	listClusters,
	updateCluster,
} from "../../api/pipelineApi";
import { useAuth } from "../../hooks/useAuth";

const { Text } = Typography;

interface ClusterFormValues {
	name: string;
	displayName: string;
	description?: string;
	isDefault?: boolean;
	status?: string;
	k8sApiEndpoint?: string;
	k8sAudience?: string;
	k8sCaData?: string;
	argoServerUrl?: string;
	argoNamespace?: string;
	koordInstalled?: boolean;
	clientQps?: number;
	clientBurst?: number;
}

// ClusterManager renders an admin-only CRUD panel for K8s clusters (CYB-3425
// PR 3). Non-admin users don't see the panel at all — cluster reads still
// happen elsewhere (target pool dropdown), so this component's only job is
// to write.
//
// The backend REJECTS writes from non-admins at the router, so this is
// defense-in-depth for UX; a curl-armed non-admin still bounces off 403.
export default function ClusterManager() {
	const { user } = useAuth();
	const isAdmin = user?.role === "admin";

	const [clusters, setClusters] = useState<Cluster[]>([]);
	const [loading, setLoading] = useState(false);
	const [modalOpen, setModalOpen] = useState(false);
	const [saving, setSaving] = useState(false);
	const [editing, setEditing] = useState<Cluster | null>(null);
	const [form] = Form.useForm<ClusterFormValues>();

	const load = useCallback(async () => {
		setLoading(true);
		try {
			const cs = await listClusters();
			setClusters(cs);
		} catch (err) {
			message.error(err instanceof Error ? err.message : "加载集群失败");
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		if (isAdmin) void load();
	}, [isAdmin, load]);

	const openCreate = () => {
		setEditing(null);
		form.resetFields();
		form.setFieldsValue({
			status: "available",
			koordInstalled: false,
			clientQps: 50,
			clientBurst: 100,
		});
		setModalOpen(true);
	};

	const openEdit = (c: Cluster) => {
		setEditing(c);
		form.setFieldsValue({
			name: c.name,
			displayName: c.displayName,
			description: c.description,
			isDefault: c.isDefault,
			status: c.status || "available",
			k8sApiEndpoint: c.k8sApiEndpoint,
			k8sAudience: c.k8sAudience,
			k8sCaData: c.k8sCaData,
			argoServerUrl: c.argoServerUrl,
			argoNamespace: c.argoNamespace,
			koordInstalled: c.koordInstalled,
			clientQps: c.clientQps ?? 50,
			clientBurst: c.clientBurst ?? 100,
		});
		setModalOpen(true);
	};

	const handleSave = async () => {
		let values: ClusterFormValues;
		try {
			values = await form.validateFields();
		} catch {
			return;
		}
		// Trim string fields; empty strings are semantic "unset" that the backend
		// then reads from env vars (default-cluster compatibility path).
		const body: Partial<Cluster> = {
			name: values.name.trim(),
			displayName: values.displayName.trim(),
			description: values.description?.trim(),
			isDefault: !!values.isDefault,
			status: (values.status || "available").trim(),
			k8sApiEndpoint: values.k8sApiEndpoint?.trim(),
			k8sAudience: values.k8sAudience?.trim(),
			k8sCaData: values.k8sCaData?.trim(),
			argoServerUrl: values.argoServerUrl?.trim(),
			argoNamespace: values.argoNamespace?.trim(),
			koordInstalled: !!values.koordInstalled,
			// Cleared InputNumber yields null; send 0 so the backend applies its
			// documented "0 → default" (50 / 100) path rather than a null literal.
			clientQps: values.clientQps ?? 0,
			clientBurst: values.clientBurst ?? 0,
		};

		setSaving(true);
		try {
			if (editing) {
				await updateCluster(editing.id, body);
				message.success("已更新集群");
			} else {
				await createCluster(body);
				message.success("已创建集群");
			}
			setModalOpen(false);
			await load();
		} catch (err) {
			// Backend returns 409 with `code:"cluster_name_exists"` for duplicate
			// names; the ApiError message covers that verbatim.
			message.error(err instanceof Error ? err.message : "操作失败");
		} finally {
			setSaving(false);
		}
	};

	const handleDelete = (c: Cluster) => {
		if (c.isDefault) {
			message.warning("默认集群不能删除,请先取消默认再删");
			return;
		}
		Modal.confirm({
			title: "删除集群",
			content: `确认删除「${c.displayName || c.name}」？如果还有 target 引用它,后端会拒绝。`,
			okText: "删除",
			okType: "danger",
			onOk: async () => {
				try {
					await deleteCluster(c.id);
					message.success("已删除");
					await load();
				} catch (err) {
					// 409 = still referenced by targets; message body carries "references: N"
					message.error(err instanceof Error ? err.message : "删除失败");
				}
			},
		});
	};

	if (!isAdmin) return null;

	const columns = [
		{
			title: "名称",
			key: "name",
			render: (_: unknown, c: Cluster) => (
				<Text strong>
					{c.displayName || c.name}
					{c.isDefault ? (
						<Tag color="blue" style={{ marginLeft: 6 }}>
							默认
						</Tag>
					) : null}
				</Text>
			),
		},
		{
			title: "ID",
			dataIndex: "name",
			key: "id",
			width: 180,
			render: (name: string) => (
				<Text style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>
					{name}
				</Text>
			),
		},
		{
			title: "API endpoint",
			dataIndex: "k8sApiEndpoint",
			key: "endpoint",
			render: (v?: string) =>
				v ? (
					<Text style={{ fontFamily: "var(--font-mono)", fontSize: 11 }}>
						{v}
					</Text>
				) : (
					<Text type="secondary" style={{ fontSize: 12 }}>
						env-derived
					</Text>
				),
		},
		{
			title: "Koord",
			dataIndex: "koordInstalled",
			key: "koord",
			width: 80,
			render: (v: boolean) =>
				v ? <Tag color="green">已装</Tag> : <Tag>未装</Tag>,
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 80,
			render: (s: string) => (
				<Tag color={s === "available" ? "green" : "default"}>
					{s === "available" ? "可用" : s || "—"}
				</Tag>
			),
		},
		{
			key: "actions",
			width: 90,
			render: (_: unknown, c: Cluster) => (
				<Space size={4}>
					<Tooltip title="编辑">
						<Button
							type="text"
							size="small"
							icon={<EditOutlined />}
							onClick={() => openEdit(c)}
							aria-label={`edit-cluster-${c.name}`}
						/>
					</Tooltip>
					<Tooltip title={c.isDefault ? "默认集群不能删" : "删除"}>
						<Button
							type="text"
							size="small"
							danger
							disabled={c.isDefault}
							icon={<DeleteOutlined />}
							onClick={() => handleDelete(c)}
							aria-label={`delete-cluster-${c.name}`}
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
				title="集群管理"
				style={{ marginBottom: 12 }}
				extra={
					<Space size={8}>
						<Button
							size="small"
							icon={<ReloadOutlined />}
							onClick={load}
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
							新建集群
						</Button>
					</Space>
				}
			>
				<Table
					size="small"
					rowKey="id"
					dataSource={clusters}
					columns={columns}
					loading={loading}
					pagination={false}
					locale={{ emptyText: "暂无集群" }}
				/>
				<div style={{ marginTop: 12, color: "var(--gray-400)", fontSize: 12 }}>
					<Text type="secondary">
						集群定义 K8s 执行目的地。API endpoint / audience / CA data 留空
						时后端 fallback 到 env 默认(为 default cluster
						兼容)。新集群配好这些字段后,resource pool
						就可以在下拉里选中这个集群。
					</Text>
				</div>
			</Card>

			<Modal
				title={editing ? "编辑集群" : "新建集群"}
				open={modalOpen}
				onOk={handleSave}
				onCancel={() => setModalOpen(false)}
				confirmLoading={saving}
				okText={editing ? "保存" : "创建"}
				width={720}
				destroyOnClose
			>
				<Form form={form} layout="vertical" style={{ marginTop: 16 }}>
					<Form.Item
						name="name"
						label="名称(ID,创建后不建议改)"
						rules={[
							{ required: true, message: "请输入名称" },
							{
								pattern: /^[a-zA-Z0-9_-]+$/,
								message: "仅支持字母、数字、-、_",
							},
						]}
					>
						<Input placeholder="例如: delivery-clust" />
					</Form.Item>
					<Form.Item
						name="displayName"
						label="显示名(可读中文)"
						rules={[{ required: true, message: "请输入显示名" }]}
					>
						<Input placeholder="例如: 客户交付集群" />
					</Form.Item>
					<Form.Item name="description" label="描述">
						<Input.TextArea rows={2} placeholder="可选,一句话说明用途" />
					</Form.Item>
					<Space size="large">
						<Form.Item
							name="koordInstalled"
							label="Koord 已安装"
							valuePropName="checked"
						>
							<Checkbox>装了 Koordinator scheduler + ElasticQuota</Checkbox>
						</Form.Item>
						<Form.Item
							name="isDefault"
							label="设为默认"
							valuePropName="checked"
						>
							<Checkbox>新建 target 时默认选此集群(全局仅能一个默认)</Checkbox>
						</Form.Item>
					</Space>

					<Typography.Title level={5} style={{ marginTop: 8 }}>
						K8s API 连接
					</Typography.Title>
					<Text type="secondary" style={{ fontSize: 12 }}>
						留空则使用后端 env vars(K8S_API_ENDPOINT / K8S_AUDIENCE /
						K8S_CA_DATA),默认集群保持这样即可。跨项目集群必填。
					</Text>
					<Form.Item
						name="k8sApiEndpoint"
						label="API endpoint"
						style={{ marginTop: 8 }}
					>
						<Input placeholder="例如: https://34.44.27.160" />
					</Form.Item>
					<Form.Item name="k8sAudience" label="WIF audience">
						<Input placeholder="例如: //iam.googleapis.com/projects/PROJECT/locations/us-central1/workloadIdentityPools/POOL/providers/PROVIDER" />
					</Form.Item>
					<Form.Item name="k8sCaData" label="CA data(base64 或 PEM)">
						<Input.TextArea rows={3} placeholder="base64 encoded CA cert" />
					</Form.Item>

					<Typography.Title level={5} style={{ marginTop: 8 }}>
						Argo Workflow
					</Typography.Title>
					<Form.Item name="argoServerUrl" label="Argo server URL">
						<Input placeholder="例如: http://argo-server.cyber-databrew-dev:2746" />
					</Form.Item>
					<Form.Item name="argoNamespace" label="默认 Argo namespace">
						<Input placeholder="例如: cyber-databrew-dev" />
					</Form.Item>
					<Form.Item
						name="clientQps"
						label="K8s 客户端 QPS"
						tooltip="后端调用本集群 K8s API 的每秒请求上限。留空/0 用默认 50。改后几秒生效,无需重启。批量下发吞吐受此限制。"
					>
						<InputNumber
							min={1}
							max={1000}
							style={{ width: "100%" }}
							placeholder="50"
						/>
					</Form.Item>
					<Form.Item
						name="clientBurst"
						label="K8s 客户端 Burst"
						tooltip="突发请求上限(令牌桶容量),一般设为 QPS 的 ~2 倍。留空/0 用默认 100。"
					>
						<InputNumber
							min={1}
							max={2000}
							precision={0}
							style={{ width: "100%" }}
							placeholder="100"
						/>
					</Form.Item>
				</Form>
			</Modal>
		</>
	);
}
