import { describe, expect, it } from "vitest";
import type { ElasticQuota, ExecutionTarget } from "../../api/pipelineApi";
import {
	clusterKey,
	distinctClusterKeys,
	KOORD_EQ_LABEL_KEY,
	matchPoolEq,
	parseCpuMillis,
	parseMemMi,
	poolAvailability,
	poolAvailabilityLabel,
	poolEqName,
	poolFreeSummary,
	poolUsageSummary,
} from "./poolUsage";

function target(overrides: Partial<ExecutionTarget> = {}): ExecutionTarget {
	return {
		id: "t1",
		name: "pool-1",
		cluster: "default",
		namespace: "video-proc-dev",
		argoServerConfigured: true,
		status: "available",
		isDefault: false,
		...overrides,
	};
}

function eq(overrides: Partial<ElasticQuota> = {}): ElasticQuota {
	return {
		name: "cyberorigin-delivery-low",
		namespace: "video-proc-dev",
		min: { cpu: "1", memory: "2Gi" },
		max: { cpu: "10", memory: "20Gi" },
		used: { cpu: "5", memory: "8Gi" },
		utilizationPercent: { cpu: 45.4, memory: 30.6 },
		...overrides,
	};
}

describe("poolUsage", () => {
	describe("clusterKey", () => {
		it("uses clusterId, falling back to '' for the default cluster", () => {
			expect(clusterKey({ clusterId: "c-1" })).toBe("c-1");
			expect(clusterKey({ clusterId: undefined })).toBe("");
		});
	});

	describe("poolEqName", () => {
		it("returns the koord EQ label value when present", () => {
			const t = target({
				resourceDefaults: {
					scheduling: {
						podLabels: { [KOORD_EQ_LABEL_KEY]: "cyberorigin-delivery-low" },
					},
				},
			});
			expect(poolEqName(t)).toBe("cyberorigin-delivery-low");
		});

		it("returns undefined for a namespace-coarse pool with no EQ label", () => {
			expect(poolEqName(target())).toBeUndefined();
			expect(
				poolEqName(target({ resourceDefaults: { scheduling: {} } })),
			).toBeUndefined();
		});

		it("treats a blank label value as no EQ", () => {
			const t = target({
				resourceDefaults: {
					scheduling: { podLabels: { [KOORD_EQ_LABEL_KEY]: "   " } },
				},
			});
			expect(poolEqName(t)).toBeUndefined();
		});
	});

	describe("matchPoolEq", () => {
		const wiredTarget = target({
			clusterId: "c-default",
			namespace: "video-proc-dev",
			resourceDefaults: {
				scheduling: {
					podLabels: { [KOORD_EQ_LABEL_KEY]: "cyberorigin-delivery-low" },
				},
			},
		});

		it("matches an EQ by name within the pool's cluster", () => {
			const found = matchPoolEq(wiredTarget, {
				"c-default": [eq(), eq({ name: "other" })],
			});
			expect(found?.name).toBe("cyberorigin-delivery-low");
			expect(found?.utilizationPercent.cpu).toBe(45.4);
		});

		it("prefers the EQ in the pool's own namespace on a name collision", () => {
			const found = matchPoolEq(wiredTarget, {
				"c-default": [
					eq({ namespace: "other-ns", used: { cpu: "1", memory: "1Gi" } }),
					eq({
						namespace: "video-proc-dev",
						used: { cpu: "9", memory: "9Gi" },
					}),
				],
			});
			expect(found?.namespace).toBe("video-proc-dev");
			expect(found?.used.cpu).toBe("9");
		});

		it("returns undefined for a non-koord pool", () => {
			expect(matchPoolEq(target(), { "": [eq()] })).toBeUndefined();
		});

		it("returns undefined when the cluster has no matching EQ", () => {
			expect(
				matchPoolEq(wiredTarget, { "c-default": [eq({ name: "nope" })] }),
			).toBeUndefined();
			// wrong cluster bucket → no match
			expect(matchPoolEq(wiredTarget, { other: [eq()] })).toBeUndefined();
		});
	});

	describe("poolUsageSummary", () => {
		it("renders rounded CPU / Mem percentages", () => {
			expect(poolUsageSummary(eq())).toBe("CPU 45% · Mem 31%");
		});
	});

	describe("distinctClusterKeys", () => {
		it("always includes the default '' key and dedupes cluster ids", () => {
			const keys = distinctClusterKeys([
				target({ clusterId: "c-1" }),
				target({ clusterId: "c-1" }),
				target({ clusterId: undefined }),
			]);
			expect(new Set(keys)).toEqual(new Set(["", "c-1"]));
		});

		it("returns just the default key for an empty pool list", () => {
			expect(distinctClusterKeys([])).toEqual([""]);
		});
	});
});

describe("poolUsage — availability & headroom (CYB-3486)", () => {
	const eq = (over: Partial<ElasticQuota> = {}): ElasticQuota => ({
		name: "q",
		namespace: "ns",
		min: { cpu: "2", memory: "4Gi" },
		max: { cpu: "10", memory: "20Gi" },
		used: { cpu: "0", memory: "0" },
		utilizationPercent: { cpu: 0, memory: 0 },
		...over,
	});

	it("parses cpu quantities to millicores", () => {
		expect(parseCpuMillis("1")).toBe(1000);
		expect(parseCpuMillis("500m")).toBe(500);
		expect(parseCpuMillis("2")).toBe(2000);
		expect(parseCpuMillis("")).toBe(0);
	});

	it("parses memory quantities to Mi", () => {
		expect(parseMemMi("64Mi")).toBe(64);
		expect(parseMemMi("2Gi")).toBe(2048);
		expect(parseMemMi("1024Ki")).toBe(1);
		expect(parseMemMi("")).toBe(0);
	});

	it("reports 充足 inside the guaranteed floor (used ≤ min)", () => {
		const a = poolAvailability(eq({ used: { cpu: "1", memory: "2Gi" } }));
		expect(a).toBe("ample");
		expect(poolAvailabilityLabel(a)).toBe("充足");
	});

	it("reports 紧张 when borrowing past min", () => {
		const a = poolAvailability(eq({ used: { cpu: "5", memory: "2Gi" } }));
		expect(a).toBe("tight");
		expect(poolAvailabilityLabel(a)).toBe("紧张");
	});

	it("reports 满 at the ceiling (used ≥ max on either dim)", () => {
		const a = poolAvailability(eq({ used: { cpu: "10", memory: "2Gi" } }));
		expect(a).toBe("full");
		expect(poolAvailabilityLabel(a)).toBe("满");
	});

	it("takes the worse of cpu / memory", () => {
		expect(poolAvailability(eq({ used: { cpu: "1", memory: "20Gi" } }))).toBe(
			"full",
		);
	});

	it("summarizes free headroom to max", () => {
		expect(poolFreeSummary(eq({ used: { cpu: "4", memory: "4Gi" } }))).toBe(
			"空闲 6c / 16Gi",
		);
		expect(poolFreeSummary(eq({ used: { cpu: "10", memory: "20Gi" } }))).toBe(
			"空闲 0m / 0Mi",
		);
	});
});
