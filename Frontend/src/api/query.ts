import { apiClient } from "./client";
import type { Asset } from "./types";

export interface QueryPredicate {
	field: string;
	op: string;
	value?: unknown;
}

export interface QueryExpr {
	and?: QueryExpr[];
	or?: QueryExpr[];
	not?: QueryExpr;
	pred?: QueryPredicate;
}

export interface QuerySort {
	field: string;
	direction?: "asc" | "desc";
}

export interface QueryRequest {
	schema_version?: string;
	mode?: "structured" | "keyword" | "semantic" | "similar";
	scope: { resource: "assets" };
	select?: { fields?: string[] };
	where?: QueryExpr;
	sort?: QuerySort[];
	page?: { page?: number; page_size?: number; offset?: number; limit?: number };
	facets?: Array<{ field: string; size?: number }>;
	debug?: { explain?: boolean };
}

export interface QueryDebugPlan {
	steps?: Array<{
		engine?: string;
		mode?: string;
	}>;
}

export interface QueryFacetBucket {
	value: string;
	count: number;
}

export interface QueryRunResponse {
	items: Asset[];
	total: number;
	page: number;
	page_size: number;
	columns?: string[];
	column_defs?: Array<{ name: string; type: string; nullable: boolean }>;
	offset?: number;
	limit?: number;
	facets?: Record<string, QueryFacetBucket[]>;
	warnings?: string[];
	debug_plan?: QueryDebugPlan;
}

export interface QueryValidateResponse {
	valid: boolean;
	normalized_query?: QueryRequest;
	warnings?: string[];
	field_capabilities?: Array<{
		field?: string;
		engines?: string[];
	}>;
	debug_plan?: QueryDebugPlan;
}

export interface SavedQuery {
	saved_query_id: string;
	name: string;
	description?: string;
	resource: string;
	schema_version: string;
	query_ir_json: QueryRequest;
	owner?: string;
	created_at: string;
	updated_at: string;
}

export interface SavedQueryListResponse {
	items: SavedQuery[];
}

export const queryApi = {
	run: (query: QueryRequest) =>
		apiClient.post<QueryRunResponse>("/queries/run", query).then((r) => r.data),

	validate: (query: QueryRequest) =>
		apiClient
			.post<QueryValidateResponse>("/queries/validate", query)
			.then((r) => r.data),

	listSavedQueries: () =>
		apiClient.get<SavedQueryListResponse>("/saved-queries").then((r) => r.data),

	getSavedQuery: (id: string) =>
		apiClient.get<SavedQuery>(`/saved-queries/${id}`).then((r) => r.data),

	createSavedQuery: (payload: { name: string; description?: string; query_ir_json: QueryRequest }) =>
		apiClient.post<SavedQuery>("/saved-queries", payload).then((r) => r.data),

	updateSavedQuery: (
		id: string,
		payload: { name: string; description?: string; query_ir_json: QueryRequest },
	) => apiClient.patch<SavedQuery>(`/saved-queries/${id}`, payload).then((r) => r.data),

	deleteSavedQuery: (id: string) =>
		apiClient.delete(`/saved-queries/${id}`).then((r) => r.data),
};
