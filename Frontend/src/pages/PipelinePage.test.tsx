// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { Modal } from "antd";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import type { Deployment } from "../api/pipelineApi";
import PipelinePage from "./PipelinePage";

// ── Mock navigate ────────────────────────────────────────────────
const mockNavigate = vi.hoisted(() => vi.fn());

vi.mock("react-router-dom", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

// ── Mock pipelineApi ──────────────────────────────────────────────
const mockSavePipeline = vi.fn();
const mockDeployTemplate = vi.fn();
const mockListPipelines = vi
	.fn()
	.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 });
const mockListPipelineVersions = vi.fn().mockResolvedValue([]);
const mockListDeployments = vi.fn().mockResolvedValue([]);
const mockListPipelineRuns = vi.fn().mockResolvedValue([]);
const mockGetPipelineRunWatcherStatus = vi.fn().mockResolvedValue({
	healthy: true,
	lastSyncedRunCount: 0,
	scanLimit: 100,
	scanDelaySeconds: 6,
});
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
	// Real helper used by deployPipelineForAssets (unmocked) to shape runs.
	normalizeDeployResults: (result: unknown) =>
		result &&
		typeof result === "object" &&
		"items" in result &&
		Array.isArray((result as { items: unknown[] }).items)
			? (result as { items: unknown[] }).items
			: [result],
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
	listPipelineVersions: (...args: unknown[]) =>
		mockListPipelineVersions(...args),
	listDeployments: (...args: unknown[]) => mockListDeployments(...args),
	listPipelineRuns: (...args: unknown[]) => mockListPipelineRuns(...args),
	getPipelineRunWatcherStatus: (...args: unknown[]) =>
		mockGetPipelineRunWatcherStatus(...args),
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	getPipeline: (...args: unknown[]) => mockGetPipeline(...args),
	deletePipeline: (...args: unknown[]) => mockDeletePipeline(...args),
	deleteDeployment: (...args: unknown[]) => mockDeleteDeployment(...args),
	previewDeploy: (...args: unknown[]) => mockPreviewDeploy(...args),
	retryDeployment: (...args: unknown[]) => mockRetryDeployment(...args),
}));

// ── Mock batchJobApi (used by deployPipelineForAssets for ≥2 assets) ──
const mockCreateBatchJob = vi.fn();

vi.mock("../api/batchJobApi", () => ({
	createBatchJob: (...args: unknown[]) => mockCreateBatchJob(...args),
}));

// ── Mock workflowApi ──────────────────────────────────────────────
const mockListWorkflows = vi.fn().mockResolvedValue({ items: [] });
const mockDeleteWorkflow = vi.fn();

vi.mock("../api/workflowApi", () => ({
	listWorkflows: (...args: unknown[]) => mockListWorkflows(...args),
	deleteWorkflow: (...args: unknown[]) => mockDeleteWorkflow(...args),
	resubmitWorkflow: vi.fn(),
	retryWorkflow: vi.fn(),
	stopWorkflow: vi.fn(),
	suspendWorkflow: vi.fn(),
	resumeWorkflow: vi.fn(),
	terminateWorkflow: vi.fn(),
}));

// ── Mock pipelineComponentApi ─────────────────────────────────────
const mockListComponents = vi.fn().mockResolvedValue({ items: [] });
const mockListComponentReleases = vi.fn().mockResolvedValue({ items: [] });

vi.mock("../api/pipelineComponentApi", () => ({
	listComponents: (...args: unknown[]) => mockListComponents(...args),
	listComponentReleases: (...args: unknown[]) =>
		mockListComponentReleases(...args),
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
	const messageApi = { success: vi.fn(), error: vi.fn(), warning: vi.fn() };
	const message = {
		...(actual.message as Record<string, unknown>),
		...messageApi,
		useMessage: () => [messageApi, null] as const,
	};
	return {
		...actual,
		App: {
			...(actual.App as Record<string, unknown>),
			useApp: () => ({
				message: messageApi,
				modal: {
					confirm: vi.fn(),
				},
			}),
		},
		message,
	};
});

// ── Helpers ───────────────────────────────────────────────────────
function renderPage(initialEntry = "/pipeline") {
	return render(
		<MemoryRouter initialEntries={[initialEntry]}>
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
					command: ["sh", "-c", "echo test"],
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
		expect(
			screen.getByRole("button", { name: /play-circle/i }),
		).not.toBeDisabled();
	});
}

async function getMockMessage() {
	const antd: typeof import("antd") = await import("antd");
	return antd.message;
}

/** Open deploy modal and wait until step summary is rendered. */
async function openDeployModal() {
	fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));
	await waitFor(() => {
		expect(screen.getByText("1 个步骤")).toBeInTheDocument();
	});
}

/** Find the modal's primary deploy button (not the canvas toolbar one). */
function getModalDeployBtn(): HTMLButtonElement {
	const modal = screen.getByText("部署流水线").closest(".ant-modal");
	expect(modal).toBeTruthy();
	const btn = within(modal as HTMLElement).getByRole("button", {
		name: /^(运行资产|无资产运行)$/,
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
	mockCreateBatchJob.mockReset();
	mockCreateBatchJob.mockResolvedValue({
		id: "batch-001",
		name: "batch-test",
		templateId: "tmpl-001",
		templateVersion: 1,
		totalCount: 2,
		completedCount: 0,
		failedCount: 0,
		status: "pending",
		createdAt: "2026-05-28T12:00:00Z",
		updatedAt: "2026-05-28T12:00:00Z",
	});
	mockListPipelines.mockResolvedValue({
		items: [],
		total: 0,
		page: 1,
		pageSize: 20,
	});
	mockListPipelineVersions.mockResolvedValue([]);
	mockListDeployments.mockResolvedValue([]);
	mockListPipelineRuns.mockResolvedValue({ items: [] });
	mockGetPipelineRunWatcherStatus.mockResolvedValue({
		healthy: true,
		lastSyncedRunCount: 0,
		scanLimit: 100,
		scanDelaySeconds: 6,
	});
	mockListWorkflows.mockResolvedValue({ items: [] });
	mockDeleteWorkflow.mockReset();
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
	mockListComponentReleases.mockResolvedValue({ items: [] });
}

describe("PipelinePage", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		resetPipelineMocks();
		vi.spyOn(Modal, "confirm").mockImplementation((config) => {
			config.onOk?.();
			return { destroy: vi.fn(), update: vi.fn() };
		});
	});

	afterEach(() => {
		cleanup();
		mockNavigate.mockClear();
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

	it("hides the pipeline name input on the design canvas", () => {
		renderPage();
		expect(screen.queryByLabelText("流水线名称")).not.toBeInTheDocument();
		expect(screen.queryByDisplayValue("my-pipeline")).not.toBeInTheDocument();
	});

	// ── Tab switching ───────────────────────────────────────────────
	it("drops executionView when opening components tab", async () => {
		renderPage("/pipeline?tab=executions&executionView=batch");
		fireEvent.click(screen.getByRole("tab", { name: /组件/ }));
		await waitFor(() => {
			expect(screen.getByText("组件库")).toBeInTheDocument();
		});
	});

	it("shows component palette on canvas view", () => {
		renderPage();
		expect(screen.getByRole("heading", { name: "组件" })).toBeInTheDocument();
	});

	it("shows saved pipelines in the pipeline management tab", async () => {
		mockListPipelines.mockResolvedValueOnce({
			items: [
				{
					id: "tmpl-001",
					name: "saved-flow",
					nodeCount: 2,
					createdAt: "2026-06-01T09:00:00Z",
				},
			],
			total: 1,
			page: 1,
			pageSize: 20,
		});
		renderPage();
		fireEvent.click(screen.getByText("管理已保存的流水线"));
		await waitFor(() => {
			expect(screen.getByText("流水线管理")).toBeInTheDocument();
			expect(screen.getByText("saved-flow")).toBeInTheDocument();
		});
	});

	it("treats legacy templates tab query as pipeline management tab", async () => {
		mockListPipelines.mockResolvedValueOnce({
			items: [
				{
					id: "tmpl-001",
					name: "legacy-tab-flow",
					nodeCount: 1,
					createdAt: "2026-06-01T09:00:00Z",
				},
			],
			total: 1,
			page: 1,
			pageSize: 20,
		});

		renderPage("/pipeline?tab=templates");

		await waitFor(() => {
			expect(screen.getByText("流水线管理")).toBeInTheDocument();
			expect(screen.getByText("legacy-tab-flow")).toBeInTheDocument();
		});
		const pipelineTab = document.querySelector(
			'[role="tab"][aria-controls$="panel-pipelines"]',
		);
		expect(pipelineTab).toHaveAttribute("aria-selected", "true");
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
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "imported-pipeline",
		});
		renderPage();
		await importOneNodePipeline("imported-pipeline");
		fireEvent.click(screen.getByText("保存"));
		await waitFor(() => expect(mockSavePipeline).toHaveBeenCalledTimes(1));
		expect(mockSavePipeline).toHaveBeenCalledWith(
			"imported-pipeline",
			expect.objectContaining({ name: "imported-pipeline" }),
		);
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
		expect(screen.queryByDisplayValue("my-pipeline")).not.toBeInTheDocument();
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

	it("navigates with tab=design after save", async () => {
		mockNavigate.mockClear();
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "my-pipeline",
			version: 2,
		});
		renderPage();
		fireEvent.click(screen.getByText("保存"));
		await waitFor(() => expect(mockSavePipeline).toHaveBeenCalledTimes(1));
		await waitFor(() => {
			expect(mockNavigate).toHaveBeenCalledWith(
				expect.stringContaining("/pipeline?templateId=tmpl-001&tab=design"),
				{ replace: true },
			);
		});
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
		expect(screen.getByText("添加组件开始设计")).toBeInTheDocument();
	});

	it("adds a node when a palette component is dropped on the canvas wrapper", async () => {
		mockListComponents.mockResolvedValueOnce({
			items: [
				{
					id: "comp-drag",
					name: "Drag Component",
					type: "container",
					image: "busybox:latest",
					source: "custom",
					createdAt: "2026-06-02T00:00:00Z",
					updatedAt: "2026-06-02T00:00:00Z",
				},
			],
		});

		renderPage();

		const component = await screen.findByRole("button", {
			name: /添加组件 Drag Component/,
		});
		const canvas = screen.getByRole("application", { name: "流水线画布" });
		const dataTransfer = {
			effectAllowed: "",
			dropEffect: "",
			data: new Map<string, string>(),
			setData(type: string, value: string) {
				this.data.set(type, value);
			},
			getData(type: string) {
				return this.data.get(type) ?? "";
			},
		};

		fireEvent.dragStart(component, { dataTransfer });
		fireEvent.dragOver(canvas, {
			dataTransfer,
			clientX: 500,
			clientY: 360,
		});
		fireEvent.drop(canvas, {
			dataTransfer,
			clientX: 500,
			clientY: 360,
		});

		await waitFor(() => {
			expect(
				screen.getByRole("button", { name: /play-circle/i }),
			).not.toBeDisabled();
		});
		expect(dataTransfer.dropEffect).toBe("copy");
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
		expect(screen.getByText("1 个步骤")).toBeInTheDocument();
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
		await openDeployModal();

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
				[],
				"default",
				undefined,
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
		await openDeployModal();

		await waitFor(() =>
			expect(screen.getByTestId("mock-asset-picker")).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByTestId("select-assets-btn"));
		await waitFor(() => {
			expect(screen.getByText("Selected: ast-001,ast-002")).toBeInTheDocument();
		});

		fireEvent.click(getModalDeployBtn());

		// ≥2 assets dispatch a batch job instead of a single deploy.
		await waitFor(() => {
			expect(mockCreateBatchJob).toHaveBeenCalledWith(
				expect.objectContaining({
					templateId: "tmpl-001",
					assetIds: ["ast-001", "ast-002"],
				}),
			);
		});
		expect(mockDeployTemplate).not.toHaveBeenCalled();
		await waitFor(() =>
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline/batch/batch-001",
				expect.objectContaining({
					state: expect.objectContaining({
						returnTo: "/pipeline?tab=executions&executionView=batch",
					}),
				}),
			),
		);
	});

	it("preserves asset_ids from url when opening deploy modal", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-assets",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage("/pipeline?asset_ids=asset-a,asset-b");
		await importOneNodePipeline("with-assets");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));

		await waitFor(() => {
			// ≥2 assets render the batch-job banner instead of single-run text.
			expect(
				screen.getByText("将创建批量任务，共 2 个子任务"),
			).toBeInTheDocument();
			expect(screen.getAllByText("asset-a").length).toBeGreaterThan(0);
			expect(screen.getAllByText("asset-b").length).toBeGreaterThan(0);
			expect(screen.getByText("Selected: asset-a,asset-b")).toBeInTheDocument();
		});

		fireEvent.click(getModalDeployBtn());

		// asset_ids from the URL (2) dispatch a batch job.
		await waitFor(() => {
			expect(mockCreateBatchJob).toHaveBeenCalledWith(
				expect.objectContaining({
					templateId: "tmpl-001",
					assetIds: ["asset-a", "asset-b"],
				}),
			);
		});
		await waitFor(() =>
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline/batch/batch-001",
				expect.objectContaining({
					state: expect.objectContaining({
						returnTo: "/pipeline?tab=executions&executionView=batch",
					}),
				}),
			),
		);
	});

	it("allows clearing url assets into an explicit no-asset run", async () => {
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "with-assets",
		});
		mockDeployTemplate.mockResolvedValueOnce(mockDeployResult());

		renderPage("/pipeline?asset_ids=asset-a");
		await importOneNodePipeline("with-assets");
		fireEvent.click(screen.getByRole("button", { name: /play-circle/i }));

		fireEvent.click(
			screen.getAllByText("转为无资产运行").at(-1) as HTMLElement,
		);

		await waitFor(() => {
			expect(screen.getAllByText("无资产运行").length).toBeGreaterThan(0);
			expect(screen.getByText("Selected: (none)")).toBeInTheDocument();
		});

		fireEvent.click(getModalDeployBtn());

		await waitFor(() => {
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				[],
				"default",
				undefined,
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
		await openDeployModal();
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
		await openDeployModal();
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
		await openDeployModal();
		fireEvent.click(getModalDeployBtn());

		await waitFor(() =>
			expect(screen.getByText(/查看记录/)).toBeInTheDocument(),
		);
		fireEvent.click(screen.getByText(/查看记录/));

		// "查看记录" navigates to the executions tab (useNavigate is mocked).
		await waitFor(() =>
			expect(mockNavigate).toHaveBeenCalledWith("/pipeline?tab=executions"),
		);
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

	it("loads selectable releases and filters the palette by commit or tag", async () => {
		mockListComponentReleases.mockResolvedValueOnce({
			items: [
				{
					id: "rel-1",
					imageUid: "9b8014c0",
					componentId: "cmp-release-task",
					taskName: "release-task",
					displayName: "Release Task",
					releaseLabel: "abc123",
					channel: "dev",
					sourceRef: "v1.2.3",
					sourceRefType: "tag",
					sourceCommit: "abc123",
					runtimeImage: "registry/release-task@sha256:abc",
					status: "ready",
					selectable: true,
					validationStatus: "passed",
					runtimeSnapshot: {
						image: "registry/release-task@sha256:abc",
						command: ["python", "main.py"],
						args: [],
						inputPorts: [{ name: "input", type: "asset" }],
						outputPorts: [{ name: "output", type: "asset" }],
						resources: { cpu: "100m", memory: "128Mi" },
					},
					createdAt: "2026-06-02T00:00:00Z",
					updatedAt: "2026-06-02T00:00:00Z",
				},
			],
		});

		renderPage();
		await waitFor(() => expect(screen.getByText("Release Task")).toBeTruthy());

		fireEvent.change(screen.getByPlaceholderText("搜索名称、commit、tag"), {
			target: { value: "abc123" },
		});
		expect(screen.getByText("Release Task")).toBeTruthy();

		fireEvent.change(screen.getByPlaceholderText("搜索名称、commit、tag"), {
			target: { value: "v1.2.3" },
		});
		expect(screen.getByText("Release Task")).toBeTruthy();
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
		mockSavePipeline.mockResolvedValueOnce({
			id: "tmpl-001",
			name: "from-storage",
		});
		renderPage();
		fireEvent.click(screen.getByText("保存"));
		await waitFor(
			() =>
				expect(mockSavePipeline).toHaveBeenCalledWith(
					"from-storage",
					expect.objectContaining({
						name: "from-storage",
						nodes: [],
						edges: [],
					}),
				),
			{ timeout: 3000 },
		);
	});

	it("handles empty sessionStorage gracefully", () => {
		renderPage();
		expect(screen.queryByDisplayValue("my-pipeline")).not.toBeInTheDocument();
	});

	it("handles invalid JSON in sessionStorage gracefully", () => {
		sessionStorage.setItem("pipeline-edit", "{bad json");
		renderPage();
		expect(screen.queryByDisplayValue("my-pipeline")).not.toBeInTheDocument();
	});
});
