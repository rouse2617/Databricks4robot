import { Alert, Button, Card, Empty, Space, Tag, Typography } from "antd";
import type { QueryDebugPlan, QueryRequest } from "../../api/query";

const { Text } = Typography;
const ENGINE_LABELS: Record<string, string> = {
	postgres: "PostgreSQL",
	elasticsearch: "Elasticsearch",
};

export interface QueryInspectorPanelProps {
	open: boolean;
	onToggle: () => void;
	validation: {
		fetchStatus: "idle" | "loading" | "success" | "error";
		valid: boolean;
		error: string | null;
		normalizedQuery: QueryRequest | null;
		warnings: string[];
		fieldCapabilities: Array<{ field: string; engines: string[] }>;
		debugPlan: QueryDebugPlan | null;
	};
	runDebugPlan?: QueryDebugPlan;
	runWarnings?: string[];
}

function renderJson(value: unknown) {
	return (
		<pre
			style={{
				margin: 0,
				padding: 12,
				background: "#0F172A",
				color: "#E2E8F0",
				borderRadius: 8,
				overflowX: "auto",
				fontSize: 12,
			}}
		>
			{JSON.stringify(value, null, 2)}
		</pre>
	);
}

export default function QueryInspectorPanel({
	open,
	onToggle,
	validation,
	runDebugPlan,
	runWarnings = [],
}: QueryInspectorPanelProps) {
	return (
		<Card
			size="small"
			title="查询诊断（Query Inspector）"
			extra={
				<Button size="small" onClick={onToggle}>
					{open ? "收起" : "展开"}
				</Button>
			}
			style={{ marginBottom: 12 }}
		>
			{!open ? (
				<Text type="secondary">
					用于查看查询是否有效、字段由哪个引擎执行、以及最终执行计划。
				</Text>
			) : validation.fetchStatus === "idle" && validation.normalizedQuery == null ? (
				<Empty
					image={Empty.PRESENTED_IMAGE_SIMPLE}
					description="执行一次查询后，这里会显示校验结果与执行计划。"
				/>
			) : (
				<div style={{ display: "grid", gap: 12 }}>
					{validation.error && (
						<Alert type="error" showIcon message="查询校验失败" description={validation.error} />
					)}

					{validation.warnings.length > 0 && (
						<Alert
							type="warning"
							showIcon
							message="校验提示"
							description={validation.warnings.join("；")}
						/>
					)}

					{runWarnings.length > 0 && (
						<Alert
							type="info"
							showIcon
							message="执行提示"
							description={runWarnings.join("；")}
						/>
					)}

					<div>
						<Text strong>状态</Text>
						<div style={{ marginTop: 6 }}>
							<Tag color={validation.valid ? "success" : "default"}>
								{validation.valid ? "校验通过" : validation.fetchStatus}
							</Tag>
							<Text type="secondary" style={{ marginInlineStart: 8 }}>
								{validation.valid
									? "查询结构和字段能力检查通过"
									: "当前查询还未通过校验"}
							</Text>
						</div>
					</div>

					<div>
						<Text strong>字段能力（Field Capabilities）</Text>
						<div style={{ marginTop: 6, display: "grid", gap: 6 }}>
							{validation.fieldCapabilities.length === 0 ? (
								<Text type="secondary">暂无字段能力信息</Text>
							) : (
								validation.fieldCapabilities.map((item) => (
									<Space key={item.field} size={6} wrap>
										<Tag color="blue">{item.field}</Tag>
										{item.engines.map((engine) => (
											<Tag key={`${item.field}-${engine}`}>
												{ENGINE_LABELS[engine] ?? engine}
											</Tag>
										))}
									</Space>
								))
							)}
						</div>
						<Text type="secondary" style={{ fontSize: 12 }}>
							表示该字段可由哪些后端引擎直接执行（下推）过滤/排序。
						</Text>
					</div>

					<div>
						<Text strong>规范化请求（Normalized Query）</Text>
						<div style={{ marginTop: 6 }}>
							{validation.normalizedQuery ? (
								renderJson(validation.normalizedQuery)
							) : (
								<Text type="secondary">暂无</Text>
							)}
						</div>
					</div>

					<div>
						<Text strong>校验阶段计划（Validate Debug Plan）</Text>
						<div style={{ marginTop: 6 }}>
							{validation.debugPlan ? (
								renderJson(validation.debugPlan)
							) : (
								<Text type="secondary">暂无</Text>
							)}
						</div>
					</div>

					<div>
						<Text strong>执行阶段计划（Run Debug Plan）</Text>
						<div style={{ marginTop: 6 }}>
							{runDebugPlan ? renderJson(runDebugPlan) : <Text type="secondary">暂无</Text>}
						</div>
					</div>
				</div>
			)}
		</Card>
	);
}
