import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { Node } from "@xyflow/react";
import { Alert, Button, Collapse, Form, Input, Modal, Select } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
	listRuntimeMounts,
	type RuntimeMountCatalog,
	type RuntimeSecretMountResource,
	type RuntimeStorageMountResource,
} from "../../api/pipelineApi";
import {
	type PipelineConfig,
	type PipelineConfigVersion,
	pipelineConfigApi,
} from "../../api/pipelineConfigs";
import type {
	Argument,
	PipelineNodeData,
	PipelineNodeRuntimeConfig,
	PipelineNodeRuntimeSecretMount,
	PipelineNodeRuntimeStorageMount,
	Port,
} from "./types";

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
type RuntimeSecretFormItem = {
	resourceId?: string;
	mountPath?: string;
};
type RuntimeStorageFormItem = {
	resourceId?: string;
	mountPath?: string;
	readOnly?: boolean;
};

type FormValues = {
	label: string;
	command?: string[];
	source?: string;
	args: string[];
	env?: EnvFormItem[];
	inputPorts?: PortFormItem[];
	outputPorts?: PortFormItem[];
	runtimeConfigId?: string;
	runtimeConfigVersion?: number;
	runtimeConfigMountPath?: string;
	runtimeConfigTargetFilename?: string;
	runtimeSecrets?: RuntimeSecretFormItem[];
	storageMounts?: RuntimeStorageFormItem[];
	cpu: string;
	memory: string;
	disk: string;
};

const DEFAULT_INPUT_PORTS: Port[] = [{ name: "input", type: "asset" }];
const DEFAULT_OUTPUT_PORTS: Port[] = [{ name: "output", type: "asset" }];
const DEFAULT_CONFIG_MOUNT_PATH = "/workspace/configs";
const EMPTY_RUNTIME_MOUNT_CATALOG: RuntimeMountCatalog = {
	secrets: [],
	storage: [],
};

function shortHash(value?: string) {
	return value ? value.slice(0, 8) : "—";
}

function formatConfigVersionTime(value?: string) {
	if (!value) return "";
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return value;
	const pad = (item: number) => item.toString().padStart(2, "0");
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function configVersionOptionLabel(
	version: PipelineConfigVersion,
	currentVersion?: number,
) {
	return [
		`v${version.version}`,
		version.version === currentVersion ? "当前" : null,
		version.summary || "无说明",
		formatConfigVersionTime(version.createdAt),
		`sha:${shortHash(version.contentSha256)}`,
	]
		.filter(Boolean)
		.join(" · ");
}

function httpStatus(err: unknown): number | undefined {
	if (!err || typeof err !== "object") return undefined;
	const response = (err as { response?: { status?: unknown } }).response;
	return typeof response?.status === "number" ? response.status : undefined;
}

function savedRuntimeConfigVersion(
	runtimeConfig: PipelineNodeRuntimeConfig | undefined,
	configId: string,
): PipelineConfigVersion | null {
	if (runtimeConfig?.configId !== configId || !runtimeConfig.version) {
		return null;
	}
	return {
		id: `${configId}:v${runtimeConfig.version}`,
		configId,
		version: runtimeConfig.version,
		status: "ready",
		contentSha256: "",
		contentSizeBytes: 0,
		summary: "模板中已保存的版本引用",
		author: "",
		createdAt: "",
	};
}

function runtimeConfigDisplayName(name: string, version: number): string {
	const suffix = ` · v${version}`;
	return name.endsWith(suffix) ? name : `${name}${suffix}`;
}

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

function runtimeSecretsToForm(
	items: PipelineNodeRuntimeSecretMount[] | undefined,
): RuntimeSecretFormItem[] {
	return (items || []).map((item) => ({
		resourceId: item.resourceId,
		mountPath: item.mountPath,
	}));
}

function storageMountsToForm(
	items: PipelineNodeRuntimeStorageMount[] | undefined,
): RuntimeStorageFormItem[] {
	return (items || []).map((item) => ({
		resourceId: item.resourceId,
		mountPath: item.mountPath,
		readOnly: item.readOnly,
	}));
}

function runtimeSecretsFromForm(
	items: RuntimeSecretFormItem[] | undefined,
	resources: RuntimeSecretMountResource[],
): PipelineNodeRuntimeSecretMount[] {
	const byId = new Map(resources.map((item) => [item.id, item]));
	const seen = new Set<string>();
	const out: PipelineNodeRuntimeSecretMount[] = [];
	for (const item of items || []) {
		const resourceId = item.resourceId?.trim();
		if (!resourceId || seen.has(resourceId)) continue;
		seen.add(resourceId);
		const resource = byId.get(resourceId);
		out.push({
			resourceId,
			mountPath:
				item.mountPath?.trim() || resource?.defaultMountPath || "/mnt/secrets",
			displayName: resource?.name || resourceId,
		});
	}
	return out;
}

function storageMountsFromForm(
	items: RuntimeStorageFormItem[] | undefined,
	resources: RuntimeStorageMountResource[],
): PipelineNodeRuntimeStorageMount[] {
	const byId = new Map(resources.map((item) => [item.id, item]));
	const seen = new Set<string>();
	const out: PipelineNodeRuntimeStorageMount[] = [];
	for (const item of items || []) {
		const resourceId = item.resourceId?.trim();
		if (!resourceId || seen.has(resourceId)) continue;
		seen.add(resourceId);
		const resource = byId.get(resourceId);
		const readOnly =
			typeof item.readOnly === "boolean" ? item.readOnly : resource?.readOnly;
		out.push({
			resourceId,
			mountPath:
				item.mountPath?.trim() ||
				resource?.defaultMountPath ||
				defaultRuntimeStorageMountPath(resource),
			...(typeof readOnly === "boolean" ? { readOnly } : {}),
			displayName: resource?.name || resourceId,
		});
	}
	return out;
}

function defaultRuntimeStorageMountPath(
	resource: RuntimeStorageMountResource | undefined,
) {
	return resource?.kind === "pvc" ? "/workspace/shared" : "/workspace/scratch";
}

function defaultRuntimeSecretFormItem(
	resource: RuntimeSecretMountResource | undefined,
): RuntimeSecretFormItem {
	return resource
		? {
				resourceId: resource.id,
				mountPath: resource.defaultMountPath || "/mnt/secrets",
			}
		: {};
}

function defaultRuntimeStorageFormItem(
	resource: RuntimeStorageMountResource | undefined,
): RuntimeStorageFormItem {
	return resource
		? {
				resourceId: resource.id,
				mountPath:
					resource.defaultMountPath || defaultRuntimeStorageMountPath(resource),
				readOnly: resource.readOnly,
			}
		: {};
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
				<div
					style={{
						display: "grid",
						gap: 8,
						border: "1px solid #e2e8f0",
						borderRadius: 8,
						padding: 12,
						minWidth: 0,
					}}
				>
					<div
						style={{
							display: "flex",
							justifyContent: "space-between",
							alignItems: "center",
							gap: 8,
						}}
					>
						<strong>{title}</strong>
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
								gridTemplateColumns: "minmax(0, 1fr) 96px 32px",
								gap: 8,
								alignItems: "center",
								minWidth: 0,
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
							结果要传给后续步骤时，例如输出名为 output，就写入
							{" /tmp/outputs/output"}。
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
	const [configs, setConfigs] = useState<PipelineConfig[]>([]);
	const [configVersions, setConfigVersions] = useState<PipelineConfigVersion[]>(
		[],
	);
	const [configsLoading, setConfigsLoading] = useState(false);
	const [configsError, setConfigsError] = useState<string | null>(null);
	const [configVersionsLoading, setConfigVersionsLoading] = useState(false);
	const [configVersionsError, setConfigVersionsError] = useState<string | null>(
		null,
	);
	const [configVersionsNotice, setConfigVersionsNotice] = useState<
		string | null
	>(null);
	const [runtimeMountCatalog, setRuntimeMountCatalog] =
		useState<RuntimeMountCatalog>(EMPTY_RUNTIME_MOUNT_CATALOG);
	const [runtimeMountsLoading, setRuntimeMountsLoading] = useState(false);
	const [runtimeMountsError, setRuntimeMountsError] = useState<string | null>(
		null,
	);
	const [runtimeSecretAdvancedKeys, setRuntimeSecretAdvancedKeys] = useState<
		string[]
	>([]);

	const componentType = String(node.data.type || "").toLowerCase();
	const isScriptType = componentType === "script";
	const selectedConfigId = Form.useWatch("runtimeConfigId", form);
	const selectedConfig = useMemo(
		() => configs.find((config) => config.id === selectedConfigId),
		[configs, selectedConfigId],
	);
	const configOptions = useMemo(() => {
		const options = configs.map((config) => ({
			value: config.id,
			label: `${config.name} · 当前 v${config.currentVersion} · ${config.lifecycle}`,
		}));
		const runtimeConfig = node.data.runtimeConfig;
		const shouldShowSavedReference =
			selectedConfigId &&
			runtimeConfig?.configId === selectedConfigId &&
			!configs.some((config) => config.id === selectedConfigId);
		if (shouldShowSavedReference) {
			options.unshift({
				value: selectedConfigId,
				label: `${
					runtimeConfig.displayName ||
					runtimeConfig.fileName ||
					runtimeConfig.targetFilename ||
					selectedConfigId
				} · 模板引用`,
			});
		}
		return options;
	}, [configs, node.data.runtimeConfig, selectedConfigId]);
	const configVersionOptions = useMemo(
		() =>
			configVersions.map((version) => ({
				value: version.version,
				label: configVersionOptionLabel(
					version,
					selectedConfig?.currentVersion,
				),
			})),
		[configVersions, selectedConfig?.currentVersion],
	);
	const runtimeSecretOptions = useMemo(
		() =>
			runtimeMountCatalog.secrets.map((resource) => ({
				value: resource.id,
				label: `${resource.name} · ${resource.defaultMountPath}${
					resource.targetIds?.length ? ` · ${resource.targetIds.join("/")}` : ""
				}`,
			})),
		[runtimeMountCatalog.secrets],
	);
	const runtimeStorageOptions = useMemo(
		() =>
			runtimeMountCatalog.storage.map((resource) => ({
				value: resource.id,
				label: `${resource.name} · ${resource.kind} · ${resource.defaultMountPath}${
					resource.targetIds?.length ? ` · ${resource.targetIds.join("/")}` : ""
				}`,
			})),
		[runtimeMountCatalog.storage],
	);

	useEffect(() => {
		if (!open) return;
		const runtimeConfig = node.data.runtimeConfig;
		form.setFieldsValue({
			label: node.data.label || "",
			source: node.data.source || "",
			command: node.data.command || [],
			args: normalizeArgs(node.data.args),
			env: envToPairs(node.data.env),
			inputPorts: normalizePorts(node.data.inputPorts, DEFAULT_INPUT_PORTS),
			outputPorts: normalizePorts(node.data.outputPorts, DEFAULT_OUTPUT_PORTS),
			runtimeConfigId: runtimeConfig?.configId,
			runtimeConfigVersion: runtimeConfig?.version,
			runtimeConfigMountPath:
				runtimeConfig?.mountPath || DEFAULT_CONFIG_MOUNT_PATH,
			runtimeConfigTargetFilename:
				runtimeConfig?.targetFilename || runtimeConfig?.fileName || "",
			runtimeSecrets: runtimeSecretsToForm(node.data.runtimeSecrets),
			storageMounts: storageMountsToForm(node.data.storageMounts),
			cpu: node.data.cpu || "",
			memory: node.data.memory || "",
			disk: node.data.disk || "",
		});
		setRuntimeSecretAdvancedKeys(
			node.data.runtimeSecrets?.length ? ["runtime-secrets"] : [],
		);
	}, [form, node.data, open]);

	useEffect(() => {
		if (!open) return;
		let active = true;
		setConfigsLoading(true);
		setConfigsError(null);
		pipelineConfigApi
			.list()
			.then((result) => {
				if (!active) return;
				setConfigs(
					(result.items || []).filter(
						(config) => config.lifecycle !== "deprecated",
					),
				);
			})
			.catch((err: unknown) => {
				if (!active) return;
				setConfigsError(
					err instanceof Error ? err.message : "配置列表加载失败",
				);
			})
			.finally(() => {
				if (active) setConfigsLoading(false);
			});
		return () => {
			active = false;
		};
	}, [open]);

	useEffect(() => {
		if (!open) return;
		let active = true;
		setRuntimeMountsLoading(true);
		setRuntimeMountsError(null);
		listRuntimeMounts()
			.then((catalog) => {
				if (!active) return;
				setRuntimeMountCatalog({
					secrets: catalog.secrets || [],
					storage: catalog.storage || [],
				});
			})
			.catch((err: unknown) => {
				if (!active) return;
				setRuntimeMountCatalog(EMPTY_RUNTIME_MOUNT_CATALOG);
				setRuntimeMountsError(
					err instanceof Error ? err.message : "运行挂载资源加载失败",
				);
			})
			.finally(() => {
				if (active) setRuntimeMountsLoading(false);
			});
		return () => {
			active = false;
		};
	}, [open]);

	useEffect(() => {
		if (!open || !selectedConfigId) {
			setConfigVersions([]);
			setConfigVersionsError(null);
			setConfigVersionsNotice(null);
			setConfigVersionsLoading(false);
			return;
		}
		const existingVersion = savedRuntimeConfigVersion(
			node.data.runtimeConfig,
			selectedConfigId,
		);
		const selectedConfigIsListed = configs.some(
			(config) => config.id === selectedConfigId,
		);
		if (existingVersion && !selectedConfigIsListed) {
			setConfigVersionsError(null);
			if (configsLoading) {
				setConfigVersions([]);
				setConfigVersionsNotice(null);
				setConfigVersionsLoading(true);
				return;
			}
			setConfigVersions([existingVersion]);
			setConfigVersionsLoading(false);
			setConfigVersionsNotice(
				"当前账号不能查看该配置详情；保存节点时会保留模板里的配置版本引用。",
			);
			if (!form.getFieldValue("runtimeConfigVersion")) {
				form.setFieldsValue({
					runtimeConfigVersion: existingVersion.version,
				});
			}
			return;
		}
		let active = true;
		setConfigVersionsLoading(true);
		setConfigVersionsError(null);
		setConfigVersionsNotice(null);
		pipelineConfigApi
			.get(selectedConfigId)
			.then((config) => {
				if (!active) return;
				const readyVersions = (config.versions || []).filter(
					(version) => !version.status || version.status === "ready",
				);
				setConfigVersions(readyVersions);
				const currentVersion = form.getFieldValue("runtimeConfigVersion");
				const currentVersionStillAvailable = readyVersions.some(
					(version) => version.version === currentVersion,
				);
				if (!currentVersionStillAvailable) {
					const defaultVersion =
						readyVersions.find(
							(version) => version.version === config.currentVersion,
						)?.version ?? readyVersions[0]?.version;
					form.setFieldsValue({ runtimeConfigVersion: defaultVersion });
				}
				setConfigVersionsNotice(null);
			})
			.catch((err: unknown) => {
				if (!active) return;
				if (existingVersion && [403, 404].includes(httpStatus(err) || 0)) {
					setConfigVersions([existingVersion]);
					setConfigVersionsError(null);
					setConfigVersionsNotice(
						"当前账号不能查看该配置详情；保存节点时会保留模板里的配置版本引用。",
					);
					if (!form.getFieldValue("runtimeConfigVersion")) {
						form.setFieldsValue({
							runtimeConfigVersion: existingVersion.version,
						});
					}
					return;
				}
				setConfigVersions([]);
				setConfigVersionsNotice(null);
				setConfigVersionsError(
					err instanceof Error ? err.message : "配置版本加载失败",
				);
			})
			.finally(() => {
				if (active) setConfigVersionsLoading(false);
			});
		return () => {
			active = false;
		};
	}, [
		configs,
		configsLoading,
		form,
		node.data.runtimeConfig,
		open,
		selectedConfigId,
	]);

	const handleSubmit = async () => {
		const values = await form.validateFields();
		let runtimeConfig: PipelineNodeRuntimeConfig | undefined;
		if (values.runtimeConfigId) {
			const existing =
				node.data.runtimeConfig?.configId === values.runtimeConfigId
					? node.data.runtimeConfig
					: undefined;
			const selectedVersion = configVersions.find(
				(version) => version.version === values.runtimeConfigVersion,
			);
			const version =
				selectedVersion?.version ||
				values.runtimeConfigVersion ||
				existing?.version ||
				0;
			const fileName =
				selectedConfig?.name ||
				existing?.fileName ||
				values.runtimeConfigTargetFilename ||
				"runtime-config.yaml";
			if (!version) {
				form.setFields([
					{
						name: "runtimeConfigVersion",
						errors: ["请选择可用的 ready 配置版本"],
					},
				]);
				return;
			}
			const displayName =
				selectedConfig?.name || existing?.displayName || fileName;
			runtimeConfig = {
				mode: "saved",
				configId: values.runtimeConfigId,
				version,
				fileName,
				mountPath:
					values.runtimeConfigMountPath?.trim() || DEFAULT_CONFIG_MOUNT_PATH,
				targetFilename: values.runtimeConfigTargetFilename?.trim() || fileName,
				displayName: runtimeConfigDisplayName(displayName, version),
			};
		}
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
			runtimeConfig,
			runtimeSecrets: runtimeSecretsFromForm(
				values.runtimeSecrets,
				runtimeMountCatalog.secrets,
			),
			storageMounts: storageMountsFromForm(
				values.storageMounts,
				runtimeMountCatalog.storage,
			),
			cpu: values.cpu || "",
			memory: values.memory || "",
			disk: values.disk || "",
		};

		onSave(node.id, nextData);
		onCancel();
	};

	const handleConfigChange = (configId?: string) => {
		if (!configId) {
			form.setFieldsValue({
				runtimeConfigVersion: undefined,
				runtimeConfigTargetFilename: "",
			});
			return;
		}
		const selectedConfig = configs.find((config) => config.id === configId);
		if (!selectedConfig) return;
		const currentTarget = form.getFieldValue("runtimeConfigTargetFilename");
		form.setFieldsValue({
			runtimeConfigVersion: undefined,
			runtimeConfigMountPath:
				form.getFieldValue("runtimeConfigMountPath") ||
				DEFAULT_CONFIG_MOUNT_PATH,
			runtimeConfigTargetFilename: currentTarget || selectedConfig.name,
		});
	};

	const handleRuntimeSecretChange = (index: number, resourceId?: string) => {
		const resource = runtimeMountCatalog.secrets.find(
			(item) => item.id === resourceId,
		);
		const current = [
			...((form.getFieldValue("runtimeSecrets") ||
				[]) as RuntimeSecretFormItem[]),
		];
		current[index] = {
			...current[index],
			resourceId,
			mountPath: resource?.defaultMountPath || current[index]?.mountPath,
		};
		form.setFieldsValue({ runtimeSecrets: current });
		form.setFields([
			{ name: ["runtimeSecrets", index, "resourceId"], errors: [] },
		]);
	};

	const handleRuntimeStorageChange = (index: number, resourceId?: string) => {
		const resource = runtimeMountCatalog.storage.find(
			(item) => item.id === resourceId,
		);
		const current = [
			...((form.getFieldValue("storageMounts") ||
				[]) as RuntimeStorageFormItem[]),
		];
		current[index] = {
			...current[index],
			resourceId,
			mountPath: resource?.defaultMountPath || current[index]?.mountPath,
			readOnly: resource?.readOnly ?? current[index]?.readOnly,
		};
		form.setFieldsValue({ storageMounts: current });
		form.setFields([
			{ name: ["storageMounts", index, "resourceId"], errors: [] },
		]);
	};

	return (
		<Modal
			title="节点配置"
			open={open}
			onCancel={onCancel}
			width={640}
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
				<Form.Item
					label="命令（可选）"
					name="command"
					extra="留空时使用镜像 Dockerfile 中的 ENTRYPOINT/CMD；只有需要覆盖镜像入口时填写。"
				>
					<Select
						mode="tags"
						options={[]}
						placeholder="可选：例如 sh、-c；按 Enter 添加片段"
					/>
				</Form.Item>
				<Form.Item
					label="参数（可选）"
					name="args"
					extra="留空时使用镜像默认参数；如果使用 sh -c，参数只填写脚本本体。连接下游时请确保运行时写入 /tmp/outputs/<输出名>。"
				>
					<Select
						mode="tags"
						options={[]}
						placeholder="可选：按 Enter 添加参数"
					/>
				</Form.Item>
				<div
					style={{
						display: "grid",
						gridTemplateColumns: "1fr",
						gap: 12,
						marginBottom: 16,
					}}
				>
					<PortConfigList
						name="inputPorts"
						title="输入数据"
						emptyPort={{ name: "input", type: "asset" }}
					/>
					<PortConfigList
						name="outputPorts"
						title="输出结果"
						emptyPort={{ name: "output", type: "asset" }}
						outputPathHint
					/>
				</div>
				<div
					style={{
						border: "1px solid #e2e8f0",
						borderRadius: 8,
						padding: 12,
						marginBottom: 16,
					}}
				>
					<strong>配置文件</strong>
					<Form.Item
						label="配置"
						name="runtimeConfigId"
						style={{ marginTop: 12 }}
					>
						<Select
							allowClear
							loading={configsLoading}
							options={configOptions}
							placeholder="选择有 ready 版本的配置"
							onChange={handleConfigChange}
						/>
					</Form.Item>
					<Form.Item
						label="版本"
						name="runtimeConfigVersion"
						rules={
							selectedConfigId
								? [{ required: true, message: "请选择配置版本" }]
								: undefined
						}
					>
						<Select
							loading={configVersionsLoading}
							options={configVersionOptions}
							placeholder="选择配置版本"
							disabled={!selectedConfigId}
						/>
					</Form.Item>
					{configsError ? (
						<Alert
							type="warning"
							showIcon
							message="配置列表加载失败"
							description={configsError}
							style={{ marginBottom: 12 }}
						/>
					) : null}
					{configVersionsError ? (
						<Alert
							type="warning"
							showIcon
							message="配置版本加载失败"
							description={configVersionsError}
							style={{ marginBottom: 12 }}
						/>
					) : null}
					{configVersionsNotice ? (
						<Alert
							type="info"
							showIcon
							message="保留模板配置引用"
							description={configVersionsNotice}
							style={{ marginBottom: 12 }}
						/>
					) : null}
					{!configsLoading &&
					!configsError &&
					configs.length === 0 &&
					!selectedConfigId ? (
						<Alert
							type="info"
							showIcon
							message="暂无可用配置"
							style={{ marginBottom: 12 }}
						/>
					) : null}
					{selectedConfigId &&
					!configVersionsLoading &&
					!configVersionsError &&
					configVersions.length === 0 ? (
						<Alert
							type="info"
							showIcon
							message="暂无 ready 配置版本"
							style={{ marginBottom: 12 }}
						/>
					) : null}
					<div
						style={{
							display: "grid",
							gridTemplateColumns: "1fr 1fr",
							gap: 12,
						}}
					>
						<Form.Item label="挂载目录" name="runtimeConfigMountPath">
							<Input
								placeholder={DEFAULT_CONFIG_MOUNT_PATH}
								disabled={!selectedConfigId}
							/>
						</Form.Item>
						<Form.Item label="目标文件名" name="runtimeConfigTargetFilename">
							<Input
								placeholder="app-config.yaml"
								disabled={!selectedConfigId}
							/>
						</Form.Item>
					</div>
				</div>
				<div
					style={{
						border: "1px solid #e2e8f0",
						borderRadius: 8,
						padding: 12,
						marginBottom: 16,
					}}
				>
					<strong>存储挂载</strong>
					{runtimeMountsError ? (
						<Alert
							type="warning"
							showIcon
							message="运行挂载资源加载失败"
							description={runtimeMountsError}
							style={{ marginTop: 12, marginBottom: 12 }}
						/>
					) : null}
					<Form.List name="storageMounts">
						{(fields, { add, remove }) => (
							<div style={{ display: "grid", gap: 8, marginTop: 12 }}>
								{fields.map(({ key, ...field }) => (
									<div
										key={key}
										style={{
											display: "grid",
											gridTemplateColumns:
												"minmax(0, 1fr) minmax(0, 1fr) 96px auto",
											gap: 8,
											alignItems: "center",
										}}
									>
										<Form.Item
											{...field}
											name={[field.name, "resourceId"]}
											rules={[{ required: true, message: "请选择存储资源" }]}
											noStyle
										>
											<Select
												aria-label="选择存储资源"
												allowClear
												loading={runtimeMountsLoading}
												options={runtimeStorageOptions}
												placeholder="选择存储资源"
												onChange={(value) =>
													handleRuntimeStorageChange(field.name, value)
												}
											/>
										</Form.Item>
										<Form.Item
											{...field}
											name={[field.name, "mountPath"]}
											noStyle
										>
											<Input
												aria-label="存储挂载路径"
												placeholder="/workspace/scratch"
											/>
										</Form.Item>
										<Form.Item
											{...field}
											name={[field.name, "readOnly"]}
											noStyle
										>
											<Select
												aria-label="存储读写模式"
												options={[
													{ value: true, label: "只读" },
													{ value: false, label: "读写" },
												]}
											/>
										</Form.Item>
										<Button
											type="text"
											icon={<MinusCircleOutlined />}
											onClick={() => remove(field.name)}
											danger
										/>
									</div>
								))}
								<Button
									type="dashed"
									icon={<PlusOutlined />}
									onClick={() =>
										add(
											defaultRuntimeStorageFormItem(
												runtimeMountCatalog.storage.length === 1
													? runtimeMountCatalog.storage[0]
													: undefined,
											),
										)
									}
									disabled={runtimeMountsLoading}
								>
									新增存储挂载
								</Button>
								{!runtimeMountsLoading &&
								!runtimeMountsError &&
								runtimeMountCatalog.storage.length === 0 ? (
									<Alert type="info" showIcon message="暂无平台存储资源" />
								) : null}
							</div>
						)}
					</Form.List>
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
				<Collapse
					size="small"
					style={{ marginTop: 4 }}
					activeKey={runtimeSecretAdvancedKeys}
					onChange={(keys) => {
						const nextKeys = Array.isArray(keys) ? keys : [keys];
						setRuntimeSecretAdvancedKeys(nextKeys.map(String));
					}}
					items={[
						{
							key: "runtime-secrets",
							label: "高级配置：密钥挂载",
							children: (
								<div style={{ display: "grid", gap: 12 }}>
									<Alert
										type="warning"
										showIcon
										message="密钥挂载由平台维护"
										description="只选择平台允许的密钥资源和挂载目录，不在这里填写或展示密钥内容。"
									/>
									{runtimeMountsError ? (
										<Alert
											type="warning"
											showIcon
											message="运行挂载资源加载失败"
											description={runtimeMountsError}
										/>
									) : null}
									<Form.List name="runtimeSecrets">
										{(fields, { add, remove }) => (
											<div style={{ display: "grid", gap: 8 }}>
												{fields.map(({ key, ...field }) => (
													<div
														key={key}
														style={{
															display: "grid",
															gridTemplateColumns:
																"minmax(0, 1fr) minmax(0, 1fr) auto",
															gap: 8,
															alignItems: "center",
														}}
													>
														<Form.Item
															{...field}
															name={[field.name, "resourceId"]}
															rules={[
																{
																	required: true,
																	message: "请选择密钥资源",
																},
															]}
															noStyle
														>
															<Select
																aria-label="选择密钥资源"
																allowClear
																loading={runtimeMountsLoading}
																options={runtimeSecretOptions}
																placeholder="选择密钥资源"
																onChange={(value) =>
																	handleRuntimeSecretChange(field.name, value)
																}
															/>
														</Form.Item>
														<Form.Item
															{...field}
															name={[field.name, "mountPath"]}
															noStyle
														>
															<Input
																aria-label="密钥挂载路径"
																placeholder="/mnt/secrets"
															/>
														</Form.Item>
														<Button
															type="text"
															icon={<MinusCircleOutlined />}
															onClick={() => remove(field.name)}
															danger
														/>
													</div>
												))}
												<Button
													type="dashed"
													icon={<PlusOutlined />}
													onClick={() =>
														add(
															defaultRuntimeSecretFormItem(
																runtimeMountCatalog.secrets.length === 1
																	? runtimeMountCatalog.secrets[0]
																	: undefined,
															),
														)
													}
													disabled={runtimeMountsLoading}
												>
													新增密钥挂载
												</Button>
												{!runtimeMountsLoading &&
												!runtimeMountsError &&
												runtimeMountCatalog.secrets.length === 0 ? (
													<Alert
														type="info"
														showIcon
														message="暂无平台密钥资源"
													/>
												) : null}
											</div>
										)}
									</Form.List>
								</div>
							),
						},
					]}
				/>
			</Form>
		</Modal>
	);
}
