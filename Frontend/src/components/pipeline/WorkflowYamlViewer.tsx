import { Button, Modal } from "antd";
import { Fragment, useMemo } from "react";
import type { WorkflowNodeStatus } from "../../api/workflowApi";

type WorkflowYamlViewerProps = {
	nodeData: WorkflowNodeStatus | null;
	onClose: () => void;
};

const tokenStyles = {
	key: { color: "#2f81f7", fontWeight: 600 },
	value: { color: "#2f9e44", fontWeight: 500 },
	string: { color: "#f08c00" },
	number: { color: "#2f9e44", fontWeight: 500 },
	boolean: { color: "#2f9e44", fontWeight: 500 },
	null: { color: "#999", fontStyle: "italic" },
	punct: { color: "#64748b" },
};

function formatJsonValue(value: unknown, indent: number = 0): React.ReactNode {
	if (value === null) {
		return <span style={tokenStyles.null}>null</span>;
	}
	if (typeof value === "string") {
		return <span style={tokenStyles.string}>{JSON.stringify(value)}</span>;
	}
	if (typeof value === "number") {
		return <span style={tokenStyles.number}>{String(value)}</span>;
	}
	if (typeof value === "boolean") {
		return <span style={tokenStyles.boolean}>{String(value)}</span>;
	}
	if (value instanceof Date) {
		return (
			<span style={tokenStyles.string}>
				{JSON.stringify(value.toISOString())}
			</span>
		);
	}
	if (Array.isArray(value)) {
		if (value.length === 0) {
			return (
				<>
					<span style={tokenStyles.punct}>[</span>
					<span style={tokenStyles.punct}>]</span>
				</>
			);
		}
		const childIndent = indent + 1;
		return (
			<>
				<span style={tokenStyles.punct}>[</span>
				{value.map((item, idx) => {
					const elementKey = `json-${idx}`;
					return (
						<Fragment key={elementKey}>
							<br />
							<span style={{ whiteSpace: "pre" }}>
								{"  ".repeat(childIndent)}
							</span>
							{formatJsonValue(item, childIndent)}
							{idx < value.length - 1 ? (
								<span style={tokenStyles.punct}>,</span>
							) : null}
						</Fragment>
					);
				})}
				<br />
				<span style={{ whiteSpace: "pre" }}>{"  ".repeat(indent)}</span>
				<span style={tokenStyles.punct}>]</span>
			</>
		);
	}
	if (typeof value === "object") {
		const entries = Object.entries(value);
		if (entries.length === 0) {
			return (
				<>
					<span style={tokenStyles.punct}>{"{"}</span>
					<span style={tokenStyles.punct}>{"}"}</span>
				</>
			);
		}

		const childIndent = indent + 1;
		return (
			<>
				<span style={tokenStyles.punct}>{"{"}</span>
				{entries.map(([entryKey, entryValue], index) => (
					<Fragment key={`${entryKey}`}>
						<br />
						<span style={{ whiteSpace: "pre" }}>
							{"  ".repeat(childIndent)}
						</span>
						<span style={tokenStyles.key}>{JSON.stringify(entryKey)}</span>
						<span style={tokenStyles.punct}>: </span>
						{formatJsonValue(entryValue, childIndent)}
						{index < entries.length - 1 ? (
							<span style={tokenStyles.punct}>,</span>
						) : null}
					</Fragment>
				))}
				<br />
				<span style={{ whiteSpace: "pre" }}>{"  ".repeat(indent)}</span>
				<span style={tokenStyles.punct}>{"}"}</span>
			</>
		);
	}

	return <span style={tokenStyles.value}>{String(value)}</span>;
}

export function WorkflowYamlViewer({
	nodeData,
	onClose,
}: WorkflowYamlViewerProps) {
	const title = useMemo(
		() => `${nodeData?.displayName || nodeData?.name || "Node"} YAML`,
		[nodeData?.displayName, nodeData?.name],
	);

	return (
		<Modal
			open={!!nodeData}
			title={title}
			width={800}
			onCancel={onClose}
			footer={
				<Button type="primary" onClick={onClose}>
					关闭
				</Button>
			}
		>
			<div style={{ maxHeight: "70vh", overflow: "auto" }}>
				<pre
					style={{
						margin: 0,
						padding: 12,
						background: "#0f172a",
						color: "#f8fafc",
						borderRadius: 6,
						fontFamily:
							'"SF Mono", "Fira Code", "Consolas", "Liberation Mono", Menlo, monospace',
						fontSize: 12,
						lineHeight: 1.5,
						whiteSpace: "pre",
					}}
				>
					{nodeData ? formatJsonValue(nodeData) : null}
				</pre>
			</div>
		</Modal>
	);
}
