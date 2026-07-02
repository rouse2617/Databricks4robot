import { lazy, Suspense, useEffect, useRef, useState } from "react";

const ReactECharts = lazy(() => import("echarts-for-react"));

type IdleWindow = Window & {
	requestIdleCallback?: (
		callback: () => void,
		options?: { timeout?: number },
	) => number;
	cancelIdleCallback?: (id: number) => void;
};

function scheduleChartImport(callback: () => void, delayMs: number) {
	const idleWindow = window as IdleWindow;
	let idleId: number | null = null;
	const timeoutId = window.setTimeout(() => {
		if (typeof idleWindow.requestIdleCallback === "function") {
			idleId = idleWindow.requestIdleCallback(callback, { timeout: 2000 });
			return;
		}
		callback();
	}, delayMs);
	return () => {
		window.clearTimeout(timeoutId);
		if (idleId !== null) {
			idleWindow.cancelIdleCallback?.(idleId);
		}
	};
}

export function LazyECharts({
	option,
	style,
	defer = true,
	delayMs = 3000,
}: {
	option: unknown;
	style?: React.CSSProperties;
	defer?: boolean;
	delayMs?: number;
}) {
	const containerRef = useRef<HTMLDivElement | null>(null);
	const [enabled, setEnabled] = useState(!defer);

	useEffect(() => {
		if (!defer || enabled) return;
		const node = containerRef.current;
		let cancelScheduled: (() => void) | null = null;

		const enable = () => {
			cancelScheduled?.();
			cancelScheduled = scheduleChartImport(() => setEnabled(true), delayMs);
		};

		if (node && typeof IntersectionObserver !== "undefined") {
			const observer = new IntersectionObserver(
				(entries) => {
					if (entries.some((entry) => entry.isIntersecting)) {
						observer.disconnect();
						enable();
					}
				},
				{ rootMargin: "0px" },
			);
			observer.observe(node);
			return () => {
				observer.disconnect();
				cancelScheduled?.();
			};
		}

		enable();
		return () => cancelScheduled?.();
	}, [defer, delayMs, enabled]);

	const placeholder = (
		<div
			ref={containerRef}
			style={{
				...style,
				display: "flex",
				alignItems: "center",
				justifyContent: "center",
				minHeight:
					typeof style?.height === "number"
						? style.height
						: (style?.height ?? 240),
				color: "#94a3b8",
				fontSize: 12,
				background:
					"linear-gradient(180deg, rgba(248,250,252,0.72), rgba(255,255,255,0.92))",
			}}
			aria-busy="true"
		>
			加载图表...
		</div>
	);

	if (!enabled) {
		return placeholder;
	}

	return (
		<Suspense fallback={placeholder}>
			<ReactECharts option={option} style={style} />
		</Suspense>
	);
}
