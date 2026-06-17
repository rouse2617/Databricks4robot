import { describe, expect, it } from "vitest";
import {
	COLUMN_LABELS,
	formatBusinessStatusLabel,
	resolveBusinessStatusTagColor,
	SEARCH_MODE_LABELS,
} from "./productVocabulary";

describe("productVocabulary", () => {
	it("maps common business statuses to Chinese", () => {
		expect(formatBusinessStatusLabel("pending")).toBe("待处理");
		expect(formatBusinessStatusLabel("running")).toBe("运行中");
		expect(formatBusinessStatusLabel("ok")).toBe("成功");
		expect(formatBusinessStatusLabel("cancelled")).toBe("已取消");
		expect(formatBusinessStatusLabel("delivered")).toBe("已交付");
	});

	it("uses semantic tag colors", () => {
		expect(resolveBusinessStatusTagColor("ok")).toBe("success");
		expect(resolveBusinessStatusTagColor("running")).toBe("processing");
		expect(resolveBusinessStatusTagColor("cancelled")).toBe("default");
		expect(resolveBusinessStatusTagColor("blocked")).toBe("warning");
		expect(resolveBusinessStatusTagColor("failed")).toBe("error");
	});

	it("exposes column and search mode labels", () => {
		expect(COLUMN_LABELS.assetId).toBe("资产 ID");
		expect(SEARCH_MODE_LABELS.structured).toBe("结构化");
	});
});
