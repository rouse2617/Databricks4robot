import { Button, Empty, Result, Spin, Typography } from "antd";
import type { ReactNode } from "react";

const { Text } = Typography;

export interface ContentLoadingStateProps {
	title: string;
	hint?: string;
	minHeight?: number | string;
}

/** In-content loading with context (prefer over bare spinner). */
export function ContentLoadingState({
	title,
	hint = "通常需要 1–3 秒",
	minHeight = 240,
}: ContentLoadingStateProps) {
	return (
		<div
			className="page-content-state page-content-state--loading"
			style={{ minHeight, display: "grid", placeItems: "center" }}
			role="status"
			aria-live="polite"
			aria-busy="true"
		>
			<div style={{ display: "grid", gap: 8, justifyItems: "center" }}>
				<Spin size="large" />
				<Text strong>{title}</Text>
				{hint ? (
					<Text type="secondary" style={{ fontSize: 12 }}>
						{hint}
					</Text>
				) : null}
			</div>
		</div>
	);
}

export interface ContentEmptyStateProps {
	title: string;
	description?: ReactNode;
	actions?: ReactNode;
}

export function ContentEmptyState({
	title,
	description,
	actions,
}: ContentEmptyStateProps) {
	return (
		<Empty
			image={Empty.PRESENTED_IMAGE_SIMPLE}
			description={
				<div style={{ display: "grid", gap: 4 }}>
					<span>{title}</span>
					{description ? (
						<Text type="secondary" style={{ fontSize: 12 }}>
							{description}
						</Text>
					) : null}
				</div>
			}
		>
			{actions}
		</Empty>
	);
}

export interface ContentErrorStateProps {
	title?: string;
	description?: string;
	onRetry?: () => void;
}

export function ContentErrorState({
	title = "加载失败",
	description,
	onRetry,
}: ContentErrorStateProps) {
	return (
		<Result
			status="error"
			title={title}
			subTitle={description}
			extra={
				onRetry ? (
					<Button type="primary" onClick={onRetry}>
						重试
					</Button>
				) : undefined
			}
		/>
	);
}
