// @vitest-environment jsdom

import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useVisibleInterval } from "./useVisibleInterval";

function setHidden(hidden: boolean) {
	Object.defineProperty(document, "visibilityState", {
		configurable: true,
		get: () => (hidden ? "hidden" : "visible"),
	});
	Object.defineProperty(document, "hidden", {
		configurable: true,
		get: () => hidden,
	});
	document.dispatchEvent(new Event("visibilitychange"));
}

describe("useVisibleInterval", () => {
	beforeEach(() => {
		vi.useFakeTimers();
		setHidden(false);
	});
	afterEach(() => {
		vi.useRealTimers();
		setHidden(false);
	});

	it("fires on each interval while the tab is visible", () => {
		const cb = vi.fn();
		renderHook(() => useVisibleInterval(cb, 1000, true));
		expect(cb).not.toHaveBeenCalled(); // no immediate fire on mount
		vi.advanceTimersByTime(1000);
		expect(cb).toHaveBeenCalledTimes(1);
		vi.advanceTimersByTime(2000);
		expect(cb).toHaveBeenCalledTimes(3);
	});

	it("never fires when disabled", () => {
		const cb = vi.fn();
		renderHook(() => useVisibleInterval(cb, 1000, false));
		vi.advanceTimersByTime(5000);
		expect(cb).not.toHaveBeenCalled();
	});

	it("pauses while hidden and refreshes immediately on resume", () => {
		const cb = vi.fn();
		renderHook(() => useVisibleInterval(cb, 1000, true));
		vi.advanceTimersByTime(1000);
		expect(cb).toHaveBeenCalledTimes(1);

		setHidden(true);
		vi.advanceTimersByTime(5000);
		expect(cb).toHaveBeenCalledTimes(1); // paused: no new fires

		setHidden(false); // resume → one immediate refresh…
		expect(cb).toHaveBeenCalledTimes(2);
		vi.advanceTimersByTime(1000); // …then the interval continues
		expect(cb).toHaveBeenCalledTimes(3);
	});

	it("stops firing after unmount", () => {
		const cb = vi.fn();
		const { unmount } = renderHook(() => useVisibleInterval(cb, 1000, true));
		unmount();
		vi.advanceTimersByTime(5000);
		expect(cb).not.toHaveBeenCalled();
	});
});
