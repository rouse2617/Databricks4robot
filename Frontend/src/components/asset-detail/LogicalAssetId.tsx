import { CopyOutlined, LinkOutlined } from "@ant-design/icons";
import { Tooltip, Typography } from "antd";

const { Text } = Typography;

export interface LogicalAssetIdProps {
	logicalAssetId: string;
	/** Show "logical" prefix label before the id */
	showLabel?: boolean;
	className?: string;
}

export default function LogicalAssetId({
	logicalAssetId,
	showLabel = true,
	className,
}: LogicalAssetIdProps) {
	return (
		<span
			className={`inline-flex items-center gap-1.5 text-xs text-[#64748B] max-w-[200px] ${className ?? ""}`}
			title={logicalAssetId}
		>
			{showLabel ? (
				<span className="inline-flex items-center gap-0.5 text-[#94A3B8]">
					<LinkOutlined className="text-[10px]" />
					<span>逻辑ID</span>
				</span>
			) : null}
			<Tooltip title={logicalAssetId} placement="top">
				<Text
					copyable={{ text: logicalAssetId, icon: <CopyOutlined /> }}
					className="font-mono truncate m-0 text-xs text-[#64748B]"
				>
					{logicalAssetId}
				</Text>
			</Tooltip>
		</span>
	);
}
