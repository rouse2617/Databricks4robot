import { useEffect, useRef } from "react";

interface PreviewChannel {
  topic: string;
  label: string;
  width?: number;
  height?: number;
}

interface Marker {
  timestampSec: number;
  label?: string;
  color?: string;
}

interface VideoPanelProps {
  channel?: PreviewChannel;
  src: string;
  currentTime: number;
  playing: boolean;
  onTimeUpdate: (time: number) => void;
  objectFit?: "cover" | "contain";
  markers?: Marker[];
  durationSec?: number;
}

export default function VideoPanel({
  src,
  currentTime,
  playing,
  onTimeUpdate,
  objectFit = "cover",
  markers,
  durationSec = 0,
}: VideoPanelProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    if (playing) {
      v.play().catch(() => {});
    } else {
      v.pause();
    }
  }, [playing]);

  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    if (Math.abs(v.currentTime - currentTime) > 0.5) {
      v.currentTime = currentTime;
    }
  }, [currentTime]);

  // Draw canvas overlay: timeline + markers
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // Match canvas size to parent
    const parent = canvas.parentElement;
    if (parent) {
      canvas.width = parent.clientWidth;
      canvas.height = parent.clientHeight;
    }

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    if (!markers || markers.length === 0 || durationSec <= 0) return;

    const w = canvas.width;
    const h = canvas.height;

    // Draw a thin timeline at the bottom
    const timelineY = h - 16;
    const timelineH = 4;
    ctx.fillStyle = "rgba(255,255,255,0.15)";
    ctx.fillRect(8, timelineY, w - 16, timelineH);

    // Draw markers as vertical lines + labels
    for (const m of markers) {
      const x = 8 + (m.timestampSec / durationSec) * (w - 16);
      if (x < 8 || x > w - 8) continue;

      // Vertical line
      ctx.strokeStyle = m.color || "rgba(0, 200, 255, 0.7)";
      ctx.lineWidth = 1;
      ctx.setLineDash([3, 3]);
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, timelineY - 2);
      ctx.stroke();
      ctx.setLineDash([]);

      // Dot on timeline
      ctx.fillStyle = m.color || "rgba(0, 200, 255, 0.9)";
      ctx.beginPath();
      ctx.arc(x, timelineY + timelineH / 2, 3, 0, Math.PI * 2);
      ctx.fill();

      // Label above timeline
      if (m.label && x > 20) {
        ctx.fillStyle = "rgba(255,255,255,0.6)";
        ctx.font = "10px monospace";
        ctx.textAlign = "center";
        ctx.fillText(m.label, x, timelineY - 4);
      }
    }

    // Current position indicator (playhead)
    const playheadX = 8 + (currentTime / (durationSec || 1)) * (w - 16);
    ctx.fillStyle = "#ff4444";
    ctx.beginPath();
    ctx.arc(playheadX, timelineY + timelineH / 2, 4, 0, Math.PI * 2);
    ctx.fill();
  }, [currentTime, markers, durationSec]);

  return (
    <div
      style={{
        position: "relative",
        width: "100%",
        height: "100%",
        background: "#000",
        overflow: "hidden",
      }}
    >
      <video
        ref={videoRef}
        src={src}
        style={{
          width: "100%",
          height: "100%",
          objectFit,
          objectPosition: "50% 50%",
        }}
        preload="auto"
        playsInline
        muted
        onTimeUpdate={() => {
          if (videoRef.current) {
            onTimeUpdate(videoRef.current.currentTime);
          }
        }}
      />

      {/* Canvas overlay for markers */}
      <canvas
        ref={canvasRef}
        style={{
          position: "absolute",
          inset: 0,
          width: "100%",
          height: "100%",
          pointerEvents: "none",
          zIndex: 5,
        }}
      />

      {/* Bottom-left timestamp overlay */}
      <div
        style={{
          position: "absolute",
          bottom: 8,
          left: 8,
          background: "rgba(0,0,0,0.65)",
          padding: "2px 8px",
          borderRadius: "var(--radius-sm)",
          pointerEvents: "none",
          fontFamily: "var(--font-mono)",
          fontSize: 11,
          color: "rgba(255,255,255,0.7)",
          zIndex: 6,
        }}
      >
        {formatTime(currentTime)}
      </div>

      {/* Frame number badge */}
      <div
        style={{
          position: "absolute",
          bottom: 8,
          right: 8,
          background: "rgba(0,0,0,0.65)",
          padding: "2px 8px",
          borderRadius: "var(--radius-sm)",
          pointerEvents: "none",
          fontFamily: "var(--font-mono)",
          fontSize: 10,
          color: "rgba(255,255,255,0.5)",
          zIndex: 6,
        }}
      >
        #{(currentTime * 30).toFixed(0)}
      </div>
    </div>
  );
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}
