// CYB-4303: dashboard 数据时长分布 card.
//
// Replaces the two event cards ("事件分布" + "最新日事件类型分布") with a
// single asset-duration histogram + summary stats. Buckets always ship in
// the CYB-4294 canonical order — the backend fills any missing bucket with
// zero — so this component doesn't need to reason about gaps.
//
// The bar-chart style mirrors CYB-4294's BatchAssetLookup histogram (flexbox
// bars, count on top, label below) so single-asset and fleet-scope views
// speak the same visual language.

import { Card, Col, Empty, Row, Segmented, Statistic } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
	type DurationDistributionResponse,
	dashboardApi,
} from "../../api/dashboard";
import { extractApiErrorMessage } from "../../lib/apiError";
import { formatDurationMs } from "../../pages/presets/durations";

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

const SEGMENTED_OPTIONS: Array<{ label: string; value: AssetTypeSegmentValue }> =
	[
		{ label: "全部", value: "" },
		{ label: "raw_mcap", value: "raw_mcap" },
		{ label: "segment", value: "segment" },
		{ label: "clip", value: "clip" },
	];

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
				height: 200,
			}}
			aria-label="duration distribution histogram"
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
						<div
							style={{
								fontSize: 12,
								marginBottom: 4,
								color: "var(--color-text-secondary)",
								fontVariantNumeric: "tabular-nums",
							}}
						>
							{b.count.toLocaleString()}
						</div>
						<div
							style={{
								width: "100%",
								height: `${heightPct}%`,
								background: "var(--color-primary, #1677ff)",
								borderRadius: "4px 4px 0 0",
								// Non-zero counts always show a visible sliver so bars never
								// disappear behind the count label even when max is huge.
								minHeight: b.count > 0 ? 8 : 0,
							}}
						/>
						<div
							style={{
								marginTop: 6,
								fontSize: 12,
								color: "var(--color-text-secondary, #64748b)",
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

	const bars = useMemo(
		() =>
			(data?.buckets ?? []).map((b) => ({ label: b.label, count: b.count })),
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
					<HistogramBars buckets={bars} />
					{data && (
						<Row gutter={[16, 8]} style={{ marginTop: 20 }}>
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
				</>
			)}
		</Card>
	);
}
