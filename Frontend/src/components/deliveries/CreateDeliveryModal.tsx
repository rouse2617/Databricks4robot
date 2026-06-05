import { Alert, Form, Input, Modal, message } from "antd";
import { useState } from "react";
import { deliveriesApi } from "../../api/deliveries";
import { describeApiError } from "../../lib/apiError";

export interface CreateDeliveryModalProps {
	open: boolean;
	assetIds: string[];
	onClose: () => void;
	onSuccess: (deliveryId: string) => void;
}

interface SubmitErrorState {
	message: string;
	description?: string;
}

function isFormValidationError(err: unknown): boolean {
	return typeof err === "object" && err !== null && "errorFields" in err;
}

function detailValue(details: unknown, key: string): string | undefined {
	if (typeof details !== "object" || details === null) return undefined;
	const value = (details as Record<string, unknown>)[key];
	return typeof value === "string" ? value : undefined;
}

function getAssetIdsError(
	messageText: string,
	details: unknown,
): string | null {
	const assetId = detailValue(details, "asset_id");
	if (
		messageText.includes("asset_ids must be 8 alphanumeric characters") ||
		messageText.includes("asset_id must be 8 alphanumeric characters")
	) {
		return assetId
			? `资产 ID「${assetId}」格式不正确；请改成 8 位字母或数字。`
			: "资产 ID 格式不正确；请改成 8 位字母或数字。";
	}
	if (messageText.toLowerCase().includes("asset") && assetId) {
		return `请检查资产 ID「${assetId}」是否存在且可交付。`;
	}
	return null;
}

function buildSubmitError(err: unknown): {
	submitError: SubmitErrorState;
	assetIdsError: string | null;
} {
	const apiError = describeApiError(err, "创建交付失败");
	const assetIdsError = getAssetIdsError(apiError.message, apiError.details);
	const requestText = apiError.requestId
		? `请求 ID：${apiError.requestId}`
		: undefined;

	if (apiError.status === 409) {
		return {
			submitError: {
				message: "该交付已存在",
				description: requestText
					? `请刷新列表确认是否已创建。${requestText}`
					: "请刷新列表确认是否已创建。",
			},
			assetIdsError: null,
		};
	}

	return {
		submitError: {
			message: assetIdsError ?? apiError.message,
			description: requestText,
		},
		assetIdsError,
	};
}

export default function CreateDeliveryModal({
	open,
	assetIds,
	onClose,
	onSuccess,
}: CreateDeliveryModalProps) {
	const [form] = Form.useForm();
	const [submitting, setSubmitting] = useState(false);
	const [msg, msgCtx] = message.useMessage();
	const [manualAssetIdsText, setManualAssetIdsText] = useState("");
	const [submitError, setSubmitError] = useState<SubmitErrorState | null>(null);
	const [manualAssetIdsError, setManualAssetIdsError] = useState<string | null>(
		null,
	);

	const parsedManualAssetIds = manualAssetIdsText
		.split(/[\n,]/)
		.map((v) => v.trim())
		.filter((v) => v.length > 0);
	const dedupedManualAssetIds = Array.from(new Set(parsedManualAssetIds));
	const effectiveAssetIds =
		assetIds.length > 0 ? assetIds : dedupedManualAssetIds;

	const handleOk = async () => {
		setSubmitError(null);
		setManualAssetIdsError(null);
		if (effectiveAssetIds.length === 0) {
			const nextError = "请至少选择 1 个资产后再创建交付。";
			setManualAssetIdsError(nextError);
			setSubmitError({ message: nextError });
			msg.error(nextError);
			return;
		}
		try {
			const values = await form.validateFields();
			setSubmitting(true);

			const idempotencyKey = crypto.randomUUID();
			const delivery = await deliveriesApi.commit(
				{
					customer_id: values.customer_id,
					contract_id: values.contract_id || undefined,
					note: values.note || undefined,
					owner: values.owner || undefined,
					asset_ids: effectiveAssetIds,
				},
				idempotencyKey,
			);

			msg.success("交付创建成功");
			form.resetFields();
			setManualAssetIdsText("");
			setSubmitError(null);
			setManualAssetIdsError(null);
			onSuccess(delivery.delivery_id);
		} catch (err: unknown) {
			if (isFormValidationError(err)) {
				// form validation error — do nothing
			} else {
				const next = buildSubmitError(err);
				setSubmitError(next.submitError);
				setManualAssetIdsError(next.assetIdsError);
				msg.error(next.submitError.message);
			}
		} finally {
			setSubmitting(false);
		}
	};

	const handleCancel = () => {
		form.resetFields();
		setManualAssetIdsText("");
		setSubmitError(null);
		setManualAssetIdsError(null);
		onClose();
	};

	return (
		<Modal
			title="新建交付"
			open={open}
			onOk={handleOk}
			onCancel={handleCancel}
			confirmLoading={submitting}
			okButtonProps={{ disabled: effectiveAssetIds.length === 0 }}
			okText="提交"
			cancelText="取消"
			maskClosable={!submitting}
			keyboard={!submitting}
			destroyOnClose
			afterClose={() => {
				form.resetFields();
				setManualAssetIdsText("");
				setSubmitError(null);
				setManualAssetIdsError(null);
			}}
		>
			{msgCtx}
			{assetIds.length > 0 ? (
				<div
					style={{
						background: "#f6ffed",
						border: "1px solid #b7eb8f",
						borderRadius: 4,
						padding: "8px 12px",
						marginBottom: 16,
						fontSize: 13,
					}}
				>
					已选择 {assetIds.length} 个资产
				</div>
			) : (
				<div
					style={{
						background: "#fffbe6",
						border: "1px solid #ffe58f",
						borderRadius: 4,
						padding: "8px 12px",
						marginBottom: 16,
						fontSize: 13,
					}}
				>
					当前未选择资产，请手动填写 asset_ids（逗号或换行分隔）
				</div>
			)}
			{submitError && (
				<Alert
					type="error"
					showIcon
					style={{ marginBottom: 16 }}
					message={submitError.message}
					description={submitError.description}
				/>
			)}
			<Form form={form} layout="vertical">
				{assetIds.length === 0 && (
					<Form.Item
						label="资产 IDs（必填）"
						htmlFor="delivery-asset-ids"
						required
						validateStatus={manualAssetIdsError ? "error" : undefined}
						help={
							manualAssetIdsError ??
							"每行或逗号分隔；资产 ID 应为 8 位字母或数字。"
						}
					>
						<Input.TextArea
							id="delivery-asset-ids"
							name="asset_ids"
							rows={4}
							value={manualAssetIdsText}
							onChange={(e) => {
								setManualAssetIdsText(e.target.value);
								setManualAssetIdsError(null);
								setSubmitError(null);
							}}
							placeholder={"例如：\n7VBGimAO,Ab12Cd34\nXy98LmN0"}
						/>
					</Form.Item>
				)}
				<Form.Item
					name="customer_id"
					label="客户 ID"
					rules={[{ required: true, message: "请输入客户 ID" }]}
				>
					<Input
						id="delivery-customer-id"
						name="customer_id"
						placeholder="请输入客户 ID"
						onChange={() => setSubmitError(null)}
					/>
				</Form.Item>
				<Form.Item name="contract_id" label="合同号">
					<Input
						id="delivery-contract-id"
						name="contract_id"
						placeholder="可选"
					/>
				</Form.Item>
				<Form.Item name="note" label="备注">
					<Input.TextArea
						id="delivery-note"
						name="note"
						rows={3}
						placeholder="可选"
					/>
				</Form.Item>
				<Form.Item name="owner" label="Owner">
					<Input id="delivery-owner" name="owner" placeholder="可选" />
				</Form.Item>
			</Form>
		</Modal>
	);
}
