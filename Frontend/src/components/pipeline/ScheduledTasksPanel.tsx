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
	MoreOutlined,
	PauseCircleOutlined,
	PlayCircleOutlined,
	PlusOutlined,
	ReloadOutlined,
	ThunderboltOutlined,
} from "@ant-design/icons";
import {
	App,
	Button,
	Card,
	Drawer,
	Dropdown,
	Form,
	Input,
	InputNumber,
	message,
	Select,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
	type ExecutionTarget,
	listExecutionTargets,
	listPipelines,
	type PipelineTemplate,
} from "../../api/pipelineApi";
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

// Filled Badge-style tags (colored bg + border) read as first-class status
// pills on white rather than thin outlines.
const STATUS_TAG_STYLE: React.CSSProperties = {
	borderRadius: 4,
	fontWeight: 500,
	padding: "0 8px",
	margin: 0,
};
function StatusTag({ rule }: { rule: ScheduledTask }) {
	if (!rule.enabled) {
		return (
			<Tag
				style={{
					...STATUS_TAG_STYLE,
					background: "#f5f5f5",
					color: "#595959",
					borderColor: "#d9d9d9",
				}}
			>
				已暂停
			</Tag>
		);
	}
	const s = rule.lastRunStatus;
	if (s === "failed") {
		return (
			<Tag
				style={{
					...STATUS_TAG_STYLE,
					background: "#fff1f0",
					color: "#a8071a",
					borderColor: "#ffa39e",
				}}
			>
				上次失败
			</Tag>
		);
	}
	if (s === "empty") {
		return (
			<Tag
				style={{
					...STATUS_TAG_STYLE,
					background: "#fffbe6",
					color: "#874d00",
					borderColor: "#ffe58f",
				}}
			>
				上次无数据
			</Tag>
		);
	}
	if (s === "succeeded") {
		return (
			<Tag
				style={{
					...STATUS_TAG_STYLE,
					background: "#f6ffed",
					color: "#237804",
					borderColor: "#b7eb8f",
				}}
			>
				运行中
			</Tag>
		);
	}
	return (
		<Tag
			style={{
				...STATUS_TAG_STYLE,
				background: "#e6f4ff",
				color: "#0958d9",
				borderColor: "#91caff",
			}}
		>
			已启用
		</Tag>
	);
}

function formatWhen(ts?: string | null): string {
	if (!ts) return "—";
	const d = new Date(ts);
	if (Number.isNaN(d.getTime())) return ts;
	return d.toLocaleString();
}

// monoStyle keeps ids / timestamps aligned across rows without lifting a full
// design system change; scoped inline so it's easy to spot when styling grows.
const monoStyle: React.CSSProperties = {
	fontFamily:
		'ui-monospace, SFMono-Regular, Menlo, Consolas, "Roboto Mono", monospace',
};

export function ScheduledTasksPanel() {
	const { modal } = App.useApp();
	const [rules, setRules] = useState<ScheduledTask[]>([]);
	const [total, setTotal] = useState(0);
	const [loading, setLoading] = useState(false);
	const [q, setQ] = useState("");
	// Templates + targets are loaded once, cached, and reused for both the
	// picker Selects in the drawer and the id→name display in the table so
	// users don't stare at raw UUIDs.
	const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
	const [targets, setTargets] = useState<ExecutionTarget[]>([]);
	const [selectsLoading, setSelectsLoading] = useState(false);
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

	// One-shot load of templates + targets. The lists are small (dozens);
	// loading them eagerly keeps the drawer instant and lets the table render
	// human names in place of raw UUIDs.
	useEffect(() => {
		let cancelled = false;
		setSelectsLoading(true);
		// Backend caps page_size at 200 and — importantly — FALLS BACK to
		// pageSize=20 when the request exceeds the cap (handlers/pagination.go:16),
		// so passing 500 silently returned only 20 rows. Use pageSize=200 and
		// paginate through the tail until we've collected `total` rows so the
		// Select can offer every template, not just the first page.
		const fetchAllPipelines = async () => {
			const collected: PipelineTemplate[] = [];
			let page = 1;
			const pageSize = 200;
			// Bounded loop: if total is somehow lying, cap at 10 pages (2000
			// templates) so a runaway backend can't spin the tab forever.
			for (let i = 0; i < 10; i++) {
				const resp = await listPipelines({
					page,
					pageSize,
					excludeAutoDrafts: true,
				});
				collected.push(...resp.items);
				if (collected.length >= resp.total || resp.items.length < pageSize)
					break;
				page += 1;
			}
			return collected;
		};
		Promise.all([fetchAllPipelines(), listExecutionTargets()])
			.then(([tmpls, tgts]) => {
				if (cancelled) return;
				setTemplates(tmpls);
				setTargets(tgts);
			})
			.catch((err) => {
				if (!cancelled) {
					message.error(
						err instanceof Error ? err.message : "加载模板/资源池选项失败",
					);
				}
			})
			.finally(() => {
				if (!cancelled) setSelectsLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, []);

	// id → display "name (id-prefix…)" for pretty-printing in the table.
	const templateNameByID = useMemo(() => {
		const m = new Map<string, string>();
		for (const t of templates) m.set(t.id, t.name);
		return m;
	}, [templates]);
	const targetNameByID = useMemo(() => {
		const m = new Map<string, string>();
		for (const t of targets) m.set(t.id, t.name);
		return m;
	}, [targets]);

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

	const confirmDelete = useCallback(
		(rule: ScheduledTask) => {
			modal.confirm({
				title: `确认删除规则「${rule.name}」?`,
				content: "删除后此规则不再触发定时下发,已产生的批量任务不受影响。",
				okText: "删除",
				okType: "danger",
				cancelText: "取消",
				onOk: () => onDelete(rule),
			});
		},
		[modal, onDelete],
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
			// Name column is bounded (was consuming all remaining width and pushing
			// everything else into a squeezed sliver on the right). Id below the
			// name is single-line ellipsized with a hover tooltip carrying the
			// full id + a copy button — no more UUID line-wrap.
			{
				title: "名称",
				dataIndex: "name",
				width: 240,
				render: (name: string, rule) => (
					<Space direction="vertical" size={2} style={{ maxWidth: 220 }}>
						<Text
							strong
							ellipsis={{ tooltip: name }}
							style={{ display: "block", maxWidth: 220 }}
						>
							{name}
						</Text>
						<Text
							type="secondary"
							copyable={{ text: rule.id, tooltips: ["复制", "已复制"] }}
							ellipsis={{ tooltip: rule.id }}
							style={{
								...monoStyle,
								fontSize: 11,
								display: "block",
								maxWidth: 220,
							}}
						>
							{rule.id}
						</Text>
					</Space>
				),
			},
			// Pipeline / target: prefer the human name; the id is hidden behind a
			// small "复制" affordance instead of a second visible line, so long
			// names / long UUIDs stay on one row and columns don't fight for space.
			{
				title: "流水线",
				dataIndex: "templateId",
				width: 220,
				render: (id: string) => {
					const name = templateNameByID.get(id);
					return (
						<Space size={4}>
							<Text
								ellipsis={{ tooltip: name ?? id }}
								style={{ maxWidth: 170 }}
							>
								{name ?? id}
							</Text>
							<Text
								type="secondary"
								copyable={{ text: id, tooltips: ["复制 id", "已复制"] }}
								style={{ fontSize: 0 }}
							/>
						</Space>
					);
				},
			},
			{
				title: "资源池",
				dataIndex: "targetId",
				width: 160,
				render: (id: string) => {
					const name = targetNameByID.get(id);
					return (
						<Space size={4}>
							<Text
								ellipsis={{ tooltip: name ?? id }}
								style={{ maxWidth: 110 }}
							>
								{name ?? id}
							</Text>
							<Text
								type="secondary"
								copyable={{ text: id, tooltips: ["复制 id", "已复制"] }}
								style={{ fontSize: 0 }}
							/>
						</Space>
					);
				},
			},
			{
				title: "触发模式",
				dataIndex: "triggerMode",
				width: 130,
				render: (mode: ScheduledTaskTriggerMode) =>
					TRIGGER_MODE_LABELS[mode] ?? mode,
			},
			{
				title: "频率",
				width: 90,
				render: (_v, rule) => {
					const s = rule.triggerConfig?.intervalSeconds;
					if (!s) return "—";
					if (s % 3600 === 0) return `每 ${s / 3600}h`;
					if (s % 60 === 0) return `每 ${s / 60}m`;
					return `每 ${s}s`;
				},
			},
			// State tag now uses filled bg for readability on white; see StatusTag.
			{
				title: "状态",
				width: 110,
				render: (_v, rule) => <StatusTag rule={rule} />,
			},
			{
				title: "上次运行",
				width: 240,
				render: (_v, rule) => (
					<Space direction="vertical" size={0}>
						<Text style={{ ...monoStyle, fontSize: 12 }}>
							{formatWhen(rule.lastRunAt)}
						</Text>
						{rule.lastBatchId ? (
							<Link
								to={`/pipeline/batch/${encodeURIComponent(rule.lastBatchId)}`}
								style={{ fontSize: 11 }}
							>
								批量 → {rule.lastBatchId.slice(0, 12)}…
							</Link>
						) : null}
						{rule.lastError ? (
							<Tooltip title={rule.lastError}>
								<Text
									type="danger"
									ellipsis
									style={{ fontSize: 11, display: "block", maxWidth: 220 }}
								>
									{rule.lastError.slice(0, 60)}
								</Text>
							</Tooltip>
						) : null}
					</Space>
				),
			},
			{
				title: "上次成功",
				width: 160,
				render: (_v, rule) =>
					rule.lastSuccessAt ? (
						<Text style={{ ...monoStyle, fontSize: 12 }}>
							{formatWhen(rule.lastSuccessAt)}
						</Text>
					) : (
						<Text type="secondary" style={{ fontSize: 12 }}>
							—
						</Text>
					),
			},
			// Operations: 高频外露 (编辑 · 立即),低频/危险收入更多菜单
			// (启用/暂停 + 删除). Dropdown items handle their own guards
			// (danger label, confirm modal for delete).
			{
				title: "操作",
				width: 150,
				fixed: "right",
				render: (_v, rule) => {
					const menuItems = [
						{
							key: "toggle",
							icon: rule.enabled ? (
								<PauseCircleOutlined />
							) : (
								<PlayCircleOutlined />
							),
							label: rule.enabled ? "暂停" : "启用",
							onClick: () => void onToggle(rule),
						},
						{ type: "divider" as const },
						{
							key: "delete",
							icon: <DeleteOutlined />,
							label: "删除",
							danger: true,
							onClick: () => confirmDelete(rule),
						},
					];
					return (
						<Space size={4}>
							<Button
								size="small"
								icon={<EditOutlined />}
								onClick={() => openEdit(rule)}
							>
								编辑
							</Button>
							<Tooltip title="立即运行">
								<Button
									size="small"
									type="primary"
									ghost
									icon={<ThunderboltOutlined />}
									onClick={() => void onRunNow(rule)}
								>
									立即
								</Button>
							</Tooltip>
							<Dropdown menu={{ items: menuItems }} trigger={["click"]}>
								<Button size="small" icon={<MoreOutlined />} />
							</Dropdown>
						</Space>
					);
				},
			},
		],
		[
			confirmDelete,
			onRunNow,
			onToggle,
			openEdit,
			templateNameByID,
			targetNameByID,
		],
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
				scroll={{ x: 1500 }}
				tableLayout="fixed"
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
							label="流水线"
							rules={[{ required: true, message: "必选" }]}
							style={{ flex: 1 }}
						>
							<Select
								showSearch
								loading={selectsLoading}
								placeholder="从流水线管理里选一个模板"
								optionFilterProp="label"
								options={templates.map((t) => ({
									value: t.id,
									label: `${t.name}${
										t.activeVersion ? " · v" + t.activeVersion : ""
									}`,
								}))}
							/>
						</Form.Item>
						<Form.Item
							name="targetId"
							label="资源池"
							rules={[{ required: true, message: "必选" }]}
							style={{ flex: 1 }}
						>
							<Select
								showSearch
								loading={selectsLoading}
								placeholder="选一个执行目标"
								optionFilterProp="label"
								options={targets.map((t) => ({
									value: t.id,
									label: `${t.name}${t.namespace ? " · " + t.namespace : ""}`,
								}))}
							/>
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
