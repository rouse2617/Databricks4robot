import { Tag, Typography } from "antd";
import {
	formatWorkflowLabelKey,
	getDisplayLabelEntries,
} from "../../lib/workflowLabels";

export function WorkflowLabels({
	labels,
}: {
	labels?: Record<string, string>;
}) {
	const entries = getDisplayLabelEntries(labels);
	if (entries.length === 0) {
		return <span style={{ color: "#9ca3af" }}>—</span>;
	}

	return (
		<div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
			{entries.map(([key, value]) => (
				<Tag key={key} style={{ marginInlineEnd: 0 }}>
					<span style={{ color: "#64748b" }}>
						{formatWorkflowLabelKey(key)}
					</span>
					<span style={{ margin: "0 3px", color: "#94a3b8" }}>·</span>
					{/* Asset ids are 36-char UUIDs that operators paste into
					    other tools (Grace lookup, queries/run filters), so give
					    them a one-click copy. Other labels stay plain text.
					    CYB-4470 B.1: 复制图标 hover 才显现 —— 用 .copy-cell
					    包装；CYB-4470 B.2: 列头已是"资产 ID"，单元格再写 "asset_id="
					    是冗余前缀 —— 渲染时去掉，复制的也是裸 ID。 */}
					{key === "asset_id" ? (
						<span className="copy-cell">
							<Typography.Text copyable={{ text: stripAssetIdPrefix(value) }}>
								{stripAssetIdPrefix(value)}
							</Typography.Text>
						</span>
					) : (
						<span>{value}</span>
					)}
				</Tag>
			))}
		</div>
	);
}

/**
 * 去掉 "asset_id=" 前缀（兼容老数据中可能仍带前缀的标签值）。
 * 列头已经叫"资产 ID"，单元格不应该再写一遍前缀。
 */
function stripAssetIdPrefix(value: string): string {
	return value.replace(/^asset_id=/, "");
}
