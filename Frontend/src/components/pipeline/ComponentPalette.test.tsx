// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ComponentPalette } from "./ComponentPalette";
import type { RegisteredComponent } from "./types";

const sampleComponent: RegisteredComponent = {
	id: "comp-1",
	name: "Load Data",
	image: "LD",
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
});
