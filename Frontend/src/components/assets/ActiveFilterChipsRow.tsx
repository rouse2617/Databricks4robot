// ─── ActiveFilterChipsRow — Presentational Component ───
// Renders active filter chips with remove/clear-all actions.
// Validates: Requirements R3

import { Tag, Button } from "antd";
import { CloseCircleOutlined } from "@ant-design/icons";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

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
}

// ─── Component ───

export default function ActiveFilterChipsRow({
  chips,
  onRemoveChip,
  onClearAll,
}: ActiveFilterChipsRowProps) {
  if (chips.length === 0) return null;

  return (
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
      <Button
        type="link"
        size="small"
        icon={<CloseCircleOutlined />}
        onClick={onClearAll}
      >
        清除全部
      </Button>
    </div>
  );
}
