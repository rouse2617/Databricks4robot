// ─── Assets Discovery Workbench — URL Serialization & Parsing ───
// Validates: Requirements R8
//
// Serialize QueryState → URLSearchParams and parse URLState back.
// Default values are omitted from the URL to keep it clean.

import type { QueryState, FilterChip, SearchMode, ViewMode } from "./assetsDiscoveryTypes";
import { DEFAULT_COLUMNS } from "./assetsDiscoveryTypes";

// ─── Defaults (used to decide what to omit) ───

const DEFAULT_MODE: SearchMode = "structured";
const DEFAULT_SORT = "-updated_at";
const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_VIEW: ViewMode = "table";

// ─── Helpers ───

function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

/**
 * Serialize a FilterChip into the URL-friendly `field:op:value` format.
 * Array values are joined with comma.
 */
function serializeFilterChip(chip: FilterChip): string {
  const val = Array.isArray(chip.value) ? chip.value.join(",") : chip.value;
  return `${chip.field}:${chip.op}:${val}`;
}

/**
 * Parse a `field:op:value` string back into a partial FilterChip.
 * The `id` and `source` are synthesized since they aren't in the URL.
 */
function parseFilterParam(raw: string): FilterChip | null {
  // Split on first two colons only — value may contain colons
  const firstColon = raw.indexOf(":");
  if (firstColon === -1) return null;
  const secondColon = raw.indexOf(":", firstColon + 1);
  if (secondColon === -1) return null;

  const field = raw.slice(0, firstColon);
  const op = raw.slice(firstColon + 1, secondColon);
  const rawValue = raw.slice(secondColon + 1);

  if (!field || !op) return null;

  // If value contains commas, treat as array
  const value = rawValue.includes(",") ? rawValue.split(",") : rawValue;

  return {
    id: `url_${field}_${op}_${typeof value === "string" ? value : value.join(",")}`,
    field,
    op,
    value,
    source: "search",
  };
}

// ─── Public API ───

/**
 * Read optional quick-preview asset id from the URL (`preview` query param).
 */
export function parsePreviewAssetIdFromUrl(sp: URLSearchParams): string | null {
  const raw = sp.get("preview");
  if (!raw) return null;
  const trimmed = raw.trim();
  return trimmed ? trimmed : null;
}

/**
 * Serialize a QueryState into URLSearchParams, omitting default values.
 * When `previewAssetId` is a non-empty string, adds `preview=<uuid>` for shareable workbench state.
 */
export function serializeQueryStateToUrl(state: QueryState, previewAssetId?: string | null): URLSearchParams {
  const sp = new URLSearchParams();

  if (state.searchMode !== DEFAULT_MODE) {
    sp.set("mode", state.searchMode);
  }

  if (state.queryText) {
    sp.set("q", state.queryText);
  }

  for (const chip of state.activeFilters) {
    sp.append("filter", serializeFilterChip(chip));
  }

  if (state.sort !== DEFAULT_SORT) {
    sp.set("sort", state.sort);
  }

  if (state.page !== DEFAULT_PAGE) {
    sp.set("page", String(state.page));
  }

  if (state.pageSize !== DEFAULT_PAGE_SIZE) {
    sp.set("page_size", String(state.pageSize));
  }

  if (state.viewMode !== DEFAULT_VIEW) {
    sp.set("view", state.viewMode);
  }

  if (!arraysEqual(state.selectedColumns, DEFAULT_COLUMNS)) {
    sp.set("columns", state.selectedColumns.join(","));
  }

  if (previewAssetId && previewAssetId.trim()) {
    sp.set("preview", previewAssetId.trim());
  }

  return sp;
}

/**
 * Parse URLSearchParams back into a partial QueryState.
 * Only fields present in the URL are included in the result.
 * Unknown params are silently ignored.
 */
export function parseQueryStateFromUrl(sp: URLSearchParams): Partial<QueryState> {
  const result: Partial<QueryState> = {};

  const mode = sp.get("mode");
  if (mode) {
    result.searchMode = mode as SearchMode;
  }

  const q = sp.get("q");
  if (q !== null) {
    result.queryText = q;
  }

  const filterParams = sp.getAll("filter");
  if (filterParams.length > 0) {
    const chips: FilterChip[] = [];
    for (const raw of filterParams) {
      const chip = parseFilterParam(raw);
      if (chip) chips.push(chip);
    }
    result.activeFilters = chips;
  }

  const sort = sp.get("sort");
  if (sort) {
    result.sort = sort;
  }

  const page = sp.get("page");
  if (page) {
    const n = Number.parseInt(page, 10);
    if (!Number.isNaN(n) && n > 0) {
      result.page = n;
    }
  }

  const pageSize = sp.get("page_size");
  if (pageSize) {
    const n = Number.parseInt(pageSize, 10);
    if (!Number.isNaN(n) && n > 0) {
      result.pageSize = n;
    }
  }

  const view = sp.get("view");
  if (view) {
    result.viewMode = view as ViewMode;
  }

  const columns = sp.get("columns");
  if (columns) {
    result.selectedColumns = columns.split(",").filter(Boolean);
  }

  return result;
}
