import { PlusOutlined } from "@ant-design/icons";
import { Alert, Button, Collapse, Input, Typography } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { toAssetStyleId } from "../../lib/idDisplay";
import type { RegisteredComponent } from "./types";

const { Text } = Typography;

interface Props {
	components: RegisteredComponent[];
	onChange: (c: RegisteredComponent[]) => void;
	/** Called when a component is saved (create or update). */
	onSaveApi?: (comp: RegisteredComponent, isNew: boolean) => Promise<void>;
	/** Called when a component is deleted. */
	onDeleteApi?: (id: string) => Promise<void>;
}

function blank(): RegisteredComponent {
	return {
		id: `c${Date.now()}`,
		name: "",
		type: "container",
		source: "custom",
		image: "",
		command: ["sh", "-c"],
		args: [],
		env: [],
		cpu: "",
		memory: "",
		disk: "",
		gpu: "",
		computeTier: "",
	};
}

function getTypeLabel(type: string): string {
	switch (type) {
		case "container":
			return "Container";
		case "script":
			return "Script";
		default:
			return "自定义";
	}
}

function getGroupOrder(typeLabel: string): number {
	switch (typeLabel) {
		case "Container":
			return 0;
		case "Script":
			return 1;
		default:
			return 2;
	}
}

export function ComponentManager({
	components,
	onChange,
	onSaveApi,
	onDeleteApi,
}: Props) {
	const [editing, setEditing] = useState<RegisteredComponent | null>(null);
	const [isNew, setIsNew] = useState(false);
	const [commandText, setCommandText] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [saving, setSaving] = useState(false);
	const [deleting, setDeleting] = useState(false);
	const [queryText, setQueryText] = useState("");

	const filtered = useMemo(() => {
		const kw = queryText.trim().toLowerCase();
		if (!kw) return components;
		return components.filter((component) =>
			component.name.toLowerCase().includes(kw),
		);
	}, [components, queryText]);

	const groups = useMemo(() => {
		const buckets = new Map<string, RegisteredComponent[]>();
		for (const comp of filtered) {
			const group = getTypeLabel(comp.type || "");
			const arr = buckets.get(group);
			if (arr) {
				arr.push(comp);
			} else {
				buckets.set(group, [comp]);
			}
		}
		return Array.from(buckets.entries())
			.sort(
				(a, b) =>
					getGroupOrder(a[0]) - getGroupOrder(b[0]) || a[0].localeCompare(b[0]),
			)
			.map(([label, comps]) => ({
				key: label,
				label: `${label} (${comps.length})`,
				comps,
			}));
	}, [filtered]);

	useEffect(() => {
		if (editing) {
			setCommandText(JSON.stringify(editing.command));
		}
	}, [editing]);

	const save = useCallback(
		async (c: RegisteredComponent) => {
			setSaving(true);
			setError(null);
			try {
				if (onSaveApi) {
					await onSaveApi(c, isNew);
				}

				let updated: RegisteredComponent[];
				if (isNew) {
					updated = [...components, c];
				} else {
					updated = components.map((x) => (x.id === c.id ? c : x));
				}
				onChange(updated);
				setEditing(null);
			} catch (err) {
				setError(err instanceof Error ? err.message : "组件保存失败");
			} finally {
				setSaving(false);
			}
		},
		[components, isNew, onChange, onSaveApi],
	);

	const remove = useCallback(
		async (id: string) => {
			setDeleting(true);
			setError(null);
			try {
				if (onDeleteApi) {
					await onDeleteApi(id);
				}

				onChange(components.filter((x) => x.id !== id));
				if (editing?.id === id) setEditing(null);
			} catch (err) {
				setError(err instanceof Error ? err.message : "组件删除失败");
			} finally {
				setDeleting(false);
			}
		},
		[components, editing?.id, onChange, onDeleteApi],
	);

	return (
		<div className="component-manager">
			<div className="cm-header">
				<h3>组件注册表</h3>
				<Button
					type="primary"
					size="small"
					icon={<PlusOutlined />}
					onClick={() => {
						setEditing(blank());
						setIsNew(true);
					}}
				>
					新建
				</Button>
			</div>

			<div className="cm-list">
				<div className="cm-search">
					<Input.Search
						allowClear
						value={queryText}
						onChange={(event) => setQueryText(event.target.value)}
						placeholder="按名称搜索"
						size="small"
					/>
				</div>

				{filtered.length === 0 && (
					<Text
						type="secondary"
						style={{ padding: 40, textAlign: "center", display: "block" }}
					>
						{queryText ? "未找到匹配组件" : "暂无注册组件"}
					</Text>
				)}
				{groups.length > 0 && (
					<Collapse
						size="small"
						items={groups.map((group) => ({
							key: group.key,
							label: group.label,
							children: (
								<div className="cm-group-list">
									{group.comps.map((c) => (
										<button
											key={c.id}
											type="button"
											className={`cm-item ${editing?.id === c.id ? "active" : ""}`}
											onClick={() => {
												setEditing(c);
												setIsNew(false);
											}}
										>
											<div className="cm-item-name">{c.name}</div>
											<div className="cm-item-id" title={`完整 ID: ${c.id}`}>
												ID: {toAssetStyleId(c.id)}
											</div>
											<div className="cm-item-image">{c.image}</div>
										</button>
									))}
								</div>
							),
						}))}
					/>
				)}
			</div>

			{editing && (
				<div className="cm-form">
					<h4>{isNew ? "新建组件" : "编辑组件"}</h4>
					<div className="cm-form-fields">
						<div className="cm-field">
							<label htmlFor="cm-name">名称</label>
							<Input
								id="cm-name"
								value={editing.name}
								onChange={(e) =>
									setEditing({ ...editing, name: e.target.value })
								}
								placeholder="my-component"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-image">镜像</label>
							<Input
								id="cm-image"
								value={editing.image}
								onChange={(e) =>
									setEditing({ ...editing, image: e.target.value })
								}
								placeholder="repo/image:tag"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-command">命令 (JSON)</label>
							<Input
								id="cm-command"
								value={commandText}
								onChange={(e) => setCommandText(e.target.value)}
								onBlur={() => {
									try {
										const parsed = JSON.parse(commandText);
										if (Array.isArray(parsed)) {
											setEditing({ ...editing, command: parsed });
										}
									} catch {
										/* ignore */
									}
								}}
								placeholder='["sh", "-c"]'
							/>
							<Text type="secondary" className="cm-field-hint">
								声明输出的组件需要在运行时写入 /tmp/outputs/output，否则 Argo
								会将节点标记为失败。
							</Text>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-cpu">CPU</label>
							<Input
								id="cm-cpu"
								value={editing.cpu}
								onChange={(e) =>
									setEditing({ ...editing, cpu: e.target.value })
								}
								placeholder="500m"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-memory">内存</label>
							<Input
								id="cm-memory"
								value={editing.memory}
								onChange={(e) =>
									setEditing({ ...editing, memory: e.target.value })
								}
								placeholder="256Mi"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-disk">磁盘</label>
							<Input
								id="cm-disk"
								value={editing.disk}
								onChange={(e) =>
									setEditing({ ...editing, disk: e.target.value })
								}
								placeholder="1Gi"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-gpu">GPU</label>
							<Input
								id="cm-gpu"
								value={editing.gpu}
								onChange={(e) =>
									setEditing({ ...editing, gpu: e.target.value })
								}
								placeholder="1"
							/>
						</div>
						<div className="cm-field">
							<label htmlFor="cm-compute-tier">计算档位</label>
							<Input
								id="cm-compute-tier"
								value={editing.computeTier}
								onChange={(e) =>
									setEditing({ ...editing, computeTier: e.target.value })
								}
								placeholder="gpu-l4"
							/>
						</div>
					</div>
					<div className="cm-form-actions">
						<Button
							type="primary"
							loading={saving}
							disabled={deleting}
							onClick={() => save(editing)}
						>
							{isNew ? "创建" : "保存"}
						</Button>
						{!isNew && (
							<Button
								danger
								loading={deleting}
								disabled={saving}
								onClick={() => remove(editing.id)}
							>
								删除
							</Button>
						)}
						<Button
							disabled={saving || deleting}
							onClick={() => setEditing(null)}
						>
							取消
						</Button>
					</div>
					{error && (
						<Alert
							type="error"
							showIcon
							message={error}
							style={{ marginTop: 12 }}
						/>
					)}
				</div>
			)}
		</div>
	);
}
