const API = "/api/v1";

export interface PortDef {
	name: string;
	type: string;
	desc?: string;
	default_value?: string;
}

export interface EnvVarDef {
	name: string;
	value?: string;
}

/** Backend PipelineComponent model shape. */
export interface PipelineComponentAPI {
	id: string;
	name: string;
	description: string;
	image: string;
	tag: string;
	source: string;
	inputPorts: PortDef[];
	outputPorts: PortDef[];
	resources?: Record<string, unknown>;
	envVars?: EnvVarDef[];
	createdAt: string;
	updatedAt: string;
}

class ApiError extends Error {
	code: string;
	status: number;
	constructor(status: number, code: string, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

async function request<T>(
	method: string,
	path: string,
	body?: unknown,
): Promise<T> {
	const res = await fetch(`${API}${path}`, {
		method,
		headers: body ? { "Content-Type": "application/json" } : undefined,
		body: body ? JSON.stringify(body) : undefined,
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

/** List all registered pipeline components. */
export function listComponents(): Promise<{ items: PipelineComponentAPI[] }> {
	return request("GET", "/components");
}

/** Create a new pipeline component. */
export function createComponent(
	pc: Omit<PipelineComponentAPI, "id" | "createdAt" | "updatedAt">,
): Promise<PipelineComponentAPI> {
	return request("POST", "/components", pc);
}

/** Update an existing pipeline component. */
export function updateComponent(
	id: string,
	pc: Partial<PipelineComponentAPI>,
): Promise<PipelineComponentAPI> {
	return request("PUT", `/components/${encodeURIComponent(id)}`, pc);
}

/** Delete a pipeline component. */
export function deleteComponent(id: string): Promise<void> {
	return request("DELETE", `/components/${encodeURIComponent(id)}`);
}
