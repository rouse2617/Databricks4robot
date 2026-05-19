import { ArrowLeftOutlined, ReloadOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Descriptions,
	message,
	Spin,
	Table,
	Tag,
	Typography,
} from "antd";
import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { assetsApi } from "../api/assets";
import { deliveriesApi } from "../api/deliveries";
import type { Asset, Delivery, DeliveryItem } from "../api/types";
import {
	formatDurationSeconds,
	getAssetStateColor,
	getLifecycleState,
} from "../lib/assetPresentation";
import { navigateToAssetDetail } from "../lib/assets/assetWorkbenchNavigation";
import { formatDateTime } from "../lib/dateTime";

const { Title } = Typography;

const STATUS_COLOR: Record<string, string> = {
	draft: "default",
	pending: "processing",
	delivered: "success",
	accepted: "green",
	rejected: "error",
};

export default function DeliveryDetailPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const location = useLocation();
	const createdDeliveryId =
		(location.state as { createdDeliveryId?: string } | null)
			?.createdDeliveryId ?? null;
	const [msg, msgCtx] = message.useMessage();

	const [delivery, setDelivery] = useState<Delivery | null>(null);
	const [loading, setLoading] = useState(true);
	const [assets, setAssets] = useState<Asset[]>([]);
	const [deliveryItems, setDeliveryItems] = useState<DeliveryItem[]>([]);
	const [assetsLoading, setAssetsLoading] = useState(false);
	/** Rows from GET …/items; resolved rows after assetsApi.get per item */
	const [relatedCounts, setRelatedCounts] = useState({
		itemRows: 0,
		resolvedAssets: 0,
		rejectedFetches: 0,
	});

	// antd useMessage may return a fresh reference per render; capture in ref so
	// fetchDelivery's identity stays stable (otherwise useEffect re-fires every
	// render → infinite refetch).
	const msgRef = useRef(msg);
	msgRef.current = msg;

	const fetchDelivery = useCallback(async () => {
		if (!id) return;
		setLoading(true);
		try {
			const d = await deliveriesApi.get(id);
			setDelivery(d);
		} catch {
			msgRef.current.error("加载交付详情失败");
		} finally {
			setLoading(false);
		}
	}, [id]);

	const fetchAssets = useCallback(async () => {
		if (!id) return;
		setAssetsLoading(true);
		try {
			const items: DeliveryItem[] = await deliveriesApi.listItems(id);
			setDeliveryItems(items);
			const assetResults = await Promise.allSettled(
				items.map((item) => assetsApi.get(item.asset_id)),
			);
			const resolved = assetResults
				.filter(
					(r): r is PromiseFulfilledResult<Asset> => r.status === "fulfilled",
				)
				.map((r) => r.value);
			setRelatedCounts({
				itemRows: items.length,
				resolvedAssets: resolved.length,
				rejectedFetches: assetResults.filter((r) => r.status === "rejected")
					.length,
			});
			setAssets(resolved);
		} catch {
			setRelatedCounts({ itemRows: 0, resolvedAssets: 0, rejectedFetches: 0 });
			setDeliveryItems([]);
			setAssets([]);
		} finally {
			setAssetsLoading(false);
		}
	}, [id]);

	useEffect(() => {
		fetchDelivery();
		fetchAssets();
	}, [fetchDelivery, fetchAssets]);

	useEffect(() => {
		if (!createdDeliveryId || createdDeliveryId !== id) return;
		msg.success("交付创建成功");
		navigate(location.pathname, { replace: true, state: null });
	}, [createdDeliveryId, id, location.pathname, msg, navigate]);

	if (loading) {
		return (
			<div style={{ textAlign: "center", padding: 80 }}>
				<Spin size="large" />
			</div>
		);
	}

	if (!delivery) {
		return (
			<div style={{ textAlign: "center", padding: 80 }}>
				<Typography.Text type="secondary">交付记录未找到</Typography.Text>
			</div>
		);
	}

	// Keep rows even when asset details cannot be fetched (e.g. soft-deleted assets).
	const assetById = new Map(assets.map((a) => [a.asset_id, a]));
	const assetRows = deliveryItems.map((it) => ({
		asset_id: it.asset_id,
		asset: assetById.get(it.asset_id) ?? null,
	}));

	const assetColumns = [
		{
			title: "资产 ID",
			dataIndex: "asset_id",
			key: "asset_id",
			width: 240,
			ellipsis: true,
			render: (val: string) => (
				<button
					type="button"
					className="link-like-button"
					onClick={() => navigateToAssetDetail(navigate, val)}
				>
					{val}
				</button>
			),
		},
		{
			title: "生命周期",
			key: "status",
			width: 100,
			render: (_: unknown, row: { asset: Asset | null }) =>
				row.asset ? (
					<Tag color={getAssetStateColor(row.asset)}>
						{getLifecycleState(row.asset) || "—"}
					</Tag>
				) : (
					<Tag color="default">未知</Tag>
				),
		},
		{
			title: "时长 (s)",
			key: "duration",
			width: 100,
			render: (_: unknown, row: { asset: Asset | null }) =>
				row.asset ? formatDurationSeconds(row.asset) : "—",
		},
		{
			title: "Owner",
			key: "owner",
			width: 120,
			render: (_: unknown, row: { asset: Asset | null }) =>
				row.asset?.owner ?? "—",
		},
		{
			title: "更新时间",
			key: "updated_at",
			width: 180,
			render: (_: unknown, row: { asset: Asset | null }) =>
				formatDateTime(row.asset?.updated_at),
		},
		{
			title: "备注",
			key: "note",
			width: 200,
			render: (_: unknown, row: { asset: Asset | null }) =>
				row.asset ? null : (
					<Typography.Text type="secondary">资产不存在或不可见</Typography.Text>
				),
		},
	];

	return (
		<div>
			{msgCtx}

			{/* Header */}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 12,
					marginBottom: 16,
				}}
			>
				<Button
					icon={<ArrowLeftOutlined />}
					type="text"
					onClick={() => navigate("/deliveries")}
				/>
				<Title level={4} style={{ margin: 0 }}>
					交付详情
				</Title>
				<Tag color={STATUS_COLOR[delivery.status] ?? "default"}>
					{delivery.status}
				</Tag>
				<Button
					icon={<ReloadOutlined />}
					size="small"
					onClick={() => {
						fetchDelivery();
						fetchAssets();
					}}
				>
					刷新
				</Button>
			</div>

			{/* Basic info */}
			<Descriptions
				bordered
				column={2}
				size="small"
				style={{ marginBottom: 24 }}
			>
				<Descriptions.Item label="交付 ID">
					{delivery.delivery_id}
				</Descriptions.Item>
				<Descriptions.Item label="客户 ID">
					{delivery.customer_id}
				</Descriptions.Item>
				<Descriptions.Item label="合同号">
					{delivery.contract_id ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="Owner">{delivery.owner}</Descriptions.Item>
				<Descriptions.Item label="资产数">
					{delivery.asset_count}
				</Descriptions.Item>
				<Descriptions.Item label="交付时间">
					{formatDateTime(delivery.delivered_at)}
				</Descriptions.Item>
				<Descriptions.Item label="Manifest URI" span={2}>
					{delivery.manifest_uri ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="备注" span={2}>
					{delivery.note ?? "—"}
				</Descriptions.Item>
				<Descriptions.Item label="创建时间">
					{formatDateTime(delivery.created_at)}
				</Descriptions.Item>
				<Descriptions.Item label="更新时间">
					{formatDateTime(delivery.updated_at)}
				</Descriptions.Item>
			</Descriptions>

			{/* Related assets */}
			<Title level={5} style={{ marginBottom: 12 }}>
				关联资产
			</Title>
			{delivery.asset_count > 0 &&
				relatedCounts.itemRows === 0 &&
				!assetsLoading && (
					<Alert
						type="warning"
						showIcon
						style={{ marginBottom: 12 }}
						message="资产数与关联列表不一致"
						description={
							"详情里记录的资产数大于 0，但交付明细项列表为空。通常是 mock/种子数据未写入 delivery_items，或后端列表接口未返回行。可刷新重试或核对交付数据。"
						}
					/>
				)}
			{relatedCounts.itemRows > 0 && relatedCounts.rejectedFetches > 0 && (
				<Alert
					type="info"
					showIcon
					style={{ marginBottom: 12 }}
					message="部分资产详情未加载"
					description={`${relatedCounts.resolvedAssets} / ${relatedCounts.itemRows} 条资产拉取成功，其余请求失败（可检查权限或资产是否存在）。`}
				/>
			)}
			{delivery.asset_count > 0 &&
				relatedCounts.itemRows > 0 &&
				delivery.asset_count !== relatedCounts.itemRows && (
					<Alert
						type="info"
						showIcon
						style={{ marginBottom: 12 }}
						message="资产数与明细行数不一致"
						description={`交付摘要 asset_count=${delivery.asset_count}，明细项 ${relatedCounts.itemRows} 条。若以明细为准，可忽略摘要差异或联系后端对齐字段。`}
					/>
				)}
			<Table
				rowKey="asset_id"
				columns={assetColumns}
				dataSource={assetRows}
				loading={assetsLoading}
				pagination={false}
				size="small"
				scroll={{ x: 800 }}
				locale={{ emptyText: "暂无关联资产" }}
			/>
		</div>
	);
}
