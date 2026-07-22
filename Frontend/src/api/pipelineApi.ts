import type { Pipeline } from "../components/pipeline/types";
import { request } from "./pipelineClient";

export interface PipelineTemplate {
	id: string;
	name: string;
	version: number;
	versionCount?: number;
	activeVersion?: number;
	scope?: string;
	owner?: string;
	pipeline: Pipeline;
	nodeCount: number;
	createdAt: string;
	updatedAt?: string;
}

export interface Deployment {
	id: string;
	templateId?: string;
	templateVersion?: number;
	pipelineName: string;
	workflowName: string;
	status: string;
	nodeCount: number;
	scope?: string;
	owner?: string;
	assetIds?: string[];
	assetCount?: number;
	executionTarget?: ExecutionTarget;
	createdAt: string;
	finishedAt?: string;
	manifest?: string;
	pipelineJSON?: Pipeline;
}

export interface PipelineRunNodeProgress {
	focusNodeId?: string;
	focusNodeName?: string;
	focusStatus?: string;
	message?: string;
	label?: string;
	parallelRunning?: number;
}

export interface PipelineRunNode {
	id: string;
	runId: string;
	pipelineNodeId: string;
	argoNodeId?: string;
	argoNodeName?: string;
	displayName?: string;
	templateName?: string;
	type?: string;
	phase?: string;
	message?: string;
	podName?: string;
	hostNodeName?: string;
	children?: string[];
	inputs?: Record<string, unknown>;
	outputs?: Record<string, unknown>;
	resourcesDuration?: Record<string, number>;
	logRef?: string;
	estimatedCostUsd?: number;
	startedAt?: string;
	finishedAt?: string;
	updatedAt?: string;
	createdAt?: string;
}

export interface PipelineRun extends Deployment {
	templateName?: string;
	executionTargetId?: string;
	targetSnapshot?: Record<string, unknown>;
	argoNamespace?: string;
	argoWorkflowUid?: string;
	message?: string;
	failureReason?: string;
	blockingReason?: string;
	blockingMessage?: string;
	noAssetRun?: boolean;
	nodes?: PipelineRunNode[];
	ledgerState?: string;
	startedAt?: string;
	updatedAt?: string;
	totalEstimatedCost?: number | null;
	batchJobId?: string;
	nodeProgress?: PipelineRunNodeProgress;
	/** Source video duration in seconds (from video_durations; may be absent). */
	videoDurationSec?: number;
}

export interface PipelineRunListResponse {
	items: PipelineRun[];
	total: number;
	page?: number;
	pageSize?: number;
}

export interface PipelineRunEvent {
	id: string;
	runId: string;
	workflowName?: string;
	eventType: string;
	subjectType: "run" | "workflow" | "node" | "pod" | string;
	subjectId: string;
	status?: string;
	message?: string;
	reason?: string;
	payload?: Record<string, unknown>;
	sequence: number;
	occurredAt: string;
	observedAt: string;
	createdAt: string;
}

export interface PipelineRunEventsResponse {
	items: PipelineRunEvent[];
	nextCursor?: number;
	total: number;
}

export interface PipelineRunWatcherState {
	id: string;
	lastSyncedAt?: string;
	lastScanStartedAt?: string;
	lastScanFinishedAt?: string;
	lastSuccessAt?: string;
	lastErrorAt?: string;
	activeScanLimit: number;
	lastSyncedRunCount: number;
	consecutiveFailures: number;
	totalScans: number;
	totalErrors: number;
	scanLagSeconds?: number;
	lastError?: string;
	healthy: boolean;
	stale: boolean;
	updatedAt?: string;
}

export interface PipelineRunAssetNode {
	id: string;
	runId: string;
	assetId: string;
	pipelineNodeId: string;
	argoNodeId?: string;
	displayName?: string;
	status?: string;
	message?: string;
	podName?: string;
	logRef?: string;
	estimatedCostUsd?: number;
	costSource: string;
	startedAt?: string;
	finishedAt?: string;
	updatedAt: string;
}

export interface PipelineRunAssetNodeListResponse {
	items: PipelineRunAssetNode[];
	nextCursor?: string;
	total: number;
	summary: {
		assetCount: number;
		nodeCount: number;
		statuses: Record<string, number>;
		totalEstimatedCostUsd?: number;
		costSource: string;
	};
}

export interface PipelineRunCostSummary {
	runId: string;
	totalEstimatedCostUsd?: number;
	costSource: string;
	nodeSummaries: Array<{
		nodeId: string;
		displayName?: string;
		status?: string;
		podCount: number;
		estimatedCostUsd?: number;
		costSource: string;
		durationSeconds?: number;
	}>;
	assetNodeSummaries: Array<{
		assetId: string;
		nodeId: string;
		displayName?: string;
		status?: string;
		estimatedCostUsd?: number;
		costSource: string;
	}>;
	generatedAt: string;
}

export interface TargetToleration {
	key: string;
	operator: "Equal" | "Exists";
	value?: string;
	effect: "NoSchedule" | "NoExecute" | "PreferNoSchedule";
}

// TargetScheduling is the pool's scheduling config (CYB-3486 pool.3), stored
// under resource_defaults.scheduling (JSONB). The backend injects every field
// onto each Argo Workflow pod verbatim — it hardcodes NO scheduler / quota
// semantics. A Koordinator ElasticQuota pool is just a config that happens to
// carry schedulerName="koord-scheduler" plus the EQ pod label; the frontend
// hardcodes neither. All fields optional; empty = cluster / K8s defaults.
export interface TargetScheduling {
	// Pod scheduler → wf.Spec.SchedulerName. Empty → cluster default scheduler
	// (scheduler-agnostic pool).
	schedulerName?: string;
	// PriorityClass applied to every pod → wf.Spec.PodPriorityClassName. Empty
	// → K8s global default. Must already exist in the target cluster.
	priorityClassName?: string;
	// Pod labels merged onto every workflow pod (e.g. a Koordinator ElasticQuota
	// label). The admin supplies both key and value.
	podLabels?: Record<string, string>;
	// Pod annotations merged onto every workflow pod.
	podAnnotations?: Record<string, string>;
	// nodeSelector / tolerations are ALSO read from here by the backend
	// (scheduling.go accepts nodeSelector, nodeSelectors, templateNodeSelector,
	// tolerations, templateTolerations as aliases at both top-level and nested).
	// PoolManager edits via top-level templateNodeSelector / templateTolerations,
	// but the display renderer reads from both locations to stay in sync with
	// the backend's alias-aware resolution.
	nodeSelector?: Record<string, string>;
	nodeSelectors?: Record<string, string>;
	templateNodeSelector?: Record<string, string>;
	tolerations?: TargetToleration[];
	templateTolerations?: TargetToleration[];
}

export interface TargetResourceDefaults {
	computeTier?: string;
	templateTolerations?: TargetToleration[];
	templateNodeSelector?: Record<string, string>;
	// Pool scheduling directives (scheduler, priorityclass, pod labels/annotations).
	scheduling?: TargetScheduling;
	// Other fields (terminal config, etc.) are preserved verbatim on PUT.
	[key: string]: unknown;
}

export interface ExecutionTarget {
	id: string;
	name: string;
	// clusterId is the FK to clusters.id (CYB-3425). Optional to keep
	// backwards compatibility with old responses; new targets always carry it.
	clusterId?: string;
	// cluster is the legacy free-text field; still returned by the backend for
	// display. New callers should populate clusterId instead.
	cluster: string;
	namespace: string;
	serviceAccount?: string;
	argoServerConfigured: boolean;
	argoInsecureSkipTls?: boolean;
	status: "available" | "unavailable";
	enabled?: boolean;
	isDefault: boolean;
	description?: string;
	// resourceDefaults carries the pool's scheduling config under `.scheduling`
	// (CYB-3486 pool.3). The dedicated elastic_quota_name / priority_class_name
	// columns were dropped in #439 — do not reintroduce top-level fields for them.
	resourceDefaults?: TargetResourceDefaults;
	quotaPolicy?: Record<string, unknown>;
	labels?: Record<string, string>;
	createdAt?: string;
	updatedAt?: string;
}

export interface RuntimeSecretMountResource {
	id: string;
	name: string;
	description?: string;
	kind: "secretProviderClass";
	secretProviderClass: string;
	defaultMountPath: string;
	readOnly: boolean;
	targetIds?: string[];
}

export interface RuntimeStorageMountResource {
	id: string;
	name: string;
	description?: string;
	kind: "pvc" | "emptyDir";
	pvcName?: string;
	defaultMountPath: string;
	readOnly: boolean;
	allowWrite: boolean;
	targetIds?: string[];
}

export interface RuntimeMountCatalog {
	secrets: RuntimeSecretMountResource[];
	storage: RuntimeStorageMountResource[];
}

export type DeployConfigSourceMode = "saved" | "upload" | "inline";

export interface DeployConfigSelectionBase {
	mode: DeployConfigSourceMode;
	fileName: string;
	mountPath: string;
	targetFilename: string;
}

export interface SavedDeployConfigSelection extends DeployConfigSelectionBase {
	mode: "saved";
	configId: string;
	version: number;
}

export interface UploadDeployConfigSelection extends DeployConfigSelectionBase {
	mode: "upload";
	content: string;
}

export interface InlineDeployConfigSelection extends DeployConfigSelectionBase {
	mode: "inline";
	content: string;
}

export type DeployConfigSelection =
	| SavedDeployConfigSelection
	| UploadDeployConfigSelection
	| InlineDeployConfigSelection;

export function listRuntimeMounts(): Promise<RuntimeMountCatalog> {
	return request<Partial<RuntimeMountCatalog>>(
		"GET",
		"/pipeline/runtime-mounts",
	).then((raw) => ({
		secrets: raw.secrets ?? [],
		storage: raw.storage ?? [],
	}));
}

export function previewDeploy(
	pipeline: Pipeline,
): Promise<{ manifest: string }> {
	return request<{ manifest?: string }>("POST", "/deploy?dryRun=true", {
		pipeline,
	}).then((resp) => ({ manifest: resp.manifest || "" }));
}

export function listPipelines(
	params: ListPipelinesParams = {},
): Promise<ListPipelinesResponse> {
	const page = params.page ?? 1;
	const pageSize = params.pageSize ?? 20;
	const qs = new URLSearchParams();
	if (params.page) qs.set("page", String(params.page));
	if (params.pageSize) qs.set("page_size", String(params.pageSize));
	if (params.q) qs.set("q", params.q);
	if (params.scope) qs.set("scope", params.scope);
	if (params.sort) qs.set("sort", params.sort);
	if (params.excludeAutoDrafts) qs.set("exclude_auto_drafts", "true");
	const query = qs.toString();
	return request<
		Partial<ListPipelinesResponse> & {
			items?: PipelineTemplate[];
			page_size?: number;
		}
	>("GET", `/pipelines${query ? `?${query}` : ""}`).then((raw) => {
		const items = raw.items ?? [];
		const resolvedPageSize = raw.pageSize ?? raw.page_size ?? pageSize;
		const hasServerTotal = typeof raw.total === "number";
		const total = typeof raw.total === "number" ? raw.total : items.length;
		let normalizedItems = items;
		if (!hasServerTotal && items.length > resolvedPageSize) {
			const start = (page - 1) * resolvedPageSize;
			normalizedItems = items.slice(start, start + resolvedPageSize);
		}
		return {
			items: normalizedItems,
			total,
			page: raw.page ?? page,
			pageSize: resolvedPageSize,
		};
	});
}

export interface ListPipelinesParams {
	page?: number;
	pageSize?: number;
	q?: string;
	scope?: string;
	sort?: "updated_at_desc" | "name_asc" | "name_desc" | "created_at_desc";
	// CYB-3390: drop auto-named single-step drafts (pipeline-<10+ digits>)
	// at the DB layer so pagination reflects the curated set rather than
	// asking the client to skip past pages of throwaway drafts.
	excludeAutoDrafts?: boolean;
}

export interface ListPipelinesResponse {
	items: PipelineTemplate[];
	total: number;
	page: number;
	pageSize: number;
}

export interface PipelineDiffNode {
	id: string;
	component?: Record<string, unknown>;
}

export interface PipelineDiffEdge {
	source: string;
	target: string;
}

export interface PipelineDiff {
	added_nodes: PipelineDiffNode[];
	removed_nodes: PipelineDiffNode[];
	modified_nodes: PipelineDiffNode[];
	added_edges: PipelineDiffEdge[];
	removed_edges: PipelineDiffEdge[];
}

export function getPipelineDiff(
	id1: string,
	id2: string,
): Promise<PipelineDiff> {
	return request<PipelineDiff>("GET", `/pipelines/${id1}/diff/${id2}`);
}

export function getPipeline(id: string): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("GET", `/pipelines/${id}`);
}

export function listPipelineVersions(
	templateId: string,
): Promise<PipelineTemplate[]> {
	return request<{ items: PipelineTemplate[] }>(
		"GET",
		`/pipelines/${templateId}/versions`,
	).then((r) => r.items);
}

export function savePipeline(
	name: string,
	pipeline: Pipeline,
): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("POST", "/pipelines", { name, pipeline });
}

export function deletePipeline(id: string): Promise<void> {
	return request<void>("DELETE", `/pipelines/${id}`);
}

export function promotePipeline(id: string): Promise<PipelineTemplate> {
	return request<PipelineTemplate>("POST", `/pipelines/${id}/promote`);
}

export function deployTemplate(
	templateId: string,
	assetIds?: string[],
	targetId?: string,
	version?: number,
	configSelection?: DeployConfigSelection,
): Promise<DeployTemplateResponse> {
	return request<DeployTemplateResponse>(
		"POST",
		`/pipeline-runs/template/${templateId}`,
		{
			asset_ids: assetIds ?? [],
			target_id: targetId,
			version,
			configSelection,
		},
	);
}

export type DeployTemplateResponse =
	| Deployment
	| { items: Deployment[]; total?: number };

export function normalizeDeployResults(
	result: DeployTemplateResponse,
): Deployment[] {
	if (
		result &&
		typeof result === "object" &&
		"items" in result &&
		Array.isArray(result.items)
	) {
		return result.items;
	}
	return [result as Deployment];
}

export function listDeployments(): Promise<Deployment[]> {
	return request<{ items: Deployment[] }>("GET", "/deployments").then(
		(r) => r.items,
	);
}

export function getPipelineRunByWorkflowName(
	workflowName: string,
): Promise<PipelineRun> {
	return request<PipelineRun>(
		"GET",
		`/pipeline-runs/by-workflow/${encodeURIComponent(workflowName)}`,
	);
}

export function getPipelineRun(id: string): Promise<PipelineRun> {
	return request<PipelineRun>(
		"GET",
		`/pipeline-runs/${encodeURIComponent(id)}`,
	);
}

export function deletePipelineRun(id: string): Promise<void> {
	return request<void>("DELETE", `/pipeline-runs/${encodeURIComponent(id)}`);
}

export function listPipelineRuns(options?: {
	view?: "summary" | "full";
	excludeBatch?: boolean;
	batchJobId?: string;
	status?: string;
	pipelineNodeId?: string;
	nodeStatus?: string;
	page?: number;
	pageSize?: number;
}): Promise<PipelineRunListResponse> {
	const view = options?.view ?? "full";
	const search = new URLSearchParams();
	if (view === "summary") search.set("view", "summary");
	if (options?.excludeBatch) search.set("excludeBatch", "true");
	if (options?.batchJobId) search.set("batchJobId", options.batchJobId);
	if (options?.status) search.set("status", options.status);
	if (options?.pipelineNodeId) {
		search.set("pipelineNodeId", options.pipelineNodeId);
	}
	if (options?.nodeStatus) search.set("nodeStatus", options.nodeStatus);
	if (options?.page) search.set("page", String(options.page));
	if (options?.pageSize) search.set("pageSize", String(options.pageSize));
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunListResponse>("GET", `/pipeline-runs${suffix}`);
}

export function listPipelineRunEvents(
	runId: string,
	params?: {
		limit?: number;
		cursor?: number;
		subjectType?: string;
		eventType?: string;
		status?: string;
		q?: string;
		from?: string;
		to?: string;
	},
): Promise<PipelineRunEventsResponse> {
	const search = new URLSearchParams();
	if (params?.limit) search.set("limit", String(params.limit));
	if (params?.cursor) search.set("cursor", String(params.cursor));
	if (params?.subjectType) search.set("subjectType", params.subjectType);
	if (params?.eventType) search.set("eventType", params.eventType);
	if (params?.status) search.set("status", params.status);
	if (params?.q) search.set("q", params.q);
	if (params?.from) search.set("from", params.from);
	if (params?.to) search.set("to", params.to);
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunEventsResponse>(
		"GET",
		`/pipeline-runs/${runId}/events${suffix}`,
	);
}

export function getPipelineRunWatcherStatus(): Promise<PipelineRunWatcherState> {
	return request<PipelineRunWatcherState>(
		"GET",
		"/pipeline-runs/watcher/status",
	);
}

export function listPipelineRunAssetNodes(
	runId: string,
	params?: {
		limit?: number;
		cursor?: string;
		assetId?: string;
		nodeId?: string;
		status?: string;
		orderBy?: "cost" | "duration" | "status";
	},
): Promise<PipelineRunAssetNodeListResponse> {
	const search = new URLSearchParams();
	if (params?.limit) search.set("limit", String(params.limit));
	if (params?.cursor) search.set("cursor", params.cursor);
	if (params?.assetId) search.set("assetId", params.assetId);
	if (params?.nodeId) search.set("nodeId", params.nodeId);
	if (params?.status) search.set("status", params.status);
	if (params?.orderBy) search.set("orderBy", params.orderBy);
	const suffix = search.toString() ? `?${search.toString()}` : "";
	return request<PipelineRunAssetNodeListResponse>(
		"GET",
		`/pipeline-runs/${runId}/asset-nodes${suffix}`,
	);
}

export function getPipelineRunCostSummary(
	runId: string,
): Promise<PipelineRunCostSummary> {
	return request<PipelineRunCostSummary>(
		"GET",
		`/pipeline-runs/${runId}/cost-summary`,
	);
}

export function listExecutionTargets(): Promise<ExecutionTarget[]> {
	return request<{ items: ExecutionTarget[] }>(
		"GET",
		"/execution-targets",
	).then((r) => r.items);
}

export function createExecutionTarget(
	body: Partial<ExecutionTarget>,
): Promise<ExecutionTarget> {
	return request<ExecutionTarget>("POST", "/execution-targets", body);
}

export function updateExecutionTarget(
	id: string,
	body: Partial<ExecutionTarget>,
): Promise<ExecutionTarget> {
	return request<ExecutionTarget>(
		"PUT",
		`/execution-targets/${encodeURIComponent(id)}`,
		body,
	);
}

export function deleteExecutionTarget(id: string): Promise<void> {
	return request<void>(
		"DELETE",
		`/execution-targets/${encodeURIComponent(id)}`,
	);
}

// Cluster is a K8s execution destination (CYB-3425). Read is exposed to any
// authenticated user (targets need it for the pool dropdown); write is
// admin-only via /api/v1/admin/clusters.
export interface Cluster {
	id: string;
	name: string;
	displayName: string;
	description?: string;
	isDefault: boolean;
	status: string;
	k8sApiEndpoint?: string;
	k8sAudience?: string;
	k8sCaData?: string;
	argoServerUrl?: string;
	argoNamespace?: string;
	koordInstalled: boolean;
	// K8s client rate limits to this cluster's API (rest.Config QPS/Burst).
	// Editable online; the backend rebuilds this cluster's clients on save.
	// Unset/0 → backend default 50/100. CYB-3486.
	clientQps?: number;
	clientBurst?: number;
	createdAt?: string;
	updatedAt?: string;
	deletedAt?: string;
}

export function listClusters(): Promise<Cluster[]> {
	return request<{ items: Cluster[] }>("GET", "/clusters").then(
		(r) => r.items ?? [],
	);
}

export function createCluster(body: Partial<Cluster>): Promise<Cluster> {
	return request<Cluster>("POST", "/admin/clusters", body);
}

export function updateCluster(
	id: string,
	body: Partial<Cluster>,
): Promise<Cluster> {
	return request<Cluster>(
		"PUT",
		`/admin/clusters/${encodeURIComponent(id)}`,
		body,
	);
}

export function deleteCluster(id: string): Promise<void> {
	return request<void>("DELETE", `/admin/clusters/${encodeURIComponent(id)}`);
}

// ElasticQuota is a read-only view of a Koordinator elastic-quota pool
// (scheduling.sigs.k8s.io/v1alpha1). It shows live min / max / used and
// utilisation across the cluster; the backend returns an empty list when
// Koordinator is not installed.
export interface ElasticQuota {
	name: string;
	namespace: string;
	min: { cpu: string; memory: string };
	max: { cpu: string; memory: string };
	used: { cpu: string; memory: string };
	utilizationPercent: { cpu: number; memory: number };
}

// listElasticQuotas fetches the Koordinator ElasticQuota pools on the given
// cluster. Empty clusterId falls back to the default cluster on the backend
// (CYB-3486). Callers that want the traditional single-cluster view can omit.
export function listElasticQuotas(clusterId?: string): Promise<ElasticQuota[]> {
	const trimmed = (clusterId ?? "").trim();
	const path = trimmed
		? `/elastic-quotas?clusterId=${encodeURIComponent(trimmed)}`
		: "/elastic-quotas";
	return request<{ items: ElasticQuota[] }>("GET", path).then(
		(r) => r.items ?? [],
	);
}

export function deleteDeployment(id: string): Promise<void> {
	return request<void>("DELETE", `/deployments/${id}`);
}

export function retryDeployment(id: string): Promise<Deployment> {
	return request<Deployment>("POST", `/deployments/${id}/retry`, {});
}

// Pipeline stats and recommendations for smart grouping
export interface PipelineStats {
	userStats: {
		clickCounts: Record<string, number>;
		runCounts: Record<string, number>;
		executionTimes: Record<string, number>;
		lastAccessTimes: Record<string, string>;
		window: string;
		computedAt: string;
	};
	recommendations: Array<{
		pipelineId: string;
		name: string;
		score: number;
		reason: string;
		clickCount: number;
		runCount: number;
		lastAccess?: string;
	}>;
}

export function getPipelineStats(window = "30d"): Promise<PipelineStats> {
	return request<PipelineStats>("GET", `/pipelines/stats?window=${window}`);
}

export function updatePipelineWithVersion(
	id: string,
	pipeline: Pipeline,
	baseVersion: number,
	note?: string,
): Promise<{ version: number; updatedAt: string }> {
	return request<{ version: number; updatedAt: string }>(
		"PUT",
		`/pipelines/${id}`,
		{ pipeline, baseVersion, note },
	);
}
