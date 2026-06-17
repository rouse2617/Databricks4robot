import { Alert, Card, Space, Table, Tabs, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useState } from "react";
import type { AlgoRegistryItem } from "../api/algoRegistry";
import { type MetricRegistryItem, registryApi } from "../api/registry";
import type { TagRegistryItem } from "../api/tagRegistry";
import { ContentErrorState, ContentLoadingState } from "../components/common/PageContentState";
import { COLUMN_LABELS } from "../lib/productVocabulary";

const { Title, Text } = Typography;

export default function RegistryCenterPage() {
	const [algos, setAlgos] = useState<AlgoRegistryItem[]>([]);
	const [tags, setTags] = useState<TagRegistryItem[]>([]);
	const [metrics, setMetrics] = useState<MetricRegistryItem[]>([]);
	const [states, setStates] = useState<string[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		let cancelled = false;
		(async () => {
			try {
				setLoading(true);
				const [a, t, m, s] = await Promise.all([
					registryApi.listAlgos(),
					registryApi.listTags(),
					registryApi.listMetrics(),
					registryApi.listLifecycleStates(),
				]);
				if (cancelled) return;
				setAlgos(a);
				setTags(t);
				setMetrics(m);
				setStates(s);
			} catch {
				if (!cancelled) setError("加载注册中心失败");
			} finally {
				if (!cancelled) setLoading(false);
			}
		})();
		return () => {
			cancelled = true;
		};
	}, []);

	const algoCols: ColumnsType<AlgoRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{ title: "名称", dataIndex: "name" },
		{ title: "版本", dataIndex: "version" },
		{
			title: COLUMN_LABELS.dependsOn,
			dataIndex: "depends_on",
			render: (v: string[]) => v?.join(", ") || "—",
		},
	];
	const tagCols: ColumnsType<TagRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{ title: "类型", dataIndex: "type" },
		{
			title: "说明",
			dataIndex: "description",
			render: (v?: string) => v || "—",
		},
	];
	const metricCols: ColumnsType<MetricRegistryItem> = [
		{ title: "Key", dataIndex: "key", render: (v) => <Text code>{v}</Text> },
		{
			title: "展示名",
			dataIndex: "display_name",
			render: (v?: string) => v || "—",
		},
		{ title: "类型", dataIndex: "metric_type" },
		{
			title: COLUMN_LABELS.queryable,
			dataIndex: "queryable",
			render: (v?: boolean) =>
				v ? <Tag color="green">是</Tag> : <Tag>否</Tag>,
		},
	];

	return (
		<div>
			<Title level={4} style={{ marginTop: 0 }}>
				注册中心
			</Title>
			<Alert
				type="info"
				showIcon
				style={{ marginBottom: 16 }}
				message="数据模型与业务能力纳管"
				description="统一展示 Algo/Tag/Metric/Lifecycle 四类注册表，作为 API、CDC、搜索与 UI 的口径来源。"
			/>
			{error ? (
				<ContentErrorState
					title="注册表加载失败"
					description={error}
					onRetry={() => window.location.reload()}
				/>
			) : loading ? (
				<ContentLoadingState title="正在加载注册表…" />
			) : (
				<Card size="small">
				<Tabs
					items={[
						{
							key: "lifecycle",
							label: `Lifecycle (${states.length})`,
							children: (
								<>
									<Typography.Paragraph
										type="secondary"
										style={{ margin: 0, marginBottom: 8, fontSize: 12 }}
									>
										资产生命周期状态枚举，源自 <code>lifecycle_states</code>{" "}
										配置； 后端 <code>chk_lifecycle_state</code>{" "}
										约束以此为口径。
									</Typography.Paragraph>
									<Space wrap>
										{states.map((s) => (
											<Tag key={s} color="blue">
												{s}
											</Tag>
										))}
									</Space>
								</>
							),
						},
						{
							key: "metrics",
							label: `Metrics (${metrics.length})`,
							children: (
								<>
									<Typography.Paragraph
										type="secondary"
										style={{ margin: 0, marginBottom: 8, fontSize: 12 }}
									>
										评估指标定义。queryable=yes 的指标在「指标检索」页面可用。
									</Typography.Paragraph>
									<Table
										rowKey="key"
										pagination={false}
										size="small"
										columns={metricCols}
										dataSource={metrics}
									/>
								</>
							),
						},
						{
							key: "tags",
							label: `Tags (${tags.length})`,
							children: (
								<>
									<Typography.Paragraph
										type="secondary"
										style={{ margin: 0, marginBottom: 8, fontSize: 12 }}
									>
										资产标签类型与允许取值，可由资产管理界面写入。
									</Typography.Paragraph>
									<Table
										rowKey="key"
										pagination={false}
										size="small"
										columns={tagCols}
										dataSource={tags}
									/>
								</>
							),
						},
						{
							key: "algos",
							label: `Algos (${algos.length})`,
							children: (
								<>
									<Typography.Paragraph
										type="secondary"
										style={{ margin: 0, marginBottom: 8, fontSize: 12 }}
									>
										算法注册项与依赖关系，决定 pipeline 节点可引用的算法版本。
									</Typography.Paragraph>
									<Table
										rowKey="key"
										pagination={false}
										size="small"
										columns={algoCols}
										dataSource={algos}
									/>
								</>
							),
						},
					]}
				/>
			</Card>
			)}
		</div>
	);
}
