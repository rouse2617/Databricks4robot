// @vitest-environment jsdom
import { describe, expect, it } from "vitest";
import type { AlgoRegistryItem } from "../../api/algoRegistry";
import type { Asset } from "../../api/types";
import { collectFailedPairs } from "./useRetryAllFailed";

const makeAsset = (id: string, algoResults: Record<string, string>): Asset => ({
	asset_id: id,
	mcap_file_id: "mcap-1",
	start_timestamp_ns: 0,
	end_timestamp_ns: 0,
	duration_ms: 10000,
	duration_sec: 10,
	reviewer: "",
	status: "approved",
	owner: "",
	delivery_count: 0,
	algo_results: algoResults,
	tags: {},
	files: {},
	lifecycle_meta: {},
	created_at: "",
	updated_at: "",
	version: 1,
});

const algos: AlgoRegistryItem[] = [
	{
		key: "hand_tracking@1.2.0",
		name: "hand_tracking",
		version: "1.2.0",
		depends_on: [],
	},
	{ key: "deface@2.0.0", name: "deface", version: "2.0.0", depends_on: [] },
];

describe("collectFailedPairs", () => {
	it("returns empty array when no assets", () => {
		expect(collectFailedPairs([], algos)).toEqual([]);
	});

	it("returns empty array when no algorithms", () => {
		const assets = [makeAsset("a1", { "hand_tracking@1.2.0": "failed" })];
		expect(collectFailedPairs(assets, [])).toEqual([]);
	});

	it("returns empty array when no failed statuses", () => {
		const assets = [
			makeAsset("a1", {
				"hand_tracking@1.2.0": "ok",
				"deface@2.0.0": "running",
			}),
		];
		expect(collectFailedPairs(assets, algos)).toEqual([]);
	});

	it("collects plain string 'failed' status", () => {
		const assets = [
			makeAsset("a1", {
				"hand_tracking@1.2.0": "failed",
				"deface@2.0.0": "ok",
			}),
		];
		const pairs = collectFailedPairs(assets, algos);
		expect(pairs).toEqual([{ assetId: "a1", algoKey: "hand_tracking@1.2.0" }]);
	});

	it("collects JSON object with status=failed", () => {
		const assets = [
			makeAsset("a1", {
				"hand_tracking@1.2.0": JSON.stringify({
					status: "failed",
					reason: "timeout",
				}),
			}),
		];
		const pairs = collectFailedPairs(assets, algos);
		expect(pairs).toEqual([{ assetId: "a1", algoKey: "hand_tracking@1.2.0" }]);
	});

	it("collects 'error' status as failed", () => {
		const assets = [makeAsset("a1", { "deface@2.0.0": "error" })];
		const pairs = collectFailedPairs(assets, algos);
		expect(pairs).toEqual([{ assetId: "a1", algoKey: "deface@2.0.0" }]);
	});

	it("collects multiple failed pairs across assets and algos", () => {
		const assets = [
			makeAsset("a1", {
				"hand_tracking@1.2.0": "failed",
				"deface@2.0.0": "failed",
			}),
			makeAsset("a2", {
				"hand_tracking@1.2.0": "ok",
				"deface@2.0.0": "failed",
			}),
		];
		const pairs = collectFailedPairs(assets, algos);
		expect(pairs).toHaveLength(3);
		expect(pairs).toContainEqual({
			assetId: "a1",
			algoKey: "hand_tracking@1.2.0",
		});
		expect(pairs).toContainEqual({ assetId: "a1", algoKey: "deface@2.0.0" });
		expect(pairs).toContainEqual({ assetId: "a2", algoKey: "deface@2.0.0" });
	});

	it("ignores assets with missing algo_results", () => {
		const assets = [makeAsset("a1", {})];
		expect(collectFailedPairs(assets, algos)).toEqual([]);
	});
});
