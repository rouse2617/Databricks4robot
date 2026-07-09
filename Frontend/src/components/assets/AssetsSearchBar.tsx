// ─── AssetsSearchBar — Presentational Component ───
// Mode selector + search input + help button + tokenizer.
// Validates: Requirements R2

import { QuestionCircleOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Input, Select, Space, Tag, Tooltip, Typography } from "antd";
import { useCallback, useRef } from "react";
import type {
	QueryToken,
	SearchMode,
} from "../../lib/assets/assetsDiscoveryTypes";
import { SEARCH_MODE_LABELS } from "../../lib/productVocabulary";

const { Text } = Typography;

type SearchFieldType = "enum" | "numeric" | "timestamp" | "string";

interface SearchFieldSpec {
	key: string;
	label: string;
	type: SearchFieldType;
	values?: string[];
}

// ─── Known Fields ───

export const SEARCH_FIELD_SPECS: SearchFieldSpec[] = [
	{ key: "asset_id", label: "Asset ID", type: "string" },
	{ key: "mcap_file_id", label: "MCAP ID", type: "string" },
	{ key: "owner", label: "Owner", type: "string" },
	{ key: "reviewer", label: "Reviewer", type: "string" },
	{
		key: "lifecycle_state",
		label: "生命周期",
		type: "enum",
		values: [
			"created",
			"processing",
			"ready",
			"rejected",
			"delivered",
			"archived",
			"superseded",
		],
	},
	{
		key: "status",
		label: "状态(legacy)",
		type: "enum",
		values: ["approved", "rejected", "superseded", "archived"],
	},
	{
		key: "env",
		label: "环境",
		type: "enum",
		values: ["kitchen", "outdoor", "warehouse", "office", "factory"],
	},
	{ key: "scene", label: "场景", type: "string" },
	{ key: "task", label: "任务", type: "string" },
	{ key: "batch", label: "批次", type: "string" },
	{ key: "duration_ms", label: "时长", type: "numeric" },
	{ key: "created_at", label: "创建时间", type: "timestamp" },
	{ key: "updated_at", label: "更新时间", type: "timestamp" },
	{ key: "delivery_count", label: "交付次数", type: "numeric" },
	{ key: "tag.priority", label: "优先级", type: "string" },
	{ key: "tag.quality", label: "质量", type: "string" },
	{ key: "tag.notes", label: "备注", type: "string" },
	{
		key: "algo_status",
		label: "算法状态",
		type: "enum",
		values: ["ok", "failed", "running", "pending", "blocked"],
	},
];

const KNOWN_FIELDS = new Set(SEARCH_FIELD_SPECS.map((field) => field.key));
const FIELD_SPEC_BY_KEY = new Map(
	SEARCH_FIELD_SPECS.map((field) => [field.key, field]),
);

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

interface StructuredCandidate {
	field: string;
	op: string;
	value: string;
}

export interface DraftIssue {
	token: string;
	message: string;
}

function extractStructuredCandidate(raw: string): StructuredCandidate | null {
	for (const { pattern, op } of OP_PATTERNS) {
		const idx = raw.indexOf(pattern);
		if (idx > 0) {
			return {
				field: raw.slice(0, idx),
				op,
				value: raw.slice(idx + pattern.length),
			};
		}
	}
	return null;
}

function isNumericValue(value: string): boolean {
	if (value.trim() === "") return false;
	return !Number.isNaN(Number(value));
}

function isTimestampValue(value: string): boolean {
	if (value.trim() === "") return false;
	return !Number.isNaN(Date.parse(value));
}

export function getDraftIssues(text: string): DraftIssue[] {
	return splitRespectingQuotes(text.trim())
		.map((raw): DraftIssue | null => {
			const candidate = extractStructuredCandidate(raw);
			if (!candidate) return null;

			const spec = FIELD_SPEC_BY_KEY.get(candidate.field);
			if (!spec) return null;

			if (!candidate.value.trim()) {
				return {
					token: raw,
					message: `字段 ${candidate.field} 还缺少值`,
				};
			}

			if (spec.type === "enum") {
				if (!["eq", "ne"].includes(candidate.op)) {
					return {
						token: raw,
						message: `字段 ${candidate.field} 只支持 = 和 !=`,
					};
				}
				if (spec.values && !spec.values.includes(candidate.value)) {
					return {
						token: raw,
						message: `字段 ${candidate.field} 不支持值 ${candidate.value}`,
					};
				}
			}

			if (spec.type === "numeric" && !isNumericValue(candidate.value)) {
				return {
					token: raw,
					message: `字段 ${candidate.field} 需要数值，例如 ${candidate.field}:60000`,
				};
			}

			if (spec.type === "timestamp" && !isTimestampValue(candidate.value)) {
				return {
					token: raw,
					message: `字段 ${candidate.field} 需要有效时间，例如 ${candidate.field}:2026-05-01T00:00:00Z`,
				};
			}

			return null;
		})
		.filter((issue): issue is DraftIssue => issue !== null);
}

function replaceCurrentToken(text: string, replacement: string): string {
	if (text === "" || /\s$/.test(text)) {
		return `${text}${replacement}`;
	}
	const lastSpaceIdx = Math.max(
		text.lastIndexOf(" "),
		text.lastIndexOf("\n"),
		text.lastIndexOf("\t"),
	);
	if (lastSpaceIdx === -1) {
		return replacement;
	}
	return `${text.slice(0, lastSpaceIdx + 1)}${replacement}`;
}

export function getFieldSuggestions(text: string): SearchFieldSpec[] {
	const parts = splitRespectingQuotes(text);
	const currentToken = /\s$/.test(text)
		? ""
		: parts.length > 0
			? parts[parts.length - 1]
			: "";
	if (!currentToken || extractStructuredCandidate(currentToken)) {
		return [];
	}
	const needle = currentToken.toLowerCase();
	return SEARCH_FIELD_SPECS.filter((field) =>
		field.key.startsWith(needle),
	).slice(0, 5);
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

function getSearchPlaceholder(mode: SearchMode): string {
	const placeholders: Record<SearchMode, string> = {
		structured: "例如：env:warehouse algo_status:failed asset_type:video…",
		keyword: "搜索关键词：Asset ID、MCAP、Owner、Tag 等…",
		semantic: "用自然语言描述你要找什么…（例如：失败的视频处理）",
		similar: "上传或指定资产 ID 来寻找相似内容…",
	};
	return placeholders[mode] ?? "输入搜索条件…";
}

const SEARCH_MODE_OPTIONS: {
	value: SearchMode;
	label: string;
	disabled: boolean;
	tooltip?: string;
}[] = [
	{
		value: "structured",
		label: SEARCH_MODE_LABELS.structured,
		disabled: false,
	},
	{ value: "keyword", label: SEARCH_MODE_LABELS.keyword, disabled: false },
	{
		value: "semantic",
		label: SEARCH_MODE_LABELS.semantic,
		disabled: false,
	},
	{
		value: "similar",
		label: SEARCH_MODE_LABELS.similar,
		disabled: false,
	},
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
	committedQueryText,
	onDraftChange,
	onCommitQuery,
	onModeChange,
}: AssetsSearchBarProps) {
	const draftIssues = getDraftIssues(draftText);
	const suggestions = getFieldSuggestions(draftText);
	const commitLockRef = useRef(false);

	const tryCommitQuery = useCallback(() => {
		if (commitLockRef.current) return;
		const issues = getDraftIssues(draftText);
		if (issues.length > 0) return;
		const trimmed = draftText.trim();
		if (trimmed === committedQueryText.trim()) return;

		commitLockRef.current = true;
		try {
			const tokens = tokenizeDraftText(draftText);
			onCommitQuery(trimmed, tokens);
		} finally {
			queueMicrotask(() => {
				commitLockRef.current = false;
			});
		}
	}, [committedQueryText, draftText, onCommitQuery]);

	return (
		<div style={{ display: "flex", alignItems: "flex-start", gap: 8 }}>
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

			<div style={{ flex: 1, minWidth: 0 }}>
				{/* Search Input */}
				<Input
					id="assets-search-input"
					placeholder={getSearchPlaceholder(searchMode)}
					prefix={<SearchOutlined />}
					value={draftText}
					onChange={(e) => onDraftChange(e.target.value)}
					onPressEnter={() => {
						tryCommitQuery();
					}}
					onBlur={() => {
						tryCommitQuery();
					}}
					allowClear
					onClear={() => onCommitQuery("", [])}
					style={{ flex: 1 }}
					status={draftIssues.length > 0 ? "error" : undefined}
					aria-label="资产搜索"
					name="assets_search"
					autoComplete="off"
					data-testid="assets-search-input"
				/>
				{(suggestions.length > 0 || draftIssues.length > 0) && (
					<div
						style={{
							marginTop: 6,
							display: "flex",
							flexDirection: "column",
							gap: 6,
						}}
					>
						{suggestions.length > 0 && (
							<Space size={4} wrap>
								<Text type="secondary" style={{ fontSize: 12 }}>
									字段建议
								</Text>
								{suggestions.map((field) => (
									<Tag
										key={field.key}
										color="blue"
										style={{ marginInlineEnd: 0 }}
									>
										<button
											type="button"
											className="link-like-button"
											aria-label={`使用字段建议 ${field.key}`}
											style={{ cursor: "pointer" }}
											onClick={() =>
												onDraftChange(
													replaceCurrentToken(draftText, `${field.key}:`),
												)
											}
										>
											{field.key}:
										</button>
									</Tag>
								))}
							</Space>
						)}
						{draftIssues.length > 0 && (
							<Text
								type="danger"
								style={{ fontSize: 12 }}
								data-testid="search-draft-error"
							>
								{draftIssues[0].message}
							</Text>
						)}
					</div>
				)}
			</div>

			{/* Help Button */}
			<Tooltip
				title={
					<div>
						<div style={{ fontWeight: 600, marginBottom: 4 }}>
							支持的搜索语法
						</div>
						<div>
							<code>field:value</code> — 精确匹配
						</div>
						<div>
							<code>field&gt;value</code> — 大于
						</div>
						<div>
							<code>field&gt;=value</code> — 大于等于
						</div>
						<div>
							<code>field&lt;value</code> — 小于
						</div>
						<div>
							<code>field&lt;=value</code> — 小于等于
						</div>
						<div>
							<code>field!=value</code> — 不等于
						</div>
						<div style={{ marginTop: 4, fontSize: 12, opacity: 0.85 }}>
							可用字段: env, lifecycle_state, owner, algo_status, duration_ms,
							tag.priority, tag.quality 等
						</div>
					</div>
				}
			>
				<Button
					icon={<QuestionCircleOutlined />}
					type="text"
					aria-label="搜索帮助"
				/>
			</Tooltip>
		</div>
	);
}
