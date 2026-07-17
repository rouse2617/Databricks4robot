// Build a direct link to a pod's page in the GCP / GKE console.
//
// Each pipeline step runs as one pod; this lets the run detail deep-link to
// `console.cloud.google.com/kubernetes/pod/{region}/{cluster}/{ns}/{pod}/details`.
//
// CYB-3570 — interim config: the GKE console coordinates (GCP project, region,
// GKE cluster name) are infra facts that are NOT stored in the clusters table
// (its rows only carry the API-server IP). The proper fix is
// clusters.gcp_project / gcp_region columns, which needs a DB migration —
// deferred to a human. Until then we map by namespace, which every run/node
// reliably carries. Unknown namespaces yield no link (graceful).
//
// Adding a cluster/namespace = add a rule here.

export interface GkeConsoleCoords {
	/** GKE cluster name as it appears in the console URL. */
	cluster: string;
	/** GCP project the cluster lives in. */
	project: string;
	/** Cluster region/location. */
	region: string;
}

// First matching prefix wins. Kept explicit (no catch-all default) so an
// unrecognized namespace never produces a wrong-project link.
const NAMESPACE_PREFIX_COORDS: ReadonlyArray<
	{ prefix: string } & GkeConsoleCoords
> = [
	{
		prefix: "cyber-delivery",
		cluster: "delivery-clust",
		project: "cyberorigin-delivery",
		region: "us-central1",
	},
	{
		prefix: "cyber-databrew",
		cluster: "cyber-clust",
		project: "green-valley-442103",
		region: "us-central1",
	},
	{
		prefix: "video-proc",
		cluster: "cyber-clust",
		project: "green-valley-442103",
		region: "us-central1",
	},
];

export function gkeConsoleCoordsForNamespace(
	namespace?: string | null,
): GkeConsoleCoords | null {
	const ns = (namespace ?? "").trim();
	if (!ns) return null;
	const match = NAMESPACE_PREFIX_COORDS.find((c) => ns.startsWith(c.prefix));
	return match
		? { cluster: match.cluster, project: match.project, region: match.region }
		: null;
}

/**
 * Returns the GKE console pod URL, or null when the namespace/pod is missing or
 * the namespace maps to no known cluster (caller should render no link).
 */
export function gkePodConsoleUrl(
	namespace?: string | null,
	podName?: string | null,
): string | null {
	const ns = (namespace ?? "").trim();
	const pod = (podName ?? "").trim();
	if (!ns || !pod) return null;
	const coords = gkeConsoleCoordsForNamespace(ns);
	if (!coords) return null;
	return (
		`https://console.cloud.google.com/kubernetes/pod/${coords.region}/` +
		`${coords.cluster}/${encodeURIComponent(ns)}/${encodeURIComponent(pod)}/details` +
		`?project=${coords.project}`
	);
}
