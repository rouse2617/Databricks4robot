// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import type { PipelineConfig } from "../api/pipelineConfigs";
import RegistryCenterPage from "./RegistryCenterPage";

const mockListConfigs = vi.fn();
const mockGetConfig = vi.fn();

vi.mock("../api/pipelineConfigs", () => ({
	pipelineConfigApi: {
		list: (...args: unknown[]) => mockListConfigs(...args),
		get: (...args: unknown[]) => mockGetConfig(...args),
		create: vi.fn(),
		update: vi.fn(),
		createVersion: vi.fn(),
		getVersion: vi.fn(),
		deprecate: vi.fn(),
	},
}));

vi.mock("../api/registry", () => ({
	registryApi: {
		listAlgos: vi.fn().mockResolvedValue([]),
		listTags: vi.fn().mockResolvedValue([]),
		listMetrics: vi.fn().mockResolvedValue([]),
		listLifecycleStates: vi.fn().mockResolvedValue(["draft", "ready"]),
	},
}));

const configs: PipelineConfig[] = [
	{
		id: "cfg-ready",
		name: "active-ready.yaml",
		description: "ready config",
		owner: "alice@example.com",
		scope: "dev",
		tags: ["active"],
		fileType: "yaml",
		lifecycle: "ready",
		currentVersion: 1,
		versionCount: 1,
		versions: [
			{
				id: "cfg-ready-v1",
				configId: "cfg-ready",
				version: 1,
				status: "ready",
				contentSha256: "abcdef0123456789",
				contentSizeBytes: 10,
				summary: "ready",
				author: "alice@example.com",
				createdAt: "2026-06-18T00:00:00Z",
			},
		],
		createdAt: "2026-06-18T00:00:00Z",
		updatedAt: "2026-06-18T00:00:00Z",
	},
	{
		id: "cfg-draft",
		name: "active-draft.yaml",
		description: "draft config",
		owner: "alice@example.com",
		scope: "dev",
		tags: ["active"],
		fileType: "yaml",
		lifecycle: "draft",
		currentVersion: 1,
		versionCount: 1,
		versions: [
			{
				id: "cfg-draft-v1",
				configId: "cfg-draft",
				version: 1,
				status: "draft",
					contentSha256: "draft-sha",
				contentSizeBytes: 10,
				summary: "draft",
				author: "alice@example.com",
				createdAt: "2026-06-18T00:00:00Z",
			},
		],
		createdAt: "2026-06-18T00:00:00Z",
		updatedAt: "2026-06-18T00:00:00Z",
	},
	{
		id: "cfg-archived",
		name: "archived-config.yaml",
		description: "archived config",
		owner: "alice@example.com",
		scope: "dev",
		tags: ["old"],
		fileType: "yaml",
		lifecycle: "deprecated",
		currentVersion: 1,
		versionCount: 1,
		versions: [
			{
				id: "cfg-archived-v1",
				configId: "cfg-archived",
				version: 1,
				status: "deprecated",
					contentSha256: "archived-sha",
				contentSizeBytes: 10,
				summary: "archived",
				author: "alice@example.com",
				createdAt: "2026-06-18T00:00:00Z",
			},
		],
		createdAt: "2026-06-18T00:00:00Z",
		updatedAt: "2026-06-18T00:00:00Z",
	},
];

beforeAll(() => {
	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
});

beforeEach(() => {
	mockListConfigs.mockResolvedValue({ items: configs });
	mockGetConfig.mockImplementation((id: string) =>
		Promise.resolve(configs.find((config) => config.id === id)),
	);
});

afterEach(() => {
	vi.clearAllMocks();
	cleanup();
});

describe("RegistryCenterPage", () => {
	it("keeps archived configs out of the default workspace list", async () => {
		render(<RegistryCenterPage />);

		await screen.findByText("active-ready.yaml");

		expect(screen.getByText("active-ready.yaml")).toBeInTheDocument();
		expect(screen.getByText("active-draft.yaml")).toBeInTheDocument();
		expect(screen.queryByText("archived-config.yaml")).not.toBeInTheDocument();

		fireEvent.click(screen.getByText("归档区 1"));

		expect(screen.getByText("archived-config.yaml")).toBeInTheDocument();
		expect(screen.queryByText("active-ready.yaml")).not.toBeInTheDocument();
		expect(screen.queryByText("基于当前版本新建")).not.toBeInTheDocument();
	});
});
