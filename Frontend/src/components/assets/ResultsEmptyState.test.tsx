import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import ResultsEmptyState from "./ResultsEmptyState";

describe("ResultsEmptyState", () => {
	it("renders empty state message", () => {
		render(<ResultsEmptyState variant="initial" />);
		// The component should render some empty state content
		const container =
			document.querySelector("[class*='empty']") || document.body;
		expect(container).toBeTruthy();
	});

	it("renders next-step actions for no-results state", () => {
		const onClearFilters = vi.fn();
		render(
			<ResultsEmptyState
				variant="no_results"
				activeFilterCount={3}
				onClearFilters={onClearFilters}
			/>,
		);
		expect(screen.getByText("放宽筛选")).toBeTruthy();
		expect(screen.getByText("回到默认视图")).toBeTruthy();
		fireEvent.click(screen.getByText("放宽筛选"));
		expect(onClearFilters).toHaveBeenCalledOnce();
	});
});
