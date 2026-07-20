import { describe, expect, it } from "vitest";
import {
	humanizeDlqReason,
	suppressionChip,
	validateDispatcherForm,
} from "./dispatcher";

describe("validateDispatcherForm (CYB-3679)", () => {
	const valid = { max_concurrency: 20, submit_batch: 20, rate_per_sec: 10 };

	it("accepts in-range values and boundaries", () => {
		expect(validateDispatcherForm(valid)).toEqual([]);
		expect(
			validateDispatcherForm({
				max_concurrency: 256,
				submit_batch: 200,
				rate_per_sec: 100,
			}),
		).toEqual([]);
		expect(
			validateDispatcherForm({
				max_concurrency: 1,
				submit_batch: 1,
				rate_per_sec: 0.1,
			}),
		).toEqual([]);
	});

	it("rejects out-of-range concurrency", () => {
		expect(
			validateDispatcherForm({ ...valid, max_concurrency: 0 }),
		).toHaveLength(1);
		expect(
			validateDispatcherForm({ ...valid, max_concurrency: 257 }),
		).toHaveLength(1);
	});

	it("rejects out-of-range batch and rate", () => {
		expect(validateDispatcherForm({ ...valid, submit_batch: 0 })).toHaveLength(
			1,
		);
		expect(
			validateDispatcherForm({ ...valid, rate_per_sec: 101 }),
		).toHaveLength(1);
	});

	it("rejects non-finite values and reports every problem", () => {
		expect(
			validateDispatcherForm({
				max_concurrency: Number.NaN,
				submit_batch: -1,
				rate_per_sec: Number.POSITIVE_INFINITY,
			}),
		).toHaveLength(3);
	});
});

describe("suppressionChip", () => {
	it("maps reasons to chips", () => {
		expect(suppressionChip("paused")).toEqual({
			label: "已暂停",
			color: "red",
		});
		expect(suppressionChip("aimd_backoff")).toEqual({
			label: "背压降速中",
			color: "orange",
		});
	});
	it("healthy channels get no chip", () => {
		expect(suppressionChip("")).toBeNull();
		expect(suppressionChip(undefined)).toBeNull();
	});
});

describe("humanizeDlqReason", () => {
	it("translates known failure classes", () => {
		expect(
			humanizeDlqReason(
				"invalid pipeline: pipeline has 501 steps, exceeding the maximum of 500",
			),
		).toBe("流水线步骤数超上限,请拆分批次");
		expect(humanizeDlqReason("资源规格不支持: cpu=64")).toBe(
			"资源规格超出集群上限,请调低模板资源",
		);
		expect(humanizeDlqReason("template tpl-x not found")).toBe(
			"模板不存在或已删除",
		);
		expect(humanizeDlqReason("asset abc not found")).toBe("资产不存在或已删除");
		expect(humanizeDlqReason("transpile: bad edge")).toBe(
			"流水线定义不合法(编译失败)",
		);
		expect(humanizeDlqReason("failed after max submit attempts (5)")).toBe(
			"重试次数用尽(集群持续异常)",
		);
	});
	it("falls back to the raw message / unknown", () => {
		expect(humanizeDlqReason("some obscure infra error")).toBe(
			"some obscure infra error",
		);
		expect(humanizeDlqReason(undefined)).toBe("未知错误");
		expect(humanizeDlqReason("")).toBe("未知错误");
	});
});
