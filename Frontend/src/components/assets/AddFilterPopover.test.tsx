import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
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

	it("supports selecting quick field then cancel", () => {
		render(<AddFilterPopover onAddFilter={vi.fn()} />);
		const trigger = screen.getByRole("button", { name: /添加筛选/ });
		fireEvent.click(trigger);
		fireEvent.click(screen.getByText(/\+ 生命周期/));
		expect(screen.getByText("选择操作符")).toBeTruthy();
		fireEvent.click(screen.getByRole("button", { name: /取\s*消/ }));
		expect(trigger.className.includes("ant-popover-open")).toBe(false);
	});
});
