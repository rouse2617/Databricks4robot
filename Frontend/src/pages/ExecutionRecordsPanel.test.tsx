// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { ExecutionRecordsPanel } from "./ExecutionRecordsPanel";

function renderPanel(entry: string) {
	return render(
		<MemoryRouter initialEntries={[entry]}>
			<Routes>
				<Route path="/pipeline" element={<ExecutionRecordsPanel active />} />
			</Routes>
		</MemoryRouter>,
	);
}

describe("ExecutionRecordsPanel", () => {
	it("opens batch tab when executionView=batch is in the url", () => {
		renderPanel("/pipeline?tab=executions&executionView=batch");
		expect(screen.getByText("批次名称")).toBeTruthy();
	});

	it("switches to batch tab and updates the url", () => {
		renderPanel("/pipeline?tab=executions");
		fireEvent.click(screen.getByText("批量任务"));
		expect(window.location.search).toContain("executionView=batch");
	});
});
