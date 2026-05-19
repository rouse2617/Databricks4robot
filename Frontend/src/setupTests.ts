import { afterAll, beforeAll } from "vitest";

const PSEUDO_STYLE_WARNING =
	"Not implemented: Window's getComputedStyle() method: with pseudo-elements";

const originalConsoleError = console.error;
const originalGetComputedStyle = window.getComputedStyle.bind(window);

beforeAll(() => {
	Object.defineProperty(window, "getComputedStyle", {
		configurable: true,
		value: ((elt: Element, _pseudoElt?: string) => {
			// jsdom does not implement pseudo-element style resolution.
			return originalGetComputedStyle(elt);
		}) as typeof window.getComputedStyle,
	});

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
