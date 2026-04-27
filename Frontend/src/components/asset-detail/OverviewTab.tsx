import { Card, Descriptions } from "antd";
import dayjs from "dayjs";
import type { Asset } from "../../api/types";

interface Props {
  asset: Asset;
}

export default function OverviewTab({ asset }: Props) {
  return (
    <Card size="small">
      <Descriptions column={{ xs: 1, sm: 2 }} bordered size="small">
        <Descriptions.Item label="Asset ID">
          <code className="text-xs">{asset.asset_id}</code>
        </Descriptions.Item>
        <Descriptions.Item label="MCAP File">
          <code className="text-xs">{asset.mcap_file_id}</code>
        </Descriptions.Item>
        <Descriptions.Item label="Segment Locator">
          <code className="text-xs">{asset.segment_locator ?? "—"}</code>
        </Descriptions.Item>
        <Descriptions.Item label="起始时间 (ns)">
          {asset.start_timestamp_ns}
        </Descriptions.Item>
        <Descriptions.Item label="结束时间 (ns)">
          {asset.end_timestamp_ns ?? "—"}
        </Descriptions.Item>
        <Descriptions.Item label="时长">
          {asset.duration_sec ? `${asset.duration_sec.toFixed(3)} s` : "—"}
        </Descriptions.Item>
        <Descriptions.Item label="类型">{asset.type ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="环境">{asset.env ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="任务">{asset.task ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="审核人">{asset.reviewer ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="Owner">{asset.owner ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="版本">{asset.version}</Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {dayjs(asset.created_at).format("YYYY-MM-DD HH:mm:ss")}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {dayjs(asset.updated_at).format("YYYY-MM-DD HH:mm:ss")}
        </Descriptions.Item>
      </Descriptions>
    </Card>
  );
}
