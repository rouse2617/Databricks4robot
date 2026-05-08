import { CopyOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Empty, message, Typography } from "antd";

const { Text } = Typography;

interface Props {
	files: Record<string, string>;
}

export default function FilesTab({ files }: Props) {
	const handleCopy = async (uri: string) => {
		try {
			await navigator.clipboard.writeText(uri);
			message.success("已复制到剪贴板");
		} catch {
			message.error("复制失败");
		}
	};

	const entries = Object.entries(files);

	return (
		<Card size="small">
			<Text type="secondary" className="text-xs mb-2 block">
				文件引用注册表
			</Text>
			{entries.length > 0 ? (
				<Descriptions bordered size="small" column={1}>
					{entries.map(([k, v]) => (
						<Descriptions.Item key={k} label={k}>
							<div className="flex items-center gap-2">
								<code className="text-xs" style={{ wordBreak: "break-all" }}>
									{v}
								</code>
								<Button
									type="text"
									size="small"
									icon={<CopyOutlined />}
									onClick={() => handleCopy(v)}
									title="复制 URI"
								/>
							</div>
						</Descriptions.Item>
					))}
				</Descriptions>
			) : (
				<Empty
					description="暂无文件引用"
					image={Empty.PRESENTED_IMAGE_SIMPLE}
				/>
			)}
		</Card>
	);
}
