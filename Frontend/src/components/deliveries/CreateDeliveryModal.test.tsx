// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { AxiosError } from "axios";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { deliveriesApi } from "../../api/deliveries";
import CreateDeliveryModal from "./CreateDeliveryModal";

vi.mock("../../api/deliveries", () => ({
	deliveriesApi: {
		commit: vi.fn(),
	},
}));

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

afterEach(() => {
	vi.restoreAllMocks();
	cleanup();
});

function backendError() {
	return new AxiosError(
		"Request failed with status code 400",
		"ERR_BAD_REQUEST",
		undefined,
		undefined,
		{
			status: 400,
			statusText: "Bad Request",
			headers: {},
			config: {} as never,
			data: {
				code: "INVALID_ARGUMENT",
				message: "asset_ids must be 8 alphanumeric characters",
				request_id: "req-123",
				details: { asset_id: "not-real-asset-id" },
			},
		},
	);
}

describe("CreateDeliveryModal", () => {
	it("keeps the form open and shows actionable asset-id errors from backend", async () => {
		vi.spyOn(crypto, "randomUUID").mockReturnValue(
			"00000000-0000-4000-8000-000000000000",
		);
		vi.mocked(deliveriesApi.commit).mockRejectedValue(backendError());

		render(
			<CreateDeliveryModal
				open
				assetIds={[]}
				onClose={vi.fn()}
				onSuccess={vi.fn()}
			/>,
		);

		fireEvent.change(screen.getByPlaceholderText(/例如/), {
			target: { value: "not-real-asset-id" },
		});
		fireEvent.change(screen.getByPlaceholderText("请输入客户 ID"), {
			target: { value: "ux-customer" },
		});
		fireEvent.click(screen.getByRole("button", { name: /提\s*交/ }));

		await waitFor(() => {
			expect(deliveriesApi.commit).toHaveBeenCalledWith(
				expect.objectContaining({
					customer_id: "ux-customer",
					asset_ids: ["not-real-asset-id"],
				}),
				"00000000-0000-4000-8000-000000000000",
			);
		});

		expect(
			await screen.findAllByText(
				"资产 ID「not-real-asset-id」格式不正确；请改成 8 位字母或数字。",
			),
		).not.toHaveLength(0);
		expect(screen.getByText("请求 ID：req-123")).toBeTruthy();
		expect(screen.getByDisplayValue("not-real-asset-id")).toBeTruthy();
	});
});
