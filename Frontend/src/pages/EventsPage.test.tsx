import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EventsPage from "./EventsPage";

const { listEventsMock } = vi.hoisted(() => ({
	listEventsMock: vi.fn(),
}));

vi.mock("../api/assets", () => ({
	assetsApi: {
		listEvents: listEventsMock,
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
	});

	it("renders the page title and search input", () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		expect(screen.getByText("事件流总览")).toBeTruthy();
		expect(screen.getByPlaceholderText("输入 8 位 Asset ID")).toBeTruthy();
	});

	it("renders event type filter select", () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("查看资产事件流，支持按 Asset ID 和事件类型筛选。"),
		).toBeTruthy();
	});

	it("does not call listEvents until asset id is a full canonical 8-char id", async () => {
		render(
			<MemoryRouter>
				<EventsPage />
			</MemoryRouter>,
		);
		const input = screen.getByPlaceholderText(
			"输入 8 位 Asset ID",
		) as HTMLInputElement;

		fireEvent.change(input, { target: { value: "partial" } });
		await waitFor(() => {
			expect(listEventsMock).not.toHaveBeenCalled();
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
});
