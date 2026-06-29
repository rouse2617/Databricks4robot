import { useRef, useMemo, useCallback, useState, useEffect } from "react";
import { Input, Tag, Typography } from "antd";

const { Text } = Typography;

interface LogEntry {
  timestamp: number;
  level: "debug" | "info" | "warn" | "error";
  message: string;
  topic?: string;
}

interface LogPanelProps {
  logs?: LogEntry[];
  maxVisible?: number;
  filter?: string;
  /** Current playback time in seconds — matching log entries are highlighted */
  currentTimeSec?: number;
}

const LEVEL_COLORS: Record<string, string> = {
  debug: "default",
  info: "blue",
  warn: "orange",
  error: "red",
};

const ROW_HEIGHT = 22;

/**
 * Virtual-scrolling log panel for high-volume log streams (e.g. /rosout).
 * Renders only visible rows based on scroll position.
 */
export default function LogPanel({
  logs = [],
  maxVisible = 100_000,
  filter: _filter = "",
  currentTimeSec = 0,
}: LogPanelProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [containerHeight, setContainerHeight] = useState(400);
  const [search, setSearch] = useState("");

  const filtered = useMemo(() => {
    let result = logs;
    if (search) {
      const q = search.toLowerCase();
      result = result.filter(
        (l) =>
          l.message.toLowerCase().includes(q) ||
          l.topic?.toLowerCase().includes(q),
      );
    }
    return result.slice(-maxVisible);
  }, [logs, search, maxVisible]);

  const totalHeight = filtered.length * ROW_HEIGHT;
  const startIdx = Math.floor(scrollTop / ROW_HEIGHT);
  const visibleCount = Math.ceil(containerHeight / ROW_HEIGHT) + 2;
  const visibleLogs = filtered.slice(
    startIdx,
    Math.min(startIdx + visibleCount, filtered.length),
  );

  const onScroll = useCallback(() => {
    if (containerRef.current) {
      setScrollTop(containerRef.current.scrollTop);
    }
  }, []);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => setContainerHeight(el.clientHeight));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        background: "var(--gray-900)",
        borderRadius: "var(--radius-md)",
        overflow: "hidden",
        height: "100%",
        minHeight: 200,
      }}
    >
      {/* Header */}
      <div
        style={{
          padding: "6px 12px",
          display: "flex",
          alignItems: "center",
          gap: 8,
          borderBottom: "1px solid rgba(255,255,255,0.06)",
          flexShrink: 0,
        }}
      >
        <Text
          style={{
            fontSize: "var(--font-size-xs)",
            color: "var(--gray-400)",
            whiteSpace: "nowrap",
          }}
        >
          日志 ({filtered.length})
        </Text>
        <Input.Search
          size="small"
          placeholder="过滤..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ maxWidth: 200 }}
        />
      </div>

      {/* Virtual scroll container */}
      <div
        ref={containerRef}
        onScroll={onScroll}
        style={{
          flex: 1,
          overflow: "auto",
          fontFamily: "var(--font-mono)",
          fontSize: 11,
          lineHeight: `${ROW_HEIGHT}px`,
        }}
      >
        <div style={{ height: totalHeight, position: "relative" }}>
          {visibleLogs.map((log, i) => {
            const logTs = log.timestamp / 1000; // ms → sec
            const isTimeMatch = currentTimeSec > 0 && Math.abs(logTs - currentTimeSec) < 0.5;
            return (
            <div
              key={startIdx + i}
              style={{
                position: "absolute",
                top: (startIdx + i) * ROW_HEIGHT,
                left: 0,
                right: 0,
                height: ROW_HEIGHT,
                display: "flex",
                alignItems: "center",
                gap: 6,
                padding: "0 8px",
                borderBottom: "1px solid rgba(255,255,255,0.03)",
                borderLeft: isTimeMatch ? "3px solid #00b4d8" : "3px solid transparent",
                background: isTimeMatch
                  ? "rgba(0, 180, 216, 0.12)"
                  : log.level === "error"
                    ? "rgba(255,0,0,0.05)"
                    : log.level === "warn"
                      ? "rgba(255,200,0,0.03)"
                      : "transparent",
              }}
            >
              <Tag
                color={LEVEL_COLORS[log.level] || "default"}
                style={{ fontSize: 9, lineHeight: "16px", padding: "0 4px", margin: 0 }}
              >
                {log.level}
              </Tag>
              <span style={{ color: "rgba(255,255,255,0.35)", whiteSpace: "nowrap" }}>
                {new Date(log.timestamp).toISOString().slice(11, 23)}
              </span>
              <span style={{ color: "rgba(255,255,255,0.8)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                {log.message}
              </span>
            </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// Generate mock logs for testing
export function generateMockLogs(count: number): LogEntry[] {
  const levels: LogEntry["level"][] = ["debug", "info", "warn", "error"];
  const msgs = [
    "Node started",
    "Subscribed to /camera/front",
    "Received frame 1024",
    "Processing batch...",
    "Publishing result to /tracking/output",
    "Transform lookup failed: no transform between map and odom",
    "Battery voltage: 11.8V",
    "GPS fix acquired",
    "IMU calibration complete",
    "Connection lost to /cmd_vel",
    "Reconnecting...",
    "Shutdown complete",
  ];
  const base = Date.now() - 3600_000;
  return Array.from({ length: count }, (_, i) => ({
    timestamp: base + i * 100,
    level: levels[Math.floor(Math.random() * levels.length)],
    message: msgs[Math.floor(Math.random() * msgs.length)],
    topic: ["/rosout", "/diagnostics", "/tf"][Math.floor(Math.random() * 3)],
  }));
}
