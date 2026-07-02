import {
	CheckCircleOutlined,
	ExclamationCircleOutlined,
	FlagOutlined,
	InboxOutlined,
} from "@ant-design/icons";
import { Button } from "antd";
import type { ReactNode } from "react";
import { createFilterChip } from "../../lib/assets/assetsDiscoveryActions";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

interface QuickFilterDef {
	field: string;
	op: string;
	value: string;
	label: string;
	icon: ReactNode;
	color: string;
}

const QUICK_FILTERS: QuickFilterDef[] = [
	{
		field: "lifecycle_state",
		op: "eq",
		value: "ready",
		label: "已就绪",
		icon: <CheckCircleOutlined />,
		color: "#10b981",
	},
	{
		field: "algo_status",
		op: "eq",
		value: "failed",
		label: "算法失败",
		icon: <ExclamationCircleOutlined />,
		color: "#ef4444",
	},
	{
		field: "tags_flat.priority",
		op: "eq",
		value: "high",
		label: "高优先级",
		icon: <FlagOutlined />,
		color: "#f59e0b",
	},
	{
		field: "delivery_count",
		op: "eq",
		value: "0",
		label: "未交付",
		icon: <InboxOutlined />,
		color: "#3b82f6",
	},
];

function isSameQuickFilter(chip: FilterChip, def: QuickFilterDef): boolean {
	return (
		chip.field === def.field && chip.op === def.op && chip.value === def.value
	);
}

export interface QuickFiltersRowProps {
	activeFilters: FilterChip[];
	onAddFilter: (chip: FilterChip) => void;
	onRemoveFilter: (id: string) => void;
}

export default function QuickFiltersRow({
	activeFilters,
	onAddFilter,
	onRemoveFilter,
}: QuickFiltersRowProps) {
	return (
		<div
			style={{
				display: "flex",
				alignItems: "center",
				flexWrap: "wrap",
				gap: 8,
			}}
		>
			{QUICK_FILTERS.map((def) => {
				const matched = activeFilters.filter((chip) =>
					isSameQuickFilter(chip, def),
				);
				const isActive = matched.length > 0;

				return (
					<Button
						key={`${def.field}:${def.op}:${def.value}`}
						size="small"
						shape="round"
						icon={def.icon}
						type={isActive ? "primary" : "default"}
						style={
							isActive
								? {
										background: def.color,
										borderColor: def.color,
										boxShadow: "none",
									}
								: { color: "#4b5563" }
						}
						onClick={() => {
							if (isActive) {
								matched.forEach((chip) => {
									onRemoveFilter(chip.id);
								});
								return;
							}
							onAddFilter(
								createFilterChip(def.field, def.op, def.value, "add_filter"),
							);
						}}
					>
						{def.label}
					</Button>
				);
			})}
		</div>
	);
}
