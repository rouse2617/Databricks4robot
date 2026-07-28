// CYB-4304: shared types for the preset-driven `BatchAssetLookup` component.
//
// The shell (see `BatchAssetLookup.tsx`) is preset-agnostic — every batch
// lookup (durations, lineage, cost, …) plugs a `BatchLookupPreset<Item, F>`
// into the same component. Presets own their columns, filters, histogram,
// CSV rows and the fetch call; the shell only wires state and rendering.

import type { ReactNode } from "react";
import type { ColumnsType } from "antd/es/table";

// ─── Request / response envelope ──────────────────────────────────────────

export type BatchLookupIdType = "auto" | "asset_id" | "grace_video_id";

export interface BatchLookupRequest<Filters = Record<string, unknown>> {
	ids: string[];
	id_type: BatchLookupIdType;
	filters: Filters;
}

export interface BatchLookupResponse<Item> {
	items: Item[];
	missing_ids: string[];
	// Presets whose backend doesn't range-filter server-side may omit this.
	filtered_out_ids?: string[];
	// Arbitrary numeric bag — preset.extraStats interprets it. The shell only
	// renders the always-present matched/missing/filtered_out cards on its own.
	stats: Record<string, number>;
}

// ─── UI atoms the shell renders on behalf of the preset ───────────────────

export interface StatEntry {
	label: string;
	value: string | number;
	hint?: string;
}

export interface BucketRow {
	label: string;
	count: number;
}

// ─── Preset contract ──────────────────────────────────────────────────────

export interface BatchLookupPreset<
	Item extends { input_id: string },
	Filters = Record<string, unknown>,
> {
	// Short slug used for logging/tests only — e.g. "durations".
	key: string;
	// Placeholder shown inside the paste textarea.
	placeholder: string;
	// Hard cap on how many ids the shell will submit.
	maxIds: number;
	// Preset does the network call and returns the raw response envelope.
	fetch: (
		req: BatchLookupRequest<Filters>,
	) => Promise<BatchLookupResponse<Item>>;
	// Extra filter inputs rendered below the id_type radio.
	filters?: {
		initial: Filters;
		render: (
			filters: Filters,
			setFilters: (next: Filters) => void,
		) => ReactNode;
		// Return null when valid, or an error string that is shown as a hint
		// and used to disable the Submit button.
		validate?: (filters: Filters) => string | null;
	};
	// AntD Table columns for the results row. Presets whose column set
	// depends on filters (e.g. costs' group_by=asset_algo adds an algo_key
	// column) can pass a function; the shell resolves it against the
	// current filters on every render.
	columns: ColumnsType<Item> | ((filters: Filters) => ColumnsType<Item>);
	// Preset-specific stat cards rendered after the always-present three.
	extraStats?: (res: BatchLookupResponse<Item>) => StatEntry[];
	// Optional histogram rendered above the results Table.
	histogram?: (items: Item[]) => BucketRow[];
	// CSV export config. filename receives the matched (items.length) count
	// plus the filters that were used to fetch — presets whose filename
	// wants to include the query window (e.g. costs) read those directly;
	// simpler presets ignore the second arg.
	csv: {
		filename: (matchedCount: number, filters?: Filters) => string;
		header: string[];
		row: (item: Item) => (string | number)[];
	};
}

// ─── Paste-blob parsing helper ────────────────────────────────────────────

// Parse a paste blob into a list of ids. Splits on any whitespace or comma,
// trims, drops empties. Preserves order. Returns { ids, duplicates } where
// duplicates counts the number of first-hit duplicates the user pasted.
// (Moved verbatim from CYB-4294's AssetDurationLookup.tsx so every preset
// hits the same parser.)
export function parseIdBlob(raw: string): {
	ids: string[];
	duplicates: number;
} {
	if (!raw) return { ids: [], duplicates: 0 };
	const seen = new Set<string>();
	const out: string[] = [];
	let dupCount = 0;
	for (const token of raw.split(/[\s,]+/)) {
		const t = token.trim();
		if (!t) continue;
		if (seen.has(t)) {
			dupCount += 1;
			continue;
		}
		seen.add(t);
		out.push(t);
	}
	return { ids: out, duplicates: dupCount };
}
