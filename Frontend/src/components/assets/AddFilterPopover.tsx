// ─── AddFilterPopover — Field/Operator/Value filter builder ───
// Validates: Requirements R12

import { useState, useMemo } from "react";
import { Popover, Button, Select, Input, Space, Tag } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { createFilterChip } from "../../lib/assets/assetsDiscoveryActions";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";

// ─── Field Metadata ───

type FieldType = "enum" | "numeric" | "string" | "timestamp";

interface FieldMeta {
  key: string;
  label: string;
  type: FieldType;
  values?: string[];
}

const SEARCHABLE_FIELDS: FieldMeta[] = [
  { key: "status", label: "状态", type: "enum", values: ["approved", "rejected", "superseded", "archived"] },
  { key: "env", label: "环境", type: "enum", values: ["kitchen", "outdoor", "warehouse", "office", "factory"] },
  { key: "algo_status", label: "算法状态", type: "enum", values: ["ok", "failed", "running", "pending", "blocked"] },
  { key: "tag.priority", label: "优先级", type: "enum", values: ["critical", "high", "medium", "low"] },
  { key: "tag.quality", label: "质量", type: "enum", values: ["excellent", "good", "acceptable", "poor", "unusable"] },
  { key: "duration_sec", label: "时长(秒)", type: "numeric" },
  { key: "delivery_count", label: "交付次数", type: "numeric" },
  { key: "asset_id", label: "Asset ID", type: "string" },
  { key: "mcap_file_id", label: "MCAP ID", type: "string" },
  { key: "owner", label: "Owner", type: "string" },
  { key: "reviewer", label: "Reviewer", type: "string" },
  { key: "tag.notes", label: "备注", type: "string" },
  { key: "created_at", label: "创建时间", type: "timestamp" },
  { key: "updated_at", label: "更新时间", type: "timestamp" },
];

const QUICK_FIELDS = ["status", "owner", "tag.priority", "algo_status", "duration_sec", "env"];

// ─── Operator options by field type ───

function getOperators(type: FieldType): { value: string; label: string }[] {
  switch (type) {
    case "enum":
      return [
        { value: "eq", label: "等于" },
        { value: "ne", label: "不等于" },
        { value: "in", label: "包含(多选)" },
      ];
    case "numeric":
      return [
        { value: "eq", label: "等于" },
        { value: "gt", label: "大于" },
        { value: "lt", label: "小于" },
        { value: "gte", label: "大于等于" },
        { value: "lte", label: "小于等于" },
        { value: "between", label: "范围" },
      ];
    case "string":
      return [
        { value: "eq", label: "等于" },
        { value: "ilike", label: "包含" },
      ];
    case "timestamp":
      return [
        { value: "eq", label: "等于" },
        { value: "gt", label: "晚于" },
        { value: "lt", label: "早于" },
      ];
  }
}

// ─── Props ───

export interface AddFilterPopoverProps {
  onAddFilter: (chip: FilterChip) => void;
}

// ─── Component ───

export default function AddFilterPopover({ onAddFilter }: AddFilterPopoverProps) {
  const [open, setOpen] = useState(false);
  const [selectedField, setSelectedField] = useState<string | null>(null);
  const [selectedOp, setSelectedOp] = useState<string | null>(null);
  const [value, setValue] = useState<string>("");

  const fieldMeta = useMemo(
    () => SEARCHABLE_FIELDS.find((f) => f.key === selectedField) ?? null,
    [selectedField],
  );

  const operators = useMemo(
    () => (fieldMeta ? getOperators(fieldMeta.type) : []),
    [fieldMeta],
  );

  const reset = () => {
    setSelectedField(null);
    setSelectedOp(null);
    setValue("");
  };

  const handleSubmit = () => {
    if (!selectedField || !selectedOp || !value.trim()) return;
    const chip = createFilterChip(selectedField, selectedOp, value.trim(), "add_filter");
    onAddFilter(chip);
    reset();
    setOpen(false);
  };

  const handleCancel = () => {
    reset();
    setOpen(false);
  };

  const handleFieldChange = (key: string) => {
    setSelectedField(key);
    setSelectedOp(null);
    setValue("");
  };

  const content = (
    <div style={{ width: 300 }}>
      <Space direction="vertical" style={{ width: "100%" }} size={8}>
        {/* Field selector */}
        <Select
          placeholder="选择字段"
          value={selectedField}
          onChange={handleFieldChange}
          style={{ width: "100%" }}
          size="small"
          showSearch
          optionFilterProp="label"
          options={SEARCHABLE_FIELDS.map((f) => ({ value: f.key, label: `${f.label} (${f.key})` }))}
        />

        {/* Operator selector */}
        {fieldMeta && (
          <Select
            placeholder="选择操作符"
            value={selectedOp}
            onChange={setSelectedOp}
            style={{ width: "100%" }}
            size="small"
            options={operators}
          />
        )}

        {/* Value input */}
        {fieldMeta && selectedOp && (
          fieldMeta.type === "enum" ? (
            <Select
              placeholder="选择值"
              value={value || undefined}
              onChange={(v) => setValue(v)}
              style={{ width: "100%" }}
              size="small"
              options={(fieldMeta.values ?? []).map((v) => ({ value: v, label: v }))}
            />
          ) : (
            <Input
              placeholder="输入值"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onPressEnter={handleSubmit}
              size="small"
              type={fieldMeta.type === "numeric" ? "number" : "text"}
            />
          )
        )}

        {/* Action buttons */}
        <Space>
          <Button size="small" type="primary" onClick={handleSubmit} disabled={!selectedField || !selectedOp || !value.trim()}>
            添加
          </Button>
          <Button size="small" onClick={handleCancel}>
            取消
          </Button>
        </Space>

        {/* Quick field buttons */}
        <div>
          <div style={{ fontSize: 11, color: "#999", marginBottom: 4 }}>快捷字段</div>
          <Space size={4} wrap>
            {QUICK_FIELDS.map((key) => {
              const meta = SEARCHABLE_FIELDS.find((f) => f.key === key);
              return meta ? (
                <Tag
                  key={key}
                  style={{ cursor: "pointer", fontSize: 11 }}
                  onClick={() => handleFieldChange(key)}
                >
                  {meta.label}
                </Tag>
              ) : null;
            })}
          </Space>
        </div>
      </Space>
    </div>
  );

  return (
    <Popover
      content={content}
      title="添加筛选条件"
      trigger="click"
      open={open}
      onOpenChange={setOpen}
      placement="bottomLeft"
    >
      <Button size="small" icon={<PlusOutlined />}>
        添加筛选
      </Button>
    </Popover>
  );
}
