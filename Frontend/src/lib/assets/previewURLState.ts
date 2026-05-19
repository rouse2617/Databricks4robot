// ─── Preview URL State Helpers ───
// Small utility inspired by Foxglove ds/ds.* URL state design.
// Keeps preview-specific params parse/serialize logic centralized.

import {
	LATEST_PREVIEW_LAYOUT_VERSION,
	migrateLayout,
} from "./previewLayoutMigration";

export interface PreviewURLState {
	preview?: string;
	source?: string;
	topic?: string;
	time?: number;
	ds?: string;
	dsParams?: Record<string, string>;
	layoutVersion?: number;
}

function parsePreviewTimeSeconds(rawValue: string | null): number | undefined {
	const raw = rawValue?.trim() ?? "";
	if (!raw) return undefined;

	const isNumericText = /^[+]?(?:\d+\.?\d*|\.\d+)$/.test(raw);
	if (isNumericText) {
		const numeric = Number.parseFloat(raw);
		if (Number.isFinite(numeric) && numeric >= 0) {
			return numeric;
		}
	}

	const dateMillis = Date.parse(raw);
	if (Number.isFinite(dateMillis) && dateMillis >= 0) {
		// Keep preview time in seconds for hook state.
		return dateMillis / 1000;
	}

	return undefined;
}

function readPreviewTime(sp: URLSearchParams): number | undefined {
	const previewTime = parsePreviewTimeSeconds(sp.get("preview_time"));
	if (previewTime !== undefined) {
		return previewTime;
	}
	return parsePreviewTimeSeconds(sp.get("time"));
}

export function parsePreviewURLState(sp: URLSearchParams): PreviewURLState {
	const preview = sp.get("preview")?.trim() ?? "";
	const source = sp.get("preview_source")?.trim() ?? "";
	const topic = sp.get("preview_topic")?.trim() ?? "";
	const ds = sp.get("ds")?.trim() ?? "";
	const versionRaw = sp.get("preview_layout_version")?.trim() ?? "";
	const parsedVersion = versionRaw
		? Number.parseInt(versionRaw, 10)
		: Number.NaN;
	const dsParams: Record<string, string> = {};
	for (const [key, value] of sp.entries()) {
		if (!key.startsWith("ds.")) continue;
		const clean = key.slice(3).trim();
		const trimmedValue = value.trim();
		if (!clean || !trimmedValue) continue;

		if (!(clean in dsParams)) {
			dsParams[clean] = trimmedValue;
			continue;
		}

		if (clean === "url") {
			dsParams[clean] = `${dsParams[clean]},${trimmedValue}`;
		}
	}

	const migrated = migrateLayout({
		version: Number.isInteger(parsedVersion) ? parsedVersion : undefined,
		preview,
		source,
		topic,
		time: readPreviewTime(sp),
		ds,
		dsParams,
	});

	return {
		preview: migrated.preview ?? undefined,
		source: migrated.previewPanel.sourceId ?? undefined,
		topic: migrated.previewPanel.primaryTopic ?? undefined,
		time: migrated.previewPanel.timeSec ?? undefined,
		ds: migrated.ds ?? undefined,
		dsParams:
			Object.keys(migrated.dsParams).length > 0 ? migrated.dsParams : undefined,
		layoutVersion: migrated.version,
	};
}

export function updatePreviewURLState(
	sp: URLSearchParams,
	state: PreviewURLState,
): URLSearchParams {
	const next = new URLSearchParams(sp.toString());
	const migrated = migrateLayout({
		version: state.layoutVersion,
		preview: state.preview,
		source: state.source,
		topic: state.topic,
		time: state.time,
		ds: state.ds,
		dsParams: state.dsParams,
	});

	if ("preview" in state) {
		if (migrated.preview) {
			next.set("preview", migrated.preview);
		} else {
			next.delete("preview");
		}
	}

	if ("topic" in state) {
		if (migrated.previewPanel.primaryTopic) {
			next.set("preview_topic", migrated.previewPanel.primaryTopic);
		} else {
			next.delete("preview_topic");
		}
	}

	if ("source" in state) {
		if (migrated.previewPanel.sourceId) {
			next.set("preview_source", migrated.previewPanel.sourceId);
		} else {
			next.delete("preview_source");
		}
	}

	if ("time" in state) {
		if (migrated.previewPanel.timeSec != null) {
			next.set("preview_time", String(migrated.previewPanel.timeSec));
			next.set("time", String(migrated.previewPanel.timeSec));
		} else {
			next.delete("preview_time");
			next.delete("time");
		}
	}

	if ("ds" in state) {
		if (migrated.ds) {
			next.set("ds", migrated.ds);
		} else {
			next.delete("ds");
		}
	}

	if ("dsParams" in state) {
		Array.from(next.keys()).forEach((key) => {
			if (key.startsWith("ds.")) {
				next.delete(key);
			}
		});
		const entries = Object.entries(migrated.dsParams).sort(([left], [right]) =>
			left.localeCompare(right),
		);
		for (const [k, v] of entries) {
			const key = k.trim();
			const value = v.trim();
			if (!key || !value) continue;
			next.append(`ds.${key}`, value);
		}
	}

	next.set("preview_layout_version", String(LATEST_PREVIEW_LAYOUT_VERSION));
	next.sort();

	return next;
}
