import type { AlgoRegistryItem } from "./algoRegistry";
import { apiClient } from "./client";
import type { TagRegistryItem } from "./tagRegistry";

export interface MetricRegistryItem {
	key: string;
	display_name: string;
	metric_type: string;
	metric_unit?: string;
	target_type?: string;
	higher_is_better?: boolean;
	default_aggregation?: string;
	queryable?: boolean;
	description?: string;
}

export interface ActionLabelRegistry {
	primary_labels: string[];
	labels: string[];
}

export const registryApi = {
	listAlgos: () =>
		apiClient
			.get<{ items: AlgoRegistryItem[] }>("/algo-registry")
			.then((r) => r.data.items),

	listTags: () =>
		apiClient
			.get<{ items: TagRegistryItem[] }>("/tag-registry")
			.then((r) => r.data.items),

	listMetrics: () =>
		apiClient
			.get<{ items: MetricRegistryItem[] }>("/metric-registry")
			.then((r) => r.data.items),

	listLifecycleStates: () =>
		apiClient
			.get<{ items: string[] }>("/lifecycle-states")
			.then((r) => r.data.items),

	getActionLabelRegistry: () =>
		apiClient
			.get<ActionLabelRegistry>("/action-label-registry")
			.then((r) => r.data),
};
