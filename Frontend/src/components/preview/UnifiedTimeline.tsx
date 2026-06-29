import { useState } from "react";
import { Button, Input, Select, Tooltip } from "antd";
import RangeSlider from "./RangeSlider";
import {
  CaretRightOutlined, PauseOutlined,
  StepBackwardOutlined, StepForwardOutlined,
  FullscreenOutlined, ScissorOutlined, ColumnWidthOutlined,
} from "@ant-design/icons";

export interface RangeHighlight {
  startSec: number;
  endSec: number;
  color?: string;
  label?: string;
}

interface UnifiedTimelineProps {
  currentTime: number;
  durationSec: number;
  playing: boolean;
  playbackRate: number;
  coverMode: boolean;
  onToggleCover: () => void;
  startTimestampNs?: number;
  range?: RangeHighlight;
  onRangeChange?: (range: { startSec: number; endSec: number }) => void;
  onPlayPause: () => void;
  onSeek: (time: number) => void;
  onChangeRate: (rate: number) => void;
  onFullscreen?: () => void;
}

function formatTime(s: number): string {
  return `${Math.floor(s / 60).toString().padStart(2, "0")}:${Math.floor(s % 60).toString().padStart(2, "0")}`;
}

function formatTimestamp(ns: number): string {
  return new Date(ns / 1e6).toISOString().slice(11, 23);
}

const RATES = [0.25, 0.5, 1, 2, 4, 8];

/** Click-to-edit timestamp. Type "MM:SS" or bare seconds. */
function TimestampInput({ currentTime, durationSec, onSeek }: {
  currentTime: number; durationSec: number; onSeek: (t: number) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState("");
  const submit = () => {
    setEditing(false);
    const t = text.trim();
    if (!t) return;
    let sec = 0;
    const m = t.match(/^(\d+):(\d+)$/);
    if (m) sec = parseInt(m[1]) * 60 + parseInt(m[2]);
    else sec = parseFloat(t);
    if (isFinite(sec) && sec >= 0) onSeek(Math.min(sec, durationSec));
  };
  if (editing) {
    return <Input size="small" value={text} onChange={e => setText(e.target.value)}
      onPressEnter={submit} onBlur={submit} autoFocus
      style={{ width: 100, fontFamily: "var(--font-mono)", fontSize: "var(--font-size-sm)" }} placeholder="MM:SS" />;
  }
  return (
    <span onClick={() => { setText(formatTime(currentTime)); setEditing(true); }}
      style={{ cursor: "pointer", fontSize: "var(--font-size-sm)", fontFamily: "var(--font-mono)", color: "var(--gray-700)", borderBottom: "1px dashed transparent" }}
      title="点击输入时间">
      {formatTime(currentTime)} / {formatTime(durationSec)}
    </span>
  );
}

export default function UnifiedTimeline({
  currentTime, durationSec, playing, playbackRate, coverMode,
  onToggleCover, startTimestampNs, range, onRangeChange,
  onPlayPause, onSeek, onChangeRate, onFullscreen,
}: UnifiedTimelineProps) {
  const currentNs = startTimestampNs ? startTimestampNs + currentTime * 1e9 : 0;

  return (
    <div style={{ borderTop: "1px solid var(--gray-200)", padding: "var(--space-3) var(--space-6)", background: "#fff", display: "flex", flexDirection: "column", gap: 8 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, width: "100%" }}>
        <Tooltip title="前一帧"><Button type="text" icon={<StepBackwardOutlined />} size="small" onClick={() => onSeek(Math.max(0, currentTime - 1 / 30))} /></Tooltip>
        <Button type="primary" shape="circle" icon={playing ? <PauseOutlined /> : <CaretRightOutlined />} onClick={onPlayPause} size="small" />
        <Tooltip title="后一帧"><Button type="text" icon={<StepForwardOutlined />} size="small" onClick={() => onSeek(Math.min(durationSec, currentTime + 1 / 30))} /></Tooltip>
        <Select value={playbackRate} onChange={onChangeRate} size="small" style={{ width: 58 }} options={RATES.map(r => ({ value: r, label: `${r}x` }))} />

        <div style={{ flex: 1, minWidth: 0 }}>
          <RangeSlider
            currentTime={currentTime}
            durationSec={durationSec}
            range={range ? { startSec: range.startSec, endSec: range.endSec } : null}
            onSeek={onSeek}
            onRangeChange={onRangeChange}
          />
        </div>

        <TimestampInput currentTime={currentTime} durationSec={durationSec} onSeek={onSeek} />

        {currentNs > 0 && (
          <span style={{ fontSize: "var(--font-size-xs)", fontFamily: "var(--font-mono)", color: "var(--gray-400)", whiteSpace: "nowrap", minWidth: 90 }}>
            {formatTimestamp(currentNs)}
          </span>
        )}

        <Tooltip title={coverMode ? "完整画面" : "裁剪满屏"}>
          <Button type="text" size="small"
            icon={coverMode ? <ColumnWidthOutlined style={{ color: "var(--gray-500)" }} /> : <ScissorOutlined style={{ color: "var(--color-primary)" }} />}
            onClick={onToggleCover} />
        </Tooltip>
        {onFullscreen && (
          <Tooltip title="全屏"><Button type="text" icon={<FullscreenOutlined />} size="small" onClick={onFullscreen} /></Tooltip>
        )}
      </div>
    </div>
  );
}
