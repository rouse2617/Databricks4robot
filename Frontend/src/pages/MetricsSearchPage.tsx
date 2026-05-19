import {
	Alert,
	Button,
	Card,
	InputNumber,
	Select,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { evalApi, type MetricsSearchFilter } from "../api/eval";
import { type MetricRegistryItem, registryApi } from "../api/registry";

const { Title, Text } = Typography;

type Row = {
	id: string;
};

export default function MetricsSearchPage() {
	const [metrics, setMetrics] = useState<MetricRegistryItem[]>([]);
	const [lifecycleStates, setLifecycleStates] = useState<string[]>([]);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const [metricKey, setMetricKey] = useState<string>("");
	const [op, setOp] = useState<MetricsSearchFilter["op"]>("gte");
	const [value, setValue] = useState<number>(0.05);
	const [lifecycleState, setLifecycleState] = useState<string>("");

	const [rows, setRows] = useState<Row[]>([]);
	const [total, setTotal] = useState(0);

	useEffect(() => {
		let cancelled = false;
		(async () => {
			try {
				const [metricItems, states] = await Promise.all([
					registryApi.listMetrics(),
					registryApi.listLifecycleStates(),
				]);
				if (cancelled) return;
				const queryable = metricItems.filter((m) => m.queryable);
				setMetrics(queryable);
				setLifecycleStates(states);
				if (queryable.length > 0) setMetricKey(queryable[0].key);
			} catch {
				if (!cancelled) setError("加载注册表失败");
			}
		})();
		return () => {
			cancelled = true;
		};
	}, []);

	const metricOptions = useMemo(
		() =>
			metrics.map((m) => ({
				label: `${m.key}${m.display_name ? ` (${m.display_name})` : ""}`,
				value: m.key,
			})),
		[metrics],
	);

	const runSearch = async () => {
		if (!metricKey) return;
		setLoading(true);
		setError(null);
		try {
			const resp = await evalApi.searchByMetrics(
				[{ metric_key: metricKey, op, value }],
				lifecycleState || undefined,
				1,
				100,
			);
			setRows((resp.asset_ids ?? []).map((id) => ({ id })));
			setTotal(resp.total ?? 0);
		} catch {
			setRows([]);
			setTotal(0);
			setError("指标检索失败，请检查后端状态");
		} finally {
			setLoading(false);
		}
	};

	return (
		<div>
			<Title level={4} style={{ marginTop: 0 }}>
				指标检索
			</Title>
			<Alert
				type="info"
				showIcon
				style={{ marginBottom: 16 }}
				message="按已写入的评估指标筛选资产"
				description="仅支持注册表中标记为可检索的指标。设置阈值与可选的生命周期条件后，查询满足条件的资产列表。"
			/>

			<Card size="small" style={{ marginBottom: 16 }}>
				<Space wrap>
					<Select
						style={{ width: 360 }}
						placeholder="选择指标"
						options={metricOptions}
						value={metricKey || undefined}
						onChange={setMetricKey}
					/>
					<Select
						value={op}
						onChange={(v) => setOp(v)}
						options={[
							{ label: ">=", value: "gte" },
							{ label: ">", value: "gt" },
							{ label: "=", value: "eq" },
							{ label: "<", value: "lt" },
							{ label: "<=", value: "lte" },
						]}
						style={{ width: 90 }}
					/>
					<InputNumber
						value={value}
						onChange={(v) => setValue(typeof v === "number" ? v : 0)}
						step={0.01}
					/>
					<Select
						allowClear
						placeholder="生命周期（可选）"
						value={lifecycleState || undefined}
						onChange={(v) => setLifecycleState(v ?? "")}
						options={lifecycleStates.map((s) => ({ label: s, value: s }))}
						style={{ width: 180 }}
					/>
					<Button
						type="primary"
						onClick={() => void runSearch()}
						loading={loading}
					>
						检索
					</Button>
				</Space>
			</Card>

			{error && (
				<Alert
					type="error"
					showIcon
					message={error}
					style={{ marginBottom: 12 }}
				/>
			)}

			<Card
				size="small"
				title={
					<span>
						命中资产 <Tag color="blue">{total}</Tag>
					</span>
				}
			>
				<Table<Row>
					rowKey="id"
					loading={loading}
					dataSource={rows}
					pagination={false}
					columns={[
						{
							title: "Asset ID",
							dataIndex: "id",
							render: (id: string) => (
								<Space>
									<Text code>{id}</Text>
									<Link to={`/assets/${id}`}>打开详情</Link>
								</Space>
							),
						},
					]}
					locale={{ emptyText: "暂无结果" }}
				/>
			</Card>
		</div>
	);
}
