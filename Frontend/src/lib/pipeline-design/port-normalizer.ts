import type { Port } from "../../components/pipeline/types";

export const DEFAULT_INPUT_PORT = "input";
export const DEFAULT_OUTPUT_PORT = "output";

export const defaultInputPorts = (): Port[] => [
	{ name: DEFAULT_INPUT_PORT, type: "string" },
];
export const defaultOutputPorts = (): Port[] => [
	{ name: DEFAULT_OUTPUT_PORT, type: "string" },
];

export function normalizePorts(
	ports: Port[] | undefined,
	fallback: () => Port[],
): Port[] {
	if (!ports || ports.length === 0) return fallback();
	const seen = new Set<string>();
	const normalized: Port[] = [];
	for (const port of ports) {
		const name = port.name?.trim();
		if (!name || seen.has(name)) continue;
		seen.add(name);
		normalized.push({
			name,
			type: port.type?.trim() || "string",
			...(port.desc?.trim() ? { desc: port.desc.trim() } : {}),
			...(port.default_value?.trim()
				? { default_value: port.default_value.trim() }
				: {}),
		});
	}
	return normalized.length > 0 ? normalized : fallback();
}
