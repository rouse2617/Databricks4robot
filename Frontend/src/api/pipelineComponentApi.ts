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

export interface ComponentReleaseRuntimeSnapshot {
	image: string;
	command?: string[];
	args?: string[];
	env?: Record<string, string>;
	inputPorts: PortDef[];
	outputPorts: PortDef[];
	resources?: Record<string, unknown>;
}

export interface PipelineComponentReleaseAPI {
	id: string;
	imageUid?: string;
	componentId: string;
	taskName: string;
	taskPath?: string;
	displayName?: string;
	owner?: string;
	releaseLabel: string;
	channel: string;
	sourceRepo?: string;
	sourceRef?: string;
	sourceRefType?: string;
	sourceCommit?: string;
	buildId?: string;
	imageRepo?: string;
	imageTag?: string;
	imageDigest?: string;
	runtimeImage: string;
	status: string;
	selectable: boolean;
	validationStatus: string;
	validationErrors?: string[];
	runtimeSnapshot: ComponentReleaseRuntimeSnapshot;
	technicalMetadata?: Record<string, unknown>;
	createdAt: string;
	updatedAt: string;
	lastSyncedAt?: string;
}

export type PipelineComponentReleasePayload = Partial<
	Omit<
		PipelineComponentReleaseAPI,
		| "id"
		| "status"
		| "selectable"
		| "validationStatus"
		| "validationErrors"
		| "createdAt"
		| "updatedAt"
		| "lastSyncedAt"
	>
> & {
	componentId?: string;
	taskName?: string;
	releaseLabel?: string;
	runtimeImage?: string;
};

export interface ComponentReleaseIngestSource {
	provider?: string;
	repo?: string;
	ref?: string;
	refType?: string;
	commit?: string;
	buildId?: string;
	trigger?: string;
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

/** List generated, validated component releases. */
export function listComponentReleases(params?: {
	q?: string;
	componentId?: string;
	taskName?: string;
	status?: string;
	channel?: string;
	selectable?: boolean;
}): Promise<{ items: PipelineComponentReleaseAPI[] }> {
	const query = new URLSearchParams();
	if (params?.q) query.set("q", params.q);
	if (params?.componentId) query.set("componentId", params.componentId);
	if (params?.taskName) query.set("taskName", params.taskName);
	if (params?.status) query.set("status", params.status);
	if (params?.channel) query.set("channel", params.channel);
	if (params?.selectable !== undefined) {
		query.set("selectable", String(params.selectable));
	}
	const suffix = query.toString() ? `?${query.toString()}` : "";
	return request<{ items: PipelineComponentReleaseAPI[] }>(
		"GET",
		`/pipeline-component-releases${suffix}`,
	);
}

/** Get one generated component release. */
export function getComponentRelease(
	id: string,
): Promise<PipelineComponentReleaseAPI> {
	return request<PipelineComponentReleaseAPI>(
		"GET",
		`/pipeline-component-releases/${encodeURIComponent(id)}`,
	);
}

/** Sync generated release records from CI/platform tooling. */
export function syncComponentReleases(
	items: PipelineComponentReleasePayload[],
	source?: ComponentReleaseIngestSource,
): Promise<{
	items: PipelineComponentReleaseAPI[];
}> {
	const body = source ? { source, items } : { items };
	return request<{ items: PipelineComponentReleaseAPI[] }>(
		"POST",
		"/pipeline-component-releases/sync",
		body,
	);
}
