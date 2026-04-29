import { describe, it, expect } from "vitest";
import {
  serializeQueryStateToUrl,
  parseQueryStateFromUrl,
} from "./assetsDiscoveryUrl";
import {
  DEFAULT_COLUMNS,
  type QueryState,
  type FilterChip,
} from "./assetsDiscoveryTypes";

// ─── Helpers ───

function defaultQueryState(): QueryState {
  return {
    searchMode: "structured",
    queryText: "",
    activeFilters: [],
    sort: "-updated_at",
    page: 1,
    pageSize: 20,
    viewMode: "table",
    selectedColumns: [...DEFAULT_COLUMNS],
  };
}

function chip(
  field: string,
  op: string,
  value: string | string[],
  source: FilterChip["source"] = "facet",
): FilterChip {
  const v = Array.isArray(value) ? value.join(",") : value;
  return { id: `${field}_${op}_${v}`, field, op, value, source };
}

// ─── serializeQueryStateToUrl ───

describe("serializeQueryStateToUrl", () => {
  it("returns empty params for default state", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.toString()).toBe("");
  });

  it("includes mode when not default", () => {
    const state = { ...defaultQueryState(), searchMode: "keyword" as const };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("mode")).toBe("keyword");
  });

  it("omits mode when default (structured)", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("mode")).toBe(false);
  });

  it("includes q when queryText is non-empty", () => {
    const state = { ...defaultQueryState(), queryText: "env:warehouse" };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("q")).toBe("env:warehouse");
  });

  it("omits q when queryText is empty", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("q")).toBe(false);
  });

  it("serializes filter chips as repeatable params", () => {
    const state = {
      ...defaultQueryState(),
      activeFilters: [
        chip("status", "eq", "approved"),
        chip("env", "eq", "warehouse"),
      ],
    };
    const sp = serializeQueryStateToUrl(state);
    const filters = sp.getAll("filter");
    expect(filters).toHaveLength(2);
    expect(filters).toContain("status:eq:approved");
    expect(filters).toContain("env:eq:warehouse");
  });

  it("serializes array filter values with comma", () => {
    const state = {
      ...defaultQueryState(),
      activeFilters: [chip("duration_ms", "between", ["10", "200"])],
    };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.getAll("filter")).toEqual(["duration_ms:between:10,200"]);
  });

  it("includes sort when not default", () => {
    const state = { ...defaultQueryState(), sort: "created_at" };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("sort")).toBe("created_at");
  });

  it("omits sort when default (-updated_at)", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("sort")).toBe(false);
  });

  it("includes page when not 1", () => {
    const state = { ...defaultQueryState(), page: 3 };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("page")).toBe("3");
  });

  it("omits page when 1", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("page")).toBe(false);
  });

  it("includes page_size when not 20", () => {
    const state = { ...defaultQueryState(), pageSize: 50 };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("page_size")).toBe("50");
  });

  it("omits page_size when 20", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("page_size")).toBe(false);
  });

  it("includes view when not table", () => {
    const state = { ...defaultQueryState(), viewMode: "compact" as const };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("view")).toBe("compact");
  });

  it("omits view when table", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("view")).toBe(false);
  });

  it("includes columns when different from default", () => {
    const state = {
      ...defaultQueryState(),
      selectedColumns: ["asset_id", "env"],
    };
    const sp = serializeQueryStateToUrl(state);
    expect(sp.get("columns")).toBe("asset_id,env");
  });

  it("omits columns when matching default", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    expect(sp.has("columns")).toBe(false);
  });
});

// ─── parseQueryStateFromUrl ───

describe("parseQueryStateFromUrl", () => {
  it("returns empty object for empty params", () => {
    const result = parseQueryStateFromUrl(new URLSearchParams());
    expect(result).toEqual({});
  });

  it("parses mode", () => {
    const sp = new URLSearchParams("mode=keyword");
    expect(parseQueryStateFromUrl(sp).searchMode).toBe("keyword");
  });

  it("parses q", () => {
    const sp = new URLSearchParams("q=env:warehouse");
    expect(parseQueryStateFromUrl(sp).queryText).toBe("env:warehouse");
  });

  it("parses repeatable filter params", () => {
    const sp = new URLSearchParams();
    sp.append("filter", "status:eq:approved");
    sp.append("filter", "env:eq:warehouse");
    const result = parseQueryStateFromUrl(sp);
    expect(result.activeFilters).toHaveLength(2);
    expect(result.activeFilters![0].field).toBe("status");
    expect(result.activeFilters![0].op).toBe("eq");
    expect(result.activeFilters![0].value).toBe("approved");
    expect(result.activeFilters![1].field).toBe("env");
    expect(result.activeFilters![1].value).toBe("warehouse");
  });

  it("parses filter with array value (comma-separated)", () => {
    const sp = new URLSearchParams("filter=duration_ms:between:10,200");
    const result = parseQueryStateFromUrl(sp);
    expect(result.activeFilters).toHaveLength(1);
    expect(result.activeFilters![0].value).toEqual(["10", "200"]);
  });

  it("parses sort", () => {
    const sp = new URLSearchParams("sort=created_at");
    expect(parseQueryStateFromUrl(sp).sort).toBe("created_at");
  });

  it("parses page as number", () => {
    const sp = new URLSearchParams("page=5");
    expect(parseQueryStateFromUrl(sp).page).toBe(5);
  });

  it("ignores invalid page values", () => {
    const sp = new URLSearchParams("page=abc");
    expect(parseQueryStateFromUrl(sp).page).toBeUndefined();
  });

  it("ignores non-positive page values", () => {
    const sp = new URLSearchParams("page=0");
    expect(parseQueryStateFromUrl(sp).page).toBeUndefined();
  });

  it("parses page_size as number", () => {
    const sp = new URLSearchParams("page_size=50");
    expect(parseQueryStateFromUrl(sp).pageSize).toBe(50);
  });

  it("parses view", () => {
    const sp = new URLSearchParams("view=compact");
    expect(parseQueryStateFromUrl(sp).viewMode).toBe("compact");
  });

  it("parses columns as comma-separated", () => {
    const sp = new URLSearchParams("columns=asset_id,env,duration");
    expect(parseQueryStateFromUrl(sp).selectedColumns).toEqual([
      "asset_id",
      "env",
      "duration",
    ]);
  });

  it("ignores unknown params gracefully", () => {
    const sp = new URLSearchParams("foo=bar&baz=qux&page=2");
    const result = parseQueryStateFromUrl(sp);
    expect(result.page).toBe(2);
    expect((result as Record<string, unknown>)["foo"]).toBeUndefined();
    expect((result as Record<string, unknown>)["baz"]).toBeUndefined();
  });

  it("handles malformed filter params gracefully", () => {
    const sp = new URLSearchParams();
    sp.append("filter", "badformat");
    sp.append("filter", "also:bad");
    sp.append("filter", "status:eq:approved");
    const result = parseQueryStateFromUrl(sp);
    // Only the valid one is parsed
    expect(result.activeFilters).toHaveLength(1);
    expect(result.activeFilters![0].field).toBe("status");
  });

  it("handles filter value containing colons", () => {
    const sp = new URLSearchParams("filter=tag.notes:ilike:foo:bar:baz");
    const result = parseQueryStateFromUrl(sp);
    expect(result.activeFilters).toHaveLength(1);
    expect(result.activeFilters![0].field).toBe("tag.notes");
    expect(result.activeFilters![0].op).toBe("ilike");
    expect(result.activeFilters![0].value).toBe("foo:bar:baz");
  });
});

// ─── Roundtrip: serialize → parse ───

describe("roundtrip serialize → parse", () => {
  it("roundtrips a complex state", () => {
    const state: QueryState = {
      searchMode: "keyword",
      queryText: "env:warehouse algo_status:failed",
      activeFilters: [
        chip("tag.priority", "eq", "high"),
        chip("status", "eq", "approved"),
      ],
      sort: "created_at",
      page: 3,
      pageSize: 50,
      viewMode: "compact",
      selectedColumns: ["asset_id", "env", "duration"],
    };

    const sp = serializeQueryStateToUrl(state);
    const parsed = parseQueryStateFromUrl(sp);

    expect(parsed.searchMode).toBe("keyword");
    expect(parsed.queryText).toBe("env:warehouse algo_status:failed");
    expect(parsed.activeFilters).toHaveLength(2);
    expect(parsed.activeFilters![0].field).toBe("tag.priority");
    expect(parsed.activeFilters![1].field).toBe("status");
    expect(parsed.sort).toBe("created_at");
    expect(parsed.page).toBe(3);
    expect(parsed.pageSize).toBe(50);
    expect(parsed.viewMode).toBe("compact");
    expect(parsed.selectedColumns).toEqual(["asset_id", "env", "duration"]);
  });

  it("roundtrips default state to empty", () => {
    const sp = serializeQueryStateToUrl(defaultQueryState());
    const parsed = parseQueryStateFromUrl(sp);
    expect(parsed).toEqual({});
  });
});
