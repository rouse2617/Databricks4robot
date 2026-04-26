// ─── ResultsEmptyState — Presentational Component ───
// Three variants: initial empty, no results (filters too narrow), error state.
// Validates: Requirements R1

import { Button, Result, Space, Typography } from "antd";
import {
  InboxOutlined,
  SearchOutlined,
  UploadOutlined,
  ReloadOutlined,
} from "@ant-design/icons";

const { Text } = Typography;

// ─── Props ───

export interface ResultsEmptyStateProps {
  variant: "initial" | "no_results" | "error";
  error?: string | null;
  activeFilterCount?: number;
  onClearFilters?: () => void;
  onRetry?: () => void;
}

// ─── Component ───

export default function ResultsEmptyState({
  variant,
  error,
  activeFilterCount = 0,
  onClearFilters,
  onRetry,
}: ResultsEmptyStateProps) {
  if (variant === "error") {
    return (
      <Result
        status="error"
        title="查询失败"
        subTitle={error ?? "未知错误"}
        extra={
          onRetry && (
            <Button type="primary" icon={<ReloadOutlined />} onClick={onRetry}>
              重试
            </Button>
          )
        }
      />
    );
  }

  if (variant === "no_results") {
    return (
      <Result
        icon={<SearchOutlined style={{ color: "#bfbfbf" }} />}
        title="没有匹配的资产"
        subTitle={
          activeFilterCount > 0 ? (
            <Text type="secondary">
              当前有 {activeFilterCount} 个筛选条件，尝试减少筛选范围
            </Text>
          ) : undefined
        }
        extra={
          <Space>
            {onClearFilters && (
              <Button onClick={onClearFilters}>清空筛选</Button>
            )}
            {onClearFilters && (
              <Button type="primary" onClick={onClearFilters}>
                回到全部资产
              </Button>
            )}
          </Space>
        }
      />
    );
  }

  // variant === "initial"
  return (
    <Result
      icon={<InboxOutlined style={{ fontSize: 48, color: "#bfbfbf" }} />}
      title="平台还没有数据"
      extra={
        <Button type="primary" icon={<UploadOutlined />}>
          上传 MCAP
        </Button>
      }
    />
  );
}
