import { describe, it, expect } from "vitest";
import {
  splitRespectingQuotes,
  parseOneToken,
  tokenizeDraftText,
} from "./AssetsSearchBar";

// ─── splitRespectingQuotes ───

describe("splitRespectingQuotes", () => {
  it("splits simple whitespace-separated tokens", () => {
    expect(splitRespectingQuotes("env:warehouse algo_status:failed")).toEqual([
      "env:warehouse",
      "algo_status:failed",
    ]);
  });

  it("keeps quoted strings as single tokens", () => {
    expect(splitRespectingQuotes('"hello world" foo')).toEqual([
      "hello world",
      "foo",
    ]);
  });

  it("handles single quotes", () => {
    expect(splitRespectingQuotes("'hello world' bar")).toEqual([
      "hello world",
      "bar",
    ]);
  });

  it("returns empty array for empty/whitespace input", () => {
    expect(splitRespectingQuotes("")).toEqual([]);
    expect(splitRespectingQuotes("   ")).toEqual([]);
  });

  it("handles multiple spaces between tokens", () => {
    expect(splitRespectingQuotes("a   b")).toEqual(["a", "b"]);
  });
});

// ─── parseOneToken ───

describe("parseOneToken", () => {
  it("parses field:value as eq", () => {
    const t = parseOneToken("env:warehouse");
    expect(t.field).toBe("env");
    expect(t.op).toBe("eq");
    expect(t.value).toBe("warehouse");
  });

  it("parses field>value as gt", () => {
    const t = parseOneToken("duration_sec>60");
    expect(t.field).toBe("duration_sec");
    expect(t.op).toBe("gt");
    expect(t.value).toBe("60");
  });

  it("parses field>=value as gte", () => {
    const t = parseOneToken("duration_sec>=100");
    expect(t.field).toBe("duration_sec");
    expect(t.op).toBe("gte");
    expect(t.value).toBe("100");
  });

  it("parses field<value as lt", () => {
    const t = parseOneToken("delivery_count<5");
    expect(t.field).toBe("delivery_count");
    expect(t.op).toBe("lt");
    expect(t.value).toBe("5");
  });

  it("parses field<=value as lte", () => {
    const t = parseOneToken("delivery_count<=10");
    expect(t.field).toBe("delivery_count");
    expect(t.op).toBe("lte");
    expect(t.value).toBe("10");
  });

  it("parses field!=value as ne", () => {
    const t = parseOneToken("status!=archived");
    expect(t.field).toBe("status");
    expect(t.op).toBe("ne");
    expect(t.value).toBe("archived");
  });

  it("parses tag.priority:high (dotted field)", () => {
    const t = parseOneToken("tag.priority:high");
    expect(t.field).toBe("tag.priority");
    expect(t.op).toBe("eq");
    expect(t.value).toBe("high");
  });

  it("falls back to _fulltext for unknown fields", () => {
    const t = parseOneToken("unknownfield:value");
    expect(t.field).toBe("_fulltext");
    expect(t.op).toBe("ilike");
    expect(t.value).toBe("unknownfield:value");
  });

  it("falls back to _fulltext for plain text", () => {
    const t = parseOneToken("hello");
    expect(t.field).toBe("_fulltext");
    expect(t.op).toBe("ilike");
    expect(t.value).toBe("hello");
  });

  it("falls back to _fulltext when value is empty after operator", () => {
    const t = parseOneToken("env:");
    expect(t.field).toBe("_fulltext");
    expect(t.op).toBe("ilike");
    expect(t.value).toBe("env:");
  });

  it("generates unique ids with search_ prefix", () => {
    const t = parseOneToken("env:warehouse");
    expect(t.id).toMatch(/^search_env_/);
    expect(t.source).toBe("search");
  });
});

// ─── tokenizeDraftText ───

describe("tokenizeDraftText", () => {
  it("returns empty array for empty input", () => {
    expect(tokenizeDraftText("")).toEqual([]);
    expect(tokenizeDraftText("   ")).toEqual([]);
  });

  it("parses mixed field tokens and free text", () => {
    const tokens = tokenizeDraftText("env:warehouse some free text algo_status:failed");
    expect(tokens).toHaveLength(3);
    expect(tokens[0].field).toBe("env");
    expect(tokens[0].op).toBe("eq");
    expect(tokens[0].value).toBe("warehouse");
    expect(tokens[1].field).toBe("_fulltext");
    expect(tokens[1].op).toBe("ilike");
    expect(tokens[1].value).toBe("some free text");
    expect(tokens[2].field).toBe("algo_status");
    expect(tokens[2].op).toBe("eq");
    expect(tokens[2].value).toBe("failed");
  });

  it("merges adjacent free text into one _fulltext token", () => {
    const tokens = tokenizeDraftText("hello world");
    expect(tokens).toHaveLength(1);
    expect(tokens[0].field).toBe("_fulltext");
    expect(tokens[0].value).toBe("hello world");
  });

  it("handles only structured tokens", () => {
    const tokens = tokenizeDraftText("env:warehouse status:approved");
    expect(tokens).toHaveLength(2);
    expect(tokens[0].field).toBe("env");
    expect(tokens[1].field).toBe("status");
  });

  it("handles comparison operators", () => {
    const tokens = tokenizeDraftText("duration_sec>60 delivery_count<=3");
    expect(tokens).toHaveLength(2);
    expect(tokens[0].op).toBe("gt");
    expect(tokens[1].op).toBe("lte");
  });

  it("all tokens have source 'search'", () => {
    const tokens = tokenizeDraftText("env:warehouse hello");
    for (const t of tokens) {
      expect(t.source).toBe("search");
    }
  });
});
