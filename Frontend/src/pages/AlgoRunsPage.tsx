import { ReloadOutlined } from "@ant-design/icons";
import {
	Button,
	DatePicker,
	Select,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import dayjs from "dayjs";
import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { type AlgoRegistryItem, algoRegistryApi } from "../api/algoRegistry";
import { type AlgoRun, algoRunsApi } from "../api/algoRuns";
import { formatDateTime } from "../lib/dateTime";

const { Title } = Typography;
const { RangePicker } = DatePicker;

const STATUS_OPTIONS = [
	{ label: "全部", value: "" },
	{ label: "待处理", value: "pending" },
	{ label: "运行中", value: "running" },
	{ label: "成功", value: "ok" },
	{ label: "失败", value: "failed" },
	{ label: "已取消", value: "cancelled" },
];

const STATUS_COLOR: Record<string, string> = {
	pending: "default",
	running: "processing",
	ok: "success",
	failed: "error",
	cancelled: "warning",
};

export default function AlgoRunsPage() {
	const navigate = useNavigate();
	const [items, setItems] = useState<AlgoRun[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(20);
	const [status, setStatus] = useState("");
	const [algoName, setAlgoName] = useState("");
	const [dateRange, setDateRange] = useState<
		[dayjs.Dayjs | null, dayjs.Dayjs | null]
	>([null, null]);
	const [loading, setLoading] = useState(false);
	const [registry, setRegistry] = useState<AlgoRegistryItem[]>([]);

	useEffect(() => {
		algoRegistryApi
			.list()
			.then(setRegistry)
			.catch(() => {});
	}, []);

	const fetchData = useCallback(async () => {
		setLoading(true);
		try {
			const res = await algoRunsApi.list({
				page,
				page_size: pageSize,
				algo_name: algoName || undefined,
				status: status || undefined,
				started_after: dateRange[0]?.toISOString(),
				started_before: dateRange[1]?.toISOString(),
			});
			setItems(res.items ?? []);
			setTotal(res.total);
		} catch {
			// API may not be available yet; show empty state
			setItems([]);
			setTotal(0);
		} finally {
			setLoading(false);
		}
	}, [page, pageSize, status, algoName, dateRange]);

	useEffect(() => {
		fetchData();
	}, [fetchData]);

	// Deduplicate algo names from registry for the dropdown
	const algoNameOptions = [
		{ label: "全部算法", value: "" },
		...registry.map((r) => ({ label: `${r.name}@${r.version}`, value: r.key })),
	];

	const columns = [
		{
			title: "Run ID",
			dataIndex: "run_id",
			key: "run_id",
			width: 200,
			ellipsis: true,
			render: (id: string) => (
				<Link
					to={`/algo-runs/${id}`}
					onClick={(e) => e.stopPropagation()}
					className="font-mono text-xs"
				>
					{id}
				</Link>
			),
		},
		{
			title: "算法",
			dataIndex: "algo_name",
			key: "algo_name",
			width: 180,
			render: (name: string, record: AlgoRun) => (
				<code className="text-xs">
					{name}@{record.algo_version}
				</code>
			),
		},
		{
			title: "状态",
			dataIndex: "status",
			key: "status",
			width: 100,
			render: (val: string) => (
				<Tag color={STATUS_COLOR[val] ?? "default"}>{val}</Tag>
			),
		},
		{
			title: "开始时间",
			dataIndex: "started_at",
			key: "started_at",
			width: 180,
			render: (val?: string) => formatDateTime(val),
		},
		{
			title: "耗时",
			key: "duration",
			width: 140,
			render: (_: unknown, record: AlgoRun) => {
				if (!record.started_at || !record.finished_at) return "—";
				const sec = dayjs(record.finished_at).diff(
					dayjs(record.started_at),
					"second",
				);
				if (sec < 60) return `${sec}s`;
				const min = Math.floor(sec / 60);
				return `${min}m ${sec % 60}s`;
			},
		},
		{
			title: "处理资产",
			dataIndex: "assets_processed",
			key: "assets_processed",
			width: 100,
			render: (val?: number) => val ?? "—",
		},
		{
			title: "触发方",
			dataIndex: "triggered_by",
			key: "triggered_by",
			width: 120,
		},
		{
			title: "创建时间",
			dataIndex: "created_at",
			key: "created_at",
			width: 180,
			render: (val?: string) => formatDateTime(val),
		},
	];

	return (
		<div>
			<Title level={4} style={{ margin: "0 0 16px 0" }}>
				算法运行记录
			</Title>

			{/* Toolbar: filters + refresh */}
			<Space style={{ marginBottom: 12 }} wrap>
				<Select
					id="algo-name-filter"
					value={algoName}
					onChange={(v) => {
						setAlgoName(v);
						setPage(1);
					}}
					options={algoNameOptions}
					style={{ width: 200 }}
					placeholder="算法筛选"
					showSearch
					optionFilterProp="label"
				/>
				<Select
					id="algo-status-filter"
					value={status}
					onChange={(v) => {
						setStatus(v);
						setPage(1);
					}}
					options={STATUS_OPTIONS}
					style={{ width: 140 }}
					placeholder="状态过滤"
				/>
				<RangePicker
					id="algo-date-range"
					value={dateRange}
					onChange={(dates) => {
						setDateRange(dates as [dayjs.Dayjs | null, dayjs.Dayjs | null]);
						setPage(1);
					}}
					placeholder={["开始日期", "结束日期"]}
					style={{ width: 260 }}
				/>
				<Button icon={<ReloadOutlined />} onClick={fetchData}>
					刷新
				</Button>
			</Space>

			<Table
				rowKey="run_id"
				columns={columns}
				dataSource={items}
				loading={loading}
				pagination={{
					current: page,
					pageSize,
					total,
					showSizeChanger: true,
					showQuickJumper: total > 200,
					showTotal: (t) => `共 ${t} 条`,
					onChange: (p, ps) => {
						setPage(p);
						setPageSize(ps);
					},
				}}
				onRow={(record) => ({
					onClick: (e) => {
						const el = e.target as HTMLElement;
						if (
							el.closest(
								"a, button, input, textarea, select, label, [role='checkbox'], .ant-pagination, .ant-select, .ant-pagination-item",
							)
						) {
							return;
						}
						navigate(`/algo-runs/${record.run_id}`);
					},
					style: { cursor: "pointer" },
				})}
				size="middle"
				scroll={{ x: 1200 }}
			/>
		</div>
	);
}
