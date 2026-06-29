import { useEffect, useRef, useCallback } from "react";

interface CanvasRendererProps {
  /** Width of the canvas in pixels */
  width?: number;
  /** Height of the canvas in pixels */
  height?: number;
  /** Object fit behavior */
  objectFit?: "cover" | "contain";
  /** Called with a draw function that renders the current time */
  currentTime?: number;
  /** Animation frame callback for custom rendering */
  onFrame?: (ctx: CanvasRenderingContext2D, time: number, width: number, height: number) => void;
}

/**
 * Canvas-based video/image renderer.
 * Phase 3 foundation: renders to Canvas instead of <video>,
 * enabling overlays, compositing, and WebGL acceleration later.
 */
export default function CanvasRenderer({
  width = 640,
  height = 480,
  objectFit = "contain",
  currentTime = 0,
  onFrame,
}: CanvasRendererProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animRef = useRef<number>(0);

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const cw = canvas.width;
    const ch = canvas.height;

    // Clear
    ctx.fillStyle = "#000";
    ctx.fillRect(0, 0, cw, ch);

    // Call custom frame renderer
    if (onFrame) {
      onFrame(ctx, currentTime, cw, ch);
      return;
    }

    // Default: draw a test pattern with current time
    ctx.fillStyle = "#1a1a2e";
    ctx.fillRect(0, 0, cw, ch);

    // Grid
    ctx.strokeStyle = "rgba(255,255,255,0.05)";
    ctx.lineWidth = 1;
    for (let x = 0; x < cw; x += 40) {
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, ch);
      ctx.stroke();
    }
    for (let y = 0; y < ch; y += 40) {
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(cw, y);
      ctx.stroke();
    }

    // Time display
    ctx.fillStyle = "rgba(255,255,255,0.8)";
    ctx.font = "14px monospace";
    ctx.textAlign = "center";
    ctx.fillText(`Canvas · ${currentTime.toFixed(2)}s`, cw / 2, ch / 2);

    // Resolution badge
    ctx.fillStyle = "rgba(255,255,255,0.3)";
    ctx.font = "11px monospace";
    ctx.textAlign = "right";
    ctx.fillText(`${cw}×${ch}`, cw - 8, ch - 8);
  }, [currentTime, onFrame]);

  useEffect(() => {
    // Animation loop
    let running = true;
    const loop = () => {
      if (!running) return;
      draw();
      animRef.current = requestAnimationFrame(loop);
    };
    loop();
    return () => {
      running = false;
      cancelAnimationFrame(animRef.current);
    };
  }, [draw]);

  return (
    <canvas
      ref={canvasRef}
      width={width}
      height={height}
      style={{
        width: "100%",
        height: "100%",
        objectFit,
        display: "block",
      }}
    />
  );
}
