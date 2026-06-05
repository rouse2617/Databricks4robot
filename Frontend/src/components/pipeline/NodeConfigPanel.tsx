import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { Node } from "@ant-design/pro-flow";
import { Button, Form, Input, Modal, Select } from "antd";
import { useEffect } from "react";
import { resourceQuantityRule } from "../../lib/pipelineResourceValidation";
import type { Argument, PipelineNodeData } from "./types";

interface NodeConfigPanelProps {
	open: boolean;
	node: Node<PipelineNodeData>;
	onCancel: () => void;
	onSave: (id: string, data: Partial<PipelineNodeData>) => void;
}

type EnvFormItem = { name?: string; value?: string };

type FormValues = {
	label: string;
	command?: string[];
	source?: string;
	args: string[];
	env?: EnvFormItem[];
	cpu: string;
	memory: string;
	disk: string;
};

function normalizeArgs(args: Argument[] | undefined): string[] {
	if (!args || args.length === 0) return [];
	return args
		.map((item) => item.value ?? item.name)
		.filter((value): value is string => Boolean(value));
}

function envToPairs(items: Argument[] | undefined): EnvFormItem[] {
	if (!items || items.length === 0) return [];
	return items.map((item) => ({
		name: item.name || "",
		value: item.value || "",
	}));
}

function formPairsToEnv(items: EnvFormItem[]): Argument[] {
	if (!items || items.length === 0) return [];
	return items
		.filter((item) => item?.name?.trim())
		.map((item) => ({
			name: item.name?.trim() || "",
			value: item.value || "",
		}));
}

export function NodeConfigPanel({
	open,
	node,
	onCancel,
	onSave,
}: NodeConfigPanelProps) {
	const [form] = Form.useForm<FormValues>();

	const componentType = String(node.data.type || "").toLowerCase();
	const isScriptType = componentType === "script";

	useEffect(() => {
		if (!open) return;
		form.setFieldsValue({
			label: node.data.label || "",
			source: node.data.source || "",
			command: node.data.command || [],
			args: normalizeArgs(node.data.args),
			env: envToPairs(node.data.env),
			cpu: node.data.cpu || "",
			memory: node.data.memory || "",
			disk: node.data.disk || "",
		});
	}, [form, node.data, open]);

	const handleSubmit = async () => {
		const values = await form.validateFields();
		const nextData: Partial<PipelineNodeData> = {
			label: values.label,
			source: isScriptType ? values.source : node.data.source || "",
			command: values.command || [],
			args: (values.args || []).map((value) => ({
				name: value,
				value,
			})),
			env: formPairsToEnv(values.env || []),
			cpu: values.cpu || "",
			memory: values.memory || "",
			disk: values.disk || "",
		};

		onSave(node.id, nextData);
		onCancel();
	};

	return (
		<Modal
			title="节点配置"
			open={open}
			onCancel={onCancel}
			width={560}
			footer={
				<div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
					<Button onClick={onCancel}>取消</Button>
					<Button type="primary" onClick={() => form.submit()}>
						保存
					</Button>
				</div>
			}
		>
			<Form form={form} layout="vertical" onFinish={handleSubmit}>
				<Form.Item label="名称" name="label">
					<Input id="node-config-label" name="label" placeholder="节点名称" />
				</Form.Item>
				{isScriptType && (
					<Form.Item label="source" name="source">
						<Input.TextArea
							id="node-config-source"
							name="source"
							rows={3}
							placeholder="Python 脚本 source"
						/>
					</Form.Item>
				)}
				<Form.Item label="命令" name="command">
					<Select
						id="node-config-command"
						mode="tags"
						options={[]}
						placeholder="按 Enter 添加命令片段"
					/>
				</Form.Item>
				<Form.Item label="参数" name="args">
					<Select
						id="node-config-args"
						mode="tags"
						options={[]}
						placeholder="按 Enter 添加参数"
					/>
				</Form.Item>
				<div
					style={{
						fontSize: 12,
						color: "#64748b",
						marginBottom: 16,
						lineHeight: 1.5,
					}}
				>
					输出组件需写入 /tmp/outputs/output，否则 Argo 会将节点标记为失败
				</div>
				<Form.Item label="环境变量">
					<Form.List name="env">
						{(fields, { add, remove }) => (
							<div style={{ display: "grid", gap: 8 }}>
								{fields.map((field) => (
									<div
										key={field.key}
										style={{
											display: "grid",
											gridTemplateColumns: "1fr 1fr auto",
											gap: 8,
										}}
									>
										<Form.Item
											{...field}
											name={[field.name, "name"]}
											rules={[{ required: true, message: "请输入环境变量名" }]}
											noStyle
										>
											<Input
												id={`node-config-env-name-${field.key}`}
												name={`env-${field.key}-name`}
												placeholder="KEY"
											/>
										</Form.Item>
										<Form.Item {...field} name={[field.name, "value"]} noStyle>
											<Input
												id={`node-config-env-value-${field.key}`}
												name={`env-${field.key}-value`}
												placeholder="VALUE"
											/>
										</Form.Item>
										<Button
											type="text"
											icon={<MinusCircleOutlined />}
											onClick={() => remove(field.name)}
											danger
										>
											移除
										</Button>
									</div>
								))}
								<Button
									type="dashed"
									icon={<PlusOutlined />}
									onClick={() => add()}
								>
									新增环境变量
								</Button>
							</div>
						)}
					</Form.List>
				</Form.Item>
				<div
					style={{
						display: "grid",
						gridTemplateColumns: "1fr 1fr",
						gap: 12,
					}}
				>
					<Form.Item
						label="CPU"
						name="cpu"
						extra="例如 500m、1"
						rules={[resourceQuantityRule("cpu")]}
					>
						<Input id="node-config-cpu" name="cpu" placeholder="500m" />
					</Form.Item>
					<Form.Item
						label="内存"
						name="memory"
						extra="必须带单位，例如 512Mi、1Gi"
						rules={[resourceQuantityRule("memory")]}
					>
						<Input id="node-config-memory" name="memory" placeholder="256Mi" />
					</Form.Item>
					<Form.Item
						label="磁盘"
						name="disk"
						extra="必须带单位，例如 1Gi、20Gi"
						rules={[resourceQuantityRule("disk")]}
					>
						<Input id="node-config-disk" name="disk" placeholder="1Gi" />
					</Form.Item>
				</div>
			</Form>
		</Modal>
	);
}
