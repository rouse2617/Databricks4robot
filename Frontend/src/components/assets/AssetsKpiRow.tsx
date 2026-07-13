// CYB-3382 #3 — 顶部 KPI 概览卡片
// 4 张卡:总数 / ready:created 比例 / 30d 算法失败 / 30d 即将过期
// 数据全部复用 useAssetsDiscovery 已 fetch 的 total + aggregations,
// 不调用新 API。某个 metric 数据缺失时显示 `—`,不 crash,不阻塞列表。

import { Card, Col, Row, Statistic } from "antd";
import { useMemo } from "react";

type AggBucket = { key: string; doc_count: number };
type Aggregations = Record<string, AggBucket[]>;

export interface AssetsKpiRowProps {
	total: number | undefined;
	aggregations: Aggregations | undefined;
	loading?: boolean;
}

function bucketCount(
	agg: Aggregations | undefined,
	field: string,
	key: string,
): number | undefined {
	if (!agg) return undefined;
	const buckets = agg[field];
	if (!buckets) return undefined;
	const hit = buckets.find((b) => b.key === key);
	return hit?.doc_count;
}

function displayValue(v: number | undefined): string {
	if (v === undefined || Number.isNaN(v)) return "—";
	return String(v);
}

export default function AssetsKpiRow({
	total,
	aggregations,
	loading,
}: AssetsKpiRowProps) {
	const ratio = useMemo(() => {
		const ready = bucketCount(aggregations, "lifecycle_state", "ready");
		const created = bucketCount(aggregations, "lifecycle_state", "created");
		if (ready === undefined || created === undefined) return "—";
		if (created === 0) return `${ready}:0`;
		return `${ready}:${created}`;
	}, [aggregations]);

	const algoFailed = useMemo(
		() => bucketCount(aggregations, "algo_status", "failed"),
		[aggregations],
	);

	// 「30 天即将过期」目前后端 aggregations 没有直接暴露 expire_at 分桶;
	// 短期方案:若 aggregations 有 `expiring_30d` 字段(未来加)则用它,
	// 否则显示 `—`。避免虚构数字。
	const expiring30d = useMemo(() => {
		const buckets = aggregations?.expiring_30d;
		if (!buckets) return undefined;
		return buckets.reduce((sum, b) => sum + b.doc_count, 0);
	}, [aggregations]);

	return (
		<Row gutter={12} style={{ marginBottom: 12 }}>
			<Col xs={12} sm={6}>
				<Card size="small" bordered>
					<Statistic
						title="总数"
						value={displayValue(total)}
						loading={loading}
					/>
				</Card>
			</Col>
			<Col xs={12} sm={6}>
				<Card size="small" bordered>
					<Statistic title="ready : created" value={ratio} loading={loading} />
				</Card>
			</Col>
			<Col xs={12} sm={6}>
				<Card size="small" bordered>
					<Statistic
						title="算法失败"
						value={displayValue(algoFailed)}
						loading={loading}
					/>
				</Card>
			</Col>
			<Col xs={12} sm={6}>
				<Card size="small" bordered>
					<Statistic
						title="30 天即将过期"
						value={displayValue(expiring30d)}
						loading={loading}
					/>
				</Card>
			</Col>
		</Row>
	);
}
