import type { ElasticQuota, ExecutionTarget } from "../../api/pipelineApi";

// Standard Koordinator ElasticQuota pod-label key. This is a frontend-only
// convenience constant — it mirrors what the pool editor writes into
// resource_defaults.scheduling.podLabels so the Deploy flow can map a pool back
// to its live ElasticQuota usage. The backend stays generic (PR #439) and never
// depends on this key.
export const KOORD_EQ_LABEL_KEY = "quota.scheduling.koordinator.sh/name";

// clusterKey normalizes a target's cluster for the eqsByCluster lookup map.
// Targets without an explicit clusterId fall back to the default cluster, keyed
// by "" (the same empty value listElasticQuotas() treats as "default cluster").
export function clusterKey(target: Pick<ExecutionTarget, "clusterId">): string {
	return target.clusterId ?? "";
}

// poolEqName returns the ElasticQuota a pool is wired to (the koord EQ pod-label
// value), or undefined for a coarse namespace-level pool with no EQ configured.
export function poolEqName(target: ExecutionTarget): string | undefined {
	const raw =
		target.resourceDefaults?.scheduling?.podLabels?.[KOORD_EQ_LABEL_KEY];
	const name = (raw ?? "").trim();
	return name || undefined;
}

// matchPoolEq finds the live ElasticQuota (usage / min / max) for a pool within
// the EQs fetched for its cluster. Matches by EQ name, preferring the pool's own
// namespace when multiple namespaces expose an EQ of the same name (koord EQ
// names are unique per namespace, and the pod inherits the pool's namespace).
export function matchPoolEq(
	target: ExecutionTarget,
	eqsByCluster: Record<string, ElasticQuota[]>,
): ElasticQuota | undefined {
	const name = poolEqName(target);
	if (!name) return undefined;
	const list = eqsByCluster[clusterKey(target)] ?? [];
	const sameNamespace = list.find(
		(eq) => eq.name === name && eq.namespace === target.namespace,
	);
	return sameNamespace ?? list.find((eq) => eq.name === name);
}

// poolUsageSummary is the short "CPU 45% · Mem 30%" suffix shown on a pool
// dropdown item so deployers aren't picking blind.
export function poolUsageSummary(eq: ElasticQuota): string {
	const cpu = Math.round(eq.utilizationPercent.cpu);
	const mem = Math.round(eq.utilizationPercent.memory);
	return `CPU ${cpu}% · Mem ${mem}%`;
}

// distinctClusterKeys returns the set of cluster keys to fetch ElasticQuotas
// for, given the current pool list — always including "" (default cluster) so
// there is at least one fetch even when no target carries a clusterId.
export function distinctClusterKeys(targets: ExecutionTarget[]): string[] {
	const keys = new Set<string>([""]);
	for (const t of targets) keys.add(clusterKey(t));
	return Array.from(keys);
}

// --- Pool availability (CYB-3486) ------------------------------------------
// A bare utilization % can't answer "will my task fit?". These turn an EQ's raw
// used/min/max into a schedulability signal + absolute headroom. Koord semantics:
//   used ≤ min       → 充足: inside the guaranteed floor, always schedulable
//   min < used < max → 紧张: borrowing past the floor (needs slack elsewhere)
//   used ≥ max       → 满: at the ceiling, can't take more

// parseCpuMillis parses a k8s CPU quantity ("1", "500m", "2") to millicores.
export function parseCpuMillis(q: string): number {
	const s = (q ?? "").trim();
	if (s === "") return 0;
	if (s.endsWith("m")) return parseFloat(s.slice(0, -1)) || 0;
	return (parseFloat(s) || 0) * 1000;
}

// parseMemMi parses a k8s memory quantity ("2Gi", "64Mi", "512Ki") to Mebibytes.
// A bare number is treated as bytes.
export function parseMemMi(q: string): number {
	const m = (q ?? "").trim().match(/^(\d+(?:\.\d+)?)\s*(Ki|Mi|Gi|Ti)?$/);
	if (!m) return 0;
	const v = parseFloat(m[1]) || 0;
	switch (m[2]) {
		case "Ki":
			return v / 1024;
		case "Mi":
			return v;
		case "Gi":
			return v * 1024;
		case "Ti":
			return v * 1024 * 1024;
		default:
			return v / (1024 * 1024);
	}
}

export type PoolAvailability = "ample" | "tight" | "full";

function dimLevel(used: number, min: number, max: number): 0 | 1 | 2 {
	if (max > 0 && used >= max) return 2;
	if (used > min) return 1;
	return 0;
}

// poolAvailability collapses CPU + memory into the worse of the two dimensions.
export function poolAvailability(eq: ElasticQuota): PoolAvailability {
	const level = Math.max(
		dimLevel(
			parseCpuMillis(eq.used.cpu),
			parseCpuMillis(eq.min.cpu),
			parseCpuMillis(eq.max.cpu),
		),
		dimLevel(
			parseMemMi(eq.used.memory),
			parseMemMi(eq.min.memory),
			parseMemMi(eq.max.memory),
		),
	);
	return level === 2 ? "full" : level === 1 ? "tight" : "ample";
}

// poolAvailabilityLabel is the short zh label for a PoolAvailability.
export function poolAvailabilityLabel(a: PoolAvailability): string {
	return a === "full" ? "满" : a === "tight" ? "紧张" : "充足";
}

function fmtCpuMillis(m: number): string {
	if (m >= 1000) {
		const c = m / 1000;
		return `${Number.isInteger(c) ? c : c.toFixed(1)}c`;
	}
	return `${Math.round(m)}m`;
}

function fmtMemMi(mi: number): string {
	if (mi >= 1024) {
		const g = mi / 1024;
		return `${Number.isInteger(g) ? g : g.toFixed(1)}Gi`;
	}
	return `${Math.round(mi)}Mi`;
}

// poolFreeSummary is the "空闲 4c / 16Gi" absolute headroom to max, so a deployer
// can tell whether a task fits without mentally computing max − used.
export function poolFreeSummary(eq: ElasticQuota): string {
	const cpuFree = Math.max(
		0,
		parseCpuMillis(eq.max.cpu) - parseCpuMillis(eq.used.cpu),
	);
	const memFree = Math.max(
		0,
		parseMemMi(eq.max.memory) - parseMemMi(eq.used.memory),
	);
	return `空闲 ${fmtCpuMillis(cpuFree)} / ${fmtMemMi(memFree)}`;
}
