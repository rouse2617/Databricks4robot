// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PipelineEmptyState } from "./PipelineEmptyState";

describe("PipelineEmptyState", () => {
	it("renders canvas empty state", () => {
		render(
			<PipelineEmptyState
				variant="canvas"
				title="拖入组件开始设计"
				hint="从左侧拖入步骤"
			/>,
		);
		expect(screen.getByText("拖入组件开始设计")).toBeTruthy();
		expect(screen.getByText("从左侧拖入步骤")).toBeTruthy();
	});

	it("calls action when clicked", () => {
		const onClick = vi.fn();
		render(
			<PipelineEmptyState
				variant="config"
				title="节点配置"
				action={{ label: "重试", onClick }}
			/>,
		);
		fireEvent.click(screen.getByRole("button", { name: "重试" }));
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
