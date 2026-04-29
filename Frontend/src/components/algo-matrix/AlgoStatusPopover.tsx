import { Popover, Descriptions, Button, message } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { useState, type ReactNode } from "react";
import { assetsApi } from "../../api/assets";
import dayjs from "dayjs";

interface AlgoDetail {
  status?: string;
  started_at?: string;
  finished_at?: string;
  run_id?: string;
  output_uri?: string;
  reason?: string;
}

interface AlgoStatusPopoverProps {
  assetId: string;
  algoKey: string;
  detail: AlgoDetail | null;
  children: ReactNode;
  onReset?: () => void;
}

function formatTime(v?: string) {
  if (!v) return "—";
  return dayjs(v).format("YYYY-MM-DD HH:mm:ss");
}

export default function AlgoStatusPopover({
  assetId,
  algoKey,
  detail,
  children,
  onReset,
}: AlgoStatusPopoverProps) {
  const [resetting, setResetting] = useState(false);

  const handleReset = async () => {
    setResetting(true);
    try {
      await assetsApi.resetAlgo(assetId, algoKey);
      message.success(`已重置 ${algoKey}`);
      onReset?.();
    } catch {
      message.error(`重置 ${algoKey} 失败`);
    } finally {
      setResetting(false);
    }
  };

  const content = detail ? (
    <div style={{ minWidth: 280 }}>
      <Descriptions column={1} size="small" bordered>
        <Descriptions.Item label="状态">{detail.status}</Descriptions.Item>
        <Descriptions.Item label="开始时间">
          {formatTime(detail.started_at)}
        </Descriptions.Item>
        <Descriptions.Item label="结束时间">
          {formatTime(detail.finished_at)}
        </Descriptions.Item>
        <Descriptions.Item label="Run ID">
          {detail.run_id || "—"}
        </Descriptions.Item>
        <Descriptions.Item label="输出路径">
          {detail.output_uri ? (
            <span style={{ wordBreak: "break-all", fontSize: 12 }}>
              {detail.output_uri}
            </span>
          ) : (
            "—"
          )}
        </Descriptions.Item>
        <Descriptions.Item label="原因">
          {detail.reason || "—"}
        </Descriptions.Item>
      </Descriptions>
      <div style={{ marginTop: 8, textAlign: "right" }}>
        <Button
          size="small"
          icon={<ReloadOutlined />}
          loading={resetting}
          onClick={handleReset}
        >
          重置
        </Button>
      </div>
    </div>
  ) : (
    <div style={{ color: "#999" }}>无数据</div>
  );

  return (
    <Popover
      content={content}
      title={`${algoKey} 详情`}
      trigger="click"
      placement="right"
    >
      {children}
    </Popover>
  );
}
