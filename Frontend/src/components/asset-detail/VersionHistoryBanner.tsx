import { ArrowRightOutlined, BranchesOutlined, ExclamationCircleFilled } from "@ant-design/icons";
import { Button, Typography } from "antd";
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
		<div
			role="status"
			className="flex items-center justify-between gap-4 px-4 py-2.5 mb-4 rounded-lg border"
			style={{
				background: "#FFFBEB",
				borderColor: "#FDE68A",
			}}
		>
			<div className="flex items-center gap-2.5 min-w-0">
				<ExclamationCircleFilled className="text-[#D97706] text-base shrink-0" />
				<div className="flex items-center gap-2 min-w-0 flex-wrap">
					<BranchesOutlined className="text-[#92400E] text-xs" />
					<Text className="text-sm text-[#92400E]">
						正在查看历史版本
					</Text>
					<span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-[#FEF3C7] text-xs font-mono text-[#92400E] font-medium">
						v{viewing.revision}
					</span>
					<ArrowRightOutlined className="text-[#D97706] text-[10px]" />
					<span className="text-sm text-[#92400E]">
						当前版本
					</span>
					<span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-[#ECFDF5] text-xs font-mono text-[#059669] font-medium">
						v{current.revision}
					</span>
				</div>
			</div>
			<Button
				size="small"
				className="shrink-0"
				style={{
					color: "#92400E",
					borderColor: "#FDE68A",
					background: "#FEF3C7",
				}}
				onClick={() => onJumpToCurrent(current.asset_id)}
			>
				跳转到当前版
			</Button>
		</div>
	);
}
