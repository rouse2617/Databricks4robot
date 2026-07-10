// ─── AssetsFacetSidebar — Presentational Component ───
// 5 collapsible facet groups for the assets discovery workbench.
// Validates: Requirements R4

import {
	Badge,
	Button,
	Checkbox,
	Collapse,
	DatePicker,
	Input,
	InputNumber,
	Switch,
	Tag,
	Typography,
} from "antd";
import { useCallback, useEffect, useRef, useState } from "react";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

// ─── Facet Option Constants ───

export const STATUS_OPTIONS = [
	"approved",
	"rejected",
	"superseded",
	"archived",
];
export const LIFECYCLE_OPTIONS = [
	"created",
	"processing",
	"ready",
	"rejected",
	"delivered",
	"archived",
	"superseded",
];
export const ASSET_TYPE_OPTIONS = [
	"segment",
	"clip",
	"frame_set",
	"derived_asset",
];
export const RETENTION_TIER_OPTIONS = ["standard", "archive", "cold"];
export const ENV_OPTIONS = [
	"kitchen",
	"outdoor",
	"warehouse",
	"office",
	"factory",
];
export const ALGO_STATUS_OPTIONS = [
	"ok",
	"failed",
	"running",
	"pending",
	"blocked",
];
export const PRIORITY_OPTIONS = ["critical", "high", "medium", "low"];
export const QUALITY_OPTIONS = [
	"excellent",
	"good",
	"acceptable",
	"poor",
	"unusable",
];

export const GROUP_KEYS = [
	"quick",
	"frequent",
	"basic",
	"capture",
	"algorithm",
	"delivery",
	"tags",
] as const;

// ─── Props ───

/** Aggregation key mapping from ES agg names to filter field names. */
const AGG_KEY_MAP: Record<string, string> = {
	lifecycle_state_agg: "lifecycle_state",
	asset_type_agg: "asset_type",
	owner_agg: "owner",
	vendor_agg: "mcap.vendor_id",
	scene_agg: "mcap.scene_id",
	priority_agg: "tag.priority",
	quality_agg: "tag.quality",
	env_agg: "env",
};

export interface AssetsFacetSidebarProps {
	activeFilters: FilterChip[];
	expandedGroups: string[];
	rangeDrafts: Record<string, { min?: number; max?: number }>;
	dateDrafts: Record<string, { start?: string; end?: string }>;
	aggregations?: Record<string, { key: string; doc_count: number }[]>;
	layout?: "vertical" | "horizontal";
	compact?: boolean;
	onToggleFacet: (field: string, value: string) => void;
	onApplyRange: (field: string, min?: number, max?: number) => void;
	onApplyDate: (field: string, start?: string, end?: string) => void;
	onRangeDraftChange: (field: string, min?: number, max?: number) => void;
	onDateDraftChange: (field: string, start?: string, end?: string) => void;
	onToggleGroup: (group: string) => void;
}

// ─── Helpers ───

/** Extract checked values for a given field from active filters. */
export function getCheckedValues(
	filters: FilterChip[],
	field: string,
): string[] {
	return filters
		.filter((f) => f.field === field)
		.flatMap((f) => (Array.isArray(f.value) ? f.value : [f.value]));
}

/** Check if a switch-type facet is active. */
export function isSwitchActive(filters: FilterChip[], field: string): boolean {
	return filters.some((f) => f.field === field);
}

/** Get the text value for an input-type facet. */
export function getInputValue(filters: FilterChip[], field: string): string {
	const chip = filters.find((f) => f.field === field);
	if (!chip) return "";
	return Array.isArray(chip.value) ? (chip.value[0] ?? "") : chip.value;
}

// ─── Sub-components for each facet type ───

function facetFieldStyle(
	compact = false,
	columnSpan: "full" | "half" = "full",
): React.CSSProperties {
	return {
		marginBottom: compact ? 8 : 12,
		minWidth: 0,
		gridColumn: columnSpan === "half" ? "span 1" : "1 / -1",
	};
}

function facetGridStyle(minWidth = 160): React.CSSProperties {
	return {
		display: "grid",
		gridTemplateColumns: `repeat(auto-fit, minmax(min(${minWidth}px, 100%), 1fr))`,
		gap: 8,
		alignItems: "start",
	};
}

function CheckboxFacet({
	label,
	field,
	options,
	activeFilters,
	onToggleFacet,
	counts,
	compact = false,
	span = "full",
}: {
	label: string;
	field: string;
	options: string[];
	activeFilters: FilterChip[];
	onToggleFacet: (field: string, value: string) => void;
	counts?: Record<string, number>;
	compact?: boolean;
	span?: "full" | "half";
}) {
	const checked = getCheckedValues(activeFilters, field);
	const optionsWithLabels = options.map((opt) => {
		const count = counts?.[opt];
		return {
			label: count !== undefined ? `${opt} (${count.toLocaleString()})` : opt,
			value: opt,
		};
	});
	return (
		<div style={facetFieldStyle(compact, span)}>
			<Text
				type="secondary"
				style={{ fontSize: 12, display: "block", marginBottom: 4 }}
			>
				{label}
			</Text>
			<Checkbox.Group
				value={checked}
				onChange={(newValues) => {
					const added = (newValues as string[]).filter(
						(v) => !checked.includes(v),
					);
					const removed = checked.filter(
						(v) => !(newValues as string[]).includes(v),
					);
					for (const v of added) onToggleFacet(field, v);
					for (const v of removed) onToggleFacet(field, v);
				}}
				style={facetGridStyle(compact ? 132 : 144)}
			>
				{optionsWithLabels.map((option) => (
					<Checkbox
						key={option.value}
						value={option.value}
						style={{ marginInlineStart: 0, lineHeight: 1.4 }}
					>
						{option.label}
					</Checkbox>
				))}
			</Checkbox.Group>
		</div>
	);
}

function CheckableTagFacet({
	label,
	field,
	options,
	activeFilters,
	onToggleFacet,
	counts,
	span = "full",
}: {
	label: string;
	field: string;
	options: string[];
	activeFilters: FilterChip[];
	onToggleFacet: (field: string, value: string) => void;
	counts?: Record<string, number>;
	span?: "full" | "half";
}) {
	const checked = getCheckedValues(activeFilters, field);
	return (
		<div style={facetFieldStyle(false, span)}>
			<Text
				type="secondary"
				style={{ fontSize: 12, display: "block", marginBottom: 6 }}
			>
				{label}
			</Text>
			<div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
				{options.map((opt) => {
					const count = counts?.[opt];
					const isChecked = checked.includes(opt);
					const label = count !== undefined ? `${opt} (${count})` : opt;
					return (
						<Tag.CheckableTag
							key={opt}
							checked={isChecked}
							onChange={() => onToggleFacet(field, opt)}
							style={{
								cursor: "pointer",
								borderRadius: 16,
								paddingInline: 10,
								height: 28,
								display: "flex",
								alignItems: "center",
								fontSize: 12,
								lineHeight: "28px",
								border: isChecked ? "1px solid #1890ff" : "1px solid #d9d9d9",
								background: isChecked ? "#e6f7ff" : "#fff",
								color: isChecked ? "#1890ff" : "rgba(0,0,0,0.65)",
							}}
						>
							{label}
						</Tag.CheckableTag>
					);
				})}
			</div>
		</div>
	);
}

function chipToggleValue(chip: FilterChip): string {
	const v = chip.value;
	return Array.isArray(v) ? String(v[0]) : String(v);
}

/** Removes every active filter chip for `field` (FACET_TOGGLE toggles each off). */
function stripFieldFilters(
	activeFilters: FilterChip[],
	field: string,
	onToggleFacet: (field: string, value: string) => void,
) {
	for (const chip of activeFilters.filter((c) => c.field === field)) {
		onToggleFacet(field, chipToggleValue(chip));
	}
}

function InputFacet({
	label,
	field,
	placeholder,
	activeFilters,
	onToggleFacet,
	compact = false,
	span = "full",
}: {
	label: string;
	field: string;
	placeholder: string;
	activeFilters: FilterChip[];
	onToggleFacet: (field: string, value: string) => void;
	compact?: boolean;
	span?: "full" | "half";
}) {
	const committed = getInputValue(activeFilters, field);
	const [draft, setDraft] = useState(committed);
	const commitLockRef = useRef(false);

	useEffect(() => {
		setDraft(committed);
	}, [committed]);

	const commitDraft = useCallback(() => {
		if (commitLockRef.current) return;
		const trimmed = draft.trim();
		const committedTrim = committed.trim();
		if (trimmed === committedTrim) return;

		commitLockRef.current = true;
		try {
			stripFieldFilters(activeFilters, field, onToggleFacet);
			if (trimmed) {
				onToggleFacet(field, trimmed);
			}
		} finally {
			queueMicrotask(() => {
				commitLockRef.current = false;
			});
		}
	}, [activeFilters, committed, draft, field, onToggleFacet]);

	return (
		<div style={facetFieldStyle(compact, span)}>
			<Text
				type="secondary"
				style={{ fontSize: 12, display: "block", marginBottom: 4 }}
			>
				{label}
			</Text>
			<Input
				size="small"
				placeholder={placeholder}
				value={draft}
				style={{ width: "100%" }}
				onChange={(e) => {
					setDraft(e.target.value);
				}}
				onPressEnter={() => {
					commitDraft();
				}}
				onBlur={() => {
					commitDraft();
				}}
				allowClear
				onClear={() => {
					setDraft("");
					stripFieldFilters(activeFilters, field, onToggleFacet);
				}}
			/>
		</div>
	);
}

function RangeFacet({
	label,
	field,
	rangeDrafts,
	onRangeDraftChange,
	onApplyRange,
	compact = false,
	span = "half",
}: {
	label: string;
	field: string;
	rangeDrafts: Record<string, { min?: number; max?: number }>;
	onRangeDraftChange: (field: string, min?: number, max?: number) => void;
	onApplyRange: (field: string, min?: number, max?: number) => void;
	compact?: boolean;
	span?: "full" | "half";
}) {
	const draft = rangeDrafts[field] ?? {};
	return (
		<div style={facetFieldStyle(compact, span)}>
			<Text
				type="secondary"
				style={{ fontSize: 12, display: "block", marginBottom: 4 }}
			>
				{label}
			</Text>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr) auto",
					gap: 6,
					alignItems: "center",
				}}
			>
				<InputNumber
					size="small"
					placeholder="Min"
					value={draft.min}
					onChange={(v) => onRangeDraftChange(field, v ?? undefined, draft.max)}
					style={{ width: "100%" }}
				/>
				<InputNumber
					size="small"
					placeholder="Max"
					value={draft.max}
					onChange={(v) => onRangeDraftChange(field, draft.min, v ?? undefined)}
					style={{ width: "100%" }}
				/>
				<Button
					size="small"
					type="primary"
					onClick={() => onApplyRange(field, draft.min, draft.max)}
				>
					应用
				</Button>
			</div>
		</div>
	);
}

function DateRangeFacet({
	label,
	field,
	dateDrafts,
	onDateDraftChange,
	onApplyDate,
	compact = false,
	span = "half",
}: {
	label: string;
	field: string;
	dateDrafts: Record<string, { start?: string; end?: string }>;
	onDateDraftChange: (field: string, start?: string, end?: string) => void;
	onApplyDate: (field: string, start?: string, end?: string) => void;
	compact?: boolean;
	span?: "full" | "half";
}) {
	const draft = dateDrafts[field] ?? {};
	return (
		<div style={facetFieldStyle(compact, span)}>
			<Text
				type="secondary"
				style={{ fontSize: 12, display: "block", marginBottom: 4 }}
			>
				{label}
			</Text>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "minmax(0, 1fr) auto",
					gap: 6,
					alignItems: "center",
				}}
			>
				<DatePicker.RangePicker
					size="small"
					onChange={(_dates, dateStrings) => {
						const [start, end] = dateStrings;
						onDateDraftChange(field, start || undefined, end || undefined);
					}}
					style={{ width: "100%" }}
				/>
				<Button
					size="small"
					type="primary"
					onClick={() => onApplyDate(field, draft.start, draft.end)}
				>
					应用
				</Button>
			</div>
		</div>
	);
}

function SwitchFacet({
	label,
	field,
	activeFilters,
	onToggleFacet,
	compact = false,
	span = "half",
}: {
	label: string;
	field: string;
	activeFilters: FilterChip[];
	onToggleFacet: (field: string, value: string) => void;
	compact?: boolean;
	span?: "full" | "half";
}) {
	const active = isSwitchActive(activeFilters, field);
	return (
		<div
			style={{
				...facetFieldStyle(compact, span),
				display: "flex",
				alignItems: "center",
				gap: 8,
			}}
		>
			<Switch
				size="small"
				checked={active}
				onChange={() => onToggleFacet(field, "true")}
			/>
			<Text style={{ fontSize: 13 }}>{label}</Text>
		</div>
	);
}

// ─── Group Label Map ───

const GROUP_LABELS: Record<string, string> = {
	quick: "快速筛选",
	frequent: "常用筛选",
	basic: "基础",
	capture: "采集",
	algorithm: "算法",
	delivery: "交付",
	tags: "标签",
};

const GROUP_FIELDS: Record<string, string[]> = {
	quick: ["asset_type", "algo_status"],
	frequent: ["owner", "retention_tier", "created_at"],
	basic: ["lifecycle_state", "reviewer"],
	capture: [
		"env",
		"mcap.vendor_id",
		"mcap.device_id",
		"mcap.scene_id",
		"duration_ms",
		"updated_at",
	],
	algorithm: ["algo_status"],
	delivery: ["has:delivery", "delivery_count"],
	tags: ["tag.priority", "tag.quality", "tags.scene"],
};

// Quick filter groups that should always be visible in horizontal layout
const QUICK_FILTER_GROUPS = ["quick"];

// ─── Main Component ───

export default function AssetsFacetSidebar({
	activeFilters,
	expandedGroups,
	rangeDrafts,
	dateDrafts,
	aggregations,
	layout = "vertical",
	compact = false,
	onToggleFacet,
	onApplyRange,
	onApplyDate,
	onRangeDraftChange,
	onDateDraftChange,
	onToggleGroup,
}: AssetsFacetSidebarProps) {
	// Build counts map per field from aggregations
	const fieldCounts: Record<string, Record<string, number>> = {};
	if (aggregations) {
		for (const [aggKey, buckets] of Object.entries(aggregations)) {
			const field = AGG_KEY_MAP[aggKey];
			if (field && buckets) {
				const counts: Record<string, number> = {};
				for (const b of buckets) {
					counts[b.key] = b.doc_count;
				}
				fieldCounts[field] = counts;
			}
		}
	}

	const items = [
		{
			key: "quick",
			label: GROUP_LABELS.quick,
			children: (
				<CheckboxFacet
					compact={compact}
					label="资产类型"
					field="asset_type"
					options={ASSET_TYPE_OPTIONS}
					activeFilters={activeFilters}
					onToggleFacet={onToggleFacet}
					counts={fieldCounts.asset_type}
				/>
			),
		},
		{
			key: "basic",
			label: GROUP_LABELS.basic,
			children: (
				<>
					<CheckboxFacet
						compact={compact}
						label="生命周期"
						field="lifecycle_state"
						options={LIFECYCLE_OPTIONS}
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
						counts={fieldCounts.lifecycle_state}
					/>
					<CheckboxFacet
						compact={compact}
						label="保留层级"
						field="retention_tier"
						options={RETENTION_TIER_OPTIONS}
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<InputFacet
						compact={compact}
						span="half"
						label="Owner"
						field="owner"
						placeholder="搜索 owner"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<InputFacet
						compact={compact}
						span="half"
						label="Reviewer"
						field="reviewer"
						placeholder="搜索 reviewer"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
				</>
			),
		},
		{
			key: "capture",
			label: GROUP_LABELS.capture,
			children: (
				<>
					{fieldCounts.env && Object.keys(fieldCounts.env).length > 0 && (
						<CheckboxFacet
							compact={compact}
							label="环境"
							field="env"
							// CYB-3297 Phase B: env is free-form data — list the real
							// values from the aggregation (most frequent first) instead
							// of a hardcoded set, so users can discover what exists.
							options={Object.keys(fieldCounts.env).sort(
								(a, b) => (fieldCounts.env[b] ?? 0) - (fieldCounts.env[a] ?? 0),
							)}
							activeFilters={activeFilters}
							onToggleFacet={onToggleFacet}
							counts={fieldCounts.env}
						/>
					)}
					<InputFacet
						compact={compact}
						span="half"
						label="Vendor"
						field="mcap.vendor_id"
						placeholder="vendor_id"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<InputFacet
						compact={compact}
						span="half"
						label="设备"
						field="mcap.device_id"
						placeholder="device_id"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<InputFacet
						compact={compact}
						span="half"
						label="场景 ID"
						field="mcap.scene_id"
						placeholder="scene_id"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<RangeFacet
						compact={compact}
						span="half"
						label="时长 (毫秒)"
						field="duration_ms"
						rangeDrafts={rangeDrafts}
						onRangeDraftChange={onRangeDraftChange}
						onApplyRange={onApplyRange}
					/>
					<DateRangeFacet
						compact={compact}
						span="half"
						label="创建时间"
						field="created_at"
						dateDrafts={dateDrafts}
						onDateDraftChange={onDateDraftChange}
						onApplyDate={onApplyDate}
					/>
					<DateRangeFacet
						compact={compact}
						span="half"
						label="更新时间"
						field="updated_at"
						dateDrafts={dateDrafts}
						onDateDraftChange={onDateDraftChange}
						onApplyDate={onApplyDate}
					/>
				</>
			),
		},
		{
			key: "algorithm",
			label: GROUP_LABELS.algorithm,
			children: (
				<CheckableTagFacet
					label="算法状态"
					field="algo_status"
					options={ALGO_STATUS_OPTIONS}
					activeFilters={activeFilters}
					onToggleFacet={onToggleFacet}
					counts={fieldCounts.algo_status}
					span="full"
				/>
			),
		},
		{
			key: "delivery",
			label: GROUP_LABELS.delivery,
			children: (
				<>
					<SwitchFacet
						compact={compact}
						span="half"
						label="有交付"
						field="has:delivery"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
					<RangeFacet
						compact={compact}
						span="half"
						label="交付次数"
						field="delivery_count"
						rangeDrafts={rangeDrafts}
						onRangeDraftChange={onRangeDraftChange}
						onApplyRange={onApplyRange}
					/>
				</>
			),
		},
		{
			key: "tags",
			label: GROUP_LABELS.tags,
			children: (
				<>
					<CheckboxFacet
						compact={compact}
						label="优先级"
						field="tag.priority"
						options={PRIORITY_OPTIONS}
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
						counts={fieldCounts["tag.priority"]}
					/>
					<CheckboxFacet
						compact={compact}
						label="质量"
						field="tag.quality"
						options={QUALITY_OPTIONS}
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
						counts={fieldCounts["tag.quality"]}
					/>
					<InputFacet
						compact={compact}
						span="half"
						label="场景标签"
						field="tags.scene"
						placeholder="eq 操作在关键词模式下走 nested"
						activeFilters={activeFilters}
						onToggleFacet={onToggleFacet}
					/>
				</>
			),
		},
	];

	const horizontalItems = items.map((item) => {
		const groupKey = item.key as string;
		const isExpanded = expandedGroups.includes(groupKey);
		const activeCount = activeFilters.filter((chip) =>
			GROUP_FIELDS[item.key]?.includes(chip.field),
		).length;

		return {
			...item,
			groupKey,
			isExpanded,
			activeCount,
		};
	});

	if (layout === "horizontal") {
		const expandedGroupCount = horizontalItems.filter(
			(item) =>
				item.isExpanded &&
				!QUICK_FILTER_GROUPS.includes(item.key) &&
				item.key !== "frequent",
		).length;
		const quickGroup = horizontalItems.find((item) => item.key === "quick");
		const frequentGroup = horizontalItems.find(
			(item) => item.key === "frequent",
		);
		const otherGroups = horizontalItems.filter(
			(item) =>
				!QUICK_FILTER_GROUPS.includes(item.key) && item.key !== "frequent",
		);

		return (
			<div
				style={{
					display: "flex",
					flexDirection: "column",
					gap: compact ? 8 : 12,
				}}
			>
				{quickGroup && (
					<div
						style={{
							display: "grid",
							gridTemplateColumns: "repeat(auto-fit, minmax(200px, 280px))",
							justifyContent: "start",
							gap: compact ? 10 : 12,
							alignItems: "start",
							padding: compact ? "10px 12px" : 12,
							border: "1px solid #e5e7eb",
							borderRadius: 10,
							background: "#fff",
							boxShadow: "0 1px 2px rgba(15, 23, 42, 0.04)",
						}}
					>
						{quickGroup.children}
					</div>
				)}

				{frequentGroup && (
					<div
						style={{
							display: "grid",
							gridTemplateColumns: "repeat(auto-fit, minmax(200px, 280px))",
							justifyContent: "start",
							gap: compact ? 10 : 12,
							alignItems: "start",
							padding: compact ? "10px 12px" : 12,
							border: "1px solid #e5e7eb",
							borderRadius: 10,
							background: "#fff",
							boxShadow: "0 1px 2px rgba(15, 23, 42, 0.04)",
						}}
					>
						{frequentGroup.children}
					</div>
				)}

				<div
					style={{
						display: "flex",
						flexWrap: "wrap",
						gap: 8,
						alignItems: "center",
					}}
				>
					{otherGroups.map((item) => (
						<Badge
							key={item.key}
							count={item.activeCount}
							size="small"
							offset={[-6, 6]}
						>
							<Button
								size="small"
								type={item.activeCount > 0 ? "primary" : "default"}
								ghost={item.activeCount > 0}
								onClick={() => onToggleGroup(item.groupKey)}
								style={{
									borderRadius: 999,
									paddingInline: 10,
									height: 28,
								}}
							>
								{item.isExpanded ? "▾ " : "▸ "}
								{typeof item.label === "string"
									? item.label
									: GROUP_LABELS[item.key]}
							</Button>
						</Badge>
					))}
				</div>

				{expandedGroupCount > 0 && (
					<div
						style={{
							display: "grid",
							gridTemplateColumns: "repeat(auto-fit, minmax(240px, 280px))",
							justifyContent: "start",
							gap: compact ? 10 : 12,
							alignItems: "start",
							maxHeight: compact ? 320 : undefined,
							overflowY: compact ? "auto" : undefined,
							paddingRight: compact ? 4 : 0,
						}}
					>
						{otherGroups
							.filter((item) => item.isExpanded)
							.map((item) => (
								<section
									key={item.key}
									style={{
										minWidth: 0,
										padding: compact ? "10px 12px" : 12,
										border: "1px solid #e5e7eb",
										borderRadius: 10,
										background: "#fff",
										boxShadow: "0 1px 2px rgba(15, 23, 42, 0.04)",
									}}
								>
									<div
										style={{
											display: "flex",
											alignItems: "center",
											justifyContent: "space-between",
											gap: 8,
											marginBottom: 8,
										}}
									>
										<Text strong style={{ fontSize: 12, color: "#334155" }}>
											{typeof item.label === "string"
												? item.label
												: GROUP_LABELS[item.key]}
										</Text>
										{item.activeCount > 0 && (
											<Text type="secondary" style={{ fontSize: 11 }}>
												{item.activeCount} 已选
											</Text>
										)}
									</div>
									<div
										style={{
											display: "grid",
											gridTemplateColumns:
												"repeat(auto-fit, minmax(min(280px, 100%), 1fr))",
											gap: compact ? 8 : 12,
											alignItems: "start",
										}}
									>
										{item.children}
									</div>
								</section>
							))}
					</div>
				)}
			</div>
		);
	}

	return (
		<Collapse
			activeKey={expandedGroups}
			onChange={(keys) => {
				const newKeys = Array.isArray(keys) ? keys : [keys];
				// Determine which group was toggled
				const allGroups = GROUP_KEYS as readonly string[];
				for (const g of allGroups) {
					const wasExpanded = expandedGroups.includes(g);
					const isExpanded = newKeys.includes(g);
					if (wasExpanded !== isExpanded) {
						onToggleGroup(g);
					}
				}
			}}
			size="small"
			bordered={false}
			items={items}
			style={{ background: "transparent" }}
		/>
	);
}
