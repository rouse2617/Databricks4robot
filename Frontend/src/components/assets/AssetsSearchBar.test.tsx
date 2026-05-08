// @vitest-environment jsdom

import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import AssetsSearchBar, {
	getDraftIssues,
	getFieldSuggestions,
	parseOneToken,
	splitRespectingQuotes,
	tokenizeDraftText,
} from "./AssetsSearchBar";

beforeAll(() => {
	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
});

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
		const t = parseOneToken("duration_ms>60000");
		expect(t.field).toBe("duration_ms");
		expect(t.op).toBe("gt");
		expect(t.value).toBe("60000");
	});

	it("parses field>=value as gte", () => {
		const t = parseOneToken("duration_ms>=100000");
		expect(t.field).toBe("duration_ms");
		expect(t.op).toBe("gte");
		expect(t.value).toBe("100000");
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
		const tokens = tokenizeDraftText(
			"env:warehouse some free text algo_status:failed",
		);
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
		const tokens = tokenizeDraftText("duration_ms>60000 delivery_count<=3");
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

describe("getFieldSuggestions", () => {
	it("suggests matching structured fields from the current token prefix", () => {
		const suggestions = getFieldSuggestions("own");
		expect(suggestions.map((item) => item.key)).toContain("owner");
	});

	it("does not suggest after a structured operator is present", () => {
		expect(getFieldSuggestions("owner:alice")).toEqual([]);
	});
});

describe("getDraftIssues", () => {
	it("flags invalid enum values", () => {
		const issues = getDraftIssues("env:warehouze");
		expect(issues).toHaveLength(1);
		expect(issues[0].message).toContain("env");
	});

	it("flags invalid numeric values", () => {
		const issues = getDraftIssues("duration_ms:abc");
		expect(issues).toHaveLength(1);
		expect(issues[0].message).toContain("duration_ms");
	});
});

describe("AssetsSearchBar component", () => {
	it("renders field suggestions for partial structured prefixes", () => {
		render(
			<AssetsSearchBar
				searchMode="structured"
				draftText="lif"
				committedQueryText=""
				onDraftChange={vi.fn()}
				onCommitQuery={vi.fn()}
				onModeChange={vi.fn()}
			/>,
		);
		expect(screen.getByText("lifecycle_state:")).toBeTruthy();
	});

	it("shows immediate validation feedback for invalid structured tokens", () => {
		render(
			<AssetsSearchBar
				searchMode="structured"
				draftText="env:warehouze"
				committedQueryText=""
				onDraftChange={vi.fn()}
				onCommitQuery={vi.fn()}
				onModeChange={vi.fn()}
			/>,
		);
		expect(screen.getByTestId("search-draft-error").textContent).toContain(
			"env",
		);
	});

	it("blocks submit when the draft contains invalid structured tokens", () => {
		const onCommitQuery = vi.fn();
		render(
			<AssetsSearchBar
				searchMode="structured"
				draftText="duration_ms:abc"
				committedQueryText=""
				onDraftChange={vi.fn()}
				onCommitQuery={onCommitQuery}
				onModeChange={vi.fn()}
			/>,
		);
		fireEvent.keyDown(screen.getByTestId("assets-search-input"), {
			key: "Enter",
			code: "Enter",
		});
		expect(onCommitQuery).not.toHaveBeenCalled();
	});
});
