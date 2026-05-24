import { Card, Empty, message, Spin, Table, Tag } from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { assetsApi } from "../../api/assets";
import { deliveriesApi } from "../../api/deliveries";
import type { Delivery } from "../../api/types";

interface Props {
	assetId: string;
}

const statusColor: Record<string, string> = {
	delivered: "success",
	pending: "processing",
	draft: "default",
	accepted: "success",
	rejected: "error",
};

export default function DeliveryHistoryTab({ assetId }: Props) {
	const [loading, setLoading] = useState(true);
	const [deliveries, setDeliveries] = useState<Delivery[]>([]);

	useEffect(() => {
		let cancelled = false;

		const run = async () => {
			setLoading(true);
			try {
				const res = await assetsApi.listDeliveries(assetId, 1, 100);
				const ids = res.items ?? [];
				if (ids.length === 0) {
					if (!cancelled) setDeliveries([]);
					return;
				}
				const items = await Promise.all(ids.map((id) => deliveriesApi.get(id)));
				if (!cancelled) setDeliveries(items);
			} catch {
				if (!cancelled) {
					message.error("加载交付历史失败");
					setDeliveries([]);
				}
			} finally {
				if (!cancelled) setLoading(false);
			}
		};

		run();
		return () => {
			cancelled = true;
		};
	}, [assetId]);

	if (loading) {
		return (
			<Card size="small">
				<div className="flex justify-center py-8">
					<Spin />
				</div>
			</Card>
		);
	}

	if (deliveries.length === 0) {
		return (
			<Card size="small">
				<Empty
					description="该资产尚未交付"
					image={Empty.PRESENTED_IMAGE_SIMPLE}
				/>
			</Card>
		);
	}

	return (
		<Card size="small">
			<Table
				rowKey="delivery_id"
				dataSource={deliveries}
				size="small"
				pagination={false}
				columns={[
					{
						title: "交付 ID",
						dataIndex: "delivery_id",
						width: 220,
						render: (v: string) => <code className="text-xs">{v}</code>,
					},
					{
						title: "客户",
						dataIndex: "customer_id",
						width: 140,
					},
					{
						title: "状态",
						dataIndex: "status",
						width: 100,
						render: (s: string) => (
							<Tag color={statusColor[s] ?? "default"}>{s}</Tag>
						),
					},
					{
						title: "交付时间",
						dataIndex: "delivered_at",
						width: 160,
						render: (v: string) =>
							v ? dayjs(v).format("YYYY-MM-DD HH:mm") : "—",
					},
					{
						title: "资产数",
						dataIndex: "asset_count",
						width: 80,
					},
				]}
			/>
		</Card>
	);
}
