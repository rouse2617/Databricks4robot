// ─── SavedViewSelector — Dropdown for built-in + custom saved views ───
// Validates: Requirements R11

import { Select, Button, Space, Tooltip } from "antd";
import { DeleteOutlined, SaveOutlined } from "@ant-design/icons";
import type { SavedView } from "../../lib/assets/assetsDiscoveryTypes";

export interface SavedViewSelectorProps {
  views: SavedView[];
  currentViewId: string | null;
  onSelectView: (viewId: string) => void;
  onDeleteView: (viewId: string) => void;
  onOpenSaveDialog: () => void;
}

export default function SavedViewSelector({
  views,
  currentViewId,
  onSelectView,
  onDeleteView,
  onOpenSaveDialog,
}: SavedViewSelectorProps) {
  return (
    <Space size={4}>
      <Select
        value={currentViewId ?? undefined}
        onChange={onSelectView}
        placeholder="选择视图"
        allowClear
        size="small"
        style={{ width: 160 }}
        options={views.map((v) => ({
          value: v.id,
          label: (
            <span style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <span>{v.name}</span>
              {!v.builtin && (
                <DeleteOutlined
                  style={{ color: "#999", fontSize: 11, marginLeft: 8 }}
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteView(v.id);
                  }}
                />
              )}
            </span>
          ),
        }))}
        optionRender={(option) => option.label}
      />
      <Tooltip title="保存当前视图">
        <Button
          size="small"
          icon={<SaveOutlined />}
          onClick={onOpenSaveDialog}
        />
      </Tooltip>
    </Space>
  );
}
