import {
	DeleteOutlined,
	EditOutlined,
	MinusCircleOutlined,
	PlusOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	AutoComplete,
	Button,
	Card,
	Divider,
	Form,
	Input,
	Modal,
	message,
	Select,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
	type Cluster,
	createExecutionTarget,
	deleteExecutionTarget,
	type ElasticQuota,
	type ExecutionTarget,
	listClusters,
	listElasticQuotas,
	listExecutionTargets,
	type TargetScheduling,
	type TargetToleration,
	updateExecutionTarget,
} from "../../api/pipelineApi";
import ClusterManager from "./ClusterManager";
import { KOORD_EQ_LABEL_KEY, poolUsageSummary } from "./poolUsage";

const { Text } = Typography;

interface QuotaInfo {
	cpu: { used: string; hard: string };
	memory: { used: string; hard: string };
}

// A key/value row in a Form.List-backed map editor (nodeSelector, pod labels,
// pod annotations all share this shape).
interface KVEntry {
	key: string;
	value: string;
}

interface FormValues {
	name: string;
	namespace: string;
	clusterId?: string;
	description?: string;
	templateTolerations?: TargetToleration[];
	templateNodeSelector?: KVEntry[];
	// CYB-3486 pool.3: pool scheduling config. All optional; written to
	// resourceDefaults.scheduling. Empty = cluster / K8s defaults (any scheduler).
	// The backend stays generic; the frontend maps the "Koord 资源池" picker onto
	// the standard EQ pod-label (KOORD_EQ_LABEL_KEY) at save time.
	schedulerName?: string;
	priorityClassName?: string;
	// elasticQuotaName is a UI-only convenience field holding the chosen
	// Koordinator ElasticQuota. On save it is folded into
	// podLabels[KOORD_EQ_LABEL_KEY]; on load it is split back out of podLabels so
	// the picker owns that one key and the manual podLabels editor never shows it.
	elasticQuotaName?: string;
	podLabels?: KVEntry[];
	podAnnotations?: KVEntry[];
}

const TOLERATION_EFFECTS = [
	{ value: "NoSchedule" },
	{ value: "NoExecute" },
	{ value: "PreferNoSchedule" },
] as const;

const TOLERATION_OPERATORS = [{ value: "Equal" }, { value: "Exists" }] as const;

function kvMapToEntries(m: Record<string, string> | undefined): KVEntry[] {
	if (!m) return [];
	return Object.entries(m).map(([key, value]) => ({ key, value }));
}

function kvEntriesToMap(
	entries: KVEntry[] | undefined,
): Record<string, string> {
	const out: Record<string, string> = {};
	for (const e of entries ?? []) {
		const k = e.key?.trim();
		if (!k) continue;
		out[k] = (e.value ?? "").trim();
	}
	return out;
}

export default function PoolManager() {
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [quotas, setQuotas] = useState<Record<string, QuotaInfo>>({});
	const [elasticQuotas, setElasticQuotas] = useState<ElasticQuota[]>([]);
	const [clusters, setClusters] = useState<Cluster[]>([]);
	const [loading, setLoading] = useState(false);
	const [modalOpen, setModalOpen] = useState(false);
	const [saving, setSaving] = useState(false);
	const [editTarget, setEditTarget] = useState<ExecutionTarget | null>(null);
	// EQs for the cluster currently selected IN THE MODAL (may differ from the
	// default-cluster `elasticQuotas` the page pre-fetches). Feeds the "Koord
	// 资源池" picker and the namespace suggestions.
	const [modalEqs, setModalEqs] = useState<ElasticQuota[]>([]);
	const [modalEqsLoading, setModalEqsLoading] = useState(false);
	// Node-constraint section (tolerations / nodeSelector) is advanced; collapsed
	// by default so the common path stays short.
	const [showAdvanced, setShowAdvanced] = useState(false);
	const [form] = Form.useForm<FormValues>();
	const watchedClusterId = Form.useWatch("clusterId", form);
	const watchedNamespace = Form.useWatch("namespace", form);
	const watchedScheduler = Form.useWatch("schedulerName", form);
	const watchedEqName = Form.useWatch("elasticQuotaName", form);

	const fetchData = useCallback(async () => {
		setLoading(true);
		try {
			const [t, q, eq, cs] = await Promise.all([
				listExecutionTargets(),
				fetch("/api/v1/resource-quotas").then((r) => r.json()),
				listElasticQuotas().catch(() => [] as ElasticQuota[]),
				listClusters().catch(() => [] as Cluster[]),
			]);
			setTargets(t);
			setQuotas(q.items || {});
			setElasticQuotas(eq);
			setClusters(cs);
		} catch {
			/* ignore */
		}
		setLoading(false);
	}, []);

	useEffect(() => {
		fetchData();
		// Poll every 60s (koord-scheduler status loop is 60s upstream; a matching
		// cadence is enough — 15s just spammed backend access logs). Pause the
		// timer when the tab is hidden so an abandoned tab doesn't keep polling
		// forever, and refetch on re-focus so the panel isn't stale.
		let timer: ReturnType<typeof setInterval> | null = null;
		const start = () => {
			timer ??= setInterval(fetchData, 60000);
		};
		const stop = () => {
			if (timer) {
				clearInterval(timer);
				timer = null;
			}
		};
		const onVisibilityChange = () => {
			if (document.hidden) {
				stop();
			} else {
				fetchData();
				start();
			}
		};
		if (!document.hidden) {
			start();
		}
		document.addEventListener("visibilitychange", onVisibilityChange);
		return () => {
			stop();
			document.removeEventListener("visibilitychange", onVisibilityChange);
		};
	}, [fetchData]);

	// Fetch ElasticQuotas for the cluster chosen in the modal so the "Koord 资源池"
	// picker and namespace suggestions reflect the RIGHT cluster (the page-level
	// `elasticQuotas` only ever holds the default cluster's). Re-runs when the
	// modal opens or the cluster picker changes.
	useEffect(() => {
		if (!modalOpen) return;
		let cancelled = false;
		setModalEqsLoading(true);
		listElasticQuotas(watchedClusterId)
			.then((eqs) => {
				if (!cancelled) setModalEqs(eqs);
			})
			.catch(() => {
				if (!cancelled) setModalEqs([]);
			})
			.finally(() => {
				if (!cancelled) setModalEqsLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [modalOpen, watchedClusterId]);

	// Namespace suggestions (zero new backend API — CYB-3486): the namespaces the
	// selected cluster's EQs live in, unioned with namespaces already used by
	// pools on that cluster. The field stays free-text, so an unlisted namespace
	// is still typable.
	const namespaceOptions = useMemo(() => {
		const set = new Set<string>();
		for (const eq of modalEqs) {
			if (eq.namespace) set.add(eq.namespace);
		}
		const clusterKeyVal = watchedClusterId ?? "";
		for (const t of targets) {
			if ((t.clusterId ?? "") === clusterKeyVal && t.namespace) {
				set.add(t.namespace);
			}
		}
		return Array.from(set)
			.sort()
			.map((ns) => ({ value: ns }));
	}, [modalEqs, targets, watchedClusterId]);

	// EQ picker options, scoped to the pool's namespace when one is set (koord EQs
	// are namespace-scoped; a pod may only use an EQ in its own namespace). Before
	// a namespace is chosen, show all of the cluster's EQs.
	const eqOptions = useMemo(() => {
		const ns = (watchedNamespace ?? "").trim();
		const list = ns ? modalEqs.filter((eq) => eq.namespace === ns) : modalEqs;
		return list.map((eq) => ({
			value: eq.name,
			label: `${eq.name}　—　${poolUsageSummary(eq)}`,
		}));
	}, [modalEqs, watchedNamespace]);

	const selectedEq = useMemo(
		() => modalEqs.find((eq) => eq.name === watchedEqName),
		[modalEqs, watchedEqName],
	);

	const openCreate = () => {
		setEditTarget(null);
		form.resetFields();
		setShowAdvanced(false);
		setModalEqs([]);
		setModalOpen(true);
	};

	const openEdit = (target: ExecutionTarget) => {
		setEditTarget(target);
		// Prefer FK clusterId; fall back to matching legacy cluster string by name.
		const clusterId =
			target.clusterId ??
			clusters.find((c) => c.name === target.cluster)?.id ??
			clusters.find((c) => c.isDefault)?.id;
		const scheduling = target.resourceDefaults?.scheduling ?? {};
		// Split the koord EQ label out of podLabels so the "Koord 资源池" picker
		// owns it and the manual podLabels editor shows only the OTHER labels.
		const allLabels = scheduling.podLabels ?? {};
		const eqName = allLabels[KOORD_EQ_LABEL_KEY] ?? "";
		const otherLabels = { ...allLabels };
		delete otherLabels[KOORD_EQ_LABEL_KEY];
		// Auto-expand the node-constraint section when the pool actually uses it.
		setShowAdvanced(
			(target.resourceDefaults?.templateTolerations?.length ?? 0) > 0 ||
				Object.keys(target.resourceDefaults?.templateNodeSelector ?? {})
					.length > 0,
		);
		form.setFieldsValue({
			name: target.name,
			namespace: target.namespace,
			clusterId,
			description: target.description,
			templateTolerations: target.resourceDefaults?.templateTolerations ?? [],
			templateNodeSelector: kvMapToEntries(
				target.resourceDefaults?.templateNodeSelector,
			),
			schedulerName: scheduling.schedulerName ?? "",
			priorityClassName: scheduling.priorityClassName ?? "",
			elasticQuotaName: eqName,
			podLabels: kvMapToEntries(otherLabels),
			podAnnotations: kvMapToEntries(scheduling.podAnnotations),
		});
		setModalOpen(true);
	};

	const handleSave = async () => {
		let values: FormValues;
		try {
			values = await form.validateFields();
		} catch {
			// antd shows the field-level error; do not toast a generic message.
			return;
		}
		// Normalize tolerations: drop empty rows, trim strings, drop `value`
		// when operator=Exists so the body matches server expectation.
		const tolerations: TargetToleration[] = (values.templateTolerations ?? [])
			.map((t) => ({
				key: (t.key ?? "").trim(),
				operator: t.operator ?? "Equal",
				value: t.operator === "Exists" ? undefined : (t.value ?? "").trim(),
				effect: t.effect ?? "NoSchedule",
			}))
			.filter((t) => t.key.length > 0);
		const nodeSelector = kvEntriesToMap(values.templateNodeSelector);
		const podLabels = kvEntriesToMap(values.podLabels);
		const podAnnotations = kvEntriesToMap(values.podAnnotations);
		const schedulerName = (values.schedulerName ?? "").trim();
		const priorityClassName = (values.priorityClassName ?? "").trim();
		// Fold the "Koord 资源池" picker into the standard EQ pod-label. The picker
		// is meaningful only under koord-scheduler, so bind the label to it: a
		// non-koord scheduler drops any EQ label (dead weight other schedulers
		// ignore), and the picker owns this key either way — a stale value left in
		// the manual editor never wins.
		const eqName = (values.elasticQuotaName ?? "").trim();
		if (schedulerName === "koord-scheduler" && eqName) {
			podLabels[KOORD_EQ_LABEL_KEY] = eqName;
		} else {
			delete podLabels[KOORD_EQ_LABEL_KEY];
		}

		// Assemble the scheduling sub-object (CYB-3486 pool.3). Start from
		// whatever the server already had under `.scheduling` so keys this editor
		// doesn't manage (a future scheduling knob, or nodeSelector/tolerations if
		// a target stored them nested) survive the PUT. A blank input deletes its
		// key so clearing a field genuinely unsets it rather than writing "".
		const scheduling: TargetScheduling = {
			...(editTarget?.resourceDefaults?.scheduling ?? {}),
		};
		if (schedulerName) scheduling.schedulerName = schedulerName;
		else delete scheduling.schedulerName;
		if (priorityClassName) scheduling.priorityClassName = priorityClassName;
		else delete scheduling.priorityClassName;
		if (Object.keys(podLabels).length > 0) scheduling.podLabels = podLabels;
		else delete scheduling.podLabels;
		if (Object.keys(podAnnotations).length > 0) {
			scheduling.podAnnotations = podAnnotations;
		} else {
			delete scheduling.podAnnotations;
		}

		// Preserve every field the server sent us (terminal config, computeTier,
		// custom keys, etc.) so a PUT only touches what the user edited.
		const preservedDefaults: TargetResourceDefaultsPatch = {
			...(editTarget?.resourceDefaults ?? {}),
			templateTolerations: tolerations,
			templateNodeSelector:
				Object.keys(nodeSelector).length > 0 ? nodeSelector : undefined,
			scheduling: Object.keys(scheduling).length > 0 ? scheduling : undefined,
		};
		if (preservedDefaults.templateNodeSelector === undefined) {
			delete preservedDefaults.templateNodeSelector;
		}
		if (preservedDefaults.scheduling === undefined) {
			delete preservedDefaults.scheduling;
		}

		// Backend still requires the legacy `cluster` string field; look it up
		// from the selected clusterId (falling back to "default" if the picker
		// list is empty for any reason).
		const selectedCluster = clusters.find((c) => c.id === values.clusterId);
		const body: Partial<ExecutionTarget> = {
			name: values.name.trim(),
			namespace: values.namespace.trim(),
			clusterId: values.clusterId,
			cluster: selectedCluster?.name ?? "default",
			description: values.description?.trim(),
			status: "available",
			resourceDefaults: preservedDefaults,
		};

		setSaving(true);
		try {
			if (editTarget) {
				await updateExecutionTarget(editTarget.id, {
					...editTarget,
					...body,
				});
				message.success("已更新");
			} else {
				await createExecutionTarget(body);
				message.success("已创建");
			}
			setModalOpen(false);
			await fetchData();
		} catch (err) {
			const msg = err instanceof Error ? err.message : "操作失败";
			message.error(msg);
		} finally {
			setSaving(false);
		}
	};

	const handleDelete = (target: ExecutionTarget) => {
		Modal.confirm({
			title: "删除资源池",
			content: `确认删除「${target.name}」？此操作不可撤销。`,
			okText: "删除",
			okType: "danger",
			onOk: async () => {
				try {
					await deleteExecutionTarget(target.id);
					message.success("已删除");
					await fetchData();
				} catch (err) {
					const msg = err instanceof Error ? err.message : "删除失败";
					message.error(msg);
				}
			},
		});
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
			// CYB-3486 ux.1: 集群列淡化为二级信息。 用户看资源池时的心智
			// 应该是"池的容量 / 池的调度",cluster 是管理员维度的实现细节。
			// 保留列不删,但字号 / 颜色降到与"默认"标签同层次的辅助信息。
			title: (
				<Text type="secondary" style={{ fontSize: 11, fontWeight: "normal" }}>
					集群
				</Text>
			),
			key: "cluster",
			width: 140,
			render: (_: unknown, r: ExecutionTarget) => {
				// Prefer FK lookup; fall back to legacy string.
				const cluster = clusters.find(
					(c) => c.id === r.clusterId || c.name === r.cluster,
				);
				const label = cluster?.displayName || cluster?.name || r.cluster;
				return (
					<Text type="secondary" style={{ fontSize: 11 }}>
						{label}
						{cluster?.koordInstalled ? (
							<Tag color="green" style={{ marginLeft: 6, fontSize: 10 }}>
								Koord
							</Tag>
						) : null}
					</Text>
				);
			},
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
			title: "调度配置",
			key: "scheduling",
			width: 260,
			render: (_: unknown, r: ExecutionTarget) => {
				const rd = r.resourceDefaults;
				const sched = rd?.scheduling ?? {};
				const tols = rd?.templateTolerations ?? [];
				const sel = rd?.templateNodeSelector ?? {};
				const podLabels = sched.podLabels ?? {};
				const schedulerName = sched.schedulerName?.trim();
				const priorityClassName = sched.priorityClassName?.trim();
				if (
					tols.length === 0 &&
					Object.keys(sel).length === 0 &&
					Object.keys(podLabels).length === 0 &&
					!schedulerName &&
					!priorityClassName
				) {
					return <Text type="secondary">—</Text>;
				}
				return (
					<Space size={2} wrap>
						{schedulerName ? (
							<Tag color="purple" style={{ fontSize: 11 }}>
								sched: {schedulerName}
							</Tag>
						) : null}
						{priorityClassName ? (
							<Tag color="gold" style={{ fontSize: 11 }}>
								prio: {priorityClassName}
							</Tag>
						) : null}
						{Object.entries(podLabels).map(([k, v]) => (
							<Tag color="magenta" key={`pl-${k}`} style={{ fontSize: 11 }}>
								{k}={v}
							</Tag>
						))}
						{tols.map((t) => (
							<Tag
								key={`tol-${t.key}-${t.value ?? ""}-${t.effect}`}
								style={{ fontSize: 11 }}
							>
								{t.key}
								{t.operator === "Equal" && t.value ? `=${t.value}` : ""}
							</Tag>
						))}
						{Object.entries(sel).map(([k, v]) => (
							<Tag color="cyan" key={`sel-${k}`} style={{ fontSize: 11 }}>
								{k}={v}
							</Tag>
						))}
					</Space>
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
								aria-label={`edit-${r.name}`}
							/>
						</Tooltip>
						<Tooltip title="删除">
							<Button
								type="text"
								size="small"
								danger
								icon={<DeleteOutlined />}
								onClick={() => handleDelete(r)}
								aria-label={`delete-${r.name}`}
							/>
						</Tooltip>
					</Space>
				),
		},
	];

	return (
		<>
			<ClusterManager />
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
						。调度约束(toleration / nodeSelector)会随每个 Argo Workflow
						下发,与集群节点污点必须匹配才能调度。
					</Typography.Text>
				</div>
			</Card>

			{elasticQuotas.length > 0 || clusters.length > 1 ? (
				<ElasticQuotaPanel
					initialQuotas={elasticQuotas}
					clusters={clusters}
					loading={loading}
				/>
			) : null}

			<Modal
				title={editTarget ? "编辑资源池" : "新建资源池"}
				open={modalOpen}
				onOk={handleSave}
				onCancel={() => setModalOpen(false)}
				confirmLoading={saving}
				okText={editTarget ? "保存" : "创建"}
				width={720}
				destroyOnClose
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
						name="clusterId"
						label="集群"
						initialValue={
							clusters.find((c) => c.isDefault)?.id ?? clusters[0]?.id
						}
						rules={[{ required: true, message: "请选择集群" }]}
					>
						<Select
							placeholder="选择集群"
							options={clusters.map((c) => ({
								value: c.id,
								label: (
									<span>
										{c.displayName || c.name}
										{c.koordInstalled ? (
											<Tag
												color="green"
												style={{ marginLeft: 6, fontSize: 10 }}
											>
												Koord
											</Tag>
										) : null}
									</span>
								),
							}))}
						/>
					</Form.Item>
					<Form.Item
						name="namespace"
						label="K8s 命名空间"
						rules={[{ required: true, message: "请选择或输入命名空间" }]}
						extra="可从集群已知命名空间中选,也可直接输入。"
					>
						<AutoComplete
							options={namespaceOptions}
							placeholder="例如: cyber-delivery-prod"
							filterOption={(input, option) =>
								(option?.value ?? "")
									.toLowerCase()
									.includes(input.toLowerCase())
							}
						/>
					</Form.Item>
					<Form.Item name="description" label="描述">
						<Input.TextArea rows={2} placeholder="可选,一句话说明用途" />
					</Form.Item>

					<Divider style={{ margin: "12px 0" }} orientation="left" plain>
						<Text type="secondary" style={{ fontSize: 12 }}>
							调度配置(可选)
						</Text>
					</Divider>

					<Typography.Paragraph
						type="secondary"
						style={{ fontSize: 12, marginBottom: 12 }}
					>
						留空 → 使用集群默认调度器(适配任意调度器,不接入弹性配额)。 如需接入
						Koordinator 弹性配额:调度器填 <Text code>koord-scheduler</Text>
						,下方会出现该集群的资源池供选择。
					</Typography.Paragraph>

					<Form.Item name="schedulerName" label="调度器">
						<Input placeholder="留空 = 集群默认调度器;例如 koord-scheduler" />
					</Form.Item>

					{watchedScheduler?.trim() === "koord-scheduler" ? (
						<Form.Item
							name="elasticQuotaName"
							label="Koord 资源池(选填)"
							extra="从所选集群实时拉取;选中后自动写入 EQ pod label,无需手填。"
						>
							<Select
								allowClear
								showSearch
								loading={modalEqsLoading}
								placeholder={
									eqOptions.length > 0
										? "选择一个 ElasticQuota"
										: "该集群/命名空间未发现 ElasticQuota"
								}
								options={eqOptions}
								notFoundContent={
									modalEqsLoading ? "加载中…" : "无可用 ElasticQuota"
								}
								onChange={(val) => {
									// koord EQs are namespace-scoped; align the pool's
									// namespace to the chosen quota so the pod lands where
									// the EQ actually lives.
									const eq = modalEqs.find((e) => e.name === val);
									if (eq?.namespace && eq.namespace !== watchedNamespace) {
										form.setFieldValue("namespace", eq.namespace);
										message.info(
											`已将命名空间设为 ${eq.namespace}(匹配所选资源池)`,
										);
									}
								}}
							/>
						</Form.Item>
					) : null}
					{watchedScheduler?.trim() === "koord-scheduler" && selectedEq ? (
						<div style={{ marginTop: -8, marginBottom: 16 }}>
							<Text type="secondary" style={{ fontSize: 12 }}>
								<Space size="large">
									<span>
										min {selectedEq.min.cpu} / {selectedEq.min.memory}
									</span>
									<span>
										max {selectedEq.max.cpu} / {selectedEq.max.memory}
									</span>
									<span>
										used {selectedEq.used.cpu} / {selectedEq.used.memory}
									</span>
								</Space>
							</Text>
						</div>
					) : null}

					<Form.Item name="priorityClassName" label="PriorityClass">
						<Input placeholder="留空 = K8s 全局默认;例如 cyber-databrew-prod" />
					</Form.Item>
					<Form.Item
						label="Pod labels"
						extra="额外的 pod 标签(EQ label 由上方资源池管理,无需在此重复)。"
					>
						<KeyValueListEditor
							name="podLabels"
							keyPlaceholder="label key"
							valuePlaceholder="label value"
							addLabel="添加 pod label"
							ariaPrefix="podlabel"
						/>
					</Form.Item>
					<Form.Item label="Pod annotations">
						<KeyValueListEditor
							name="podAnnotations"
							keyPlaceholder="annotation key"
							valuePlaceholder="annotation value"
							addLabel="添加 pod annotation"
							ariaPrefix="podannotation"
						/>
					</Form.Item>

					<Divider style={{ margin: "12px 0" }} orientation="left" plain>
						<Button
							type="link"
							size="small"
							style={{ padding: 0, fontSize: 12 }}
							onClick={() => setShowAdvanced((v) => !v)}
						>
							{showAdvanced ? "▾" : "▸"} 节点约束(可选)——
							与集群节点污点匹配才能调度
						</Button>
					</Divider>

					<div style={{ display: showAdvanced ? "block" : "none" }}>
						<Form.Item label="Tolerations(容忍污点)">
							<Form.List name="templateTolerations">
								{(fields, { add, remove }) => (
									<>
										{fields.map((field) => (
											<Space
												key={field.key}
												align="baseline"
												style={{ display: "flex", marginBottom: 4 }}
											>
												<Form.Item
													name={[field.name, "key"]}
													rules={[
														{
															required: true,
															message: "key 必填",
															whitespace: true,
														},
													]}
													style={{ marginBottom: 0, width: 180 }}
												>
													<Input placeholder="key(如 compute-tier)" />
												</Form.Item>
												<Form.Item
													name={[field.name, "operator"]}
													initialValue="Equal"
													style={{ marginBottom: 0, width: 100 }}
												>
													<Select options={[...TOLERATION_OPERATORS]} />
												</Form.Item>
												<Form.Item
													name={[field.name, "value"]}
													dependencies={[
														["templateTolerations", field.name, "operator"],
													]}
													rules={[
														({ getFieldValue }) => ({
															validator(_, val) {
																const op = getFieldValue([
																	"templateTolerations",
																	field.name,
																	"operator",
																]);
																if (
																	op === "Exists" ||
																	(val != null && String(val).trim() !== "")
																) {
																	return Promise.resolve();
																}
																return Promise.reject(
																	new Error("operator=Equal 时 value 必填"),
																);
															},
														}),
													]}
													style={{ marginBottom: 0, width: 180 }}
												>
													<Input placeholder="value(如 med)" />
												</Form.Item>
												<Form.Item
													name={[field.name, "effect"]}
													initialValue="NoSchedule"
													style={{ marginBottom: 0, width: 160 }}
												>
													<Select options={[...TOLERATION_EFFECTS]} />
												</Form.Item>
												<MinusCircleOutlined
													onClick={() => remove(field.name)}
													aria-label={`remove-toleration-${field.name}`}
												/>
											</Space>
										))}
										<Button
											type="dashed"
											size="small"
											onClick={() =>
												add({ operator: "Equal", effect: "NoSchedule" })
											}
											icon={<PlusOutlined />}
										>
											添加 toleration
										</Button>
									</>
								)}
							</Form.List>
						</Form.Item>

						<Form.Item label="Node Selector(节点选择)">
							<KeyValueListEditor
								name="templateNodeSelector"
								keyPlaceholder="label key"
								valuePlaceholder="label value"
								addLabel="添加 nodeSelector"
								ariaPrefix="nodeselector"
								requireValue
							/>
						</Form.Item>
					</div>
				</Form>
			</Modal>
		</>
	);
}

type TargetResourceDefaultsPatch = Record<string, unknown> & {
	templateTolerations?: TargetToleration[];
	templateNodeSelector?: Record<string, string>;
	scheduling?: TargetScheduling;
};

interface KeyValueListEditorProps {
	// Form.List field name in the enclosing Form (e.g. "podLabels").
	name: string;
	keyPlaceholder: string;
	valuePlaceholder: string;
	addLabel: string;
	// aria-label prefix for the remove icon → `remove-${ariaPrefix}-${index}`.
	ariaPrefix: string;
	// When true, both key and value are required (matches the nodeSelector
	// editor). When false (pod labels / annotations), blank rows are simply
	// dropped on save by kvEntriesToMap, so no hard validation is imposed.
	requireValue?: boolean;
}

// KeyValueListEditor is the shared key/value map editor used by nodeSelector,
// pod labels and pod annotations — a Form.List of {key, value} rows plus an
// "add" button. The parent converts entries to a map with kvEntriesToMap.
function KeyValueListEditor({
	name,
	keyPlaceholder,
	valuePlaceholder,
	addLabel,
	ariaPrefix,
	requireValue = false,
}: KeyValueListEditorProps) {
	const keyRules = requireValue
		? [{ required: true, message: "key 必填", whitespace: true }]
		: undefined;
	const valueRules = requireValue
		? [{ required: true, message: "value 必填", whitespace: true }]
		: undefined;
	return (
		<Form.List name={name}>
			{(fields, { add, remove }) => (
				<>
					{fields.map((field) => (
						<Space
							key={field.key}
							align="baseline"
							style={{ display: "flex", marginBottom: 4 }}
						>
							<Form.Item
								name={[field.name, "key"]}
								rules={keyRules}
								style={{ marginBottom: 0, width: 220 }}
							>
								<Input placeholder={keyPlaceholder} />
							</Form.Item>
							<Form.Item
								name={[field.name, "value"]}
								rules={valueRules}
								style={{ marginBottom: 0, width: 220 }}
							>
								<Input placeholder={valuePlaceholder} />
							</Form.Item>
							<MinusCircleOutlined
								onClick={() => remove(field.name)}
								aria-label={`remove-${ariaPrefix}-${field.name}`}
							/>
						</Space>
					))}
					<Button
						type="dashed"
						size="small"
						onClick={() => add({ key: "", value: "" })}
						icon={<PlusOutlined />}
					>
						{addLabel}
					</Button>
				</>
			)}
		</Form.List>
	);
}

interface ElasticQuotaPanelProps {
	// initialQuotas is the panel's first render (whatever PoolManager pre-fetched
	// on the default cluster). When the user switches cluster via the picker
	// below, the panel takes over and refetches with the chosen clusterId.
	initialQuotas: ElasticQuota[];
	clusters: Cluster[];
	loading: boolean;
}

// ElasticQuotaPanel renders a read-only view of Koordinator ElasticQuota pools.
// Shows a cluster picker when there is more than one cluster (CYB-3486) so
// operators can inspect quotas on any registered cluster — the underlying API
// routes by ?clusterId= (PR 4b).
function ElasticQuotaPanel({
	initialQuotas,
	clusters,
	loading,
}: ElasticQuotaPanelProps) {
	const defaultClusterId = "cluster-default";
	const [selectedClusterId, setSelectedClusterId] = useState(defaultClusterId);
	const [quotas, setQuotas] = useState<ElasticQuota[]>(initialQuotas);
	const [refetchLoading, setRefetchLoading] = useState(false);

	// Keep quotas in sync with the parent's pre-fetched value when the user has
	// not yet switched cluster (initialQuotas can arrive slightly after mount).
	useEffect(() => {
		if (selectedClusterId === defaultClusterId) {
			setQuotas(initialQuotas);
		}
	}, [initialQuotas, selectedClusterId]);

	const handleClusterChange = useCallback(async (clusterId: string) => {
		setSelectedClusterId(clusterId);
		setRefetchLoading(true);
		try {
			const list = await listElasticQuotas(clusterId);
			setQuotas(list);
		} catch {
			setQuotas([]);
		}
		setRefetchLoading(false);
	}, []);

	// CYB-3486 ux.1: "池" 面板按池组织 — 每一行是一个池,池主字段是名字,
	// 命名空间降为二级标签(mono/secondary),不再单独占列。 这样表格阅读
	// 顺序变成"池是谁 → 它多满 → 谁在用",跟用户的心智直接对齐。
	const columns = [
		{
			title: "池",
			key: "pool",
			render: (_: unknown, r: ElasticQuota) => (
				<Space direction="vertical" size={0}>
					<Text strong>{r.name}</Text>
					<Text
						type="secondary"
						style={{ fontFamily: "var(--font-mono)", fontSize: 11 }}
					>
						ns: {r.namespace}
					</Text>
				</Space>
			),
		},
		{
			title: "CPU (used / min / max)",
			key: "cpu",
			width: 260,
			render: (_: unknown, r: ElasticQuota) => (
				<QuotaBar
					used={r.used.cpu}
					min={r.min.cpu}
					max={r.max.cpu}
					percent={r.utilizationPercent.cpu}
				/>
			),
		},
		{
			title: "Memory (used / min / max)",
			key: "memory",
			width: 260,
			render: (_: unknown, r: ElasticQuota) => (
				<QuotaBar
					used={r.used.memory}
					min={r.min.memory}
					max={r.max.memory}
					percent={r.utilizationPercent.memory}
				/>
			),
		},
	];

	// Sort quotas by namespace so pools sharing a namespace visually cluster
	// together — cheap "grouping" without giving up the flat table.
	const sortedQuotas = [...quotas].sort((a, b) => {
		if (a.namespace !== b.namespace)
			return a.namespace.localeCompare(b.namespace);
		return a.name.localeCompare(b.name);
	});

	const showPicker = clusters.length > 1;

	return (
		<Card
			size="small"
			style={{ marginTop: 12 }}
			title={
				<Space size="middle" wrap>
					<span>Koordinator 弹性配额池 (ElasticQuota)</span>
					{showPicker ? (
						<Select
							size="small"
							value={selectedClusterId}
							onChange={handleClusterChange}
							loading={refetchLoading}
							style={{ minWidth: 220 }}
							options={clusters.map((c) => ({
								value: c.id,
								label: c.displayName || c.name,
							}))}
						/>
					) : null}
				</Space>
			}
		>
			<Table
				size="small"
				rowKey={(r) => `${r.namespace}/${r.name}`}
				dataSource={sortedQuotas}
				columns={columns}
				loading={loading || refetchLoading}
				pagination={false}
				locale={{
					emptyText: refetchLoading
						? "加载中…"
						: "该集群未配置 ElasticQuota(或 Koordinator 未安装)",
				}}
			/>
			<div style={{ marginTop: 12, color: "var(--gray-400)", fontSize: 12 }}>
				<Typography.Text type="secondary">
					资源池由 Koordinator 提供集群级弹性配额:空闲时可跨池借用,max
					为硬上限。 面板每 60 秒自动刷新(切走后暂停); 配额本身由 kubectl /
					GitOps 管理,不在此处编辑。
				</Typography.Text>
			</div>
		</Card>
	);
}

interface QuotaBarProps {
	used: string;
	min: string;
	max: string;
	percent: number;
}

function QuotaBar({ used, min, max, percent }: QuotaBarProps) {
	const pct = Math.max(0, Math.min(100, percent));
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
					minWidth: 110,
					textAlign: "right",
				}}
			>
				{used} / {min} / {max}
			</Text>
		</div>
	);
}
