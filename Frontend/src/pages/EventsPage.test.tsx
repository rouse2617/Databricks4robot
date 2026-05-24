import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EventsPage from "./EventsPage";

const { listEventsMock, listGlobalEventsMock } = vi.hoisted(() => ({
	listEventsMock: vi.fn(),
	listGlobalEventsMock: vi.fn(),
}));

vi.mock("../api/assets", () => ({
	assetsApi: {
		listEvents: listEventsMock,
		listGlobalEvents: listGlobalEventsMock,
	},
}));

// Mock matchMedia for antd responsive components
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

describe("EventsPage", () => {
	beforeEach(() => {
		listEventsMock.mockReset();
		listEventsMock.mockResolvedValue({ items: [], limit: 50 });
		listGlobalEventsMock.mockReset();
		listGlobalEventsMock.mockResolvedValue({ items: [], next_cursor: null });
	});

	it("renders the page title and search input", () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		expect(screen.getByText("事件流总览")).toBeTruthy();
		expect(screen.getByPlaceholderText("输入 Asset ID 筛选")).toBeTruthy();
	});

	it("renders event type filter select", () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		expect(screen.getByText("刷新")).toBeTruthy();
	});

	it("does not call listEvents until asset id is a full canonical 8-char id", async () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		const input = screen.getByPlaceholderText(
			"输入 Asset ID 筛选",
		) as HTMLInputElement;

		await waitFor(() => {
			expect(listGlobalEventsMock).toHaveBeenCalledTimes(1);
		});
		listGlobalEventsMock.mockClear();

		fireEvent.change(input, { target: { value: "partial" } });
		await waitFor(() => {
			expect(listEventsMock).not.toHaveBeenCalled();
			expect(listGlobalEventsMock).not.toHaveBeenCalled();
		});

		const full = "7VBGimAO";
		fireEvent.change(input, { target: { value: full } });
		await waitFor(() => {
			expect(listEventsMock).toHaveBeenCalledTimes(1);
		});
		expect(listEventsMock).toHaveBeenCalledWith(
			full,
			expect.objectContaining({ event_type: undefined, limit: 100 }),
		);
	});

	it("reads asset_id from the URL query string on load", async () => {
		const id = "7VBGimAO";
		render(
			<MemoryRouter initialEntries={[`/events?asset_id=${id}`]}>
				<EventsPage />
			</MemoryRouter>,
		);
		const input = screen.getByPlaceholderText(
			"输入 Asset ID 筛选",
		) as HTMLInputElement;
		await waitFor(() => {
			expect(input.value).toBe(id);
		});
		await waitFor(() => {
			expect(listEventsMock).toHaveBeenCalledWith(
				id,
				expect.objectContaining({ limit: 100 }),
			);
		});
	});
});
