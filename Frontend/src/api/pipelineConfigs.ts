import { apiClient } from "./client";

export type PipelineConfigLifecycle = "draft" | "ready" | "deprecated";
export type PipelineConfigFileType = "yaml" | "json";

export interface PipelineConfigVersion {
	id: string;
	configId: string;
	version: number;
	status: PipelineConfigLifecycle;
	content?: string;
	contentSha256: string;
	contentSizeBytes: number;
	summary: string;
	author: string;
	createdAt: string;
}

export interface PipelineConfig {
	id: string;
	name: string;
	description: string;
	owner: string;
	scope: string;
	tags: string[];
	fileType: PipelineConfigFileType;
	lifecycle: PipelineConfigLifecycle;
	currentVersion: number;
	versionCount: number;
	versions?: PipelineConfigVersion[];
	createdAt: string;
	updatedAt: string;
}

export interface PipelineConfigListResponse {
	items: PipelineConfig[];
}

export interface PipelineConfigCreateRequest {
	name: string;
	description?: string;
	tags?: string[];
	fileType?: PipelineConfigFileType;
	lifecycle?: PipelineConfigLifecycle;
	scope?: string;
	content: string;
	summary?: string;
}

export interface PipelineConfigUpdateRequest {
	name: string;
	description?: string;
	tags?: string[];
	fileType?: PipelineConfigFileType;
	lifecycle?: PipelineConfigLifecycle;
}

export interface PipelineConfigVersionCreateRequest {
	status?: PipelineConfigLifecycle;
	content: string;
	summary?: string;
}

export const pipelineConfigApi = {
	list: () =>
		apiClient
			.get<PipelineConfigListResponse>("/pipeline-configs")
			.then((r) => r.data),

	get: (id: string) =>
		apiClient
			.get<PipelineConfig>(`/pipeline-configs/${id}`)
			.then((r) => r.data),

	create: (payload: PipelineConfigCreateRequest) =>
		apiClient
			.post<PipelineConfig>("/pipeline-configs", payload)
			.then((r) => r.data),

	update: (id: string, payload: PipelineConfigUpdateRequest) =>
		apiClient
			.put<PipelineConfig>(`/pipeline-configs/${id}`, payload)
			.then((r) => r.data),

	createVersion: (id: string, payload: PipelineConfigVersionCreateRequest) =>
		apiClient
			.post<PipelineConfigVersion>(`/pipeline-configs/${id}/versions`, payload)
			.then((r) => r.data),

	getVersion: (id: string, version: number) =>
		apiClient
			.get<PipelineConfigVersion>(`/pipeline-configs/${id}/versions/${version}`)
			.then((r) => r.data),

	updateVersionStatus: (id: string, version: number, status: PipelineConfigLifecycle) =>
		apiClient
			.put<PipelineConfigVersion>(`/pipeline-configs/${id}/versions/${version}/status`, { status })
			.then((r) => r.data),

	updateVersionContent: (id: string, version: number, content: string, summary: string) =>
		apiClient
			.put<PipelineConfigVersion>(`/pipeline-configs/${id}/versions/${version}`, { content, summary })
			.then((r) => r.data),

	deprecate: (id: string) =>
		apiClient
			.post<PipelineConfig>(`/pipeline-configs/${id}/deprecate`, {})
			.then((r) => r.data),
};
