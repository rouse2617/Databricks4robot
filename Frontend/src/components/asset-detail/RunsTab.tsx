// ─── RunsTab — Pipeline runs that used this asset ───
// CYB-4297: reverse "asset → pipeline runs" view.
//
// Business context: users identify assets by grace_video_id but pipeline runs
// only stored asset_ids server-side. Until now you could dispatch a pipeline
// FROM an asset (via /pipeline?asset_ids=…) but never see WHAT ran ON an
// asset. This tab closes that loop by calling the new
// GET /assets/:id/runs endpoint (backend resolves both id forms).

import { Card, Empty, message, Select, Space, Spin, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
	listPipelineRunsByAsset,
	type PipelineRun,
} from "../../api/pipelineApi";
import { formatDateTime } from "../../lib/dateTime";

interface Props {
	assetId: string;
}

// Match the status vocabulary used across the pipeline list pages so users
// don't see one thing here and another on /runs.
const STATUS_COLOR: Record<string, string> = {
	Succeeded: "success",
	Running: "processing",
	Pending: "default",
	Failed: "error",
	Error: "error",
	Cancelled: "warning",
	Suspended: "warning",
};

const STATUS_OPTIONS = [
	{ label: "全部状态", value: "" },
	{ label: "运行中", value: "Running" },
	{ label: "等待中", value: "Pending" },
	{ label: "成功", value: "Succeeded" },
	{ label: "失败", value: "Failed" },
	{ label: "已取消", value: "Cancelled" },
];

const PAGE_SIZE = 20;

export default function RunsTab({ assetId }: Props) {
	const [loading, setLoading] = useState(true);
	const [items, setItems] = useState<PipelineRun[]>([]);
	const [total, setTotal] = useState(0);
	const [page, setPage] = useState(1);
	const [status, setStatus] = useState<string>("");

	useEffect(() => {
		let cancelled = false;
		const run = async () => {
			setLoading(true);
			try {
				const res = await listPipelineRunsByAsset(assetId, {
					status: status || undefined,
					page,
					pageSize: PAGE_SIZE,
				});
				if (cancelled) return;
				setItems(res.items ?? []);
				setTotal(res.total ?? 0);
			} catch (err) {
				if (cancelled) return;
				message.error("加载运行历史失败");
				setItems([]);
				setTotal(0);
			} finally {
				if (!cancelled) setLoading(false);
			}
		};
		run();
		return () => {
			cancelled = true;
		};
	}, [assetId, status, page]);

	const columns: ColumnsType<PipelineRun> = useMemo(
		() => [
			{
				title: "流水线",
				key: "pipeline",
				render: (_v, r) => {
					const name =
						r.pipelineName || r.templateName || r.workflowName || r.id;
					return (
						<Link to={`/runs/${encodeURIComponent(r.id)}`}>{name}</Link>
					);
				},
			},
			{
				title: "状态",
				dataIndex: "status",
				key: "status",
				width: 120,
				render: (s?: string) => {
					const label = s || "—";
					return (
						<Tag color={STATUS_COLOR[label] ?? "default"}>{label}</Tag>
					);
				},
			},
			{
				title: "批次",
				dataIndex: "batchJobId",
				key: "batchJobId",
				width: 140,
				render: (batchId?: string) => {
					if (!batchId) return <span style={{ color: "#94a3b8" }}>—</span>;
					return (
						<Link to={`/pipeline/batch/${encodeURIComponent(batchId)}`}>
							{batchId.slice(0, 8)}
						</Link>
					);
				},
			},
			{
				title: "目标",
				dataIndex: "executionTargetId",
				key: "executionTargetId",
				width: 160,
				render: (v?: string) => v || "—",
			},
			{
				title: "进度",
				dataIndex: "progress",
				key: "progress",
				width: 90,
				render: (v?: string) => v || "—",
			},
			{
				title: "优先级",
				dataIndex: "priority",
				key: "priority",
				width: 90,
				render: (v?: number | null) =>
					typeof v === "number" ? v : <span style={{ color: "#94a3b8" }}>—</span>,
			},
			{
				title: "开始时间",
				dataIndex: "startedAt",
				key: "startedAt",
				width: 170,
				render: (v?: string) => formatDateTime(v),
			},
			{
				title: "结束时间",
				dataIndex: "finishedAt",
				key: "finishedAt",
				width: 170,
				render: (v?: string) => formatDateTime(v),
			},
		],
		[],
	);

	if (loading && items.length === 0) {
		return (
			<Card size="small">
				<div className="flex justify-center py-8">
					<Spin />
				</div>
			</Card>
		);
	}

	// True empty state — asset has literally never been in any pipeline_run.
	// Distinct from "filtered result is empty" which keeps toolbar visible.
	if (!loading && total === 0 && !status) {
		return (
			<Card size="small">
				<Empty
					description={
						<span>
							该资产尚未参与任何流水线运行 ·{" "}
							<Link
								to={`/pipeline?tab=pipelines&asset_ids=${encodeURIComponent(
									assetId,
								)}`}
							>
								前往下发
							</Link>
						</span>
					}
				/>
			</Card>
		);
	}

	return (
		<Card size="small">
			<Space style={{ marginBottom: 12 }}>
				<Select
					value={status}
					onChange={(v) => {
						setStatus(v);
						setPage(1);
					}}
					options={STATUS_OPTIONS}
					style={{ width: 140 }}
					size="small"
				/>
			</Space>
			<Table<PipelineRun>
				size="small"
				rowKey="id"
				loading={loading}
				columns={columns}
				dataSource={items}
				pagination={{
					current: page,
					pageSize: PAGE_SIZE,
					total,
					showTotal: (t) => `共 ${t} 条`,
					showSizeChanger: false,
					onChange: (p) => setPage(p),
				}}
			/>
		</Card>
	);
}
