import { Button, Checkbox, Divider, Empty, Space } from "antd";
import { useState } from "react";
import type { PipelineTemplate } from "../api/pipelineApi";

interface ComparisonPanelProps {
	version1: PipelineTemplate;
	version2: PipelineTemplate;
	onClose: () => void;
}

export default function ComparisonPanel({
	version1,
	version2,
	onClose,
}: ComparisonPanelProps) {
	const [showOnlyDiff, setShowOnlyDiff] = useState(false);

	// Deep compare two objects to find differences
	const diff = findDifferences(version1.pipeline, version2.pipeline);

	if (Object.keys(diff).length === 0 && showOnlyDiff) {
		return (
			<div style={{ padding: "24px", textAlign: "center" }}>
				<Empty description="两个版本没有差异" />
				<Button onClick={onClose} style={{ marginTop: "16px" }}>
					关闭
				</Button>
			</div>
		);
	}

	return (
		<div style={{ padding: "24px" }}>
			<div
				style={{
					marginBottom: "16px",
					display: "flex",
					justifyContent: "space-between",
				}}
			>
				<h3>
					版本对比：v{version1.version} ({version1.updatedAt?.slice(0, 10)}) vs
					v{version2.version} ({version2.updatedAt?.slice(0, 10)})
				</h3>
				<Space>
					<Checkbox
						checked={showOnlyDiff}
						onChange={(e) => setShowOnlyDiff(e.target.checked)}
					>
						仅显示不同
					</Checkbox>
					<Button onClick={onClose}>关闭</Button>
				</Space>
			</div>

			<Divider />

			<div
				style={{
					display: "grid",
					gridTemplateColumns: "1fr 1fr",
					gap: "24px",
					maxHeight: "600px",
					overflow: "auto",
				}}
			>
				{/* Version 1 */}
				<div>
					<h4>版本 {version1.version}</h4>
					<pre
						style={{
							backgroundColor: "#f5f5f5",
							padding: "12px",
							borderRadius: "4px",
							overflow: "auto",
							maxHeight: "500px",
						}}
					>
						{JSON.stringify(version1.pipeline, null, 2)}
					</pre>
				</div>

				{/* Version 2 */}
				<div>
					<h4>版本 {version2.version}</h4>
					<pre
						style={{
							backgroundColor: "#f5f5f5",
							padding: "12px",
							borderRadius: "4px",
							overflow: "auto",
							maxHeight: "500px",
						}}
					>
						{JSON.stringify(version2.pipeline, null, 2)}
					</pre>
				</div>
			</div>

			{/* Diff summary */}
			{Object.keys(diff).length > 0 && (
				<>
					<Divider />
					<div style={{ marginTop: "16px" }}>
						<h4>变更摘要 ({Object.keys(diff).length} 项)</h4>
						<ul>
							{Object.entries(diff).map(([key, value]) => (
								<li key={key}>
									<strong>{key}</strong>: {value.from} → {value.to}
								</li>
							))}
						</ul>
					</div>
				</>
			)}
		</div>
	);
}

function findDifferences(
	obj1: any,
	obj2: any,
	path = "",
): Record<string, { from: string; to: string }> {
	const diff: Record<string, { from: string; to: string }> = {};

	const allKeys = new Set([
		...Object.keys(obj1 || {}),
		...Object.keys(obj2 || {}),
	]);

	for (const key of allKeys) {
		const currentPath = path ? `${path}.${key}` : key;
		const val1 = obj1?.[key];
		const val2 = obj2?.[key];

		if (JSON.stringify(val1) !== JSON.stringify(val2)) {
			diff[currentPath] = {
				from: JSON.stringify(val1),
				to: JSON.stringify(val2),
			};
		}
	}

	return diff;
}
