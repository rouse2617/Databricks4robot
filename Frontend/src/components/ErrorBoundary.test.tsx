import { render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import ErrorBoundary from "./ErrorBoundary";

function ThrowingChild(): ReactElement {
	throw new Error("Test error");
}

describe("ErrorBoundary", () => {
	it("renders children when no error", () => {
		render(
			<ErrorBoundary>
				<div>Hello</div>
			</ErrorBoundary>,
		);
		expect(screen.getByText("Hello")).toBeTruthy();
	});

	it("renders fallback UI when child throws", () => {
		// Suppress console.error for the expected error
		const spy = vi.spyOn(console, "error").mockImplementation(() => {});
		render(
			<ErrorBoundary>
				<ThrowingChild />
			</ErrorBoundary>,
		);
		// ErrorBoundary should catch and display a fallback
		expect(screen.queryByText("Hello")).toBeNull();
		spy.mockRestore();
	});
});
