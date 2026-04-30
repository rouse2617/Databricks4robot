import { Tag, Typography } from "antd";
import {
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  FlagOutlined,
  InboxOutlined,
} from "@ant-design/icons";
import type { ReactNode } from "react";
import { createFilterChip } from "../../lib/assets/assetsDiscoveryActions";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

interface QuickFilterDef {
  field: string;
  op: string;
  value: string;
  label: string;
  icon: ReactNode;
  color?: string;
}

const QUICK_FILTERS: QuickFilterDef[] = [
  {
    field: "lifecycle_state",
    op: "eq",
    value: "ready",
    label: "Ready 资产",
    icon: <CheckCircleOutlined />,
    color: "green",
  },
  {
    field: "algo_status",
    op: "eq",
    value: "failed",
    label: "算法失败",
    icon: <ExclamationCircleOutlined />,
    color: "red",
  },
  {
    field: "tags_flat.priority",
    op: "eq",
    value: "high",
    label: "高优先级",
    icon: <FlagOutlined />,
    color: "orange",
  },
  {
    field: "has:delivery",
    op: "eq",
    value: "false",
    label: "未交付",
    icon: <InboxOutlined />,
    color: "blue",
  },
];

function isSameQuickFilter(chip: FilterChip, def: QuickFilterDef): boolean {
  return chip.field === def.field && chip.op === def.op && chip.value === def.value;
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
    <div style={{ display: "flex", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
      <Text type="secondary" style={{ fontSize: 12 }}>
        快捷筛选
      </Text>
      {QUICK_FILTERS.map((def) => {
        const matched = activeFilters.filter((chip) => isSameQuickFilter(chip, def));
        const isActive = matched.length > 0;

        return (
          <Tag
            key={`${def.field}:${def.op}:${def.value}`}
            icon={def.icon}
            color={isActive ? def.color : undefined}
            style={{ cursor: "pointer", userSelect: "none", marginInlineEnd: 0 }}
            aria-pressed={isActive}
            onClick={() => {
              if (isActive) {
                matched.forEach((chip) => onRemoveFilter(chip.id));
                return;
              }
              onAddFilter(createFilterChip(def.field, def.op, def.value, "add_filter"));
            }}
          >
            {def.label}
          </Tag>
        );
      })}
    </div>
  );
}
