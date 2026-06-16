import { describe, expect, it } from "vitest";
import { formatRerunFeedback } from "./BatchJobDetailPage";

describe("formatRerunFeedback", () => {
	it("maps failed rerun to error feedback", () => {
		expect(
			formatRerunFeedback({
				status: "failed",
				matchedCount: 2,
			}),
		).toEqual({
			level: "error",
			text: "重跑失败，没有可重新提交的子任务",
		});
	});

	it("maps partial rerun to warning feedback", () => {
		expect(
			formatRerunFeedback({
				status: "partial_success",
				matchedCount: 2,
				retriedCount: 1,
				skipped: [{ itemId: "item-2", reason: "duplicate" }],
			}),
		).toEqual({
			level: "warning",
			text: "重跑部分提交成功：1 / 2",
		});
	});

	it("maps successful rerun to success feedback", () => {
		expect(
			formatRerunFeedback({
				status: "succeeded",
				matchedCount: 1,
				retriedCount: 1,
				skipped: [],
			}),
		).toEqual({
			level: "success",
			text: "重跑已提交",
		});
	});
});
