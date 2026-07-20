import axios from "axios";
import { apiClient } from "./client";

export class ApiError extends Error {
	code: string;
	status: number;
	constructor(status: number, code: string, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

// request routes through the shared axios apiClient (client.ts) — the single
// source of truth for baseURL, dev-token injection, and credentials — instead
// of a second hand-rolled fetch wrapper. The public contract is preserved
// exactly (see openspec/changes/unify-frontend-http-client):
//   - HTTP errors surface as ApiError(status, code, message) parsed from the
//     backend {code,message,error} envelope; 204 / non-JSON → undefined.
//   - Mutations carry X-Requested-With (CSRF hint).
//   - Pipeline 401s do NOT trigger the global logout: these endpoints predate
//     that behavior and callers handle 401 locally, so we opt out via
//     skipAuthRedirect (the fetch version never dispatched UNAUTHORIZED_EVENT).
//   - Network/timeout errors propagate as AxiosError (describeApiError renders
//     them as a proper network message).
export async function request<T>(
	method: string,
	path: string,
	body?: unknown,
): Promise<T> {
	const headers: Record<string, string> = {};
	if (["POST", "PUT", "DELETE"].includes(method.toUpperCase())) {
		headers["X-Requested-With"] = "XMLHttpRequest";
	}
	try {
		const res = await apiClient.request<T>({
			method,
			url: path,
			data: body,
			headers,
			skipAuthRedirect: true,
		});
		if (res.status === 204) return undefined as T;
		const contentType = String(res.headers?.["content-type"] ?? "");
		if (!contentType.includes("application/json")) {
			return undefined as T;
		}
		return res.data;
	} catch (err) {
		if (axios.isAxiosError(err) && err.response) {
			const data = err.response.data as
				| { code?: string; message?: string; error?: string }
				| undefined;
			const code = data?.code || "UNKNOWN";
			const message =
				data?.message || data?.error || `HTTP ${err.response.status}`;
			throw new ApiError(err.response.status, code, message);
		}
		// Network / timeout (no response): let the AxiosError propagate —
		// describeApiError turns it into a proper network/timeout message.
		throw err;
	}
}
