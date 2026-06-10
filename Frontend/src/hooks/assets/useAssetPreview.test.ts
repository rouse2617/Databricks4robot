// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import type { Asset } from "../../api/types";
import {
	buildPlaceholderPreviewManifest,
	buildPreviewManifestFromSources,
} from "./useAssetPreview";

function makeAsset(overrides: Partial<Asset>): Asset {
	return {
		asset_id: "abc123",
		mcap_file_id: "mcap-1",
		start_timestamp_ns: 0,
		end_timestamp_ns: 0,
		reviewer: "",
		owner: "",
		delivery_count: 0,
		algo_results: {},
		tags: {},
		files: {},
		lifecycle_meta: {},
		created_at: "",
		updated_at: "",
		version: 1,
		...overrides,
	};
}

describe("buildPlaceholderPreviewManifest", () => {
	it("returns thumbnail mode when thumbnail exists", () => {
		const m = buildPlaceholderPreviewManifest(
			makeAsset({
				files: { thumbnail: "x.jpg" },
			}),
		);
		expect(m.mode).toBe("thumbnail");
		expect(m.availability).toBe("ready");
		expect(m.previewVideoUrl).toBeNull();
	});

	it("returns missing when no thumbnail exists", () => {
		const m = buildPlaceholderPreviewManifest(
			makeAsset({
				lifecycle_state: "incoming",
				files: {},
			}),
		);
		expect(m.mode).toBe("none");
		expect(m.availability).toBe("missing");
		expect(m.previewVideoUrl).toBeNull();
	});
});

describe("buildPreviewManifestFromSources", () => {
	it("prefers foxglove source and returns mcap mode", () => {
		const m = buildPreviewManifestFromSources(
			makeAsset({ lifecycle_state: "ready" }),
			null,
			{
				asset_id: "abc123",
				source_id: "remote-file",
				ds: "remote-file",
				ds_params: { url: "https://example.com/a.mcap" },
			},
			undefined,
		);
		expect(m.mode).toBe("mcap");
		expect(m.previewVideoUrl).toContain(
			"/api/v1/preview/assets/abc123/segment.mp4",
		);
		expect(m.mcapUrl).toBe("https://example.com/a.mcap");
		expect(m.ds).toBe("remote-file");
		expect(m.dsParams?.asset_id).toBe("abc123");
	});

	it("includes grace token and topic in segment preview url", () => {
		Object.defineProperty(document, "cookie", {
			configurable: true,
			get: () => "databrew_session=dev-token",
		});
		const m = buildPreviewManifestFromSources(
			makeAsset({ lifecycle_state: "ready" }),
			null,
			{
				asset_id: "abc123",
				source_id: "remote-file",
				ds: "remote-file",
				ds_params: { url: "https://example.com/a.mcap" },
			},
			{ previewTopic: "/camera/front/image_raw/compressed" },
		);
		expect(m.previewVideoUrl).toContain("databrew_token=dev-token");
		expect(m.previewVideoUrl).toContain(
			"topic=%2Fcamera%2Ffront%2Fimage_raw%2Fcompressed",
		);
	});

	it("falls back to placeholder when foxglove source is missing", () => {
		const m = buildPreviewManifestFromSources(
			makeAsset({ files: { thumbnail: "x.jpg" } }),
			null,
			null,
			undefined,
		);
		expect(m.mode).toBe("thumbnail");
		expect(m.mcapUrl).toBeUndefined();
	});
});
