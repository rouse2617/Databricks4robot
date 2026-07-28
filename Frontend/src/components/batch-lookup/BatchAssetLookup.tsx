// CYB-4304: preset-driven batch-lookup shell.
//
// Owns the paste textarea, id_type radio, filter slot, submit wiring, stats
// cards, optional histogram, results table, missing/filtered-out sections
// and CSV export. Everything preset-specific (columns, filters, buckets,
// fetch call, csv shape) comes from `preset` — see `./types.ts`.
//
// The concrete durations page (`pages/AssetDurationLookup.tsx`) is now a
// 5-line shim that mounts this component with `durationsPreset`; further
// lookups (lineage, cost) will plug their own preset onto the same shell.

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
	message,
	Radio,
	Row,
	Space,
	Statistic,
	Table,
	Typography,
} from "antd";
import { useMemo, useState } from "react";
import type {
	BatchLookupIdType,
	BatchLookupPreset,
	BatchLookupResponse,
} from "./types";
import { parseIdBlob } from "./types";

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

interface Props<Item extends { input_id: string }, Filters> {
	preset: BatchLookupPreset<Item, Filters>;
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
// stay comparable even when the biggest bucket is small. Matches the
// CYB-4294 look-and-feel exactly.
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
			aria-label="batch lookup histogram"
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

// Build the CSV blob and trigger a download. Preset-agnostic — pulls the
// filename / header / row shape from `preset.csv`. `filters` (optional) is
// forwarded to preset.csv.filename so a preset can embed the query window
// in the filename (see costs preset). Callers with no filter context can
// omit the arg.
export function downloadCsv<Item extends { input_id: string }, Filters = unknown>(
	items: Item[],
	matchedCount: number,
	csv: BatchLookupPreset<Item, Filters>["csv"],
	filters?: Filters,
): void {
	const header = `${csv.header.join(",")}\n`;
	const rows = items
		.map((it) => {
			const cells = csv.row(it);
			return cells
				.map((c) => `"${String(c ?? "").replace(/"/g, '""')}"`)
				.join(",");
		})
		.join("\n");
	const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8" });
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = csv.filename(matchedCount, filters);
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
	URL.revokeObjectURL(url);
}

export default function BatchAssetLookup<
	Item extends { input_id: string },
	Filters,
>({ preset }: Props<Item, Filters>) {
	const [rawInput, setRawInput] = useState("");
	const [idType, setIdType] = useState<BatchLookupIdType>("auto");
	const [filters, setFilters] = useState<Filters>(
		() => (preset.filters?.initial ?? ({} as Filters)),
	);
	const [submitting, setSubmitting] = useState(false);
	const [result, setResult] = useState<BatchLookupResponse<Item> | null>(null);

	const parsed = useMemo(() => parseIdBlob(rawInput), [rawInput]);

	const filterError = useMemo<string | null>(() => {
		if (!preset.filters?.validate) return null;
		return preset.filters.validate(filters);
	}, [preset.filters, filters]);

	const disabled =
		submitting ||
		parsed.ids.length === 0 ||
		parsed.ids.length > preset.maxIds ||
		filterError !== null;

	// Histogram is optional and — per CYB-4305 — can be filter-conditional:
	// the lineage preset returns `undefined` for depth=1 to suppress the
	// "分布" card entirely while still rendering it for depth="all". The
	// shell must therefore distinguish "no card" (undefined) from "empty
	// buckets" ([]).
	const buckets = useMemo(
		() =>
			result && preset.histogram
				? preset.histogram(result.items, filters)
				: undefined,
		[result, preset.histogram, filters],
	);
	const showHistogram = buckets !== undefined;

	const extraStats = useMemo(
		() => (result && preset.extraStats ? preset.extraStats(result) : []),
		[result, preset.extraStats],
	);

	// preset.columns may be either a static array or a filter-driven callback
	// (see costs preset: asset_algo mode adds an algo_key column). Resolve
	// once per render so the Table receives a stable ColumnsType.
	const columns = useMemo(
		() =>
			typeof preset.columns === "function"
				? preset.columns(filters)
				: preset.columns,
		[preset.columns, filters],
	);

	async function handleSubmit() {
		setSubmitting(true);
		try {
			const resp = await preset.fetch({
				ids: parsed.ids,
				id_type: idType,
				filters,
			});
			setResult(resp);
		} catch (err) {
			const e = err as {
				message?: string;
				response?: { data?: { message?: string } };
			};
			const msg = e?.response?.data?.message ?? e?.message ?? "查询失败";
			message.error(msg);
		} finally {
			setSubmitting(false);
		}
	}

	const countHint = `${parsed.ids.length} ids${
		parsed.duplicates > 0 ? ` (${parsed.duplicates} duplicates ignored)` : ""
	}${
		parsed.ids.length > preset.maxIds
			? ` — 超过上限 ${preset.maxIds}`
			: ""
	}`;

	const missingCount = result?.missing_ids.length ?? 0;
	const filteredOutIds = result?.filtered_out_ids ?? [];
	const showFilteredOutCard = filteredOutIds.length > 0;

	return (
		<div style={{ padding: "0 24px 24px" }}>
			<Paragraph type="secondary">
				粘贴 asset_id 或 grace_video_id(混合支持),一次最多{" "}
				{preset.maxIds.toLocaleString()} 个。服务端返回匹配项、缺失项和统计信息。
			</Paragraph>

			<Card style={{ marginTop: 16 }}>
				<Form layout="vertical">
					<Form.Item label="ID 列表(换行、空格或逗号分隔)">
						<TextArea
							rows={8}
							value={rawInput}
							placeholder={preset.placeholder}
							onChange={(e) => setRawInput(e.target.value)}
							data-testid="ids-textarea"
						/>
						<Text
							type={parsed.ids.length > preset.maxIds ? "danger" : "secondary"}
							style={{ display: "block", marginTop: 4 }}
						>
							{countHint}
						</Text>
					</Form.Item>
					<Row gutter={16}>
						<Col span={preset.filters ? 8 : 24}>
							<Form.Item label="ID 类型(仅前端提示)">
								<Radio.Group
									value={idType}
									onChange={(e) => setIdType(e.target.value)}
								>
									<Radio.Button value="auto">auto</Radio.Button>
									<Radio.Button value="asset_id">asset_id</Radio.Button>
									<Radio.Button value="grace_video_id">
										grace_video_id
									</Radio.Button>
								</Radio.Group>
							</Form.Item>
						</Col>
						{preset.filters ? preset.filters.render(filters, setFilters) : null}
					</Row>
					{filterError ? (
						<Text
							type="danger"
							style={{ display: "block", marginBottom: 8 }}
							data-testid="filter-error"
						>
							{filterError}
						</Text>
					) : null}
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
									value={result.items.length}
									data-testid="stat-matched"
								/>
							</Col>
							<Col span={4}>
								<Statistic
									title="missing"
									value={missingCount}
									data-testid="stat-missing"
								/>
							</Col>
							{preset.filters || showFilteredOutCard ? (
								<Col span={4}>
									<Statistic
										title="filtered out"
										value={filteredOutIds.length}
										data-testid="stat-filtered-out"
									/>
								</Col>
							) : null}
							{extraStats.map((s) => (
								<Col span={4} key={s.label}>
									<Statistic title={s.label} value={s.value} />
								</Col>
							))}
						</Row>
					</Card>

					{showHistogram ? (
						<Card style={{ marginTop: 16 }} title="分布">
							{result.items.length === 0 ? (
								<Empty description="无匹配项" />
							) : (
								<HistogramBars buckets={buckets ?? []} />
							)}
						</Card>
					) : null}

					<Card
						style={{ marginTop: 16 }}
						title={`结果 (${result.items.length})`}
						extra={
							<Button
								icon={<DownloadOutlined />}
								onClick={() =>
									downloadCsv(
										result.items,
										result.items.length,
										preset.csv,
										filters,
									)
								}
								disabled={result.items.length === 0}
								data-testid="export-csv-button"
							>
								导出 CSV
							</Button>
						}
					>
						<Table<Item>
							dataSource={result.items}
							columns={columns}
							rowKey={(r) => r.input_id}
							size="small"
							pagination={{ pageSize: 20, showSizeChanger: true }}
							data-testid="results-table"
						/>
					</Card>

					{(missingCount > 0 || showFilteredOutCard) && (
						<Card style={{ marginTop: 16 }}>
							<Collapse
								defaultActiveKey={
									missingCount > 0
										? ["missing"]
										: showFilteredOutCard
											? ["filtered"]
											: []
								}
								items={[
									missingCount > 0
										? {
												key: "missing",
												label: `未匹配的 ID (${missingCount})`,
												children: (
													<Space
														direction="vertical"
														style={{ width: "100%" }}
													>
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
															data-testid="copy-missing-button"
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
									showFilteredOutCard
										? {
												key: "filtered",
												label: `范围外的 ID (${filteredOutIds.length})`,
												children: (
													<Space
														direction="vertical"
														style={{ width: "100%" }}
													>
														<Alert
															type="info"
															showIcon
															message="这些 ID 存在,但不在你指定的区间内。"
														/>
														<Button
															icon={<CopyOutlined />}
															size="small"
															onClick={async () => {
																if (
																	await copyToClipboard(
																		filteredOutIds.join("\n"),
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
															{filteredOutIds.join("\n")}
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
