import "@testing-library/jest-dom/vitest";
import { afterAll, beforeAll } from "vitest";

const PSEUDO_STYLE_WARNING =
	"Not implemented: Window's getComputedStyle() method: with pseudo-elements";

const originalConsoleError = console.error;
const originalGetComputedStyle = window.getComputedStyle.bind(window);

class MockResizeObserver {
	observe() {}
	unobserve() {}
	disconnect() {}
}

beforeAll(() => {
	Object.defineProperty(window, "getComputedStyle", {
		configurable: true,
		value: ((elt: Element, _pseudoElt?: string) => {
			// jsdom does not implement pseudo-element style resolution.
			return originalGetComputedStyle(elt);
		}) as typeof window.getComputedStyle,
	});

	window.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver;

	console.error = (...args: unknown[]) => {
		if (typeof args[0] === "string" && args[0].includes(PSEUDO_STYLE_WARNING)) {
			return;
		}
		originalConsoleError(...args);
	};
});

afterAll(() => {
	Object.defineProperty(window, "getComputedStyle", {
		configurable: true,
		value: originalGetComputedStyle,
	});
	console.error = originalConsoleError;
});
