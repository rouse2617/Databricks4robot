import { PlusOutlined } from "@ant-design/icons";
import {
	AutoComplete,
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
	Typography,
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

// Values longer than this are truncated in the chip; the full text stays
// available via a click-to-expand popover (CYB-3246 Task A).
const VALUE_TRUNCATE_LEN = 40;

// TagValueText renders a tag value, truncating long strings with an affordance
// to view the full content in a popover. Short values render inline unchanged.
function TagValueText({ value }: { value: string }) {
	const isLong = value.length > VALUE_TRUNCATE_LEN;
	if (!isLong) {
		return <span>{value}</span>;
	}
	const preview = `${value.slice(0, VALUE_TRUNCATE_LEN)}…`;
	return (
		<Popover
			trigger="click"
			title="标签完整内容"
			content={
				<Typography.Paragraph
					style={{ maxWidth: 360, marginBottom: 0, whiteSpace: "pre-wrap" }}
					copyable
				>
					{value}
				</Typography.Paragraph>
			}
		>
			<span
				style={{ cursor: "pointer", textDecoration: "underline dotted" }}
				title="点击查看完整内容"
			>
				{preview}
			</span>
		</Popover>
	);
}

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
		const nextDetailed = localDetailed.filter(
			(r) =>
				!(r.tag_key === key && (!sourceType || r.source_type === sourceType)),
		);
		setLocalDetailed(nextDetailed);
		// If this was the last source for this tag key, also remove from flat map
		const hasRemaining = nextDetailed.some((r) => r.tag_key === key);
		if (!sourceType || !hasRemaining) {
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
		// Optimistic local insert so the new tag shows up immediately, matching
		// the UX of handleDeleteSource. localTags is a Record<string,string> and
		// localDetailed is an array — update each in its own shape and snapshot
		// the previous state so we can roll back precisely on failure.
		const prevTags = localTags;
		const prevDetailed = localDetailed;
		const optimisticRow: AssetTagDetail = {
			tag_key: newKey,
			tag_value: newValue,
			source_type: newSource,
			source_name: newSourceName || undefined,
			source_version: newSourceVersion || undefined,
		};
		setLocalTags((prev) => ({ ...prev, [newKey]: newValue }));
		setLocalDetailed((prev) => [
			...prev.filter(
				(r) => !(r.tag_key === newKey && r.source_type === newSource),
			),
			optimisticRow,
		]);
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
			// Roll back optimistic insert on failure.
			setLocalTags(prevTags);
			setLocalDetailed(prevDetailed);
			message.error(err instanceof Error ? err.message : "添加标签失败");
		} finally {
			setSaving(false);
		}
	};

	const groupedEntries = Object.entries(grouped);

	// Registered keys are offered as suggestions, but the key field accepts any
	// free-form key (open vocabulary, CYB-3246) — the backend stores unknown
	// keys as string tags.
	const keyOptions = registry.map((r) => ({
		value: r.key,
		label: r.description ? `${r.key} — ${r.description}` : r.key,
	}));
	const regItem = registry.find((r) => r.key === newKey);
	const isEnumKey = regItem?.type === "enum" && !!regItem.values?.length;

	const addContent = (
		<div style={{ width: 280 }}>
			<div className="mb-2">
				<AutoComplete
					placeholder="标签 Key（可输入自定义）"
					value={newKey}
					onChange={(v) => setNewKey(v)}
					style={{ width: "100%" }}
					size="small"
					options={keyOptions}
					filterOption={(input, option) =>
						String(option?.value ?? "")
							.toLowerCase()
							.includes(input.toLowerCase())
					}
					allowClear
				/>
			</div>
			<div className="mb-2">
				{isEnumKey ? (
					<Select
						placeholder="选择值"
						value={newValue || undefined}
						onChange={setNewValue}
						style={{ width: "100%" }}
						size="small"
						options={regItem?.values?.map((v) => ({ label: v, value: v }))}
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
				<Space direction="vertical" style={{ width: "100%" }} size={10}>
					{groupedEntries.map(([k, rows]) => (
						<div
							key={k}
							style={{
								display: "flex",
								alignItems: "flex-start",
								gap: 8,
								flexWrap: "wrap",
								padding: "6px 8px",
								borderRadius: 6,
								border: "1px solid var(--color-border-secondary, #f0f0f0)",
							}}
						>
							<Tag closable={false} style={{ marginRight: 0, fontWeight: 600 }}>
								{k}
							</Tag>
							<Space wrap size={[6, 6]} style={{ flex: 1 }}>
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
										<span
											style={{
												display: "inline-flex",
												alignItems: "center",
												gap: 6,
												maxWidth: "100%",
												padding: "2px 6px",
												borderRadius: 6,
												background: "var(--color-fill-quaternary, #fafafa)",
												border:
													"1px solid var(--color-border-secondary, #f0f0f0)",
											}}
										>
											<TagValueText value={row.tag_value} />
											<Tag
												color={SOURCE_COLORS[row.source_type] ?? "default"}
												style={{ margin: 0, fontSize: 11, lineHeight: "16px" }}
											>
												{row.source_type}
												{row.source_version ? `@${row.source_version}` : ""}
											</Tag>
											<Popconfirm
												title={`删除 ${row.source_type} 的 ${k}？`}
												okText="删除"
												cancelText="取消"
												okButtonProps={{ danger: true }}
												onConfirm={() => handleDeleteSource(k, row.source_type)}
											>
												<Button
													type="text"
													size="small"
													danger
													style={{ padding: "0 2px", height: 18 }}
												>
													×
												</Button>
											</Popconfirm>
										</span>
									</Tooltip>
								))}
								{rows.length > 1 ? (
									<Popconfirm
										title={`删除 ${k} 的全部来源？`}
										okText="清空"
										cancelText="取消"
										okButtonProps={{ danger: true }}
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
				<Empty
					description="暂无标签。可手动添加，或等待上游算法打标完成。"
					image={Empty.PRESENTED_IMAGE_SIMPLE}
				/>
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
