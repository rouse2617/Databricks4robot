// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import type {
	PipelineComponentAPI,
	PipelineComponentReleaseAPI,
} from "../api/pipelineComponentApi";
import { ComponentManager } from "./ComponentManager";

const apiMocks = vi.hoisted(() => ({
	createComponent: vi.fn(),
	deleteComponent: vi.fn(),
	listComponents: vi.fn(),
	listComponentReleases: vi.fn(),
	syncComponentReleases: vi.fn(),
	updateComponent: vi.fn(),
}));

vi.mock("../api/pipelineComponentApi", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("../api/pipelineComponentApi")>();
	return {
		...actual,
		createComponent: apiMocks.createComponent,
		deleteComponent: apiMocks.deleteComponent,
		listComponents: apiMocks.listComponents,
		listComponentReleases: apiMocks.listComponentReleases,
		syncComponentReleases: apiMocks.syncComponentReleases,
		updateComponent: apiMocks.updateComponent,
	};
});

const components: PipelineComponentAPI[] = [
	{
		id: "ef937c98-fa16-4025-ade1-67f59a007024",
		name: "报告生成",
		type: "container",
		description: "汇总全流程指标",
		image: "alpine:3.18",
		tag: "latest",
		source: "custom",
		command: ["sh", "-c"],
		args: ["echo ok"],
		env: {},
		resources: {
			cpu: "4000m",
			memory: "16Gi",
			disk: "50Gi",
			gpu: "1",
			computeTier: "gpu-l4",
		},
		inputPorts: [{ name: "summary", type: "string" }],
		outputPorts: [{ name: "report", type: "string" }],
		createdAt: "2026-06-02T15:04:20+08:00",
		updatedAt: "2026-06-02T15:04:20+08:00",
	},
];

const releaseFromRegistry: PipelineComponentReleaseAPI = {
	id: "b126b7fd-9fa8-5fea-bf1d-0e2d331577be",
	imageUid: "26a77873",
	componentId: "hand-track-stereo-databrew-test",
	taskName: "hand-track-stereo-databrew-test",
	displayName: "hand-track-stereo-databrew-test",
	owner: "ci",
	releaseLabel: "97f1ae4",
	channel: "candidate",
	sourceRepo: "CyberOrigin2077/automated-processing-gcloud",
	sourceRef: "97f1ae4-test-ref",
	sourceRefType: "commit",
	sourceCommit: "97f1ae4-test-ref",
	buildId: "build-123",
	imageRepo:
		"us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/hand-track-stereo",
	imageTag: "manual-20260617-055352",
	imageDigest: "sha256:test-digest-303d372",
	runtimeImage:
		"us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/hand-track-stereo:manual-20260617-055352",
	status: "ready",
	selectable: true,
	validationStatus: "passed",
	validationErrors: [],
	runtimeSnapshot: {
		image:
			"us-central1-docker.pkg.dev/green-valley-442103/video-proc-images/hand-track-stereo:manual-20260617-055352",
		command: ["python", "src/main.py"],
		inputPorts: [{ name: "input", type: "asset" }],
		outputPorts: [{ name: "output", type: "asset" }],
		resources: {
			cpu: "14000m",
			memory: "55Gi",
			gpu: "1",
			gpuType: "nvidia-l4",
		},
	},
	technicalMetadata: {
		taskDir: "tasks/hand_track_stereo_databrew_test",
		cloudBuildTrigger: "hand-track-stereo-build-trigger",
	},
	createdAt: "2026-06-18T07:30:00Z",
	updatedAt: "2026-06-18T07:35:00Z",
	lastSyncedAt: "2026-06-18T07:35:00Z",
};

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

function getInputByPlaceholder(container: HTMLElement, placeholder: string) {
	const input = container.querySelector<HTMLInputElement>(
		`input[placeholder="${placeholder}"]`,
	);
	expect(input).toBeTruthy();
	return input as HTMLInputElement;
}

function getResourceInput(container: HTMLElement, name: string) {
	const input = container.querySelector<HTMLInputElement>(
		`input[data-testid="component-resource-${name}"]`,
	);
	expect(input).toBeTruthy();
	return input as HTMLInputElement;
}

beforeEach(() => {
	apiMocks.createComponent.mockReset();
	apiMocks.deleteComponent.mockReset();
	apiMocks.listComponents.mockReset();
	apiMocks.listComponentReleases.mockReset();
	apiMocks.syncComponentReleases.mockReset();
	apiMocks.updateComponent.mockReset();
	apiMocks.listComponents.mockResolvedValue({ items: components });
	apiMocks.listComponentReleases.mockResolvedValue({ items: [] });
});

afterEach(() => {
	cleanup();
});

describe("page ComponentManager", () => {
	it("prefills the edit form with the selected component", async () => {
		render(<ComponentManager />);

		expect(await screen.findByText("报告生成")).toBeTruthy();
		fireEvent.click(screen.getByRole("button", { name: /编辑组件/ }));

		await waitFor(() => {
			expect(getInputByPlaceholder(document.body, "normalize-mcap").value).toBe(
				"报告生成",
			);
			expect(
				getInputByPlaceholder(
					document.body,
					"registry.example.com/databrew/worker",
				).value,
			).toBe("alpine:3.18");
			expect(getResourceInput(document.body, "cpu").value).toBe("4000m");
			expect(getResourceInput(document.body, "memory").value).toBe("16Gi");
			expect(getResourceInput(document.body, "disk").value).toBe("50Gi");
			expect(getResourceInput(document.body, "gpu").value).toBe("1");
			expect(getResourceInput(document.body, "compute-tier").value).toBe(
				"gpu-l4",
			);
		});
	});

	it("saves gpu and compute tier resource fields", async () => {
		render(<ComponentManager />);

		expect(await screen.findByText("报告生成")).toBeTruthy();
		fireEvent.click(screen.getByRole("button", { name: /编辑组件/ }));

		await waitFor(() => {
			expect(getResourceInput(document.body, "gpu").value).toBe("1");
		});
		fireEvent.change(getResourceInput(document.body, "gpu"), {
			target: { value: "2" },
		});
		fireEvent.change(getResourceInput(document.body, "compute-tier"), {
			target: { value: "gpu-a100" },
		});
		fireEvent.click(screen.getByRole("button", { name: "保 存" }));

		await waitFor(() => {
			expect(apiMocks.updateComponent).toHaveBeenCalled();
		});
		expect(apiMocks.updateComponent.mock.calls[0][1].resources).toMatchObject({
			cpu: "4000m",
			memory: "16Gi",
			disk: "50Gi",
			gpu: "2",
			computeTier: "gpu-a100",
		});
	}, 60000);

	it("keeps validation errors local and still allows closing the create modal", async () => {
		render(<ComponentManager />);

		expect(await screen.findByText("报告生成")).toBeTruthy();
		fireEvent.click(screen.getByRole("button", { name: /新建组件/ }));
		fireEvent.click(screen.getByRole("button", { name: "创 建" }));

		await waitFor(() => {
			expect(apiMocks.createComponent).not.toHaveBeenCalled();
			expect(screen.getByRole("dialog", { name: "新建组件" })).toBeTruthy();
		});

		fireEvent.click(screen.getByRole("button", { name: "关 闭" }));

		await waitFor(() => {
			expect(
				document.body.querySelector('input[placeholder="normalize-mcap"]'),
			).toBeNull();
		});
	}, 60000);

	it("shows release task path and separates image tag from digest identity", async () => {
		apiMocks.listComponents.mockResolvedValue({ items: [] });
		apiMocks.listComponentReleases.mockResolvedValue({
			items: [releaseFromRegistry],
		});

		render(<ComponentManager />);

		expect(
			await screen.findByText("hand-track-stereo-databrew-test"),
		).toBeTruthy();
		expect(screen.getByText("来自版本库 · 1 个构建版本")).toBeTruthy();
		expect(screen.getByText("commit-97f1ae4")).toBeTruthy();
		expect(screen.getByText("镜像 Tag：manual-20260617-055352")).toBeTruthy();
		expect(screen.getByText("Digest：26a77873")).toBeTruthy();

		fireEvent.click(screen.getByRole("button", { name: /查看当前版本/ }));

		await waitFor(() => {
			expect(screen.getByRole("dialog", { name: "镜像构建详情" })).toBeTruthy();
		});
		expect(
			screen.getByText("tasks/hand_track_stereo_databrew_test"),
		).toBeTruthy();
		expect(screen.getByText("hand-track-stereo-build-trigger")).toBeTruthy();
		expect(
			screen.getAllByText("manual-20260617-055352").length,
		).toBeGreaterThan(0);
	}, 60000);
});
