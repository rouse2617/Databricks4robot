import { Button, Space, Tooltip } from "antd";
import type { ReactNode } from "react";

export interface HoverAction {
	key: string;
	label: string;
	icon?: ReactNode;
	onClick: (e: React.MouseEvent) => void;
	danger?: boolean;
}

interface HoverActionBarProps {
	actions: HoverAction[];
	hovered: boolean;
	size?: "small" | "middle";
}

// CYB-4477 P2-2：行 hover 浮出的快捷操作栏。两页面（BatchJobList /
// WorkflowExecutionList）共用，默认 opacity:0，hover 时 opacity:1，
// stopPropagation 防止误触发行的 onClick（跳转行详情）。复用 antd
// Tooltip，让长 label 的批量动作在窄列里也能完整识别。
export function HoverActionBar({
	actions,
	hovered,
	size = "small",
}: HoverActionBarProps) {
	return (
		<Space
			size={4}
			onClick={(e) => e.stopPropagation()}
			style={{
				opacity: hovered ? 1 : 0,
				transition: "opacity 0.15s ease-in-out",
				pointerEvents: hovered ? "auto" : "none",
			}}
		>
			{actions.map((a) => (
				<Tooltip key={a.key} title={a.label}>
					<Button
						type="text"
						size={size}
						icon={a.icon}
						danger={a.danger}
						onClick={a.onClick}
					/>
				</Tooltip>
			))}
		</Space>
	);
}