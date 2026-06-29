import { useEffect, useRef } from "react";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";

interface TimeSeriesChartProps {
  /** Chart title */
  title?: string;
  /** Series data: [timestamps, values1, values2, ...] */
  data: [number[], ...number[][]];
  /** Series names (one per value column) */
  series: string[];
  /** Line colors per series */
  colors?: string[];
  /** Height in CSS px (default: 200) */
  height?: number;
}

const DEFAULT_COLORS = ["#00b4d8", "#ff6b6b", "#51cf66", "#ffd43b"];

export default function TimeSeriesChart({
  title,
  data,
  series,
  colors = DEFAULT_COLORS,
  height = 200,
}: TimeSeriesChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<uPlot | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    if (data[0].length === 0) return;

    const opts: uPlot.Options = {
      width: containerRef.current.clientWidth || 400,
      height,
      cursor: { show: true },
      legend: { show: true, live: false },
      axes: [
        {
          stroke: "rgba(255,255,255,0.4)",
          grid: { stroke: "rgba(255,255,255,0.06)" },
          ticks: { stroke: "rgba(255,255,255,0.1)" },
        },
        {
          stroke: "rgba(255,255,255,0.4)",
          grid: { stroke: "rgba(255,255,255,0.06)" },
          ticks: { stroke: "rgba(255,255,255,0.1)" },
        },
      ],
      scales: {
        x: { time: true },
        y: { auto: true },
      },
      series: [
        {
          label: "Time",
          value: (_self, raw) => {
            if (!raw) return "-";
            return new Date(raw * 1000).toISOString().slice(11, 23);
          },
        },
        ...series.map((name, i) => ({
          label: name,
          stroke: colors[i % colors.length],
          width: 1.5,
          points: { show: data[0].length < 200 },
        })),
      ],
    };

    const chart = new uPlot(opts, data, containerRef.current);
    chartRef.current = chart;

    return () => {
      chart.destroy();
      chartRef.current = null;
    };
  }, [data, series, colors, height]);

  return (
    <div
      style={{
        background: "var(--gray-900)",
        borderRadius: "var(--radius-md)",
        overflow: "hidden",
      }}
    >
      {title && (
        <div
          style={{
            padding: "6px 12px",
            fontSize: "var(--font-size-xs)",
            color: "var(--gray-400)",
            borderBottom: "1px solid rgba(255,255,255,0.06)",
          }}
        >
          {title}
        </div>
      )}
      <div ref={containerRef} />
    </div>
  );
}

/**
 * Generate mock time series data for testing.
 * Returns [timestamps, values] arrays.
 */
export function generateMockData(
  points: number,
  freq: number = 0.5,
): [number[], number[]] {
  const t: number[] = [];
  const v: number[] = [];
  const start = Date.now() / 1000 - points;
  for (let i = 0; i < points; i++) {
    t.push(start + i);
    v.push(Math.sin(i * freq) + Math.random() * 0.3);
  }
  return [t, v];
}
