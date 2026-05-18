// ─── BulkActionBar — Bulk action toolbar above results table ───
// Renders when ≥1 row is selected. Shows selection count, action buttons,
// "select all filtered" link, and a warning for large batch operations.
// Validates: Requirements R9

import {
	DeleteOutlined,
	ExportOutlined,
	RobotOutlined,
	SendOutlined,
	TagOutlined,
} from "@ant-design/icons";
import { Alert, Button, Space, Tooltip, Typography } from "antd";
import type { SelectionMode } from "../../lib/assets/assetsDiscoveryTypes";

const { Text } = Typography;

export interface BulkActionBarProps {
	selectedCount: number;
	selectionMode: SelectionMode;
	totalFiltered: number;
	onCreateDelivery: () => void;
	onRunAlgo: () => void;
	/** When true, the Run Algo button is rendered as disabled with an explanatory tooltip */
	disabledRunAlgo?: boolean;
	onBatchTag: () => void;
	onBatchDeleteTag?: () => void;
	onExportIds: () => void;
	onSelectAllFiltered: () => void;
	onClearSelection: () => void;
}

export default function BulkActionBar({
	selectedCount,
	selectionMode,
	totalFiltered,
	onCreateDelivery,
	onRunAlgo,
	disabledRunAlgo = false,
	onBatchTag,
	onBatchDeleteTag,
	onExportIds,
	onSelectAllFiltered,
	onClearSelection,
}: BulkActionBarProps) {
	if (selectedCount === 0) return null;

	return (
		<div
			style={{
				background: "#e6f7ff",
				border: "1px solid #91d5ff",
				borderRadius: 4,
				padding: "8px 16px",
				marginBottom: 8,
			}}
		>
			<Space size={12} wrap align="center">
				<Text strong>已选 {selectedCount} 条</Text>

				<Button size="small" icon={<SendOutlined />} onClick={onCreateDelivery}>
					创建交付
				</Button>
				<Tooltip
					title={disabledRunAlgo ? "批量触发算法功能开发中" : undefined}
					placement="top"
				>
					<Button
						size="small"
						icon={<RobotOutlined />}
						onClick={disabledRunAlgo ? undefined : onRunAlgo}
						disabled={disabledRunAlgo}
					>
						触发算法
					</Button>
				</Tooltip>
				<Button size="small" icon={<TagOutlined />} onClick={onBatchTag}>
					批量打 Tag
				</Button>
				{onBatchDeleteTag && (
					<Button
						size="small"
						icon={<DeleteOutlined />}
						onClick={onBatchDeleteTag}
						danger
					>
						删除 Tag
					</Button>
				)}
				<Button size="small" icon={<ExportOutlined />} onClick={onExportIds}>
					导出 ID
				</Button>
				<Button size="small" type="link" onClick={onClearSelection}>
					取消选择
				</Button>

				{selectionMode === "explicit_rows" && totalFiltered > selectedCount && (
					<button
						type="button"
						onClick={onSelectAllFiltered}
						style={{ fontSize: 13 }}
					>
						选择全部 {totalFiltered} 条筛选结果
					</button>
				)}
			</Space>

			{selectedCount > 100 && (
				<Alert
					type="warning"
					message="批量操作超过 100 条资产，请确认"
					showIcon
					style={{ marginTop: 8 }}
					banner
				/>
			)}
		</div>
	);
}
