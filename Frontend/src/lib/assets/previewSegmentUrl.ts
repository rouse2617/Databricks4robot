import type { FoxgloveSourceResponse } from "../../api/assets";
import type { PreviewManifest } from "./assetsDiscoveryTypes";

export function readGraceSessionCookie(): string | null {
	if (typeof document === "undefined" || !document.cookie) {
		return null;
	}
	for (const part of document.cookie.split(";")) {
		const [k, ...rest] = part.trim().split("=");
		if (k !== "grace_session") continue;
		const raw = rest.join("=");
		if (!raw) return null;
		try {
			return decodeURIComponent(raw);
		} catch {
			return raw;
		}
	}
	return null;
}

export function parseWindowNsFromHints(foxgloveSource?: FoxgloveSourceResponse | null): {
	startNs: number | null;
	endNs: number | null;
} {
	const hints = foxgloveSource?.hints;
	if (!hints || typeof hints !== "object") {
		return { startNs: null, endNs: null };
	}
	const windowValue = (hints as Record<string, unknown>).window;
	if (!windowValue || typeof windowValue !== "object") {
		return { startNs: null, endNs: null };
	}
	const win = windowValue as Record<string, unknown>;
	const parseNs = (v: unknown): number | null => {
		if (typeof v === "number" && Number.isFinite(v)) return v;
		if (typeof v === "string" && v.trim().length > 0) {
			const n = Number(v);
			if (Number.isFinite(n)) return n;
		}
		return null;
	};
	return {
		startNs: parseNs(win.start_timestamp_ns),
		endNs: parseNs(win.end_timestamp_ns),
	};
}

/** Append grace_token and asset window query params for segment.mp4 playback. */
export function enrichSegmentPreviewUrl(
	pathOrUrl: string,
	opts?: {
		topic?: string;
		startNs?: number | null;
		endNs?: number | null;
	},
): string {
	const qIndex = pathOrUrl.indexOf("?");
	const path = qIndex >= 0 ? pathOrUrl.slice(0, qIndex) : pathOrUrl;
	const params = new URLSearchParams(qIndex >= 0 ? pathOrUrl.slice(qIndex + 1) : "");
	if (opts?.topic && !params.has("topic")) {
		params.set("topic", opts.topic);
	}
	if (opts?.startNs != null && !params.has("start_ns")) {
		params.set("start_ns", String(Math.trunc(opts.startNs)));
	}
	if (opts?.endNs != null && !params.has("end_ns")) {
		params.set("end_ns", String(Math.trunc(opts.endNs)));
	}
	const graceToken = readGraceSessionCookie();
	if (graceToken && !params.has("grace_token")) {
		params.set("grace_token", graceToken);
	}
	const qs = params.toString();
	return qs ? `${path}?${qs}` : path;
}

export function buildSegmentPreviewUrl(
	assetId: string,
	opts?: {
		topic?: string;
		startNs?: number | null;
		endNs?: number | null;
	},
): string {
	const path = `/api/v1/preview/assets/${encodeURIComponent(assetId)}/segment.mp4`;
	return enrichSegmentPreviewUrl(path, opts);
}

export function formatPreviewCodecLabel(codec?: string): string {
	if (!codec) return "";
	const c = codec.toLowerCase();
	if (c === "h265" || c === "hevc") return "HEVC";
	if (c === "h264") return "H.264";
	return codec.toUpperCase();
}

export function isHevcPreviewCodec(codec?: string): boolean {
	if (!codec) return false;
	const c = codec.toLowerCase();
	return c === "h265" || c === "hevc" || c === "hev1" || c === "hvc1";
}

export function activePreviewSource(manifest: PreviewManifest) {
	const id =
		manifest.activeSourceId ??
		manifest.recommendedSourceId ??
		manifest.sources?.[0]?.id;
	return manifest.sources?.find((s) => s.id === id) ?? null;
}
