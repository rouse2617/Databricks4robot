import { useState } from "react";
import { Modal, Form, Input, message } from "antd";
import { deliveriesApi } from "../../api/deliveries";

export interface CreateDeliveryModalProps {
  open: boolean;
  assetIds: string[];
  onClose: () => void;
  onSuccess: (deliveryId: string) => void;
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

  const handleOk = async () => {
    if (assetIds.length === 0) {
      msg.error("请至少选择 1 个资产后再创建交付");
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
          asset_ids: assetIds.length > 0 ? assetIds : undefined,
        },
        idempotencyKey,
      );

      msg.success("交付创建成功");
      form.resetFields();
      onSuccess(delivery.delivery_id);
    } catch (err: any) {
      if (err?.response?.status === 409) {
        msg.error("该交付已存在（幂等冲突）");
      } else if (err?.errorFields) {
        // form validation error — do nothing
      } else {
        msg.error("创建交付失败");
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    form.resetFields();
    onClose();
  };

  return (
    <Modal
      title="新建交付"
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      confirmLoading={submitting}
      okButtonProps={{ disabled: assetIds.length === 0 }}
      okText="提交"
      cancelText="取消"
      destroyOnHidden
    >
      {msgCtx}
      {assetIds.length > 0 && (
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
      )}
      <Form form={form} layout="vertical">
        <Form.Item
          name="customer_id"
          label="客户 ID"
          rules={[{ required: true, message: "请输入客户 ID" }]}
        >
          <Input placeholder="请输入客户 ID" />
        </Form.Item>
        <Form.Item name="contract_id" label="合同号">
          <Input placeholder="可选" />
        </Form.Item>
        <Form.Item name="note" label="备注">
          <Input.TextArea rows={3} placeholder="可选" />
        </Form.Item>
        <Form.Item name="owner" label="Owner">
          <Input placeholder="可选" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
