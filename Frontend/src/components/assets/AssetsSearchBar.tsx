// ─── AssetsSearchBar — Presentational Component ───
// Mode selector + search input + help button + tokenizer.
// Validates: Requirements R2

import { Select, Input, Button, Tooltip } from "antd";
import { SearchOutlined, QuestionCircleOutlined } from "@ant-design/icons";
import type { SearchMode, QueryToken } from "../../lib/assets/assetsDiscoveryTypes";

// ─── Known Fields ───

const KNOWN_FIELDS = new Set([
  "asset_id",
  "mcap_file_id",
  "owner",
  "reviewer",
  "lifecycle_state",
  "status",
  "env",
  "scene",
  "task",
  "batch",
  "duration_ms",
  "created_at",
  "updated_at",
  "delivery_count",
  "tag.priority",
  "tag.quality",
  "tag.notes",
  "algo_status",
]);

// ─── Operator Patterns (order matters: >= before >, <= before <, != before :) ───

const OP_PATTERNS: { pattern: string; op: string }[] = [
  { pattern: ">=", op: "gte" },
  { pattern: "<=", op: "lte" },
  { pattern: "!=", op: "ne" },
  { pattern: ">", op: "gt" },
  { pattern: "<", op: "lt" },
  { pattern: ":", op: "eq" },
];

// ─── Tokenizer ───

/**
 * Split input by whitespace, respecting quoted strings.
 * e.g. `env:warehouse "hello world"` → ["env:warehouse", "hello world"]
 */
export function splitRespectingQuotes(input: string): string[] {
  const tokens: string[] = [];
  let current = "";
  let inQuote: string | null = null;

  for (const ch of input) {
    if (inQuote) {
      if (ch === inQuote) {
        inQuote = null;
      } else {
        current += ch;
      }
    } else if (ch === '"' || ch === "'") {
      inQuote = ch;
    } else if (/\s/.test(ch)) {
      if (current) {
        tokens.push(current);
        current = "";
      }
    } else {
      current += ch;
    }
  }
  if (current) tokens.push(current);
  return tokens;
}

/**
 * Parse a single raw token string into a QueryToken.
 * Tries to match `field<op>value` patterns against known fields.
 * Falls back to `_fulltext` with op "ilike".
 */
export function parseOneToken(raw: string): QueryToken {
  for (const { pattern, op } of OP_PATTERNS) {
    const idx = raw.indexOf(pattern);
    if (idx > 0) {
      const field = raw.slice(0, idx);
      const value = raw.slice(idx + pattern.length);
      if (KNOWN_FIELDS.has(field) && value.length > 0) {
        return {
          id: `search_${field}_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
          field,
          op,
          value,
          source: "search",
        };
      }
    }
  }
  // Unmatched → fulltext
  return {
    id: `search__fulltext_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
    field: "_fulltext",
    op: "ilike",
    value: raw,
    source: "search",
  };
}

/**
 * Parse the full draft text into an array of QueryTokens.
 * Splits by whitespace (respecting quotes), then parses each piece.
 * Adjacent fulltext tokens are merged into one.
 */
export function tokenizeDraftText(text: string): QueryToken[] {
  const trimmed = text.trim();
  if (!trimmed) return [];

  const rawParts = splitRespectingQuotes(trimmed);
  const tokens: QueryToken[] = [];
  const fulltextParts: string[] = [];

  for (const part of rawParts) {
    const token = parseOneToken(part);
    if (token.field === "_fulltext") {
      fulltextParts.push(token.value as string);
    } else {
      // Flush accumulated fulltext first
      if (fulltextParts.length > 0) {
        tokens.push({
          id: `search__fulltext_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
          field: "_fulltext",
          op: "ilike",
          value: fulltextParts.join(" "),
          source: "search",
        });
        fulltextParts.length = 0;
      }
      tokens.push(token);
    }
  }

  // Flush remaining fulltext
  if (fulltextParts.length > 0) {
    tokens.push({
      id: `search__fulltext_${Date.now()}_${Math.random().toString(36).slice(2, 6)}`,
      field: "_fulltext",
      op: "ilike",
      value: fulltextParts.join(" "),
      source: "search",
    });
  }

  return tokens;
}

// ─── Search Mode Options ───

const SEARCH_MODE_OPTIONS: {
  value: SearchMode;
  label: string;
  disabled: boolean;
  tooltip?: string;
}[] = [
  { value: "structured", label: "Structured", disabled: false },
  { value: "keyword", label: "Keyword", disabled: false },
  { value: "semantic", label: "Semantic", disabled: true, tooltip: "Coming soon" },
  { value: "similar", label: "Similar", disabled: true, tooltip: "Coming soon" },
];

// ─── Props ───

export interface AssetsSearchBarProps {
  searchMode: SearchMode;
  draftText: string;
  committedQueryText: string;
  onDraftChange: (text: string) => void;
  onCommitQuery: (text: string, tokens: QueryToken[]) => void;
  onModeChange: (mode: SearchMode) => void;
}

// ─── Component ───

export default function AssetsSearchBar({
  searchMode,
  draftText,
  onDraftChange,
  onCommitQuery,
  onModeChange,
}: AssetsSearchBarProps) {
  const handlePressEnter = () => {
    const tokens = tokenizeDraftText(draftText);
    onCommitQuery(draftText.trim(), tokens);
  };

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      {/* Search Mode Selector */}
      <Select
        value={searchMode}
        onChange={onModeChange}
        style={{ width: 130 }}
        size="middle"
      >
        {SEARCH_MODE_OPTIONS.map((opt) =>
          opt.disabled ? (
            <Select.Option key={opt.value} value={opt.value} disabled>
              <Tooltip title={opt.tooltip}>{opt.label}</Tooltip>
            </Select.Option>
          ) : (
            <Select.Option key={opt.value} value={opt.value}>
              {opt.label}
            </Select.Option>
          ),
        )}
      </Select>

      {/* Search Input */}
      <Input
        placeholder="搜索 Asset、MCAP、Owner、Tag，或输入 env:warehouse algo_status:failed"
        prefix={<SearchOutlined />}
        value={draftText}
        onChange={(e) => onDraftChange(e.target.value)}
        onPressEnter={handlePressEnter}
        allowClear
        onClear={() => onCommitQuery("", [])}
        style={{ flex: 1 }}
        aria-label="资产搜索"
        data-testid="assets-search-input"
      />

      {/* Help Button */}
      <Tooltip
        title={
          <div>
            <div style={{ fontWeight: 600, marginBottom: 4 }}>支持的搜索语法</div>
            <div><code>field:value</code> — 精确匹配</div>
            <div><code>field&gt;value</code> — 大于</div>
            <div><code>field&gt;=value</code> — 大于等于</div>
            <div><code>field&lt;value</code> — 小于</div>
            <div><code>field&lt;=value</code> — 小于等于</div>
            <div><code>field!=value</code> — 不等于</div>
            <div style={{ marginTop: 4, fontSize: 12, opacity: 0.85 }}>
              可用字段: env, lifecycle_state, owner, algo_status, duration_ms, tag.priority, tag.quality 等
            </div>
          </div>
        }
      >
        <Button icon={<QuestionCircleOutlined />} type="text" aria-label="搜索帮助" />
      </Tooltip>
    </div>
  );
}
