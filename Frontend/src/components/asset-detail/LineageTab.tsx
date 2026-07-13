// ─── LineageTab — Storage source + child assets ───
// CYB-3388: scoped down to just the two things the blood-line view actually
// needs to tell the user — where the raw bytes live and what child assets
// were carved out of this asset. Algo / delivery / eval columns used to also
// live here but they are already primary tabs of the detail page; keeping
// them here duplicated the info and forced users to guess which copy was
// authoritative. For raw_mcap assets (asset_id === mcap_file_id) the
// "upstream" card is retitled to "存储" and hides the mcap_file_id /
// ingest_state fields, since the asset itself is the source — showing
// "上游 = 自己" was the top confusion point in user UX walkthroughs.

import { FileOutlined, LinkOutlined } from "@ant-design/icons";
import {
	Alert,
	Card,
	Col,
	Descriptions,
	List,
	Row,
	Spin,
	Tag,
	Typography,
} from "antd";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { assetsApi } from "../../api/assets";
import type { PipelineLineage } from "../../api/types";

const { Text, Link } = Typography;

interface LineageData {
	asset_id: string;
	upstream: {
		mcap_file_id?: string;
		mcap_uri?: string;
		ingest_state?: string;
	};
	downstream: {
		// CYB-3388: algo_results / deliveries / eval_results still ship from the
		// backend for callers that rely on the full lineage graph, but the
		// LineageTab UI no longer renders them — the primary tabs at the top of
		// AssetDetailPage own those views. Kept in the type for wire-shape
		// stability; intentionally unused here.
		algo_results?: unknown[];
		deliveries?: unknown[];
		eval_results?: unknown[];
		// CYB-3293: child assets (clip/action/frame/task) under this asset.
		children?: Array<{
			asset_id: string;
			asset_type?: string;
			parent_asset_id?: string;
		}>;
	};
}

export interface LineageTabProps {
	assetId: string;
	/** CYB-3388: current asset's type — when 'raw_mcap' the upstream card is
	 * retitled and mcap_file_id / ingest_state rows are hidden because the
	 * asset itself IS the source (asset_id === mcap_file_id by design). */
	assetType?: string;
	/** CYB-3279: immediate parent asset (e.g. segment) for child assets like action. */
	parentAssetId?: string;
}

export default function LineageTab({
	assetId,
	assetType,
	parentAssetId,
}: LineageTabProps) {
	const navigate = useNavigate();
	const [data, setData] = useState<LineageData | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [pipelineLineage, setPipelineLineage] =
		useState<PipelineLineage | null>(null);
	const [pipelineLoading, setPipelineLoading] = useState(true);

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

	useEffect(() => {
		if (!assetId) return;
		setPipelineLoading(true);
		assetsApi
			.getPipelineLineage(assetId)
			.then((pl) => setPipelineLineage(pl))
			.catch(() => {
				// pipeline-lineage endpoint may be unavailable — silently ignore
			})
			.finally(() => setPipelineLoading(false));
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

	const hasPipelineLineage =
		pipelineLineage &&
		(pipelineLineage.workflow_name || pipelineLineage.deployment_id);
	// CYB-3388: for raw_mcap the asset itself is the storage source — mcap_file_id
	// equals asset_id by design (see CYB-3376). Hide the self-reference rows to
	// avoid the "upstream = me" confusion; keep the GCS URI which is the one
	// signal users actually want here.
	const isSelfSourced =
		assetType === "raw_mcap" || data.upstream.mcap_file_id === assetId;

	return (
		<div style={{ padding: "0 4px" }}>
			<Row gutter={16}>
				<Col xs={24} md={10}>
					{/* Pipeline provenance */}
					{!pipelineLoading && hasPipelineLineage && (
						<Card
							size="small"
							title={
								<span>
									<LinkOutlined /> Pipeline 血缘
								</span>
							}
							style={{ marginBottom: 16 }}
						>
							<Descriptions column={1} size="small">
								{(pipelineLineage?.deployment_id ||
									pipelineLineage?.workflow_name) && (
									<Descriptions.Item label="运行">
										<Text
											code
											style={{ cursor: "pointer", color: "#1677ff" }}
											onClick={() => {
												const runId = pipelineLineage?.deployment_id;
												const wf = pipelineLineage?.workflow_name;
												if (runId) {
													navigate(`/runs/${encodeURIComponent(runId)}`);
												} else if (wf) {
													navigate(
														`/pipeline/executions/${encodeURIComponent(wf)}`,
													);
												}
											}}
										>
											{pipelineLineage?.deployment_id ??
												pipelineLineage?.workflow_name}
										</Text>
									</Descriptions.Item>
								)}
								{pipelineLineage?.pipeline_name && (
									<Descriptions.Item label="Pipeline">
										<Text code>{pipelineLineage?.pipeline_name}</Text>
									</Descriptions.Item>
								)}
								{pipelineLineage?.node_id && (
									<Descriptions.Item label="节点">
										<Text code>{pipelineLineage?.node_id}</Text>
									</Descriptions.Item>
								)}
								{pipelineLineage?.produced_at && (
									<Descriptions.Item label="产生时间">
										<Text type="secondary">
											{new Date(pipelineLineage?.produced_at).toLocaleString()}
										</Text>
									</Descriptions.Item>
								)}
							</Descriptions>
							{pipelineLineage?.input_assets &&
								pipelineLineage?.input_assets.length > 0 && (
									<div style={{ marginTop: 8 }}>
										<Text strong style={{ fontSize: 12 }}>
											输入资产:
										</Text>
										<div
											style={{
												marginTop: 4,
												display: "flex",
												gap: 4,
												flexWrap: "wrap",
											}}
										>
											{pipelineLineage?.input_assets.map((aid) => (
												<Tag
													key={aid}
													style={{
														cursor: "pointer",
														fontFamily: "monospace",
														fontSize: 11,
													}}
													onClick={() =>
														navigate(`/assets/${encodeURIComponent(aid)}`)
													}
												>
													{aid.slice(0, 16)}...
												</Tag>
											))}
										</div>
									</div>
								)}
						</Card>
					)}

					{/* Storage / upstream */}
					<Card
						size="small"
						title={
							<span>
								<FileOutlined /> {isSelfSourced ? "存储" : "上游"}
							</span>
						}
					>
						{/* CYB-3279: child assets (action/clip/frame/task) show the immediate parent asset above the raw MCAP; hidden for segments whose parent_asset_id equals the mcap shown below. */}
						{parentAssetId && parentAssetId !== data.upstream.mcap_file_id && (
							<Descriptions column={1} size="small" style={{ marginBottom: 8 }}>
								<Descriptions.Item label="父资产">
									<Link onClick={() => navigate(`/assets/${parentAssetId}`)}>
										<Text code>{parentAssetId}</Text>
									</Link>
								</Descriptions.Item>
							</Descriptions>
						)}
						{data.upstream.mcap_file_id ? (
							<Descriptions column={1} size="small">
								{!isSelfSourced && (
									<Descriptions.Item label="MCAP File ID">
										<Text code>{data.upstream.mcap_file_id}</Text>
									</Descriptions.Item>
								)}
								<Descriptions.Item label="存储 URI">
									<Text
										ellipsis
										style={{ fontFamily: "monospace", fontSize: 12 }}
									>
										{data.upstream.mcap_uri || "—"}
									</Text>
								</Descriptions.Item>
								{!isSelfSourced && (
									<Descriptions.Item label="入库状态">
										<Tag>{data.upstream.ingest_state || "—"}</Tag>
									</Descriptions.Item>
								)}
							</Descriptions>
						) : (
							<Text type="secondary">无上游 MCAP 文件关联</Text>
						)}
					</Card>
				</Col>

				<Col xs={24} md={14}>
					{/* Children (downstream entity assets only) */}
					<Card
						size="small"
						title={
							<span>
								<LinkOutlined /> 子资产
							</span>
						}
					>
						{!data.downstream.children ||
						data.downstream.children.length === 0 ? (
							<Text type="secondary">暂无子资产</Text>
						) : (
							<List
								size="small"
								dataSource={data.downstream.children}
								renderItem={(c) => (
									<List.Item>
										<Link onClick={() => navigate(`/assets/${c.asset_id}`)}>
											<Text code style={{ fontSize: 12 }}>
												{c.asset_id}
											</Text>
										</Link>
										{c.asset_type && (
											<Tag style={{ marginLeft: 8 }}>{c.asset_type}</Tag>
										)}
									</List.Item>
								)}
							/>
						)}
					</Card>
				</Col>
			</Row>
		</div>
	);
}
