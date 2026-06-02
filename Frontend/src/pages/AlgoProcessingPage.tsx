import { RedoOutlined, RobotOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Checkbox,
	Modal,
	Progress,
	Space,
	Spin,
	Tooltip,
	Typography,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import { type AlgoRegistryItem, algoRegistryApi } from "../api/algoRegistry";
import { assetsApi } from "../api/assets";
import type { Asset } from "../api/types";
import AlgoMatrixGrid from "../components/algo-matrix/AlgoMatrixGrid";
import {
	collectFailedPairs,
	useRetryAllFailed,
} from "../hooks/algo-matrix/useRetryAllFailed";
import { type CellStatus, getAlgoStatusFromResults } from "../lib/algoStatus";

const { Title } = Typography;

const STATUS_OPTIONS: { label: string; value: CellStatus; color: string }[] = [
	{ label: "成功", value: "ok", color: "#16a34a" },
	{ label: "失败", value: "failed", color: "#dc2626" },
	{ label: "运行中", value: "running", color: "#d97706" },
	{ label: "待处理", value: "pending", color: "#64748b" },
	{ label: "已阻塞", value: "blocked", color: "#64748b" },
];

export default function AlgoProcessingPage() {
	const [algorithms, setAlgorithms] = useState<AlgoRegistryItem[]>([]);
	const [assets, setAssets] = useState<Asset[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(50);
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [statusFilter, setStatusFilter] = useState<CellStatus[]>([]);

	const {
		executing: retrying,
		progress: retryProgress,
		total: retryTotal,
		execute: executeRetry,
	} = useRetryAllFailed();

	// Fetch algo registry once
	useEffect(() => {
		algoRegistryApi
			.list()
			.then(setAlgorithms)
			.catch(() => {
				setError("无法加载算法注册表");
			});
	}, []);

	// Fetch assets (paginated)
	const fetchAssets = useCallback(async () => {
		setLoading(true);
		setError(null);
		try {
			const res = await assetsApi.list({ page, page_size: pageSize });
			setAssets(res.items ?? []);
			setTotal(res.total);
		} catch {
			setError("加载资产数据失败");
		} finally {
			setLoading(false);
		}
	}, [page, pageSize]);

	useEffect(() => {
		fetchAssets();
	}, [fetchAssets]);

	// Client-side status filtering
	const filteredAssets =
		statusFilter.length === 0
			? assets
			: assets.filter((asset) =>
					algorithms.some((algo) => {
						const s = getAlgoStatusFromResults(asset.algo_results, algo.key);
						return statusFilter.includes(s);
					}),
				);

	const handlePageChange = (newPage: number, newSize: number) => {
		setPage(newPage);
		setPageSize(newSize);
	};

	// When applying a filter, keep the UI consistent by treating the "total" as the
	// filtered result size for the currently loaded page (this page is client-side filtered).
	const effectiveTotal =
		statusFilter.length === 0 ? total : filteredAssets.length;

	// Collect failed pairs from currently displayed (filtered) assets
	const failedPairs = collectFailedPairs(filteredAssets, algorithms);

	const handleRetryAllFailed = async () => {
		if (failedPairs.length === 0) return;
		const result = await executeRetry(failedPairs);
		Modal.info({
			title: "批量重试完成",
			content: (
				<div>
					<p>成功：{result.success} 个</p>
					<p>失败：{result.failed} 个</p>
				</div>
			),
			okText: "确定",
			onOk: fetchAssets,
		});
	};

	return (
		<div>
			<Title level={4} style={{ margin: 0, marginBottom: 8 }}>
				<RobotOutlined style={{ marginRight: 8 }} />
				算法处理矩阵
			</Title>
			<Typography.Paragraph
				type="secondary"
				style={{ margin: 0, marginBottom: 12, fontSize: 12 }}
			>
				每个单元格显示该 asset
				对应算法的处理结果。鼠标悬停查看详情，点击有运行历史的格子可查看日志/参数。
			</Typography.Paragraph>

			{error && (
				<Alert
					message={error}
					type="error"
					showIcon
					closable
					style={{ marginBottom: 16 }}
				/>
			)}

			<Card
				size="small"
				style={{ marginBottom: 16 }}
				styles={{ body: { padding: "8px 16px" } }}
			>
				<Space
					wrap
					style={{ width: "100%", marginBottom: 8 }}
					aria-label="状态图例"
				>
					<span style={{ fontWeight: 500, fontSize: 12 }}>图例：</span>
					{STATUS_OPTIONS.map((opt) => (
						<span
							key={opt.value}
							style={{
								display: "inline-flex",
								alignItems: "center",
								gap: 4,
								fontSize: 12,
								color: opt.color,
							}}
						>
							<span
								style={{
									display: "inline-block",
									width: 8,
									height: 8,
									borderRadius: 2,
									backgroundColor: opt.color,
								}}
							/>
							{opt.label}
						</span>
					))}
				</Space>
				<Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
					<Space wrap>
						<span style={{ fontWeight: 500 }}>状态过滤：</span>
						<Checkbox.Group
							options={STATUS_OPTIONS}
							value={statusFilter}
							onChange={(vals) => {
								setStatusFilter(vals as CellStatus[]);
								setPage(1);
							}}
						/>
					</Space>
					<Tooltip
						title={
							failedPairs.length === 0
								? "当前筛选结果中没有失败任务"
								: undefined
						}
					>
						<span>
							<Button
								type="primary"
								danger
								icon={<RedoOutlined />}
								loading={retrying}
								disabled={failedPairs.length === 0}
								onClick={handleRetryAllFailed}
							>
								重试全部失败
								{failedPairs.length > 0 ? ` (${failedPairs.length})` : ""}
							</Button>
						</span>
					</Tooltip>
				</Space>
				{retrying && (
					<Progress
						percent={retryProgress}
						status="active"
						size="small"
						style={{ marginTop: 8 }}
						format={() =>
							`${Math.round((retryProgress / 100) * retryTotal)}/${retryTotal}`
						}
					/>
				)}
			</Card>

			{loading && algorithms.length === 0 ? (
				<div style={{ textAlign: "center", padding: 60 }}>
					<Spin size="large" />
				</div>
			) : (
				<AlgoMatrixGrid
					assets={filteredAssets}
					algorithms={algorithms}
					loading={loading}
					total={effectiveTotal}
					page={page}
					pageSize={pageSize}
					onPageChange={handlePageChange}
					onRefresh={fetchAssets}
					onResetFilters={() => setStatusFilter([])}
				/>
			)}
		</div>
	);
}
