import { useEffect, useRef, useCallback, useState, useMemo } from "react";
import { LoadingOutlined } from "@ant-design/icons";

export interface BoundingBox {
  /** Normalized center x [0, 1] */
  x: number;
  /** Normalized center y [0, 1] */
  y: number;
  /** Normalized width [0, 1] */
  width: number;
  /** Normalized height [0, 1] */
  height: number;
  label: string;
  color?: string;
  confidence?: number;
}

interface CanvasVideoPanelProps {
  src: string;
  currentTime: number;
  playing?: boolean;
  objectFit?: "cover" | "contain";
  onTimeUpdate?: (time: number) => void;
  boxes?: BoundingBox[];
  /** Gaze point — normalized [0,1] rendered as crosshair */
  gazePoint?: { x: number; y: number };
}

// Stable empty default so the overlay effect's dep array doesn't change identity
// on every render (a fresh `[]` literal would force a redraw each time).
const NO_BOXES: BoundingBox[] = [];

/**
 * Canvas-based video panel with WebCodecs H.264 decoding + Canvas rendering.
 * Falls back to native <video> when WebCodecs is unavailable.
 *
 * Supports pan (drag), zoom (scroll wheel), and bounding-box annotation overlay.
 */
type VideoStatus = "loading" | "playing" | "error" | "idle";

export default function CanvasVideoPanel({
  src,
  currentTime,
  playing = false,
  objectFit = "cover",
  onTimeUpdate,
  boxes = NO_BOXES,
  gazePoint,
}: CanvasVideoPanelProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasSizeRef = useRef({ w: 0, h: 0 });
  const retryCountRef = useRef(0);
  const useNative = true // force native video;

  // ── Video state tracking ──
  const [status, setStatus] = useState<VideoStatus>(src ? "loading" : "idle");
  const [errorMsg, setErrorMsg] = useState("");

  // Reset status and retry counter when src changes
  useEffect(() => {
    setStatus(src ? "loading" : "idle");
    setErrorMsg("");
    retryCountRef.current = 0;
  }, [src]);

  const handleVideoEvent = useCallback((type: string) => {
    switch (type) {
      case "canplay":
        setStatus("playing");
        break;
      case "playing": {
        setStatus("playing");
        // Sync to currentTime when video starts — keeps multi-camera in sync
        const v = videoRef.current;
        if (v && currentTime > 0 && Math.abs(v.currentTime - currentTime) > 0.5) {
          v.currentTime = currentTime;
        }
        break;
      }
      case "waiting":
        setStatus("loading");
        break;
      case "stalled":
        setStatus("loading");
        break;
      case "suspend":
        break;
    }
  }, [currentTime]);

  const handleVideoError = useCallback(() => {
    const v = videoRef.current;
    if (!v) return;
    const code = v.error?.code;
    // MEDIA_ERR_DECODE (3): backend streams a cold H.265→H.264 transcode live;
    // the fMP4 is sometimes malformed on first delivery but the cache is
    // committed within seconds. Retry up to 2× after a 3s delay.
    if (code === 3 && retryCountRef.current < 2) {
      retryCountRef.current++;
      setStatus("loading");
      setTimeout(() => {
        const vid = videoRef.current;
        if (vid?.src) vid.load();
      }, 3000);
      return;
    }
    const msg = v.error ? `MEDIA_${code}: ${v.error.message}` : "视频加载失败";
    setErrorMsg(msg);
    setStatus("error");
  }, []);

  // ── Pan & Zoom state ──
  const [transform, setTransform] = useState({ scale: 1, x: 0, y: 0 });
  const [showZoom, setShowZoom] = useState(false);
  const zoomTimer = useRef<ReturnType<typeof setTimeout>>(null);
  const dragRef = useRef({ active: false, startX: 0, startY: 0, tx: 0, ty: 0 });

  const flashZoom = useCallback((scale: number) => {
    setShowZoom(true);
    if (zoomTimer.current) clearTimeout(zoomTimer.current);
    zoomTimer.current = setTimeout(() => setShowZoom(false), 1000);
    return scale;
  }, []);

  // ── Wheel zoom ──
  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY > 0 ? -0.15 : 0.15;
      setTransform((prev) => {
        const next = Math.max(0.5, Math.min(8, prev.scale + delta));
        flashZoom(next);
        return { scale: next, x: prev.x, y: prev.y };
      });
    },
    [flashZoom],
  );

  // ── Pan (mouse drag) ──
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return;
    setTransform((prev) => {
      if (prev.scale <= 1) return prev;
      dragRef.current = { active: true, startX: e.clientX, startY: e.clientY, tx: prev.x, ty: prev.y };
      return prev;
    });
    e.preventDefault();
  }, []);

  useEffect(() => {
    const onMove = (e: MouseEvent) => {
      if (!dragRef.current.active) return;
      const dx = e.clientX - dragRef.current.startX;
      const dy = e.clientY - dragRef.current.startY;
      setTransform((prev) => ({ ...prev, x: dragRef.current.tx + dx, y: dragRef.current.ty + dy }));
    };
    const onUp = () => { dragRef.current.active = false; };
    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    return () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
  }, []);

  // ── Double-click reset ──
  const handleDoubleClick = useCallback(() => {
    setTransform({ scale: 1, x: 0, y: 0 });
    flashZoom(1);
  }, [flashZoom]);

  // ── Sync playing state ──
  useEffect(() => {
    const v = videoRef.current;
    if (!v || useNative || !v.src) return;
    if (playing) { setTimeout(() => v.play().catch(() => {}), Math.random() * 1500); }
    else v.pause();
  }, [playing, useNative]);

  // ── Sync currentTime ──
  useEffect(() => {
    const v = videoRef.current;
    if (!v || useNative || !v.src) return;
    if (Math.abs(v.currentTime - currentTime) > 0.3) v.currentTime = currentTime;
  }, [currentTime, useNative]);

  const handleTimeUpdate = useCallback(() => {
    if (videoRef.current && onTimeUpdate) onTimeUpdate(videoRef.current.currentTime);
  }, [onTimeUpdate]);

  // ── Canvas overlay: timestamp + frame + annotations ──
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    // Only touch canvas.width/height when the size actually changed. Assigning
    // them every frame reallocates the backing buffer (expensive) and reading
    // clientWidth/Height every frame forces a synchronous layout reflow.
    const parent = canvas.parentElement;
    if (parent) {
      const pw = parent.clientWidth;
      const ph = parent.clientHeight;
      if (pw !== canvasSizeRef.current.w || ph !== canvasSizeRef.current.h) {
        canvas.width = pw;
        canvas.height = ph;
        canvasSizeRef.current = { w: pw, h: ph };
      }
    }

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // ── Bounding box annotations ──
    for (const box of boxes) {
      const cx = box.x * canvas.width;
      const cy = box.y * canvas.height;
      const bw = box.width * canvas.width;
      const bh = box.height * canvas.height;
      const x1 = cx - bw / 2;
      const y1 = cy - bh / 2;
      const color = box.color || "#00ff88";

      ctx.fillStyle = "rgba(0,255,136,0.08)";
      ctx.fillRect(x1, y1, bw, bh);

      ctx.strokeStyle = color;
      ctx.lineWidth = 2;
      ctx.strokeRect(x1, y1, bw, bh);

      const label = `${box.label}${box.confidence ? ` ${Math.round(box.confidence * 100)}%` : ""}`;
      ctx.font = "bold 11px monospace";
      const labelW = ctx.measureText(label).width + 12;
      ctx.fillStyle = color;
      ctx.fillRect(x1, Math.max(0, y1 - 20), labelW, 20);
      ctx.fillStyle = "#000";
      ctx.textAlign = "left";
      ctx.fillText(label, x1 + 6, Math.max(0, y1 - 20) + 14);
    }

    // ── Gaze point ──
    if (gazePoint) {
      const gx = gazePoint.x * canvas.width;
      const gy = gazePoint.y * canvas.height;
      const r = 12;

      // Outer circle
      ctx.beginPath();
      ctx.arc(gx, gy, r, 0, Math.PI * 2);
      ctx.strokeStyle = "rgba(255, 60, 60, 0.7)";
      ctx.lineWidth = 2;
      ctx.stroke();

      // Inner cross
      ctx.beginPath();
      ctx.moveTo(gx - r, gy);
      ctx.lineTo(gx - 4, gy);
      ctx.moveTo(gx + 4, gy);
      ctx.lineTo(gx + r, gy);
      ctx.moveTo(gx, gy - r);
      ctx.lineTo(gx, gy - 4);
      ctx.moveTo(gx, gy + 4);
      ctx.lineTo(gx, gy + r);
      ctx.strokeStyle = "rgba(255, 60, 60, 0.9)";
      ctx.lineWidth = 1.5;
      ctx.stroke();

      // Center dot
      ctx.beginPath();
      ctx.arc(gx, gy, 2.5, 0, Math.PI * 2);
      ctx.fillStyle = "rgba(255, 0, 0, 0.9)";
      ctx.fill();
    }

    // Timestamp badge
    const mins = Math.floor(currentTime / 60);
    const secs = Math.floor(currentTime % 60);
    const ts = `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
    ctx.fillStyle = "rgba(0,0,0,0.6)";
    ctx.fillRect(8, canvas.height - 28, 82, 22);
    ctx.fillStyle = "rgba(255,255,255,0.8)";
    ctx.font = "12px monospace";
    ctx.textAlign = "left";
    ctx.fillText(`🎬 ${ts}`, 14, canvas.height - 12);

    // Frame counter
    const frame = (currentTime * 30).toFixed(0);
    ctx.fillStyle = "rgba(0,0,0,0.6)";
    ctx.fillRect(canvas.width - 70, canvas.height - 28, 62, 22);
    ctx.fillStyle = "rgba(255,255,255,0.5)";
    ctx.textAlign = "right";
    ctx.fillText(`#${frame}`, canvas.width - 12, canvas.height - 12);
  }, [currentTime, boxes, gazePoint]);

  const cursor = transform.scale > 1 ? (dragRef.current.active ? "grabbing" : "grab") : "default";

  const content = (
    <div
      ref={containerRef}
      onWheel={handleWheel}
      onMouseDown={handleMouseDown}
      onDoubleClick={handleDoubleClick}
      style={{ position: "absolute", inset: 0, overflow: "hidden", cursor }}
    >
      <div
        style={{
          position: "absolute", inset: 0,
          transform: `scale(${transform.scale}) translate(${transform.x / transform.scale}px, ${transform.y / transform.scale}px)`,
          transformOrigin: "center center",
          transition: dragRef.current.active ? "none" : "transform 0.15s ease-out",
        }}
      >
        <video
          ref={videoRef}
          src={src}
          style={{ width: "100%", height: "100%", objectFit }}
          preload="auto" playsInline muted
          onTimeUpdate={handleTimeUpdate}
          onCanPlay={() => handleVideoEvent("canplay")}
          onPlaying={() => handleVideoEvent("playing")}
          onWaiting={() => handleVideoEvent("waiting")}
          onStalled={() => handleVideoEvent("stalled")}
          onError={handleVideoError}
        />
        <canvas
          ref={canvasRef}
          style={{ position: "absolute", inset: 0, width: "100%", height: "100%", pointerEvents: "none", zIndex: 5 }}
        />
      </div>
      {showZoom && (
        <div style={{
          position: "absolute", top: "50%", left: "50%", transform: "translate(-50%, -50%)",
          background: "rgba(0,0,0,0.75)", color: "#fff", padding: "4px 14px", borderRadius: 20,
          fontSize: 13, fontFamily: "var(--font-mono)", pointerEvents: "none", zIndex: 99, transition: "opacity 0.3s",
        }}>
          {Math.round(transform.scale * 100)}%
        </div>
      )}
    </div>
  );

  return (
    <div style={{ position: "relative", width: "100%", height: "100%", background: "#000" }}>
      {content}

      {/* ── Loading overlay ── */}
      {status === "loading" && (
        <div style={{
          position: "absolute", inset: 0, zIndex: 100,
          display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
          background: "rgba(0,0,0,0.5)", gap: 8, transition: "opacity 0.3s",
        }}>
          <LoadingOutlined style={{ color: "rgba(255,255,255,0.7)", fontSize: 28 }} />
          <span style={{ color: "rgba(255,255,255,0.6)", fontSize: 12, fontFamily: "var(--font-mono)" }}>
            加载视频流…
          </span>
        </div>
      )}

      {/* ── Error overlay ── */}
      {status === "error" && (
        <div style={{
          position: "absolute", inset: 0, zIndex: 100,
          display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
          background: "rgba(0,0,0,0.7)", gap: 6, padding: 20,
        }}>
          <span style={{ fontSize: 24 }}>⚠️</span>
          <span style={{ color: "rgba(255,255,255,0.7)", fontSize: 11, fontFamily: "var(--font-mono)", textAlign: "center", lineHeight: 1.4 }}>
            {errorMsg || "视频无法播放"}
          </span>
        </div>
      )}
    </div>
  );
}
