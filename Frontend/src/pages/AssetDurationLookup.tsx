// CYB-4294: batch asset-duration lookup page.
//
// Paste a mixed list of asset_id / grace_video_id inputs, optionally filter
// by duration, submit → server returns matched items + missing + filtered-out
// + stats + 5-bucket histogram data (bucket boundaries hard-coded in
// openspec/changes/CYB-4294-asset-durations/specs/assets/spec.md).

import { CopyOutlined, DownloadOutlined, SearchOutlined } from "@ant-design/icons";
import {
	Alert,
	Button,
	Card,
	Col,
	Collapse,
	Empty,
	Form,
	Input,
	InputNumber,
	message,
	Radio,
	Row,
	Space,
	Statistic,
	Table,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo, useState } from "react";
import { assetsApi } from "../api/assets";
import type {
	AssetDurationIdType,
	AssetDurationsResponse,
	DurationLookupItem,
} from "../api/assets";

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const MAX_IDS = 5000;

// Histogram bucket boundaries (ms). Boundaries are `[lo, hi)` — the last
// bucket is open-ended. Matches spec.md.
export const HISTOGRAM_BUCKETS: Array<{
	label: string;
	loMs: number;
	hiMs: number; // Number.POSITIVE_INFINITY for the top bucket
}> = [
	{ label: "<1min", loMs: 0, hiMs: 60_000 },
	{ label: "1-10min", loMs: 60_000, hiMs: 600_000 },
	{ label: "10-30min", loMs: 600_000, hiMs: 1_800_000 },
	{ label: "30-60min", loMs: 1_800_000, hiMs: 3_600_000 },
	{ label: "60min+", loMs: 3_600_000, hiMs: Number.POSITIVE_INFINITY },
];

// Parse a paste blob into a list of ids. Splits on any whitespace or comma,
// trims, drops empties. Preserves order. Returns { ids, duplicates } where
// duplicates counts the number of first-hit duplicates the user pasted.
export function parseIdBlob(raw: string): { ids: string[]; duplicates: number } {
	if (!raw) return { ids: [], duplicates: 0 };
	const seen = new Set<string>();
	const out: string[] = [];
	let dupCount = 0;
	for (const token of raw.split(/[\s,]+/)) {
		const t = token.trim();
		if (!t) continue;
		if (seen.has(t)) {
			dupCount += 1;
			continue;
		}
		seen.add(t);
		out.push(t);
	}
	return { ids: out, duplicates: dupCount };
}

export function bucketItems(
	items: DurationLookupItem[],
): Array<{ label: string; count: number }> {
	const counts = HISTOGRAM_BUCKETS.map((b) => ({ label: b.label, count: 0 }));
	for (const it of items) {
		for (let i = 0; i < HISTOGRAM_BUCKETS.length; i += 1) {
			const b = HISTOGRAM_BUCKETS[i];
			if (it.duration_ms >= b.loMs && it.duration_ms < b.hiMs) {
				counts[i].count += 1;
				break;
			}
		}
	}
	return counts;
}

export function exportDurationsCsv(
	items: DurationLookupItem[],
	matchedCount: number,
): void {
	const header = "input_id,asset_id,grace_video_id,duration_ms,formatted\n";
	const rows = items
		.map((it) => {
			const cells = [
				it.input_id,
				it.asset_id,
				it.grace_video_id ?? "",
				String(it.duration_ms),
				it.formatted,
			];
			return cells.map((c) => `"${String(c).replace(/"/g, '""')}"`).join(",");
		})
		.join("\n");
	const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8" });
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = `asset-durations-${matchedCount}.csv`;
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
	URL.revokeObjectURL(url);
}

async function copyToClipboard(text: string): Promise<boolean> {
	if (navigator.clipboard?.writeText) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {
			// fall through
		}
	}
	try {
		const ta = document.createElement("textarea");
		ta.value = text;
		ta.style.position = "fixed";
		ta.style.opacity = "0";
		document.body.appendChild(ta);
		ta.select();
		document.execCommand("copy");
		document.body.removeChild(ta);
		return true;
	} catch {
		return false;
	}
}

// Histogram bar chart — no chart dependency, just flexbox. Every bucket
// gets a fixed-width column with a fill proportion (max=1) so the bars
// stay comparable even when the biggest bucket is small.
function HistogramBars({
	buckets,
}: {
	buckets: Array<{ label: string; count: number }>;
}) {
	const max = Math.max(1, ...buckets.map((b) => b.count));
	return (
		<div
			style={{
				display: "flex",
				alignItems: "flex-end",
				gap: 12,
				height: 160,
			}}
			aria-label="duration histogram"
		>
			{buckets.map((b) => {
				const heightPct = Math.round((b.count / max) * 100);
				return (
					<div
						key={b.label}
						style={{
							flex: 1,
							display: "flex",
							flexDirection: "column",
							alignItems: "center",
							height: "100%",
						}}
					>
						<div style={{ fontSize: 12, marginBottom: 4 }}>{b.count}</div>
						<div
							style={{
								width: "100%",
								height: `${heightPct}%`,
								background: "#1677ff",
								borderRadius: "4px 4px 0 0",
								minHeight: b.count > 0 ? 4 : 0,
							}}
						/>
						<div
							style={{
								marginTop: 6,
								fontSize: 12,
								color: "#64748b",
								textAlign: "center",
							}}
						>
							{b.label}
						</div>
					</div>
				);
			})}
		</div>
	);
}

export default function AssetDurationLookup() {
	const [rawInput, setRawInput] = useState("");
	const [idType, setIdType] = useState<AssetDurationIdType>("auto");
	const [minMs, setMinMs] = useState<number | null>(null);
	const [maxMs, setMaxMs] = useState<number | null>(null);
	const [submitting, setSubmitting] = useState(false);
	const [result, setResult] = useState<AssetDurationsResponse | null>(null);

	const parsed = useMemo(() => parseIdBlob(rawInput), [rawInput]);

	const disabled =
		submitting ||
		parsed.ids.length === 0 ||
		parsed.ids.length > MAX_IDS ||
		(minMs !== null && maxMs !== null && minMs > maxMs);

	const buckets = useMemo(
		() => (result ? bucketItems(result.items) : []),
		[result],
	);

	const columns: ColumnsType<DurationLookupItem> = [
		{ title: "input_id", dataIndex: "input_id", key: "input_id" },
		{ title: "asset_id", dataIndex: "asset_id", key: "asset_id" },
		{
			title: "grace_video_id",
			dataIndex: "grace_video_id",
			key: "grace_video_id",
			render: (v?: string) => v ?? "-",
		},
		{
			title: "duration_ms",
			dataIndex: "duration_ms",
			key: "duration_ms",
			sorter: (a, b) => a.duration_ms - b.duration_ms,
			defaultSortOrder: "descend",
			align: "right",
		},
		{ title: "formatted", dataIndex: "formatted", key: "formatted" },
	];

	async function handleSubmit() {
		setSubmitting(true);
		try {
			const req = {
				ids: parsed.ids,
				id_type: idType,
				...(minMs !== null && minMs > 0 ? { min_duration_ms: minMs } : {}),
				...(maxMs !== null && maxMs > 0 ? { max_duration_ms: maxMs } : {}),
			};
			const resp = await assetsApi.lookupDurations(req);
			setResult(resp);
		} catch (err) {
			const e = err as { message?: string; response?: { data?: { message?: string } } };
			const msg = e?.response?.data?.message ?? e?.message ?? "查询失败";
			message.error(msg);
		} finally {
			setSubmitting(false);
		}
	}

	const countHint = `${parsed.ids.length} ids${
		parsed.duplicates > 0 ? ` (${parsed.duplicates} duplicates ignored)` : ""
	}${parsed.ids.length > MAX_IDS ? ` — 超过上限 ${MAX_IDS}` : ""}`;

	return (
		<div>
			<Title level={3}>资产时长批量查询</Title>
			<Paragraph type="secondary">
				粘贴 asset_id 或 grace_video_id（混合支持），一次最多 {MAX_IDS.toLocaleString()}{" "}
				个。可选按时长区间过滤,服务端返回匹配项、缺失项、时长统计和分布直方图。
			</Paragraph>

			<Card style={{ marginTop: 16 }}>
				<Form layout="vertical">
					<Form.Item label="ID 列表(换行、空格或逗号分隔)">
						<TextArea
							rows={8}
							value={rawInput}
							placeholder={"aaaaaaaa\nbbbbbbbb\n019f8319-0ef3-7da0-815c-30f23d21e7f1"}
							onChange={(e) => setRawInput(e.target.value)}
							data-testid="ids-textarea"
						/>
						<Text
							type={parsed.ids.length > MAX_IDS ? "danger" : "secondary"}
							style={{ display: "block", marginTop: 4 }}
						>
							{countHint}
						</Text>
					</Form.Item>
					<Row gutter={16}>
						<Col span={8}>
							<Form.Item label="ID 类型(仅前端提示)">
								<Radio.Group
									value={idType}
									onChange={(e) => setIdType(e.target.value)}
								>
									<Radio.Button value="auto">auto</Radio.Button>
									<Radio.Button value="asset_id">asset_id</Radio.Button>
									<Radio.Button value="grace_video_id">grace_video_id</Radio.Button>
								</Radio.Group>
							</Form.Item>
						</Col>
						<Col span={8}>
							<Form.Item label="最小时长">
								<InputNumber
									value={minMs}
									onChange={(v) => setMinMs(v)}
									min={0}
									addonAfter="ms"
									style={{ width: "100%" }}
									placeholder="0 = 不限"
								/>
							</Form.Item>
						</Col>
						<Col span={8}>
							<Form.Item label="最大时长">
								<InputNumber
									value={maxMs}
									onChange={(v) => setMaxMs(v)}
									min={0}
									addonAfter="ms"
									style={{ width: "100%" }}
									placeholder="0 = 不限"
								/>
							</Form.Item>
						</Col>
					</Row>
					<Button
						type="primary"
						icon={<SearchOutlined />}
						loading={submitting}
						disabled={disabled}
						onClick={handleSubmit}
						data-testid="submit-button"
					>
						查询
					</Button>
				</Form>
			</Card>

			{result ? (
				<>
					<Card style={{ marginTop: 16 }} title="统计">
						<Row gutter={16}>
							<Col span={4}>
								<Statistic
									title="matched"
									value={result.stats.matched_count}
									data-testid="stat-matched"
								/>
							</Col>
							<Col span={4}>
								<Statistic
									title="missing"
									value={result.stats.missing_count}
								/>
							</Col>
							<Col span={4}>
								<Statistic
									title="filtered out"
									value={result.stats.filtered_out_count}
								/>
							</Col>
							<Col span={4}>
								<Statistic title="total (ms)" value={result.stats.total_ms} />
							</Col>
							<Col span={4}>
								<Statistic title="mean (ms)" value={result.stats.mean_ms} />
							</Col>
							<Col span={4}>
								<Statistic title="max (ms)" value={result.stats.max_ms} />
							</Col>
						</Row>
						<Row gutter={16} style={{ marginTop: 16 }}>
							<Col span={4}>
								<Statistic title="min (ms)" value={result.stats.min_ms} />
							</Col>
							<Col span={4}>
								<Statistic title="p50 (ms)" value={result.stats.p50_ms} />
							</Col>
							<Col span={4}>
								<Statistic title="p90 (ms)" value={result.stats.p90_ms} />
							</Col>
						</Row>
					</Card>

					<Card style={{ marginTop: 16 }} title="时长分布">
						{result.items.length === 0 ? (
							<Empty description="无匹配项" />
						) : (
							<HistogramBars buckets={buckets} />
						)}
					</Card>

					<Card
						style={{ marginTop: 16 }}
						title={`结果 (${result.items.length})`}
						extra={
							<Button
								icon={<DownloadOutlined />}
								onClick={() =>
									exportDurationsCsv(result.items, result.stats.matched_count)
								}
								disabled={result.items.length === 0}
								data-testid="export-csv-button"
							>
								导出 CSV
							</Button>
						}
					>
						<Table<DurationLookupItem>
							dataSource={result.items}
							columns={columns}
							rowKey={(r) => r.input_id}
							size="small"
							pagination={{ pageSize: 50, showSizeChanger: true }}
							data-testid="results-table"
						/>
					</Card>

					{(result.missing_ids.length > 0 ||
						result.filtered_out_ids.length > 0) && (
						<Card style={{ marginTop: 16 }}>
							<Collapse
								items={[
									result.missing_ids.length > 0
										? {
												key: "missing",
												label: `未匹配的 ID (${result.missing_ids.length})`,
												children: (
													<Space direction="vertical" style={{ width: "100%" }}>
														<Alert
															type="warning"
															showIcon
															message="这些 ID 在平台上找不到对应资产,可能已删除或从未存在。"
														/>
														<Button
															icon={<CopyOutlined />}
															size="small"
															onClick={async () => {
																if (
																	await copyToClipboard(
																		result.missing_ids.join("\n"),
																	)
																) {
																	message.success("已复制到剪贴板");
																}
															}}
														>
															复制全部
														</Button>
														<pre
															style={{
																maxHeight: 240,
																overflow: "auto",
																background: "#f8fafc",
																padding: 8,
																borderRadius: 4,
																fontSize: 12,
															}}
														>
															{result.missing_ids.join("\n")}
														</pre>
													</Space>
												),
											}
										: null,
									result.filtered_out_ids.length > 0
										? {
												key: "filtered",
												label: `范围外的 ID (${result.filtered_out_ids.length})`,
												children: (
													<Space direction="vertical" style={{ width: "100%" }}>
														<Alert
															type="info"
															showIcon
															message="这些 ID 存在,但时长不在你指定的区间内。"
														/>
														<Button
															icon={<CopyOutlined />}
															size="small"
															onClick={async () => {
																if (
																	await copyToClipboard(
																		result.filtered_out_ids.join("\n"),
																	)
																) {
																	message.success("已复制到剪贴板");
																}
															}}
														>
															复制全部
														</Button>
														<pre
															style={{
																maxHeight: 240,
																overflow: "auto",
																background: "#f8fafc",
																padding: 8,
																borderRadius: 4,
																fontSize: 12,
															}}
														>
															{result.filtered_out_ids.join("\n")}
														</pre>
													</Space>
												),
											}
										: null,
								].filter(
									(x): x is NonNullable<typeof x> => x !== null,
								)}
							/>
						</Card>
					)}
				</>
			) : null}
		</div>
	);
}
