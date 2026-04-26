// @vitest-environment jsdom
import { describe, it, expect } from "vitest";
import { parseAlgoEntries, countStatuses } from "./AlgoSummaryCell";

// ─── parseAlgoEntries ───

describe("parseAlgoEntries", () => {
  it("returns empty array for undefined input", () => {
    expect(parseAlgoEntries(undefined)).toEqual([]);
  });

  it("returns empty array for empty object", () => {
    expect(parseAlgoEntries({})).toEqual([]);
  });

  it("extracts entries from keys ending in :status", () => {
    const results = parseAlgoEntries({
      "hand_tracking@1.2.0:status": "ok",
      "hand_tracking@1.2.0:run_id": "run_123",
      "body_tracking@1.0.0:status": "failed",
    });
    expect(results).toEqual([
      { key: "hand_tracking", status: "ok" },
      { key: "body_tracking", status: "failed" },
    ]);
  });

  it("ignores keys not ending in :status", () => {
    const results = parseAlgoEntries({
      "sam2@1.0.0:run_id": "abc",
      "sam2@1.0.0:output_uri": "gs://bucket/path",
    });
    expect(results).toEqual([]);
  });

  it("strips version from algo key", () => {
    const results = parseAlgoEntries({
      "head_tracking@2.0.0:status": "running",
    });
    expect(results[0].key).toBe("head_tracking");
  });
});

// ─── countStatuses ───

describe("countStatuses", () => {
  it("returns empty object for empty entries", () => {
    expect(countStatuses([])).toEqual({});
  });

  it("counts statuses correctly", () => {
    const entries = [
      { key: "a", status: "ok" },
      { key: "b", status: "ok" },
      { key: "c", status: "failed" },
      { key: "d", status: "pending" },
    ];
    expect(countStatuses(entries)).toEqual({ ok: 2, failed: 1, pending: 1 });
  });
});
