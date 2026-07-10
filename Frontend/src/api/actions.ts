import { apiClient } from "./client";

export type ActionSourceType = "human" | "algo" | "rule" | "system";

export interface Action {
	action_id: string;
	asset_id: string;
	start_ns: number;
	end_ns: number;
	action_index?: number | null;
	primary_label?: string;
	labels: string[];
	description?: string;
	attrs: Record<string, unknown>;
	source_type: ActionSourceType;
	source_name?: string;
	source_version?: string;
	run_id?: string;
	confidence?: number | null;
	external_id?: string;
	tenant_id?: string | null;
	project_id?: string | null;
	is_deleted: boolean;
	version: number;
	created_at: string;
	updated_at: string;
}

export interface ActionListResponse {
	items: Action[];
	asset_id: string;
	total: number;
}

export interface ActionCreateInput {
	start_ns: number;
	end_ns: number;
	action_index?: number;
	primary_label?: string;
	labels?: string[];
	description?: string;
	attrs?: Record<string, unknown>;
	source_type?: ActionSourceType;
	source_name?: string;
	source_version?: string;
	run_id?: string;
	confidence?: number;
	external_id?: string;
}

export interface ActionListParams {
	/** Match primary_label OR membership in labels[]. Applied client-side. */
	label?: string;
	limit?: number;
	offset?: number;
}

// --- CYB-3268: first-class asset (asset_type='action') adapter -----------------
// The backend now serves /assets/:id/actions from the `assets` table, so rows
// are first-class asset shape (asset_id / parent_asset_id / start_timestamp_ns /
// metadata.*) rather than the legacy actions-table shape (action_id / start_ns /
// top-level primary_label). This adapter maps the wire row to the `Action` view
// model so `ActionsTimelineTab.tsx` stays unchanged. It is defensive about both
// shapes so a frontend/backend deploy-order gap cannot break the tab.

type Row = Record<string, unknown>;

const asString = (v: unknown): string | undefined =>
	typeof v === "string" ? v : undefined;
const asNumber = (v: unknown): number => (typeof v === "number" ? v : 0);
const asStringArray = (v: unknown): string[] =>
	Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : [];

/** Map a wire row (first-class asset OR legacy action) to the `Action` view model. */
function toAction(row: Row): Action {
	const isFirstClass =
		row.asset_type !== undefined ||
		row.parent_asset_id !== undefined ||
		row.start_timestamp_ns !== undefined;
	if (!isFirstClass) {
		// Legacy actions-table row already matches the Action shape.
		return {
			...(row as unknown as Action),
			labels: asStringArray(row.labels),
		};
	}
	const md = (row.metadata ?? {}) as Row;
	return {
		// First-class: the asset's own id is the action id; parent is the seg.
		action_id: asString(row.asset_id) ?? asString(row.action_id) ?? "",
		asset_id: asString(row.parent_asset_id) ?? asString(row.asset_id) ?? "",
		start_ns: asNumber(row.start_timestamp_ns ?? row.start_ns),
		end_ns: asNumber(row.end_timestamp_ns ?? row.end_ns),
		action_index: (md.action_index as number | undefined) ?? null,
		primary_label: asString(md.primary_label),
		labels: asStringArray(md.labels),
		description: asString(md.description),
		attrs: md,
		source_type: (asString(md.source_type) as ActionSourceType) ?? "human",
		source_name: asString(md.source_name),
		source_version: asString(md.source_version),
		run_id: asString(md.run_id),
		confidence: (md.confidence as number | undefined) ?? null,
		external_id: asString(md.external_id),
		tenant_id: asString(row.tenant_id) ?? null,
		project_id: asString(row.project_id) ?? null,
		is_deleted: row.is_deleted === true,
		version: asNumber(row.version),
		created_at: asString(row.created_at) ?? "",
		updated_at: asString(row.updated_at) ?? "",
	};
}

/** Map the `Action` create input to the first-class `createChildAssetRequest` body. */
function toCreateBody(payload: ActionCreateInput): Record<string, unknown> {
	return {
		start_timestamp_ns: payload.start_ns,
		end_timestamp_ns: payload.end_ns,
		metadata: {
			primary_label: payload.primary_label,
			labels: payload.labels,
			description: payload.description,
			source_type: payload.source_type,
			source_name: payload.source_name,
			source_version: payload.source_version,
			run_id: payload.run_id,
			confidence: payload.confidence,
			external_id: payload.external_id,
			action_index: payload.action_index,
		},
	};
}

export const actionsApi = {
	list: (assetId: string, params: ActionListParams = {}) =>
		apiClient
			.get<Row | Row[]>(`/assets/${assetId}/actions`, {
				params: { limit: params.limit, offset: params.offset },
			})
			.then((r) => {
				const data = r.data;
				const rawItems: Row[] = Array.isArray(data)
					? data
					: ((data.items as Row[] | undefined) ?? []);
				let items = rawItems.map(toAction);
				// Backend GET is limit/offset only (CYB-3268); filter by label client-side.
				if (params.label) {
					const needle = params.label;
					items = items.filter(
						(a) => a.primary_label === needle || a.labels.includes(needle),
					);
				}
				const out: ActionListResponse = {
					items,
					asset_id: assetId,
					total: items.length,
				};
				return out;
			}),

	create: (assetId: string, payload: ActionCreateInput) =>
		apiClient
			.post<Row>(`/assets/${assetId}/actions`, toCreateBody(payload))
			.then((r) => toAction(r.data)),
};
