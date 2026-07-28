// CYB-4333: workbench-shell tests. Two outer tabs render; `?view=lookup`
// hosts the BatchLookupPage; legacy `?view=durations|costs` URLs redirect
// (replace) to `?view=lookup&preset=<v>` while preserving other query
// params.

import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AssetsWorkbenchPage from "./AssetsWorkbenchPage";

// The workbench lazy-imports these; stub them so tests don't pull the whole
// AssetsPage / BatchAssetLookup tree just to verify tab wiring.
vi.mock("./AssetsPage", () => ({
	default: () => <div data-testid="assets-list-stub">assets list</div>,
}));
vi.mock("./BatchLookupPage", () => ({
	default: () => <div data-testid="batch-lookup-stub">batch lookup</div>,
}));

// AntD Tabs rely on matchMedia; jsdom does not implement it.
Object.defineProperty(window, "matchMedia", {
	writable: true,
	value: (query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false,
	}),
});

function LocationEcho() {
	const location = useLocation();
	return (
		<output data-testid="location-search">{`${location.pathname}${location.search}`}</output>
	);
}

function renderAt(entry: string) {
	return render(
		<MemoryRouter initialEntries={[entry]}>
			<Routes>
				<Route
					path="/assets"
					element={
						<>
							<LocationEcho />
							<AssetsWorkbenchPage />
						</>
					}
				/>
			</Routes>
		</MemoryRouter>,
	);
}

beforeEach(() => {
	vi.restoreAllMocks();
});

describe("AssetsWorkbenchPage tab strip", () => {
	it("renders exactly two tabs — 资产列表 and 批量查询", async () => {
		renderAt("/assets");
		expect(screen.getByText("资产列表")).toBeTruthy();
		expect(screen.getByText("批量查询")).toBeTruthy();
		// No stale labels from the pre-CYB-4333 3-tab strip.
		expect(screen.queryByText("时长批量查询")).toBeNull();
		expect(screen.queryByText("成本查询")).toBeNull();
	});

	it("renders AssetsPage on the default (list) view", async () => {
		renderAt("/assets");
		expect(await screen.findByTestId("assets-list-stub")).toBeTruthy();
	});

	it("renders BatchLookupPage when `?view=lookup`", async () => {
		renderAt("/assets?view=lookup");
		expect(await screen.findByTestId("batch-lookup-stub")).toBeTruthy();
	});
});

describe("AssetsWorkbenchPage legacy redirects", () => {
	it("`?view=durations` replaces the URL with `?view=lookup&preset=durations`", async () => {
		renderAt("/assets?view=durations");
		await waitFor(() => {
			const echo = screen.getByTestId("location-search").textContent ?? "";
			expect(echo).toContain("view=lookup");
			expect(echo).toContain("preset=durations");
		});
		expect(await screen.findByTestId("batch-lookup-stub")).toBeTruthy();
	});

	it("`?view=costs` replaces the URL with `?view=lookup&preset=costs`", async () => {
		renderAt("/assets?view=costs");
		await waitFor(() => {
			const echo = screen.getByTestId("location-search").textContent ?? "";
			expect(echo).toContain("view=lookup");
			expect(echo).toContain("preset=costs");
		});
		expect(await screen.findByTestId("batch-lookup-stub")).toBeTruthy();
	});

	it("preserves other query params when redirecting a legacy view", async () => {
		renderAt("/assets?view=durations&keep=me");
		await waitFor(() => {
			const echo = screen.getByTestId("location-search").textContent ?? "";
			expect(echo).toContain("view=lookup");
			expect(echo).toContain("preset=durations");
			expect(echo).toContain("keep=me");
		});
	});
});
