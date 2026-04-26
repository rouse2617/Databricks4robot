// ─── useSavedViews — localStorage-based saved views (Phase 1) ───
// Manages built-in + custom saved views for the assets discovery workbench.
// Validates: Requirements R11

import { useEffect, useCallback } from "react";
import type { AssetsDiscoveryAction } from "../../lib/assets/assetsDiscoveryActions";
import type {
  AssetsDiscoveryState,
  SavedView,
} from "../../lib/assets/assetsDiscoveryTypes";

const STORAGE_KEY = "assets_saved_views";

// ─── Built-in Views ───

export const BUILTIN_VIEWS: SavedView[] = [
  { id: "all", name: "全部资产", builtin: true, queryState: {} },
  { id: "recent", name: "最近更新", builtin: true, queryState: { sort: "-updated_at" } },
  {
    id: "algo_failed",
    name: "算法失败",
    builtin: true,
    queryState: {
      activeFilters: [
        { id: "builtin_algo_failed", field: "algo_status", op: "eq", value: "failed", source: "saved_view" },
      ],
    },
  },
  {
    id: "pending_delivery",
    name: "待交付",
    builtin: true,
    queryState: {
      activeFilters: [
        { id: "builtin_pending", field: "has:delivery", op: "eq", value: "false", source: "saved_view" },
      ],
    },
  },
];

// ─── localStorage helpers ───

function loadCustomViews(): SavedView[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.map((v: SavedView) => ({ ...v, builtin: false }));
  } catch {
    return [];
  }
}

function persistCustomViews(views: SavedView[]): void {
  const custom = views.filter((v) => !v.builtin);
  localStorage.setItem(STORAGE_KEY, JSON.stringify(custom));
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
    const custom = loadCustomViews();
    dispatch({
      type: "SAVED_VIEW_LOAD",
      payload: { views: [...BUILTIN_VIEWS, ...custom] },
    });
  }, [dispatch]);

  // Persist custom views whenever views change (skip initial empty)
  useEffect(() => {
    if (views.length > 0) {
      persistCustomViews(views);
    }
  }, [views]);

  const selectView = useCallback(
    (viewId: string) => {
      const view = views.find((v) => v.id === viewId);
      if (!view) return;
      dispatch({ type: "APPLY_SAVED_VIEW", payload: { view } });
    },
    [views, dispatch],
  );

  const saveView = useCallback(
    (name: string) => {
      dispatch({ type: "SAVED_VIEW_SAVE", payload: { name } });
    },
    [dispatch],
  );

  const deleteView = useCallback(
    (viewId: string) => {
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
