import { Button, Empty, List, Spin, Tag, Typography } from "antd";
import type { AssetMetricItem, EvalResultItem } from "../../api/eval";

const { Text } = Typography;

function formatMetricValue(metric: AssetMetricItem): string {
	if (metric.metric_value != null) return String(metric.metric_value);
	if (metric.metric_value_int != null) return String(metric.metric_value_int);
	if (metric.metric_value_text != null) return metric.metric_value_text;
	if (metric.metric_value_bool != null)
		return metric.metric_value_bool ? "true" : "false";
	return "-";
}

export default function EvalMetricsTab(props: {
	loading: boolean;
	evalResults: EvalResultItem[];
	metrics: AssetMetricItem[];
	onRefresh: () => void;
}) {
	const { loading, evalResults, metrics, onRefresh } = props;

	if (loading) {
		return <Spin />;
	}

	if (evalResults.length === 0 && metrics.length === 0) {
		return <Empty description="暂无评测结果或指标" />;
	}

	return (
		<div style={{ display: "grid", gap: 16 }}>
			<div className="flex items-center justify-between">
				<Text strong>评测结果</Text>
				<Button size="small" onClick={onRefresh}>
					刷新
				</Button>
			</div>
			<List
				size="small"
				dataSource={evalResults}
				locale={{ emptyText: "暂无评测结果" }}
				renderItem={(item) => (
					<List.Item>
						<div
							style={{
								width: "100%",
								display: "flex",
								justifyContent: "space-between",
								gap: 8,
							}}
						>
							<div>
								<Text strong>{item.eval_name}</Text>
								<Text type="secondary"> @{item.eval_version}</Text>
								<div>
									<Tag color={item.status === "ok" ? "green" : "orange"}>
										{item.status}
									</Tag>
									{item.run_id ? <Tag>{item.run_id}</Tag> : null}
								</div>
							</div>
							<Text type="secondary">{item.created_at}</Text>
						</div>
					</List.Item>
				)}
			/>

			<Text strong>可检索指标</Text>
			<List
				size="small"
				dataSource={metrics}
				locale={{ emptyText: "暂无指标投影" }}
				renderItem={(m) => (
					<List.Item>
						<div
							style={{
								width: "100%",
								display: "flex",
								justifyContent: "space-between",
								gap: 8,
							}}
						>
							<div>
								<Text code>{m.metric_key}</Text>
								<Text type="secondary"> ({m.metric_type})</Text>
								<div>
									<Text>
										{m.eval_name}@{m.eval_version}
									</Text>
								</div>
							</div>
							<Text strong>{formatMetricValue(m)}</Text>
						</div>
					</List.Item>
				)}
			/>
		</div>
	);
}
