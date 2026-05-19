import { describe, expect, it } from "vitest";
import { parsePreviewURLState, updatePreviewURLState } from "./previewURLState";

describe("previewURLState", () => {
	it("parses preview/topic/time from URLSearchParams", () => {
		const sp = new URLSearchParams(
			"preview=asset-123&preview_source=live_topic_0&preview_topic=%2Fcamera%2Ffront&preview_time=12.5",
		);
		const state = parsePreviewURLState(sp);
		expect(state.preview).toBe("asset-123");
		expect(state.source).toBe("live_topic_0");
		expect(state.topic).toBe("/camera/front");
		expect(state.time).toBe(12.5);
		expect(state.layoutVersion).toBe(2);
	});

	it("drops invalid preview_time values", () => {
		const sp = new URLSearchParams("preview=asset-123&preview_time=bad");
		const state = parsePreviewURLState(sp);
		expect(state.preview).toBe("asset-123");
		expect(state.time).toBeUndefined();
	});

	it("falls back to time when preview_time is invalid", () => {
		const sp = new URLSearchParams("preview_time=bad&time=15.25");
		const state = parsePreviewURLState(sp);
		expect(state.time).toBe(15.25);
	});

	it("parses RFC3339 time for compatibility", () => {
		const sp = new URLSearchParams("time=2025-01-02T03:04:05Z");
		const state = parsePreviewURLState(sp);
		expect(state.time).toBe(1735787045);
	});

	it("keeps first duplicate ds key and merges ds.url", () => {
		const sp = new URLSearchParams(
			"ds=remote-file&ds.asset_id=one&ds.asset_id=two&ds.url=http%3A%2F%2Fa&ds.url=http%3A%2F%2Fb",
		);
		const state = parsePreviewURLState(sp);
		expect(state.ds).toBe("remote-file");
		expect(state.dsParams).toEqual({
			asset_id: "one",
			url: "http://a,http://b",
		});
	});

	it("updates and removes preview params", () => {
		const initial = new URLSearchParams("mode=structured");
		const updated = updatePreviewURLState(initial, {
			preview: "asset-123",
			source: "live_topic_0",
			topic: "/camera/front",
			time: 8,
		});
		expect(updated.get("mode")).toBe("structured");
		expect(updated.get("preview")).toBe("asset-123");
		expect(updated.get("preview_source")).toBe("live_topic_0");
		expect(updated.get("preview_topic")).toBe("/camera/front");
		expect(updated.get("preview_time")).toBe("8");
		expect(updated.get("time")).toBe("8");
		expect(updated.get("preview_layout_version")).toBe("2");

		const cleaned = updatePreviewURLState(updated, {
			preview: undefined,
			source: undefined,
			topic: undefined,
			time: undefined,
		});
		expect(cleaned.get("preview")).toBeNull();
		expect(cleaned.get("preview_source")).toBeNull();
		expect(cleaned.get("preview_topic")).toBeNull();
		expect(cleaned.get("preview_time")).toBeNull();
		expect(cleaned.get("time")).toBeNull();
		expect(cleaned.get("mode")).toBe("structured");
	});

	it("writes ds params in stable sorted order", () => {
		const updated = updatePreviewURLState(new URLSearchParams(), {
			ds: "remote-file",
			dsParams: {
				zeta: "last",
				alpha: "first",
			},
		});
		expect(updated.toString()).toBe(
			"ds=remote-file&ds.alpha=first&ds.zeta=last&preview_layout_version=2",
		);
	});

	it("falls back safely for unknown preview layout version", () => {
		const sp = new URLSearchParams(
			"preview_layout_version=999&preview_source=legacy-source&preview_topic=%2Flegacy%2Ftopic",
		);
		const state = parsePreviewURLState(sp);
		expect(state.source).toBe("legacy-source");
		expect(state.topic).toBe("/legacy/topic");
		expect(state.layoutVersion).toBe(2);
	});
});
