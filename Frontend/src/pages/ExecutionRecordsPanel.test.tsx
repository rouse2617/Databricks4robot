// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { ExecutionRecordsPanel } from "./ExecutionRecordsPanel";

vi.mock("./BatchJobList", () => ({
	BatchJobList: () => <div>批次名称</div>,
}));

vi.mock("./WorkflowExecutionList", () => ({
	WorkflowExecutionList: () => <div>Workflow 名称</div>,
}));

function LocationEcho() {
	const location = useLocation();
	return <output data-testid="location-search">{location.search}</output>;
}

function renderPanel(entry: string) {
	return render(
		<MemoryRouter initialEntries={[entry]}>
			<Routes>
				<Route
					path="/pipeline"
					element={
						<>
							<LocationEcho />
							<ExecutionRecordsPanel active />
						</>
					}
				/>
			</Routes>
		</MemoryRouter>,
	);
}

describe("ExecutionRecordsPanel", () => {
	it("opens batch tab when executionView=batch is in the url", async () => {
		renderPanel("/pipeline?tab=executions&executionView=batch");
		expect(await screen.findByText("批次名称")).toBeTruthy();
	});

	it("switches to batch tab and updates the url", async () => {
		renderPanel("/pipeline?tab=executions");
		fireEvent.click(screen.getByText("批量任务"));
		expect(await screen.findByText("批次名称")).toBeTruthy();
		expect(screen.getByTestId("location-search").textContent).toContain(
			"executionView=batch",
		);
	});
});
