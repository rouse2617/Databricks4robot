// @vitest-environment jsdom

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { usePipelineKeyboardShortcuts } from "./usePipelineKeyboardShortcuts";

function ShortcutHarness(props: {
	onSave: () => void;
	onDeploy: () => void;
	onClear: () => void;
	canDeploy: boolean;
	modalOpen: boolean;
}) {
	usePipelineKeyboardShortcuts(props);
	return <div>ready</div>;
}

describe("usePipelineKeyboardShortcuts", () => {
	it("triggers save on ctrl+s", () => {
		const onSave = vi.fn();
		render(
			<ShortcutHarness
				onSave={onSave}
				onDeploy={vi.fn()}
				onClear={vi.fn()}
				canDeploy
				modalOpen={false}
			/>,
		);
		window.dispatchEvent(
			new KeyboardEvent("keydown", { key: "s", ctrlKey: true, bubbles: true }),
		);
		expect(onSave).toHaveBeenCalledTimes(1);
	});
});
