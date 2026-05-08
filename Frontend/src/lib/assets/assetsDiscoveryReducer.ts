// ─── Assets Discovery Workbench — Page-Level Reducer ───
// Pure reducer handling all 9 state domains for the discovery workbench.
// Validates: Requirements R13

import type { AssetsDiscoveryAction } from "./assetsDiscoveryActions";
import { createFilterChip } from "./assetsDiscoveryActions";
import type {
	AssetsDiscoveryState,
	QueryState,
	SavedView,
} from "./assetsDiscoveryTypes";

// ─── Helpers ───

/** Return new state with page reset to 1 and results marked stale. */
function withQueryReset(
	state: AssetsDiscoveryState,
	queryState: QueryState,
): AssetsDiscoveryState {
	return {
		...state,
		queryState: { ...queryState, page: 1 },
		resultsState: { ...state.resultsState, isStale: true },
	};
}

function normalizeSearchFiltersForMode(
	mode: QueryState["searchMode"],
	filters: QueryState["activeFilters"],
): QueryState["activeFilters"] {
	if (mode !== "keyword") {
		return filters;
	}
	return filters.filter((chip) => chip.field !== "_fulltext");
}

// ─── Reducer ───

export function assetsDiscoveryReducer(
	state: AssetsDiscoveryState,
	action: AssetsDiscoveryAction,
): AssetsDiscoveryState {
	switch (action.type) {
		// ── Route / Hydration ──

		case "URL_HYDRATE": {
			const partial = action.payload.queryState;
			const pid = action.payload.previewAssetId;
			const nextMode = partial.searchMode ?? state.queryState.searchMode;
			const nextFilters = partial.activeFilters
				? normalizeSearchFiltersForMode(nextMode, partial.activeFilters)
				: state.queryState.activeFilters;
			return {
				...state,
				queryState: {
					...state.queryState,
					...partial,
					activeFilters: nextFilters,
				},
				previewState: {
					...state.previewState,
					activeAssetId: pid,
					fetchStatus: pid ? "loading" : "idle",
					summary: pid ? state.previewState.summary : null,
				},
				routerState: { ...state.routerState, urlHydrated: true },
				resultsState: { ...state.resultsState, isStale: true },
			};
		}

		case "MARK_URL_HYDRATED":
			return {
				...state,
				routerState: { ...state.routerState, urlHydrated: true },
			};

		case "SET_CURRENT_PATH":
			return {
				...state,
				routerState: { ...state.routerState, currentPath: action.payload.path },
			};

		// ── Search UI ──

		case "SET_SEARCH_DRAFT":
			return {
				...state,
				searchUiState: {
					...state.searchUiState,
					draftText: action.payload.text,
				},
			};

		case "SEARCH_DRAFT_CHANGE":
			return {
				...state,
				searchUiState: {
					...state.searchUiState,
					draftText: action.payload.text,
				},
			};

		case "SEARCH_COMMIT": {
			const newFilters = [
				...state.queryState.activeFilters.filter((c) => c.source !== "search"),
				...action.payload.tokens,
			];
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: normalizeSearchFiltersForMode(
					state.queryState.searchMode,
					newFilters,
				),
			});
		}

		case "SEARCH_MODE_CHANGE":
			return {
				...state,
				searchUiState: { ...state.searchUiState },
				queryState: { ...state.queryState, searchMode: action.payload.mode },
			};

		case "TOGGLE_SUGGESTIONS":
			return {
				...state,
				searchUiState: {
					...state.searchUiState,
					suggestionsOpen: action.payload.open,
				},
			};

		case "SET_HIGHLIGHTED_INDEX":
			return {
				...state,
				searchUiState: {
					...state.searchUiState,
					highlightedIndex: action.payload.index,
				},
			};

		case "TOGGLE_HELP":
			return {
				...state,
				searchUiState: {
					...state.searchUiState,
					helpOpen: action.payload.open,
				},
			};

		// ── Query Commit ──

		case "COMMIT_QUERY_TEXT": {
			const newFilters = [
				...state.queryState.activeFilters.filter((c) => c.source !== "search"),
				...action.payload.tokens,
			];
			return withQueryReset(state, {
				...state.queryState,
				queryText: action.payload.text,
				activeFilters: normalizeSearchFiltersForMode(
					state.queryState.searchMode,
					newFilters,
				),
			});
		}

		case "ADD_FILTER_CHIP":
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: [...state.queryState.activeFilters, action.payload.chip],
			});

		case "REMOVE_FILTER_CHIP":
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: state.queryState.activeFilters.filter(
					(c) => c.id !== action.payload.id,
				),
			});

		case "CLEAR_ALL_FILTERS":
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: [],
				queryText: "",
			});

		case "SET_SORT":
			return withQueryReset(state, {
				...state.queryState,
				sort: action.payload.sort,
			});

		case "SET_PAGE":
			return {
				...state,
				queryState: { ...state.queryState, page: action.payload.page },
				resultsState: { ...state.resultsState, isStale: true },
			};

		case "SET_PAGE_SIZE":
			return withQueryReset(state, {
				...state.queryState,
				pageSize: action.payload.pageSize,
			});

		case "SET_VIEW_MODE":
			return {
				...state,
				queryState: { ...state.queryState, viewMode: action.payload.mode },
			};

		case "SET_SELECTED_COLUMNS":
			return {
				...state,
				queryState: {
					...state.queryState,
					selectedColumns: action.payload.columns,
				},
			};

		case "SET_SEARCH_MODE":
			return withQueryReset(state, {
				...state.queryState,
				searchMode: action.payload.mode,
				activeFilters: normalizeSearchFiltersForMode(
					action.payload.mode,
					state.queryState.activeFilters,
				),
			});

		case "APPLY_SAVED_VIEW": {
			const view: SavedView = action.payload.view;
			return {
				...withQueryReset(state, {
					...state.queryState,
					...view.queryState,
				}),
				searchUiState: {
					...state.searchUiState,
					draftText: view.queryState.queryText ?? "",
				},
				selectionState: {
					selectedIds: new Set(),
					mode: "none",
				},
				savedViewState: {
					...state.savedViewState,
					currentViewId: view.id,
				},
			};
		}

		// ── Facet UI ──

		case "FACET_TOGGLE": {
			const { field, value } = action.payload;
			const existing = state.queryState.activeFilters.find(
				(c) => c.field === field && c.value === value,
			);
			const newFilters = existing
				? state.queryState.activeFilters.filter((c) => c !== existing)
				: [
						...state.queryState.activeFilters,
						createFilterChip(field, "eq", value, "facet"),
					];
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: newFilters,
			});
		}

		case "FACET_RANGE_DRAFT":
			return {
				...state,
				facetUiState: {
					...state.facetUiState,
					rangeDrafts: {
						...state.facetUiState.rangeDrafts,
						[action.payload.field]: {
							min: action.payload.min,
							max: action.payload.max,
						},
					},
				},
			};

		case "FACET_RANGE_APPLY": {
			const { field, min, max } = action.payload;
			const withoutField = state.queryState.activeFilters.filter(
				(c) => c.field !== field,
			);
			const rangeValue = [
				min !== undefined ? String(min) : "",
				max !== undefined ? String(max) : "",
			];
			const newFilters = [
				...withoutField,
				createFilterChip(field, "between", rangeValue, "facet"),
			];
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: newFilters,
			});
		}

		case "FACET_DATE_DRAFT":
			return {
				...state,
				facetUiState: {
					...state.facetUiState,
					dateDrafts: {
						...state.facetUiState.dateDrafts,
						[action.payload.field]: {
							start: action.payload.start,
							end: action.payload.end,
						},
					},
				},
			};

		case "FACET_DATE_APPLY": {
			const { field, start, end } = action.payload;
			const withoutField = state.queryState.activeFilters.filter(
				(c) => c.field !== field,
			);
			const dateValue = [start ?? "", end ?? ""];
			const newFilters = [
				...withoutField,
				createFilterChip(field, "between", dateValue, "facet"),
			];
			return withQueryReset(state, {
				...state.queryState,
				activeFilters: newFilters,
			});
		}

		case "FACET_GROUP_TOGGLE": {
			const { group } = action.payload;
			const groups = state.facetUiState.expandedGroups;
			const newGroups = groups.includes(group)
				? groups.filter((g) => g !== group)
				: [...groups, group];
			return {
				...state,
				facetUiState: { ...state.facetUiState, expandedGroups: newGroups },
			};
		}

		case "TOGGLE_ADD_FILTER":
			return {
				...state,
				facetUiState: {
					...state.facetUiState,
					addFilterOpen: action.payload.open,
				},
			};

		// ── Results Fetch ──

		case "RESULTS_LOADING":
			return {
				...state,
				resultsState: {
					...state.resultsState,
					fetchStatus: "loading",
					error: null,
				},
			};

		case "RESULTS_SUCCESS":
			return {
				...state,
				resultsState: {
					items: action.payload.items,
					total: action.payload.total,
					totalApprox: action.payload.totalApprox,
					fetchStatus: "success",
					error: null,
					isStale: false,
					aggregations: action.payload.aggregations,
					debugPlan: action.payload.debugPlan,
					warnings: action.payload.warnings ?? [],
				},
			};

		case "RESULTS_ERROR":
			return {
				...state,
				resultsState: {
					...state.resultsState,
					fetchStatus: "error",
					error: action.payload.error,
					isStale: false,
				},
			};

		case "MARK_RESULTS_FRESH":
			return {
				...state,
				resultsState: { ...state.resultsState, isStale: false },
			};

		case "VALIDATION_LOADING":
			return {
				...state,
				validationState: {
					...state.validationState,
					fetchStatus: "loading",
					error: null,
				},
			};

		case "VALIDATION_SUCCESS":
			return {
				...state,
				validationState: {
					...state.validationState,
					fetchStatus: "success",
					valid: action.payload.valid,
					error: null,
					normalizedQuery: action.payload.normalizedQuery,
					warnings: action.payload.warnings,
					fieldCapabilities: action.payload.fieldCapabilities,
					debugPlan: action.payload.debugPlan,
					inspectorOpen:
						action.payload.warnings.length > 0
							? true
							: state.validationState.inspectorOpen,
				},
			};

		case "VALIDATION_ERROR":
			return {
				...state,
				validationState: {
					...state.validationState,
					fetchStatus: "error",
					error: action.payload.error,
					valid: false,
					inspectorOpen: true,
				},
			};

		case "TOGGLE_INSPECTOR":
			return {
				...state,
				validationState: {
					...state.validationState,
					inspectorOpen: action.payload.open,
				},
			};

		// ── Selection ──

		case "TOGGLE_ROW_SELECTION": {
			const { id } = action.payload;
			const newIds = new Set(state.selectionState.selectedIds);
			if (newIds.has(id)) {
				newIds.delete(id);
			} else {
				newIds.add(id);
			}
			return {
				...state,
				selectionState: {
					selectedIds: newIds,
					mode: newIds.size === 0 ? "none" : "explicit_rows",
				},
			};
		}

		case "SELECT_ALL_FILTERED":
			return {
				...state,
				selectionState: {
					...state.selectionState,
					mode: "all_filtered_results",
				},
			};

		case "CLEAR_SELECTION":
			return {
				...state,
				selectionState: {
					selectedIds: new Set(),
					mode: "none",
				},
			};

		case "SET_SELECTION_MODE":
			return {
				...state,
				selectionState: { ...state.selectionState, mode: action.payload.mode },
			};

		// ── Preview ──

		case "SET_ACTIVE_PREVIEW_ASSET":
			return {
				...state,
				previewState: {
					...state.previewState,
					activeAssetId: action.payload.assetId,
					fetchStatus: action.payload.assetId ? "loading" : "idle",
					summary: action.payload.assetId ? state.previewState.summary : null,
					// Default preview starts collapsed to leave width for the table; expand when user picks a row.
					collapsed: action.payload.assetId
						? false
						: state.previewState.collapsed,
				},
			};

		case "PREVIEW_LOADING":
			return {
				...state,
				previewState: { ...state.previewState, fetchStatus: "loading" },
			};

		case "RECEIVE_PREVIEW_SUCCESS":
			return {
				...state,
				previewState: {
					...state.previewState,
					fetchStatus: "success",
					summary: action.payload.asset,
					availability: action.payload.manifest.availability,
				},
			};

		case "RECEIVE_PREVIEW_ERROR":
			return {
				...state,
				previewState: {
					...state.previewState,
					fetchStatus: "error",
					summary: null,
				},
			};

		case "PREVIEW_COLLAPSE_TOGGLE":
			return {
				...state,
				previewState: {
					...state.previewState,
					collapsed: !state.previewState.collapsed,
				},
			};

		case "PREVIEW_CLEAR":
			return {
				...state,
				previewState: {
					activeAssetId: null,
					fetchStatus: "idle",
					summary: null,
					availability: "missing",
					collapsed: state.previewState.collapsed,
				},
			};

		// ── Saved Views ──

		case "SAVED_VIEW_SELECT":
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					currentViewId: action.payload.viewId,
				},
			};

		case "SAVED_VIEW_SAVE": {
			const newView: SavedView = {
				id: `custom_${Date.now()}`,
				name: action.payload.name,
				builtin: false,
				queryState: { ...state.queryState },
			};
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					views: [...state.savedViewState.views, newView],
					currentViewId: newView.id,
					saveDialogOpen: false,
				},
			};
		}

		case "SAVED_VIEW_DELETE": {
			const viewToDelete = state.savedViewState.views.find(
				(v) => v.id === action.payload.viewId,
			);
			if (!viewToDelete || viewToDelete.builtin) return state;
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					views: state.savedViewState.views.filter(
						(v) => v.id !== action.payload.viewId,
					),
					currentViewId:
						state.savedViewState.currentViewId === action.payload.viewId
							? null
							: state.savedViewState.currentViewId,
				},
			};
		}

		case "SAVED_VIEW_RENAME": {
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					views: state.savedViewState.views.map((v) =>
						v.id === action.payload.viewId
							? { ...v, name: action.payload.name }
							: v,
					),
				},
			};
		}

		case "SAVED_VIEW_LOAD":
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					views: action.payload.views,
				},
			};

		case "TOGGLE_SAVE_DIALOG":
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					saveDialogOpen: action.payload.open,
				},
			};

		case "SET_CURRENT_VIEW":
			return {
				...state,
				savedViewState: {
					...state.savedViewState,
					currentViewId: action.payload.viewId,
				},
			};

		// ── Layout ──

		case "FACET_COLLAPSE_TOGGLE":
			return {
				...state,
				layoutState: {
					...state.layoutState,
					facetCollapsed: !state.layoutState.facetCollapsed,
				},
			};

		case "LAYOUT_RESPONSIVE":
			return {
				...state,
				layoutState: {
					...state.layoutState,
					facetCollapsed: action.payload.isMobile,
					previewCollapsed: action.payload.isMobile || action.payload.isTablet,
				},
			};

		case "TOGGLE_COLUMNS_POPOVER":
			return {
				...state,
				layoutState: {
					...state.layoutState,
					columnsPopoverOpen: action.payload.open,
				},
			};

		default:
			return state;
	}
}
