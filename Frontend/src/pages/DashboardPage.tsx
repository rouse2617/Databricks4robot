import {
	ArrowDownOutlined,
	ArrowUpOutlined,
	BarChartOutlined,
	BookOutlined,
	ClockCircleOutlined,
	DatabaseOutlined,
	LineChartOutlined,
	PieChartOutlined,
	ReloadOutlined,
	RiseOutlined,
	SendOutlined,
	TableOutlined,
	ThunderboltOutlined,
} from "@ant-design/icons";
import {
	Alert,
	Card,
	Col,
	Divider,
	Empty,
	Radio,
	Row,
	Space,
	Spin,
	Table,
	Tag,
	Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import type { ReactNode } from "react";
import { useEffect, useMemo, useRef, useState } from "react";
import type { FailureClusterItem } from "../api/lakehouse";
import {
	type LakehouseAssetGrowthResponse,
	type LakehouseEventDailyResponse,
	type LakehouseEventTypeShareResponse,
	type LakehouseOverviewResponse,
	lakehouseApi,
} from "../api/lakehouse";
import { LazyECharts } from "../components/analytics/LazyECharts";
import PageLoading from "../components/common/PageLoading";
import { extractApiErrorMessage } from "../lib/apiError";
import { getAppVersionLabel } from "../lib/appVersion";

const { Title, Text } = Typography;

const dashSurface = {
	pageBg: "var(--color-bg-layout, #f8fafc)",
	cardBorder: "1px solid rgba(148, 163, 184, 0.45)",
	cardRadius: 12,
	cardShadow:
		"0 1px 2px rgba(15, 23, 42, 0.05), 0 2px 8px rgba(15, 23, 42, 0.04)",
	heroBorder: "1px solid rgba(148, 163, 184, 0.35)",
} as const;

interface KpiTrend {
	direction: "up" | "down" | "flat";
	text: string;
}

function KpiCard({
	title,
	value,
	suffix,
	icon,
	hint,
	trend,
	iconTint = "blue",
}: {
	title: string;
	value: string | number;
	suffix?: string;
	icon: ReactNode;
	hint?: ReactNode;
	trend?: KpiTrend;
	iconTint?: "blue" | "slate";
}) {
	const iconBg =
		iconTint === "slate"
			? "rgba(100, 116, 139, 0.12)"
			: "rgba(37, 99, 235, 0.08)";
	const iconFg =
		iconTint === "slate"
			? "var(--color-text-secondary, #64748b)"
			: "var(--color-primary, #2563eb)";
	const trendColor =
		trend?.direction === "up"
			? "var(--color-success, #10b981)"
			: trend?.direction === "down"
				? "var(--color-error, #ef4444)"
				: "var(--color-text-secondary)";
	return (
		<Card
			variant="borderless"
			style={{
				flex: 1,
				height: "100%",
				minHeight: 132,
				borderRadius: dashSurface.cardRadius,
				border: dashSurface.cardBorder,
				boxShadow: dashSurface.cardShadow,
				background: "var(--color-bg-container, #fff)",
			}}
			styles={{
				body: {
					padding: "20px 22px",
					height: "100%",
					display: "flex",
					flexDirection: "column",
					justifyContent: "space-between",
				},
			}}
		>
			<div
				style={{
					display: "flex",
					justifyContent: "space-between",
					alignItems: "flex-start",
				}}
			>
				<div>
					<Text
						type="secondary"
						style={{ fontSize: 12, fontWeight: 500, letterSpacing: "0.05em" }}
					>
						{title}
					</Text>
					<div
						style={{
							marginTop: 8,
							display: "flex",
							alignItems: "baseline",
							gap: 4,
						}}
					>
						<span
							style={{
								fontSize: 28,
								fontWeight: 700,
								color: "var(--color-text)",
								fontVariantNumeric: "tabular-nums",
							}}
						>
							{value}
						</span>
						{suffix && (
							<span
								style={{
									fontSize: 14,
									color: "var(--color-text-secondary)",
									fontWeight: 500,
								}}
							>
								{suffix}
							</span>
						)}
					</div>
					{(hint || trend) && (
						<div style={{ marginTop: 6, fontSize: 12, color: trendColor }}>
							{trend?.direction === "up" && <ArrowUpOutlined />}
							{trend?.direction === "down" && <ArrowDownOutlined />}
							{trend && <span style={{ marginLeft: 4 }}>{trend.text}</span>}
							{hint && !trend && (
								<Text type="secondary" style={{ fontSize: 12 }}>
									{hint}
								</Text>
							)}
							{hint && trend && (
								<Text type="secondary" style={{ fontSize: 12, marginLeft: 8 }}>
									· {hint}
								</Text>
							)}
						</div>
					)}
				</div>
				<div
					style={{
						width: 44,
						height: 44,
						borderRadius: 12,
						background: iconBg,
						display: "flex",
						alignItems: "center",
						justifyContent: "center",
					}}
				>
					<span style={{ color: iconFg, fontSize: 19 }}>{icon}</span>
				</div>
			</div>
		</Card>
	);
}

function TableEmptyState({
	summary,
	note,
}: {
	summary: string;
	note?: string | null;
}) {
	return (
		<div style={{ padding: "12px 4px 8px" }}>
			<Empty
				image={Empty.PRESENTED_IMAGE_SIMPLE}
				styles={{ image: { height: 44 } }}
				description={
					<Space direction="vertical" size={6} style={{ maxWidth: 420 }}>
						<Text type="secondary" style={{ fontSize: 13, lineHeight: 1.5 }}>
							{summary}
						</Text>
						{note ? (
							<Text
								type="secondary"
								style={{
									fontSize: 12,
									lineHeight: 1.55,
									display: "block",
									textAlign: "center",
								}}
							>
								{note}
							</Text>
						) : null}
					</Space>
				}
			/>
		</div>
	);
}

interface DashboardMetrics {
	assetTotal: number | null;
	bronzeEventRows: number | null;
	todayNew: number | null;
	yesterdayNew: number | null;
	weekNew: number | null;
	day7Avg: number | null;
	goldLatestDate: string | null;
	dataLagHours: number | null;
}

interface AssetGrowthPoint {
	date: string;
	newAssets: number;
	cumulative: number;
}

interface DailyEventPoint {
	date: string;
	eventType: string;
	count: number;
}

interface EventTypeShareRow {
	key: string;
	eventType: string;
	count: number;
	ratio: number;
	last30Count: number;
}

interface EndpointErrors {
	overview?: string;
	growth?: string;
	daily?: string;
	share?: string;
	tables?: string;
	quality?: string;
	training?: string;
	recompute?: string;
	customer?: string;
	failureClusters?: string;
}

interface LakeTableCountRow {
	key: string;
	table_name: string;
	row_count: number;
}

interface QualityDistRow {
	key: string;
	quality: string;
	asset_count: number;
}

function toFiniteNumber(value: unknown): number | null {
	if (typeof value === "number" && Number.isFinite(value)) return value;
	if (typeof value === "string" && value.trim() !== "") {
		const parsed = Number(value);
		if (Number.isFinite(parsed)) return parsed;
	}
	return null;
}

function asRecord(value: unknown): Record<string, unknown> | null {
	if (typeof value === "object" && value !== null && !Array.isArray(value)) {
		return value as Record<string, unknown>;
	}
	return null;
}

function asRecordArray(value: unknown): Record<string, unknown>[] {
	if (!Array.isArray(value)) return [];
	return value
		.map((item) => asRecord(item))
		.filter((item): item is Record<string, unknown> => item !== null);
}

function formatLakehouseScalar(value: unknown): string {
	if (value == null || value === "") return "—";
	if (typeof value === "object") {
		try {
			return JSON.stringify(value);
		} catch {
			return String(value);
		}
	}
	if (typeof value === "string" && /^\d{4}-\d{2}-\d{2}T/.test(value)) {
		return dayjs(value).format("YYYY-MM-DD HH:mm");
	}
	return String(value);
}

function normalizeTableCountRows(raw: unknown): LakeTableCountRow[] {
	const r = asRecord(raw) ?? {};
	const items = Array.isArray(r.items) ? r.items : [];
	return items
		.map((it, idx) => {
			const row = asRecord(it) ?? {};
			const name = typeof row.table_name === "string" ? row.table_name : "";
			const rc = toFiniteNumber(row.row_count) ?? 0;
			return {
				key: name || `t-${idx}`,
				table_name: name,
				row_count: rc,
			};
		})
		.filter((row) => row.table_name)
		.sort((a, b) => b.row_count - a.row_count);
}

function normalizeQualityDistRows(raw: unknown): QualityDistRow[] {
	const items = asRecordArray(asRecord(raw)?.items);
	return items
		.map((it, idx) => {
			const q =
				typeof it.quality === "string"
					? it.quality
					: typeof it.tag_value === "string"
						? it.tag_value
						: "";
			return {
				key: q || `q-${idx}`,
				quality: q || "—",
				asset_count: toFiniteNumber(it.asset_count) ?? 0,
			};
		})
		.sort((a, b) => b.asset_count - a.asset_count);
}

function lakehouseItemRows(raw: unknown): Record<string, unknown>[] {
	return asRecordArray(asRecord(raw)?.items);
}

function normalizeOverview(raw: LakehouseOverviewResponse): DashboardMetrics {
	const r = asRecord(raw) ?? {};
	return {
		assetTotal: toFiniteNumber(r.asset_total ?? r.silver_asset_rows),
		bronzeEventRows: toFiniteNumber(
			r.bronze_event_rows ?? r.bronze_rows ?? r.bronze_row_count,
		),
		todayNew: toFiniteNumber(r.today_new_assets),
		yesterdayNew: toFiniteNumber(r.yesterday_new_assets),
		weekNew: toFiniteNumber(r.week_new_assets),
		day7Avg: toFiniteNumber(r.day7_avg_new_assets),
		goldLatestDate:
			typeof r.gold_latest_date === "string" && r.gold_latest_date
				? r.gold_latest_date
				: null,
		dataLagHours: toFiniteNumber(r.data_lag_hours),
	};
}

function normalizeGrowth(
	raw: LakehouseAssetGrowthResponse,
): AssetGrowthPoint[] {
	const items = asRecordArray(asRecord(raw)?.items);
	return items
		.map((it) => ({
			date: typeof it.event_date === "string" ? it.event_date : "",
			newAssets: toFiniteNumber(it.new_assets) ?? 0,
			cumulative: toFiniteNumber(it.cumulative_assets) ?? 0,
		}))
		.filter((p) => p.date)
		.sort((a, b) => a.date.localeCompare(b.date));
}

function normalizeDailyByType(
	raw: LakehouseEventDailyResponse,
): DailyEventPoint[] {
	const items = asRecordArray(asRecord(raw)?.items);
	return items
		.map((it) => ({
			date: typeof it.event_date === "string" ? it.event_date : "",
			eventType: typeof it.event_type === "string" ? it.event_type : "unknown",
			count: toFiniteNumber(it.asset_count) ?? 0,
		}))
		.filter((it) => it.date);
}

function normalizeShare(
	raw: LakehouseEventTypeShareResponse,
	daily: DailyEventPoint[],
): { rows: EventTypeShareRow[]; date: string | null } {
	const r = asRecord(raw) ?? {};
	const items = asRecordArray(r.items);
	const last30ByType = new Map<string, number>();
	for (const d of daily) {
		last30ByType.set(
			d.eventType,
			(last30ByType.get(d.eventType) ?? 0) + d.count,
		);
	}
	const rows = items
		.map((it, idx) => {
			const eventType =
				typeof it.event_type === "string" ? it.event_type : `idx-${idx}`;
			const count = toFiniteNumber(it.asset_count) ?? 0;
			let ratio = toFiniteNumber(it.ratio) ?? 0;
			if (ratio > 1 && ratio <= 100) ratio /= 100;
			return {
				key: eventType,
				eventType,
				count,
				ratio: Math.max(0, Math.min(1, ratio)),
				last30Count: last30ByType.get(eventType) ?? 0,
			};
		})
		.filter((row) => row.count > 0)
		.sort((a, b) => b.count - a.count);
	const date = typeof r.date === "string" ? r.date : null;
	return { rows, date };
}

function freshnessTag(latestDate: string | null, lagHours: number | null) {
	if (!latestDate) return { color: "default", label: "无数据" };
	const hours = lagHours ?? 0;
	if (hours <= 24)
		return { color: "green", label: `滞后 ${Math.round(hours)} 小时` };
	if (hours <= 72)
		return { color: "orange", label: `滞后 ${Math.round(hours)} 小时` };
	return { color: "red", label: `滞后 ${Math.round(hours)} 小时` };
}

const chartTooltip = {
	trigger: "axis",
	axisPointer: { type: "cross" as const, crossStyle: { color: "#94a3b8" } },
	backgroundColor: "rgba(15, 23, 42, 0.92)",
	borderWidth: 0,
	textStyle: { color: "#f8fafc", fontSize: 12 },
};

function buildGrowthOption(points: AssetGrowthPoint[]) {
	const labels = points.map((p) => dayjs(p.date).format("MM-DD"));
	const fullDates = points.map((p) => p.date);
	return {
		color: ["#3b82f6", "#22c55e"],
		textStyle: { color: "#64748b", fontSize: 11 },
		tooltip: {
			...chartTooltip,
			formatter: (params: unknown) => {
				const arr = Array.isArray(params) ? params : [params];
				const idx = (arr[0] as { dataIndex?: number })?.dataIndex ?? 0;
				const day = fullDates[idx] ?? "";
				const lines = [`<b>${day}</b>`];
				for (const p of arr) {
					const x = p as {
						marker?: string;
						seriesName?: string;
						value?: number;
					};
					lines.push(
						`${x.marker ?? ""} ${x.seriesName ?? ""}: <b>${(x.value ?? 0).toLocaleString()}</b>`,
					);
				}
				return lines.join("<br/>");
			},
		},
		legend: {
			data: ["每日新增资产", "累计活跃资产"],
			top: 6,
			textStyle: { fontSize: 12, color: "#64748b" },
		},
		grid: { left: 64, right: 52, top: 48, bottom: 68 },
		toolbox: {
			right: 8,
			top: 4,
			feature: {
				dataZoom: {
					yAxisIndex: false,
					title: { zoom: "框选缩放", back: "还原" },
				},
				restore: { title: "还原" },
				saveAsImage: { name: "资产增长", title: "导出图片" },
			},
			iconStyle: { borderColor: "#94a3b8" },
		},
		dataZoom: [
			{
				type: "slider",
				xAxisIndex: 0,
				height: 22,
				bottom: 6,
				borderColor: "#e2e8f0",
				fillerColor: "rgba(59, 130, 246, 0.2)",
				handleStyle: { color: "#3b82f6", borderColor: "#3b82f6" },
				textStyle: { color: "#64748b", fontSize: 10 },
				moveHandleSize: 6,
			},
			{
				type: "inside",
				xAxisIndex: 0,
				zoomOnMouseWheel: true,
				moveOnMouseMove: true,
				moveOnMouseWheel: true,
			},
		],
		xAxis: {
			type: "category",
			boundaryGap: true,
			data: labels,
			axisLine: { lineStyle: { color: "#e2e8f0" } },
			axisLabel: { fontSize: 11, rotate: labels.length > 24 ? 32 : 0 },
		},
		yAxis: [
			{
				type: "value",
				name: "新增",
				splitLine: { lineStyle: { type: "dashed" as const, color: "#f1f5f9" } },
				axisLabel: {
					fontSize: 11,
					formatter: (value: number) => {
						if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
						if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k`;
						return value.toString();
					},
				},
			},
			{
				type: "value",
				name: "累计",
				splitLine: { show: false },
				axisLabel: {
					fontSize: 11,
					formatter: (value: number) => {
						if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
						if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k`;
						return value.toString();
					},
				},
			},
		],
		series: [
			{
				name: "每日新增资产",
				type: "bar",
				barMaxWidth: 36,
				data: points.map((p) => p.newAssets),
				itemStyle: {
					color: {
						type: "linear",
						x: 0,
						y: 0,
						x2: 0,
						y2: 1,
						colorStops: [
							{ offset: 0, color: "#60a5fa" },
							{ offset: 1, color: "#2563eb" },
						],
					},
					borderRadius: [6, 6, 0, 0],
				},
			},
			{
				name: "累计活跃资产",
				type: "line",
				yAxisIndex: 1,
				smooth: 0.35,
				showSymbol: points.length <= 16,
				symbolSize: 6,
				lineStyle: { width: 3, color: "#22c55e" },
				itemStyle: { color: "#22c55e" },
				areaStyle: {
					color: "rgba(34, 197, 94, 0.12)",
				},
				data: points.map((p) => p.cumulative),
			},
		],
	};
}

function buildStackedOption(daily: DailyEventPoint[]) {
	const dates = Array.from(new Set(daily.map((d) => d.date))).sort();
	const types = Array.from(new Set(daily.map((d) => d.eventType)));
	const byType: Record<string, Record<string, number>> = {};
	for (const t of types) byType[t] = {};
	for (const d of daily) byType[d.eventType][d.date] = d.count;
	const palette = [
		"#3b82f6",
		"#22c55e",
		"#f59e0b",
		"#ef4444",
		"#8b5cf6",
		"#06b6d4",
	];
	const labels = dates.map((d) => dayjs(d).format("MM-DD"));
	return {
		color: palette,
		textStyle: { color: "#64748b", fontSize: 11 },
		tooltip: chartTooltip,
		legend: {
			data: types,
			top: 6,
			type: "scroll",
			textStyle: { fontSize: 12, color: "#64748b" },
		},
		grid: { left: 64, right: 12, top: 48, bottom: 68 },
		toolbox: {
			right: 8,
			top: 4,
			feature: {
				dataZoom: {
					yAxisIndex: false,
					title: { zoom: "框选缩放", back: "还原" },
				},
				restore: { title: "还原" },
				saveAsImage: { name: "事件分布", title: "导出图片" },
			},
			iconStyle: { borderColor: "#94a3b8" },
		},
		dataZoom: [
			{
				type: "slider",
				xAxisIndex: 0,
				height: 22,
				bottom: 6,
				borderColor: "#e2e8f0",
				fillerColor: "rgba(59, 130, 246, 0.15)",
				handleStyle: { color: "#3b82f6" },
				textStyle: { color: "#64748b", fontSize: 10 },
			},
			{
				type: "inside",
				xAxisIndex: 0,
				zoomOnMouseWheel: true,
				moveOnMouseMove: true,
			},
		],
		xAxis: {
			type: "category",
			data: labels,
			axisLine: { lineStyle: { color: "#e2e8f0" } },
			axisLabel: { fontSize: 11, rotate: labels.length > 24 ? 32 : 0 },
		},
		yAxis: {
			type: "value",
			splitLine: { lineStyle: { type: "dashed" as const, color: "#f1f5f9" } },
			axisLabel: {
				fontSize: 11,
				formatter: (value: number) => {
					if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
					if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k`;
					return value.toString();
				},
			},
		},
		series: types.map((t, i) => ({
			name: t,
			type: "bar",
			stack: "events",
			barMaxWidth: 40,
			emphasis: { focus: "series" as const },
			data: dates.map((d) => byType[t][d] ?? 0),
			itemStyle: { color: palette[i % palette.length] },
		})),
	};
}

function computeWindowStats(
	growth: AssetGrowthPoint[],
	daily: DailyEventPoint[],
	windowDays: number,
) {
	const evTotal = daily.reduce((s, d) => s + d.count, 0);
	const newSum = growth.reduce((s, g) => s + g.newAssets, 0);
	let peakDate = "";
	let peakVal = 0;
	for (const g of growth) {
		if (g.newAssets > peakVal) {
			peakVal = g.newAssets;
			peakDate = g.date;
		}
	}
	const byType = new Map<string, number>();
	for (const d of daily) {
		byType.set(d.eventType, (byType.get(d.eventType) ?? 0) + d.count);
	}
	let topType = "—";
	let topCount = 0;
	for (const [t, c] of byType) {
		if (c > topCount) {
			topCount = c;
			topType = t;
		}
	}
	const denom = Math.max(1, windowDays);
	return {
		evTotal,
		newSum,
		avgEventsPerDay: evTotal / denom,
		avgNewAssetsPerDay: newSum / denom,
		peakDate,
		peakVal,
		topType,
		topTypeCount: topCount,
	};
}

const FAILURE_MODE_LABELS: Record<string, string> = {
	timeout: "超时",
	algo_error: "算法异常",
	sensor_fault: "传感器故障",
	env_mismatch: "环境不匹配",
	low_quality: "数据质量低",
	annotation_drift: "标注偏移",
	unknown: "未知",
};

export default function DashboardPage() {
	const versionLabel = getAppVersionLabel();
	const [windowDays, setWindowDays] = useState<7 | 30 | 60 | 90>(30);
	const [pageReady, setPageReady] = useState(false);
	const [rangeBusy, setRangeBusy] = useState(false);
	const firstFetch = useRef(true);

	const [metrics, setMetrics] = useState<DashboardMetrics>({
		assetTotal: null,
		bronzeEventRows: null,
		todayNew: null,
		yesterdayNew: null,
		weekNew: null,
		day7Avg: null,
		goldLatestDate: null,
		dataLagHours: null,
	});
	const [growth, setGrowth] = useState<AssetGrowthPoint[]>([]);
	const [daily, setDaily] = useState<DailyEventPoint[]>([]);
	const [shareRows, setShareRows] = useState<EventTypeShareRow[]>([]);
	const [shareDate, setShareDate] = useState<string | null>(null);
	const [tableCountRows, setTableCountRows] = useState<LakeTableCountRow[]>([]);
	const [qualityRows, setQualityRows] = useState<QualityDistRow[]>([]);
	const [trainingRows, setTrainingRows] = useState<Record<string, unknown>[]>(
		[],
	);
	const [recomputeRows, setRecomputeRows] = useState<Record<string, unknown>[]>(
		[],
	);
	const [customerRows, setCustomerRows] = useState<Record<string, unknown>[]>(
		[],
	);
	const [failureClusters, setFailureClusters] = useState<FailureClusterItem[]>(
		[],
	);
	const [failureClustersErr, setFailureClustersErr] = useState<string | null>(
		null,
	);
	const [lakeNotes, setLakeNotes] = useState<{
		quality: string | null;
		training: string | null;
		recompute: string | null;
		customer: string | null;
	}>({
		quality: null,
		training: null,
		recompute: null,
		customer: null,
	});
	const [errors, setErrors] = useState<EndpointErrors>({});

	useEffect(() => {
		let cancelled = false;
		const load = async () => {
			if (!firstFetch.current) setRangeBusy(true);
			else setRangeBusy(false);
			setErrors({});
			const errs: EndpointErrors = {};

			try {
				// Silver/Gold materializations for training/recompute are still on
				// the roadmap (ADR-001 §Hybrid layout); their cards render an
				// explicit "尚未物化" placeholder. quality / customer-replay are
				// re-enabled in this build — both run directly against PG.
				const SILVER_GOLD_NOTE = "Silver/Gold 表尚未物化，详见路线图 P1。";
				const defaultShareDate = dayjs().format("YYYY-MM-DD");
				const [ov, gr, dl, tbl, qual, cust, fc, sh] = await Promise.allSettled([
					lakehouseApi.overview(),
					lakehouseApi.assetGrowth(windowDays),
					lakehouseApi.eventDaily(windowDays),
					lakehouseApi.tables(),
					lakehouseApi.qualityDistribution(`${windowDays}d`),
					lakehouseApi.customerReplay(),
					lakehouseApi.failureClusters(windowDays),
					lakehouseApi.eventTypeShare(defaultShareDate),
				]);

				let m: DashboardMetrics = {
					assetTotal: null,
					bronzeEventRows: null,
					todayNew: null,
					yesterdayNew: null,
					weekNew: null,
					day7Avg: null,
					goldLatestDate: null,
					dataLagHours: null,
				};
				if (ov.status === "fulfilled") m = normalizeOverview(ov.value);
				else errs.overview = extractApiErrorMessage(ov.reason, "概览加载失败");

				const growthPoints =
					gr.status === "fulfilled" ? normalizeGrowth(gr.value) : [];
				if (gr.status === "rejected")
					errs.growth = extractApiErrorMessage(gr.reason, "资产增长加载失败");

				const dailyPoints =
					dl.status === "fulfilled" ? normalizeDailyByType(dl.value) : [];
				if (dl.status === "rejected")
					errs.daily = extractApiErrorMessage(dl.reason, "事件趋势加载失败");

				let tableRows: LakeTableCountRow[] = [];
				if (tbl.status === "fulfilled")
					tableRows = normalizeTableCountRows(tbl.value);
				else errs.tables = extractApiErrorMessage(tbl.reason, "表行数加载失败");

				let qRows: QualityDistRow[] = [];
				const qNote: string | null = null;
				if (qual.status === "fulfilled") {
					qRows = normalizeQualityDistRows(qual.value);
				} else {
					errs.quality = extractApiErrorMessage(
						qual.reason,
						"Quality 分布加载失败",
					);
				}

				let custR: Record<string, unknown>[] = [];
				const custNote: string | null = null;
				if (cust.status === "fulfilled") {
					custR = lakehouseItemRows(cust.value);
				} else {
					errs.customer = extractApiErrorMessage(
						cust.reason,
						"客户交付回放加载失败",
					);
				}

				const trainR: Record<string, unknown>[] = [];
				const trainNote: string | null = SILVER_GOLD_NOTE;
				const recR: Record<string, unknown>[] = [];
				const recNote: string | null = SILVER_GOLD_NOTE;

				let fcItems: FailureClusterItem[] = [];
				if (fc.status === "fulfilled") {
					fcItems = fc.value.items ?? [];
				} else
					errs.failureClusters = extractApiErrorMessage(
						fc.reason,
						"失败模式分布加载失败",
					);

				let selectedDate: string | null =
					dailyPoints.length > 0
						? dailyPoints[dailyPoints.length - 1].date
						: m.goldLatestDate;
				if (!selectedDate) selectedDate = dayjs().format("YYYY-MM-DD");

				let normalized: { rows: EventTypeShareRow[]; date: string | null } = {
					rows: [],
					date: selectedDate,
				};
				if (sh.status === "fulfilled")
					normalized = normalizeShare(sh.value, dailyPoints);
				else
					errs.share = extractApiErrorMessage(
						sh.reason,
						"事件类型分布加载失败",
					);

				if (cancelled) return;
				setMetrics(m);
				setGrowth(growthPoints);
				setDaily(dailyPoints);
				setShareRows(normalized.rows);
				setShareDate(normalized.date ?? selectedDate);
				setTableCountRows(tableRows);
				setQualityRows(qRows);
				setTrainingRows(trainR);
				setRecomputeRows(recR);
				setCustomerRows(custR);
				setFailureClusters(fcItems);
				setFailureClustersErr(
					errs.failureClusters ? errs.failureClusters : null,
				);
				setLakeNotes({
					quality: qNote,
					training: trainNote,
					recompute: recNote,
					customer: custNote,
				});
				setErrors(errs);
			} catch (e) {
				console.error("[DashboardPage] unexpected error in load():", e);
				if (Object.keys(errs).length > 0) {
					setErrors(errs);
				}
			} finally {
				firstFetch.current = false;
				if (!cancelled) {
					setPageReady(true);
					setRangeBusy(false);
				}
			}
		};
		void load();
		return () => {
			cancelled = true;
		};
	}, [windowDays]);

	const windowStats = useMemo(
		() => computeWindowStats(growth, daily, windowDays),
		[growth, daily, windowDays],
	);

	const todayTrend = useMemo<KpiTrend | undefined>(() => {
		if (metrics.todayNew == null || metrics.yesterdayNew == null)
			return undefined;
		if (metrics.yesterdayNew === 0) {
			if (metrics.todayNew === 0)
				return { direction: "flat", text: "与昨日持平" };
			return { direction: "up", text: "昨日为零" };
		}
		const diff = metrics.todayNew - metrics.yesterdayNew;
		const pct = (diff / metrics.yesterdayNew) * 100;
		if (Math.abs(pct) < 0.5) return { direction: "flat", text: "与昨日持平" };
		return {
			direction: pct >= 0 ? "up" : "down",
			text: `较昨日 ${pct >= 0 ? "+" : ""}${pct.toFixed(1)}%`,
		};
	}, [metrics.todayNew, metrics.yesterdayNew]);

	const freshness = freshnessTag(metrics.goldLatestDate, metrics.dataLagHours);

	const shareColumns: ColumnsType<EventTypeShareRow> = useMemo(
		() => [
			{
				title: "事件类型",
				dataIndex: "eventType",
				key: "eventType",
				render: (v: string) => <Tag>{v}</Tag>,
			},
			{
				title: "当日数量",
				dataIndex: "count",
				key: "count",
				align: "right",
				width: 120,
				render: (v: number) => v.toLocaleString(),
			},
			{
				title: "当日占比",
				key: "ratio",
				width: 220,
				render: (_: unknown, row) => (
					<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
						<div
							style={{
								flex: 1,
								height: 6,
								borderRadius: 99,
								background: "#F1F5F9",
								overflow: "hidden",
							}}
						>
							<div
								style={{
									width: `${(row.ratio * 100).toFixed(1)}%`,
									height: "100%",
									background: "var(--color-primary)",
								}}
							/>
						</div>
						<Text type="secondary" style={{ fontSize: 12, minWidth: 48 }}>
							{(row.ratio * 100).toFixed(1)}%
						</Text>
					</div>
				),
			},
			{
				title: `近 ${windowDays} 天累计`,
				dataIndex: "last30Count",
				key: "last30Count",
				align: "right",
				width: 140,
				render: (v: number) => v.toLocaleString(),
			},
		],
		[windowDays],
	);

	const tableCountColumns: ColumnsType<LakeTableCountRow> = useMemo(
		() => [
			{
				title: "表名",
				dataIndex: "table_name",
				key: "table_name",
				render: (v: string) => (
					<Text code style={{ fontSize: 12 }}>
						{v}
					</Text>
				),
			},
			{
				title: "行数",
				dataIndex: "row_count",
				key: "row_count",
				align: "right",
				width: 120,
				render: (v: number) => v.toLocaleString(),
			},
		],
		[],
	);

	const qualityColumns: ColumnsType<QualityDistRow> = useMemo(
		() => [
			{
				title: "quality 取值",
				dataIndex: "quality",
				key: "quality",
				render: (v: string) => <Tag>{v}</Tag>,
			},
			{
				title: "资产数",
				dataIndex: "asset_count",
				key: "asset_count",
				align: "right",
				width: 120,
				render: (v: number) => v.toLocaleString(),
			},
		],
		[],
	);

	const trainingColumns: ColumnsType<Record<string, unknown>> = useMemo(
		() => [
			{
				title: "asset_id",
				dataIndex: "asset_id",
				key: "asset_id",
				ellipsis: true,
				width: 200,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "lifecycle",
				dataIndex: "lifecycle_state",
				key: "lifecycle_state",
				width: 100,
				render: (v: unknown) =>
					typeof v === "string" && v ? (
						<Tag>{v}</Tag>
					) : (
						formatLakehouseScalar(v)
					),
			},
			{
				title: "asset_type",
				dataIndex: "asset_type",
				key: "asset_type",
				width: 100,
				ellipsis: true,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "algo_key",
				dataIndex: "algo_key",
				key: "algo_key",
				ellipsis: true,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "algo_status",
				dataIndex: "algo_status",
				key: "algo_status",
				width: 96,
				render: (v: unknown) =>
					typeof v === "string" && v ? (
						<Tag>{v}</Tag>
					) : (
						formatLakehouseScalar(v)
					),
			},
			{
				title: "quality",
				dataIndex: "quality",
				key: "quality",
				width: 88,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "created_at",
				dataIndex: "created_at",
				key: "created_at",
				width: 140,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
		],
		[],
	);

	const recomputeColumns: ColumnsType<Record<string, unknown>> = useMemo(
		() => [
			{
				title: "asset_id",
				dataIndex: "asset_id",
				key: "asset_id",
				ellipsis: true,
				width: 200,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "algo",
				key: "algo",
				width: 160,
				ellipsis: true,
				render: (_: unknown, row) => {
					const n = typeof row.algo_name === "string" ? row.algo_name : "";
					const ver =
						typeof row.algo_version === "string" ? row.algo_version : "";
					if (!n && !ver) return "—";
					return `${n || "?"}@${ver || "?"}`;
				},
			},
			{
				title: "status",
				dataIndex: "status",
				key: "status",
				width: 88,
				render: (v: unknown) =>
					typeof v === "string" && v ? (
						<Tag>{v}</Tag>
					) : (
						formatLakehouseScalar(v)
					),
			},
			{
				title: "updated_at",
				dataIndex: "updated_at",
				key: "updated_at",
				width: 140,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
		],
		[],
	);

	const customerColumns: ColumnsType<Record<string, unknown>> = useMemo(
		() => [
			{
				title: "delivery_id",
				dataIndex: "delivery_id",
				key: "delivery_id",
				ellipsis: true,
				width: 200,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "status",
				dataIndex: "status",
				key: "status",
				width: 88,
				render: (v: unknown) =>
					typeof v === "string" && v ? (
						<Tag>{v}</Tag>
					) : (
						formatLakehouseScalar(v)
					),
			},
			{
				title: "delivered_at",
				dataIndex: "delivered_at",
				key: "delivered_at",
				width: 140,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "asset_id",
				dataIndex: "asset_id",
				key: "asset_id",
				ellipsis: true,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
			{
				title: "mcap_file_id",
				dataIndex: "mcap_file_id",
				key: "mcap_file_id",
				ellipsis: true,
				render: (v: unknown) => formatLakehouseScalar(v),
			},
		],
		[],
	);

	const cardElevated = {
		flex: 1,
		borderRadius: dashSurface.cardRadius,
		border: dashSurface.cardBorder,
		boxShadow: dashSurface.cardShadow,
		background: "var(--color-bg-container, #fff)",
	} as const;

	const cardHeaderBar = {
		borderBottom: "1px solid rgba(226, 232, 240, 0.95)",
		minHeight: 44,
		fontWeight: 600,
	};

	const failing = Object.entries(errors).filter(([, m]) => !!m);

	const growthOption = useMemo(() => buildGrowthOption(growth), [growth]);
	const stackedOption = useMemo(() => buildStackedOption(daily), [daily]);
	const failureClusterOption = useMemo(
		() => ({
			tooltip: {
				trigger: "item",
				formatter: "{b}: {c} ({d}%)",
			},
			legend: {
				type: "scroll",
				orient: "vertical",
				right: 10,
			},
			series: [
				{
					type: "pie",
					radius: ["40%", "70%"],
					center: ["40%", "50%"],
					data: failureClusters.map((fc) => ({
						name: FAILURE_MODE_LABELS[fc.failure_mode] ?? fc.failure_mode,
						value: fc.affected_assets,
					})),
					emphasis: {
						itemStyle: { shadowBlur: 10 },
					},
				},
			],
		}),
		[failureClusters],
	);

	if (!pageReady) return <PageLoading />;

	return (
		<div style={{ maxWidth: 1400, paddingBottom: 8 }}>
			<div
				style={{
					marginBottom: 20,
					padding: "14px 18px",
					borderRadius: dashSurface.cardRadius,
					border: dashSurface.heroBorder,
					background: "var(--color-bg-container, #fff)",
					boxShadow: dashSurface.cardShadow,
					display: "flex",
					flexWrap: "wrap",
					alignItems: "flex-start",
					justifyContent: "space-between",
					gap: 16,
				}}
			>
				<div style={{ minWidth: 0, flex: "1 1 240px" }}>
					<Title level={4} style={{ margin: 0, fontWeight: 600 }}>
						业务概览
					</Title>
					<Text type="secondary" style={{ fontSize: 13, lineHeight: 1.55 }}>
						数据来源：湖仓（Iceberg via BigQuery）·
						按日聚合；图表底部滑块与框选可缩放时间范围
					</Text>
					<Text type="secondary" style={{ fontSize: 12, display: "block" }}>
						前端版本：{versionLabel}
					</Text>
				</div>
				<Radio.Group
					id="dashboard-time-range"
					value={windowDays}
					onChange={(e) => setWindowDays(e.target.value)}
					optionType="button"
					buttonStyle="solid"
				>
					<Radio.Button value={7}>近 7 天</Radio.Button>
					<Radio.Button value={30}>近 30 天</Radio.Button>
					<Radio.Button value={60}>近 60 天</Radio.Button>
					<Radio.Button value={90}>近 90 天</Radio.Button>
				</Radio.Group>
				{rangeBusy && <Spin size="small" style={{ marginLeft: 12 }} />}
			</div>

			{failing.length > 0 && (
				<Alert
					type="warning"
					showIcon
					message="部分指标加载失败，已展示可用数据"
					description={
						<ul
							style={{
								margin: "8px 0 0",
								paddingLeft: 20,
								marginBottom: 0,
							}}
						>
							{failing.map(([k, m]) => (
								<li key={k} style={{ marginBottom: 4 }}>
									<Text strong style={{ textTransform: "capitalize" }}>
										{k}
									</Text>
									{" — "}
									{m}
								</li>
							))}
						</ul>
					}
					style={{
						marginBottom: 16,
						borderRadius: dashSurface.cardRadius,
						border: dashSurface.cardBorder,
					}}
				/>
			)}

			<div>
				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="资产总量"
							value={metrics.assetTotal?.toLocaleString() ?? "—"}
							hint="活跃资产（当前态）"
							icon={<DatabaseOutlined />}
						/>
					</Col>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="今日新增"
							value={metrics.todayNew?.toLocaleString() ?? "—"}
							trend={todayTrend}
							icon={<ThunderboltOutlined />}
						/>
					</Col>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="近 7 天新增"
							value={metrics.weekNew?.toLocaleString() ?? "—"}
							icon={<RiseOutlined />}
						/>
					</Col>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="7 日日均新增"
							value={
								metrics.day7Avg != null
									? Math.round(metrics.day7Avg).toLocaleString()
									: "—"
							}
							icon={<LineChartOutlined />}
						/>
					</Col>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="数据新鲜度"
							value={
								metrics.goldLatestDate
									? dayjs(metrics.goldLatestDate).format("MM-DD")
									: "—"
							}
							suffix={metrics.goldLatestDate ? "最新日" : undefined}
							hint={
								metrics.goldLatestDate ? (
									<Tag color={freshness.color} style={{ marginLeft: 0 }}>
										{freshness.label}
									</Tag>
								) : (
									"等待入湖"
								)
							}
							icon={<ClockCircleOutlined />}
						/>
					</Col>
					<Col xs={24} sm={12} lg={4} style={{ display: "flex" }}>
						<KpiCard
							title="累计事件日志"
							value={metrics.bronzeEventRows?.toLocaleString() ?? "—"}
							hint="Bronze 层原始事件条数"
							icon={<DatabaseOutlined />}
							iconTint="slate"
						/>
					</Col>
				</Row>

				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} sm={12} lg={6} style={{ display: "flex" }}>
						<KpiCard
							title={`所选 ${windowDays} 天 · 事件总量`}
							value={windowStats.evTotal.toLocaleString()}
							hint={`日均 ${Math.round(windowStats.avgEventsPerDay).toLocaleString()} 条`}
							icon={<LineChartOutlined />}
							iconTint="slate"
						/>
					</Col>
					<Col xs={24} sm={12} lg={6} style={{ display: "flex" }}>
						<KpiCard
							title={`所选 ${windowDays} 天 · 新增资产`}
							value={windowStats.newSum.toLocaleString()}
							hint={`日均 ${Math.round(windowStats.avgNewAssetsPerDay).toLocaleString()} 个`}
							icon={<ThunderboltOutlined />}
							iconTint="slate"
						/>
					</Col>
					<Col xs={24} sm={12} lg={6} style={{ display: "flex" }}>
						<KpiCard
							title="峰值日新增"
							value={
								windowStats.peakVal > 0
									? windowStats.peakVal.toLocaleString()
									: "—"
							}
							hint={
								windowStats.peakDate
									? dayjs(windowStats.peakDate).format("YYYY-MM-DD")
									: "无单日峰值"
							}
							icon={<RiseOutlined />}
							iconTint="slate"
						/>
					</Col>
					<Col xs={24} sm={12} lg={6} style={{ display: "flex" }}>
						<KpiCard
							title="主力事件类型"
							value={windowStats.topType}
							hint={`${windowStats.topTypeCount.toLocaleString()} 次（区间内合计）`}
							icon={<PieChartOutlined />}
							iconTint="slate"
						/>
					</Col>
				</Row>

				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} lg={12} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title={`资产增长（${windowDays} 天）`}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 16 },
							}}
						>
							{growth.length > 0 ? (
								<LazyECharts
									option={growthOption}
									style={{ height: 340, width: "100%" }}
								/>
							) : (
								<TableEmptyState summary="所选时间范围内暂无资产增长序列" />
							)}
						</Card>
					</Col>
					<Col xs={24} lg={12} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title={`事件分布（${windowDays} 天）`}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 16 },
							}}
						>
							{daily.length > 0 ? (
								<LazyECharts
									option={stackedOption}
									style={{ height: 340, width: "100%" }}
								/>
							) : (
								<TableEmptyState summary="所选时间范围内暂无按日事件分布" />
							)}
						</Card>
					</Col>
				</Row>

				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title={`最新日事件类型分布${shareDate ? ` · ${shareDate}` : ""}`}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 16 },
							}}
						>
							{shareRows.length > 0 ? (
								<Table
									rowKey="key"
									columns={shareColumns}
									dataSource={shareRows}
									size="small"
									pagination={false}
								/>
							) : (
								<TableEmptyState summary="暂无最新日事件类型占比数据" />
							)}
						</Card>
					</Col>
				</Row>

				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title={`近 ${windowDays} 天失败模式分布`}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 16 },
							}}
						>
							{failureClustersErr ? (
								<TableEmptyState summary={failureClustersErr} />
							) : failureClusters.length === 0 ? (
								<TableEmptyState
									summary={`近 ${windowDays} 天暂无算法失败数据`}
								/>
							) : (
								<LazyECharts
									style={{ height: 300 }}
									option={failureClusterOption}
								/>
							)}
						</Card>
					</Col>
				</Row>

				<Divider orientation="left" plain style={{ margin: "8px 0 16px" }}>
					<Text strong>湖仓业务表</Text>
				</Divider>
				<div style={{ marginBottom: 16 }}>
					<Text type="secondary" style={{ fontSize: 13, lineHeight: 1.55 }}>
						MVP 示例筛选：训练 / 重算 / 交付为接口默认参数抽样，每表最多 100
						行； Quality 分布为近 {windowDays}{" "}
						天新建资产（与上方时间范围开关联动）。
					</Text>
				</div>

				<Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
					<Col xs={24} lg={12} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title="Iceberg 表行数"
							extra={
								<TableOutlined style={{ color: "var(--color-primary)" }} />
							}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 12 },
							}}
						>
							{tableCountRows.length > 0 ? (
								<Table
									rowKey="key"
									columns={tableCountColumns}
									dataSource={tableCountRows}
									size="small"
									pagination={false}
									scroll={{ y: 260 }}
								/>
							) : (
								<TableEmptyState
									summary="暂无表级行数统计"
									note="请确认 BigQuery 已启用且目标 dataset 下 Iceberg 外部表已创建。"
								/>
							)}
						</Card>
					</Col>
					<Col xs={24} lg={12} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title={`近 ${windowDays} 天新建资产 · quality 分布`}
							extra={
								<BarChartOutlined style={{ color: "var(--color-primary)" }} />
							}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 12 },
							}}
						>
							{qualityRows.length > 0 ? (
								<Table
									rowKey="key"
									columns={qualityColumns}
									dataSource={qualityRows}
									size="small"
									pagination={false}
									scroll={{ y: 260 }}
								/>
							) : (
								<TableEmptyState
									summary="暂无 quality 标签分布"
									note={lakeNotes.quality}
								/>
							)}
						</Card>
					</Col>
				</Row>

				<Row gutter={[16, 16]}>
					<Col xs={24} lg={8} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title="训练集快照条目"
							extra={<BookOutlined style={{ color: "var(--color-primary)" }} />}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 12 },
							}}
						>
							<Text
								type="secondary"
								style={{ display: "block", marginBottom: 8, fontSize: 12 }}
							>
								gold_dataset_snapshot_items · snapshot_id =
								mvp_hand_tracking_quality_v1
							</Text>
							{trainingRows.length > 0 ? (
								<Table
									rowKey={(_, i) => `train-${i}`}
									columns={trainingColumns}
									dataSource={trainingRows}
									size="small"
									scroll={{ x: 900, y: 260 }}
									pagination={{ pageSize: 8, showSizeChanger: false }}
								/>
							) : (
								<TableEmptyState
									summary="暂无训练集快照抽样行"
									note={lakeNotes.training}
								/>
							)}
						</Card>
					</Col>
					<Col xs={24} lg={8} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title="算法重算候选"
							extra={
								<ReloadOutlined style={{ color: "var(--color-primary)" }} />
							}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 12 },
							}}
						>
							<Text
								type="secondary"
								style={{ display: "block", marginBottom: 8, fontSize: 12 }}
							>
								silver_asset_algo_latest · algo_key = hand_tracking@1.2.0
							</Text>
							{recomputeRows.length > 0 ? (
								<Table
									rowKey={(_, i) => `rec-${i}`}
									columns={recomputeColumns}
									dataSource={recomputeRows}
									size="small"
									scroll={{ x: 640, y: 260 }}
									pagination={{ pageSize: 8, showSizeChanger: false }}
								/>
							) : (
								<TableEmptyState
									summary="暂无算法重算候选抽样行"
									note={lakeNotes.recompute}
								/>
							)}
						</Card>
					</Col>
					<Col xs={24} lg={8} style={{ display: "flex" }}>
						<Card
							variant="borderless"
							title="客户交付回放"
							extra={<SendOutlined style={{ color: "var(--color-primary)" }} />}
							size="small"
							style={cardElevated}
							styles={{
								header: cardHeaderBar,
								body: { padding: 12 },
							}}
						>
							<Text
								type="secondary"
								style={{ display: "block", marginBottom: 8, fontSize: 12 }}
							>
								silver_deliveries_current · customer_id = urn:grace:customer:A
							</Text>
							{customerRows.length > 0 ? (
								<Table
									rowKey={(_, i) => `cust-${i}`}
									columns={customerColumns}
									dataSource={customerRows}
									size="small"
									scroll={{ x: 720, y: 260 }}
									pagination={{ pageSize: 8, showSizeChanger: false }}
								/>
							) : (
								<TableEmptyState
									summary="暂无客户交付回放抽样行"
									note={lakeNotes.customer}
								/>
							)}
						</Card>
					</Col>
				</Row>
			</div>
		</div>
	);
}
