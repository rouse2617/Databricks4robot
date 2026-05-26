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
	Tooltip,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { assetsApi } from "../../api/assets";
import { type TagRegistryItem, tagRegistryApi } from "../../api/tagRegistry";
import type { AssetTagDetail } from "../../api/types";

interface Props {
	assetId: string;
	tags: Record<string, string>;
	tagsDetailed?: AssetTagDetail[];
	onUpdate: () => void;
}

const SOURCE_OPTIONS = [
	{ label: "human", value: "human" },
	{ label: "algo_sdk", value: "algo_sdk" },
	{ label: "rule_engine", value: "rule_engine" },
	{ label: "system", value: "system" },
	{ label: "compliance", value: "compliance" },
];

const SOURCE_COLORS: Record<string, string> = {
	human: "blue",
	algo_sdk: "geekblue",
	rule_engine: "purple",
	system: "default",
	compliance: "red",
};

// Sources that the registry requires identity fields for. Mirrors
// backend/config/tag_registry.yaml — kept here to avoid an extra round-trip
// to GET /api/v1/tag-registry on the add panel.
const REQUIRES_SOURCE_NAME = new Set([
	"human",
	"algo_sdk",
	"rule_engine",
	"compliance",
]);
const REQUIRES_SOURCE_VERSION = new Set(["algo_sdk", "rule_engine"]);

export default function TagsTab({
	assetId,
	tags: initialTags,
	tagsDetailed,
	onUpdate,
}: Props) {
	const [saving, setSaving] = useState(false);
	const [addOpen, setAddOpen] = useState(false);
	const [newKey, setNewKey] = useState("");
	const [newValue, setNewValue] = useState("");
	const [newSource, setNewSource] = useState("human");
	const [newSourceName, setNewSourceName] = useState("");
	const [newSourceVersion, setNewSourceVersion] = useState("");
	const [registry, setRegistry] = useState<TagRegistryItem[]>([]);
	const [localTags, setLocalTags] = useState(initialTags);
	const [localDetailed, setLocalDetailed] = useState<AssetTagDetail[]>(
		tagsDetailed ?? [],
	);

	useEffect(() => {
		setLocalTags(initialTags);
	}, [initialTags]);
	useEffect(() => {
		setLocalDetailed(tagsDetailed ?? []);
	}, [tagsDetailed]);

	useEffect(() => {
		tagRegistryApi
			.list()
			.then(setRegistry)
			.catch(() => {});
	}, []);

	// Group detailed rows by tag_key so the UI can render one Tag chip per
	// key with the source chips inside it.
	const grouped = useMemo(() => {
		const out: Record<string, AssetTagDetail[]> = {};
		for (const row of localDetailed) {
			if (!row?.tag_key) continue;
			if (!out[row.tag_key]) out[row.tag_key] = [];
			out[row.tag_key].push(row);
		}
		// Fall back to flat map for keys that have no detailed row (older
		// data path or environments that did not populate tags_detailed).
		for (const [k, v] of Object.entries(localTags)) {
			if (!out[k]) {
				out[k] = [
					{
						tag_key: k,
						tag_value: v,
						source_type: "system",
					},
				];
			}
		}
		return out;
	}, [localDetailed, localTags]);

	const handleDeleteSource = async (
		key: string,
		sourceType: string | undefined,
	) => {
		setLocalDetailed((prev) =>
			prev.filter(
				(r) =>
					!(r.tag_key === key && (!sourceType || r.source_type === sourceType)),
			),
		);
		if (!sourceType) {
			setLocalTags((prev) => {
				const next = { ...prev };
				delete next[key];
				return next;
			});
		}
		setSaving(true);
		try {
			await assetsApi.deleteTag(assetId, key, sourceType);
			message.success(
				sourceType ? `已删除 ${sourceType} 的 ${key}` : `已删除标签 ${key}`,
			);
			onUpdate();
		} catch {
			setLocalTags(initialTags);
			setLocalDetailed(tagsDetailed ?? []);
			message.error("删除标签失败");
		} finally {
			setSaving(false);
		}
	};

	const handleAdd = async () => {
		if (!newKey) return;
		if (REQUIRES_SOURCE_NAME.has(newSource) && !newSourceName) {
			message.warning(`${newSource} 需要填写 source_name`);
			return;
		}
		if (REQUIRES_SOURCE_VERSION.has(newSource) && !newSourceVersion) {
			message.warning(`${newSource} 需要填写 source_version`);
			return;
		}
		setSaving(true);
		try {
			await assetsApi.upsertTag(assetId, {
				key: newKey,
				value: newValue,
				source_type: newSource,
				source_name: newSourceName || undefined,
				source_version: newSourceVersion || undefined,
			});
			message.success("标签已添加");
			setNewKey("");
			setNewValue("");
			setNewSourceName("");
			setNewSourceVersion("");
			setAddOpen(false);
			onUpdate();
		} catch (err) {
			message.error(err instanceof Error ? err.message : "添加标签失败");
		} finally {
			setSaving(false);
		}
	};

	const groupedEntries = Object.entries(grouped);

	const availableKeys = registry.map((r) => r.key);
	const regItem = registry.find((r) => r.key === newKey);

	const addContent = (
		<div style={{ width: 280 }}>
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
				{regItem?.type === "enum" && regItem.values?.length ? (
					<Select
						placeholder="选择值"
						value={newValue || undefined}
						onChange={setNewValue}
						style={{ width: "100%" }}
						size="small"
						options={regItem.values.map((v) => ({ label: v, value: v }))}
					/>
				) : (
					<Input
						placeholder="输入值"
						value={newValue}
						onChange={(e) => setNewValue(e.target.value)}
						size="small"
					/>
				)}
			</div>
			<div className="mb-2">
				<Select
					value={newSource}
					onChange={setNewSource}
					style={{ width: "100%" }}
					size="small"
					options={SOURCE_OPTIONS}
				/>
			</div>
			{REQUIRES_SOURCE_NAME.has(newSource) ? (
				<div className="mb-2">
					<Input
						placeholder="source_name (必填)"
						value={newSourceName}
						onChange={(e) => setNewSourceName(e.target.value)}
						size="small"
					/>
				</div>
			) : null}
			{REQUIRES_SOURCE_VERSION.has(newSource) ? (
				<div className="mb-2">
					<Input
						placeholder="source_version (必填)"
						value={newSourceVersion}
						onChange={(e) => setNewSourceVersion(e.target.value)}
						size="small"
					/>
				</div>
			) : null}
			<Button
				type="primary"
				size="small"
				block
				disabled={!newKey || !newValue}
				loading={saving}
				onClick={handleAdd}
			>
				添加
			</Button>
		</div>
	);

	return (
		<Card size="small">
			{groupedEntries.length > 0 ? (
				<Space direction="vertical" style={{ width: "100%" }} size={6}>
					{groupedEntries.map(([k, rows]) => (
						<div key={k}>
							<Tag closable={false} style={{ marginRight: 8 }}>
								<strong>{k}</strong>
							</Tag>
							<Space wrap size={[6, 4]}>
								{rows.map((row) => (
									<Tooltip
										key={`${k}|${row.source_type}|${row.source_version || ""}|${row.source_name || ""}`}
										title={[
											row.source_name && `name: ${row.source_name}`,
											row.source_version && `version: ${row.source_version}`,
											row.applied_at && `at: ${row.applied_at}`,
										]
											.filter(Boolean)
											.join("  ·  ")}
									>
										<Tag
											color={SOURCE_COLORS[row.source_type] ?? "default"}
											closable={false}
										>
											<span>{row.tag_value}</span>
											<span style={{ marginLeft: 6, opacity: 0.7 }}>
												[{row.source_type}
												{row.source_version ? `@${row.source_version}` : ""}]
											</span>
											<Popconfirm
												title={`删除 ${row.source_type} 的 ${k}？`}
												onConfirm={() => handleDeleteSource(k, row.source_type)}
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
									</Tooltip>
								))}
								{rows.length > 1 ? (
									<Popconfirm
										title={`删除 ${k} 的全部来源？`}
										onConfirm={() => handleDeleteSource(k, undefined)}
									>
										<Button size="small" type="text" danger>
											清空
										</Button>
									</Popconfirm>
								) : null}
							</Space>
						</div>
					))}
				</Space>
			) : (
				<Empty description="暂无标签" image={Empty.PRESENTED_IMAGE_SIMPLE} />
			)}

			<div style={{ marginTop: 12 }}>
				<Popover
					content={addContent}
					trigger="click"
					open={addOpen}
					onOpenChange={setAddOpen}
				>
					<Button size="small" icon={<PlusOutlined />}>
						添加标签
					</Button>
				</Popover>
			</div>
		</Card>
	);
}
