import { Card, Collapse, Descriptions, Spin, Tag, Typography } from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import ReactJson from "react-json-view";
import { type AssetMetadataResponse, assetsApi } from "../../api/assets";
import type { Asset } from "../../api/types";
import {
	formatDurationSeconds,
	getAssetType,
	getLifecycleState,
} from "../../lib/assetPresentation";
import { formatDateTime } from "../../lib/dateTime";

const { Text } = Typography;

interface Props {
	asset: Asset;
}

export default function OverviewTab({ asset }: Props) {
	const [metadata, setMetadata] = useState<AssetMetadataResponse | null>(null);
	const [metadataLoading, setMetadataLoading] = useState(false);
	const [metadataError, setMetadataError] = useState<string | null>(null);

	useEffect(() => {
		setMetadataLoading(true);
		setMetadataError(null);
		assetsApi
			.getMetadata(asset.asset_id)
			.then((data) => {
				setMetadata(data);
				setMetadataLoading(false);
			})
			.catch((err) => {
				setMetadataError(err?.message || "加载元数据失败");
				setMetadataLoading(false);
			});
	}, [asset.asset_id]);

	return (
		<Card size="small">
			<Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
				<Descriptions.Item label="Asset ID">
					<Text code copyable className="text-xs">
						{asset.asset_id}
					</Text>
				</Descriptions.Item>
				<Descriptions.Item label="MCAP File">
					{asset.mcap_file_id ? (
						<Text code copyable className="text-xs">
							{asset.mcap_file_id}
						</Text>
					) : (
						<code className="text-xs">—</code>
					)}
				</Descriptions.Item>
				<Descriptions.Item label="Segment Locator">
					{asset.segment_locator ? (
						<Text
							code
							copyable
							ellipsis={{ tooltip: asset.segment_locator }}
							className="text-xs"
							style={{ maxWidth: "100%" }}
						>
							{asset.segment_locator}
						</Text>
					) : (
						<code className="text-xs">—</code>
					)}
				</Descriptions.Item>
				<Descriptions.Item label="起始时间 (ns)">
					{asset.start_timestamp_ns}
				</Descriptions.Item>
				<Descriptions.Item label="结束时间 (ns)">
					{asset.end_timestamp_ns ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="时长">
					{formatDurationSeconds(asset, 3)}
				</Descriptions.Item>
				<Descriptions.Item label="资产类型">
					{getAssetType(asset) || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="保留层级">
					{asset.retention_tier ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="生命周期">
					{getLifecycleState(asset) || "—"}
				</Descriptions.Item>
				<Descriptions.Item label="状态 (legacy)">
					<span style={{ color: "#8c8c8c" }}>{asset.status ?? "—"}</span>
				</Descriptions.Item>
				<Descriptions.Item label="环境">{asset.env ?? "—"}</Descriptions.Item>
				<Descriptions.Item label="任务">{asset.task ?? "—"}</Descriptions.Item>
				<Descriptions.Item label="审核人">
					{asset.reviewer ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Owner">
					{asset.owner ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="过期时间">
					{asset.expire_at ? dayjs(asset.expire_at).fromNow() : "—"}
				</Descriptions.Item>
				<Descriptions.Item label="逻辑资产 ID">
					<code className="text-xs">
						{asset.logical_asset_id ?? asset.asset_id}
					</code>
				</Descriptions.Item>
				<Descriptions.Item label="资产版本 (revision)">
					{asset.revision ?? "—"}
					{asset.is_current === false ? (
						<Tag color="default" className="ml-2">
							历史
						</Tag>
					) : asset.is_current === true ? (
						<Tag color="blue" className="ml-2">
							当前
						</Tag>
					) : null}
				</Descriptions.Item>
				<Descriptions.Item label="创建时间">
					{formatDateTime(asset.created_at)}
				</Descriptions.Item>
				<Descriptions.Item label="更新时间">
					{formatDateTime(asset.updated_at)}
				</Descriptions.Item>
			</Descriptions>
			<Collapse
				style={{ marginTop: 16 }}
				items={[
					{
						key: "metadata",
						label: "高级 / 元数据",
						children: (
							<div>
								{metadataLoading && <Spin />}
								{metadataError && (
									<div style={{ color: "#ff4d4f" }}>{metadataError}</div>
								)}
								{metadata && (
									<ReactJson
										src={metadata}
										collapsed={1}
										name={false}
										enableClipboard={true}
										displayDataTypes={false}
										quotesOnKeys={false}
										theme="rjv-default"
									/>
								)}
							</div>
						),
					},
				]}
			/>
		</Card>
	);
}
