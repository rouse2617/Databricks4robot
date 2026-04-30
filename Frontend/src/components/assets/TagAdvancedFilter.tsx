// ─── TagAdvancedFilter — Advanced tag filtering with nested conditions ───
// P2-FE-1: Supports filtering by tag key + value + operator,
// including nested tag conditions (same tag, multiple criteria).

import { useState, useCallback } from "react";
import { Button, Select, Input, Space, Card, Tag, Divider, Typography } from "antd";
import { PlusOutlined, DeleteOutlined, FilterOutlined } from "@ant-design/icons";
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

interface TagFilterGroup {
  id: string;
  conditions: TagCondition[];
}

// ─── Constants ───

const TAG_FIELDS: { value: TagCondition["field"]; label: string }[] = [
  { value: "key", label: "Tag Key" },
  { value: "value", label: "Tag Value" },
  { value: "source_type", label: "来源类型" },
  { value: "confidence", label: "置信度" },
];

const OPERATORS: { value: TagOperator; label: string; forTypes: string[] }[] = [
  { value: "eq", label: "等于", forTypes: ["key", "value", "source_type", "confidence"] },
  { value: "ne", label: "不等于", forTypes: ["key", "value", "source_type", "confidence"] },
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

function newGroup(): TagFilterGroup {
  return { id: uid(), conditions: [newCondition()] };
}

function operatorsForField(field: TagCondition["field"]): typeof OPERATORS {
  return OPERATORS.filter((op) => op.forTypes.includes(field));
}

// ─── Props ───

export interface TagAdvancedFilterProps {
  onApply: (chips: FilterChip[]) => void;
  onClose?: () => void;
}

// ─── Component ───

export default function TagAdvancedFilter({ onApply, onClose }: TagAdvancedFilterProps) {
  const [groups, setGroups] = useState<TagFilterGroup[]>([newGroup()]);

  const addGroup = useCallback(() => {
    setGroups((prev) => [...prev, newGroup()]);
  }, []);

  const removeGroup = useCallback((groupId: string) => {
    setGroups((prev) => prev.filter((g) => g.id !== groupId));
  }, []);

  const addCondition = useCallback((groupId: string) => {
    setGroups((prev) =>
      prev.map((g) =>
        g.id === groupId ? { ...g, conditions: [...g.conditions, newCondition()] } : g,
      ),
    );
  }, []);

  const removeCondition = useCallback((groupId: string, condId: string) => {
    setGroups((prev) =>
      prev.map((g) =>
        g.id === groupId
          ? { ...g, conditions: g.conditions.filter((c) => c.id !== condId) }
          : g,
      ),
    );
  }, []);

  const updateCondition = useCallback(
    (groupId: string, condId: string, patch: Partial<TagCondition>) => {
      setGroups((prev) =>
        prev.map((g) =>
          g.id === groupId
            ? {
                ...g,
                conditions: g.conditions.map((c) =>
                  c.id === condId ? { ...c, ...patch } : c,
                ),
              }
            : g,
        ),
      );
    },
    [],
  );

  const handleApply = useCallback(() => {
    const chips: FilterChip[] = [];
    for (const group of groups) {
      for (const cond of group.conditions) {
        if (!cond.value.trim()) continue;
        // Map condition to ES nested filter field
        const field =
          cond.field === "key"
            ? `tags.${cond.value.trim()}`
            : `tags.${cond.field}`;
        const value = cond.field === "key" ? cond.value.trim() : cond.value.trim();
        // For key conditions, we create a chip that pins the tag key
        if (cond.field === "key") {
          // This will be used as a tag key filter — the value is the key name
          // The actual value filter should come from another condition in the group
          continue;
        }
        // Find the key condition in the same group
        const keyCondition = group.conditions.find((c) => c.field === "key");
        const tagKey = keyCondition?.value.trim();
        if (tagKey) {
          chips.push(
            createFilterChip(`tags.${tagKey}`, cond.op, value, "add_filter"),
          );
        } else {
          chips.push(
            createFilterChip(field, cond.op, value, "add_filter"),
          );
        }
      }
    }
    onApply(chips);
    onClose?.();
  }, [groups, onApply, onClose]);

  return (
    <Card
      size="small"
      title={
        <Space>
          <FilterOutlined />
          <span>Tag 高级筛选</span>
        </Space>
      }
      extra={onClose ? <Button size="small" type="text" onClick={onClose}>关闭</Button> : null}
      style={{ width: 480 }}
    >
      {groups.map((group, gi) => (
        <div key={group.id} style={{ marginBottom: 12 }}>
          {gi > 0 && (
            <Divider style={{ margin: "8px 0" }}>
              <Tag color="blue">AND</Tag>
            </Divider>
          )}
          <div
            style={{
              border: "1px solid #f0f0f0",
              borderRadius: 6,
              padding: 8,
              background: "#fafafa",
            }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 6 }}>
              <Text type="secondary" style={{ fontSize: 11 }}>
                条件组 {gi + 1}
              </Text>
              {groups.length > 1 && (
                <Button
                  size="small"
                  type="text"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={() => removeGroup(group.id)}
                />
              )}
            </div>
            {group.conditions.map((cond, ci) => (
              <div key={cond.id} style={{ marginBottom: 4 }}>
                {ci > 0 && (
                  <Text type="secondary" style={{ fontSize: 10, display: "block", margin: "2px 0 2px 4px" }}>
                    AND
                  </Text>
                )}
                <Space size={4} wrap>
                  <Select
                    size="small"
                    value={cond.field}
                    onChange={(v) => updateCondition(group.id, cond.id, { field: v as TagCondition["field"], op: "eq", value: "" })}
                    style={{ width: 110 }}
                    options={TAG_FIELDS}
                  />
                  <Select
                    size="small"
                    value={cond.op}
                    onChange={(v) => updateCondition(group.id, cond.id, { op: v as TagOperator })}
                    style={{ width: 90 }}
                    options={operatorsForField(cond.field)}
                  />
                  {cond.field === "source_type" ? (
                    <Select
                      size="small"
                      value={cond.value || undefined}
                      onChange={(v) => updateCondition(group.id, cond.id, { value: v })}
                      style={{ width: 120 }}
                      placeholder="选择来源"
                      options={SOURCE_TYPES.map((s) => ({ value: s, label: s }))}
                    />
                  ) : (
                    <Input
                      size="small"
                      value={cond.value}
                      onChange={(e) => updateCondition(group.id, cond.id, { value: e.target.value })}
                      placeholder={cond.field === "key" ? "Tag key" : "值"}
                      style={{ width: 120 }}
                    />
                  )}
                  {group.conditions.length > 1 && (
                    <Button
                      size="small"
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={() => removeCondition(group.id, cond.id)}
                    />
                  )}
                </Space>
              </div>
            ))}
            <Button
              size="small"
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => addCondition(group.id)}
              style={{ marginTop: 4, fontSize: 11 }}
            >
              添加条件
            </Button>
          </div>
        </div>
      ))}
      <Space style={{ marginTop: 8 }}>
        <Button size="small" type="dashed" icon={<PlusOutlined />} onClick={addGroup}>
          添加条件组
        </Button>
        <Button size="small" type="primary" onClick={handleApply}>
          应用筛选
        </Button>
      </Space>
    </Card>
  );
}
