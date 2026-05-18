// ─── AlgoSummaryCell — Presentational Component ───
// Renders "N ok / M failed / K pending" summary with hover popover.
// Validates: Requirements R6

import { Popover, Typography } from "antd";
import type { Asset } from "../../api/types";

const { Text } = Typography;

// ─── Status Color Map ───

const STATUS_COLORS: Record<string, string> = {
	ok: "#52c41a",
	failed: "#ff4d4f",
	running: "#faad14",
	pending: "#8c8c8c",
	blocked: "#8c8c8c",
};

// ─── Helpers ───

export interface AlgoEntry {
	key: string;
	status: string;
}

/**
 * Parse algo_results map: iterate keys ending in `:status`, extract algo name and status.
 */
export function parseAlgoEntries(
	algoResults: Record<string, string> | undefined,
): AlgoEntry[] {
	if (!algoResults) return [];
	const entries: AlgoEntry[] = [];
	for (const [k, v] of Object.entries(algoResults)) {
		if (k.endsWith(":status")) {
			const algoKey = k.replace(":status", "");
			const name = algoKey.split("@")[0];
			entries.push({ key: name, status: v });
		}
	}
	return entries;
}

/**
 * Count statuses from algo entries.
 */
export function countStatuses(entries: AlgoEntry[]): Record<string, number> {
	const counts: Record<string, number> = {};
	for (const e of entries) {
		counts[e.status] = (counts[e.status] ?? 0) + 1;
	}
	return counts;
}

// ─── Popover Content ───

function AlgoPopoverContent({ entries }: { entries: AlgoEntry[] }) {
	return (
		<div style={{ minWidth: 200 }}>
			{entries.map((e) => (
				<div
					key={e.key}
					style={{
						display: "flex",
						justifyContent: "space-between",
						gap: 16,
						padding: "2px 0",
					}}
				>
					<Text style={{ fontSize: 12 }}>{e.key}</Text>
					<Text
						style={{
							fontSize: 12,
							color: STATUS_COLORS[e.status] ?? "#8c8c8c",
						}}
					>
						{e.status}
					</Text>
				</div>
			))}
		</div>
	);
}

// ─── Props ───

export interface AlgoSummaryCellProps {
	record: Asset;
}

// ─── Component ───

export default function AlgoSummaryCell({ record }: AlgoSummaryCellProps) {
	const entries = parseAlgoEntries(record.algo_results);
	if (entries.length === 0) {
		return <Text type="secondary">—</Text>;
	}

	const counts = countStatuses(entries);

	// Build summary parts in display order: ok, failed, then others
	const ORDER = ["ok", "failed", "running", "pending", "blocked"];
	const parts: { label: string; count: number; color?: string }[] = [];
	for (const status of ORDER) {
		if (counts[status]) {
			parts.push({
				label: status,
				count: counts[status],
				color: status === "failed" ? "#ff4d4f" : undefined,
			});
		}
	}
	// Any remaining statuses not in ORDER
	for (const [status, count] of Object.entries(counts)) {
		if (!ORDER.includes(status)) {
			parts.push({ label: status, count });
		}
	}

	return (
		<Popover
			content={<AlgoPopoverContent entries={entries} />}
			title="算法详情"
			trigger="hover"
			placement="bottomLeft"
		>
			<span style={{ cursor: "pointer", fontSize: 12 }}>
				{parts.map((p, i) => (
					<span key={p.label}>
						{i > 0 && " / "}
						<span style={p.color ? { color: p.color } : undefined}>
							{p.count} {p.label}
						</span>
					</span>
				))}
			</span>
		</Popover>
	);
}
