import { useEffect, useState } from "react";
import { Table, Card, Spin, Empty, Tag, message } from "antd";
import dayjs from "dayjs";
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
    setLoading(true);

    assetsApi
      .listDeliveries(assetId, 1, 100)
      .then(async (firstPage) => {
        let ids = firstPage.items ?? [];
        let page = firstPage.page ?? 1;
        const pageSize = firstPage.page_size ?? 100;
        let nextToken = firstPage.next_token ?? "";

        while (nextToken && !cancelled && ids.length < (firstPage.total ?? ids.length)) {
          page += 1;
          const nextPage = await assetsApi.listDeliveries(assetId, page, pageSize);
          ids = ids.concat(nextPage.items ?? []);
          nextToken = nextPage.next_token ?? "";
        }

        if (cancelled || ids.length === 0) {
          if (!cancelled) setDeliveries([]);
          return;
        }
        const results = await Promise.allSettled(
          ids.map((did) => deliveriesApi.get(did))
        );
        if (cancelled) return;
        const items = results
          .filter((r): r is PromiseFulfilledResult<Delivery> => r.status === "fulfilled")
          .map((r) => r.value);
        setDeliveries(items);
      })
      .catch(() => {
        if (!cancelled) {
          message.error("加载交付历史失败");
          setDeliveries([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => { cancelled = true; };
  }, [assetId]);

  if (loading) {
    return (
      <Card size="small">
        <div className="flex justify-center py-8"><Spin /></div>
      </Card>
    );
  }

  if (deliveries.length === 0) {
    return (
      <Card size="small">
        <Empty description="该资产尚未交付" image={Empty.PRESENTED_IMAGE_SIMPLE} />
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
