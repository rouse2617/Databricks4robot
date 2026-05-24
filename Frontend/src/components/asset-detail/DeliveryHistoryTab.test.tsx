import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DeliveryHistoryTab from "./DeliveryHistoryTab";

const { listDeliveriesMock, getDeliveryMock, globalDeliveriesListMock } =
	vi.hoisted(() => ({
		listDeliveriesMock: vi.fn(),
		getDeliveryMock: vi.fn(),
		globalDeliveriesListMock: vi.fn(),
	}));

vi.mock("../../api/assets", () => ({
	assetsApi: {
		listDeliveries: listDeliveriesMock,
	},
}));

vi.mock("../../api/deliveries", () => ({
	deliveriesApi: {
		get: getDeliveryMock,
		list: globalDeliveriesListMock,
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

describe("DeliveryHistoryTab", () => {
	beforeEach(() => {
		listDeliveriesMock.mockReset();
		getDeliveryMock.mockReset();
		globalDeliveriesListMock.mockReset();
	});

	it("loads delivery details only for ids returned by the asset-scoped endpoint", async () => {
		listDeliveriesMock.mockResolvedValue({
			items: ["del_1", "del_2"],
			asset_id: "asset_1",
			page: 1,
			page_size: 100,
		});
		getDeliveryMock.mockImplementation((id: string) =>
			Promise.resolve({
				delivery_id: id,
				customer_id: `customer_${id}`,
				status: "delivered",
				delivered_at: "2026-05-23T00:00:00Z",
				asset_count: 1,
				owner: "qa",
				created_at: "2026-05-23T00:00:00Z",
				updated_at: "2026-05-23T00:00:00Z",
				version: 1,
			}),
		);

		render(<DeliveryHistoryTab assetId="asset_1" />);

		await waitFor(() => {
			expect(screen.getByText("del_1")).toBeTruthy();
			expect(screen.getByText("del_2")).toBeTruthy();
		});
		expect(listDeliveriesMock).toHaveBeenCalledTimes(1);
		expect(listDeliveriesMock).toHaveBeenCalledWith("asset_1", 1, 100);
		expect(getDeliveryMock).toHaveBeenCalledTimes(2);
		expect(globalDeliveriesListMock).not.toHaveBeenCalled();
	});

	it("settles to an empty state for assets without delivery ids", async () => {
		listDeliveriesMock.mockResolvedValue({
			items: [],
			asset_id: "asset_empty",
			page: 1,
			page_size: 100,
		});

		render(<DeliveryHistoryTab assetId="asset_empty" />);

		await waitFor(() => {
			expect(screen.getByText("该资产尚未交付")).toBeTruthy();
		});
		expect(getDeliveryMock).not.toHaveBeenCalled();
		expect(globalDeliveriesListMock).not.toHaveBeenCalled();
	});
});
