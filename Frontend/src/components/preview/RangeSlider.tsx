import { useRef, useCallback, useState } from "react";

interface RangeSliderProps {
  currentTime: number;
  durationSec: number;
  /** Highlighted range (optional) */
  range?: { startSec: number; endSec: number } | null;
  /** Called when user drags the playhead (seek) */
  onSeek: (time: number) => void;
  /** Called when user finishes dragging a range handle */
  onRangeChange?: (range: { startSec: number; endSec: number }) => void;
}

type DragTarget = "playhead" | "rangeStart" | "rangeEnd" | null;

function fmtTooltip(sec: number): string {
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = Math.floor(sec % 60);
  if (h > 0) return `${h}:${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

/**
 * Custom range slider with draggable playhead + range handles.
 * Fires onSeek while dragging playhead, onRangeChange on handle release.
 */
export default function RangeSlider({ currentTime, durationSec, range, onSeek, onRangeChange }: RangeSliderProps) {
  const barRef = useRef<HTMLDivElement>(null);
  const [dragTarget, setDragTarget] = useState<DragTarget>(null);
  const [hoverSec, setHoverSec] = useState<number | null>(null);
  const [hoverX, setHoverX] = useState(0);
  const dragRef = useRef<{ target: DragTarget; startX: number; startSec: number; startRange?: { startSec: number; endSec: number } }>({ target: null, startX: 0, startSec: 0 });
  // Stable refs for callbacks so we don't need to re-register listeners
  const onSeekRef = useRef(onSeek);
  const onRangeChangeRef = useRef(onRangeChange);
  onSeekRef.current = onSeek;
  onRangeChangeRef.current = onRangeChange;

  const fracToSec = useCallback((clientX: number) => {
    if (!barRef.current) return 0;
    const rect = barRef.current.getBoundingClientRect();
    return Math.max(0, Math.min(durationSec, ((clientX - rect.left) / rect.width) * durationSec));
  }, [durationSec]);

  const onMouseDown = useCallback((e: React.MouseEvent, target: DragTarget) => {
    e.preventDefault();
    const sec = fracToSec(e.clientX);
    dragRef.current = { target, startX: e.clientX, startSec: sec, startRange: range ? { ...range } : undefined };
    setDragTarget(target);

    // Register mousemove/mouseup directly (not via useEffect) so they're
    // active immediately on the NEXT tick — no re-render gap.
    const onMouseMove = (e: MouseEvent) => {
      const d = dragRef.current;
      const s = fracToSec(e.clientX);
      switch (d.target) {
        case "playhead":
          onSeekRef.current(s);
          break;
        case "rangeStart": {
          if (!d.startRange) break;
          const end = d.startRange.endSec;
          const clamped = Math.min(s, end - 0.1);
          onRangeChangeRef.current?.({ startSec: clamped, endSec: end });
          break;
        }
        case "rangeEnd": {
          if (!d.startRange) break;
          const start = d.startRange.startSec;
          const clamped = Math.max(s, start + 0.1);
          onRangeChangeRef.current?.({ startSec: start, endSec: clamped });
          break;
        }
      }
    };
    const onMouseUp = () => {
      dragRef.current = { target: null, startX: 0, startSec: 0 };
      setDragTarget(null);
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };

    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
  }, [fracToSec, range]);

  const playheadFrac = durationSec > 0 ? (currentTime / durationSec) * 100 : 0;
  const hasRange = range && durationSec > 0;
  const rangeStartFrac = hasRange ? (range.startSec / durationSec) * 100 : 0;
  const rangeEndFrac = hasRange ? (range.endSec / durationSec) * 100 : 0;

  // Handle size in pixels
  const handleSize = 20;

  return (
    <div
      ref={barRef}
      style={{
        flex: 1, height: 32, position: "relative", cursor: "pointer",
        userSelect: "none", margin: "0 4px", touchAction: "none",
      }}
      onMouseDown={(e) => {
        if (!dragRef.current.target) onSeekRef.current(fracToSec(e.clientX));
      }}
      onMouseMove={(e) => {
        if (!dragTarget) {
          setHoverSec(fracToSec(e.clientX));
          setHoverX(e.clientX - (barRef.current?.getBoundingClientRect().left || 0));
        }
      }}
      onMouseLeave={() => setHoverSec(null)}
    >
      {/* Background track */}
      <div style={{
        position: "absolute", left: 0, right: 0, top: "50%", height: 4, marginTop: -2,
        background: dragTarget ? "var(--gray-300)" : "var(--gray-200)",
        borderRadius: 2, transition: dragTarget ? "none" : "background 0.15s",
      }} />

      {/* Hover tooltip */}
      {hoverSec !== null && !dragTarget && (
        <div style={{
          position: "absolute", bottom: 28,
          left: `calc(${hoverX}px - 30px)`,
          background: "rgba(0,0,0,0.75)", color: "#fff",
          padding: "2px 6px", borderRadius: 4,
          fontSize: 10, fontFamily: "var(--font-mono)",
          whiteSpace: "nowrap", pointerEvents: "none", zIndex: 10,
        }}>
          {fmtTooltip(hoverSec)}
        </div>
      )}

      {/* Range highlight */}
      {hasRange && (
        <div style={{
          position: "absolute",
          left: `${rangeStartFrac}%`, width: `${rangeEndFrac - rangeStartFrac}%`,
          top: "50%", height: 4, marginTop: -2,
          background: "rgba(255, 68, 68, 0.35)", borderRadius: 2, pointerEvents: "none",
          transition: dragTarget ? "none" : "left 0.1s, width 0.1s",
        }} />
      )}

      {/* Range start handle — cyan hollow square (cursor) */}
      {hasRange && (
        <div
          onMouseDown={(e) => onMouseDown(e, "rangeStart")}
          style={{
            position: "absolute", left: `${rangeStartFrac}%`, top: "50%",
            width: handleSize, height: handleSize,
            marginLeft: -handleSize / 2, marginTop: -handleSize / 2,
            border: "2px solid #00bcd4",
            background: dragTarget === "rangeStart" ? "#00bcd4" : "#fff",
            borderRadius: "50%",
            cursor: "ew-resize", zIndex: 2, boxSizing: "border-box",
            transition: dragTarget === "rangeStart" ? "none" : "transform 0.1s, background 0.1s",
          }}
        />
      )}

      {/* Range end handle */}
      {hasRange && (
        <div
          onMouseDown={(e) => onMouseDown(e, "rangeEnd")}
          style={{
            position: "absolute", left: `${rangeEndFrac}%`, top: "50%",
            width: handleSize, height: handleSize,
            marginLeft: -handleSize / 2, marginTop: -handleSize / 2,
            border: "2px solid var(--color-accent)",
            background: dragTarget === "rangeEnd" ? "var(--color-accent)" : "#fff",
            borderRadius: "50%",
            cursor: "ew-resize", zIndex: 2, boxSizing: "border-box",
            transition: dragTarget === "rangeEnd" ? "none" : "transform 0.1s, background 0.1s",
          }}
        />
      )}

      {/* Playhead */}
      <div
        onMouseDown={(e) => onMouseDown(e, "playhead")}
        style={{
          position: "absolute", left: `${playheadFrac}%`, top: "50%",
          width: handleSize + 2, height: handleSize + 2,
          marginLeft: -(handleSize + 2) / 2, marginTop: -(handleSize + 2) / 2,
          border: "3px solid #fff",
          background: dragTarget === "playhead" ? "#cc0000" : "#ff4444",
          borderRadius: "50%",
          cursor: "grab", zIndex: 3, boxSizing: "border-box",
          boxShadow: dragTarget === "playhead"
            ? "0 0 6px rgba(255,68,68,0.6)"
            : "0 1px 3px rgba(0,0,0,0.3)",
          transition: dragTarget === "playhead" ? "none" : "box-shadow 0.15s",
        }}
      />
    </div>
  );
}
