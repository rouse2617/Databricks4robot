// ─── BulkActionBar — Bulk action toolbar above results table ───
// Renders when ≥1 row is selected. Shows selection count, action buttons,
// "select all filtered" link, and a warning for large batch operations.
// Validates: Requirements R9

import {
	DeleteOutlined,
	ExportOutlined,
	ForkOutlined,
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
	onRunPipeline: () => void;
	/** When true, the Run Pipeline button is rendered as disabled with an explanatory tooltip */
	disabledRunPipeline?: boolean;
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
	onRunPipeline,
	disabledRunPipeline = false,
	onBatchTag,
	onBatchDeleteTag,
	onExportIds,
	onSelectAllFiltered,
	onClearSelection,
}: BulkActionBarProps) {
	if (selectedCount === 0) return null;

	return (
		<div
			className="assets-bulk-action-bar"
			style={{
				background: "#e6f7ff",
				border: "1px solid #91d5ff",
				borderRadius: 4,
				padding: "8px 16px",
				marginBottom: 8,
			}}
		>
			<div className="assets-bulk-action-bar__content">
				<Text strong className="assets-bulk-action-bar__count">
					已选 {selectedCount} 条
				</Text>
				<Space
					size={8}
					wrap
					align="center"
					className="assets-bulk-action-bar__actions"
				>
					<Button
						size="small"
						icon={<SendOutlined />}
						onClick={onCreateDelivery}
					>
						创建交付
					</Button>
					<Tooltip
						title={
							disabledRunPipeline
								? "当前选择无法直接带入 Pipeline，请改为逐行选择资产"
								: "用已选资产运行流水线"
						}
						placement="top"
					>
						<Button
							size="small"
							icon={<ForkOutlined />}
							onClick={disabledRunPipeline ? undefined : onRunPipeline}
							disabled={disabledRunPipeline}
						>
							运行 Pipeline
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
				</Space>

				{selectionMode === "explicit_rows" && totalFiltered > selectedCount && (
					<Tooltip title="会把当前筛选命中的所有资产都加入本次批量操作，请确认筛选条件正确。">
						<Button
							size="small"
							type="link"
							danger={totalFiltered > 1000}
							onClick={onSelectAllFiltered}
							className="assets-bulk-action-bar__select-all"
						>
							选择全部 {totalFiltered} 条筛选结果
						</Button>
					</Tooltip>
				)}
			</div>

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
