// ─── ResultsEmptyState — Presentational Component ───
// Three variants: initial empty, no results (filters too narrow), error state.
// Validates: Requirements R1

import {
	InboxOutlined,
	ReloadOutlined,
	SearchOutlined,
	UploadOutlined,
} from "@ant-design/icons";
import { Button, Result, Space, Typography } from "antd";

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
					<Space direction="vertical" size={2}>
						<Text type="secondary">
							{activeFilterCount > 0
								? `当前有 ${activeFilterCount} 个筛选条件，先放宽筛选范围再试一次`
								: "先放宽条件，再回到默认视图"}
						</Text>
						<Text type="secondary" style={{ fontSize: 12 }}>
							也可以从上方“已应用筛选”里按字段清空，例如 owner、生命周期或 Tag
							条件。
						</Text>
					</Space>
				}
				extra={
					<Space>
						{onClearFilters && (
							<Button onClick={onClearFilters}>放宽筛选</Button>
						)}
						{onClearFilters && (
							<Button type="primary" onClick={onClearFilters}>
								回到默认视图
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
