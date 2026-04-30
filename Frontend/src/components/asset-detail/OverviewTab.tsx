import { Card, Descriptions } from "antd";
import dayjs from "dayjs";
import type { Asset } from "../../api/types";
import { formatDurationSeconds, getAssetType, getLifecycleState } from "../../lib/assetPresentation";

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
          {formatDurationSeconds(asset, 3)}
        </Descriptions.Item>
        <Descriptions.Item label="资产类型">{getAssetType(asset) || "—"}</Descriptions.Item>
        <Descriptions.Item label="保留层级">{asset.retention_tier ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="生命周期">{getLifecycleState(asset) || "—"}</Descriptions.Item>
        <Descriptions.Item label="状态 (legacy)">
          <span style={{ color: "#8c8c8c" }}>{asset.status ?? "—"}</span>
        </Descriptions.Item>
        <Descriptions.Item label="环境">{asset.env ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="任务">{asset.task ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="审核人">{asset.reviewer ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="Owner">{asset.owner ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="过期时间">
          {asset.expire_at ? dayjs(asset.expire_at).fromNow() : "—"}
        </Descriptions.Item>
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
