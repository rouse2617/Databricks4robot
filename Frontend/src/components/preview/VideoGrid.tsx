import { useState, useCallback } from "react";
import { Button, Tooltip } from "antd";
import {
  FullscreenOutlined,
  CompressOutlined,
} from "@ant-design/icons";
import CanvasVideoPanel from "./CanvasVideoPanel";

interface PreviewChannel {
  topic: string;
  label: string;
}

interface VideoGridProps {
  channels: PreviewChannel[];
  activeTopics: string[];
  getVideoSrc: (channel: PreviewChannel) => string;
  currentTime: number;
  playing: boolean;
  coverMode: boolean;
  onTimeUpdate: (time: number) => void;
  gazePoint?: { x: number; y: number };
}

export default function VideoGrid({
  channels,
  activeTopics,
  getVideoSrc,
  currentTime,
  playing,
  coverMode,
  onTimeUpdate,
  gazePoint,
}: VideoGridProps) {
  const active = channels.filter((c) => activeTopics.includes(c.topic));
  const [maximizedTopic, setMaximizedTopic] = useState<string | null>(null);
  // Local ordering for drag-to-reorder
  const [order, setOrder] = useState<string[]>([]);
  const [dragOver, setDragOver] = useState<string | null>(null);

  // Stable ordering: use explicit order, then fall back to active array
  const visible = (maximizedTopic
    ? active.filter((c) => c.topic === maximizedTopic)
    : active
  ).slice().sort((a, b) => {
    const oa = order.indexOf(a.topic);
    const ob = order.indexOf(b.topic);
    if (oa === -1 && ob === -1) return 0;
    if (oa === -1) return 1;
    if (ob === -1) return -1;
    return oa - ob;
  });

  const count = visible.length;

  // ── Drag & Drop ──
  const handleDragStart = useCallback((e: React.DragEvent, topic: string) => {
    e.dataTransfer.setData("text/plain", topic);
    e.dataTransfer.effectAllowed = "move";
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent, topic: string) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDragOver(topic);
  }, []);

  const handleDragLeave = useCallback(() => {
    setDragOver(null);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent, targetTopic: string) => {
      e.preventDefault();
      setDragOver(null);
      const srcTopic = e.dataTransfer.getData("text/plain");
      if (!srcTopic || srcTopic === targetTopic) return;

      // Swap positions
      setOrder((prev) => {
        const topics = active.map((c) => c.topic);
        const srcIdx = topics.indexOf(srcTopic);
        const dstIdx = topics.indexOf(targetTopic);
        if (srcIdx === -1 || dstIdx === -1) return prev;

        const next = [...(prev.length ? prev : topics)];
        // Remove src and insert at dst position
        const srcPos = next.indexOf(srcTopic);
        const dstPos = next.indexOf(targetTopic);
        if (srcPos !== -1 && dstPos !== -1) {
          next.splice(srcPos, 1);
          next.splice(dstPos, 0, srcTopic);
        }
        return next;
      });
    },
    [active],
  );

  // Determine grid template columns based on count
  const getGridStyle = (): React.CSSProperties => {
    if (count <= 1) return { gridTemplateColumns: "1fr" };
    if (count <= 2) return { gridTemplateColumns: "1fr 1fr" };
    if (count === 3) return { gridTemplateColumns: "2fr 1fr" };
    if (count <= 4) return { gridTemplateColumns: "1fr 1fr" };
    return { gridTemplateColumns: "repeat(3, 1fr)" };
  };

  return (
    <div
      style={{
        flex: 1,
        padding: 8,
        overflow: "auto",
        background: "var(--gray-50)",
        display: "flex",
      }}
    >
      {count === 0 ? (
        <div
          style={{
            flex: 1,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "var(--gray-400)",
          }}
        >
          请勾选需要查看的摄像头通道
        </div>
      ) : (
        <div
          style={{
            display: "grid",
            gap: 8,
            width: "100%",
            ...getGridStyle(),
          }}
        >
          {visible.map((ch, i) => {
            // For 3 channels: first panel spans 2 rows, second and third sit below
            const isSpecial = count === 3;
            const gridRow =
              isSpecial && i === 0 ? "1 / 3" : undefined;
            const gridCol =
              isSpecial && i === 0 ? "1 / 2" : isSpecial ? "2 / 3" : undefined;

            const isDropTarget = dragOver === ch.topic;

            return (
              <div
                key={ch.topic}
                style={{
                  display: "flex",
                  flexDirection: "column",
                  background: "#000",
                  borderRadius: "var(--radius-md)",
                  overflow: "hidden",
                  border: isDropTarget
                    ? "2px dashed var(--color-primary)"
                    : "1px solid var(--gray-200)",
                  gridRow,
                  gridColumn: gridCol,
                  minHeight: count <= 1 ? 0 : 250,
                  opacity: isDropTarget ? 0.85 : 1,
                  transition: "border 150ms, opacity 150ms",
                }}
              >
                {/* Header bar — draggable */}
                <div
                  draggable
                  onDragStart={(e) => handleDragStart(e, ch.topic)}
                  onDragOver={(e) => handleDragOver(e, ch.topic)}
                  onDragLeave={handleDragLeave}
                  onDrop={(e) => handleDrop(e, ch.topic)}
                  style={{
                    height: 28,
                    display: "flex",
                    alignItems: "center",
                    padding: "0 8px",
                    gap: 6,
                    background: "rgba(0,0,0,0.5)",
                    borderBottom: "1px solid rgba(255,255,255,0.1)",
                    flexShrink: 0,
                    cursor: "grab",
                  }}
                >
                  <span style={{ color: "rgba(255,255,255,0.4)", fontSize: 14 }}>
                    ⠿
                  </span>
                  <span
                    style={{
                      color: "rgba(255,255,255,0.7)",
                      fontSize: "var(--font-size-xs)",
                      flex: 1,
                    }}
                  >
                    {ch.label}
                  </span>

                  {/* Maximize toggle */}
                  <Tooltip title={maximizedTopic ? "恢复多视图" : "放大"}>
                    <Button
                      type="text"
                      size="small"
                      icon={
                        maximizedTopic ? (
                          <CompressOutlined style={{ color: "rgba(255,255,255,0.5)" }} />
                        ) : (
                          <FullscreenOutlined style={{ color: "rgba(255,255,255,0.5)" }} />
                        )
                      }
                      onClick={() =>
                        setMaximizedTopic(
                          maximizedTopic === ch.topic ? null : ch.topic,
                        )
                      }
                      style={{ width: 24, height: 24 }}
                    />
                  </Tooltip>
                </div>

                {/* Video */}
                <div style={{ flex: 1, position: "relative" }}>
                  <CanvasVideoPanel
                    src={getVideoSrc(ch)}
                    currentTime={currentTime}
                    playing={playing}
                    objectFit={coverMode ? "cover" : "contain"}
                    onTimeUpdate={onTimeUpdate}
                    gazePoint={gazePoint}
                  />
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
