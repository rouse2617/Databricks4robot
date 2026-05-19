// ─── LineageTab — Upstream MCAP + downstream algo/delivery/eval chain ───

import {
	FileOutlined,
	FundOutlined,
	LinkOutlined,
	RobotOutlined,
	SendOutlined,
} from "@ant-design/icons";
import { Alert, Card, Descriptions, List, Spin, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { assetsApi } from "../../api/assets";

const { Text } = Typography;

interface LineageData {
	asset_id: string;
	upstream: {
		mcap_file_id?: string;
		mcap_uri?: string;
		ingest_state?: string;
	};
	downstream: {
		algo_results: Array<{
			algo_name: string;
			algo_version?: string;
			status: string;
			run_id?: string;
			output_uri?: string;
		}>;
		deliveries: Array<{
			delivery_id: string;
			customer_id?: string;
			delivered_at?: string;
		}>;
		eval_results: Array<{
			eval_name: string;
			metric_key: string;
			metric_value?: number;
		}>;
	};
}

const algoStatusColor: Record<string, string> = {
	ok: "green",
	succeeded: "green",
	failed: "red",
	blocked: "orange",
	pending: "blue",
	running: "processing",
};

export interface LineageTabProps {
	assetId: string;
}

export default function LineageTab({ assetId }: LineageTabProps) {
	const [data, setData] = useState<LineageData | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		if (!assetId) return;
		setLoading(true);
		setError(null);
		assetsApi
			.getLineage(assetId)
			.then((d) => setData(d as unknown as LineageData))
			.catch((e) => setError(e instanceof Error ? e.message : "加载失败"))
			.finally(() => setLoading(false));
	}, [assetId]);

	if (loading)
		return <Spin style={{ display: "block", margin: "40px auto" }} />;
	if (error)
		return (
			<Alert
				type="warning"
				showIcon
				message="血缘加载失败"
				description={error}
			/>
		);
	if (!data) return <Alert type="info" showIcon message="暂无血缘数据" />;

	return (
		<div style={{ padding: "0 4px", maxWidth: 720 }}>
			{/* Upstream */}
			<Card
				size="small"
				title={
					<span>
						<FileOutlined /> 上游
					</span>
				}
				style={{ marginBottom: 16 }}
			>
				{data.upstream.mcap_file_id ? (
					<Descriptions column={1} size="small">
						<Descriptions.Item label="MCAP File ID">
							<Text code>{data.upstream.mcap_file_id}</Text>
						</Descriptions.Item>
						<Descriptions.Item label="存储 URI">
							<Text ellipsis style={{ fontFamily: "monospace", fontSize: 12 }}>
								{data.upstream.mcap_uri || "—"}
							</Text>
						</Descriptions.Item>
						<Descriptions.Item label="入库状态">
							<Tag>{data.upstream.ingest_state || "—"}</Tag>
						</Descriptions.Item>
					</Descriptions>
				) : (
					<Text type="secondary">无上游 MCAP 文件关联</Text>
				)}
			</Card>

			{/* Downstream */}
			<Card
				size="small"
				title={
					<span>
						<LinkOutlined /> 下游
					</span>
				}
			>
				{/* Algo results */}
				<div style={{ marginBottom: 16 }}>
					<Text strong style={{ fontSize: 13 }}>
						<RobotOutlined /> 算法处理
					</Text>
					{data.downstream.algo_results.length === 0 ? (
						<div style={{ color: "#999", marginTop: 4, fontSize: 12 }}>
							暂无算法结果
						</div>
					) : (
						<List
							size="small"
							dataSource={data.downstream.algo_results}
							renderItem={(a) => (
								<List.Item>
									<div>
										<Text code style={{ fontSize: 12 }}>
											{a.algo_name}
											{a.algo_version ? `@${a.algo_version}` : ""}
										</Text>
										<Tag
											color={algoStatusColor[a.status] || "default"}
											style={{ marginLeft: 8 }}
										>
											{a.status}
										</Tag>
										{a.run_id && (
											<Text
												type="secondary"
												style={{ fontSize: 11, marginLeft: 8 }}
											>
												run: {a.run_id.slice(0, 12)}…
											</Text>
										)}
									</div>
								</List.Item>
							)}
						/>
					)}
				</div>

				{/* Deliveries */}
				<div style={{ marginBottom: 16 }}>
					<Text strong style={{ fontSize: 13 }}>
						<SendOutlined /> 交付记录
					</Text>
					{data.downstream.deliveries.length === 0 ? (
						<div style={{ color: "#999", marginTop: 4, fontSize: 12 }}>
							暂无交付记录
						</div>
					) : (
						<List
							size="small"
							dataSource={data.downstream.deliveries}
							renderItem={(d) => (
								<List.Item>
									<Text code style={{ fontSize: 11 }} ellipsis>
										{d.delivery_id.slice(0, 8)}…
									</Text>
									<Text style={{ marginLeft: 8, fontSize: 12 }}>
										→ {d.customer_id || "—"}
									</Text>
									{d.delivered_at && (
										<Text
											type="secondary"
											style={{ marginLeft: 8, fontSize: 11 }}
										>
											{d.delivered_at.slice(0, 10)}
										</Text>
									)}
								</List.Item>
							)}
						/>
					)}
				</div>

				{/* Eval results */}
				<div>
					<Text strong style={{ fontSize: 13 }}>
						<FundOutlined /> 评测结果
					</Text>
					{data.downstream.eval_results.length === 0 ? (
						<div style={{ color: "#999", marginTop: 4, fontSize: 12 }}>
							暂无评测结果
						</div>
					) : (
						<List
							size="small"
							dataSource={data.downstream.eval_results}
							renderItem={(e) => (
								<List.Item>
									<Text code style={{ fontSize: 12 }}>
										{e.eval_name}
									</Text>
									<Text style={{ marginLeft: 8, fontSize: 12 }}>
										{e.metric_key}
									</Text>
									{e.metric_value !== undefined && (
										<Tag style={{ marginLeft: 8 }}>{e.metric_value}</Tag>
									)}
								</List.Item>
							)}
						/>
					)}
				</div>
			</Card>
		</div>
	);
}
