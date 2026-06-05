// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { ComponentPalette } from "./ComponentPalette";
import type { RegisteredComponent } from "./types";

const components: RegisteredComponent[] = [
	{
		id: "comp-1",
		name: "head-track-pycuvslam",
		image: "us-central1-docker.pkg.dev/project/video/head-track:latest",
		tag: "latest",
		type: "container",
		computeTier: "gpu-l4",
	},
	{
		id: "comp-2",
		name: "find-toni-stats",
		image: "us-central1-docker.pkg.dev/project/video/find-toni:manual",
		tag: "manual",
		type: "container",
		computeTier: "cpu-med",
	},
];

function renderPalette() {
	return render(
		<MemoryRouter>
			<ComponentPalette
				components={components}
				onDragStart={vi.fn()}
				onAddComponent={vi.fn()}
			/>
		</MemoryRouter>,
	);
}

describe("ComponentPalette", () => {
	it("filters components by name and compute tier", () => {
		renderPalette();

		fireEvent.change(screen.getByLabelText("搜索组件"), {
			target: { value: "gpu-l4" },
		});

		expect(screen.getByText("head-track-pycuvslam")).toBeInTheDocument();
		expect(screen.queryByText("find-toni-stats")).toBeNull();

		fireEvent.change(screen.getByLabelText("搜索组件"), {
			target: { value: "find-toni" },
		});

		expect(screen.getByText("find-toni-stats")).toBeInTheDocument();
		expect(screen.queryByText("head-track-pycuvslam")).toBeNull();
	});

	it("shows a search-specific empty state and can clear search", () => {
		renderPalette();

		fireEvent.change(screen.getByLabelText("搜索组件"), {
			target: { value: "not-found" },
		});

		expect(screen.getByText("未找到匹配组件")).toBeInTheDocument();
		expect(
			screen.getByText("换个关键词试试，或清空搜索后查看全部组件。"),
		).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "清空搜索" }));

		expect(screen.getByText("head-track-pycuvslam")).toBeInTheDocument();
		expect(screen.getByText("find-toni-stats")).toBeInTheDocument();
	});

	it("keeps drag behavior for filtered items", () => {
		const onDragStart = vi.fn();
		render(
			<MemoryRouter>
				<ComponentPalette
					components={components}
					onDragStart={onDragStart}
					onAddComponent={vi.fn()}
				/>
			</MemoryRouter>,
		);

		fireEvent.change(screen.getByLabelText("搜索组件"), {
			target: { value: "head-track" },
		});

		fireEvent.dragStart(
			screen.getByRole("button", { name: "拖入组件 head-track-pycuvslam" }),
		);

		expect(onDragStart).toHaveBeenCalledTimes(1);
		expect(onDragStart.mock.calls[0]?.[1]).toMatchObject({
			id: "comp-1",
			name: "head-track-pycuvslam",
		});
	});
});
