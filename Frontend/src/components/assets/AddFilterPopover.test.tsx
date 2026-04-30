import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import AddFilterPopover from "./AddFilterPopover";

describe("AddFilterPopover", () => {
  it("renders the add filter button", () => {
    render(<AddFilterPopover onAddFilter={vi.fn()} />);
    expect(screen.getByText("添加筛选")).toBeTruthy();
  });

  it("opens popover on click", () => {
    render(<AddFilterPopover onAddFilter={vi.fn()} />);
    fireEvent.click(screen.getByText("添加筛选"));
    expect(screen.getByText("添加筛选条件")).toBeTruthy();
  });

  it("shows quick field buttons", () => {
    render(<AddFilterPopover onAddFilter={vi.fn()} />);
    fireEvent.click(screen.getByText("添加筛选"));
    expect(screen.getByText("快捷字段")).toBeTruthy();
  });
});
