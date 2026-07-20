// @vitest-environment jsdom

import { cleanup, render } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import RangeSlider from "./RangeSlider";

afterEach(cleanup);

// 回归锁:aria-valuenow 曾引用不存在的 `currentSec`(应为 prop `currentTime`),
// 组件一渲染就 ReferenceError,预览时间轴整个白屏。vite build 不做类型检查,
// 这类错误只有渲染测试或 tsc 能拦住。
describe("RangeSlider", () => {
	it("renders without throwing and reflects currentTime in aria-valuenow", () => {
		const { container } = render(
			<RangeSlider
				currentTime={42}
				durationSec={100}
				range={{ startSec: 10, endSec: 60 }}
				onSeek={vi.fn()}
				onRangeChange={vi.fn()}
			/>,
		);
		const sliders = Array.from(container.querySelectorAll('[role="slider"]'));
		// 轨道 + 播放头 + 两个 range 手柄(手柄的 valuenow 是各自的 start/end 秒)。
		expect(sliders.length).toBeGreaterThanOrEqual(2);
		const atPlayhead = sliders.filter(
			(el) => el.getAttribute("aria-valuenow") === "42",
		);
		expect(atPlayhead.length).toBeGreaterThanOrEqual(2); // 轨道 + 播放头都反映 currentTime
		for (const el of atPlayhead) {
			expect(el.getAttribute("aria-valuemax")).toBe("100");
		}
	});

	it("renders without a range (playhead only)", () => {
		const { container } = render(
			<RangeSlider
				currentTime={0}
				durationSec={0}
				range={null}
				onSeek={vi.fn()}
			/>,
		);
		expect(container.querySelector('[role="slider"]')).toBeTruthy();
	});
});
