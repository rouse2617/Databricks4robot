export const PREVIEW_LAYOUT_VERSION_1 = 1;
export const PREVIEW_LAYOUT_VERSION_2 = 2;
export const LATEST_PREVIEW_LAYOUT_VERSION = PREVIEW_LAYOUT_VERSION_2;

type PreviewSidebarTab = "tracks" | "alerts";

interface PreviewPanelLayoutV2 {
	primaryTopic: string | null;
	sourceId: string | null;
	timeSec: number | null;
	sidebarTab: PreviewSidebarTab;
	playbackRate: string;
}

export interface PreviewLayoutV2 {
	version: 2;
	preview: string | null;
	ds: string | null;
	dsParams: Record<string, string>;
	previewPanel: PreviewPanelLayoutV2;
}

interface PreviewLayoutV1Like {
	version?: number;
	preview?: unknown;
	ds?: unknown;
	dsParams?: unknown;
	source?: unknown;
	topic?: unknown;
	time?: unknown;
	previewPanel?: {
		source?: unknown;
		topic?: unknown;
		time?: unknown;
		sidebarTab?: unknown;
		playbackRate?: unknown;
		primaryTopic?: unknown;
		sourceId?: unknown;
		timeSec?: unknown;
	} | null;
}

function asNonEmptyString(value: unknown): string | null {
	if (typeof value !== "string") return null;
	const trimmed = value.trim();
	return trimmed.length > 0 ? trimmed : null;
}

function asFiniteNonNegativeNumber(value: unknown): number | null {
	if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
		return null;
	}
	return value;
}

function asStringRecord(value: unknown): Record<string, string> {
	if (!value || typeof value !== "object") {
		return {};
	}
	const out: Record<string, string> = {};
	for (const [rawKey, rawValue] of Object.entries(
		value as Record<string, unknown>,
	)) {
		const key = rawKey.trim();
		if (!key || typeof rawValue !== "string" || !rawValue) continue;
		out[key] = rawValue;
	}
	return out;
}

function asSidebarTab(value: unknown): PreviewSidebarTab {
	return value === "alerts" ? "alerts" : "tracks";
}

function asPlaybackRate(value: unknown): string {
	const normalized = asNonEmptyString(value);
	return normalized ?? "1.0";
}

function toV2Layout(input: PreviewLayoutV1Like): PreviewLayoutV2 {
	const panel = input.previewPanel ?? null;
	return {
		version: PREVIEW_LAYOUT_VERSION_2,
		preview: asNonEmptyString(input.preview),
		ds: asNonEmptyString(input.ds),
		dsParams: asStringRecord(input.dsParams),
		previewPanel: {
			primaryTopic:
				asNonEmptyString(panel?.primaryTopic) ??
				asNonEmptyString(panel?.topic) ??
				asNonEmptyString(input.topic),
			sourceId:
				asNonEmptyString(panel?.sourceId) ??
				asNonEmptyString(panel?.source) ??
				asNonEmptyString(input.source),
			timeSec:
				asFiniteNonNegativeNumber(panel?.timeSec) ??
				asFiniteNonNegativeNumber(panel?.time) ??
				asFiniteNonNegativeNumber(input.time),
			sidebarTab: asSidebarTab(panel?.sidebarTab),
			playbackRate: asPlaybackRate(panel?.playbackRate),
		},
	};
}

function asVersion(value: unknown): number | null {
	return typeof value === "number" && Number.isInteger(value) ? value : null;
}

export function migrateLayout(layout: unknown): PreviewLayoutV2 {
	const input =
		layout && typeof layout === "object" ? (layout as PreviewLayoutV1Like) : {};
	const version = asVersion(input.version);

	if (version === PREVIEW_LAYOUT_VERSION_2) {
		return toV2Layout(input);
	}

	// Unknown/legacy version fallback: coerce to best-effort V2 shape.
	return toV2Layout(input);
}
