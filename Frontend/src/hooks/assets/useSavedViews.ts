// ─── useSavedViews — built-in presets + persisted saved queries ───
// Manages system presets and remote saved queries for the query workbench.
// Validates: Requirements R11

import { useCallback, useEffect } from "react";
import { queryApi, type QueryRequest } from "../../api/query";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
	AssetsDiscoveryState,
	QueryState,
	SavedView,
} from "../../lib/assets/assetsDiscoveryTypes";

// ─── Built-in Views ───

export const BUILTIN_VIEWS: SavedView[] = [
	{ id: "all", name: "全部资产", builtin: true, queryState: {} },
	{
		id: "recent",
		name: "最近更新",
		builtin: true,
		queryState: { sort: "-updated_at" },
	},
	{
		id: "algo_failed",
		name: "算法失败",
		builtin: true,
		queryState: {
			activeFilters: [
				{
					id: "builtin_algo_failed",
					field: "algo_status",
					op: "eq",
					value: "failed",
					source: "saved_view",
				},
			],
		},
	},
	{
		id: "pending_delivery",
		name: "待交付",
		builtin: true,
		queryState: {
			activeFilters: [
				{
					id: "builtin_pending",
					field: "has:delivery",
					op: "eq",
					value: "false",
					source: "saved_view",
				},
			],
		},
	},
];

function queryRequestToPartialQueryState(
	query: QueryRequest,
): Partial<QueryState> | null {
	const partial: Partial<QueryState> = {};
	if (query.page) {
		partial.page = query.page.page;
		partial.pageSize = query.page.page_size;
	}
	if (query.sort && query.sort.length > 0) {
		const first = query.sort[0];
		partial.sort = first.direction === "desc" ? `-${first.field}` : first.field;
	}
	if (query.select?.fields) {
		partial.selectedColumns = query.select.fields;
	}
	partial.searchMode =
		query.mode === "semantic" || query.mode === "similar" || query.mode === "keyword"
			? query.mode
			: "structured";
	const chips: QueryState["activeFilters"] = [];
	let fulltextQuery = "";
	const collect = (expr?: QueryRequest["where"]): boolean => {
		if (!expr) return true;
		if (expr.pred) {
			if (expr.pred.field === "_fulltext") {
				fulltextQuery = String(expr.pred.value ?? "").trim();
				return true;
			}
			const value = Array.isArray(expr.pred.value)
				? expr.pred.value.map((item) => String(item))
				: String(expr.pred.value ?? "");
			chips.push({
				id: `remote_${expr.pred.field}_${expr.pred.op}`,
				field: expr.pred.field,
				op: expr.pred.op,
				value,
				source: "saved_view",
			});
			return true;
		}
		if ((expr.or?.length ?? 0) > 0 || expr.not) {
			return false;
		}
		for (const child of expr.and ?? []) {
			if (!collect(child)) {
				return false;
			}
		}
		return true;
	};
	if (query.where && !collect(query.where)) {
		return null;
	}
	partial.activeFilters = chips;
	if (fulltextQuery) {
		partial.queryText = fulltextQuery;
	}
	return partial;
}

// ─── Hook ───

export function useSavedViews(
	state: AssetsDiscoveryState,
	dispatch: React.Dispatch<AssetsDiscoveryAction>,
) {
	const { savedViewState } = state;
	const { views, currentViewId } = savedViewState;

	// Load built-in + custom views on mount
	useEffect(() => {
		let cancelled = false;
		const load = async () => {
			try {
				const remote = await queryApi.listSavedQueries();
				if (cancelled) return;
				const remoteViews = (remote.items ?? []).reduce<SavedView[]>(
					(acc, item) => {
						const queryState = queryRequestToPartialQueryState(item.query_ir_json);
						if (!queryState) {
							return acc;
						}
						acc.push({
							id: `remote:${item.saved_query_id}`,
							name: item.name,
							builtin: false,
							queryState,
						});
						return acc;
					},
					[],
				);
				dispatch({
					type: "SAVED_VIEW_LOAD",
					payload: { views: [...BUILTIN_VIEWS, ...remoteViews] },
				});
				const pendingOpenId = sessionStorage.getItem("assets_open_saved_query_id");
				if (pendingOpenId) {
					const target = [...BUILTIN_VIEWS, ...remoteViews].find(
						(view) => view.id === pendingOpenId,
					);
					if (target) {
						dispatch({ type: "APPLY_SAVED_VIEW", payload: { view: target } });
					}
					sessionStorage.removeItem("assets_open_saved_query_id");
				}
			} catch {
				if (cancelled) return;
				dispatch({
					type: "SAVED_VIEW_LOAD",
					payload: { views: [...BUILTIN_VIEWS] },
				});
			}
		};
		void load();
		return () => {
			cancelled = true;
		};
	}, [dispatch]);

	const selectView = useCallback(
		(viewId: string) => {
			const view = views.find((v) => v.id === viewId);
			if (!view) return;
			dispatch({ type: "APPLY_SAVED_VIEW", payload: { view } });
		},
		[views, dispatch],
	);

	const saveView = useCallback(
		async (name: string) => {
			{
				const activeFilters = state.queryState.activeFilters.filter(
					(chip) => chip.field !== "_fulltext",
				);
				const toPredicate = (chip: QueryState["activeFilters"][number]) => ({
					pred: {
						field: chip.field,
						op: chip.op,
						value: chip.value,
					},
				});
				const query: QueryRequest = {
					schema_version: "v1",
					mode: state.queryState.searchMode,
					scope: { resource: "assets" },
					select: { fields: state.queryState.selectedColumns },
					where:
						state.queryState.searchMode !== "structured" &&
						state.queryState.queryText.trim().length > 0 &&
						activeFilters.length === 0
							? {
									pred: {
										field: "_fulltext",
										op: "ilike",
									value: state.queryState.queryText.trim(),
									},
								}
							: state.queryState.searchMode !== "structured" &&
									state.queryState.queryText.trim().length > 0
								? {
										and: [
											{
												pred: {
													field: "_fulltext",
													op: "ilike",
													value: state.queryState.queryText.trim(),
												},
											},
											...activeFilters.map(toPredicate),
										],
									}
								: activeFilters.length === 0
							? undefined
							: activeFilters.length === 1
								? toPredicate(activeFilters[0])
								: { and: activeFilters.map(toPredicate) },
					sort: [
						state.queryState.sort.startsWith("-")
							? { field: state.queryState.sort.slice(1), direction: "desc" }
							: { field: state.queryState.sort, direction: "asc" },
					],
					page: {
						page: state.queryState.page,
						page_size: state.queryState.pageSize,
					},
				};
				const remote = await queryApi.createSavedQuery({
					name,
					query_ir_json: query,
				});
				dispatch({
					type: "SAVED_VIEW_LOAD",
					payload: {
						views: [
							...views.filter((v) => v.id !== `remote:${remote.saved_query_id}`),
							{
								id: `remote:${remote.saved_query_id}`,
								name: remote.name,
								builtin: false,
								queryState:
									queryRequestToPartialQueryState(remote.query_ir_json) ?? {},
							},
						],
					},
				});
				dispatch({ type: "SET_CURRENT_VIEW", payload: { viewId: `remote:${remote.saved_query_id}` } });
				dispatch({ type: "TOGGLE_SAVE_DIALOG", payload: { open: false } });
				return;
			}
		},
		[dispatch, state.queryState, views],
	);

	const deleteView = useCallback(
		async (viewId: string) => {
			if (viewId.startsWith("remote:")) {
				await queryApi.deleteSavedQuery(viewId.replace("remote:", ""));
			}
			dispatch({ type: "SAVED_VIEW_DELETE", payload: { viewId } });
		},
		[dispatch],
	);

	return {
		views,
		currentViewId,
		selectView,
		saveView,
		deleteView,
	};
}
