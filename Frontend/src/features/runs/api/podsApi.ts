import { request } from "../../../api/pipelineClient";

export interface RunPodItem {
	nodeId?: string;
	podName: string;
	displayName?: string;
	phase?: string;
	message?: string;
	podIp?: string;
	restartCount?: number;
	conditions?: unknown[];
	containers?: unknown[];
	events?: unknown[];
}

export function getRunPods(runId: string): Promise<{ items: RunPodItem[] }> {
	return request<{ items: RunPodItem[] }>(
		"GET",
		`/runs/${encodeURIComponent(runId)}/pods`,
	);
}

export function createComponentBuild(body: {
	componentId?: string;
	componentName: string;
	repoUrl: string;
	gitRef?: string;
	dockerfile?: string;
	buildContext?: string;
	imageRepository?: string;
	imageTag?: string;
}) {
	return request("POST", "/runs/component-build", body);
}

export function createRAGBuild(body: {
	knowledgeBaseId: string;
	embeddingModel?: string;
	vectorIndexName?: string;
	releaseVersion?: string;
	datasource?: Record<string, unknown>;
}) {
	return request("POST", "/runs/rag-build", body);
}
