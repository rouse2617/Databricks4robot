import { PlusOutlined } from "@ant-design/icons";
import {
	Button,
	Card,
	Empty,
	Input,
	message,
	Popconfirm,
	Popover,
	Select,
	Space,
	Tag,
} from "antd";
import { useEffect, useState } from "react";
import { assetsApi } from "../../api/assets";
import { type TagRegistryItem, tagRegistryApi } from "../../api/tagRegistry";

interface Props {
	assetId: string;
	tags: Record<string, string>;
	onUpdate: () => void;
}

export default function TagsTab({ assetId, tags, onUpdate }: Props) {
	const [editingKey, setEditingKey] = useState<string | null>(null);
	const [editValue, setEditValue] = useState("");
	const [saving, setSaving] = useState(false);
	const [addOpen, setAddOpen] = useState(false);
	const [newKey, setNewKey] = useState("");
	const [newValue, setNewValue] = useState("");
	const [registry, setRegistry] = useState<TagRegistryItem[]>([]);

	useEffect(() => {
		tagRegistryApi
			.list()
			.then(setRegistry)
			.catch(() => {});
	}, []);

	const handleSaveEdit = async (key: string, value: string) => {
		if (value === tags[key]) {
			setEditingKey(null);
			return;
		}
		setSaving(true);
		try {
			await assetsApi.upsertTag(assetId, { key, value });
			message.success("标签已更新");
			onUpdate();
		} catch {
			message.error("更新标签失败");
		} finally {
			setSaving(false);
			setEditingKey(null);
		}
	};

	const handleDelete = async (key: string) => {
		setSaving(true);
		try {
			await assetsApi.deleteTag(assetId, key);
			message.success("标签已删除");
			onUpdate();
		} catch {
			message.error("删除标签失败");
		} finally {
			setSaving(false);
		}
	};

	const handleAdd = async () => {
		if (!newKey) return;
		setSaving(true);
		try {
			await assetsApi.upsertTag(assetId, { key: newKey, value: newValue });
			message.success("标签已添加");
			setNewKey("");
			setNewValue("");
			setAddOpen(false);
			onUpdate();
		} catch {
			message.error("添加标签失败");
		} finally {
			setSaving(false);
		}
	};

	// Keys available for adding (not already present)
	const availableKeys = registry.map((r) => r.key).filter((k) => !(k in tags));

	const tagEntries = Object.entries(tags);

	const addContent = (
		<div style={{ width: 240 }}>
			<div className="mb-2">
				<Select
					placeholder="选择标签 Key"
					value={newKey || undefined}
					onChange={setNewKey}
					style={{ width: "100%" }}
					size="small"
					options={availableKeys.map((k) => ({ label: k, value: k }))}
					showSearch
				/>
			</div>
			<div className="mb-2">
				{(() => {
					const regItem = registry.find((r) => r.key === newKey);
					if (regItem?.type === "enum" && regItem.values?.length) {
						return (
							<Select
								placeholder="选择值"
								value={newValue || undefined}
								onChange={setNewValue}
								style={{ width: "100%" }}
								size="small"
								options={regItem.values.map((v) => ({ label: v, value: v }))}
							/>
						);
					}
					return (
						<Input
							placeholder="输入值"
							value={newValue}
							onChange={(e) => setNewValue(e.target.value)}
							size="small"
						/>
					);
				})()}
			</div>
			<Button
				type="primary"
				size="small"
				block
				disabled={!newKey}
				loading={saving}
				onClick={handleAdd}
			>
				添加
			</Button>
		</div>
	);

	return (
		<Card size="small">
			{tagEntries.length > 0 ? (
				<Space wrap>
					{tagEntries.map(([k, v]) => (
						<Tag key={k} closable={false}>
							<span>{k}: </span>
							{editingKey === k ? (
								<Input
									size="small"
									style={{ width: 100, display: "inline-block" }}
									autoFocus
									defaultValue={v}
									value={editValue}
									onChange={(e) => setEditValue(e.target.value)}
									onBlur={() => handleSaveEdit(k, editValue)}
									onPressEnter={() => handleSaveEdit(k, editValue)}
									disabled={saving}
								/>
							) : (
								<button
									type="button"
									className="inline-button-reset"
									style={{ cursor: "pointer", borderBottom: "1px dashed #999" }}
									onClick={() => {
										setEditingKey(k);
										setEditValue(v);
									}}
									title="点击编辑"
								>
									{v}
								</button>
							)}
							<Popconfirm
								title={`确认删除标签 "${k}"？`}
								onConfirm={() => handleDelete(k)}
							>
								<Button
									type="text"
									size="small"
									danger
									style={{ marginLeft: 4, padding: "0 2px" }}
								>
									×
								</Button>
							</Popconfirm>
						</Tag>
					))}
				</Space>
			) : (
				<Empty description="暂无标签" image={Empty.PRESENTED_IMAGE_SIMPLE} />
			)}

			<div className="mt-3">
				<Popover
					content={addContent}
					title="新增标签"
					trigger="click"
					open={addOpen}
					onOpenChange={setAddOpen}
				>
					<Button size="small" icon={<PlusOutlined />}>
						新增标签
					</Button>
				</Popover>
			</div>
		</Card>
	);
}
