import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AlgoRunsPage from "./AlgoRunsPage";

const { algoRunsListMock, algoRegistryListMock } = vi.hoisted(() => ({
	algoRunsListMock: vi.fn(),
	algoRegistryListMock: vi.fn(),
}));

vi.mock("../api/algoRuns", () => ({
	algoRunsApi: {
		list: algoRunsListMock,
	},
}));

vi.mock("../api/algoRegistry", () => ({
	algoRegistryApi: {
		list: algoRegistryListMock,
	},
}));

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

describe("AlgoRunsPage", () => {
	beforeEach(() => {
		algoRunsListMock.mockReset();
		algoRegistryListMock.mockReset();
		algoRegistryListMock.mockResolvedValue([]);
	});

	it("shows a visible error when loading algo runs fails", async () => {
		algoRunsListMock.mockRejectedValueOnce(new Error("backend unavailable"));

		render(
			<MemoryRouter>
				<AlgoRunsPage />
			</MemoryRouter>,
		);

		expect(await screen.findByText("加载算法运行记录失败")).toBeTruthy();
		expect(await screen.findByText("backend unavailable")).toBeTruthy();
	});
});
