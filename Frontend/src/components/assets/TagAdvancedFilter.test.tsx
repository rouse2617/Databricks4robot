import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import TagAdvancedFilter, { resolveTagField } from "./TagAdvancedFilter";

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

	it("can remove condition after adding", () => {
		render(<TagAdvancedFilter onApply={vi.fn()} />);
		fireEvent.click(screen.getByText("添加条件"));
		expect(screen.getAllByText("AND").length).toBeGreaterThanOrEqual(1);
		const removeButtons = screen.getAllByRole("button").filter((btn) => {
			const className = btn.getAttribute("class") ?? "";
			return className.includes("ant-btn-dangerous");
		});
		expect(removeButtons.length).toBeGreaterThanOrEqual(1);
		fireEvent.click(removeButtons[0]);
		expect(screen.queryByText("AND")).toBeNull();
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

	it("maps priority and quality to tags_flat fields", () => {
		expect(resolveTagField("priority")).toBe("tags_flat.priority");
		expect(resolveTagField("quality")).toBe("tags_flat.quality");
		expect(resolveTagField("scene")).toBe("tags.scene");
	});

	it("applies key-only filter as tags.key equality", () => {
		const onApply = vi.fn();
		render(<TagAdvancedFilter onApply={onApply} />);
		const inputs = screen.getAllByRole("textbox");
		fireEvent.change(inputs[0], { target: { value: "scene" } });
		fireEvent.click(screen.getByText("应用筛选"));
		expect(onApply).toHaveBeenCalledWith(
			expect.arrayContaining([
				expect.objectContaining({
					field: "tags.key",
					op: "eq",
					value: "scene",
				}),
			]),
		);
	});

});
