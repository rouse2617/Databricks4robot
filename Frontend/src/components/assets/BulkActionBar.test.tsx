// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import BulkActionBar from "./BulkActionBar";

afterEach(cleanup);

const noop = () => {};

function renderBar(
	overrides: Partial<Parameters<typeof BulkActionBar>[0]> = {},
) {
	const props = {
		selectedCount: 3,
		selectionMode: "explicit_rows" as const,
		totalFiltered: 128,
		onCreateDelivery: noop,
		onRunPipeline: noop,
		onBatchTag: noop,
		onExportIds: noop,
		onSelectAllFiltered: noop,
		onClearSelection: noop,
		...overrides,
	};
	return render(<BulkActionBar {...props} />);
}

describe("BulkActionBar", () => {
	it("renders null when selectedCount is 0", () => {
		const { container } = renderBar({ selectedCount: 0 });
		expect(container.innerHTML).toBe("");
	});

	it("shows selection count text", () => {
		renderBar({ selectedCount: 5 });
		expect(screen.getByText("已选 5 条")).toBeTruthy();
	});

	it("renders all action buttons", () => {
		renderBar();
		expect(screen.getByText("创建交付")).toBeTruthy();
		expect(screen.getByText("运行 Pipeline")).toBeTruthy();
		expect(screen.getByText("批量打 Tag")).toBeTruthy();
		expect(screen.getByText("导出 ID")).toBeTruthy();
		expect(screen.getByText("取消选择")).toBeTruthy();
	});

	it("calls onCreateDelivery when 创建交付 is clicked", () => {
		const fn = vi.fn();
		renderBar({ onCreateDelivery: fn });
		fireEvent.click(screen.getByText("创建交付"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("calls onRunPipeline when 运行 Pipeline is clicked", () => {
		const fn = vi.fn();
		renderBar({ onRunPipeline: fn });
		fireEvent.click(screen.getByText("运行 Pipeline"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("calls onBatchTag when 批量打 Tag is clicked", () => {
		const fn = vi.fn();
		renderBar({ onBatchTag: fn });
		fireEvent.click(screen.getByText("批量打 Tag"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("calls onExportIds when 导出 ID is clicked", () => {
		const fn = vi.fn();
		renderBar({ onExportIds: fn });
		fireEvent.click(screen.getByText("导出 ID"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("calls onClearSelection when 取消选择 is clicked", () => {
		const fn = vi.fn();
		renderBar({ onClearSelection: fn });
		fireEvent.click(screen.getByText("取消选择"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("shows 'select all filtered' link in explicit_rows mode when totalFiltered > selectedCount", () => {
		renderBar({
			selectedCount: 3,
			selectionMode: "explicit_rows",
			totalFiltered: 128,
		});
		expect(screen.getByText("选择全部 128 条筛选结果")).toBeTruthy();
	});

	it("hides 'select all filtered' link when selectionMode is not explicit_rows", () => {
		renderBar({
			selectedCount: 3,
			selectionMode: "all_filtered_results",
			totalFiltered: 128,
		});
		expect(screen.queryByText(/选择全部.*条筛选结果/)).toBeNull();
	});

	it("hides 'select all filtered' link when totalFiltered <= selectedCount", () => {
		renderBar({
			selectedCount: 128,
			selectionMode: "explicit_rows",
			totalFiltered: 128,
		});
		expect(screen.queryByText(/选择全部.*条筛选结果/)).toBeNull();
	});

	it("calls onSelectAllFiltered when the link is clicked", () => {
		const fn = vi.fn();
		renderBar({ onSelectAllFiltered: fn });
		fireEvent.click(screen.getByText("选择全部 128 条筛选结果"));
		expect(fn).toHaveBeenCalledOnce();
	});

	it("shows warning when selectedCount > 100", () => {
		renderBar({ selectedCount: 150 });
		expect(screen.getByText("批量操作超过 100 条资产，请确认")).toBeTruthy();
	});

	it("does not show warning when selectedCount <= 100", () => {
		renderBar({ selectedCount: 50 });
		expect(screen.queryByText("批量操作超过 100 条资产，请确认")).toBeNull();
	});
});
