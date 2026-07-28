// CYB-4304: preset-agnostic tests for the BatchAssetLookup shell.
//
// These tests use a minimal in-file preset so they exercise the shell's own
// paths (paste parse, cap-disabled submit, missing/filtered-out render, CSV
// wiring, filter validation) without coupling to durations-specific details.

import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { ColumnsType } from "antd/es/table";
import { beforeEach, describe, expect, it, vi } from "vitest";
import BatchAssetLookup from "./BatchAssetLookup";
import type {
	BatchLookupPreset,
	BatchLookupRequest,
	BatchLookupResponse,
} from "./types";
import { parseIdBlob } from "./types";

// AntD components rely on matchMedia; jsdom does not implement it.
Object.defineProperty(window, "matchMedia", {
	writable: true,
	value: (query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false,
	}),
});

interface TestItem {
	input_id: string;
	label: string;
}

interface TestFilters {
	min?: number;
	max?: number;
}

const columns: ColumnsType<TestItem> = [
	{ title: "input_id", dataIndex: "input_id", key: "input_id" },
	{ title: "label", dataIndex: "label", key: "label" },
];

// Build a preset around a mock fetch so each test can assert on the call.
function makePreset(
	fetchImpl: (
		req: BatchLookupRequest<TestFilters>,
	) => Promise<BatchLookupResponse<TestItem>>,
	overrides: Partial<BatchLookupPreset<TestItem, TestFilters>> = {},
): BatchLookupPreset<TestItem, TestFilters> {
	return {
		key: "test",
		placeholder: "paste ids…",
		maxIds: 5,
		fetch: fetchImpl,
		columns,
		csv: {
			filename: (n) => `test-${n}.csv`,
			header: ["input_id", "label"],
			row: (it) => [it.input_id, it.label],
		},
		...overrides,
	};
}

describe("parseIdBlob (shell helper)", () => {
	it("splits on whitespace/commas, trims, dedupes, reports duplicates", () => {
		const result = parseIdBlob("id1, id2\n id3\n\nid1 id4,,id2");
		expect(result.ids).toEqual(["id1", "id2", "id3", "id4"]);
		expect(result.duplicates).toBe(2);
	});

	it("returns empty on empty / whitespace-only input", () => {
		expect(parseIdBlob("")).toEqual({ ids: [], duplicates: 0 });
		expect(parseIdBlob("   \n \t  ")).toEqual({ ids: [], duplicates: 0 });
	});
});

describe("BatchAssetLookup shell", () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it("shows a paste-count hint after parsing (dedupes and lists duplicates)", () => {
		const preset = makePreset(async () => ({
			items: [],
			missing_ids: [],
			stats: {},
		}));
		render(<BatchAssetLookup preset={preset} />);
		const textarea = screen.getByTestId("ids-textarea") as HTMLTextAreaElement;
		fireEvent.change(textarea, {
			target: { value: "a1, a2\na1\n\na3" },
		});
		expect(screen.getByText(/3 ids/)).toBeTruthy();
		expect(screen.getByText(/1 duplicates ignored/)).toBeTruthy();
	});

	it("submit is disabled with 0 parsed ids", () => {
		const preset = makePreset(async () => ({
			items: [],
			missing_ids: [],
			stats: {},
		}));
		render(<BatchAssetLookup preset={preset} />);
		const button = screen.getByTestId("submit-button") as HTMLButtonElement;
		expect(button.disabled).toBe(true);
	});

	it("submit is disabled when parsed ids exceed preset.maxIds", () => {
		const preset = makePreset(async () => ({
			items: [],
			missing_ids: [],
			stats: {},
		}));
		render(<BatchAssetLookup preset={preset} />);
		// preset.maxIds = 5 → paste 6 unique ids to trip the cap.
		fireEvent.change(
			screen.getByTestId("ids-textarea") as HTMLTextAreaElement,
			{ target: { value: "a b c d e f" } },
		);
		expect(
			(screen.getByTestId("submit-button") as HTMLButtonElement).disabled,
		).toBe(true);
		// The count-hint switches to the danger variant with the 超过上限 tag.
		expect(screen.getByText(/超过上限 5/)).toBeTruthy();
	});

	it("renders the missing-ids Collapse even when items is empty", async () => {
		const fetchMock = vi.fn().mockResolvedValue({
			items: [],
			missing_ids: ["zzzz", "yyyy"],
			stats: {},
		} satisfies BatchLookupResponse<TestItem>);
		const preset = makePreset(fetchMock);
		render(<BatchAssetLookup preset={preset} />);
		fireEvent.change(
			screen.getByTestId("ids-textarea") as HTMLTextAreaElement,
			{ target: { value: "zzzz\nyyyy" } },
		);
		fireEvent.click(screen.getByTestId("submit-button"));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

		expect(await screen.findByText(/未匹配的 ID \(2\)/)).toBeTruthy();
	});

	it("renders the results Table + drives CSV filename via preset.csv.filename", async () => {
		const filenameSpy = vi.fn((n: number) => `test-${n}.csv`);
		const fetchMock = vi.fn().mockResolvedValue({
			items: [
				{ input_id: "a", label: "AAA" },
				{ input_id: "b", label: "BBB" },
			],
			missing_ids: [],
			stats: {},
		} satisfies BatchLookupResponse<TestItem>);
		const preset = makePreset(fetchMock, {
			csv: {
				filename: filenameSpy,
				header: ["input_id", "label"],
				row: (it) => [it.input_id, it.label],
			},
		});

		vi.spyOn(URL, "createObjectURL").mockReturnValue(
			"blob:mock" as unknown as string,
		);
		vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
		const clickSpy = vi
			.spyOn(HTMLAnchorElement.prototype, "click")
			.mockImplementation(() => {});

		render(<BatchAssetLookup preset={preset} />);
		fireEvent.change(
			screen.getByTestId("ids-textarea") as HTMLTextAreaElement,
			{ target: { value: "a b" } },
		);
		fireEvent.click(screen.getByTestId("submit-button"));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));

		const table = await screen.findByTestId("results-table");
		const rows = within(table).getAllByRole("row");
		expect(rows.length).toBe(1 + 2); // header + 2 items

		fireEvent.click(screen.getByTestId("export-csv-button"));
		expect(filenameSpy).toHaveBeenCalledWith(2);
		expect(clickSpy).toHaveBeenCalledTimes(1);
		const anchor = clickSpy.mock.instances[0] as HTMLAnchorElement;
		expect(anchor.download).toBe("test-2.csv");
	});

	it("disables submit and renders the filter error when validate returns a message", () => {
		const preset = makePreset(
			async () => ({ items: [], missing_ids: [], stats: {} }),
			{
				filters: {
					initial: { min: 100, max: 50 },
					render: () => null,
					validate: (f) =>
						f.min != null && f.max != null && f.min > f.max
							? "start > end"
							: null,
				},
			},
		);
		render(<BatchAssetLookup preset={preset} />);
		fireEvent.change(
			screen.getByTestId("ids-textarea") as HTMLTextAreaElement,
			{ target: { value: "a" } },
		);
		expect(screen.getByTestId("filter-error").textContent).toBe("start > end");
		expect(
			(screen.getByTestId("submit-button") as HTMLButtonElement).disabled,
		).toBe(true);
	});
});
