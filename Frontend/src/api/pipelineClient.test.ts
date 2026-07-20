import { afterEach, describe, expect, it, vi } from "vitest";
import { apiClient } from "./client";
import { ApiError, request } from "./pipelineClient";

vi.mock("./client", () => ({
	apiClient: { request: vi.fn() },
}));

const mockRequest = vi.mocked(apiClient.request);

// A minimal object that axios.isAxiosError() accepts (it checks
// isObject(x) && x.isAxiosError === true).
function axiosError(response?: { status: number; data: unknown }) {
	return { isAxiosError: true, message: "Network Error", response };
}

afterEach(() => {
	vi.clearAllMocks();
});

describe("pipelineClient.request (axios-backed)", () => {
	it("returns parsed JSON on success", async () => {
		mockRequest.mockResolvedValue({
			status: 200,
			data: { id: "x", n: 1 },
			headers: { "content-type": "application/json" },
		});
		await expect(request("GET", "/things")).resolves.toEqual({ id: "x", n: 1 });
	});

	it("resolves 204 to undefined", async () => {
		mockRequest.mockResolvedValue({ status: 204, data: "", headers: {} });
		await expect(request("DELETE", "/things/1")).resolves.toBeUndefined();
	});

	it("resolves a non-JSON body to undefined", async () => {
		mockRequest.mockResolvedValue({
			status: 200,
			data: "<html>nope</html>",
			headers: { "content-type": "text/html" },
		});
		await expect(request("GET", "/things")).resolves.toBeUndefined();
	});

	it("maps an HTTP error envelope to ApiError(status, code, message)", async () => {
		mockRequest.mockRejectedValue(
			axiosError({ status: 409, data: { code: "DUP", message: "dup hash" } }),
		);
		const err = await request("POST", "/things").catch((e) => e);
		expect(err).toBeInstanceOf(ApiError);
		expect(err.status).toBe(409);
		expect(err.code).toBe("DUP");
		expect(err.message).toBe("dup hash");
	});

	it("falls back to error/HTTP-status when the envelope is sparse", async () => {
		mockRequest.mockRejectedValue(
			axiosError({ status: 400, data: { error: "bad input" } }),
		);
		const err1 = await request("POST", "/things").catch((e) => e);
		expect(err1.message).toBe("bad input");

		mockRequest.mockRejectedValue(axiosError({ status: 500, data: null }));
		const err2 = await request("GET", "/things").catch((e) => e);
		expect(err2).toBeInstanceOf(ApiError);
		expect(err2.code).toBe("UNKNOWN");
		expect(err2.message).toBe("HTTP 500");
	});

	it("re-throws network/timeout errors unchanged (not ApiError)", async () => {
		mockRequest.mockRejectedValue(axiosError(undefined));
		const err = await request("GET", "/things").catch((e) => e);
		expect(err).not.toBeInstanceOf(ApiError);
		expect(err.isAxiosError).toBe(true);
	});

	it("sends X-Requested-With + skipAuthRedirect on mutations, not on GET", async () => {
		mockRequest.mockResolvedValue({
			status: 200,
			data: {},
			headers: { "content-type": "application/json" },
		});

		await request("POST", "/things", { a: 1 });
		expect(mockRequest).toHaveBeenLastCalledWith(
			expect.objectContaining({
				method: "POST",
				url: "/things",
				data: { a: 1 },
				skipAuthRedirect: true,
				headers: { "X-Requested-With": "XMLHttpRequest" },
			}),
		);

		await request("GET", "/things");
		expect(mockRequest).toHaveBeenLastCalledWith(
			expect.objectContaining({ method: "GET", headers: {} }),
		);
	});
});
