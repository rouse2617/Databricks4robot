// ─── AssetQuickPreviewPane — Right-side quick preview panel ───
// Shows asset summary, algo status, files, and actions when a row is selected.
// Validates: Requirements R7

import {
  Card,
  Tag,
  Descriptions,
  Skeleton,
  Button,
  Tooltip,
  Typography,
  Badge,
  Divider,
} from "antd";
import {
  EyeOutlined,
  SearchOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  FileImageOutlined,
  InboxOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import type { Asset } from "../../api/types";
import type {
  FetchStatus,
  PreviewManifest,
  PreviewAvailability,
} from "../../lib/assets/assetsDiscoveryTypes";
import { parseAlgoEntries, countStatuses } from "./AlgoSummaryCell";
import { formatDurationSeconds, getAssetStateColor, getLifecycleState } from "../../lib/assetPresentation";

const { Text, Title } = Typography;

// ─── Props ───

export interface AssetQuickPreviewPaneProps {
  activeAssetId: string | null;
  fetchStatus: FetchStatus;
  asset: Asset | null;
  previewManifest: PreviewManifest | null;
  collapsed: boolean;
  onCollapse: () => void;
  onOpenDetail: (assetId: string) => void;
  onFindSimilar: (assetId: string) => void;
  onRetry?: () => void;
}

// ─── Status Color Map ───

const PRIORITY_TAG_COLOR: Record<string, string> = {
  critical: "red",
  high: "orange",
  medium: "blue",
  low: "default",
};

const QUALITY_TAG_COLOR: Record<string, string> = {
  excellent: "green",
  good: "cyan",
  acceptable: "blue",
  poor: "orange",
  unusable: "red",
};

const AVAILABILITY_BADGE: Record<PreviewAvailability, { status: "default" | "success" | "processing" | "error" | "warning"; text: string }> = {
  missing: { status: "default", text: "暂无预览" },
  processing: { status: "processing", text: "Processing" },
  ready: { status: "success", text: "Ready" },
  failed: { status: "error", text: "Failed" },
};


// ─── Algo Summary Helpers (reuses AlgoSummaryCell parsing) ───

const ALGO_STATUS_ORDER = ["ok", "failed", "running", "pending", "blocked"];

function AlgoSummaryBlock({ asset }: { asset: Asset }) {
  const entries = parseAlgoEntries(asset.algo_results);
  if (entries.length === 0) {
    return <Text type="secondary">无算法结果</Text>;
  }

  const counts = countStatuses(entries);

  const parts: { label: string; count: number; color?: string }[] = [];
  for (const status of ALGO_STATUS_ORDER) {
    if (counts[status]) {
      parts.push({
        label: status,
        count: counts[status],
        color: status === "failed" ? "#ff4d4f" : undefined,
      });
    }
  }
  for (const [status, count] of Object.entries(counts)) {
    if (!ALGO_STATUS_ORDER.includes(status)) {
      parts.push({ label: status, count });
    }
  }

  // Find most recent failure reason
  const failedEntry = entries.find((e) => e.status === "failed");
  const failureReasonKey = failedEntry
    ? Object.keys(asset.algo_results).find(
        (k) => k.startsWith(failedEntry.key) && k.endsWith(":reason"),
      )
    : null;
  const failureReason = failureReasonKey
    ? asset.algo_results[failureReasonKey]
    : null;

  return (
    <div>
      <div style={{ fontSize: 13 }}>
        {parts.map((p, i) => (
          <span key={p.label}>
            {i > 0 && " / "}
            <span style={p.color ? { color: p.color } : undefined}>
              {p.count} {p.label}
            </span>
          </span>
        ))}
      </div>
      {failureReason && (
        <Text type="secondary" style={{ fontSize: 12, display: "block", marginTop: 4 }}>
          失败原因: {failureReason}
        </Text>
      )}
    </div>
  );
}

// ─── Files Block ───

function FilesBlock({ files }: { files: Record<string, string> }) {
  const keys = Object.keys(files);
  if (keys.length === 0) {
    return <Text type="secondary">无文件</Text>;
  }
  return (
    <div>
      {keys.map((key) => (
        <div
          key={key}
          style={{
            display: "flex",
            justifyContent: "space-between",
            padding: "2px 0",
            fontSize: 12,
          }}
        >
          <Text style={{ fontSize: 12 }}>{key}</Text>
          {files[key] ? (
            <CheckCircleOutlined style={{ color: "#52c41a" }} />
          ) : (
            <CloseCircleOutlined style={{ color: "#ff4d4f" }} />
          )}
        </div>
      ))}
    </div>
  );
}

// ─── Empty State ───

function EmptyState() {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        height: "100%",
        minHeight: 300,
        color: "#bfbfbf",
      }}
    >
      <InboxOutlined style={{ fontSize: 48, marginBottom: 12 }} />
      <Text type="secondary">点击行查看预览</Text>
    </div>
  );
}

// ─── Loading State ───

function LoadingState() {
  return (
    <div style={{ padding: 16 }}>
      <Skeleton active paragraph={{ rows: 8 }} />
    </div>
  );
}

// ─── Error State ───

function ErrorState({ onRetry }: { onRetry?: () => void }) {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        height: "100%",
        minHeight: 300,
        color: "#ff4d4f",
      }}
    >
      <CloseCircleOutlined style={{ fontSize: 48, marginBottom: 12 }} />
      <Text type="danger" style={{ marginBottom: 12 }}>加载预览失败</Text>
      {onRetry && (
        <Button size="small" onClick={onRetry}>
          重试
        </Button>
      )}
    </div>
  );
}


// ─── Preview Media Placeholder ───

function PreviewMediaBlock({ manifest }: { manifest: PreviewManifest | null }) {
  const availability = manifest?.availability ?? "missing";
  const badge = AVAILABILITY_BADGE[availability];
  return (
    <div
      style={{
        background: "#fafafa",
        border: "1px dashed #d9d9d9",
        borderRadius: 4,
        padding: 24,
        textAlign: "center",
        marginBottom: 12,
      }}
    >
      <FileImageOutlined style={{ fontSize: 36, color: "#bfbfbf", marginBottom: 8 }} />
      <div>
        <Text type="secondary" style={{ fontSize: 13 }}>
          暂无预览
        </Text>
      </div>
      <Badge status={badge.status} text={<Text type="secondary" style={{ fontSize: 12 }}>{badge.text}</Text>} />
    </div>
  );
}

// ─── Loaded Content ───

function LoadedContent({
  asset,
  previewManifest,
  onOpenDetail,
  onFindSimilar,
}: {
  asset: Asset;
  previewManifest: PreviewManifest | null;
  onOpenDetail: (assetId: string) => void;
  onFindSimilar: (assetId: string) => void;
}) {
  const priority = asset.tags?.priority;
  const quality = asset.tags?.quality;

  return (
    <div style={{ padding: "0 4px" }}>
      {/* Block 1: Header */}
      <div style={{ marginBottom: 12 }}>
        <Title level={5} style={{ margin: 0, fontFamily: "monospace", fontSize: 14 }}>
          {asset.asset_id}
        </Title>
        <div style={{ marginTop: 6, display: "flex", gap: 4, flexWrap: "wrap" }}>
          <Tag color={getAssetStateColor(asset)}>{getLifecycleState(asset) || "—"}</Tag>
          {priority && (
            <Tag color={PRIORITY_TAG_COLOR[priority] ?? "default"}>
              {priority}
            </Tag>
          )}
          {quality && (
            <Tag color={QUALITY_TAG_COLOR[quality] ?? "default"}>
              {quality}
            </Tag>
          )}
        </div>
      </div>

      <Divider style={{ margin: "8px 0" }} />

      {/* Block 2: Preview Media */}
      <PreviewMediaBlock manifest={previewManifest} />

      <Divider style={{ margin: "8px 0" }} />

      {/* Block 3: Summary */}
      <Descriptions
        column={1}
        size="small"
        styles={{ label: { fontSize: 12, color: "#8c8c8c" }, content: { fontSize: 12 } }}
      >
        <Descriptions.Item label="MCAP">{asset.mcap_file_id}</Descriptions.Item>
        <Descriptions.Item label="Env">{asset.env ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="Scene">{asset.tags?.scene ?? "—"}</Descriptions.Item>
        <Descriptions.Item label="Duration">{formatDurationSeconds(asset)}</Descriptions.Item>
        <Descriptions.Item label="Owner">{asset.owner}</Descriptions.Item>
        <Descriptions.Item label="Reviewer">{asset.reviewer}</Descriptions.Item>
      </Descriptions>

      <Divider style={{ margin: "8px 0" }} />

      {/* Block 4: Algo Summary */}
      <div style={{ marginBottom: 8 }}>
        <Text strong style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
          算法摘要
        </Text>
        <AlgoSummaryBlock asset={asset} />
      </div>

      <Divider style={{ margin: "8px 0" }} />

      {/* Block 5: Files */}
      <div style={{ marginBottom: 12 }}>
        <Text strong style={{ fontSize: 12, display: "block", marginBottom: 4 }}>
          文件
        </Text>
        <FilesBlock files={asset.files ?? {}} />
      </div>

      <Divider style={{ margin: "8px 0" }} />

      {/* Actions */}
      <div style={{ display: "flex", gap: 8 }}>
        <Button
          type="primary"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => onOpenDetail(asset.asset_id)}
        >
          查看详情
        </Button>
        <Tooltip title="Coming in Phase 3">
          <Button
            size="small"
            icon={<SearchOutlined />}
            disabled
            onClick={() => onFindSimilar(asset.asset_id)}
          >
            查找相似
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}

// ─── Main Component ───

export default function AssetQuickPreviewPane({
  activeAssetId,
  fetchStatus,
  asset,
  previewManifest,
  collapsed,
  onCollapse,
  onOpenDetail,
  onFindSimilar,
  onRetry,
}: AssetQuickPreviewPaneProps) {
  // Collapsed state: thin vertical bar with expand button
  if (collapsed) {
    return (
      <div
        style={{
          width: 36,
          minHeight: 400,
          background: "#fafafa",
          borderLeft: "1px solid #f0f0f0",
          display: "flex",
          alignItems: "flex-start",
          justifyContent: "center",
          paddingTop: 12,
        }}
      >
        <Button
          type="text"
          size="small"
          icon={<MenuUnfoldOutlined />}
          onClick={onCollapse}
          title="展开预览"
        />
      </div>
    );
  }

  // Determine content
  let content: React.ReactNode;
  if (!activeAssetId) {
    content = <EmptyState />;
  } else if (fetchStatus === "loading") {
    content = <LoadingState />;
  } else if (fetchStatus === "error") {
    content = <ErrorState onRetry={onRetry} />;
  } else if (asset) {
    content = (
      <LoadedContent
        asset={asset}
        previewManifest={previewManifest}
        onOpenDetail={onOpenDetail}
        onFindSimilar={onFindSimilar}
      />
    );
  } else {
    content = <EmptyState />;
  }

  return (
    <Card
      size="small"
      style={{ width: 320, minHeight: 400, overflow: "auto" }}
      title={
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <Text strong style={{ fontSize: 13 }}>预览</Text>
          <Button
            type="text"
            size="small"
            icon={<MenuFoldOutlined />}
            onClick={onCollapse}
            title="收起预览"
          />
        </div>
      }
      styles={{ body: { padding: 12 } }}
    >
      {content}
    </Card>
  );
}
