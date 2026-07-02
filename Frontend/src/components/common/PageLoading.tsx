// Page-level loading spinner. Use for first-paint loads; use Ant `Spin`
// inline for in-place loading inside cards / drawers.

import { Spin } from "antd";

export interface PageLoadingProps {
	/** Vertical height; defaults to `60vh`. */
	height?: number | string;
	/** Optional helper text shown under the spinner. */
	tip?: string;
}

export default function PageLoading({
	height = "60vh",
	tip = "正在加载页面内容…",
}: PageLoadingProps) {
	return (
		<div
			className="flex items-center justify-center"
			style={{ height }}
			role="status"
			aria-live="polite"
			aria-busy="true"
		>
			<div style={{ display: "grid", gap: 8, justifyItems: "center" }}>
				<Spin size="large" />
				{tip ? (
					<span style={{ color: "#64748b", fontSize: 13 }}>{tip}</span>
				) : null}
			</div>
		</div>
	);
}
