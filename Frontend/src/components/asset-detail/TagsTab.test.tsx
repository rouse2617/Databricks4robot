import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import TagsTab from "./TagsTab";

describe("TagsTab", () => {
	it("renders tags", () => {
		render(
			<TagsTab
				assetId="a1"
				tags={{ quality: "good", scene: "outdoor" }}
				onUpdate={vi.fn()}
			/>,
		);
		expect(screen.getByText(/quality/)).toBeTruthy();
		expect(screen.getByText(/good/)).toBeTruthy();
		expect(screen.getByText(/scene/)).toBeTruthy();
		expect(screen.getByText(/outdoor/)).toBeTruthy();
	});

	it("renders empty state when no tags", () => {
		render(<TagsTab assetId="a1" tags={{}} onUpdate={vi.fn()} />);
		expect(screen.getByText("暂无标签")).toBeTruthy();
	});

	it("renders add tag button", () => {
		render(<TagsTab assetId="a1" tags={{}} onUpdate={vi.fn()} />);
		expect(screen.getByText("添加标签")).toBeTruthy();
	});
});
