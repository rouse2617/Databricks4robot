// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { Deployment } from "../api/pipelineApi";
import PipelinePage from "./PipelinePage";

// ── Mock pipelineApi ──────────────────────────────────────────────
const mockSavePipeline = vi.fn();
const mockDeployTemplate = vi.fn();
const mockListPipelines = vi.fn().mockResolvedValue([]);
const mockListDeployments = vi.fn().mockResolvedValue([]);
const mockGetPipeline = vi.fn();
const mockDeletePipeline = vi.fn();
const mockDeleteDeployment = vi.fn();
const mockPreviewDeploy = vi.fn();
const mockRetryDeployment = vi.fn();
const mockListExecutionTargets = vi.fn().mockResolvedValue([
	{
		id: "default",
		name: "Default Argo target",
		cluster: "default",
		namespace: "cyber-databrew-dev",
		argoServerConfigured: true,
		status: "available",
		isDefault: true,
	},
]);

vi.mock("../api/pipelineApi", () => ({
	savePipeline: (...args: unknown[]) => mockSavePipeline(...args),
	deployTemplate: (...args: unknown[]) => mockDeployTemplate(...args),
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
	listDeployments: (...args: unknown[]) => mockListDeployments(...args),
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	getPipeline: (...args: unknown[]) => mockGetPipeline(...args),
	deletePipeline: (...args: unknown[]) => mockDeletePipeline(...args),
	deleteDeployment: (...args: unknown[]) => mockDeleteDeployment(...args),
	previewDeploy: (...args: unknown[]) => mockPreviewDeploy(...args),
	retryDeployment: (...args: unknown[]) => mockRetryDeployment(...args),
}));

// ── Mock pipelineComponentApi ─────────────────────────────────────
const mockListComponents = vi.fn().mockResolvedValue({ items: [] });

vi.mock("../api/pipelineComponentApi", () => ({
	listComponents: (...args: unknown[]) => mockListComponents(...args),
	createComponent: vi.fn(),
	updateComponent: vi.fn(),
	deleteComponent: vi.fn(),
}));

// ── Mock AssetPicker ──────────────────────────────────────────────
vi.mock("../components/pipeline/AssetPicker", () => ({
	default: ({
		selectedIds,
		onSelectionChange,
	}: {
		selectedIds: string[];
		onSelectionChange: (ids: string[]) => void;
	}) => (
		<div data-testid="mock-asset-picker">
			<span>Selected: {selectedIds.join(",") || "(none)"}</span>
			<button
				type="button"
				data-testid="select-assets-btn"
				onClick={() => onSelectionChange(["ast-001", "ast-002"])}
			>
				MockSelect
			</button>
		</div>
	),
}));

// ── Mock antd (message as spies inside factory) ───────────────────
vi.mock("antd", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		message: { success: vi.fn(), error: vi.fn() },
	};
});

// ── Helpers ───────────────────────────────────────────────────────
function renderPage() {
	return render(
		<MemoryRouter>
			<PipelinePage />
		</MemoryRouter>,
	);
}

function mockDeployResult(overrides: Partial<Deployment> = {}): Deployment {
	return {
		id: "dep-001",
		pipelineName: "test-pipeline",
		workflowName: "wf-test-001",
		status: "Succeeded",
		nodeCount: 3,
		createdAt: "2026-05-28T12:00:00Z",
		...overrides,
	};
}

/** Import a pipeline with one node so canvas is non-empty for deploy tests. */
async function importOneNodePipeline(customName = "test-pipeline") {
	const pipeline = {
		name: customName,
		version: "1",
		nodes: [
			{
				id: "step-1",
				component: {
					name: "test",
					image: "busybox:latest",
					command: [],
					args: [],
				},
				inputs: [{ name: "input", type: "string" }],
				outputs: [{ name: "output", type: "string" }],
			},
		],
		edges: [],
	};
	fireEvent.click(screen.getByText("导入"));
	const modal = document.querySelector(".ant-modal");
	expect(modal).toBeTruthy();
	const textarea = modal?.querySelector("textarea") as HTMLTextAreaElement;
	expect(textarea).toBeTruthy();
	fireEvent.change(textarea, {
		target: { value: JSON.stringify(pipeline) },
	});
	fireEvent.click(
		within(modal as HTMLElement).getByRole("button", { name: /导.*入/ }),
	);
	await waitFor(() => {
		expect(screen.getByDisplayValue(customName)).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: /play-circle/i }),
		).not.toBeDisabled();
	});
}

async function getMockMessage() {
	const antd: typeof import("antd") = await import("antd");
	return antd.message;
}

/** Find the modal's primary deploy button (not the canvas toolbar one). */
function getModalDeployBtn(): HTMLButtonElement {
	const modal = screen.getByText("部署流水线").closest(".ant-modal");
	expect(modal).toBeTruthy();
	const btn = within(modal as HTMLElement).getByRole("button", {
		name: /运行/,
	});
	expect(btn).not.toBeNull();
	expect(btn).not.toBeDisabled();
	return btn as HTMLButtonElement;
}

// ── Suite ─────────────────────────────────────────────────────────
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

function resetPipelineMocks() {
	mockSavePipeline.mockReset();
	mockDeployTemplate.mockReset();
	mockListPipelines.mockResolvedValue([]);
	mockListDeployments.mockResolvedValue([]);
	mockGetPipeline.mockReset();
	mockDeletePipeline.mockReset();
	mockDeleteDeployment.mockReset();
	mockPreviewDeploy.mockReset();
	mockRetryDeployment.mockReset();
	mockListExecutionTargets.mockResolvedValue([
		{
			id: "default",
			name: "Default Argo target",
			cluster: "default",
			namespace: "cyber-databrew-dev",
			argoServerConfigured: true,
			status: "available",
			isDefault: true,
		},
	]);
	mockListComponents.mockResolvedValue({ items: [] });
}

describe("PipelinePage", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		resetPipelineMocks();
	});

	afterEach(() => {
		cleanup();
	});

	// ── Render & structure ──────────────────────────────────────────
	it("renders the design toolbar and node config panel", () => {
		renderPage();
		expect(screen.getAllByText("组件").length).toBeGreaterThanOrEqual(1);
		expect(screen.getByText("保存")).toBeInTheDocument();
		expect(screen.getByText("节点配置")).toBeInTheDocument();
		expect(
			screen.getByText("已保存流水线请到「流水线」页签管理。"),
		).toBeInTheDocument();
	});

	it("shows canvas toolbar buttons by default", () => {
		renderPage();
		expect(screen.getAllByText("部署").length).toBeGreaterThanOrEqual(1);
		expect(screen.getByText("保存")).toBeInTheDocument();
		expect(screen.getByText("导出")).toBeInTheDocument();
		expect(screen.getByText("导入")).toBeInTheDocument();
	});

	it("shows the pipeline name input with default value", () => {
		renderPage();
		expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
	});

	// ── Tab switching ───────────────────────────────────────────────
	it("shows component palette on canvas view", () => {
		renderPage();
		expect(screen.getByRole("heading", { name: "组件" })).toBeInTheDocument();
	});

	it("shows saved pipelines in the pipeline management tab", async () => {
		mockListPipelines.mockResolvedValueOnce([
			{
				id: "tmpl-001",
				name: "saved-flow",
				nodeCount: 2,
				createdAt: "2026-06-01T09:00:00Z",
			},
		]);
		renderPage();
		fireEvent.click(screen.getByText("管理已保存的流水线"));
		await waitFor(() => {
			expect(screen.getByText("流水线管理")).toBeInTheDocument();
			expect(screen.getByText("saved-flow")).toBeInTheDocument();
		});
	});

	// ── Export ──────────────────────────────────────────────────────
	it("shows JSON output on export", () => {
		renderPage();
		fireEvent.click(screen.getByText("导出"));
		const pre = document.querySelector(".json-output");
		expect(pre).toBeInTheDocument();
		expect(pre?.textContent).toContain('"version"');
	});

	// ── Import ──────────────────────────────────────────────────────
	it("loads pipeline from JSON import", async () => {
		renderPage();
		await importOneNodePipeline("imported-pipeline");
		await waitFor(() => {
			expect(screen.getByDisplayValue("imported-pipeline")).toBeInTheDocument();
		});
		expect(document.querySelector(".json-output")).not.toBeInTheDocument();
	});

	it("shows error on invalid JSON import", async () => {
		renderPage();
		fireEvent.click(screen.getByText("导入"));
		const modal = document.querySelector(".ant-modal");
		expect(modal).toBeTruthy();
		const textarea = modal?.querySelector("textarea") as HTMLTextAreaElement;
		fireEvent.change(textarea, { target: { value: "invalid json{{}" } });
		fireEvent.click(
			within(modal as HTMLElement).getByRole("button", { name: /导.*入/ }),
		);
		const msg = await getMockMessage();
		expect(msg.error).toHaveBeenCalledWith("无效的 JSON");
	});

	it("does nothing on cancelled import", () => {
		renderPage();
		fireEvent.click(screen.getByText("导入"));
		const modal = document.querySelector(".ant-modal");
		expect(modal).toBeTruthy();
		fireEvent.click(
			within(modal as HTMLElement).getByRole("button", { name: /取.*消/ }),
		);
		expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
	});

	// ── Save ────────────────────────────────────────────────────────
	it("saves pipeline on save button click", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "my-pipeline",
		});
		renderPage();
		fireEvent.click(screen.getByText("保存"));
		await waitFor(() => expect(mockSavePipeline).toHaveBeenCalledTimes(1));
		expect(mockSavePipeline).toHaveBeenCalledWith(
			"my-pipeline",
			expect.objectContaining({ name: "my-pipeline" }),
		);
		const msg = await getMockMessage();
		expect(msg.success).toHaveBeenCalledWith(expect.stringContaining("已保存"));
	});

	it("shows error on save failure", async () => {
		mockSavePipeline.mockRejectedValueOnce(new Error("Network error"));
		renderPage();
		fireEvent.click(screen.getByText("保存"));
		const msg = await getMockMessage();
		await waitFor(() =>
			expect(msg.error).toHaveBeenCalledWith(
				expect.stringContaining("保存失败"),
			),
		);
	});

	// ── Deploy dialog ──────────────────────────────────────────────
	it("disables toolbar deploy when canvas is empty", () => {
		renderPage();
		expect(screen.getByRole("button", { name: /play-circle/i })).toBeDisabled();
		expect(screen.getByText("拖入组件开始设计")).toBeInTheDocument();
	});

	it("opens deploy modal with title", async () => {
		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		expect(screen.getByText("部署流水线")).toBeInTheDocument();
	});

	it("shows node count in deploy modal (with nodes)", async () => {
		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		expect(screen.getByText("1 个节点")).toBeInTheDocument();
	});

	it("deploys pipeline with nodes", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-nodes",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage();
		await importOneNodePipeline("with-nodes");

		// Open deploy modal
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		await waitFor(() => {
			expect(screen.getByText("1 个节点")).toBeInTheDocument();
		});

		// Change workflow name
		const nameInput = document.getElementById(
			"pp-workflow-name",
		) as HTMLInputElement;
		expect(nameInput).toBeTruthy();
		fireEvent.change(nameInput, { target: { value: "my-workflow" } });

		// Click the modal's primary deploy button
		fireEvent.click(getModalDeployBtn());

		await waitFor(() => {
			expect(mockSavePipeline).toHaveBeenCalled();
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				undefined,
				"default",
			);
		});

		await waitFor(() => {
			// Ant Design 5 adds spacing in Chinese chars ("关 闭"), use regex
			expect(screen.getByText(/查看 Workflow/)).toBeInTheDocument();
			expect(screen.getByText(/关.*闭/)).toBeInTheDocument();
		});
	});

	it("deploys with selected assets", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-nodes",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));

		await waitFor(() =>
			expect(screen.getByTestId("mock-asset-picker")).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByTestId("select-assets-btn"));

		fireEvent.click(getModalDeployBtn());

		await waitFor(() => {
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				["ast-001", "ast-002"],
				"default",
			);
		});
	});

	it("shows error on deploy failure", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-nodes",
		});
		mockDeployTemplate.mockRejectedValueOnce(new Error("Cluster unavailable"));

		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		fireEvent.click(getModalDeployBtn());

		await waitFor(() => {
			expect(screen.getByText(/部署失败/)).toBeInTheDocument();
			expect(screen.getByText(/Cluster unavailable/)).toBeInTheDocument();
		});
	});

	it("navigates to workflow on '查看 Workflow'", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-nodes",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		fireEvent.click(getModalDeployBtn());

		await waitFor(() =>
			expect(screen.getByText("查看 Workflow")).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByText("查看 Workflow"));
	});

	it("opens execution records on '查看记录'", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-nodes",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage();
		await importOneNodePipeline("with-nodes");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
		fireEvent.click(getModalDeployBtn());

		await waitFor(() =>
			expect(screen.getByText(/查看记录/)).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByText(/查看记录/));

		await waitFor(() => {
			expect(screen.getByRole("tab", { name: /执行记录/ })).toHaveAttribute(
				"aria-selected",
				"true",
			);
		});
	});

	// ── Component registry ──────────────────────────────────────────
	it("loads components from API on mount", async () => {
		mockListComponents.mockResolvedValueOnce({
			items: [
				{
					id: "comp-1",
					name: "processor",
					image: "alpine",
					tag: "latest",
					source: "manual",
					inputPorts: [{ name: "input", type: "string" }],
					outputPorts: [{ name: "output", type: "string" }],
					resources: { command: ["sh", "-c"], args: [] },
					envVars: [],
				},
			],
		});
		renderPage();
		await waitFor(() => expect(mockListComponents).toHaveBeenCalledTimes(1));
	});

	it("falls back to localStorage when API fails", async () => {
		mockListComponents.mockRejectedValueOnce(new Error("API down"));
		renderPage();
		await waitFor(() => expect(mockListComponents).toHaveBeenCalledTimes(1));
	});

	// ── sessionStorage ──────────────────────────────────────────────
	it("loads pipeline from sessionStorage on mount", async () => {
		sessionStorage.setItem(
			"pipeline-edit",
			JSON.stringify({
				name: "from-storage",
				version: "1",
				nodes: [],
				edges: [],
			}),
		);
		renderPage();
		await waitFor(
			() =>
				expect(screen.getByDisplayValue("from-storage")).toBeInTheDocument(),
			{ timeout: 3000 },
		);
	});

	it("handles empty sessionStorage gracefully", () => {
		renderPage();
		expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
	});

	it("handles invalid JSON in sessionStorage gracefully", () => {
		sessionStorage.setItem("pipeline-edit", "{bad json");
		renderPage();
		expect(screen.getByDisplayValue("my-pipeline")).toBeInTheDocument();
	});
});
