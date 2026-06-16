import { type Edge, type Node, SelectType } from "@ant-design/pro-flow";
import type {
	PipelineComponentAPI,
	PipelineComponentReleaseAPI,
	PipelineComponentType,
} from "../../api/pipelineComponentApi";
import type {
	Pipeline,
	PipelineNodeData,
	Port,
	RegisteredComponent,
} from "../../components/pipeline/types";
import { formatComponentImage as formatImage } from "../../lib/pipelineComponentDisplay";
import { normalizeComponentArgs } from "../../lib/pipelineContract";

export type PipelineFlowNode = Node<PipelineNodeData>;
export type PipelineFlowEdge = Edge;

export function toRecord<T extends { id: string }>(
	items: T[],
): Record<string, T> {
	return items.reduce<Record<string, T>>((acc, item) => {
		acc[item.id] = item;
		return acc;
	}, {});
}

export function dedupeComponentsByName(
	comps: RegisteredComponent[],
): RegisteredComponent[] {
	const seen = new Set<string>();
	return comps.filter((c) => {
		const key = c.name.trim().toLowerCase();
		const versionKey = [
			c.releaseLabel,
			c.sourceCommit,
			c.tag,
			c.imageUid,
			c.source,
		]
			.filter(Boolean)
			.join(":");
		const dedupeKey = [key, versionKey || "legacy"].join(":");
		if (!key || seen.has(dedupeKey)) return false;
		seen.add(dedupeKey);
		return true;
	});
}

function normalizeComponentType(
	type: string | undefined,
): PipelineComponentType {
	const value = (type || "").trim();
	switch (value) {
		case "container":
		case "script":
		case "resource":
		case "suspend":
			return value;
		default:
			return "container";
	}
}

function normalizePorts(ports: Port[] | undefined, fallback: Port[]): Port[] {
	if (!ports || ports.length === 0) return fallback;
	const seen = new Set<string>();
	const next: Port[] = [];
	for (const port of ports) {
		const name = port.name?.trim();
		if (!name || seen.has(name)) continue;
		seen.add(name);
		next.push({
			name,
			type: port.type?.trim() || "string",
			...(port.desc?.trim() ? { desc: port.desc.trim() } : {}),
			...(port.default_value?.trim()
				? { default_value: port.default_value.trim() }
				: {}),
		});
	}
	return next.length > 0 ? next : fallback;
}

export function uniqSorted(values: string[]): string[] {
	return Array.from(new Set(values.filter(Boolean))).sort();
}

export function toStringArray(value: unknown): string[] {
	if (typeof value === "string") return [value].filter(Boolean);
	if (!Array.isArray(value)) return [];
	return value
		.map((item) => (typeof item === "string" ? item.trim() : ""))
		.filter((item) => item.length > 0);
}

export function parseAssetIds(raw: string | null | undefined): string[] {
	if (!raw) return [];
	return raw
		.split(",")
		.map((item) => item.trim())
		.filter((item) => item.length > 0);
}

export function extractNodeAssetIds(node: PipelineFlowNode | null): string[] {
	if (!node) return [];
	const data = node.data as Record<string, unknown>;
	const direct = uniqSorted([
		...toStringArray(data.asset_id),
		...toStringArray(data.assetId),
		...toStringArray(data.input_asset_id),
		...toStringArray(data._input_asset_id),
		...toStringArray(data.asset_ids),
		...toStringArray(data.assetIds),
		...toStringArray(data.input_asset_ids),
		...toStringArray(data._input_asset_ids),
	]);
	const fromAssetSelection = toStringArray(
		(data.assetSelection as Record<string, unknown> | undefined)?.assetIds,
	);
	return uniqSorted([...direct, ...fromAssetSelection]);
}

export function extractPipelineAssetIds(
	pipeline: Pipeline | Record<string, unknown>,
): string[] {
	const contract = pipeline as Record<string, unknown>;
	return uniqSorted([
		...toStringArray(contract._input_asset_ids),
		...toStringArray(contract.input_asset_ids),
		...toStringArray(
			(contract.input as Record<string, unknown> | undefined)?.asset_ids,
		),
		...toStringArray(
			(contract.assetSelection as Record<string, unknown> | undefined)
				?.assetIds,
		),
	]);
}

export function formatBytes(bytes: unknown): string {
	const n = typeof bytes === "string" ? Number(bytes) : bytes;
	if (typeof n !== "number" || !Number.isFinite(n) || n < 0) return "—";
	if (n === 0) return "0 B";
	const units = ["B", "KB", "MB", "GB", "TB", "PB"];
	let size = n;
	let i = 0;
	while (size >= 1024 && i < units.length - 1) {
		size /= 1024;
		i += 1;
	}
	return `${size.toFixed(i === 0 ? 0 : 2)} ${units[i]}`;
}

function toNumber(value: unknown): number | null {
	if (typeof value === "number") {
		return Number.isFinite(value) ? value : null;
	}
	if (typeof value === "string") {
		const parsed = Number(value);
		return Number.isFinite(parsed) ? parsed : null;
	}
	return null;
}

export function extractAssetSizeBytes(
	raw: Record<string, unknown>,
): number | null {
	const directSize =
		toNumber(raw.size) ??
		toNumber(raw.size_bytes) ??
		toNumber((raw as { bytes?: unknown }).bytes);
	if (directSize !== null) return directSize;

	const directNested = [raw.metadata, raw.files_json];
	for (const nested of directNested) {
		if (nested && typeof nested === "object") {
			const candidate = nested as Record<string, unknown>;
			const candidateSize =
				toNumber(candidate.size) ??
				toNumber(candidate.size_bytes) ??
				toNumber(candidate.byte_size) ??
				toNumber(candidate.bytes);
			if (candidateSize !== null) return candidateSize;
		}
	}

	if (raw.files_json && typeof raw.files_json === "object") {
		for (const value of Object.values(
			raw.files_json as Record<string, unknown>,
		)) {
			if (!value || typeof value !== "object") continue;
			const fileMeta = value as Record<string, unknown>;
			const fromFile =
				toNumber(fileMeta.size) ??
				toNumber(fileMeta.size_bytes) ??
				toNumber(fileMeta.byte_size) ??
				toNumber(fileMeta.bytes);
			if (fromFile !== null) return fromFile;
		}
	}

	return null;
}

export function parseAssetName(
	asset: Record<string, unknown>,
	fallbackId: string,
): string {
	if (typeof asset.name === "string" && asset.name.trim()) return asset.name;
	if (typeof asset.display_name === "string" && asset.display_name.trim()) {
		return asset.display_name;
	}
	return fallbackId || "—";
}

export function parseAssetType(
	asset: Record<string, unknown>,
	fallback: string,
): string {
	if (typeof asset.asset_type === "string" && asset.asset_type.trim()) {
		return asset.asset_type;
	}
	if (typeof asset.type === "string" && asset.type.trim()) {
		return asset.type;
	}
	return fallback;
}

export function apiToRegistered(
	api: PipelineComponentAPI,
): RegisteredComponent {
	const resources = api.resources ?? {};
	const normalizedType = normalizeComponentType(api.type);
	const normalizedSource = (api.source || "custom").trim() || "custom";
	const envFromObject = api.env
		? Object.entries(api.env).map(([name, value]) => ({
				name,
				value: value || "",
			}))
		: [];
	const envFromResource =
		Array.isArray(resources.env) && resources.env.length > 0
			? (resources.env as { name?: string; value?: string }[]).map((item) => ({
					name: item.name || "",
					value: item.value || "",
				}))
			: typeof resources.env === "object" && resources.env
				? Object.entries(resources.env as Record<string, string>).map(
						([name, value]) => ({
							name,
							value: value || "",
						}),
					)
				: [];
	return {
		id: api.id,
		name: api.name,
		type: normalizedType,
		source: normalizedSource,
		image: formatImage(api.image, api.tag),
		tag: api.tag,
		command: api.command ?? ((resources.command as string[]) || ["sh", "-c"]),
		args: normalizeComponentArgs(
			(api.args && api.args.length > 0
				? api.args
				: Array.isArray(resources.args)
					? resources.args
					: undefined) as unknown[],
		),
		env: envFromObject.length > 0 ? envFromObject : envFromResource,
		inputPorts: normalizePorts(api.inputPorts, [
			{ name: "input", type: "asset" },
		]),
		outputPorts: normalizePorts(api.outputPorts, [
			{ name: "output", type: "asset" },
		]),
		cpu: (resources.cpu as string) ?? "",
		memory: (resources.memory as string) ?? "",
		disk: (resources.disk as string) ?? "",
		gpu: (resources.gpu as string) ?? "",
		computeTier: (resources.computeTier as string) ?? "",
	};
}

export function releaseToRegistered(
	release: PipelineComponentReleaseAPI,
): RegisteredComponent {
	const snapshot = release.runtimeSnapshot ?? {
		image: release.runtimeImage,
		inputPorts: [],
		outputPorts: [],
	};
	const resources = snapshot.resources ?? {};
	const sourceRefType = (release.sourceRefType || "").trim().toLowerCase();
	return {
		id: release.id,
		name: release.displayName || release.taskName,
		type: "container",
		source: "component-release",
		image: snapshot.image || release.runtimeImage,
		tag: sourceRefType === "tag" ? release.sourceRef : release.imageTag,
		releaseLabel: release.releaseLabel,
		sourceCommit: release.sourceCommit,
		imageUid: release.imageUid,
		command:
			snapshot.command ?? ((resources.command as string[]) || ["sh", "-c"]),
		args: normalizeComponentArgs(
			(snapshot.args && snapshot.args.length > 0
				? snapshot.args
				: Array.isArray(resources.args)
					? resources.args
					: undefined) as unknown[],
		),
		env: snapshot.env
			? Object.entries(snapshot.env).map(([name, value]) => ({
					name,
					value: value || "",
				}))
			: undefined,
		inputPorts: normalizePorts(snapshot.inputPorts, [
			{ name: "input", type: "asset" },
		]),
		outputPorts: normalizePorts(snapshot.outputPorts, [
			{ name: "output", type: "asset" },
		]),
		cpu: (resources.cpu as string) ?? "",
		memory: (resources.memory as string) ?? "",
		disk: (resources.disk as string) ?? "",
		gpu: (resources.gpu as string) ?? "",
		computeTier: (resources.computeTier as string) ?? "",
	};
}

let nodeCounter = 0;

export function createPipelineNode(
	comp: RegisteredComponent,
	x: number,
	y: number,
): PipelineFlowNode {
	nodeCounter += 1;
	return {
		id: `step-${nodeCounter}`,
		type: "pipelineStep",
		position: { x, y },
		data: {
			label: comp.name,
			type: comp.type || "container",
			source: comp.source || "",
			image: comp.image,
			command: comp.command,
			args: comp.args || [],
			env: comp.env || [],
			inputPorts: comp.inputPorts || [{ name: "input", type: "asset" }],
			outputPorts: comp.outputPorts || [{ name: "output", type: "asset" }],
			cpu: comp.cpu || "",
			memory: comp.memory || "",
			disk: comp.disk || "",
			gpu: comp.gpu || "",
			computeTier: comp.computeTier || "",
			selectType: SelectType.DEFAULT,
		},
	};
}
