// ─── ActiveFilterChipsRow — Presentational Component ───
// Renders active filter chips with remove/clear-all actions.
// Validates: Requirements R3

import { Button, Space, Tag, Typography } from "antd";
import { CloseCircleOutlined } from "@ant-design/icons";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

// ─── Operator Display Map ───

const OP_LABELS: Record<string, string> = {
  eq: "=",
  ne: "≠",
  gt: ">",
  lt: "<",
  gte: "≥",
  lte: "≤",
  ilike: "~",
  between: "between",
  in: "in",
};

// ─── Chip Color Logic ───

function chipColor(chip: FilterChip): string | undefined {
  if (chip.field === "algo_status" && chip.value === "failed") return "red";
  if (chip.field === "lifecycle_state" && chip.value === "ready") return "green";
  if (chip.field === "status" && chip.value === "approved") return "green";
  return undefined; // Ant Design default
}

// ─── Display Helpers ───

export function formatOpLabel(op: string): string {
  return OP_LABELS[op] ?? op;
}

export function formatChipValue(value: string | string[]): string {
  return Array.isArray(value) ? value.join(", ") : value;
}

export function formatChipText(chip: FilterChip): string {
  return `${chip.field} ${formatOpLabel(chip.op)} ${formatChipValue(chip.value)}`;
}

// ─── Props ───

export interface ActiveFilterChipsRowProps {
  chips: FilterChip[];
  onRemoveChip: (id: string) => void;
  onClearAll: () => void;
  onClearField?: (field: string) => void;
}

// ─── Component ───

export default function ActiveFilterChipsRow({
  chips,
  onRemoveChip,
  onClearAll,
  onClearField,
}: ActiveFilterChipsRowProps) {
  if (chips.length === 0) return null;

  const uniqueFields = Array.from(new Set(chips.map((chip) => chip.field)));

  const handleClearField = (field: string) => {
    if (onClearField) {
      onClearField(field);
      return;
    }
    chips
      .filter((chip) => chip.field === field)
      .forEach((chip) => onRemoveChip(chip.id));
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <Space size={8} wrap>
        <Text type="secondary" style={{ fontSize: 12 }}>
          已应用筛选
        </Text>
        {uniqueFields.map((field) => (
          <Button
            key={field}
            type="link"
            size="small"
            onClick={() => handleClearField(field)}
            style={{ paddingInline: 0 }}
          >
            清空 {field}
          </Button>
        ))}
        <Button
          type="link"
          size="small"
          icon={<CloseCircleOutlined />}
          onClick={onClearAll}
          style={{ paddingInline: 0 }}
        >
          清除全部
        </Button>
      </Space>
      <div style={{ display: "flex", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
        {chips.map((chip) => (
          <Tag
            key={chip.id}
            closable
            color={chipColor(chip)}
            onClose={() => onRemoveChip(chip.id)}
            aria-label={`移除筛选: ${formatChipText(chip)}`}
          >
            {formatChipText(chip)}
          </Tag>
        ))}
      </div>
    </div>
  );
}
