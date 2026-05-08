// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { FilterChip } from "../../lib/assets/assetsDiscoveryTypes";
import QuickFiltersRow from "./QuickFiltersRow";

afterEach(cleanup);

function makeChip(overrides: Partial<FilterChip> = {}): FilterChip {
	return {
		id: "chip_1",
		field: "algo_status",
		op: "eq",
		value: "failed",
		source: "add_filter",
		...overrides,
	};
}

describe("QuickFiltersRow", () => {
	it("renders quick filter tags", () => {
		render(
			<QuickFiltersRow
				activeFilters={[]}
				onAddFilter={() => {}}
				onRemoveFilter={() => {}}
			/>,
		);
		expect(screen.getByText("快捷筛选")).toBeTruthy();
		expect(screen.getByText("Ready 资产")).toBeTruthy();
		expect(screen.getByText("算法失败")).toBeTruthy();
		expect(screen.getByText("高优先级")).toBeTruthy();
		expect(screen.getByText("未交付")).toBeTruthy();
	});

	it("adds a filter when an inactive quick tag is clicked", () => {
		const onAddFilter = vi.fn();
		render(
			<QuickFiltersRow
				activeFilters={[]}
				onAddFilter={onAddFilter}
				onRemoveFilter={() => {}}
			/>,
		);
		fireEvent.click(screen.getByText("算法失败"));
		expect(onAddFilter).toHaveBeenCalledOnce();
		const chip = onAddFilter.mock.calls[0][0] as FilterChip;
		expect(chip.field).toBe("algo_status");
		expect(chip.op).toBe("eq");
		expect(chip.value).toBe("failed");
	});

	it("removes matched filters when an active quick tag is clicked", () => {
		const onRemoveFilter = vi.fn();
		render(
			<QuickFiltersRow
				activeFilters={[
					makeChip({ id: "failed_1" }),
					makeChip({ id: "failed_2", source: "saved_view" }),
				]}
				onAddFilter={() => {}}
				onRemoveFilter={onRemoveFilter}
			/>,
		);
		fireEvent.click(screen.getByText("算法失败"));
		expect(onRemoveFilter).toHaveBeenCalledWith("failed_1");
		expect(onRemoveFilter).toHaveBeenCalledWith("failed_2");
	});
});
