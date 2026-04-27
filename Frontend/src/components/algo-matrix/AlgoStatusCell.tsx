import { Tooltip } from "antd";
import {
  CheckCircleFilled,
  CloseCircleFilled,
  SyncOutlined,
  MinusCircleFilled,
  LockFilled,
} from "@ant-design/icons";
import type { CSSProperties, ReactNode } from "react";

export type CellStatus = "ok" | "failed" | "running" | "pending" | "blocked" | "none";

const STATUS_CONFIG: Record<
  CellStatus,
  { color: string; bg: string; icon: ReactNode; label: string }
> = {
  ok: {
    color: "#52c41a",
    bg: "#f6ffed",
    icon: <CheckCircleFilled style={{ color: "#52c41a", fontSize: 14 }} />,
    label: "成功",
  },
  failed: {
    color: "#ff4d4f",
    bg: "#fff2f0",
    icon: <CloseCircleFilled style={{ color: "#ff4d4f", fontSize: 14 }} />,
    label: "失败",
  },
  running: {
    color: "#faad14",
    bg: "#fffbe6",
    icon: <SyncOutlined spin style={{ color: "#faad14", fontSize: 14 }} />,
    label: "运行中",
  },
  pending: {
    color: "#d9d9d9",
    bg: "#fafafa",
    icon: <MinusCircleFilled style={{ color: "#d9d9d9", fontSize: 14 }} />,
    label: "待处理",
  },
  blocked: {
    color: "#d9d9d9",
    bg: "#fafafa",
    icon: <LockFilled style={{ color: "#d9d9d9", fontSize: 14 }} />,
    label: "已阻塞",
  },
  none: {
    color: "#f5f5f5",
    bg: "#f5f5f5",
    icon: <MinusCircleFilled style={{ color: "#bfbfbf", fontSize: 14 }} />,
    label: "无数据",
  },
};

interface AlgoStatusCellProps {
  status: CellStatus;
  onClick?: () => void;
}

export default function AlgoStatusCell({ status, onClick }: AlgoStatusCellProps) {
  const cfg = STATUS_CONFIG[status];

  const style: CSSProperties = {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 32,
    height: 32,
    borderRadius: 6,
    backgroundColor: cfg.bg,
    border: `1px solid ${cfg.color}`,
    cursor: onClick ? "pointer" : "default",
    transition: "box-shadow 0.2s",
  };

  return (
    <Tooltip title={cfg.label}>
      <div
        role="button"
        tabIndex={0}
        style={style}
        onClick={onClick}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") onClick?.();
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLDivElement).style.boxShadow = `0 0 0 2px ${cfg.color}40`;
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLDivElement).style.boxShadow = "none";
        }}
      >
        {cfg.icon}
      </div>
    </Tooltip>
  );
}
