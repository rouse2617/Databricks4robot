import { CopyOutlined } from "@ant-design/icons";
import { Typography } from "antd";

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
			className={`inline-flex items-center gap-1 text-xs text-[#64748B] max-w-[180px] ${className ?? ""}`}
			title={logicalAssetId}
		>
			{showLabel ? <span>logical</span> : null}
			<Text
				copyable={{ text: logicalAssetId, icon: <CopyOutlined /> }}
				className="font-mono truncate m-0"
			>
				{logicalAssetId}
			</Text>
		</span>
	);
}
