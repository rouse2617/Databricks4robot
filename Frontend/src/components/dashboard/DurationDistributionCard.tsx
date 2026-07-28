// CYB-4303: dashboard 数据时长分布 card.
//
// Replaces the two event cards ("事件分布" + "最新日事件类型分布") with a
// single asset-duration histogram + summary stats. Buckets always ship in
// the CYB-4294 canonical order — the backend fills any missing bucket with
// zero — so this component doesn't need to reason about gaps.
//
// CYB-4323: the histogram is an ECharts bar chart (same engine as the 资产增长
// chart) rather than raw flexbox bars — this gives a real Y axis + gridlines,
// baseline-aligned bars (the old flex column top-aligned short bars, leaving
// <1min "floating"), and thousand-separated counts for free. A Divider splits
// the KPI row from the chart so the two rows no longer read as one grid.

import { Card, Col, Divider, Empty, Row, Segmented, Statistic } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
	type DurationDistributionResponse,
	dashboardApi,
} from "../../api/dashboard";
import { extractApiErrorMessage } from "../../lib/apiError";
import { formatDurationMs } from "../../pages/presets/durations";
import { LazyECharts } from "../analytics/LazyECharts";

const dashSurface = {
	cardBorder: "1px solid rgba(148, 163, 184, 0.45)",
	cardRadius: 12,
	cardShadow:
		"0 1px 2px rgba(15, 23, 42, 0.05), 0 2px 8px rgba(15, 23, 42, 0.04)",
} as const;

const cardHeaderBar = {
	borderBottom: "1px solid rgba(226, 232, 240, 0.95)",
	minHeight: 44,
	fontWeight: 600,
};

// Same segmented values the Segmented antd control accepts. `""` = 全部.
type AssetTypeSegmentValue = "" | "raw_mcap" | "segment" | "clip";

const SEGMENTED_OPTIONS: Array<{
	label: string;
	value: AssetTypeSegmentValue;
}> = [
	{ label: "全部", value: "" },
	{ label: "raw_mcap", value: "raw_mcap" },
	{ label: "segment", value: "segment" },
	{ label: "clip", value: "clip" },
];

function compactCount(value: number): string {
	if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
	if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
	return value.toString();
}

// CYB-4323: ECharts bar option for the duration histogram. Category X axis
// keeps the canonical 5 buckets; value Y axis + dashed gridlines restore the
// height reference the flexbox version lacked; bars are baseline-aligned so a
// short bucket (e.g. <1min) never floats.
function buildHistogramOption(
	buckets: Array<{ label: string; count: number }>,
) {
	return {
		grid: { left: 56, right: 24, top: 28, bottom: 32 },
		tooltip: {
			trigger: "axis" as const,
			axisPointer: { type: "shadow" as const },
			backgroundColor: "rgba(15, 23, 42, 0.92)",
			borderWidth: 0,
			textStyle: { color: "#f8fafc", fontSize: 12 },
			formatter: (params: unknown) => {
				const arr = Array.isArray(params) ? params : [params];
				const p = arr[0] as { name?: string; value?: number };
				return `${p.name ?? ""}: <b>${(p.value ?? 0).toLocaleString()}</b> 个资产`;
			},
		},
		xAxis: {
			type: "category" as const,
			data: buckets.map((b) => b.label),
			axisLine: { lineStyle: { color: "#cbd5e1" } },
			axisTick: { show: false },
			axisLabel: { fontSize: 12, color: "#475569" },
		},
		yAxis: {
			type: "value" as const,
			splitLine: { lineStyle: { type: "dashed" as const, color: "#f1f5f9" } },
			axisLabel: { fontSize: 11, color: "#64748b", formatter: compactCount },
		},
		series: [
			{
				type: "bar" as const,
				data: buckets.map((b) => b.count),
				barMaxWidth: 56,
				itemStyle: {
					color: {
						type: "linear" as const,
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
				label: {
					show: true,
					position: "top" as const,
					fontSize: 12,
					color: "#64748b",
					formatter: (p: { value?: number }) => (p.value ?? 0).toLocaleString(),
				},
			},
		],
	};
}

export default function DurationDistributionCard() {
	const [assetType, setAssetType] = useState<AssetTypeSegmentValue>("");
	const [data, setData] = useState<DurationDistributionResponse | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		let cancelled = false;
		setLoading(true);
		setError(null);
		dashboardApi
			.durationDistribution(assetType || undefined)
			.then((resp) => {
				if (cancelled) return;
				setData(resp);
			})
			.catch((err) => {
				if (cancelled) return;
				setError(extractApiErrorMessage(err, "数据时长分布加载失败"));
				setData(null);
			})
			.finally(() => {
				if (!cancelled) setLoading(false);
			});
		return () => {
			cancelled = true;
		};
	}, [assetType]);

	const histogramOption = useMemo(
		() =>
			buildHistogramOption(
				(data?.buckets ?? []).map((b) => ({ label: b.label, count: b.count })),
			),
		[data],
	);

	const totalAssets = data?.total_assets ?? 0;
	const isEmpty = !loading && !error && data !== null && totalAssets === 0;

	return (
		<Card
			variant="borderless"
			title="数据时长分布"
			size="small"
			extra={
				<Segmented<AssetTypeSegmentValue>
					options={SEGMENTED_OPTIONS}
					value={assetType}
					onChange={(v) => setAssetType(v)}
					data-testid="duration-distribution-segmented"
				/>
			}
			style={{
				flex: 1,
				borderRadius: dashSurface.cardRadius,
				border: dashSurface.cardBorder,
				boxShadow: dashSurface.cardShadow,
				background: "var(--color-bg-container, #fff)",
			}}
			styles={{
				header: cardHeaderBar,
				body: { padding: 16 },
			}}
			loading={loading && data === null}
			data-testid="duration-distribution-card"
		>
			{error ? (
				<Empty description={error} />
			) : isEmpty ? (
				<Empty description="所选类型下无资产" />
			) : (
				<>
					{/* CYB-4323: KPI summary above, chart below, split by a Divider so
					    the two rows no longer read as one grid (fixes the false
					    KPI↔bar column alignment). */}
					{data && (
						<Row gutter={[16, 8]}>
							<Col xs={12} sm={8} lg={4}>
								<Statistic
									title="总资产"
									value={data.total_assets.toLocaleString()}
								/>
							</Col>
							<Col xs={12} sm={8} lg={5}>
								<Statistic
									title="总时长"
									value={formatDurationMs(data.total_ms)}
								/>
							</Col>
							<Col xs={12} sm={8} lg={5}>
								<Statistic
									title="均值"
									value={formatDurationMs(data.mean_ms)}
								/>
							</Col>
							<Col xs={12} sm={8} lg={5}>
								<Statistic title="P50" value={formatDurationMs(data.p50_ms)} />
							</Col>
							<Col xs={12} sm={8} lg={5}>
								<Statistic title="P90" value={formatDurationMs(data.p90_ms)} />
							</Col>
						</Row>
					)}
					<Divider style={{ margin: "16px 0" }} />
					<LazyECharts
						option={histogramOption}
						style={{ height: 280, width: "100%" }}
					/>
				</>
			)}
		</Card>
	);
}
