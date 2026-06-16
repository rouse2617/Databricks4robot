// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ComponentPalette } from "./ComponentPalette";
import type { RegisteredComponent } from "./types";

const sampleComponent: RegisteredComponent = {
	id: "comp-1",
	name: "Load Data",
	image: "LD",
	tag: "v1.2.3",
	sourceCommit: "abc123",
	command: [],
	args: [],
	cpu: "100m",
	memory: "128Mi",
	disk: "1Gi",
};

describe("ComponentPalette", () => {
	it("disables palette interactions when disabled", () => {
		const onAdd = vi.fn();
		render(
			<ComponentPalette
				components={[sampleComponent]}
				onDragStart={vi.fn()}
				onAddComponent={onAdd}
				disabled
			/>,
		);
		const button = screen.getByRole("button", { name: "添加组件 Load Data" });
		expect(button).toBeDisabled();
		fireEvent.click(button);
		expect(onAdd).not.toHaveBeenCalled();
	});

	it("allows add when not disabled", () => {
		const onAdd = vi.fn();
		render(
			<ComponentPalette
				components={[sampleComponent]}
				onDragStart={vi.fn()}
				onAddComponent={onAdd}
			/>,
		);
		fireEvent.click(screen.getByRole("button", { name: "添加组件 Load Data" }));
		expect(onAdd).toHaveBeenCalledWith(sampleComponent);
	});

	it("filters by name, commit, or tag", () => {
		const otherComponent: RegisteredComponent = {
			...sampleComponent,
			id: "comp-2",
			name: "Render Report",
			image: "report:latest",
			tag: "latest",
			sourceCommit: "fff999",
		};
		render(
			<ComponentPalette
				components={[sampleComponent, otherComponent]}
				onDragStart={vi.fn()}
			/>,
		);

		fireEvent.change(screen.getByPlaceholderText("搜索名称、commit、tag"), {
			target: { value: "abc123" },
		});
		expect(screen.getByText("Load Data")).toBeTruthy();
		expect(screen.queryByText("Render Report")).toBeNull();

		fireEvent.change(screen.getByPlaceholderText("搜索名称、commit、tag"), {
			target: { value: "latest" },
		});
		expect(screen.queryByText("Load Data")).toBeNull();
		expect(screen.getByText("Render Report")).toBeTruthy();
	});
});
