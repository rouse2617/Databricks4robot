import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import TagAdvancedFilter from "./TagAdvancedFilter";

describe("TagAdvancedFilter", () => {
  it("renders with initial condition group", () => {
    render(<TagAdvancedFilter onApply={vi.fn()} />);
    expect(screen.getByText("Tag 高级筛选")).toBeTruthy();
    expect(screen.getByText("条件组 1")).toBeTruthy();
    expect(screen.getByText("当前版本只支持一个 Tag 条件组")).toBeTruthy();
  });

  it("adds a new condition when clicking add button", () => {
    render(<TagAdvancedFilter onApply={vi.fn()} />);
    fireEvent.click(screen.getByText("添加条件"));
    expect(screen.getAllByText("AND").length).toBeGreaterThanOrEqual(1);
  });

  it("calls onApply with filter chips when apply is clicked", () => {
    const onApply = vi.fn();
    render(<TagAdvancedFilter onApply={onApply} />);
    fireEvent.click(screen.getByText("应用筛选"));
    expect(onApply).toHaveBeenCalledWith([]);
  });

  it("calls onClose when close button is clicked", () => {
    const onClose = vi.fn();
    render(<TagAdvancedFilter onApply={vi.fn()} onClose={onClose} />);
    fireEvent.click(screen.getByText("关闭"));
    expect(onClose).toHaveBeenCalled();
  });
});
