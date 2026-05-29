// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	within,
} from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { DEFAULT_COLUMNS } from "../../lib/assets/assetsDiscoveryTypes";
import ColumnsConfigPopover, {
	ASSET_TABLE_ORDERED_COLUMN_KEYS,
} from "./ColumnsConfigPopover";

beforeAll(() => {
	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
});

afterEach(cleanup);

const EXPECTED_LABELS = [
	"Asset ID",
	"MCAP",
	"时长",
	"资产类型",
	"保留层级",
	"环境",
	"生命周期",
	"算法状态",
	"标签",
	"Owner",
	"过期时间",
	"更新时间",
] as const;

function renderOpen(
	selectedColumns: string[],
	overrides?: Partial<{
		open: boolean;
		onOpenChange: (open: boolean) => void;
		onColumnsChange: (columns: string[]) => void;
	}>,
) {
	const onOpenChange = overrides?.onOpenChange ?? vi.fn();
	const onColumnsChange = overrides?.onColumnsChange ?? vi.fn();
	const open = overrides?.open ?? true;
	const view = render(
		<ColumnsConfigPopover
			selectedColumns={selectedColumns}
			open={open}
			onOpenChange={onOpenChange}
			onColumnsChange={onColumnsChange}
		/>,
	);
	return { ...view, onOpenChange, onColumnsChange };
}

/** Ant Design Popover renders inner content with role="tooltip". */
function popoverPanel() {
	return screen.getByRole("tooltip", { name: /列配置/ });
}

describe("ColumnsConfigPopover", () => {
	it("renders trigger button with 列配置 label", () => {
		renderOpen(DEFAULT_COLUMNS, { open: false });
		expect(screen.getByRole("button", { name: /列配置/i })).toBeTruthy();
	});

	it("clicking trigger requests popover open", () => {
		const onOpenChange = vi.fn();
		renderOpen(DEFAULT_COLUMNS, { open: false, onOpenChange });
		fireEvent.click(screen.getByRole("button", { name: /列配置/i }));
		expect(onOpenChange.mock.calls[0]?.[0]).toBe(true);
	});

	it("when open, shows title 列配置 and all column checkboxes", () => {
		renderOpen(DEFAULT_COLUMNS);
		const region = within(popoverPanel());
		expect(region.getByText("列配置")).toBeTruthy();
		for (const label of EXPECTED_LABELS) {
			expect(
				region.getByRole("checkbox", { name: new RegExp(`^${label}`) }),
			).toBeTruthy();
		}
		expect(region.getByRole("button", { name: "恢复默认" })).toBeTruthy();
	});

	it("checkbox checked state mirrors selectedColumns", () => {
		const cols = ["asset_id", "mcap_file_id", "duration"];
		renderOpen(cols);
		const region = within(popoverPanel());
		expect(
			(region.getByRole("checkbox", { name: /^Asset ID/ }) as HTMLInputElement)
				.checked,
		).toBe(true);
		expect(
			(region.getByRole("checkbox", { name: /^MCAP/ }) as HTMLInputElement)
				.checked,
		).toBe(true);
		expect(
			(region.getByRole("checkbox", { name: /^时长/ }) as HTMLInputElement)
				.checked,
		).toBe(true);
		expect(
			(
				region.getByRole("checkbox", {
					name: /^资产类型/,
				}) as HTMLInputElement
			).checked,
		).toBe(false);
		expect(
			(
				region.getByRole("checkbox", {
					name: /^更新时间/,
				}) as HTMLInputElement
			).checked,
		).toBe(false);
	});

	it("unchecking a column calls onColumnsChange without that key", () => {
		const onColumnsChange = vi.fn();
		renderOpen([...DEFAULT_COLUMNS], { onColumnsChange });
		const region = within(popoverPanel());
		fireEvent.click(region.getByRole("checkbox", { name: /^Owner/ }));
		expect(onColumnsChange).toHaveBeenCalledTimes(1);
		expect(onColumnsChange).toHaveBeenCalledWith(
			DEFAULT_COLUMNS.filter((c) => c !== "owner"),
		);
	});

	it("checking a column inserts keys in ALL_COLUMNS order", () => {
		const onColumnsChange = vi.fn();
		renderOpen(["asset_id", "updated_at"], { onColumnsChange });
		const region = within(popoverPanel());
		fireEvent.click(region.getByRole("checkbox", { name: /^MCAP/ }));
		expect(onColumnsChange).toHaveBeenCalledWith([
			"asset_id",
			"mcap_file_id",
			"updated_at",
		]);
	});

	it("恢复默认 button calls onColumnsChange with DEFAULT_COLUMNS", () => {
		const onColumnsChange = vi.fn();
		renderOpen(["asset_id", "env", "tags"], { onColumnsChange });
		const region = within(popoverPanel());
		fireEvent.click(region.getByRole("button", { name: "恢复默认" }));
		expect(onColumnsChange).toHaveBeenCalledWith([...DEFAULT_COLUMNS]);
	});

	it("column toggles restore full selection after off-on cycle", () => {
		const onColumnsChange = vi.fn();
		let selected = [...ASSET_TABLE_ORDERED_COLUMN_KEYS];
		onColumnsChange.mockImplementation((cols: string[]) => {
			selected = cols;
		});

		const { rerender } = render(
			<ColumnsConfigPopover
				selectedColumns={selected}
				open
				onOpenChange={vi.fn()}
				onColumnsChange={onColumnsChange}
			/>,
		);

		const toggleLabel = "Owner";
		const region = within(popoverPanel());
		fireEvent.click(
			region.getByRole("checkbox", { name: new RegExp(`^${toggleLabel}`) }),
		);
		expect(selected).not.toContain("owner");

		rerender(
			<ColumnsConfigPopover
				selectedColumns={selected}
				open
				onOpenChange={vi.fn()}
				onColumnsChange={onColumnsChange}
			/>,
		);
		fireEvent.click(
			within(popoverPanel()).getByRole("checkbox", {
				name: new RegExp(`^${toggleLabel}`),
			}),
		);
		expect(selected).toContain("owner");
		expect(onColumnsChange).toHaveBeenCalledTimes(2);
	});
});
