import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import FilesTab from "./FilesTab";

// Mock matchMedia for antd responsive components
Object.defineProperty(window, "matchMedia", {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }),
});

describe("FilesTab", () => {
  it("renders file entries", () => {
    render(<FilesTab files={{ raw_mcap: "gs://bucket/file.mcap", preview: "gs://bucket/preview.mp4" }} />);
    expect(screen.getByText("raw_mcap")).toBeTruthy();
    expect(screen.getByText("gs://bucket/file.mcap")).toBeTruthy();
    expect(screen.getByText("preview")).toBeTruthy();
  });

  it("renders empty state when no files", () => {
    render(<FilesTab files={{}} />);
    expect(screen.getByText("暂无文件引用")).toBeTruthy();
  });
});
