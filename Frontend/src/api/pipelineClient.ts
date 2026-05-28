const API = "/api/v1";

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

export async function request<T>(
	method: string,
	path: string,
	body?: unknown,
): Promise<T> {
	const headers: Record<string, string> = {};
	if (body) {
		headers["Content-Type"] = "application/json";
	}
	if (["POST", "PUT", "DELETE"].includes(method.toUpperCase())) {
		headers["X-Requested-With"] = "XMLHttpRequest";
	}

	const res = await fetch(`${API}${path}`, {
		method,
		headers: Object.keys(headers).length > 0 ? headers : undefined,
		body: body ? JSON.stringify(body) : undefined,
		credentials: "include",
	});
	if (!res.ok) {
		let code = "UNKNOWN";
		let message = `HTTP ${res.status}`;
		try {
			const err = await res.json();
			code = err.code || code;
			message = err.message || err.error || message;
		} catch {
			/* ignore parse errors */
		}
		throw new ApiError(res.status, code, message);
	}
	if (res.status === 204) return undefined as T;
	const contentType = res.headers.get("content-type") || "";
	if (!contentType.includes("application/json")) {
		return undefined as T;
	}
	return res.json() as Promise<T>;
}
