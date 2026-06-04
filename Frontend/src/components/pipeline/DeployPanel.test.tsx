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
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import { App } from "antd";
import type { Deployment, PipelineTemplate } from "../../api/pipelineApi";
import { DeployPanel } from "./DeployPanel";

const mockNavigate = vi.hoisted(() => vi.fn());

vi.mock("react-router-dom", async (importOriginal) => {
	const actual = await importOriginal<Record<string, unknown>>();
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

// Mock pipeline API
const mockListPipelines = vi.fn();
const mockListPipelineVersions = vi.fn();
const mockListDeployments = vi.fn();
const mockDeployTemplate = vi.fn();
const mockBatchDeployTemplate = vi.fn();
const mockDeletePipeline = vi.fn();
const mockGetPipeline = vi.fn();
const mockListExecutionTargets = vi.fn();

vi.mock("../../api/pipelineApi", () => ({
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
	listPipelineVersions: (...args: unknown[]) =>
		mockListPipelineVersions(...args),
	listDeployments: (...args: unknown[]) => mockListDeployments(...args),
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	deployTemplate: (...args: unknown[]) => mockDeployTemplate(...args),
	batchDeployTemplate: (...args: unknown[]) => mockBatchDeployTemplate(...args),
	deletePipeline: (...args: unknown[]) => mockDeletePipeline(...args),
	getPipeline: (...args: unknown[]) => mockGetPipeline(...args),
}));

// Mock antd message to suppress console noise
vi.mock("antd", async () => {
	const actual = await vi.importActual("antd");
	return {
		...(actual as Record<string, unknown>),
		message: {
			success: vi.fn(),
			error: vi.fn(),
		},
	};
});

// Mock AssetPicker subcomponent
vi.mock("./AssetPicker", () => ({
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

const mockTemplate = (
	overrides: Partial<PipelineTemplate> = {},
): PipelineTemplate => ({
	id: "tmpl-001",
	name: "test-pipeline",
	version: 1,
	pipeline: { name: "test-pipeline", version: "1", nodes: [], edges: [] },
	nodeCount: 3,
	createdAt: "2026-05-27T12:00:00Z",
	...overrides,
});

const mockDeployment = (overrides: Partial<Deployment> = {}): Deployment => ({
	id: "dep-001",
	pipelineName: "test-pipeline",
	workflowName: "wf-test-001",
	status: "Succeeded",
	nodeCount: 3,
	createdAt: "2026-05-27T12:30:00Z",
	finishedAt: "2026-05-27T12:35:00Z",
	...overrides,
});

function renderDeployPanel(
	onEditTemplate?: (pipeline: PipelineTemplate["pipeline"]) => void,
	initialEntry = "/pipeline",
) {
	return render(
		<App>
			<MemoryRouter initialEntries={[initialEntry]}>
				<DeployPanel onEditTemplate={onEditTemplate} />
			</MemoryRouter>
		</App>,
	);
}

function renderCompactDeployPanel() {
	return render(
		<App>
			<MemoryRouter>
				<DeployPanel variant="compact" />
			</MemoryRouter>
		</App>,
	);
}

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

afterEach(() => {
	vi.restoreAllMocks();
	mockNavigate.mockClear();
	cleanup();
});

beforeEach(() => {
	vi.clearAllMocks();
	mockListPipelineVersions.mockResolvedValue([]);
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
});

describe("DeployPanel", () => {
	it("shows empty state when no templates exist", async () => {
		mockListPipelines.mockResolvedValue([]);
		renderDeployPanel();

		expect(await screen.findByText("暂无已保存的流水线模板")).toBeTruthy();
		expect(screen.queryByText("运行历史")).toBeNull();
		expect(screen.queryByText("暂无部署记录")).toBeNull();
	});

	it("loads and displays templates on mount", async () => {
		mockListPipelines.mockResolvedValue([
			mockTemplate({ id: "tmpl-001", name: "my-pipeline", nodeCount: 5 }),
		]);
		mockListDeployments.mockResolvedValue([
			mockDeployment({
				id: "dep-001",
				pipelineName: "my-pipeline",
				status: "Succeeded",
			}),
		]);
		renderDeployPanel();

		const names = await screen.findAllByText("my-pipeline");
		expect(names).toHaveLength(1);
		expect(screen.queryByText("Succeeded")).toBeNull();
		expect(mockListDeployments).not.toHaveBeenCalled();
	});

	it("calls deployTemplate on direct run", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockDeployTemplate.mockResolvedValue(
			mockDeployment({ id: "dep-002", workflowName: "wf-test-002" }),
		);
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		await waitFor(() => {
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				[],
				"default",
				1,
			);
		});
		expect(mockListPipelineVersions).toHaveBeenCalledWith("tmpl-001");
	});

	it("shows error toast when direct deploy fails", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockDeployTemplate.mockRejectedValue(new Error("K8s error"));
		const messageError = vi.fn();
		const useAppSpy = vi.spyOn(App, "useApp").mockReturnValue({
			message: {
				success: vi.fn(),
				error: messageError,
				warning: vi.fn(),
				info: vi.fn(),
				loading: vi.fn(),
				open: vi.fn(),
				destroy: vi.fn(),
			},
		} as ReturnType<typeof App.useApp>);
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		await waitFor(() => {
			expect(messageError).toHaveBeenCalledWith(
				expect.stringContaining("部署失败"),
			);
		});
		useAppSpy.mockRestore();
	});

	it("opens run modal and deploys with asset ids", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockBatchDeployTemplate.mockResolvedValue({
			batchId: "batch-001",
			items: [
				mockDeployment({ id: "dep-003a" }),
				mockDeployment({ id: "dep-003b" }),
			],
		});
		renderDeployPanel();

		// Wait for template to render
		expect(await screen.findByText("运行")).toBeTruthy();

		fireEvent.click(screen.getByText("运行"));

		// Modal should open
		await waitFor(() => {
			expect(screen.getByText("运行流水线")).toBeTruthy();
		});

		// Asset picker is shown
		expect(screen.getByTestId("mock-asset-picker")).toBeTruthy();

		// First select assets via the mock picker button
		fireEvent.click(screen.getByTestId("select-assets-btn"));

		// Click modal OK button — find the primary button in the modal footer
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		await waitFor(() => {
			expect(mockBatchDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				["ast-001", "ast-002"],
				"default",
				1,
			);
		});
		expect(mockDeployTemplate).not.toHaveBeenCalled();
	});

	it("uses asset ids from pipeline url when running a saved template", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockListPipelineVersions.mockResolvedValue([
			mockTemplate({ id: "tmpl-v2", version: 2, nodeCount: 5 }),
			mockTemplate({ id: "tmpl-001", version: 1, nodeCount: 3 }),
		]);
		mockBatchDeployTemplate.mockResolvedValue({
			batchId: "batch-002",
			items: [
				mockDeployment({ id: "dep-005a" }),
				mockDeployment({ id: "dep-005b" }),
			],
		});

		renderDeployPanel(
			undefined,
			"/pipeline?tab=pipelines&asset_ids=ast-a,ast-b",
		);

		expect(await screen.findByText("已选择 2 个资产")).toBeTruthy();
		expect(
			screen.getByText("请选择要运行的流水线和版本，确认后即可提交运行。"),
		).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));

		await waitFor(() => {
			expect(screen.getByText("运行流水线")).toBeTruthy();
			expect(
				screen.getByText("将为 2 个资产各创建 1 个独立 Workflow"),
			).toBeTruthy();
			expect(screen.getByTestId("mock-asset-picker").textContent).toContain(
				"Selected: ast-a,ast-b",
			);
		});

		fireEvent.mouseDown(screen.getByRole("combobox", { name: /模板版本/i }));
		fireEvent.click(await screen.findByText(/版本 v2/));

		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		await waitFor(() => {
			expect(mockBatchDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				["ast-a", "ast-b"],
				"default",
				2,
			);
		});
	});

	it("deploys without asset IDs when none selected", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-002" })]);
		mockListDeployments.mockResolvedValue([]);
		mockDeployTemplate.mockResolvedValue(mockDeployment({ id: "dep-004" }));
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();

		fireEvent.click(screen.getByText("运行"));

		await waitFor(() => {
			expect(screen.getByText("运行流水线")).toBeTruthy();
		});

		// Verify no assets are pre-selected
		expect(screen.getByTestId("mock-asset-picker").textContent).toContain(
			"(none)",
		);

		// Click OK without selecting assets — should pass an explicit empty asset list
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		await waitFor(() => {
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-002",
				[],
				"default",
				1,
			);
		});
	});

	it("calls handleEditTemplate with onEditTemplate callback", async () => {
		const onEdit = vi.fn();
		const pipeline = {
			name: "test-pipeline",
			version: "1",
			nodes: [],
			edges: [],
		};
		mockListPipelines.mockResolvedValue([
			mockTemplate({ id: "tmpl-001", pipeline }),
		]);
		mockListDeployments.mockResolvedValue([]);
		mockGetPipeline.mockResolvedValue(
			mockTemplate({ id: "tmpl-001", pipeline }),
		);
		renderDeployPanel(onEdit);

		expect(await screen.findByText("编辑")).toBeTruthy();
		fireEvent.click(screen.getByText("编辑"));

		await waitFor(() => {
			expect(mockGetPipeline).toHaveBeenCalledWith("tmpl-001");
			expect(onEdit).toHaveBeenCalledWith(pipeline);
		});
	});

	it("navigates to the designer with templateId when onEditTemplate is not provided", async () => {
		const pipeline = {
			name: "test-pipeline",
			version: "1",
			nodes: [],
			edges: [],
		};
		mockListPipelines.mockResolvedValue([
			mockTemplate({ id: "tmpl-001", pipeline }),
		]);
		mockListDeployments.mockResolvedValue([]);
		mockGetPipeline.mockResolvedValue(
			mockTemplate({ id: "tmpl-001", pipeline }),
		);

		renderDeployPanel();

		expect(await screen.findByText("编辑")).toBeTruthy();
		fireEvent.click(screen.getByText("编辑"));

		await waitFor(() => {
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline?templateId=tmpl-001",
			);
		});
	});

	it("shows error when load template fails on edit", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockGetPipeline.mockRejectedValue(new Error("not found"));
		const messageError = vi.fn();
		const useAppSpy = vi.spyOn(App, "useApp").mockReturnValue({
			message: {
				success: vi.fn(),
				error: messageError,
				warning: vi.fn(),
				info: vi.fn(),
				loading: vi.fn(),
				open: vi.fn(),
				destroy: vi.fn(),
			},
		} as ReturnType<typeof App.useApp>);
		renderDeployPanel();

		expect(await screen.findByText("编辑")).toBeTruthy();
		fireEvent.click(screen.getByText("编辑"));

		await waitFor(() => {
			expect(messageError).toHaveBeenCalledWith(
				expect.stringContaining("加载模板失败"),
			);
		});
		useAppSpy.mockRestore();
	});

	it("deletes a template", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockDeletePipeline.mockResolvedValue(undefined);
		renderDeployPanel();

		expect(await screen.findByText("test-pipeline")).toBeTruthy();

		fireEvent.click(screen.getByRole("button", { name: /删除流水线/i }));
		await waitFor(() => {
			expect(document.querySelector(".ant-popconfirm")).toBeTruthy();
		});
		const popconfirm = document.querySelector(".ant-popconfirm") as HTMLElement;
		fireEvent.click(within(popconfirm).getByRole("button", { name: /删.*除/ }));

		await waitFor(() => {
			expect(mockDeletePipeline).toHaveBeenCalledWith("tmpl-001");
		});
	});

	it("shows recent executions in compact mode", async () => {
		mockListPipelines.mockResolvedValue([]);
		mockListDeployments.mockResolvedValue([
			mockDeployment({ id: "dep-001", workflowName: "wf-my-workflow" }),
		]);
		renderCompactDeployPanel();

		expect(await screen.findByText("最近执行")).toBeTruthy();
		expect(await screen.findByText("查看")).toBeTruthy();
		fireEvent.click(screen.getByText("查看"));
		expect(mockNavigate).toHaveBeenCalledWith(
			"/pipeline/executions/wf-my-workflow",
		);
	});

	it("refreshes data after successful deploy", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		mockDeployTemplate.mockResolvedValue(mockDeployment({ id: "dep-005" }));
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		// After deploy, refresh() is called — verify deployTemplate was called
		await waitFor(() => {
			expect(mockDeployTemplate).toHaveBeenCalledWith(
				"tmpl-001",
				[],
				"default",
				1,
			);
		});

		// refresh() triggers listPipelines again (1 mount + 1 refresh = 2)
		expect(mockListPipelines.mock.calls.length).toBeGreaterThanOrEqual(2);
	});

	it("shows modal with no-asset run warning", async () => {
		mockListPipelines.mockResolvedValue([mockTemplate({ id: "tmpl-001" })]);
		mockListDeployments.mockResolvedValue([]);
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();

		fireEvent.click(screen.getByText("运行"));

		await waitFor(() => {
			expect(screen.getAllByText("无资产运行").length).toBeGreaterThan(0);
			expect(
				screen.getByText(
					"本次运行不会注入资产环境变量，适合调试不依赖资产输入的流水线。",
				),
			).toBeTruthy();
			expect(screen.getByText("运行流水线")).toBeTruthy();
		});
	});

	it("handles listing API failure gracefully", async () => {
		mockListPipelines.mockRejectedValue(new Error("server down"));
		mockListDeployments.mockRejectedValue(new Error("server down"));
		renderDeployPanel();

		// Should show empty state without crashing
		expect(await screen.findByText("暂无已保存的流水线模板")).toBeTruthy();
		expect(screen.queryByText("暂无部署记录")).toBeNull();
	});
});
