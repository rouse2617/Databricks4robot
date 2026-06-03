import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { Node } from "@ant-design/pro-flow";
import { Button, Form, Input, Modal, Select } from "antd";
import { useEffect } from "react";
import type { Argument, PipelineNodeData, Port } from "./types";

interface NodeConfigPanelProps {
	open: boolean;
	node: Node<PipelineNodeData>;
	onCancel: () => void;
	onSave: (id: string, data: Partial<PipelineNodeData>) => void;
}

type EnvFormItem = { name?: string; value?: string };
type PortFormItem = {
	name?: string;
	type?: string;
	desc?: string;
	default_value?: string;
};

type FormValues = {
	label: string;
	command?: string[];
	source?: string;
	args: string[];
	env?: EnvFormItem[];
	inputPorts?: PortFormItem[];
	outputPorts?: PortFormItem[];
	cpu: string;
	memory: string;
	disk: string;
};

const DEFAULT_INPUT_PORTS: Port[] = [{ name: "input", type: "asset" }];
const DEFAULT_OUTPUT_PORTS: Port[] = [{ name: "output", type: "asset" }];

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

function normalizePorts(ports: Port[] | undefined, fallback: Port[]): Port[] {
	return ports?.length ? ports : fallback;
}

function formPortsToPorts(items: PortFormItem[], fallback: Port[]): Port[] {
	if (!items || items.length === 0) return fallback;
	const seen = new Set<string>();
	const next: Port[] = [];
	for (const item of items) {
		const name = item.name?.trim();
		if (!name || seen.has(name)) continue;
		seen.add(name);
		next.push({
			name,
			type: item.type?.trim() || "string",
			...(item.desc?.trim() ? { desc: item.desc.trim() } : {}),
			...(item.default_value?.trim()
				? { default_value: item.default_value.trim() }
				: {}),
		});
	}
	return next.length > 0 ? next : fallback;
}

function PortConfigList({
	name,
	title,
	emptyPort,
	outputPathHint = false,
}: {
	name: "inputPorts" | "outputPorts";
	title: string;
	emptyPort: PortFormItem;
	outputPathHint?: boolean;
}) {
	return (
		<Form.List name={name} initialValue={[emptyPort]}>
			{(fields, { add, remove }) => (
				<div style={{ display: "grid", gap: 8 }}>
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
						}}
					>
						<span>{title}</span>
						<Button
							size="small"
							icon={<PlusOutlined />}
							onClick={() => add(emptyPort)}
						>
							新增端口
						</Button>
					</div>
					{fields.map(({ key, ...field }) => (
						<div
							key={key}
							style={{
								display: "grid",
								gridTemplateColumns: "1fr 96px auto",
								gap: 8,
							}}
						>
							<Form.Item
								{...field}
								name={[field.name, "name"]}
								rules={[{ required: true, message: "请输入端口名" }]}
								noStyle
							>
								<Input placeholder={emptyPort.name || "input"} />
							</Form.Item>
							<Form.Item
								{...field}
								name={[field.name, "type"]}
								rules={[{ required: true, message: "请输入类型" }]}
								noStyle
							>
								<Input placeholder={emptyPort.type || "asset"} />
							</Form.Item>
							<Button
								type="text"
								icon={<MinusCircleOutlined />}
								onClick={() => remove(field.name)}
								danger
							/>
						</div>
					))}
					{outputPathHint ? (
						<span style={{ fontSize: 12, color: "#64748b" }}>
							输出端口被连接时，需要写入 /tmp/outputs/端口名。
						</span>
					) : null}
				</div>
			)}
		</Form.List>
	);
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
			inputPorts: normalizePorts(node.data.inputPorts, DEFAULT_INPUT_PORTS),
			outputPorts: normalizePorts(node.data.outputPorts, DEFAULT_OUTPUT_PORTS),
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
			inputPorts: formPortsToPorts(
				values.inputPorts || [],
				DEFAULT_INPUT_PORTS,
			),
			outputPorts: formPortsToPorts(
				values.outputPorts || [],
				DEFAULT_OUTPUT_PORTS,
			),
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
					<Input placeholder="节点名称" />
				</Form.Item>
				{isScriptType && (
					<Form.Item label="source" name="source">
						<Input.TextArea rows={3} placeholder="Python 脚本 source" />
					</Form.Item>
				)}
				<Form.Item label="命令" name="command">
					<Select
						mode="tags"
						options={[]}
						placeholder="按 Enter 添加命令片段"
					/>
				</Form.Item>
				<Form.Item label="参数" name="args">
					<Select mode="tags" options={[]} placeholder="按 Enter 添加参数" />
				</Form.Item>
				<div
					style={{
						display: "grid",
						gridTemplateColumns: "1fr 1fr",
						gap: 12,
						marginBottom: 16,
					}}
				>
					<PortConfigList
						name="inputPorts"
						title="输入端口"
						emptyPort={{ name: "input", type: "asset" }}
					/>
					<PortConfigList
						name="outputPorts"
						title="输出端口"
						emptyPort={{ name: "output", type: "asset" }}
						outputPathHint
					/>
				</div>
				<Form.Item label="环境变量">
					<Form.List name="env">
						{(fields, { add, remove }) => (
							<div style={{ display: "grid", gap: 8 }}>
								{fields.map(({ key, ...field }) => (
									<div
										key={key}
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
											<Input placeholder="KEY" />
										</Form.Item>
										<Form.Item {...field} name={[field.name, "value"]} noStyle>
											<Input placeholder="VALUE" />
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
					<Form.Item label="CPU" name="cpu">
						<Input placeholder="500m" />
					</Form.Item>
					<Form.Item label="内存" name="memory">
						<Input placeholder="256Mi" />
					</Form.Item>
					<Form.Item label="磁盘" name="disk">
						<Input placeholder="1Gi" />
					</Form.Item>
				</div>
			</Form>
		</Modal>
	);
}
