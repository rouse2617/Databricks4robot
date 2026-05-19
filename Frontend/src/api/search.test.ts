import { describe, expect, it } from "vitest";
import { normalizeSearchHitToAsset } from "./search";

describe("normalizeSearchHitToAsset", () => {
	it("maps Elasticsearch asset hits into frontend Asset shape", () => {
		const asset = normalizeSearchHitToAsset({
			asset_id: "asset-1",
			mcap_file_id: "mcap-1",
			asset_type: "segment",
			lifecycle_state: "ready",
			owner: "alice",
			reviewer: "bob",
			start_timestamp_ns: 100,
			end_timestamp_ns: 200,
			duration_ms: 60000,
			delivery_count: 2,
			retention_tier: "standard",
			created_at: "2026-05-06T00:00:00Z",
			updated_at: "2026-05-06T00:01:00Z",
			version: 3,
			metadata: { scene: "warehouse" },
			tags_flat: { quality: "good" },
			algos: [
				{
					name: "hand_tracking",
					version: "1.2.0",
					status: "pending",
					reason: "queued",
				},
			],
			_highlight: { owner: ["<em>alice</em>"] },
		});

		expect(asset.asset_id).toBe("asset-1");
		expect(asset.asset_type).toBe("segment");
		expect(asset.type).toBe("segment");
		expect(asset.lifecycle_state).toBe("ready");
		expect(asset.duration_sec).toBe(60);
		expect(asset.tags).toEqual({ quality: "good" });
		expect(asset.env).toBe("warehouse");
		expect(asset.algo_results["hand_tracking@1.2.0:status"]).toBe("pending");
		expect(asset.algo_results["hand_tracking@1.2.0:reason"]).toBe("queued");
		expect(asset.files).toEqual({});
		expect(asset._highlight).toEqual({ owner: ["<em>alice</em>"] });
	});
});
