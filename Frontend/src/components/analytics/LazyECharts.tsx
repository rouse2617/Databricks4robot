import { lazy, Suspense } from "react";

const ReactECharts = lazy(() => import("echarts-for-react"));

export function LazyECharts({
	option,
	style,
}: {
	option: unknown;
	style?: React.CSSProperties;
}) {
	return (
		<Suspense fallback={<div style={style} />}>
			<ReactECharts option={option} style={style} />
		</Suspense>
	);
}
