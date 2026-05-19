import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import FilesTab from "./FilesTab";

// Mock matchMedia for antd responsive components
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

describe("FilesTab", () => {
	it("renders file entries", () => {
		render(
			<MemoryRouter>
				<FilesTab
					files={{
						raw_mcap: "gs://bucket/file.mcap",
						preview: "gs://bucket/preview.mp4",
					}}
				/>
			</MemoryRouter>,
		);
		expect(screen.getByText("raw_mcap")).toBeTruthy();
		expect(screen.getByText("gs://bucket/file.mcap")).toBeTruthy();
		expect(screen.getByText("preview")).toBeTruthy();
	});

	it("links mcap_file_id values to the mcap files page", () => {
		render(
			<MemoryRouter>
				<FilesTab files={{ raw_mcap: "WK27VTTK" }} />
			</MemoryRouter>,
		);
		const link = screen.getByRole("link", { name: /WK27VTTK/ });
		expect(link.getAttribute("href")).toBe("/mcap-files?mcap_file_id=WK27VTTK");
	});

	it("renders empty state when no files", () => {
		render(
			<MemoryRouter>
				<FilesTab files={{}} />
			</MemoryRouter>,
		);
		expect(screen.getByText("暂无文件引用")).toBeTruthy();
	});
});
