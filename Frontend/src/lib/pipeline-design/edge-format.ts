export const DEFAULT_INPUT_PORT = "input";
export const DEFAULT_OUTPUT_PORT = "output";

/** Format canvas edge endpoints for the transpiler (node-id.port-name). */
export function formatEdgeEndpoint(
	nodeId: string,
	handle: string | null | undefined,
	defaultPort: string,
): string {
	if (nodeId.includes(".")) {
		return nodeId;
	}
	const port = (handle && handle.length > 0 ? handle : defaultPort).replace(
		/^\./,
		"",
	);
	return `${nodeId}.${port}`;
}

export function splitRef(ref: string): { nodeId: string; port?: string } {
	const dot = ref.lastIndexOf(".");
	if (dot < 0) {
		return { nodeId: ref };
	}
	return { nodeId: ref.slice(0, dot), port: ref.slice(dot + 1) };
}
