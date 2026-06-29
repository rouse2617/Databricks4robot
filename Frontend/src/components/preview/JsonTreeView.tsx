import { useState, useCallback } from "react";
import { CaretRightOutlined, CaretDownOutlined } from "@ant-design/icons";

interface JsonTreeViewProps {
  data: unknown;
  /** Initial expand depth (default 2) */
  maxDepth?: number;
  /** Root label */
  rootLabel?: string;
  style?: React.CSSProperties;
}

/** Collapsible JSON tree viewer for raw message inspection. */
export default function JsonTreeView({ data, maxDepth = 2, rootLabel = "root", style }: JsonTreeViewProps) {
  return (
    <div
      style={{
        fontFamily: "var(--font-mono)",
        fontSize: 11,
        lineHeight: 1.6,
        overflow: "auto",
        maxHeight: 400,
        background: "var(--gray-900)",
        color: "rgba(255,255,255,0.85)",
        borderRadius: "var(--radius-md)",
        padding: "8px 0",
        ...style,
      }}
    >
      <JsonNode label={rootLabel} value={data} depth={0} maxDepth={maxDepth} />
    </div>
  );
}

function JsonNode({ label, value, depth, maxDepth }: { label: string; value: unknown; depth: number; maxDepth: number }) {
  const [expanded, setExpanded] = useState(depth < maxDepth);

  const toggle = useCallback(() => setExpanded((v) => !v), []);

  const indent = depth * 16 + 8;

  if (value === null) {
    return <div style={{ paddingLeft: indent, color: "var(--gray-400)" }}>{label}: <span style={{ color: "#ff6b6b" }}>null</span></div>;
  }

  if (typeof value === "boolean") {
    return <div style={{ paddingLeft: indent }}>{label}: <span style={{ color: "#ffd43b" }}>{String(value)}</span></div>;
  }

  if (typeof value === "number") {
    const numStr = Number.isInteger(value) ? String(value) : value.toFixed(6);
    return <div style={{ paddingLeft: indent }}>{label}: <span style={{ color: "#51cf66" }}>{numStr}</span></div>;
  }

  if (typeof value === "string") {
    const display = value.length > 120 ? `"${value.slice(0, 120)}…"` : `"${value}"`;
    return <div style={{ paddingLeft: indent }}>
      {label}: <span style={{ color: "#00b4d8" }}>{display}</span>
      {value.length > 120 && <span style={{ color: "var(--gray-400)", marginLeft: 4 }}>({value.length} chars)</span>}
    </div>;
  }

  if (Array.isArray(value)) {
    return (
      <div>
        <div style={{ paddingLeft: indent, cursor: "pointer", userSelect: "none" }} onClick={toggle}>
          {expanded ? <CaretDownOutlined style={{ fontSize: 10, marginRight: 4 }} /> : <CaretRightOutlined style={{ fontSize: 10, marginRight: 4 }} />}
          <span style={{ color: "var(--gray-400)" }}>{label}</span>
          <span style={{ color: "var(--gray-500)", marginLeft: 4 }}>[{value.length}]</span>
        </div>
        {expanded && value.map((item, i) => (
          <JsonNode key={i} label={`[${i}]`} value={item} depth={depth + 1} maxDepth={maxDepth} />
        ))}
      </div>
    );
  }

  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    return (
      <div>
        <div style={{ paddingLeft: indent, cursor: "pointer", userSelect: "none" }} onClick={toggle}>
          {expanded ? <CaretDownOutlined style={{ fontSize: 10, marginRight: 4 }} /> : <CaretRightOutlined style={{ fontSize: 10, marginRight: 4 }} />}
          <span style={{ color: "var(--gray-400)" }}>{label}</span>
          <span style={{ color: "var(--gray-500)", marginLeft: 4 }}>{`{${entries.length}}`}</span>
        </div>
        {expanded && entries.map(([k, v]) => (
          <JsonNode key={k} label={k} value={v} depth={depth + 1} maxDepth={maxDepth} />
        ))}
      </div>
    );
  }

  return <div style={{ paddingLeft: indent }}>{label}: {String(value)}</div>;
}
