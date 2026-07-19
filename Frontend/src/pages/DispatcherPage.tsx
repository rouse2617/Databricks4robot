import {
	Alert,
	Button,
	Card,
	Form,
	InputNumber,
	Modal,
	message,
	Popconfirm,
	Space,
	Switch,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import { type DispatcherClusterStatus, dispatcherApi } from "../api/dispatcher";
import { extractApiErrorMessage } from "../lib/apiError";
import {
	DISPATCHER_LIMITS,
	suppressionChip,
	validateDispatcherForm,
} from "../lib/dispatcher";

// CYB-3679 — dispatcher 在线调参:每集群 配置 vs 生效 + 压制原因 + 编辑/暂停。

const REFRESH_MS = 15_000;

export default function DispatcherPage() {
	const [rows, setRows] = useState<DispatcherClusterStatus[]>([]);
	const [loading, setLoading] = useState(false);
	const [editing, setEditing] = useState<DispatcherClusterStatus | null>(null);
	const [form] = Form.useForm();

	const load = useCallback(async () => {
		setLoading(true);
		try {
			setRows(await dispatcherApi.list());
		} catch (err) {
			message.error(extractApiErrorMessage(err, "加载调度配置失败"));
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		void load();
		const t = setInterval(() => void load(), REFRESH_MS);
		return () => clearInterval(t);
	}, [load]);

	const save = async (
		cluster: string,
		values: {
			max_concurrency: number;
			submit_batch: number;
			rate_per_sec: number;
			paused: boolean;
		},
	) => {
		const errs = validateDispatcherForm(values);
		if (errs.length > 0) {
			message.error(errs.join(";"));
			return;
		}
		try {
			await dispatcherApi.save(cluster, values);
			message.success(`已保存 ${cluster},下个调度周期(≤15s)生效`);
			setEditing(null);
			await load();
		} catch (err) {
			message.error(extractApiErrorMessage(err, "保存失败"));
		}
	};

	const togglePause = async (row: DispatcherClusterStatus, paused: boolean) => {
		await save(row.cluster_id, {
			max_concurrency: row.config.max_concurrency,
			submit_batch: row.config.submit_batch,
			rate_per_sec: row.config.rate_per_sec,
			paused,
		});
	};

	const columns = [
		{
			title: "集群",
			dataIndex: "cluster_id",
			key: "cluster_id",
			render: (id: string, row: DispatcherClusterStatus) => (
				<Space>
					<Typography.Text strong>{id}</Typography.Text>
					{!row.has_row && <Tag>默认值</Tag>}
				</Space>
			),
		},
		{
			title: "并发(生效 / 配置)",
			key: "concurrency",
			render: (_: unknown, row: DispatcherClusterStatus) => {
				const chip = suppressionChip(row.suppression_reason);
				return (
					<Space>
						<Typography.Text>
							{row.effective_concurrency} / {row.config.max_concurrency}
						</Typography.Text>
						{chip && (
							<Tooltip title="生效值低于配置值的原因">
								<Tag color={chip.color}>{chip.label}</Tag>
							</Tooltip>
						)}
					</Space>
				);
			},
		},
		{
			title: "单轮批量",
			key: "batch",
			render: (_: unknown, row: DispatcherClusterStatus) =>
				row.config.submit_batch,
		},
		{
			title: "速率(个/秒)",
			key: "rate",
			render: (_: unknown, row: DispatcherClusterStatus) =>
				row.config.rate_per_sec,
		},
		{
			title: "暂停下发",
			key: "paused",
			render: (_: unknown, row: DispatcherClusterStatus) => (
				<Switch
					checked={row.config.paused}
					onChange={(v) => void togglePause(row, v)}
				/>
			),
		},
		{
			title: "操作",
			key: "actions",
			render: (_: unknown, row: DispatcherClusterStatus) => (
				<Space>
					<Button
						size="small"
						onClick={() => {
							setEditing(row);
							form.setFieldsValue({
								max_concurrency: row.config.max_concurrency,
								submit_batch: row.config.submit_batch,
								rate_per_sec: row.config.rate_per_sec,
							});
						}}
					>
						编辑
					</Button>
					{row.has_row && (
						<Popconfirm
							title={`删除 ${row.cluster_id} 的自定义配置,恢复默认?`}
							onConfirm={async () => {
								try {
									await dispatcherApi.remove(row.cluster_id);
									message.success("已恢复默认");
									await load();
								} catch (err) {
									message.error(extractApiErrorMessage(err, "删除失败"));
								}
							}}
						>
							<Button size="small" danger>
								恢复默认
							</Button>
						</Popconfirm>
					)}
				</Space>
			),
		},
	];

	return (
		<Card title="调度器在线调参" loading={loading && rows.length === 0}>
			<Alert
				style={{ marginBottom: 16 }}
				type="info"
				showIcon
				message="修改在下个调度周期(≤15s)生效,无需重新部署。生效并发低于配置值表示背压(AIMD)正在保护集群。"
			/>
			<Table
				rowKey="cluster_id"
				dataSource={rows}
				columns={columns}
				pagination={false}
				size="middle"
			/>
			<Modal
				open={editing !== null}
				title={`编辑 ${editing?.cluster_id ?? ""}`}
				onCancel={() => setEditing(null)}
				onOk={() => {
					const v = form.getFieldsValue();
					if (editing) {
						void save(editing.cluster_id, {
							...v,
							paused: editing.config.paused,
						});
					}
				}}
			>
				<Form form={form} layout="vertical">
					<Form.Item
						name="max_concurrency"
						label={`并发上限(${DISPATCHER_LIMITS.maxConcurrency.min}–${DISPATCHER_LIMITS.maxConcurrency.max})`}
					>
						<InputNumber
							min={DISPATCHER_LIMITS.maxConcurrency.min}
							max={DISPATCHER_LIMITS.maxConcurrency.max}
							style={{ width: "100%" }}
						/>
					</Form.Item>
					<Form.Item
						name="submit_batch"
						label={`单轮批量(${DISPATCHER_LIMITS.submitBatch.min}–${DISPATCHER_LIMITS.submitBatch.max})`}
					>
						<InputNumber
							min={DISPATCHER_LIMITS.submitBatch.min}
							max={DISPATCHER_LIMITS.submitBatch.max}
							style={{ width: "100%" }}
						/>
					</Form.Item>
					<Form.Item
						name="rate_per_sec"
						label={`下发速率 个/秒(${DISPATCHER_LIMITS.ratePerSec.min}–${DISPATCHER_LIMITS.ratePerSec.max})`}
					>
						<InputNumber
							min={DISPATCHER_LIMITS.ratePerSec.min}
							max={DISPATCHER_LIMITS.ratePerSec.max}
							step={0.5}
							style={{ width: "100%" }}
						/>
					</Form.Item>
				</Form>
			</Modal>
		</Card>
	);
}
