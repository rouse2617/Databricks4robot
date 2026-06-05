import { ArrowLeftOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Result, Skeleton, Space, Typography } from "antd";
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { mcapFilesApi } from "../api/mcapFiles";
import type { McapFile } from "../api/types";
import McapDetailContent, {
	McapDetailActions,
} from "../components/mcap/McapDetailContent";
import { extractApiErrorMessage } from "../lib/apiError";

const { Text, Title } = Typography;

export default function McapFileDetailPage() {
	const navigate = useNavigate();
	const { id = "" } = useParams<{ id: string }>();
	const [mcapFile, setMcapFile] = useState<McapFile | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	const load = useCallback(async () => {
		const fileId = id.trim();
		if (!fileId) {
			setMcapFile(null);
			setError("缺少 MCAP File ID");
			setLoading(false);
			return;
		}
		setLoading(true);
		setError(null);
		try {
			const next = await mcapFilesApi.get(fileId);
			setMcapFile(next);
		} catch (err) {
			setMcapFile(null);
			setError(extractApiErrorMessage(err, "加载 MCAP 文件详情失败"));
		} finally {
			setLoading(false);
		}
	}, [id]);

	useEffect(() => {
		void load();
	}, [load]);

	return (
		<div style={{ padding: 24 }}>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					gap: 16,
					marginBottom: 16,
				}}
			>
				<Space size={12}>
					<Button
						icon={<ArrowLeftOutlined />}
						onClick={() => navigate("/mcap-files")}
					>
						返回
					</Button>
					<div>
						<Title level={3} style={{ margin: 0 }}>
							MCAP 文件详情
						</Title>
						<Text type="secondary" className="font-mono text-xs">
							{id || "—"}
						</Text>
					</div>
				</Space>
				<Space>
					{mcapFile ? <McapDetailActions mcapFile={mcapFile} /> : null}
					<Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
						刷新
					</Button>
				</Space>
			</div>

			{loading ? (
				<Card>
					<Skeleton active paragraph={{ rows: 8 }} />
				</Card>
			) : error ? (
				<Result
					status="error"
					title="加载 MCAP 文件详情失败"
					subTitle={error}
					extra={
						<Button type="primary" icon={<ReloadOutlined />} onClick={load}>
							重试
						</Button>
					}
				/>
			) : mcapFile ? (
				<Card>
					<McapDetailContent mcapFile={mcapFile} />
				</Card>
			) : (
				<Alert type="warning" showIcon message="未找到 MCAP 文件详情" />
			)}
		</div>
	);
}
