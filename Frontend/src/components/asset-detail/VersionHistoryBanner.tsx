import { Alert, Button, Typography } from "antd";
import type { RevisionSummary } from "../../api/types";

const { Text } = Typography;

export interface VersionHistoryBannerProps {
	revisions: RevisionSummary[];
	currentAssetId: string;
	onJumpToCurrent: (currentAssetId: string) => void;
}

export default function VersionHistoryBanner({
	revisions,
	currentAssetId,
	onJumpToCurrent,
}: VersionHistoryBannerProps) {
	const viewing = revisions.find((r) => r.asset_id === currentAssetId);
	const current = revisions.find((r) => r.is_current);

	if (!viewing || !current || viewing.is_current) {
		return null;
	}

	return (
		<Alert
			type="warning"
			showIcon
			className="mb-4"
			message={
				<div className="flex items-center justify-between gap-4 flex-wrap">
					<div className="flex items-center gap-2 min-w-0 flex-wrap">
						<Text className="text-sm">
							正在查看历史版本
						</Text>
						<Text code className="text-xs font-mono">
							v{viewing.revision}
						</Text>
						<Text type="secondary" className="text-xs">
							→ 当前版本
						</Text>
						<Text code className="text-xs font-mono">
							v{current.revision}
						</Text>
					</div>
					<Button
						size="middle"
						className="shrink-0"
						onClick={() => onJumpToCurrent(current.asset_id)}
					>
						跳转到当前版
					</Button>
				</div>
			}
		/>
	);
}
