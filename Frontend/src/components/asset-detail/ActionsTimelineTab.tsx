import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Divider,
	Empty,
	Form,
	Input,
	InputNumber,
	Modal,
	message,
	Select,
	Space,
	Spin,
	Switch,
	Tag,
	Timeline,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import {
	type Action,
	type ActionCreateInput,
	type ActionSourceType,
	actionsApi,
} from "../../api/actions";
import { registryApi } from "../../api/registry";
import { extractApiErrorMessage } from "../../lib/apiError";

const { Text } = Typography;

interface Props {
	assetId: string;
	/** When set and not `segment`, actions are N/A; list is empty and create is disabled. */
	assetType?: string;
	segStartNs?: number;
	segEndNs?: number;
}

const SOURCE_TYPES: ActionSourceType[] = ["human", "algo", "rule", "system"];

function nsToRelative(ns: number, segStartNs?: number): string {
	if (segStartNs === undefined || !Number.isFinite(segStartNs)) {
		return `${ns} ns`;
	}
	const offsetMs = (ns - segStartNs) / 1_000_000;
	return `${offsetMs.toFixed(1)} ms`;
}

export default function ActionsTimelineTab({
	assetId,
	assetType,
	segStartNs,
	segEndNs,
}: Props) {
	const [items, setItems] = useState<Action[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [filterLabel, setFilterLabel] = useState("");
	const [createOpen, setCreateOpen] = useState(false);
	const [creating, setCreating] = useState(false);
	const [allowCustomLabel, setAllowCustomLabel] = useState(false);
	const [primaryLabelOptions, setPrimaryLabelOptions] = useState<string[]>([]);
	const [labelOptions, setLabelOptions] = useState<string[]>([]);
	const [registryLoading, setRegistryLoading] = useState(false);
	const [form] = Form.useForm<
		ActionCreateInput & { labelsSelect?: string[]; labelsCsv?: string }
	>();
	const [msg, msgCtx] = message.useMessage();

	const isSegmentAsset =
		assetType === undefined || assetType === "" || assetType === "segment";

	const load = useCallback(
		(label?: string) => {
			setLoading(true);
			setError(null);
			actionsApi
				.list(assetId, label ? { label, limit: 200 } : { limit: 200 })
				.then((resp) => setItems(resp.items ?? []))
				.catch((err) =>
					setError(extractApiErrorMessage(err, "加载 action 失败")),
				)
				.finally(() => setLoading(false));
		},
		[assetId],
	);

	useEffect(() => {
		load();
	}, [load]);

	useEffect(() => {
		if (!createOpen) return;
		if (primaryLabelOptions.length > 0 || labelOptions.length > 0) return;
		setRegistryLoading(true);
		registryApi
			.getActionLabelRegistry()
			.then((data) => {
				setPrimaryLabelOptions(data.primary_labels ?? []);
				setLabelOptions(data.labels ?? []);
			})
			.catch((err) =>
				msg.error(extractApiErrorMessage(err, "加载 action label 注册表失败")),
			)
			.finally(() => setRegistryLoading(false));
	}, [createOpen, primaryLabelOptions.length, labelOptions.length, msg]);

	const onCreate = async () => {
		try {
			const values = await form.validateFields();
			setCreating(true);
			const labels = allowCustomLabel
				? (values.labelsCsv ?? "")
						.split(",")
						.map((s) => s.trim())
						.filter(Boolean)
				: (values.labelsSelect ?? []);
			const payload: ActionCreateInput = {
				start_ns: values.start_ns,
				end_ns: values.end_ns,
				primary_label: values.primary_label || undefined,
				labels,
				description: values.description || undefined,
				source_type: values.source_type ?? "human",
				source_name: values.source_name || undefined,
			};
			const created = await actionsApi.create(assetId, payload);
			msg.success("Action 创建成功");
			setCreateOpen(false);
			form.resetFields();
			// CYB-3292: show the new action immediately from the create response.
			// A plain re-list here races read-after-write and can drop it until a
			// manual refresh; the create response is authoritative for the new row.
			setItems((prev) => [
				created,
				...prev.filter((a) => a.action_id !== created.action_id),
			]);
		} catch (err) {
			// Form validation throws don't have a response body; surface API errors.
			if (err && typeof err === "object" && "response" in err) {
				msg.error(extractApiErrorMessage(err, "创建失败"));
			}
		} finally {
			setCreating(false);
		}
	};

	return (
		<Card
			size="small"
			title="Action 时间轴"
			extra={
				<Space size="small">
					<Input.Search
						allowClear
						size="small"
						placeholder="按 label 过滤"
						style={{ width: 180 }}
						onSearch={(v) => {
							setFilterLabel(v);
							load(v || undefined);
						}}
					/>
					<Button
						size="small"
						icon={<ReloadOutlined />}
						onClick={() => load(filterLabel || undefined)}
					>
						刷新
					</Button>
					<Button
						size="small"
						type="primary"
						icon={<PlusOutlined />}
						disabled={!isSegmentAsset}
						onClick={() => setCreateOpen(true)}
					>
						新建 Action
					</Button>
				</Space>
			}
		>
			{msgCtx}
			{!isSegmentAsset && (
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 12 }}
					message="Action 仅适用于 segment 类型资产"
					description={`当前资产类型为「${assetType ?? "—"}」。此处时间轴仅针对 segment；列表为空属预期。`}
				/>
			)}
			{loading && items.length === 0 ? (
				<div style={{ textAlign: "center", padding: "32px 0" }}>
					<Spin />
				</div>
			) : error ? (
				<Alert type="error" showIcon message={error} />
			) : items.length === 0 ? (
				<Empty
					description={
						isSegmentAsset
							? "该 seg 暂无 action"
							: "当前资产不是 segment，不包含 seg 内 Action"
					}
					image={Empty.PRESENTED_IMAGE_SIMPLE}
				/>
			) : (
				<Timeline
					items={items.map((a) => ({
						color:
							a.source_type === "human"
								? "blue"
								: a.source_type === "algo"
									? "green"
									: "gray",
						children: (
							<div>
								<div className="flex items-center gap-2 flex-wrap">
									{a.primary_label && <Tag color="blue">{a.primary_label}</Tag>}
									{a.labels
										.filter((l) => l !== a.primary_label)
										.map((l) => (
											<Tag key={l}>{l}</Tag>
										))}
									<Tag color="default">{a.source_type}</Tag>
									{a.source_name && (
										<Text type="secondary" className="text-xs">
											{a.source_name}
										</Text>
									)}
								</div>
								<div className="mt-1">
									<Text className="text-xs">
										[{nsToRelative(a.start_ns, segStartNs)} →{" "}
										{nsToRelative(a.end_ns, segStartNs)}]
									</Text>
								</div>
								{a.description && (
									<div>
										<Text type="secondary" className="text-xs">
											{a.description}
										</Text>
									</div>
								)}
								<div>
									<Text type="secondary" className="text-xs font-mono">
										{a.action_id}
									</Text>
								</div>
							</div>
						),
					}))}
				/>
			)}

			<Modal
				title="新建 Action"
				open={createOpen}
				onOk={onCreate}
				confirmLoading={creating}
				onCancel={() => {
					setCreateOpen(false);
					form.resetFields();
					setAllowCustomLabel(false);
				}}
				destroyOnHidden
			>
				<Form
					form={form}
					layout="vertical"
					initialValues={{
						source_type: "human",
						start_ns: segStartNs ?? 0,
						end_ns: segEndNs ?? 0,
					}}
				>
					<Form.Item
						label="start_ns"
						name="start_ns"
						rules={[{ required: true, message: "必填" }]}
					>
						<InputNumber style={{ width: "100%" }} step={1_000_000} />
					</Form.Item>
					<Form.Item
						label="end_ns"
						name="end_ns"
						rules={[{ required: true, message: "必填" }]}
					>
						<InputNumber style={{ width: "100%" }} step={1_000_000} />
					</Form.Item>
					<Divider style={{ margin: "8px 0 12px" }} />
					<div className="mb-2 flex items-center justify-between">
						<Text type="secondary">Label 选择模式</Text>
						<Space size={8}>
							<Text className="text-xs">允许自定义输入</Text>
							<Switch
								checked={allowCustomLabel}
								onChange={setAllowCustomLabel}
							/>
						</Space>
					</div>
					{allowCustomLabel ? (
						<>
							<Form.Item label="primary_label" name="primary_label">
								<Input placeholder="例如 pickup" />
							</Form.Item>
							<Form.Item label="labels (逗号分隔)" name="labelsCsv">
								<Input placeholder="pickup, left_hand" />
							</Form.Item>
						</>
					) : (
						<>
							<Form.Item label="primary_label" name="primary_label">
								<Select
									showSearch
									loading={registryLoading}
									allowClear
									placeholder="从注册表选择 primary_label"
									options={primaryLabelOptions.map((v) => ({
										value: v,
										label: v,
									}))}
									optionFilterProp="label"
								/>
							</Form.Item>
							<Form.Item label="labels" name="labelsSelect">
								<Select
									mode="multiple"
									showSearch
									loading={registryLoading}
									allowClear
									placeholder="从注册表选择 labels"
									options={labelOptions.map((v) => ({ value: v, label: v }))}
									optionFilterProp="label"
								/>
							</Form.Item>
						</>
					)}
					<Form.Item label="描述" name="description">
						<Input.TextArea rows={2} />
					</Form.Item>
					<Form.Item label="source_type" name="source_type">
						<Select
							options={SOURCE_TYPES.map((s) => ({ value: s, label: s }))}
						/>
					</Form.Item>
					<Form.Item label="source_name" name="source_name">
						<Input placeholder="annotator-001 或 algo 名" />
					</Form.Item>
				</Form>
			</Modal>
		</Card>
	);
}
