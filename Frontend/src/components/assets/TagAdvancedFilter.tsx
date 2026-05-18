// ─── TagAdvancedFilter — Advanced tag filtering with nested conditions ───
// P2-FE-1: Supports filtering by tag key + value + operator,
// including nested tag conditions (same tag, multiple criteria).

import {
	DeleteOutlined,
	FilterOutlined,
	PlusOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Input, Select, Space, Typography } from "antd";
import { useCallback, useState } from "react";
import { createFilterChip } from "../../lib/assets/assetsDiscoveryActions";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

// ─── Types ───

type TagOperator = "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "between";

interface TagCondition {
	id: string;
	field: "key" | "value" | "source_type" | "confidence";
	op: TagOperator;
	value: string;
}

// ─── Constants ───

const TAG_FIELDS: { value: TagCondition["field"]; label: string }[] = [
	{ value: "key", label: "Tag Key" },
	{ value: "value", label: "Tag Value" },
	{ value: "source_type", label: "来源类型" },
	{ value: "confidence", label: "置信度" },
];

const OPERATORS: { value: TagOperator; label: string; forTypes: string[] }[] = [
	{
		value: "eq",
		label: "等于",
		forTypes: ["key", "value", "source_type", "confidence"],
	},
	{
		value: "ne",
		label: "不等于",
		forTypes: ["key", "value", "source_type", "confidence"],
	},
	{ value: "gt", label: "大于", forTypes: ["confidence"] },
	{ value: "gte", label: "大于等于", forTypes: ["confidence"] },
	{ value: "lt", label: "小于", forTypes: ["confidence"] },
	{ value: "lte", label: "小于等于", forTypes: ["confidence"] },
];

const SOURCE_TYPES = ["human", "algo", "manual", "import"];

// ─── Helpers ───

function uid(): string {
	return `${Date.now()}_${Math.random().toString(36).slice(2, 6)}`;
}

function newCondition(): TagCondition {
	return { id: uid(), field: "key", op: "eq", value: "" };
}

function operatorsForField(field: TagCondition["field"]): typeof OPERATORS {
	return OPERATORS.filter((op) => op.forTypes.includes(field));
}

export function resolveTagField(tagKey: string): string {
	// Always use tags_flat.* (ES-native, no fallback to Postgres scan).
	return `tags_flat.${tagKey}`;
}

function resolveTagConditionField(
	cond: TagCondition,
	tagKey?: string,
): string | null {
	if (cond.field === "key") {
		return null;
	}
	if (cond.field === "value") {
		return tagKey ? resolveTagField(tagKey) : "tags_flat.value";
	}
	return `tags_flat.${cond.field}`;
}

// ─── Props ───

export interface TagAdvancedFilterProps {
	onApply: (chips: FilterChip[]) => void;
	onClose?: () => void;
}

// ─── Component ───

export default function TagAdvancedFilter({
	onApply,
	onClose,
}: TagAdvancedFilterProps) {
	const [conditions, setConditions] = useState<TagCondition[]>([
		newCondition(),
	]);

	const addCondition = useCallback(() => {
		setConditions((prev) => [...prev, newCondition()]);
	}, []);

	const removeCondition = useCallback((condId: string) => {
		setConditions((prev) => prev.filter((c) => c.id !== condId));
	}, []);

	const updateCondition = useCallback(
		(condId: string, patch: Partial<TagCondition>) => {
			setConditions((prev) =>
				prev.map((c) => (c.id === condId ? { ...c, ...patch } : c)),
			);
		},
		[],
	);

	const handleApply = useCallback(() => {
		const chips: FilterChip[] = [];
		const keyCondition = conditions.find(
			(c) => c.field === "key" && c.value.trim() !== "",
		);
		const tagKey = keyCondition?.value.trim();
		let hasNonKeyCondition = false;

		for (const cond of conditions) {
			const value = cond.value.trim();
			if (!value) {
				continue;
			}
			const field = resolveTagConditionField(cond, tagKey);
			if (!field) {
				continue;
			}
			hasNonKeyCondition = true;
			chips.push(createFilterChip(field, cond.op, value, "add_filter"));
		}

		if (tagKey && !hasNonKeyCondition) {
			// Key-only fallback: allow narrowing by presence of the tag key itself.
			chips.push(createFilterChip("tags_flat.key", "eq", tagKey, "add_filter"));
		}
		onApply(chips);
		onClose?.();
	}, [conditions, onApply, onClose]);

	return (
		<Card
			size="small"
			title={
				<Space>
					<FilterOutlined />
					<span>Tag 高级筛选</span>
				</Space>
			}
			extra={
				onClose ? (
					<Button size="small" type="text" onClick={onClose}>
						关闭
					</Button>
				) : null
			}
			style={{ width: 480 }}
		>
			<Alert
				type="info"
				showIcon
				style={{ marginBottom: 12 }}
				message="当前版本只支持一个 Tag 条件组"
				description="先选 Tag Key，再叠加 value / source_type / confidence 条件。这样能和后端 nested 查询一一对应。"
			/>
			<div
				style={{
					border: "1px solid #f0f0f0",
					borderRadius: 6,
					padding: 8,
					background: "#fafafa",
				}}
			>
				<Text
					type="secondary"
					style={{ fontSize: 11, display: "block", marginBottom: 6 }}
				>
					条件组 1
				</Text>
				{conditions.map((cond, ci) => (
					<div key={cond.id} style={{ marginBottom: 4 }}>
						{ci > 0 && (
							<Text
								type="secondary"
								style={{
									fontSize: 10,
									display: "block",
									margin: "2px 0 2px 4px",
								}}
							>
								AND
							</Text>
						)}
						<Space size={4} wrap>
							<Select
								size="small"
								value={cond.field}
								onChange={(v) =>
									updateCondition(cond.id, {
										field: v as TagCondition["field"],
										op: "eq",
										value: "",
									})
								}
								style={{ width: 110 }}
								options={TAG_FIELDS}
							/>
							<Select
								size="small"
								value={cond.op}
								onChange={(v) =>
									updateCondition(cond.id, { op: v as TagOperator })
								}
								style={{ width: 90 }}
								options={operatorsForField(cond.field)}
							/>
							{cond.field === "source_type" ? (
								<Select
									size="small"
									value={cond.value || undefined}
									onChange={(v) => updateCondition(cond.id, { value: v })}
									style={{ width: 120 }}
									placeholder="选择来源"
									options={SOURCE_TYPES.map((s) => ({ value: s, label: s }))}
								/>
							) : (
								<Input
									size="small"
									value={cond.value}
									onChange={(e) =>
										updateCondition(cond.id, { value: e.target.value })
									}
									placeholder={cond.field === "key" ? "Tag key" : "值"}
									style={{ width: 120 }}
								/>
							)}
							{conditions.length > 1 && (
								<Button
									size="small"
									type="text"
									danger
									icon={<DeleteOutlined />}
									onClick={() => removeCondition(cond.id)}
								/>
							)}
						</Space>
					</div>
				))}
				<Button
					size="small"
					type="dashed"
					icon={<PlusOutlined />}
					onClick={addCondition}
					style={{ marginTop: 4, fontSize: 11 }}
				>
					添加条件
				</Button>
			</div>
			<Space style={{ marginTop: 8 }}>
				<Button size="small" type="primary" onClick={handleApply}>
					应用筛选
				</Button>
			</Space>
		</Card>
	);
}
