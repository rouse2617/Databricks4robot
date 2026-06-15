import { App, Button, Form, Input, Space, Typography } from "antd";
import { useState } from "react";
import { uploadBackfillResult } from "../../api/batchJobApi";

const { Text } = Typography;

export interface BackfillResultsUploadProps {
	assetId?: string;
	defaultReportId?: string;
	defaultVersion?: string;
	disabled?: boolean;
	onUploaded?: () => void;
}

export function BackfillResultsUpload({
	assetId,
	defaultReportId = "report.project@1.0.0-backfill-left-eye-only",
	defaultVersion = "1.0.0",
	disabled,
	onUploaded,
}: BackfillResultsUploadProps) {
	const { message } = App.useApp();
	const [loading, setLoading] = useState(false);
	const [form] = Form.useForm();

	return (
		<Form
			form={form}
			layout="vertical"
			initialValues={{
				assetId: assetId ?? "",
				reportId: defaultReportId,
				version: defaultVersion,
				resultJson: '{"status":"ok"}',
			}}
			onFinish={async (values) => {
				setLoading(true);
				try {
					let result: Record<string, unknown> = {};
					if (values.resultJson?.trim()) {
						result = JSON.parse(values.resultJson) as Record<string, unknown>;
					}
					await uploadBackfillResult({
						assetId: values.assetId,
						reportId: values.reportId,
						version: values.version,
						manifest: { assetId: values.assetId },
						result,
					});
					message.success("回填结果已注册");
					onUploaded?.();
				} catch (err) {
					message.error(`上传失败：${String(err)}`);
				} finally {
					setLoading(false);
				}
			}}
		>
			<Text type="secondary">
				工作流已完成但缺少 derivative 报告时，可手动注册算法
				JSON（awaiting_result → registered）。
			</Text>
			<Form.Item
				label="资产 ID"
				name="assetId"
				rules={[{ required: true, message: "请输入资产 ID" }]}
			>
				<Input disabled={disabled || Boolean(assetId)} />
			</Form.Item>
			<Form.Item
				label="Report ID"
				name="reportId"
				rules={[{ required: true, message: "请输入 reportId" }]}
			>
				<Input disabled={disabled} />
			</Form.Item>
			<Form.Item
				label="Version"
				name="version"
				rules={[{ required: true, message: "请输入 version" }]}
			>
				<Input disabled={disabled} />
			</Form.Item>
			<Form.Item label="Result JSON" name="resultJson">
				<Input.TextArea rows={4} disabled={disabled} />
			</Form.Item>
			<Space>
				<Button
					type="primary"
					htmlType="submit"
					loading={loading}
					disabled={disabled}
				>
					上传结果
				</Button>
			</Space>
		</Form>
	);
}
