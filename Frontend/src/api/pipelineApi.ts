const API = "/api/v1";

export interface PipelineTemplate {
	id: string;
	name: string;
	pipeline: unknown;
	nodeCount: number;
	createdAt: string;
}

export interface Deployment {
	id: string;
	pipelineName: string;
	workflowName: string;
	status: string;
	nodeCount: number;
	createdAt: string;
	finishedAt?: string;
	manifest?: string;
	pipelineJSON?: unknown;
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

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
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

export function listPipelines(): Promise<PipelineTemplate[]> {
	return request<{ items: PipelineTemplate[] }>("GET", "/pipelines").then((r) => r.items);
}

export function savePipeline(name: string, pipeline: unknown): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("POST", "/pipelines", { name, pipeline });
}

export function deletePipeline(id: string): Promise<void> {
	return request<void>("DELETE", `/pipelines/${id}`);
}

export function deploy(pipeline: unknown, name?: string): Promise<Deployment> {
	return request<Deployment>("POST", "/deploy", { pipeline, name });
}

export function deployTemplate(templateId: string): Promise<Deployment> {
	return request<Deployment>("POST", `/deploy/template/${templateId}`);
}

export function listDeployments(): Promise<Deployment[]> {
	return request<{ items: Deployment[] }>("GET", "/deployments").then((r) => r.items);
}

export function deleteDeployment(id: string): Promise<void> {
	return request<void>("DELETE", `/deployments/${id}`);
}
