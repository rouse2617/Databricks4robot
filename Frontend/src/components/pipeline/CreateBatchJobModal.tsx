import { Alert, Form, Input, Modal, Select, Typography } from "antd";
import { useMemo, useState } from "react";
import { createBatchJob, type BatchJob } from "../../api/batchJobApi";
import type { PipelineTemplate } from "../../api/pipelineApi";
import AssetPicker from "./AssetPicker";

const { TextArea } = Input;

interface CreateBatchJobModalProps {
	open: boolean;
	templates: PipelineTemplate[];
	onClose: () => void;
	onCreated: (job: BatchJob) => void;
}

function parseAssetIds(raw: string): string[] {
	return Array.from(
		new Set(
			raw
				.split(/[\n,;\s]+/)
				.map((item) => item.trim())
				.filter(Boolean),
		),
	);
}

export function CreateBatchJobModal({
	open,
	templates,
	onClose,
	onCreated,
}: CreateBatchJobModalProps) {
	const [form] = Form.useForm<{
		name: string;
		templateId: string;
		bulkAssetIds: string;
	}>();
	const [selectedIds, setSelectedIds] = useState<string[]>([]);
	const [submitting, setSubmitting] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const templateOptions = useMemo(
		() =>
			templates.map((item) => ({
				label: `${item.name} (v${item.version})`,
				value: item.id,
			})),
		[templates],
	);

	const reset = () => {
		form.resetFields();
		setSelectedIds([]);
		setError(null);
	};

	const handleSubmit = async () => {
		setError(null);
		try {
			const values = await form.validateFields();
			const bulkIds = parseAssetIds(values.bulkAssetIds ?? "");
			const assetIds = Array.from(new Set([...selectedIds, ...bulkIds]));
			if (assetIds.length === 0) {
				setError("请至少选择一个资产，或粘贴资产 ID 列表");
				return;
			}
			setSubmitting(true);
			const job = await createBatchJob({
				name: values.name.trim(),
				templateId: values.templateId,
				assetIds,
			});
			reset();
			onCreated(job);
		} catch (err) {
			if (err instanceof Error) {
				setError(err.message);
			}
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<Modal
			open={open}
			title="新建批次任务"
			okText="创建并启动"
			cancelText="取消"
			width={760}
			confirmLoading={submitting}
			destroyOnHidden
			onCancel={() => {
				reset();
				onClose();
			}}
			onOk={() => void handleSubmit()}
		>
			<Typography.Paragraph type="secondary">
				批次任务会为每个资产各创建一条流水线执行记录。列表页只展示批次汇总，点进详情才分页查看子任务。
			</Typography.Paragraph>

			{error ? (
				<Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />
			) : null}

			<Form form={form} layout="vertical">
				<Form.Item
					label="批次名称"
					name="name"
					rules={[{ required: true, message: "请输入批次名称" }]}
				>
					<Input placeholder="例如：2026-06 Echo 补跑" maxLength={120} />
				</Form.Item>
				<Form.Item
					label="流水线模板"
					name="templateId"
					rules={[{ required: true, message: "请选择模板" }]}
				>
					<Select
						showSearch
						placeholder="选择要批量执行的模板"
						options={templateOptions}
						optionFilterProp="label"
					/>
				</Form.Item>
				<Form.Item label="搜索并选择资产">
					<AssetPicker
						selectedIds={selectedIds}
						onSelectionChange={setSelectedIds}
						maxHeight={220}
						resetKey={open ? "open" : "closed"}
					/>
				</Form.Item>
				<Form.Item
					label="批量粘贴资产 ID"
					name="bulkAssetIds"
					extra="支持换行、逗号或空格分隔，适合一次导入上千条资产 ID"
				>
					<TextArea
						rows={5}
						placeholder={"asset-id-1\nasset-id-2\nasset-id-3"}
					/>
				</Form.Item>
			</Form>
		</Modal>
	);
}
