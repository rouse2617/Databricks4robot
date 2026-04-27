// ─── AssetsFacetSidebar — Presentational Component ───
// 5 collapsible facet groups for the assets discovery workbench.
// Validates: Requirements R4

import {
  Collapse,
  Checkbox,
  Input,
  InputNumber,
  DatePicker,
  Switch,
  Button,
  Typography,
} from "antd";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

// ─── Facet Option Constants ───

export const STATUS_OPTIONS = ["approved", "rejected", "superseded", "archived"];
export const ENV_OPTIONS = ["kitchen", "outdoor", "warehouse", "office", "factory"];
export const ALGO_STATUS_OPTIONS = ["ok", "failed", "running", "pending", "blocked"];
export const PRIORITY_OPTIONS = ["critical", "high", "medium", "low"];
export const QUALITY_OPTIONS = ["excellent", "good", "acceptable", "poor", "unusable"];

export const GROUP_KEYS = ["basic", "capture", "algorithm", "delivery", "tags"] as const;

// ─── Props ───

/** Aggregation key mapping from ES agg names to facet field names. */
const AGG_KEY_MAP: Record<string, string> = {
  status_agg: "status",
  env_agg: "env",
  owner_agg: "owner",
  task_agg: "task",
};

export interface AssetsFacetSidebarProps {
  activeFilters: FilterChip[];
  expandedGroups: string[];
  rangeDrafts: Record<string, { min?: number; max?: number }>;
  dateDrafts: Record<string, { start?: string; end?: string }>;
  aggregations?: Record<string, { key: string; doc_count: number }[]>;
  onToggleFacet: (field: string, value: string) => void;
  onApplyRange: (field: string, min?: number, max?: number) => void;
  onApplyDate: (field: string, start?: string, end?: string) => void;
  onRangeDraftChange: (field: string, min?: number, max?: number) => void;
  onDateDraftChange: (field: string, start?: string, end?: string) => void;
  onToggleGroup: (group: string) => void;
}

// ─── Helpers ───

/** Extract checked values for a given field from active filters. */
export function getCheckedValues(filters: FilterChip[], field: string): string[] {
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
  return Array.isArray(chip.value) ? chip.value[0] ?? "" : chip.value;
}

// ─── Sub-components for each facet type ───

function CheckboxFacet({
  label,
  field,
  options,
  activeFilters,
  onToggleFacet,
  counts,
}: {
  label: string;
  field: string;
  options: string[];
  activeFilters: FilterChip[];
  onToggleFacet: (field: string, value: string) => void;
  counts?: Record<string, number>;
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
    <div style={{ marginBottom: 12 }}>
      <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
        {label}
      </Text>
      <Checkbox.Group
        value={checked}
        onChange={(newValues) => {
          const added = (newValues as string[]).filter((v) => !checked.includes(v));
          const removed = checked.filter((v) => !(newValues as string[]).includes(v));
          for (const v of added) onToggleFacet(field, v);
          for (const v of removed) onToggleFacet(field, v);
        }}
        options={optionsWithLabels}
        style={{ display: "flex", flexDirection: "column", gap: 4 }}
      />
    </div>
  );
}

function InputFacet({
  label,
  field,
  placeholder,
  activeFilters,
  onToggleFacet,
}: {
  label: string;
  field: string;
  placeholder: string;
  activeFilters: FilterChip[];
  onToggleFacet: (field: string, value: string) => void;
}) {
  const current = getInputValue(activeFilters, field);
  return (
    <div style={{ marginBottom: 12 }}>
      <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
        {label}
      </Text>
      <Input
        size="small"
        placeholder={placeholder}
        value={current}
        onPressEnter={(e) => {
          const val = (e.target as HTMLInputElement).value.trim();
          if (val) onToggleFacet(field, val);
        }}
        allowClear
        onChange={(e) => {
          if (!e.target.value && current) {
            onToggleFacet(field, current);
          }
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
}: {
  label: string;
  field: string;
  rangeDrafts: Record<string, { min?: number; max?: number }>;
  onRangeDraftChange: (field: string, min?: number, max?: number) => void;
  onApplyRange: (field: string, min?: number, max?: number) => void;
}) {
  const draft = rangeDrafts[field] ?? {};
  return (
    <div style={{ marginBottom: 12 }}>
      <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
        {label}
      </Text>
      <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
        <InputNumber
          size="small"
          placeholder="Min"
          value={draft.min}
          onChange={(v) => onRangeDraftChange(field, v ?? undefined, draft.max)}
          style={{ width: 70 }}
        />
        <span>–</span>
        <InputNumber
          size="small"
          placeholder="Max"
          value={draft.max}
          onChange={(v) => onRangeDraftChange(field, draft.min, v ?? undefined)}
          style={{ width: 70 }}
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
}: {
  label: string;
  field: string;
  dateDrafts: Record<string, { start?: string; end?: string }>;
  onDateDraftChange: (field: string, start?: string, end?: string) => void;
  onApplyDate: (field: string, start?: string, end?: string) => void;
}) {
  const draft = dateDrafts[field] ?? {};
  return (
    <div style={{ marginBottom: 12 }}>
      <Text type="secondary" style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
        {label}
      </Text>
      <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
        <DatePicker.RangePicker
          size="small"
          onChange={(_dates, dateStrings) => {
            const [start, end] = dateStrings;
            onDateDraftChange(field, start || undefined, end || undefined);
          }}
          style={{ flex: 1 }}
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
}: {
  label: string;
  field: string;
  activeFilters: FilterChip[];
  onToggleFacet: (field: string, value: string) => void;
}) {
  const active = isSwitchActive(activeFilters, field);
  return (
    <div style={{ marginBottom: 12, display: "flex", alignItems: "center", gap: 8 }}>
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
  basic: "基础",
  capture: "采集",
  algorithm: "算法",
  delivery: "交付",
  tags: "标签",
};

// ─── Main Component ───

export default function AssetsFacetSidebar({
  activeFilters,
  expandedGroups,
  rangeDrafts,
  dateDrafts,
  aggregations,
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
      key: "basic",
      label: GROUP_LABELS.basic,
      children: (
        <>
          <CheckboxFacet
            label="状态"
            field="status"
            options={STATUS_OPTIONS}
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
            counts={fieldCounts["status"]}
          />
          <InputFacet
            label="Owner"
            field="owner"
            placeholder="搜索 owner"
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
          />
          <InputFacet
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
          <CheckboxFacet
            label="环境"
            field="env"
            options={ENV_OPTIONS}
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
            counts={fieldCounts["env"]}
          />
          <RangeFacet
            label="时长 (秒)"
            field="duration_sec"
            rangeDrafts={rangeDrafts}
            onRangeDraftChange={onRangeDraftChange}
            onApplyRange={onApplyRange}
          />
          <DateRangeFacet
            label="创建时间"
            field="created_at"
            dateDrafts={dateDrafts}
            onDateDraftChange={onDateDraftChange}
            onApplyDate={onApplyDate}
          />
          <DateRangeFacet
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
        <CheckboxFacet
          label="算法状态"
          field="algo_status"
          options={ALGO_STATUS_OPTIONS}
          activeFilters={activeFilters}
          onToggleFacet={onToggleFacet}
        />
      ),
    },
    {
      key: "delivery",
      label: GROUP_LABELS.delivery,
      children: (
        <>
          <SwitchFacet
            label="有交付"
            field="has_delivery"
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
          />
          <RangeFacet
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
            label="优先级"
            field="tag.priority"
            options={PRIORITY_OPTIONS}
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
          />
          <CheckboxFacet
            label="质量"
            field="tag.quality"
            options={QUALITY_OPTIONS}
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
          />
          <InputFacet
            label="备注"
            field="tag.notes"
            placeholder="搜索备注"
            activeFilters={activeFilters}
            onToggleFacet={onToggleFacet}
          />
        </>
      ),
    },
  ];

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
