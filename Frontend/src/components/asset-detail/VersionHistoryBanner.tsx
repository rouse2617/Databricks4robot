import { InfoCircleOutlined } from "@ant-design/icons";
import { Alert, Button } from "antd";
import type { RevisionSummary } from "../../api/types";

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
			icon={<InfoCircleOutlined />}
			className="mb-4"
			role="status"
			style={{
				background: "#FFFBEB",
				borderColor: "#FCD34D",
			}}
			message={
				<span>
					你正在查看历史版本 <strong>v{viewing.revision}</strong>（非当前）。当前有效版本为{" "}
					<strong>v{current.revision}</strong>
				</span>
			}
			action={
				<Button
					type="link"
					size="small"
					onClick={() => onJumpToCurrent(current.asset_id)}
				>
					跳转到当前版
				</Button>
			}
		/>
	);
}
