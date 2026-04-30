// ─── ColumnsConfigPopover — Show/hide columns configuration ───
// Validates: Requirements R1

import { Popover, Button, Checkbox, Space } from "antd";
import { SettingOutlined } from "@ant-design/icons";
import { DEFAULT_COLUMNS } from "../../lib/assets/assetsDiscoveryTypes";

// ─── Available Columns ───

const ALL_COLUMNS = [
  { key: "asset_id", label: "Asset ID" },
  { key: "mcap_file_id", label: "MCAP" },
  { key: "duration", label: "时长" },
  { key: "asset_type", label: "资产类型" },
  { key: "retention_tier", label: "保留层级" },
  { key: "env", label: "环境" },
  { key: "lifecycle_state", label: "生命周期" },
  { key: "algo", label: "算法状态" },
  { key: "tags", label: "标签" },
  { key: "owner", label: "Owner" },
  { key: "expire_at", label: "过期时间" },
  { key: "updated_at", label: "更新时间" },
];

// ─── Props ───

export interface ColumnsConfigPopoverProps {
  selectedColumns: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onColumnsChange: (columns: string[]) => void;
}

// ─── Component ───

export default function ColumnsConfigPopover({
  selectedColumns,
  open,
  onOpenChange,
  onColumnsChange,
}: ColumnsConfigPopoverProps) {
  const handleToggle = (key: string, checked: boolean) => {
    if (checked) {
      // Maintain order from ALL_COLUMNS
      const newCols = ALL_COLUMNS.filter(
        (c) => selectedColumns.includes(c.key) || c.key === key,
      ).map((c) => c.key);
      onColumnsChange(newCols);
    } else {
      onColumnsChange(selectedColumns.filter((c) => c !== key));
    }
  };

  const handleRestoreDefaults = () => {
    onColumnsChange([...DEFAULT_COLUMNS]);
  };

  const content = (
    <div style={{ width: 180 }}>
      <Space direction="vertical" style={{ width: "100%" }} size={4}>
        {ALL_COLUMNS.map((col) => (
          <Checkbox
            key={col.key}
            checked={selectedColumns.includes(col.key)}
            onChange={(e) => handleToggle(col.key, e.target.checked)}
          >
            {col.label}
          </Checkbox>
        ))}
        <Button
          size="small"
          type="link"
          onClick={handleRestoreDefaults}
          style={{ padding: 0, marginTop: 4 }}
        >
          恢复默认
        </Button>
      </Space>
    </div>
  );

  return (
    <Popover
      content={content}
      title="列配置"
      trigger="click"
      open={open}
      onOpenChange={onOpenChange}
      placement="bottomRight"
    >
      <Button size="small" icon={<SettingOutlined />}>
        列配置
      </Button>
    </Popover>
  );
}
