import { useRef, useState, useCallback } from "react";
import { Tooltip } from "antd";

export interface StateSegment {
  startSec: number;
  endSec: number;
  label: string;
  color: string;
}

interface StateBandProps {
  segments: StateSegment[];
  durationSec: number;
  height?: number;
}

/** Mock action states for embodied-AI egocentric data demos. */
export function generateMockStates(durationSec: number): StateSegment[] {
  if (durationSec <= 0) return [];
  const states: StateSegment[] = [];
  let t = 0;
  const actions = [
    { label: "空闲", color: "#6b7280", dur: 0.5 },
    { label: "观察", color: "#3b82f6", dur: 1.2 },
    { label: "伸手", color: "#f59e0b", dur: 0.8 },
    { label: "抓握", color: "#ef4444", dur: 0.6 },
    { label: "移动", color: "#8b5cf6", dur: 1.5 },
    { label: "操作", color: "#10b981", dur: 2.0 },
    { label: "释放", color: "#ec4899", dur: 0.4 },
    { label: "收回", color: "#06b6d4", dur: 1.0 },
    { label: "评估", color: "#84cc16", dur: 0.7 },
  ];
  while (t < durationSec) {
    const a = actions[Math.floor(Math.random() * actions.length)];
    const dur = Math.min(a.dur + Math.random() * 0.5, durationSec - t);
    if (dur < 0.1) break;
    states.push({ startSec: t, endSec: t + dur, label: a.label, color: a.color });
    t += dur;
  }
  return states;
}

/**
 * Colored state-transition band rendered above the timeline.
 * Each segment width is proportional to its duration within the total window.
 * Hovering a segment shows its label + time range.
 */
export default function StateBand({ segments, durationSec, height = 18 }: StateBandProps) {
  const bandRef = useRef<HTMLDivElement>(null);
  const [tooltip, setTooltip] = useState<{ x: number; label: string; start: number; end: number } | null>(null);

  const handleMouseMove = useCallback(
    (e: React.MouseEvent, seg: StateSegment) => {
      const rect = bandRef.current?.getBoundingClientRect();
      if (!rect) return;
      setTooltip({ x: e.clientX, label: seg.label, start: seg.startSec, end: seg.endSec });
    },
    [],
  );

  if (durationSec <= 0 || segments.length === 0) return null;

  return (
    <div
      ref={bandRef}
      style={{
        position: "relative",
        height,
        display: "flex",
        borderRadius: 4,
        overflow: "hidden",
        cursor: "default",
        marginBottom: 2,
      }}
    >
      {segments.map((seg, i) => {
        const pct = ((seg.endSec - seg.startSec) / durationSec) * 100;
        return (
          <Tooltip
            key={i}
            title={`${seg.label}: ${seg.startSec.toFixed(1)}s – ${seg.endSec.toFixed(1)}s`}
          >
            <div
              style={{
                width: `${pct}%`,
                height: "100%",
                background: seg.color,
                opacity: 0.85,
                transition: "opacity 0.15s",
                flexShrink: 0,
              }}
              onMouseEnter={(e) => handleMouseMove(e, seg)}
              onMouseMove={(e) => handleMouseMove(e, seg)}
              onMouseLeave={() => setTooltip(null)}
            />
          </Tooltip>
        );
      })}

      {/* Floating label on hover */}
      {tooltip && (
        <div
          style={{
            position: "fixed",
            left: tooltip.x - 40,
            top: 92,
            background: "rgba(0,0,0,0.8)",
            color: "#fff",
            padding: "2px 10px",
            borderRadius: 12,
            fontSize: 11,
            fontFamily: "var(--font-mono)",
            pointerEvents: "none",
            zIndex: 1001,
            whiteSpace: "nowrap",
          }}
        >
          {tooltip.label}
        </div>
      )}
    </div>
  );
}
