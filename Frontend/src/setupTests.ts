import "@testing-library/jest-dom/vitest";
import { afterAll, beforeAll } from "vitest";

const PSEUDO_STYLE_WARNING =
	"Not implemented: Window's getComputedStyle() method: with pseudo-elements";

const originalConsoleError = console.error;
const originalGetComputedStyle = window.getComputedStyle.bind(window);

class MemoryStorage implements Storage {
	private store = new Map<string, string>();

	get length() {
		return this.store.size;
	}

	clear() {
		this.store.clear();
	}

	getItem(key: string) {
		return this.store.get(key) ?? null;
	}

	key(index: number) {
		return Array.from(this.store.keys())[index] ?? null;
	}

	removeItem(key: string) {
		this.store.delete(key);
	}

	setItem(key: string, value: string) {
		this.store.set(key, String(value));
	}
}

class MockResizeObserver {
	observe() {}
	unobserve() {}
	disconnect() {}
}

beforeAll(() => {
	for (const key of ["localStorage", "sessionStorage"] as const) {
		const storage = window[key] ?? new MemoryStorage();
		if (window[key] == null) {
			Object.defineProperty(window, key, {
				configurable: true,
				value: storage,
			});
		}
		if (globalThis[key] == null) {
			Object.defineProperty(globalThis, key, {
				configurable: true,
				value: storage,
			});
		}
	}

	Object.defineProperty(window, "getComputedStyle", {
		configurable: true,
		value: ((elt: Element, _pseudoElt?: string) => {
			// jsdom does not implement pseudo-element style resolution.
			return originalGetComputedStyle(elt);
		}) as typeof window.getComputedStyle,
	});

	window.ResizeObserver =
		MockResizeObserver as unknown as typeof ResizeObserver;

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
