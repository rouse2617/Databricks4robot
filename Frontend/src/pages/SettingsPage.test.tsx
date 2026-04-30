// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { render, screen, fireEvent, cleanup, waitFor } from "@testing-library/react";
import SettingsPage from "./SettingsPage";

// Polyfill matchMedia for Ant Design's responsive observer
beforeEach(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
});

// Mock useAuth
vi.mock("../hooks/useAuth", () => ({
  useAuth: () => ({ token: "test-token-12345678" }),
}));

// Mock admin API
const mockReindex = vi.fn();
vi.mock("../api/admin", () => ({
  adminApi: {
    reindex: (...args: unknown[]) => mockReindex(...args),
  },
}));

// Mock apiError
vi.mock("../lib/apiError", () => ({
  extractApiErrorMessage: (_err: unknown, fallback: string) => fallback,
}));

afterEach(() => {
  cleanup();
  mockReindex.mockReset();
});

describe("SettingsPage", () => {
  it("renders the settings page with session info", () => {
    render(<SettingsPage />);
    expect(screen.getByText("设置")).toBeTruthy();
    expect(screen.getByText("当前会话")).toBeTruthy();
    expect(screen.getByText("X-Grace-Token (Phase 0)")).toBeTruthy();
  });

  it("renders the reindex card", () => {
    render(<SettingsPage />);
    expect(screen.getByText("重建 ES 索引")).toBeTruthy();
    expect(screen.getByTestId("reindex-dry-run-btn")).toBeTruthy();
  });

  it("executes dry run on button click", async () => {
    mockReindex.mockResolvedValue({
      dry_run: true,
      total_assets: 100,
      indexed: 0,
      failed: 0,
      duration_ms: 500,
    });

    render(<SettingsPage />);
    fireEvent.click(screen.getByTestId("reindex-dry-run-btn"));

    await waitFor(() => {
      expect(mockReindex).toHaveBeenCalledWith(true);
    });

    await waitFor(() => {
      expect(screen.getByText("Dry Run 完成")).toBeTruthy();
    });
  });

  it("shows error when dry run fails", async () => {
    mockReindex.mockRejectedValue(new Error("Network error"));

    render(<SettingsPage />);
    fireEvent.click(screen.getByTestId("reindex-dry-run-btn"));

    await waitFor(() => {
      expect(screen.getByText("Dry run 失败")).toBeTruthy();
    });
  });

  it("shows confirm button after dry run", async () => {
    mockReindex.mockResolvedValue({
      dry_run: true,
      total_assets: 50,
      indexed: 0,
      failed: 0,
      duration_ms: 200,
    });

    render(<SettingsPage />);
    fireEvent.click(screen.getByTestId("reindex-dry-run-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("reindex-confirm-btn")).toBeTruthy();
    });
  });
});
