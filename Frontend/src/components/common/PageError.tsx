// Page-level error alert that surfaces the backend error envelope
// (code + message + request_id). Use for top-of-page failure messaging;
// inline failures inside a card should keep using `Alert` directly.

import { CopyOutlined } from "@ant-design/icons";
import { Alert, Button, message, Space, Tooltip, Typography } from "antd";
import { describeApiError } from "../../lib/apiError";

const { Text } = Typography;

export interface PageErrorProps {
	title?: string;
	error: unknown;
	onRetry?: () => void;
}

export default function PageError({
	title = "加载失败",
	error,
	onRetry,
}: PageErrorProps) {
	const described = describeApiError(error, title);

	const copyRequestId = async () => {
		if (!described.requestId) return;
		try {
			await navigator.clipboard.writeText(described.requestId);
			message.success("已复制 Request ID");
		} catch {
			message.error("复制失败");
		}
	};

	return (
		<Alert
			type="error"
			showIcon
			message={title}
			description={
				<Space direction="vertical" size={4} style={{ width: "100%" }}>
					<Text>{described.message}</Text>
					{(described.code || described.requestId) && (
						<Space size={8} wrap>
							{described.code && (
								<Text type="secondary" style={{ fontSize: 12 }}>
									code: <span className="font-mono">{described.code}</span>
								</Text>
							)}
							{described.requestId && (
								<Tooltip title="复制 Request ID 便于排查">
									<button
										type="button"
										className="link-like-button font-mono"
										aria-label={`复制 Request ID ${described.requestId}`}
										style={{ fontSize: 12 }}
										onClick={() => void copyRequestId()}
									>
										request_id: {described.requestId} <CopyOutlined />
									</button>
								</Tooltip>
							)}
						</Space>
					)}
					{onRetry && (
						<Button size="small" onClick={onRetry} style={{ marginTop: 4 }}>
							重试
						</Button>
					)}
				</Space>
			}
		/>
	);
}
