import {
	CloudUploadOutlined,
	FileOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Button,
	Empty,
	Input,
	Result,
	Select,
	Space,
	Table,
	Tag,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { mcapFilesApi } from "../api/mcapFiles";
import type { McapFile } from "../api/types";
import McapDetailDrawer from "../components/mcap/McapDetailDrawer";
import { extractApiErrorMessage } from "../lib/apiError";
import { formatDateTime } from "../lib/dateTime";

const { Title, Text } = Typography;

const ingestColor: Record<string, string> = {
	pending: "default",
	summarized: "success",
	failed: "error",
};

function formatBytes(bytes: number): string {
	if (!bytes || bytes === 0) return "—";
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024)
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

const stateFilterOptions = [
	{ value: "", label: "全部状态" },
	{ value: "pending", label: "pending" },
	{ value: "summarized", label: "summarized" },
	{ value: "failed", label: "failed" },
];

export default function McapFilesPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const focusedMcapFileId = searchParams.get("mcap_file_id")?.trim() ?? "";
	const [files, setFiles] = useState<McapFile[]>([]);
	const [total, setTotal] = useState(0);
	const [loading, setLoading] = useState(true);
	const [page, setPage] = useState(Number(searchParams.get("page")) || 1);
	const [stateFilter, setStateFilter] = useState("");
	const [ownerFilter, setOwnerFilter] = useState("");
	const [debouncedOwnerFilter, setDebouncedOwnerFilter] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [drawerOpen, setDrawerOpen] = useState(false);
	const [selectedFile, setSelectedFile] = useState<McapFile | null>(null);

	useEffect(() => {
		const timer = window.setTimeout(() => {
			setDebouncedOwnerFilter(ownerFilter.trim());
		}, 300);
		return () => window.clearTimeout(timer);
	}, [ownerFilter]);

	const load = useCallback(
		async (p = page) => {
			setLoading(true);
			setError(null);
			try {
				const data = await mcapFilesApi.list({
					page: p,
					page_size: 20,
					ingest_state: stateFilter || undefined,
					owner: debouncedOwnerFilter || undefined,
				});
				setFiles(data.items ?? []);
				setTotal(data.total ?? 0);
			} catch (err) {
				setError(extractApiErrorMessage(err, "加载 MCAP 文件失败"));
				setFiles([]);
				setTotal(0);
			} finally {
				setLoading(false);
			}
		},
		[page, stateFilter, debouncedOwnerFilter],
	);

	useEffect(() => {
		setPage(1);
	}, [stateFilter, debouncedOwnerFilter]);

	useEffect(() => {
		load(page);
	}, [page, load]);

	useEffect(() => {
		if (!focusedMcapFileId) return;
		const clearFocusParam = () => {
			const next = new URLSearchParams(searchParams);
			next.delete("mcap_file_id");
			setSearchParams(next, { replace: true });
		};

		const fromCurrentPage = files.find(
			(file) => file.mcap_file_id === focusedMcapFileId,
		);
		if (fromCurrentPage) {
			setSelectedFile(fromCurrentPage);
			setDrawerOpen(true);
			clearFocusParam();
			return;
		}

		let cancelled = false;
		void mcapFilesApi
			.get(focusedMcapFileId)
			.then((file) => {
				if (cancelled) return;
				setFiles((prev) => {
					if (prev.some((item) => item.mcap_file_id === file.mcap_file_id)) {
						return prev;
					}
					return [file, ...prev].slice(0, 20);
				});
				setTotal((prev) => (prev > 0 ? prev : 1));
				setSelectedFile(file);
				setDrawerOpen(true);
			})
			.finally(() => {
				if (cancelled) return;
				clearFocusParam();
			});

		return () => {
			cancelled = true;
		};
	}, [focusedMcapFileId, files, searchParams, setSearchParams]);

	const columns: ColumnsType<McapFile> = [
		{
			title: "MCAP File ID",
			dataIndex: "mcap_file_id",
			width: 140,
			render: (id: string) => (
				<span className="font-mono text-xs">{id?.slice(0, 12)}…</span>
			),
		},
		{
			title: "GCS Path",
			dataIndex: "gcs_path",
			ellipsis: true,
			responsive: ["md"],
			render: (v: string) => (
				<Text type="secondary" style={{ fontSize: 12 }}>
					{v || "—"}
				</Text>
			),
		},
		{
			title: "大小",
			dataIndex: "size_bytes",
			width: 100,
			render: (v: number) => formatBytes(v),
		},
		{
			title: "状态",
			dataIndex: "ingest_state",
			width: 100,
			render: (s: string) => (
				<Tag color={ingestColor[s] ?? "default"}>{s || "—"}</Tag>
			),
		},
		{
			title: "Channels",
			dataIndex: "channel_count",
			width: 90,
			responsive: ["lg"],
			render: (v: number) => v || "—",
		},
		{
			title: "Chunks",
			dataIndex: "chunk_count",
			width: 80,
			responsive: ["lg"],
			render: (v: number) => v || "—",
		},
		{
			title: "Owner",
			dataIndex: "owner",
			width: 100,
			ellipsis: true,
			render: (v: string) => v || "—",
		},
		{
			title: "更新时间",
			dataIndex: "updated_at",
			width: 180,
			render: (v: string) => formatDateTime(v),
		},
	];

	// P1 #7: Empty state with guidance
	const emptyState = (
		<Empty
			image={<CloudUploadOutlined style={{ fontSize: 48, color: "#94A3B8" }} />}
			description={
				<div>
					<Text type="secondary">暂无 MCAP 文件记录</Text>
					<br />
					<Text type="secondary" style={{ fontSize: 12 }}>
						通过 SDK 上传 MCAP 文件或调用 POST /api/v1/mcap-files/:id/finalize
						完成入库
					</Text>
				</div>
			}
		/>
	);

	return (
		<div>
			<div className="flex items-center justify-between mb-3">
				<Title level={4} style={{ margin: 0 }}>
					<FileOutlined style={{ marginRight: 8 }} />
					MCAP 文件
					<Text
						type="secondary"
						style={{ fontSize: 14, fontWeight: 400, marginLeft: 8 }}
					>
						{loading ? "…" : `(${total})`}
					</Text>
				</Title>
				<Space>
					<Input
						id="mcap-owner-filter"
						placeholder="搜索 Owner"
						value={ownerFilter}
						onChange={(e) => setOwnerFilter(e.target.value)}
						onPressEnter={() => {
							setDebouncedOwnerFilter(ownerFilter.trim());
							setPage(1);
						}}
						onBlur={() => {
							setDebouncedOwnerFilter(ownerFilter.trim());
							setPage(1);
						}}
						style={{ width: 140 }}
						allowClear
					/>
					<Select
						id="mcap-state-filter"
						value={stateFilter}
						onChange={(v) => {
							setStateFilter(v);
							setPage(1);
						}}
						options={stateFilterOptions}
						style={{ width: 130 }}
						size="middle"
					/>
					<Button icon={<ReloadOutlined />} onClick={() => load(page)}>
						刷新
					</Button>
				</Space>
			</div>

			{error ? (
				<Result
					status="error"
					title="加载 MCAP 文件失败"
					subTitle={error}
					extra={
						<Button
							type="primary"
							icon={<ReloadOutlined />}
							onClick={() => load(page)}
						>
							重试
						</Button>
					}
				/>
			) : (
				<Table
					rowKey="mcap_file_id"
					columns={columns}
					dataSource={files}
					loading={loading}
					size="small"
					scroll={{ x: 800 }}
					locale={{ emptyText: emptyState }}
					onRow={(record) => ({
						style: { cursor: "pointer" },
						onClick: () => {
							setSelectedFile(record);
							setDrawerOpen(true);
						},
					})}
					pagination={{
						current: page,
						total,
						pageSize: 20,
						showSizeChanger: false,
						showQuickJumper: total > 200,
						showTotal: (t) => `共 ${t} 条`,
						onChange: (p) => {
							setPage(p);
							setSearchParams({ page: String(p) });
						},
					}}
				/>
			)}

			<McapDetailDrawer
				open={drawerOpen}
				mcapFile={selectedFile}
				onClose={() => setDrawerOpen(false)}
			/>
		</div>
	);
}
