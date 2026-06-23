import { Card, Descriptions, Tag } from "antd";
import { useEffect, useState } from "react";
import { getRun, type DatabrewRun } from "../api/runsApi";

const TYPE_LABELS: Record<string, string> = {
	pipeline: "流水线运行",
	component_build: "组件构建",
	rag_build: "RAG 构建",
};

export function RunOverview({
	runId,
	runType,
}: {
	runId: string;
	runType?: string;
}) {
	const [run, setRun] = useState<DatabrewRun | null>(null);
	const [loading, setLoading] = useState(true);

	useEffect(() => {
		let alive = true;
		setLoading(true);
		getRun(runId)
			.then((item) => {
				if (!alive) return;
				setRun(item);
			})
			.finally(() => {
				if (alive) setLoading(false);
			});
		return () => {
			alive = false;
		};
	}, [runId]);

	const type = runType ?? run?.type ?? "pipeline";
	const summary = (run?.summary ?? {}) as Record<string, string | number>;

	return (
		<Card
			size="small"
			loading={loading}
			title={TYPE_LABELS[type] ?? type}
			style={{ margin: "0 24px 12px" }}
		>
			<Descriptions size="small" column={3}>
				<Descriptions.Item label="Run ID">{runId}</Descriptions.Item>
				<Descriptions.Item label="状态">
					{run ? <Tag>{run.statusLabel || run.status}</Tag> : "-"}
				</Descriptions.Item>
				<Descriptions.Item label="Workflow">
					{run?.runtime.resourceName ?? "-"}
				</Descriptions.Item>
				{type === "component_build" ? (
					<>
						<Descriptions.Item label="组件">
							{String(summary.component ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="Commit">
							{String(summary.commit ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="镜像 Tag">
							{String(summary.imageTag ?? "-")}
						</Descriptions.Item>
					</>
				) : null}
				{type === "rag_build" ? (
					<>
						<Descriptions.Item label="知识库">
							{String(summary.knowledgeBaseId ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="Embedding">
							{String(summary.embeddingModel ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="版本">
							{String(summary.releaseVersion ?? "-")}
						</Descriptions.Item>
					</>
				) : null}
				{type === "pipeline" ? (
					<>
						<Descriptions.Item label="模板">
							{String(summary.pipelineName ?? summary.templateId ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="资产数">
							{String(summary.assetCount ?? "-")}
						</Descriptions.Item>
						<Descriptions.Item label="节点数">
							{String(summary.nodeCount ?? "-")}
						</Descriptions.Item>
					</>
				) : null}
			</Descriptions>
		</Card>
	);
}
