import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import EventsPage from "./EventsPage";

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

describe("EventsPage", () => {
  it("renders the page title and search input", () => {
    render(
      <MemoryRouter>
        <EventsPage />
      </MemoryRouter>,
    );
    expect(screen.getByText("事件流总览")).toBeTruthy();
    expect(screen.getByPlaceholderText("输入 Asset ID")).toBeTruthy();
  });

  it("renders event type filter select", () => {
    render(
      <MemoryRouter>
        <EventsPage />
      </MemoryRouter>,
    );
    expect(screen.getByText("查看资产事件流，支持按 Asset ID 和事件类型筛选。")).toBeTruthy();
  });
});
