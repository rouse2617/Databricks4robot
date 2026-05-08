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
				const pageSize = 100;
				let ids: string[] = [];
				let page = 1;

				while (true) {
					const res = await assetsApi.listDeliveries(assetId, page, pageSize);
					const pageIds = res.items ?? [];
					ids = ids.concat(pageIds);

					// 后端这里是用 page/page_size 做切片，没有真正的 next_token 分页游标；
					// 用“当前页返回数量 < pageSize”判断是否到末页更可靠。
					if (pageIds.length < pageSize) break;
					page += 1;

					// 安全兜底，避免后端将来改为游标分页导致潜在死循环。
					if (page > 1000) break;
				}

				if (cancelled || ids.length === 0) {
					if (!cancelled) setDeliveries([]);
					return;
				}

				const results = await Promise.allSettled(
					ids.map((did) => deliveriesApi.get(did)),
				);
				if (cancelled) return;

				const items = results
					.filter(
						(r): r is PromiseFulfilledResult<Delivery> =>
							r.status === "fulfilled",
					)
					.map((r) => r.value);
				setDeliveries(items);
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
