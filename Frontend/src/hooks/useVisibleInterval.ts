import { useEffect, useRef } from "react";

/**
 * useVisibleInterval runs `callback` every `delayMs`, but only while the tab is
 * visible (document.visibilityState) and `enabled` is true. Polling pauses when
 * the tab is hidden and resumes when it is shown again, so a backgrounded tab
 * stops hammering the API (CYB-3486). On resume it fires once immediately so a
 * returning user isn't left looking at stale data for up to `delayMs`.
 *
 * The latest `callback` is always used without resetting the timer, so callers
 * can pass an inline closure without churning the interval every render.
 */
export function useVisibleInterval(
	callback: () => void,
	delayMs: number,
	enabled = true,
): void {
	const savedCallback = useRef(callback);
	savedCallback.current = callback;

	useEffect(() => {
		if (!enabled || delayMs <= 0) {
			return;
		}

		let timer: ReturnType<typeof setInterval> | null = null;
		const start = () => {
			timer ??= setInterval(() => savedCallback.current(), delayMs);
		};
		const stop = () => {
			if (timer !== null) {
				clearInterval(timer);
				timer = null;
			}
		};
		const onVisibilityChange = () => {
			if (document.hidden) {
				stop();
			} else if (timer === null) {
				// Resumed after being hidden: refresh now, then keep polling.
				savedCallback.current();
				start();
			}
		};

		if (!document.hidden) {
			start();
		}
		document.addEventListener("visibilitychange", onVisibilityChange);
		return () => {
			document.removeEventListener("visibilitychange", onVisibilityChange);
			stop();
		};
	}, [delayMs, enabled]);
}
