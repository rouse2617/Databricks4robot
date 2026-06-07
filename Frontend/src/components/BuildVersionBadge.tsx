import { InfoCircleOutlined } from "@ant-design/icons";
import { Popover, Typography } from "antd";
import { getAppBuildBadgeLabel, getAppVersionInfo } from "../lib/appVersion";

const { Text } = Typography;

interface BuildVersionBadgeProps {
	offsetLeft?: number;
	variant?: "floating" | "sidebar";
}

export default function BuildVersionBadge({
	offsetLeft,
	variant = "floating",
}: BuildVersionBadgeProps): React.JSX.Element {
	const versionInfo = getAppVersionInfo();
	const label = getAppBuildBadgeLabel();
	const className =
		variant === "sidebar"
			? "app-build-badge app-build-badge--sidebar"
			: "app-build-badge";

	return (
		<Popover
			placement="topRight"
			trigger={["hover", "click"]}
			content={
				<div className="app-build-badge__details">
					<div>
						<Text type="secondary">版本</Text>
						<Text code copyable>
							{versionInfo.version}
						</Text>
					</div>
					<div>
						<Text type="secondary">构建</Text>
						<Text code copyable>
							{versionInfo.buildRef}
						</Text>
					</div>
					<div>
						<Text type="secondary">环境</Text>
						<Text code>{versionInfo.environment}</Text>
					</div>
				</div>
			}
		>
			<button
				type="button"
				className={className}
				aria-label={`前端版本 ${label}`}
				title={label}
				style={
					variant === "floating" && offsetLeft
						? { left: offsetLeft }
						: undefined
				}
			>
				<InfoCircleOutlined aria-hidden="true" />
				<span>{label}</span>
			</button>
		</Popover>
	);
}
