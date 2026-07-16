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
