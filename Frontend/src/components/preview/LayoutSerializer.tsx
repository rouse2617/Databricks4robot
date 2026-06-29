import { Button, Tooltip, message } from "antd";
import { DownloadOutlined, UploadOutlined } from "@ant-design/icons";
import { useCallback, useRef } from "react";

export interface LayoutSnapshot {
  version: 1;
  exportedAt: string;
  activeTopics: string[];
  coverMode: boolean;
}

interface LayoutSerializerProps {
  activeTopics: string[];
  coverMode: boolean;
  onImport: (snapshot: LayoutSnapshot) => void;
}

/** Export/import layout buttons. Exported JSON contains active channels + view settings. */
export default function LayoutSerializer({ activeTopics, coverMode, onImport }: LayoutSerializerProps) {
  const fileRef = useRef<HTMLInputElement>(null);

  const handleExport = useCallback(() => {
    const snapshot: LayoutSnapshot = {
      version: 1,
      exportedAt: new Date().toISOString(),
      activeTopics,
      coverMode,
    };
    const blob = new Blob([JSON.stringify(snapshot, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `preview-layout-${snapshot.exportedAt.slice(0, 19).replace(/[T:]/g, "-")}.json`;
    a.click();
    URL.revokeObjectURL(url);
    message.success("布局已导出");
  }, [activeTopics, coverMode]);

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        try {
          const parsed = JSON.parse(reader.result as string);
          if (!parsed.version || !Array.isArray(parsed.activeTopics)) {
            message.error("无效的布局文件");
            return;
          }
          onImport({
            version: parsed.version,
            exportedAt: parsed.exportedAt || "",
            activeTopics: parsed.activeTopics,
            coverMode: parsed.coverMode ?? true,
          });
          message.success("布局已导入");
        } catch {
          message.error("JSON 解析失败");
        }
      };
      reader.readAsText(file);
      // Reset so same file can be re-imported
      e.target.value = "";
    },
    [onImport],
  );

  return (
    <>
      <input
        ref={fileRef}
        type="file"
        accept=".json"
        style={{ display: "none" }}
        onChange={handleFileChange}
      />
      <Tooltip title="导出布局">
        <Button type="text" size="small" icon={<DownloadOutlined />} onClick={handleExport} />
      </Tooltip>
      <Tooltip title="导入布局">
        <Button type="text" size="small" icon={<UploadOutlined />} onClick={() => fileRef.current?.click()} />
      </Tooltip>
    </>
  );
}
