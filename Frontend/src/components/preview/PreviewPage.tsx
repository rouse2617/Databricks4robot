import { useCallback, useState, useEffect, useMemo, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { message } from "antd";
import AssetInputBar from "./AssetInputBar";
import PreviewLayout from "./PreviewLayout";
import VideoGrid from "./VideoGrid";
import SidebarPanel from "./SidebarPanel";
import UnifiedTimeline from "./UnifiedTimeline";
import LayoutSerializer from "./LayoutSerializer";
import type { LayoutSnapshot } from "./LayoutSerializer";

function readSessionToken(): string | null {
  if (typeof document === "undefined" || !document.cookie) return null;
  for (const part of document.cookie.split(";")) {
    const [k, ...rest] = part.trim().split("=");
    if (k !== "databrew_session") continue;
    const raw = rest.join("=");
    if (!raw) return null;
    try { return decodeURIComponent(raw); } catch { return raw; }
  }
  return null;
}

interface PreviewChannel {
  topic: string;
  label: string;
}

interface PreviewSource {
  id: string;
  topic: string;
  url: string;
}

interface LoadedState {
  assetId: string;
  channels: PreviewChannel[];
  sources: PreviewSource[];
  durationMs: number;
  fileSize?: number;
  rawManifest: Record<string, unknown>;
}

const PREVIEW_BASE = "";

function topicToLabel(topic: string): string {
  const parts = topic.split("/");
  for (let i = 0; i < parts.length; i++) {
    if (parts[i] === "camera" && i + 1 < parts.length) return parts[i + 1];
  }
  return topic.split("/").filter(Boolean).pop() || topic;
}

/** Derive a sensible duration in ms from manifest — prefers window.duration_ms,
 *  falls back to stats effective window. */
function deriveDurationMs(raw: Record<string, unknown>): number {
  const win = raw.window as { duration_ms?: number } | undefined;
  if (win?.duration_ms && win.duration_ms > 0) return win.duration_ms;

  const stats = raw.stats as
    | { window_effective_start_ns?: number; window_effective_end_ns?: number }
    | undefined;
  if (
    stats?.window_effective_end_ns &&
    stats?.window_effective_start_ns &&
    stats.window_effective_end_ns > stats.window_effective_start_ns
  ) {
    return (stats.window_effective_end_ns - stats.window_effective_start_ns) / 1e6;
  }
  return 0;
}

export default function PreviewPage() {
  const [searchParams] = useSearchParams();
  const [loaded, setLoaded] = useState<LoadedState | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeTopics, setActiveTopics] = useState<string[]>([]);
  const [currentTime, setCurrentTime] = useState(0);
  const [playing, setPlaying] = useState(false);
  const [coverMode, setCoverMode] = useState(true);
  const [previewRange, setPreviewRange] = useState<{ startSec: number; endSec: number } | null>(null);

  const handleLoad = useCallback(async (assetId: string) => {
    setLoading(true);
    try {
      const res = await fetch(
        `${PREVIEW_BASE}/api/v1/preview/assets/${assetId}/manifest`,
        { credentials: "include" },
      );
      if (!res.ok) {
        message.error(`加载失败: ${res.status}`);
        return;
      }
      const data = await res.json();

      const rawSources: PreviewSource[] = data.sources || [];
      const isMP4 = data.source_type === "mp4";

      const sources = rawSources.filter((s) => s.topic || isMP4);
      if (sources.length === 0) {
        message.error("该资产没有可预览的视频通道");
        return;
      }

      const channels: PreviewChannel[] = sources.map((s) => ({
        topic: s.topic || s.id,
        label: s.topic ? topicToLabel(s.topic) : "video",
      }));

      const mcapEntry = data.mcap as { size_bytes?: number } | undefined;

      setLoaded({
        assetId,
        channels,
        sources,
        durationMs: deriveDurationMs(data),
        fileSize: mcapEntry?.size_bytes,
        rawManifest: data,
      });
      setActiveTopics(channels.filter((c) => !c.topic.toLowerCase().includes('side')).map((c) => c.topic));
      setCurrentTime(0);
      setPlaying(false);

      // Prewarm the MCAP reader cache (segment endpoint is slow on first load)
      fetch(`${PREVIEW_BASE}/api/v1/preview/assets/${assetId}/prewarm`, {
        method: "POST",
        credentials: "include",
      }).catch(() => {/* non-blocking */});
    } catch (err) {
      message.error(`请求失败: ${String(err)}`);
    } finally {
      setLoading(false);
    }
  }, []);

  const toggleTopic = useCallback((topic: string) => {
    setActiveTopics((prev) =>
      prev.includes(topic) ? prev.filter((t) => t !== topic) : [...prev, topic],
    );
  }, []);

  const getVideoSrc = useCallback(
    (ch: PreviewChannel): string => {
      const src = loaded?.sources.find((s) => s.topic === ch.topic || s.id === ch.topic);
      if (!src) return "";
      const fullUrl = `${PREVIEW_BASE}${src.url}`;
      const qIdx = fullUrl.indexOf("?");
      const baseUrl = qIdx >= 0 ? fullUrl.slice(0, qIdx) : fullUrl;
      const q = new URLSearchParams(qIdx >= 0 ? fullUrl.slice(qIdx + 1) : "");
      if (previewRange) {
        q.set("start_sec", String(previewRange.startSec));
        q.set("end_sec", String(previewRange.endSec));
      }
      const token = readSessionToken();
      if (token) q.set("databrew_token", encodeURIComponent(token));
      const qs = q.toString();
      return qs ? `${baseUrl}?${qs}` : baseUrl;
    },
    [loaded?.sources, previewRange],
  );

  // ── Throttle video timeupdate → React state ──
  // Native <video> fires timeupdate up to ~60Hz, and every active panel fires
  // its own. Feeding each one into setCurrentTime re-renders the whole preview
  // subtree dozens of times/sec → jank. Throttle to ~10Hz: the timeline + overlay
  // stay visually smooth while React work drops ~6-12x.
  /** Tracks user seek target so timeupdate doesn't jump back until video catches up. */
  const seekTargetRef = useRef<{ time: number; deadline: number } | null>(null);
  const lastEmitRef = useRef(0);
  const handleVideoTimeUpdate = useCallback((t: number) => {
    const st = seekTargetRef.current;
    if (st) {
      // Block stale timeupdate until the video reaches the seek target (or 10s).
      if (performance.now() < st.deadline && Math.abs(t - st.time) > 0.5) return;
      seekTargetRef.current = null;
      lastEmitRef.current = 0; // force an immediate emit once the seek settles
    }
    const now = performance.now();
    if (now - lastEmitRef.current < 90) return;
    lastEmitRef.current = now;
    setCurrentTime(t);
  }, []);

  // Stable metadata object so the memoized SidebarPanel (3D viewport + charts)
  // doesn't re-render on every currentTime tick.
  const sidebarMetadata = useMemo(
    () =>
      loaded
        ? {
            durationMs: loaded.durationMs,
            fileSize: loaded.fileSize,
            channelCount: loaded.channels.length,
            assetId: loaded.assetId,
          }
        : null,
    [loaded],
  );

  const statsWin = loaded?.rawManifest?.stats as
    | { window_effective_start_ns?: number }
    | undefined;
  const win = loaded?.rawManifest?.window as
    | { start_timestamp_ns?: number }
    | undefined;
  /** Effective start timestamp for Unix-time display — falls back from window to stats. */
  const effectiveStartNs = win?.start_timestamp_ns || statsWin?.window_effective_start_ns || 0;

  // Auto-load asset from URL params: /preview?asset=<id>&start=<sec>&end=<sec>
  useEffect(() => {
    const assetId = searchParams.get("asset");
    if (!assetId) return;
    handleLoad(assetId).then(() => {
      const startSec = Number(searchParams.get("start"));
      const endSec = Number(searchParams.get("end"));
      if (startSec > 0 && endSec > startSec) {
        setPreviewRange({ startSec, endSec });
      }
    });
  // Only run on mount — intentionally ignore handleLoad dep (stable useCallback)
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleImportLayout = useCallback((snapshot: LayoutSnapshot) => {
    setActiveTopics(snapshot.activeTopics);
    setCoverMode(snapshot.coverMode);
  }, []);




  // ── Keyboard shortcuts ──
  const durationSecRef = useRef(0);
  durationSecRef.current = loaded ? loaded.durationMs / 1000 : 0;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

      const seekAllVideos = (t: number) => {
        document.querySelectorAll("video").forEach((v) => { v.currentTime = t; });
      };

      switch (e.code) {
        case "Space":
          e.preventDefault();
          setPlaying((p) => {
            const next = !p;
            document.querySelectorAll("video").forEach((v) => {
              if (next) v.play().catch(() => {});
              else v.pause();
            });
            return next;
          });
          break;
        case "ArrowLeft":
          e.preventDefault();
          setCurrentTime((t) => {
            const nt = Math.max(0, t - 1 / 30);
            seekAllVideos(nt);
            return nt;
          });
          break;
        case "ArrowRight":
          e.preventDefault();
          setCurrentTime((t) => {
            const nt = Math.min(durationSecRef.current, t + 1 / 30);
            seekAllVideos(nt);
            return nt;
          });
          break;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // Auto-seek to range start when range changes
  useEffect(() => {
    if (previewRange) {
      const t = previewRange.startSec;
      setCurrentTime(t);
      document.querySelectorAll("video").forEach((v) => { v.currentTime = t; });
    }
  }, [previewRange]);

  return (
    <div style={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 4, padding: "4px 12px", background: "#fff", borderBottom: "1px solid var(--gray-100)" }}>
        <div style={{ flex: 1 }}>
          <AssetInputBar
            onLoad={handleLoad} loading={loading}
            range={previewRange}
            onRangeChange={setPreviewRange}
            startTimestampNs={effectiveStartNs}
          />
        </div>
        <LayoutSerializer
          activeTopics={loaded?.channels.filter(c => activeTopics.includes(c.topic)).map(c => c.topic) || activeTopics}
          coverMode={coverMode}
          onImport={handleImportLayout}
        />
      </div>

      {loaded ? (
        <PreviewLayout
          viewport={
            <VideoGrid
              channels={loaded.channels}
              activeTopics={activeTopics}
              getVideoSrc={getVideoSrc}
              currentTime={currentTime}
              playing={playing}
              coverMode={coverMode}
              onTimeUpdate={handleVideoTimeUpdate}
            />
          }
          sidebar={
            sidebarMetadata && (
              <SidebarPanel
                channels={loaded.channels}
                activeTopics={activeTopics}
                onToggle={toggleTopic}
                metadata={sidebarMetadata}
              />
            )
          }
          timeline={
            <UnifiedTimeline
              currentTime={currentTime}
              durationSec={loaded.durationMs / 1000}
              playbackRate={1}
              playing={playing}
              coverMode={coverMode}
              onToggleCover={() => setCoverMode((v) => !v)}
              range={previewRange ? { startSec: previewRange.startSec, endSec: previewRange.endSec, color: "rgba(0, 180, 216, 0.35)", label: "区间" } : undefined}
              onRangeChange={(r) => setPreviewRange(r)}
              startTimestampNs={effectiveStartNs}
              onPlayPause={() => {
                setPlaying((p) => {
                  const next = !p;
                  document.querySelectorAll("video").forEach((v) => {
                    if (next) v.play().catch(() => {});
                    else v.pause();
                  });
                  return next;
                });
              }}
              onSeek={(t) => {
                const clamped = Math.max(0, t);
                setCurrentTime(clamped);
                document.querySelectorAll("video").forEach((v) => { v.currentTime = clamped; });
                // Set seek target: blocks stale timeupdate until video catches up
                // or 10s deadline passes (whichever comes first).
                seekTargetRef.current = { time: clamped, deadline: performance.now() + 10000 };
              }}
              onChangeRate={(r) => {
                document.querySelectorAll("video").forEach((v) => {
                  v.playbackRate = r;
                });
              }}
            />
          }
        />
      ) : (
        <div
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: 12,
            color: "var(--gray-400)",
          }}
        >
          <span style={{ fontSize: 40 }}>🎥</span>
          <span style={{ fontSize: "var(--font-size-base)" }}>
            输入资产 ID 开始预览
          </span>
        </div>
      )}
    </div>
  );
}
