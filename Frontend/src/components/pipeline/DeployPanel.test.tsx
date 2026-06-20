// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { StrictMode } from "react";
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
import type { Deployment, PipelineTemplate } from "../../api/pipelineApi";
import { buildDeployConfigSelection, DeployPanel } from "./DeployPanel";

const mockNavigate = vi.hoisted(() => vi.fn());
const mockMessage = vi.hoisted(() => ({
	success: vi.fn(),
	error: vi.fn(),
	warning: vi.fn(),
}));

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
const mockDeployPipelineForAssets = vi.fn();
const mockDeletePipeline = vi.fn();
const mockGetPipeline = vi.fn();
const mockListExecutionTargets = vi.fn();
const mockListPipelineConfigs = vi.fn();

vi.mock("../../api/pipelineApi", () => ({
	listPipelines: (...args: unknown[]) => mockListPipelines(...args),
	listPipelineVersions: (...args: unknown[]) =>
		mockListPipelineVersions(...args),
	listDeployments: (...args: unknown[]) => mockListDeployments(...args),
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	deletePipeline: (...args: unknown[]) => mockDeletePipeline(...args),
	getPipeline: (...args: unknown[]) => mockGetPipeline(...args),
}));

vi.mock("../../api/pipelineConfigs", () => ({
	pipelineConfigApi: {
		list: (...args: unknown[]) => mockListPipelineConfigs(...args),
	},
}));

vi.mock("../../api/deployPipelineRun", () => ({
	deployPipelineForAssets: (...args: unknown[]) =>
		mockDeployPipelineForAssets(...args),
	BATCH_ASSET_THRESHOLD: 2,
}));

// Mock antd message / App.useApp to suppress console noise
vi.mock("antd", async () => {
	const actual = await vi.importActual<typeof import("antd")>("antd");
	const AppMock = ({ children }: { children: ReactNode }) => children;
	(AppMock as typeof actual.App).useApp = () => ({ message: mockMessage });
	return {
		...actual,
		message: mockMessage,
		App: AppMock,
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

const pipelinesResponse = (items: PipelineTemplate[], total?: number) => ({
	items,
	total: total ?? items.length,
	page: 1,
	pageSize: 10,
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
		<MemoryRouter initialEntries={[initialEntry]}>
			<DeployPanel onEditTemplate={onEditTemplate} />
		</MemoryRouter>,
	);
}

function renderCompactDeployPanel() {
	return render(
		<MemoryRouter>
			<DeployPanel variant="compact" />
		</MemoryRouter>,
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
	mockListPipelineConfigs.mockResolvedValue({
		items: [
			{
				id: "cfg-001",
				name: "detector.yaml",
				description: "Detector thresholds",
				owner: "sdk",
				scope: "dev",
				tags: ["vision"],
				fileType: "yaml",
				lifecycle: "ready",
				currentVersion: 2,
				versionCount: 2,
				createdAt: "2026-06-18T00:00:00Z",
				updatedAt: "2026-06-18T00:00:00Z",
			},
		],
	});
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
	mockDeployPipelineForAssets.mockImplementation(
		async (templateId, assetIds, _options) => {
			if (assetIds.length >= 2) {
				return {
					mode: "batch",
					batchJob: {
						id: "batch-001",
						name: "demo-batch",
						templateId,
						totalCount: assetIds.length,
						completedCount: 0,
						failedCount: 0,
						status: "running",
						createdAt: "2026-05-27T12:30:00Z",
						updatedAt: "2026-05-27T12:30:00Z",
					},
				};
			}
			return {
				mode: "single",
				runs: [
					mockDeployment({
						id: "dep-auto",
						workflowName: "wf-auto",
					}),
				],
			};
		},
	);
});

describe("DeployPanel", () => {
	it("shows empty state when no templates exist", async () => {
		mockListPipelines.mockResolvedValue(pipelinesResponse([]));
		renderDeployPanel();

		expect(await screen.findByText("暂无已保存的流水线模板")).toBeTruthy();
		expect(screen.queryByText("运行历史")).toBeNull();
		expect(screen.queryByText("暂无部署记录")).toBeNull();
	});

	it("loads and displays templates on mount", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([
				mockTemplate({ id: "tmpl-001", name: "my-pipeline", nodeCount: 5 }),
			]),
		);
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

	it("calls deployPipelineForAssets on direct run", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
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
			expect(mockDeployPipelineForAssets).toHaveBeenCalledWith(
				"tmpl-001",
				[],
				expect.objectContaining({
					configSelection: undefined,
					targetId: "default",
					version: 1,
				}),
			);
		});
		expect(mockListPipelineVersions).toHaveBeenCalledWith("tmpl-001");
	});

	it("shows error toast when direct deploy fails", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		mockDeployPipelineForAssets.mockRejectedValue(new Error("K8s error"));
		renderDeployPanel();

		expect(await screen.findByText("运行")).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		const deployBtn = document.querySelector(
			".ant-modal-footer .ant-btn-primary",
		);
		expect(deployBtn).toBeTruthy();
		if (deployBtn) fireEvent.click(deployBtn);

		const { message } = await import("antd");
		await waitFor(() => {
			expect(message.error).toHaveBeenCalledWith(
				expect.stringContaining("部署失败"),
			);
		});
	});

	it("opens run modal and creates batch job with two asset ids", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
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
			expect(mockDeployPipelineForAssets).toHaveBeenCalledWith(
				"tmpl-001",
				["ast-001", "ast-002"],
				expect.objectContaining({
					configSelection: undefined,
					targetId: "default",
					version: 1,
				}),
			);
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline/batch/batch-001",
				expect.anything(),
			);
		});
	});

	it("builds saved config selection payload for deploy requests", () => {
		expect(
			buildDeployConfigSelection({
				configSourceMode: "saved",
				selectedSavedConfig: {
					id: "cfg-001",
					name: "detector.yaml",
					description: "Detector thresholds",
					owner: "sdk",
					scope: "dev",
					tags: ["vision"],
					fileType: "yaml",
					lifecycle: "ready",
					currentVersion: 2,
					versionCount: 2,
					createdAt: "2026-06-18T00:00:00Z",
					updatedAt: "2026-06-18T00:00:00Z",
				},
				uploadDraftFile: null,
				inlineDraftName: "runtime-config.yaml",
				inlineDraftContent: "",
				configMountPath: "/workspace/configs",
				configTargetFilename: "detector.yaml",
			}),
		).toEqual({
			mode: "saved",
			configId: "cfg-001",
			version: 2,
			fileName: "detector.yaml",
			mountPath: "/workspace/configs",
			targetFilename: "detector.yaml",
		});
	});

	it("uses asset ids from pipeline url when running a saved template", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		mockListPipelineVersions.mockResolvedValue([
			mockTemplate({ id: "tmpl-v2", version: 2, nodeCount: 5 }),
			mockTemplate({ id: "tmpl-001", version: 1, nodeCount: 3 }),
		]);

		renderDeployPanel(
			undefined,
			"/pipeline?tab=pipelines&asset_ids=ast-a,ast-b",
		);

		expect(await screen.findByText("已选择 2 个资产")).toBeTruthy();
		expect(
			screen.getByText(/请选择要运行的流水线和版本，确认后即可提交运行。/),
		).toBeTruthy();
		fireEvent.click(screen.getByText("运行"));

		await waitFor(() => {
			expect(screen.getByText("运行流水线")).toBeTruthy();
			expect(screen.getByText("将创建批量任务，共 2 个子任务")).toBeTruthy();
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
			expect(mockDeployPipelineForAssets).toHaveBeenCalledWith(
				"tmpl-001",
				["ast-a", "ast-b"],
				expect.objectContaining({
					targetId: "default",
					version: 2,
				}),
			);
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline/batch/batch-001",
				expect.anything(),
			);
		});
	});

	it("deploys without asset IDs when none selected", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-002" })]),
		);
		mockListDeployments.mockResolvedValue([]);
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
			expect(mockDeployPipelineForAssets).toHaveBeenCalledWith(
				"tmpl-002",
				[],
				expect.objectContaining({
					targetId: "default",
					version: 1,
				}),
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
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001", pipeline })]),
		);
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
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001", pipeline })]),
		);
		mockListDeployments.mockResolvedValue([]);
		mockGetPipeline.mockResolvedValue(
			mockTemplate({ id: "tmpl-001", pipeline }),
		);

		renderDeployPanel();

		expect(await screen.findByText("编辑")).toBeTruthy();
		fireEvent.click(screen.getByText("编辑"));

		await waitFor(() => {
			expect(mockNavigate).toHaveBeenCalledWith(
				"/pipeline?templateId=tmpl-001&tab=design",
			);
		});
	});

	it("shows error when load template fails on edit", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		mockGetPipeline.mockRejectedValue(new Error("not found"));
		renderDeployPanel();

		expect(await screen.findByText("编辑")).toBeTruthy();
		fireEvent.click(screen.getByText("编辑"));

		const { message } = await import("antd");
		await waitFor(() => {
			expect(message.error).toHaveBeenCalledWith(
				expect.stringContaining("加载模板失败"),
			);
		});
	});

	it("deletes a template", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
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
		mockListPipelines.mockResolvedValue(pipelinesResponse([]));
		mockListDeployments.mockResolvedValue([
			mockDeployment({ id: "dep-001", workflowName: "wf-my-workflow" }),
		]);
		renderCompactDeployPanel();

		expect(await screen.findByText("最近执行")).toBeTruthy();
		expect(await screen.findByText("查看")).toBeTruthy();
		fireEvent.click(screen.getByText("查看"));
		expect(mockNavigate).toHaveBeenCalledWith("/runs/dep-001");
	});

	it("refreshes data after successful deploy", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
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
			expect(mockDeployPipelineForAssets).toHaveBeenCalledWith(
				"tmpl-001",
				[],
				expect.objectContaining({
					targetId: "default",
					version: 1,
				}),
			);
		});

		// refresh() triggers listPipelines again (1 mount + 1 refresh = 2)
		expect(mockListPipelines.mock.calls.length).toBeGreaterThanOrEqual(2);
	});

	it("dedupes listPipelines refresh under StrictMode", async () => {
		mockListPipelines.mockResolvedValue(pipelinesResponse([]));
		mockListDeployments.mockResolvedValue([]);
		render(
			<StrictMode>
				<MemoryRouter initialEntries={["/pipeline"]}>
					<DeployPanel />
				</MemoryRouter>
			</StrictMode>,
		);
		await waitFor(() => {
			expect(mockListPipelines).toHaveBeenCalledTimes(1);
		});
	});

	it("shows modal with no-asset run warning", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
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

	it("refetches only pipelines when template filters change", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ name: "alpha-pipeline" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		renderDeployPanel();

		await waitFor(() => {
			expect(mockListPipelines).toHaveBeenCalledTimes(1);
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1);
		});

		const searchInput = screen.getByTestId("pipeline-template-search");
		fireEvent.change(searchInput, { target: { value: "alpha" } });
		fireEvent.keyDown(searchInput, { key: "Enter", code: "Enter" });

		await waitFor(() => {
			expect(mockListPipelines).toHaveBeenCalledTimes(2);
		});
		expect(mockListExecutionTargets).toHaveBeenCalledTimes(1);
		expect(mockListPipelines.mock.calls[1]?.[0]).toMatchObject({
			q: "alpha",
			page: 1,
		});
	});

	it("shows saved config mode in deploy modal and loads platform configs", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		renderDeployPanel();

		fireEvent.click(await screen.findByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		expect(screen.getByTestId("deploy-config-panel")).toBeTruthy();
		expect(screen.getByText("高级全局配置（兼容）")).toBeTruthy();
		expect(mockListPipelineConfigs).not.toHaveBeenCalled();

		fireEvent.click(screen.getByText("启用全局配置 fallback"));
		expect(screen.getByText("选择已保存配置")).toBeTruthy();

		await waitFor(() => {
			expect(mockListPipelineConfigs).toHaveBeenCalledTimes(1);
		});
		expect(
			screen.getByRole("combobox", { name: /选择已保存配置/i }),
		).toBeTruthy();
		expect(screen.getByText("还未选择平台配置")).toBeTruthy();
	});

	it("shows upload draft metadata in deploy modal", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		renderDeployPanel();

		fireEvent.click(await screen.findByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		fireEvent.click(screen.getByText("启用全局配置 fallback"));
		fireEvent.click(screen.getByText("上传本地文件"));

		const input = screen.getByLabelText("上传配置文件") as HTMLInputElement;
		const file = new File(["threshold: 0.82\n"], "runtime.yaml", {
			type: "text/yaml",
		});
		fireEvent.change(input, { target: { files: [file] } });

		expect(await screen.findByText("runtime.yaml")).toBeTruthy();
		expect(screen.getByText(/本地文件仅作为本次 deploy 草稿/)).toBeTruthy();
	});

	it("shows inline editor summary and mount target", async () => {
		mockListPipelines.mockResolvedValue(
			pipelinesResponse([mockTemplate({ id: "tmpl-001" })]),
		);
		mockListDeployments.mockResolvedValue([]);
		renderDeployPanel();

		fireEvent.click(await screen.findByText("运行"));
		expect(await screen.findByText("运行流水线")).toBeTruthy();
		fireEvent.click(screen.getByText("启用全局配置 fallback"));
		fireEvent.click(screen.getByText("在线编辑"));

		fireEvent.change(screen.getByLabelText("在线编辑文件名"), {
			target: { value: "inline-config.yaml" },
		});
		fireEvent.change(screen.getByLabelText("在线编辑配置内容"), {
			target: { value: "threshold: 0.90\nwindow: 3\n" },
		});
		fireEvent.change(screen.getByLabelText("挂载目录"), {
			target: { value: "/workspace/configs" },
		});
		fireEvent.change(screen.getByLabelText("目标文件名"), {
			target: { value: "effective.yaml" },
		});

		expect(
			await screen.findByText(/来源：在线编辑 · inline-config.yaml/),
		).toBeTruthy();
		expect(
			screen.getByText(/挂载到 \/workspace\/configs\/effective.yaml/),
		).toBeTruthy();
	});
});
