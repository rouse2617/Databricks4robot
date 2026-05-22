import {
	CheckCircleFilled,
	CloseCircleFilled,
	LockFilled,
	MinusCircleFilled,
	SyncOutlined,
} from "@ant-design/icons";
import { Tooltip } from "antd";
import type { CSSProperties, ReactNode } from "react";

export type CellStatus =
	| "ok"
	| "failed"
	| "running"
	| "pending"
	| "blocked"
	| "none";

const STATUS_CONFIG: Record<
	CellStatus,
	{ color: string; bg: string; icon: ReactNode; label: string }
> = {
	ok: {
		color: "#16a34a",
		bg: "#f0fdf4",
		icon: <CheckCircleFilled style={{ color: "#16a34a", fontSize: 14 }} />,
		label: "成功",
	},
	failed: {
		color: "#dc2626",
		bg: "#fef2f2",
		icon: <CloseCircleFilled style={{ color: "#dc2626", fontSize: 14 }} />,
		label: "失败",
	},
	running: {
		color: "#d97706",
		bg: "#fffbeb",
		icon: <SyncOutlined spin style={{ color: "#d97706", fontSize: 14 }} />,
		label: "运行中",
	},
	pending: {
		color: "#64748b",
		bg: "#f8fafc",
		icon: <MinusCircleFilled style={{ color: "#64748b", fontSize: 14 }} />,
		label: "待处理",
	},
	blocked: {
		color: "#64748b",
		bg: "#f8fafc",
		icon: <LockFilled style={{ color: "#64748b", fontSize: 14 }} />,
		label: "已阻塞",
	},
	none: {
		color: "#94a3b8",
		bg: "#f1f5f9",
		icon: <MinusCircleFilled style={{ color: "#94a3b8", fontSize: 14 }} />,
		label: "无数据",
	},
};

interface AlgoStatusCellProps {
	status: CellStatus;
	onClick?: () => void;
}

export default function AlgoStatusCell({
	status,
	onClick,
}: AlgoStatusCellProps) {
	const cfg = STATUS_CONFIG[status];
	const interactive = !!onClick;

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
		<Tooltip
			title={
				status === "blocked"
					? "已阻塞（点击查看阻塞详情与建议动作）"
					: cfg.label
			}
		>
			<button
				type="button"
				style={style}
				onClick={onClick}
				aria-label={`算法状态：${cfg.label}`}
				onKeyDown={(e) => {
					if (e.key === "Enter" || e.key === " ") {
						e.preventDefault();
						onClick?.();
					}
				}}
				onMouseEnter={(e) => {
					e.currentTarget.style.boxShadow = `0 0 0 2px ${cfg.color}40`;
				}}
				onMouseLeave={(e) => {
					e.currentTarget.style.boxShadow = "none";
				}}
				aria-disabled={!interactive}
			>
				{cfg.icon}
			</button>
		</Tooltip>
	);
}
