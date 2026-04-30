// ─── ExpiringAssetsChip — Quick filter for assets expiring within 30 days ───
// Task 8.4: "30 天内将过期" quick chip

import { Tag } from "antd";
import { ClockCircleOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { createFilterChip } from "../../lib/assets/assetsDiscoveryActions";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

export interface ExpiringAssetsChipProps {
  activeFilters: FilterChip[];
  onAddFilter: (chip: FilterChip) => void;
  onRemoveFilter: (id: string) => void;
}

const EXPIRING_FIELD = "expire_at";
const EXPIRING_OP = "between";

function isExpiringChip(chip: FilterChip): boolean {
  return chip.field === EXPIRING_FIELD && chip.op === EXPIRING_OP && chip.source === "add_filter";
}

export default function ExpiringAssetsChip({
  activeFilters,
  onAddFilter,
  onRemoveFilter,
}: ExpiringAssetsChipProps) {
  const existingChip = activeFilters.find(isExpiringChip);
  const isActive = !!existingChip;

  const handleClick = () => {
    if (isActive && existingChip) {
      onRemoveFilter(existingChip.id);
    } else {
      const now = dayjs().toISOString();
      const in30Days = dayjs().add(30, "day").toISOString();
      const chip = createFilterChip(EXPIRING_FIELD, EXPIRING_OP, `${now},${in30Days}`, "add_filter");
      onAddFilter(chip);
    }
  };

  return (
    <Tag
      icon={<ClockCircleOutlined />}
      color={isActive ? "warning" : undefined}
      style={{ cursor: "pointer", userSelect: "none" }}
      onClick={handleClick}
      data-testid="expiring-assets-chip"
      aria-pressed={isActive}
      aria-label="30 天内将过期"
    >
      30 天内将过期
    </Tag>
  );
}
