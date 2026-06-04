import { request } from "./pipelineClient";

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

export interface PipelineComponentResources {
	cpu?: string;
	memory?: string;
	disk?: string;
	gpu?: string;
	computeTier?: string;
	type?: string;
	command?: string[];
	args?: string[];
	env?: Record<string, string> | EnvVarDef[];
	[key: string]: unknown;
}

export type PipelineComponentType =
	| "container"
	| "script"
	| "resource"
	| "suspend";

/** Backend PipelineComponent model shape. */
export interface PipelineComponentAPI {
	id: string;
	name: string;
	type: PipelineComponentType;
	description: string;
	image: string;
	tag: string;
	source: string;
	command?: string[];
	args?: string[];
	env?: Record<string, string>;
	inputPorts: PortDef[];
	outputPorts: PortDef[];
	resources?: PipelineComponentResources;
	envVars?: EnvVarDef[];
	createdAt: string;
	updatedAt: string;
}

export type PipelineComponentPayload = Pick<
	PipelineComponentAPI,
	"name" | "type" | "image"
> &
	Partial<
		Pick<
			PipelineComponentAPI,
			| "description"
			| "tag"
			| "source"
			| "command"
			| "args"
			| "env"
			| "inputPorts"
			| "outputPorts"
			| "resources"
			| "envVars"
		>
	>;

/** List all registered pipeline components. */
export function listComponents(params?: {
	q?: string;
	source?: string;
}): Promise<{ items: PipelineComponentAPI[] }> {
	const query = new URLSearchParams();
	if (params?.q) query.set("q", params.q);
	if (params?.source) query.set("source", params.source);
	const suffix = query.toString() ? `?${query.toString()}` : "";
	return request<{ items: PipelineComponentAPI[] }>(
		"GET",
		`/pipeline-components${suffix}`,
	);
}

/** Create a new pipeline component. */
export function createComponent(
	pc: PipelineComponentPayload,
): Promise<PipelineComponentAPI> {
	return request<PipelineComponentAPI>("POST", "/pipeline-components", pc);
}

/** Update an existing pipeline component. */
export function updateComponent(
	id: string,
	pc: PipelineComponentPayload,
): Promise<PipelineComponentAPI> {
	return request<PipelineComponentAPI>(
		"PUT",
		`/pipeline-components/${encodeURIComponent(id)}`,
		pc,
	);
}

/** Delete a pipeline component. */
export function deleteComponent(id: string): Promise<void> {
	return request<void>(
		"DELETE",
		`/pipeline-components/${encodeURIComponent(id)}`,
	);
}
