import { Tag } from "antd";
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
					<span>{value}</span>
				</Tag>
			))}
		</div>
	);
}
