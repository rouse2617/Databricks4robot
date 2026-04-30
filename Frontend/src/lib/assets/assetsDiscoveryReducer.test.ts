import { describe, it, expect } from "vitest";
import { assetsDiscoveryReducer } from "./assetsDiscoveryReducer";
import { createFilterChip } from "./assetsDiscoveryActions";
import {
  defaultAssetsDiscoveryState,
  type AssetsDiscoveryState,
} from "./assetsDiscoveryTypes";

/** Fresh copy of default state for each test. */
function freshState(): AssetsDiscoveryState {
  return {
    ...defaultAssetsDiscoveryState,
    queryState: { ...defaultAssetsDiscoveryState.queryState, activeFilters: [] },
    selectionState: { selectedIds: new Set(), mode: "none" },
    resultsState: { ...defaultAssetsDiscoveryState.resultsState, isStale: false },
  };
}

// ─── Query-changing actions reset page & set isStale ───

describe("query-changing actions reset page to 1 and set isStale", () => {
  const stateOnPage3: AssetsDiscoveryState = {
    ...freshState(),
    queryState: { ...freshState().queryState, page: 3 },
  };

  it("COMMIT_QUERY_TEXT resets page and marks stale", () => {
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "COMMIT_QUERY_TEXT",
      payload: { text: "env:warehouse", tokens: [] },
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("ADD_FILTER_CHIP resets page and marks stale", () => {
    const chip = createFilterChip("status", "eq", "approved", "facet");
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "ADD_FILTER_CHIP",
      payload: { chip },
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("REMOVE_FILTER_CHIP resets page and marks stale", () => {
    const chip = createFilterChip("status", "eq", "approved", "facet");
    const withChip = assetsDiscoveryReducer(stateOnPage3, {
      type: "ADD_FILTER_CHIP",
      payload: { chip },
    });
    const result = assetsDiscoveryReducer(
      { ...withChip, queryState: { ...withChip.queryState, page: 3 } },
      { type: "REMOVE_FILTER_CHIP", payload: { id: chip.id } },
    );
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("CLEAR_ALL_FILTERS resets page and marks stale", () => {
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "CLEAR_ALL_FILTERS",
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("SET_SORT resets page and marks stale", () => {
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "SET_SORT",
      payload: { sort: "created_at" },
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("SET_SEARCH_MODE resets page and marks stale", () => {
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "SET_SEARCH_MODE",
      payload: { mode: "keyword" },
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });

  it("APPLY_SAVED_VIEW resets page and marks stale", () => {
    const result = assetsDiscoveryReducer(stateOnPage3, {
      type: "APPLY_SAVED_VIEW",
      payload: {
        view: {
          id: "v1",
          name: "Test",
          builtin: false,
          queryState: { sort: "created_at" },
        },
      },
    });
    expect(result.queryState.page).toBe(1);
    expect(result.resultsState.isStale).toBe(true);
  });
});

// ─── TOGGLE_ROW_SELECTION ───

describe("TOGGLE_ROW_SELECTION", () => {
  it("adds id to selectedIds and sets mode to explicit_rows", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "TOGGLE_ROW_SELECTION",
      payload: { id: "asset_1" },
    });
    expect(result.selectionState.selectedIds.has("asset_1")).toBe(true);
    expect(result.selectionState.mode).toBe("explicit_rows");
  });

  it("removes id if already selected", () => {
    const s = freshState();
    s.selectionState.selectedIds = new Set(["asset_1"]);
    s.selectionState.mode = "explicit_rows";
    const result = assetsDiscoveryReducer(s, {
      type: "TOGGLE_ROW_SELECTION",
      payload: { id: "asset_1" },
    });
    expect(result.selectionState.selectedIds.has("asset_1")).toBe(false);
    expect(result.selectionState.mode).toBe("none");
  });

  it("does NOT change previewState", () => {
    const s = freshState();
    s.previewState = { ...s.previewState, activeAssetId: "other" };
    const result = assetsDiscoveryReducer(s, {
      type: "TOGGLE_ROW_SELECTION",
      payload: { id: "asset_1" },
    });
    expect(result.previewState.activeAssetId).toBe("other");
  });
});

// ─── SET_ACTIVE_PREVIEW_ASSET ───

describe("SET_ACTIVE_PREVIEW_ASSET", () => {
  it("updates previewState only", () => {
    const s = freshState();
    s.selectionState = { selectedIds: new Set(["x"]), mode: "explicit_rows" };
    const result = assetsDiscoveryReducer(s, {
      type: "SET_ACTIVE_PREVIEW_ASSET",
      payload: { assetId: "asset_2" },
    });
    expect(result.previewState.activeAssetId).toBe("asset_2");
    // selectionState unchanged
    expect(result.selectionState.selectedIds.has("x")).toBe(true);
    expect(result.selectionState.mode).toBe("explicit_rows");
  });

  it("clears summary when assetId is null", () => {
    const s = freshState();
    s.previewState.activeAssetId = "asset_1";
    const result = assetsDiscoveryReducer(s, {
      type: "SET_ACTIVE_PREVIEW_ASSET",
      payload: { assetId: null },
    });
    expect(result.previewState.activeAssetId).toBeNull();
    expect(result.previewState.summary).toBeNull();
  });
});

// ─── APPLY_SAVED_VIEW ───

describe("APPLY_SAVED_VIEW", () => {
  it("overwrites queryState from snapshot", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "APPLY_SAVED_VIEW",
      payload: {
        view: {
          id: "algo_failed",
          name: "Algo Failed",
          builtin: true,
          queryState: {
            sort: "created_at",
            activeFilters: [createFilterChip("algo_status", "eq", "failed", "saved_view")],
          },
        },
      },
    });
    expect(result.queryState.sort).toBe("created_at");
    expect(result.queryState.activeFilters).toHaveLength(1);
    expect(result.queryState.activeFilters[0].field).toBe("algo_status");
  });

  it("clears selectionState", () => {
    const s = freshState();
    s.selectionState = { selectedIds: new Set(["a", "b"]), mode: "explicit_rows" };
    const result = assetsDiscoveryReducer(s, {
      type: "APPLY_SAVED_VIEW",
      payload: {
        view: { id: "v1", name: "V", builtin: false, queryState: {} },
      },
    });
    expect(result.selectionState.selectedIds.size).toBe(0);
    expect(result.selectionState.mode).toBe("none");
  });

  it("updates searchUiState.draftText from view queryText", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "APPLY_SAVED_VIEW",
      payload: {
        view: {
          id: "v1",
          name: "V",
          builtin: false,
          queryState: { queryText: "env:warehouse" },
        },
      },
    });
    expect(result.searchUiState.draftText).toBe("env:warehouse");
  });

  it("sets draftText to empty when view has no queryText", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "APPLY_SAVED_VIEW",
      payload: {
        view: { id: "v1", name: "V", builtin: false, queryState: {} },
      },
    });
    expect(result.searchUiState.draftText).toBe("");
  });
});

// ─── URL_HYDRATE ───

describe("URL_HYDRATE", () => {
  it("merges partial QueryState into current state", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "URL_HYDRATE",
      payload: { queryState: { page: 5, sort: "created_at" } },
    });
    expect(result.queryState.page).toBe(5);
    expect(result.queryState.sort).toBe("created_at");
    // other fields preserved
    expect(result.queryState.pageSize).toBe(20);
    expect(result.queryState.searchMode).toBe("structured");
  });
});

// ─── FACET_TOGGLE ───

describe("FACET_TOGGLE", () => {
  it("adds a chip when no matching chip exists", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "FACET_TOGGLE",
      payload: { field: "env", value: "warehouse" },
    });
    expect(result.queryState.activeFilters).toHaveLength(1);
    expect(result.queryState.activeFilters[0].field).toBe("env");
    expect(result.queryState.activeFilters[0].value).toBe("warehouse");
    expect(result.queryState.activeFilters[0].source).toBe("facet");
  });

  it("removes chip when matching field+value exists", () => {
    const chip = createFilterChip("env", "eq", "warehouse", "facet");
    const s = freshState();
    s.queryState.activeFilters = [chip];
    const result = assetsDiscoveryReducer(s, {
      type: "FACET_TOGGLE",
      payload: { field: "env", value: "warehouse" },
    });
    expect(result.queryState.activeFilters).toHaveLength(0);
  });
});


// ─── FACET_RANGE_APPLY / FACET_DATE_APPLY ───

describe("FACET_RANGE_APPLY", () => {
  it("removes existing chips for that field and adds new range chip", () => {
    const chip = createFilterChip("duration_ms", "eq", "100", "facet");
    const s = freshState();
    s.queryState.activeFilters = [chip];
    const result = assetsDiscoveryReducer(s, {
      type: "FACET_RANGE_APPLY",
      payload: { field: "duration_ms", min: 10, max: 200 },
    });
    expect(result.queryState.activeFilters).toHaveLength(1);
    expect(result.queryState.activeFilters[0].op).toBe("between");
    expect(result.queryState.activeFilters[0].value).toEqual(["10", "200"]);
  });
});

describe("FACET_DATE_APPLY", () => {
  it("removes existing chips for that field and adds new date chip", () => {
    const s = freshState();
    s.queryState.activeFilters = [createFilterChip("created_at", "eq", "2024-01-01", "facet")];
    const result = assetsDiscoveryReducer(s, {
      type: "FACET_DATE_APPLY",
      payload: { field: "created_at", start: "2024-01-01", end: "2024-12-31" },
    });
    expect(result.queryState.activeFilters).toHaveLength(1);
    expect(result.queryState.activeFilters[0].value).toEqual(["2024-01-01", "2024-12-31"]);
  });
});

// ─── Selection actions ───

describe("SELECT_ALL_FILTERED", () => {
  it("sets mode to all_filtered_results", () => {
    const result = assetsDiscoveryReducer(freshState(), {
      type: "SELECT_ALL_FILTERED",
    });
    expect(result.selectionState.mode).toBe("all_filtered_results");
  });
});

describe("CLEAR_SELECTION", () => {
  it("resets selectedIds and mode", () => {
    const s = freshState();
    s.selectionState = { selectedIds: new Set(["a", "b"]), mode: "explicit_rows" };
    const result = assetsDiscoveryReducer(s, { type: "CLEAR_SELECTION" });
    expect(result.selectionState.selectedIds.size).toBe(0);
    expect(result.selectionState.mode).toBe("none");
  });
});

// ─── Layout toggles ───

describe("PREVIEW_COLLAPSE_TOGGLE", () => {
  it("toggles previewState.collapsed", () => {
    const s = freshState();
    expect(s.previewState.collapsed).toBe(true);
    const r1 = assetsDiscoveryReducer(s, { type: "PREVIEW_COLLAPSE_TOGGLE" });
    expect(r1.previewState.collapsed).toBe(false);
    const r2 = assetsDiscoveryReducer(r1, { type: "PREVIEW_COLLAPSE_TOGGLE" });
    expect(r2.previewState.collapsed).toBe(true);
  });
});

describe("FACET_COLLAPSE_TOGGLE", () => {
  it("toggles layoutState.facetCollapsed", () => {
    const s = freshState();
    expect(s.layoutState.facetCollapsed).toBe(false);
    const r1 = assetsDiscoveryReducer(s, { type: "FACET_COLLAPSE_TOGGLE" });
    expect(r1.layoutState.facetCollapsed).toBe(true);
    const r2 = assetsDiscoveryReducer(r1, { type: "FACET_COLLAPSE_TOGGLE" });
    expect(r2.layoutState.facetCollapsed).toBe(false);
  });
});

describe("FACET_GROUP_TOGGLE", () => {
  it("toggles group in facetUiState.expandedGroups", () => {
    const s = freshState();
    // "basic" is in default expanded groups
    expect(s.facetUiState.expandedGroups).toContain("basic");
    const r1 = assetsDiscoveryReducer(s, {
      type: "FACET_GROUP_TOGGLE",
      payload: { group: "basic" },
    });
    expect(r1.facetUiState.expandedGroups).not.toContain("basic");
    const r2 = assetsDiscoveryReducer(r1, {
      type: "FACET_GROUP_TOGGLE",
      payload: { group: "basic" },
    });
    expect(r2.facetUiState.expandedGroups).toContain("basic");
  });
});

// ─── Saved View actions ───

describe("SAVED_VIEW_SAVE", () => {
  it("creates a new SavedView from current queryState", () => {
    const s = freshState();
    s.queryState.sort = "created_at";
    const result = assetsDiscoveryReducer(s, {
      type: "SAVED_VIEW_SAVE",
      payload: { name: "My View" },
    });
    expect(result.savedViewState.views).toHaveLength(1);
    expect(result.savedViewState.views[0].name).toBe("My View");
    expect(result.savedViewState.views[0].builtin).toBe(false);
    expect(result.savedViewState.views[0].queryState.sort).toBe("created_at");
    expect(result.savedViewState.currentViewId).toBe(result.savedViewState.views[0].id);
  });
});

describe("SAVED_VIEW_DELETE", () => {
  it("removes non-builtin view by id", () => {
    const s = freshState();
    s.savedViewState.views = [
      { id: "custom_1", name: "Custom", builtin: false, queryState: {} },
    ];
    const result = assetsDiscoveryReducer(s, {
      type: "SAVED_VIEW_DELETE",
      payload: { viewId: "custom_1" },
    });
    expect(result.savedViewState.views).toHaveLength(0);
  });

  it("does NOT remove builtin views", () => {
    const s = freshState();
    s.savedViewState.views = [
      { id: "all", name: "All", builtin: true, queryState: {} },
    ];
    const result = assetsDiscoveryReducer(s, {
      type: "SAVED_VIEW_DELETE",
      payload: { viewId: "all" },
    });
    expect(result.savedViewState.views).toHaveLength(1);
  });
});

// ─── Purity check ───

describe("reducer purity", () => {
  it("does not mutate the original state", () => {
    const s = freshState();
    const original = JSON.stringify(s, (_k, v) => (v instanceof Set ? [...v] : v));
    assetsDiscoveryReducer(s, {
      type: "ADD_FILTER_CHIP",
      payload: { chip: createFilterChip("env", "eq", "warehouse", "facet") },
    });
    const after = JSON.stringify(s, (_k, v) => (v instanceof Set ? [...v] : v));
    expect(after).toBe(original);
  });
});
