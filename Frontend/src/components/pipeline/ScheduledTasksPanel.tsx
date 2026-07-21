// Scheduled Tasks panel (CYB-3744) — the 「定时任务」 tab under the Pipeline
// page. A user can list, create, edit, pause/resume, run-now, or delete rules.
// Each rule pulls asset-ids from a configured REST source on its own interval
// and creates a batch on a pipeline template; produced batches show up in
// 执行记录 → 批量任务.
//
// v1 keeps the form deliberately declarative: the top of the drawer is the
// friendly common case (name/pipeline/target/interval), while the two
// free-form JSON textareas (source config + trigger config) let power users
// express anything the backend supports without waiting for a form for every
// knob. `sourceConfig` is pre-populated with a Grace-shaped template so new
// rules are one-edit-away from working.
import {
	DeleteOutlined,
	EditOutlined,
	PauseCircleOutlined,
	PlayCircleOutlined,
	PlusOutlined,
	ReloadOutlined,
	ThunderboltOutlined,
} from "@ant-design/icons";
import {
	Button,
	Card,
	Drawer,
	Form,
	Input,
	InputNumber,
	message,
	Popconfirm,
	Select,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
	createScheduledTask,
	deleteScheduledTask,
	listScheduledTasks,
	pauseScheduledTask,
	resumeScheduledTask,
	runScheduledTaskNow,
	type ScheduledTask,
	type ScheduledTaskCreateRequest,
	type ScheduledTaskRestSource,
	type ScheduledTaskTriggerConfig,
	type ScheduledTaskTriggerMode,
	updateScheduledTask,
} from "../../api/scheduledTaskApi";

const { Text, Paragraph } = Typography;

// A grace-shaped starter so new rules render a working REST source template
// in the drawer. Users only need to edit the base_url / step_key / auth.
const DEFAULT_REST_SOURCE_TEMPLATE: ScheduledTaskRestSource = {
	base_url: "https://grace.example.com/api",
	path: "/grace/video_steps",
	auth: {
		type: "basic",
		username: "grace-service",
		secret_ref: "GRACE_PASSWORD", // pragma: allowlist secret
	},
	query: {
		static: { filter: ["step_key:eq:body_heatmap", "status:eq:success"] },
		filter_param: "filter",
		time_field: "last_status_at",
	},
	paging: {
		mode: "page_size",
		page_size: 200,
		total_path: "total",
		data_path: "data",
	},
	id_path: "data[].video_id",
};

const DEFAULT_TRIGGER_CONFIG: Record<
	ScheduledTaskTriggerMode,
	ScheduledTaskTriggerConfig
> = {
	incremental: { intervalSeconds: 3600, initialLookbackSeconds: 3600 },
	rolling: { intervalSeconds: 3600, lookbackSeconds: 1800 },
	range: {},
	ids: {},
};

const TRIGGER_MODE_LABELS: Record<ScheduledTaskTriggerMode, string> = {
	incremental: "增量 (水位线)",
	rolling: "滚动窗口",
	range: "固定范围",
	ids: "指定资产",
};

interface RuleFormValues {
	name: string;
	templateId: string;
	targetId: string;
	triggerMode: ScheduledTaskTriggerMode;
	intervalSeconds?: number;
	sourceConfigJSON: string;
	triggerConfigJSON: string;
}

function toJSON(value: unknown): string {
	try {
		return JSON.stringify(value ?? {}, null, 2);
	} catch {
		return "{}";
	}
}

function parseJSONField<T>(raw: string, fieldLabel: string): T {
	// JSON.parse accepts primitives ("null" / "123" / "\"str\"") without
	// throwing, but the caller expects an object — reject non-objects up front
	// so downstream `config.someField` accesses can't crash the UI.
	try {
		const parsed = JSON.parse(raw);
		if (
			parsed === null ||
			typeof parsed !== "object" ||
			Array.isArray(parsed)
		) {
			throw new Error("必须是一个 JSON 对象");
		}
		return parsed as T;
	} catch (err) {
		throw new Error(`${fieldLabel} 不是合法 JSON: ${(err as Error).message}`);
	}
}

// jsonObjectValidator is a Form.Item `rules[].validator` that runs
// parseJSONField semantics inline (empty → pass; primitives / arrays →
// reject), so users see the JSON error next to the textarea while typing
// rather than only via a global message on save.
function jsonObjectValidator(fieldLabel: string) {
	return (_rule: unknown, value: string) => {
		if (!value) return Promise.resolve();
		try {
			const parsed = JSON.parse(value);
			if (
				parsed === null ||
				typeof parsed !== "object" ||
				Array.isArray(parsed)
			) {
				return Promise.reject(new Error(`${fieldLabel} 必须是一个 JSON 对象`));
			}
			return Promise.resolve();
		} catch (err) {
			return Promise.reject(
				new Error(`${fieldLabel} 不是合法 JSON: ${(err as Error).message}`),
			);
		}
	};
}

function StatusTag({ rule }: { rule: ScheduledTask }) {
	if (!rule.enabled) return <Tag color="default">已暂停</Tag>;
	const s = rule.lastRunStatus;
	if (s === "failed") return <Tag color="red">上次失败</Tag>;
	if (s === "empty") return <Tag color="orange">上次无数据</Tag>;
	if (s === "succeeded") return <Tag color="green">运行中</Tag>;
	return <Tag color="blue">已启用</Tag>;
}

function formatWhen(ts?: string | null): string {
	if (!ts) return "—";
	const d = new Date(ts);
	if (Number.isNaN(d.getTime())) return ts;
	return d.toLocaleString();
}

export function ScheduledTasksPanel() {
	const [rules, setRules] = useState<ScheduledTask[]>([]);
	const [total, setTotal] = useState(0);
	const [loading, setLoading] = useState(false);
	const [q, setQ] = useState("");
	const [drawerOpen, setDrawerOpen] = useState(false);
	const [editing, setEditing] = useState<ScheduledTask | null>(null);
	const [saving, setSaving] = useState(false);
	const [form] = Form.useForm<RuleFormValues>();

	const load = useCallback(async () => {
		setLoading(true);
		try {
			const resp = await listScheduledTasks({
				q: q.trim() || undefined,
				pageSize: 100,
			});
			setRules(resp.items);
			setTotal(resp.total);
		} catch (err) {
			message.error(err instanceof Error ? err.message : "加载定时任务失败");
		}
		setLoading(false);
	}, [q]);

	useEffect(() => {
		void load();
	}, [load]);

	const openCreate = useCallback(() => {
		setEditing(null);
		form.resetFields();
		form.setFieldsValue({
			triggerMode: "incremental",
			intervalSeconds: 3600,
			sourceConfigJSON: toJSON(DEFAULT_REST_SOURCE_TEMPLATE),
			triggerConfigJSON: toJSON(DEFAULT_TRIGGER_CONFIG.incremental),
		});
		setDrawerOpen(true);
	}, [form]);

	const openEdit = useCallback(
		(rule: ScheduledTask) => {
			setEditing(rule);
			form.setFieldsValue({
				name: rule.name,
				templateId: rule.templateId,
				targetId: rule.targetId,
				triggerMode: rule.triggerMode,
				intervalSeconds: rule.triggerConfig?.intervalSeconds,
				sourceConfigJSON: toJSON(
					rule.sourceConfig ?? DEFAULT_REST_SOURCE_TEMPLATE,
				),
				triggerConfigJSON: toJSON(rule.triggerConfig ?? {}),
			});
			setDrawerOpen(true);
		},
		[form],
	);

	const onSave = useCallback(async () => {
		let values: RuleFormValues;
		try {
			values = await form.validateFields();
		} catch {
			return; // antd renders inline errors
		}
		let sourceConfig: ScheduledTaskRestSource;
		let triggerConfig: ScheduledTaskTriggerConfig;
		try {
			sourceConfig = parseJSONField<ScheduledTaskRestSource>(
				values.sourceConfigJSON,
				"数据源配置",
			);
			triggerConfig = parseJSONField<ScheduledTaskTriggerConfig>(
				values.triggerConfigJSON,
				"触发配置",
			);
		} catch (err) {
			message.error((err as Error).message);
			return;
		}
		// Convenience: sync the top-level "interval" field into triggerConfig
		// (so users who only touched the friendly control don't have to also
		// edit the JSON) — the JSON always wins if both are set.
		if (
			values.intervalSeconds != null &&
			triggerConfig.intervalSeconds == null
		) {
			triggerConfig.intervalSeconds = values.intervalSeconds;
		}
		const body: ScheduledTaskCreateRequest = {
			name: values.name.trim(),
			enabled: editing ? editing.enabled : true,
			templateId: values.templateId.trim(),
			targetId: values.targetId.trim(),
			sourceType: "rest",
			sourceConfig,
			triggerMode: values.triggerMode,
			triggerConfig,
		};
		setSaving(true);
		try {
			if (editing) {
				await updateScheduledTask(editing.id, body);
				message.success("已更新");
			} else {
				await createScheduledTask(body);
				message.success("已创建");
			}
			setDrawerOpen(false);
			await load();
		} catch (err) {
			message.error(err instanceof Error ? err.message : "保存失败");
		}
		setSaving(false);
	}, [editing, form, load]);

	const onToggle = useCallback(
		async (rule: ScheduledTask) => {
			try {
				if (rule.enabled) {
					await pauseScheduledTask(rule.id);
					message.success("已暂停");
				} else {
					await resumeScheduledTask(rule.id);
					message.success("已启用");
				}
				await load();
			} catch (err) {
				message.error(err instanceof Error ? err.message : "操作失败");
			}
		},
		[load],
	);

	const onRunNow = useCallback(async (rule: ScheduledTask) => {
		try {
			await runScheduledTaskNow(rule.id);
			message.success("已请求立即运行,下次调度周期生效");
		} catch (err) {
			message.error(err instanceof Error ? err.message : "触发失败");
		}
	}, []);

	const onDelete = useCallback(
		async (rule: ScheduledTask) => {
			try {
				await deleteScheduledTask(rule.id);
				message.success("已删除");
				await load();
			} catch (err) {
				message.error(err instanceof Error ? err.message : "删除失败");
			}
		},
		[load],
	);

	const onTriggerModeChange = useCallback(
		(mode: ScheduledTaskTriggerMode) => {
			// Replace triggerConfig with the new mode's default ONLY when the
			// user hasn't customized it (matches any mode's default template, or
			// is empty). Reading `triggerMode` from the form here would read the
			// NEW value (antd updates the store before onChange fires), which
			// was the earlier bug — instead we compare the JSON textarea itself
			// against every mode's default and preserve custom edits.
			const current = form.getFieldValue("triggerConfigJSON") as string;
			const isUntouchedDefault =
				!current ||
				Object.values(DEFAULT_TRIGGER_CONFIG).some(
					(cfg) => toJSON(cfg) === current,
				);
			if (isUntouchedDefault) {
				form.setFieldsValue({
					triggerConfigJSON: toJSON(DEFAULT_TRIGGER_CONFIG[mode]),
				});
			}
		},
		[form],
	);

	const columns: ColumnsType<ScheduledTask> = useMemo(
		() => [
			{
				title: "名称",
				dataIndex: "name",
				render: (name: string, rule) => (
					<Space direction="vertical" size={0}>
						<Text strong>{name}</Text>
						<Text
							type="secondary"
							copyable={{ text: rule.id }}
							style={{ fontSize: 12 }}
						>
							{rule.id}
						</Text>
					</Space>
				),
			},
			{ title: "流水线", dataIndex: "templateId", width: 200 },
			{ title: "资源池", dataIndex: "targetId", width: 180 },
			{
				title: "触发模式",
				dataIndex: "triggerMode",
				width: 140,
				render: (mode: ScheduledTaskTriggerMode) =>
					TRIGGER_MODE_LABELS[mode] ?? mode,
			},
			{
				title: "频率",
				width: 100,
				render: (_v, rule) => {
					const s = rule.triggerConfig?.intervalSeconds;
					if (!s) return "—";
					if (s % 3600 === 0) return `每 ${s / 3600}h`;
					if (s % 60 === 0) return `每 ${s / 60}m`;
					return `每 ${s}s`;
				},
			},
			{
				title: "状态",
				width: 110,
				render: (_v, rule) => <StatusTag rule={rule} />,
			},
			{
				title: "上次运行",
				width: 190,
				render: (_v, rule) => (
					<Space direction="vertical" size={0}>
						<Text style={{ fontSize: 12 }}>{formatWhen(rule.lastRunAt)}</Text>
						{rule.lastBatchId ? (
							<Text type="secondary" style={{ fontSize: 11 }}>
								batch: {rule.lastBatchId.slice(0, 18)}…
							</Text>
						) : null}
						{rule.lastError ? (
							<Tooltip title={rule.lastError}>
								<Text type="danger" style={{ fontSize: 11 }} ellipsis>
									{rule.lastError.slice(0, 40)}
								</Text>
							</Tooltip>
						) : null}
					</Space>
				),
			},
			{
				title: "操作",
				width: 240,
				fixed: "right",
				render: (_v, rule) => (
					<Space size={4} wrap>
						<Tooltip title="立即运行">
							<Button
								size="small"
								icon={<ThunderboltOutlined />}
								onClick={() => void onRunNow(rule)}
							>
								立即
							</Button>
						</Tooltip>
						<Button
							size="small"
							icon={
								rule.enabled ? <PauseCircleOutlined /> : <PlayCircleOutlined />
							}
							onClick={() => void onToggle(rule)}
						>
							{rule.enabled ? "暂停" : "启用"}
						</Button>
						<Button
							size="small"
							icon={<EditOutlined />}
							onClick={() => openEdit(rule)}
						>
							编辑
						</Button>
						<Popconfirm
							title={`确认删除规则「${rule.name}」?`}
							okText="删除"
							okType="danger"
							cancelText="取消"
							onConfirm={() => void onDelete(rule)}
						>
							<Button size="small" danger icon={<DeleteOutlined />}>
								删除
							</Button>
						</Popconfirm>
					</Space>
				),
			},
		],
		[onDelete, onRunNow, onToggle, openEdit],
	);

	return (
		<Card
			title={
				<Space size="middle">
					<span>定时任务</span>
					<Text type="secondary" style={{ fontSize: 12 }}>
						共 {total} 条 · 自动下发规则(替代外部 grace-sync)
					</Text>
				</Space>
			}
			extra={
				<Space>
					<Input.Search
						placeholder="搜索名称 / ID"
						allowClear
						value={q}
						onChange={(e) => setQ(e.target.value)}
						onSearch={() => void load()}
						style={{ width: 240 }}
					/>
					<Button icon={<ReloadOutlined />} onClick={() => void load()} />
					<Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
						新建规则
					</Button>
				</Space>
			}
		>
			<Table
				rowKey="id"
				loading={loading}
				columns={columns}
				dataSource={rules}
				pagination={false}
				scroll={{ x: 1200 }}
				size="middle"
			/>

			<Drawer
				title={editing ? `编辑规则 · ${editing.name}` : "新建定时任务规则"}
				open={drawerOpen}
				onClose={() => setDrawerOpen(false)}
				width={720}
				destroyOnClose
				extra={
					<Space>
						<Button onClick={() => setDrawerOpen(false)}>取消</Button>
						<Button
							type="primary"
							loading={saving}
							onClick={() => void onSave()}
						>
							保存
						</Button>
					</Space>
				}
			>
				<Form form={form} layout="vertical" preserve={false}>
					<Form.Item
						name="name"
						label="规则名称"
						rules={[{ required: true, message: "必填" }]}
					>
						<Input placeholder="e.g. grace-sea-v2-hourly" />
					</Form.Item>
					<Space size="middle" style={{ display: "flex" }}>
						<Form.Item
							name="templateId"
							label="流水线 (template id)"
							rules={[{ required: true, message: "必填" }]}
							style={{ flex: 1 }}
						>
							<Input placeholder="tpl-..." />
						</Form.Item>
						<Form.Item
							name="targetId"
							label="资源池 (target id)"
							rules={[{ required: true, message: "必填" }]}
							style={{ flex: 1 }}
						>
							<Input placeholder="cluster-default / video-proc-prod / ..." />
						</Form.Item>
					</Space>
					<Space size="middle" style={{ display: "flex" }}>
						<Form.Item
							name="triggerMode"
							label="触发模式"
							rules={[{ required: true }]}
							style={{ flex: 1 }}
						>
							<Select
								options={Object.entries(TRIGGER_MODE_LABELS).map(([v, l]) => ({
									value: v,
									label: l,
								}))}
								onChange={(v) =>
									onTriggerModeChange(v as ScheduledTaskTriggerMode)
								}
							/>
						</Form.Item>
						<Form.Item
							name="intervalSeconds"
							label="频率 (秒)"
							tooltip="incremental/rolling 用;range/ids 忽略"
							style={{ flex: 1 }}
						>
							<InputNumber
								min={30}
								step={60}
								style={{ width: "100%" }}
								placeholder="3600 = 1h"
							/>
						</Form.Item>
					</Space>

					<Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
						数据源配置 (REST) — 描述如何从上游取 asset id。凭证只填{" "}
						<code>secret_ref</code>(环境变量名), 密文由 Cloud Run{" "}
						<code>--set-secrets</code> 从 Secret Manager 挂载,前端不接触明文。
					</Paragraph>
					<Form.Item
						name="sourceConfigJSON"
						rules={[
							{ required: true, message: "必填" },
							{ validator: jsonObjectValidator("数据源配置") },
						]}
					>
						<Input.TextArea
							autoSize={{ minRows: 10, maxRows: 20 }}
							spellCheck={false}
						/>
					</Form.Item>

					<Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
						触发配置 — 按模式使用不同字段(参见 API 文档)。
					</Paragraph>
					<Form.Item
						name="triggerConfigJSON"
						rules={[
							{ required: true, message: "必填" },
							{ validator: jsonObjectValidator("触发配置") },
						]}
					>
						<Input.TextArea
							autoSize={{ minRows: 5, maxRows: 12 }}
							spellCheck={false}
						/>
					</Form.Item>
				</Form>
			</Drawer>
		</Card>
	);
}
