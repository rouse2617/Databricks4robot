import {
	DownloadOutlined,
	PauseOutlined,
	PoweroffOutlined,
	RedoOutlined,
} from "@ant-design/icons";
import { Affix, Button, Space, Typography } from "antd";
import type { BatchJob } from "../../api/batchJobApi";

const { Text } = Typography;

export type BatchAction = "retry" | "export" | "pause" | "cancel";

export interface BatchActionBarProps {
	selectedJobs: BatchJob[];
	onAction: (action: BatchAction, jobIds: string[]) => void;
	// CYB-4477 P2-1: optional busy flag — disables every button while a bulk
	// request is in flight, so the user can't enqueue a second wave while the
	// first is still resolving. Defaults to false.
	busy?: boolean;
}

/**
 * Floating batch-action bar — renders nothing when no rows are selected
 * (`selectedJobs.length === 0`) so the empty DOM stays quiet, and snaps above
 * the table (offsetTop 120 — clears the global header + tabs) once the user
 * has ≥ 1 batch ticked. Each action forwards the array of selected IDs so a
 * 25-batch "批量重试" loops through `retryFailedBatchItems` once per batch.
 *
 * The original "批量导出资产 ID" dropdown stays in the toolbar; this bar is
 * additive, not a replacement — see [[CYB-4477]] design.md.
 */
export function BatchActionBar({
	selectedJobs,
	onAction,
	busy = false,
}: BatchActionBarProps) {
	if (selectedJobs.length === 0) return null;

	const ids = selectedJobs.map((j) => j.id);
	// Spec scenario: when every selected batch is already terminal
	// (completed/failed) there is nothing to retry — disable the button so the
	// action does not look applicable.
	const retryDisabled = busy || selectedJobs.every((j) => j.failedCount === 0);

	return (
		<Affix offsetTop={120}>
			<div
				data-testid="batch-action-bar"
				style={{
					background: "#e6f4ff",
					border: "1px solid #91caff",
					borderRadius: 6,
					padding: "8px 16px",
					marginBottom: 16,
				}}
			>
				<Space>
					<Text>
						已选 <strong>{selectedJobs.length}</strong> 个批次
					</Text>
					<Button
						icon={<RedoOutlined />}
						disabled={retryDisabled}
						loading={busy}
						onClick={() => onAction("retry", ids)}
						data-testid="batch-action-retry"
					>
						批量重试
					</Button>
					<Button
						icon={<DownloadOutlined />}
						disabled={busy}
						onClick={() => onAction("export", ids)}
						data-testid="batch-action-export"
					>
						批量导出
					</Button>
					<Button
						icon={<PauseOutlined />}
						disabled={busy}
						onClick={() => onAction("pause", ids)}
						data-testid="batch-action-pause"
					>
						批量暂停
					</Button>
					<Button
						icon={<PoweroffOutlined />}
						danger
						disabled={busy}
						onClick={() => onAction("cancel", ids)}
						data-testid="batch-action-cancel"
					>
						批量取消
					</Button>
				</Space>
			</div>
		</Affix>
	);
}
