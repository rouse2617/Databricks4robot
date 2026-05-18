// ─── ExportModal — Export assets to CSV or JSON ───
// Range selection (current page / all filtered), format selection (CSV / JSON),
// disable "all" when > 10,000 and show warning.
// Validates: Requirements REQ-3.2

import {
	Alert,
	Modal,
	message,
	Progress,
	Radio,
	Space,
	Typography,
} from "antd";
import { useCallback, useState } from "react";
import { assetsApi, type ListAssetsParams } from "../../api/assets";
import type { Asset } from "../../api/types";
import { extractApiErrorMessage } from "../../lib/apiError";

const { Text } = Typography;

// ─── Types ───

export type ExportRange = "current_page" | "all_filtered";
export type ExportFormat = "csv" | "json";

export interface ExportModalProps {
	open: boolean;
	/** Currently loaded page items */
	currentPageItems: Asset[];
	/** Total filtered count from the API */
	totalFiltered: number;
	/** Current query params to re-fetch all filtered results */
	queryParams: ListAssetsParams;
	onClose: () => void;
}

// ─── Constants ───

const MAX_EXPORT_ROWS = 10_000;
const FETCH_PAGE_SIZE = 200;
const FETCH_CONCURRENCY = 3;

const CSV_COLUMNS = [
	"asset_id",
	"mcap_file_id",
	"segment_locator",
	"lifecycle_state",
	"env",
	"task",
	"duration_ms",
	"owner",
	"reviewer",
	"tags",
	"algo_summary",
	"created_at",
	"updated_at",
] as const;

// ─── Helpers (7.5 / 7.6 / 7.7) ───

/** Derive algo_summary string like "2 ok / 1 failed" from algo_results */
function buildAlgoSummary(
	algoResults: Record<string, string> | undefined,
): string {
	if (!algoResults) return "";
	const counts: Record<string, number> = {};
	for (const [k, v] of Object.entries(algoResults)) {
		if (k.endsWith(":status")) {
			counts[v] = (counts[v] ?? 0) + 1;
		}
	}
	const order = ["ok", "failed", "running", "pending", "blocked"];
	const parts: string[] = [];
	for (const s of order) {
		if (counts[s]) parts.push(`${counts[s]} ${s}`);
	}
	for (const [s, c] of Object.entries(counts)) {
		if (!order.includes(s)) parts.push(`${c} ${s}`);
	}
	return parts.join(" / ");
}

/** Escape a CSV field value (wrap in quotes if it contains comma, quote, or newline) */
function escapeCsvField(value: string): string {
	if (value.includes(",") || value.includes('"') || value.includes("\n")) {
		return `"${value.replace(/"/g, '""')}"`;
	}
	return value;
}

/** Generate CSV string from assets (7.5) */
function generateCsv(assets: Asset[]): string {
	const header = CSV_COLUMNS.join(",");
	const rows = assets.map((a) => {
		const values: string[] = [
			a.asset_id ?? "",
			a.mcap_file_id ?? "",
			a.segment_locator ?? "",
			a.lifecycle_state ?? a.status ?? "",
			a.env ?? "",
			a.task ?? "",
			a.duration_ms != null ? String(a.duration_ms) : "",
			a.owner ?? "",
			a.reviewer ?? "",
			JSON.stringify(a.tags ?? {}),
			buildAlgoSummary(a.algo_results),
			a.created_at ?? "",
			a.updated_at ?? "",
		];
		return values.map(escapeCsvField).join(",");
	});
	return [header, ...rows].join("\n");
}

/** Generate JSON string from assets (7.6) */
function generateJson(assets: Asset[]): string {
	return JSON.stringify(assets, null, 2);
}

/** Trigger browser download via Blob + URL.createObjectURL + a.click (7.7) */
function triggerDownload(
	content: string,
	filename: string,
	mimeType: string,
): void {
	const blob = new Blob([content], { type: mimeType });
	const url = URL.createObjectURL(blob);
	const a = document.createElement("a");
	a.href = url;
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}

/** Fetch all filtered results with pagination and concurrency limit (7.4) */
async function fetchAllFiltered(
	queryParams: ListAssetsParams,
	totalFiltered: number,
	onProgress: (fetched: number) => void,
): Promise<Asset[]> {
	const totalPages = Math.ceil(
		Math.min(totalFiltered, MAX_EXPORT_ROWS) / FETCH_PAGE_SIZE,
	);
	const allAssets: Asset[] = [];
	let fetched = 0;

	// Process pages in batches of FETCH_CONCURRENCY
	for (
		let batchStart = 0;
		batchStart < totalPages;
		batchStart += FETCH_CONCURRENCY
	) {
		const batchEnd = Math.min(batchStart + FETCH_CONCURRENCY, totalPages);
		const promises: Promise<Asset[]>[] = [];

		for (let pageIdx = batchStart; pageIdx < batchEnd; pageIdx++) {
			const params: ListAssetsParams = {
				...queryParams,
				page: pageIdx + 1,
				page_size: FETCH_PAGE_SIZE,
			};
			promises.push(assetsApi.list(params).then((res) => res.items));
		}

		const results = await Promise.all(promises);
		for (const items of results) {
			allAssets.push(...items);
			fetched += items.length;
			onProgress(fetched);
		}
	}

	return allAssets;
}

// ─── Component ───

export default function ExportModal({
	open,
	currentPageItems,
	totalFiltered,
	queryParams,
	onClose,
}: ExportModalProps) {
	const [range, setRange] = useState<ExportRange>("current_page");
	const [format, setFormat] = useState<ExportFormat>("csv");
	const [exporting, setExporting] = useState(false);
	const [progress, setProgress] = useState(0);

	const allDisabled = totalFiltered > MAX_EXPORT_ROWS;

	const handleExport = useCallback(async () => {
		setExporting(true);
		setProgress(0);

		try {
			let assets: Asset[];

			if (range === "current_page") {
				// 7.3: Export current page directly from loaded data
				assets = currentPageItems;
			} else {
				// 7.4: Export all filtered with paginated fetch.
				// If any batch fails, surface the error to the user instead of
				// silently aborting (previous behaviour swallowed errors in the
				// outer try/finally with no message).
				assets = await fetchAllFiltered(
					queryParams,
					totalFiltered,
					(fetched) => {
						setProgress(
							Math.round(
								(fetched / Math.min(totalFiltered, MAX_EXPORT_ROWS)) * 100,
							),
						);
					},
				);
			}

			const timestamp = new Date()
				.toISOString()
				.replace(/[:.]/g, "-")
				.slice(0, 19);

			if (format === "csv") {
				const content = generateCsv(assets);
				triggerDownload(
					content,
					`assets-${timestamp}.csv`,
					"text/csv;charset=utf-8",
				);
			} else {
				const content = generateJson(assets);
				triggerDownload(
					content,
					`assets-${timestamp}.json`,
					"application/json;charset=utf-8",
				);
			}

			message.success(`已导出 ${assets.length} 条记录`);
			onClose();
		} catch (err) {
			message.error(`导出失败：${extractApiErrorMessage(err, "请稍后重试")}`);
		} finally {
			setExporting(false);
			setProgress(0);
		}
	}, [range, format, currentPageItems, queryParams, totalFiltered, onClose]);

	return (
		<Modal
			title="导出资产数据"
			open={open}
			onOk={handleExport}
			onCancel={onClose}
			okText={exporting ? "导出中..." : "导出"}
			okButtonProps={{ loading: exporting }}
			cancelButtonProps={{ disabled: exporting }}
			closable={!exporting}
			maskClosable={!exporting}
			destroyOnHidden
		>
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				{/* Range selection */}
				<div>
					<Text strong style={{ display: "block", marginBottom: 8 }}>
						导出范围
					</Text>
					<Radio.Group
						value={range}
						onChange={(e) => setRange(e.target.value)}
						disabled={exporting}
					>
						<Radio value="current_page">
							当前页（{currentPageItems.length} 条）
						</Radio>
						<Radio value="all_filtered" disabled={allDisabled}>
							全部筛选结果（{totalFiltered} 条）
						</Radio>
					</Radio.Group>
					{allDisabled && (
						<Alert
							type="warning"
							message={`筛选结果超过 ${MAX_EXPORT_ROWS.toLocaleString()} 条，请缩小筛选范围后再导出全部`}
							showIcon
							style={{ marginTop: 8 }}
						/>
					)}
				</div>

				{/* Format selection */}
				<div>
					<Text strong style={{ display: "block", marginBottom: 8 }}>
						导出格式
					</Text>
					<Radio.Group
						value={format}
						onChange={(e) => setFormat(e.target.value)}
						disabled={exporting}
					>
						<Radio value="csv">CSV</Radio>
						<Radio value="json">JSON</Radio>
					</Radio.Group>
				</div>

				{/* Progress bar (shown during export) */}
				{exporting && range === "all_filtered" && (
					<Progress percent={progress} status="active" />
				)}
			</Space>
		</Modal>
	);
}
