import { ReactNode } from "react";

interface PreviewLayoutProps {
  viewport: ReactNode;
  sidebar: ReactNode;
  timeline: ReactNode;
}

export default function PreviewLayout({
  viewport,
  sidebar,
  timeline,
}: PreviewLayoutProps) {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100%",
        background: "var(--gray-50)",
      }}
    >
      {/* Body: viewport + sidebar */}
      <div style={{ display: "flex", flex: 1, overflow: "hidden" }}>
        {/* Main viewport */}
        <div style={{ flex: 1, overflow: "hidden", display: "flex" }}>
          {viewport}
        </div>

        {/* Right sidebar */}
        <div
          style={{
            width: 300,
            flexShrink: 0,
            borderLeft: "1px solid var(--gray-200)",
            background: "#fff",
            display: "flex",
            flexDirection: "column",
            overflow: "auto",
          }}
        >
          {sidebar}
        </div>
      </div>

      {/* Bottom timeline */}
      {timeline}
    </div>
  );
}
