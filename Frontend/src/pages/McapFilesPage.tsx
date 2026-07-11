import {
	CloudUploadOutlined,
	FileOutlined,
	ReloadOutlined,
} from "@ant-design/icons";
import {
	Button,
	Empty,
	Grid,
	Input,
	Result,
	Select,
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
import {
	COLUMN_LABELS,
	formatBusinessStatusLabel,
	resolveBusinessStatusTagColor,
} from "../lib/productVocabulary";

const { Title, Text } = Typography;
const { useBreakpoint } = Grid;

function formatBytes(bytes: number): string {
	if (!bytes || bytes === 0) return "—";
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024)
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

// CYB-3305: summarized rows without a recorded size_bytes should read "未记录"
// rather than the ambiguous "—" that pending rows use.
function formatSizeCell(bytes: number, record: McapFile): string {
	if (!bytes || bytes === 0) {
		if (record.ingest_state === "summarized") return "未记录";
		return "—";
	}
	return formatBytes(bytes);
}

const stateFilterOptions = [
	{ value: "", label: "全部状态" },
	{ value: "pending", label: "待处理" },
	{ value: "summarized", label: "已汇总" },
	{ value: "failed", label: "失败" },
];

export default function McapFilesPage() {
	const screens = useBreakpoint();
	const isNarrow =
		typeof window !== "undefined" &&
		window.innerWidth < 768 &&
		screens.md !== true;
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

	// biome-ignore lint/correctness/useExhaustiveDependencies: filter change triggers page reset
	useEffect(() => {
		setPage(1);
	}, [debouncedOwnerFilter, stateFilter]);

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
	}, []);

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
			title: COLUMN_LABELS.mcapFileId,
			dataIndex: "mcap_file_id",
			width: 140,
			render: (id: string) => (
				<span className="font-mono text-xs">{id?.slice(0, 12)}…</span>
			),
		},
		{
			title: COLUMN_LABELS.gcsPath,
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
			render: (v: number, record: McapFile) => formatSizeCell(v, record),
		},
		{
			title: "入库状态",
			dataIndex: "ingest_state",
			width: 100,
			render: (s: string) => (
				<Tag color={resolveBusinessStatusTagColor(s)}>
					{formatBusinessStatusLabel(s)}
				</Tag>
			),
		},
		{
			// CYB-3285: child-segment count replaces the (always-empty for grace)
			// channel/chunk columns.
			title: "子 segment 数",
			dataIndex: "segment_count",
			width: 110,
			render: (v: number | undefined) => v ?? 0,
		},
		{
			title: COLUMN_LABELS.owner,
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

	const emptyState = (
		<Empty
			image={<CloudUploadOutlined style={{ fontSize: 48, color: "#94A3B8" }} />}
			description={
				<div>
					<Text type="secondary">暂无 MCAP 文件</Text>
					<br />
					<Text type="secondary" style={{ fontSize: 12 }}>
						可通过 SDK 上传文件，上传完成后系统会自动入库。
					</Text>
					<br />
					<Text type="secondary" style={{ fontSize: 11 }}>
						开发接入：POST /api/v1/mcap-files/:id/finalize
					</Text>
				</div>
			}
		>
			<Button icon={<ReloadOutlined />} onClick={() => load(page)}>
				刷新
			</Button>
		</Empty>
	);

	return (
		<div>
			<div
				style={{
					display: "flex",
					alignItems: isNarrow ? "stretch" : "center",
					justifyContent: "space-between",
					gap: 12,
					flexWrap: "wrap",
					marginBottom: 12,
				}}
			>
				<Title
					level={4}
					style={{
						margin: 0,
						flex: isNarrow ? "1 1 100%" : "0 0 auto",
						minWidth: 0,
						whiteSpace: "nowrap",
					}}
				>
					<FileOutlined style={{ marginRight: 8 }} />
					MCAP 文件
					<Text
						type="secondary"
						style={{ fontSize: 14, fontWeight: 400, marginLeft: 8 }}
					>
						{loading ? "…" : `(${total})`}
					</Text>
				</Title>
				<div
					style={{
						display: "grid",
						gridTemplateColumns: isNarrow
							? "minmax(0, 1fr) minmax(0, 1fr)"
							: "140px 130px auto",
						gap: 8,
						width: isNarrow ? "100%" : "auto",
						minWidth: 0,
					}}
				>
					<Input
						id="mcap-owner-filter"
						placeholder="搜索所属方"
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
						style={{
							width: "100%",
							gridColumn: isNarrow ? "1 / -1" : undefined,
						}}
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
						style={{ width: "100%" }}
						size="middle"
					/>
					<Button
						icon={<ReloadOutlined />}
						onClick={() => load(page)}
						style={{ width: "100%" }}
					>
						刷新
					</Button>
				</div>
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
