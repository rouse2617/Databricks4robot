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
import type { PipelineComponentAPI } from "../api/pipelineComponentApi";
import { ComponentManager } from "./ComponentManager";

const apiMocks = vi.hoisted(() => ({
	createComponent: vi.fn(),
	deleteComponent: vi.fn(),
	listComponents: vi.fn(),
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
			tolerations: [
				{
					key: "nvidia.com/gpu",
					operator: "Equal",
					value: "present",
					effect: "NoSchedule",
				},
			],
		},
		inputPorts: [{ name: "summary", type: "string" }],
		outputPorts: [{ name: "report", type: "string" }],
		createdAt: "2026-06-02T15:04:20+08:00",
		updatedAt: "2026-06-02T15:04:20+08:00",
	},
];

const ANT_DESIGN_TEST_TIMEOUT_MS = 60_000;

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
	apiMocks.updateComponent.mockReset();
	apiMocks.listComponents.mockResolvedValue({ items: components });
});

afterEach(() => {
	cleanup();
});

describe("page ComponentManager", () => {
	it(
		"prefills the edit form with the selected component",
		async () => {
			render(<ComponentManager />);

			expect(await screen.findByText("报告生成")).toBeTruthy();
			fireEvent.click(screen.getByRole("button", { name: /编辑组件/ }));

			await waitFor(() => {
				expect(
					getInputByPlaceholder(document.body, "normalize-mcap").value,
				).toBe("报告生成");
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
				expect(screen.getByDisplayValue("nvidia.com/gpu")).toBeTruthy();
			});
		},
		ANT_DESIGN_TEST_TIMEOUT_MS,
	);

	it(
		"saves gpu and compute tier resource fields",
		async () => {
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
			expect(apiMocks.updateComponent.mock.calls[0][1].resources).toMatchObject(
				{
					cpu: "4000m",
					memory: "16Gi",
					disk: "50Gi",
					gpu: "2",
					computeTier: "gpu-a100",
					tolerations: [
						expect.objectContaining({
							key: "nvidia.com/gpu",
							value: "present",
							effect: "NoSchedule",
						}),
					],
				},
			);
		},
		ANT_DESIGN_TEST_TIMEOUT_MS,
	);

	it(
		"rejects bare memory resource values before create",
		async () => {
			render(<ComponentManager />);

			expect(await screen.findByText("报告生成")).toBeTruthy();
			fireEvent.click(screen.getByRole("button", { name: /新建组件/ }));
			fireEvent.change(getInputByPlaceholder(document.body, "normalize-mcap"), {
				target: { value: "bad-memory" },
			});
			fireEvent.change(
				getInputByPlaceholder(
					document.body,
					"registry.example.com/databrew/worker",
				),
				{ target: { value: "alpine:latest" } },
			);
			fireEvent.change(getResourceInput(document.body, "memory"), {
				target: { value: "1" },
			});
			fireEvent.click(screen.getByRole("button", { name: "创 建" }));

			await waitFor(() => {
				expect(apiMocks.createComponent).not.toHaveBeenCalled();
				expect(document.body.textContent).toContain("内存必须带单位");
			});
		},
		ANT_DESIGN_TEST_TIMEOUT_MS,
	);

	it(
		"allows explicit memory units on create",
		async () => {
			render(<ComponentManager />);

			expect(await screen.findByText("报告生成")).toBeTruthy();
			fireEvent.click(screen.getByRole("button", { name: /新建组件/ }));
			fireEvent.change(getInputByPlaceholder(document.body, "normalize-mcap"), {
				target: { value: "good-memory" },
			});
			fireEvent.change(
				getInputByPlaceholder(
					document.body,
					"registry.example.com/databrew/worker",
				),
				{ target: { value: "alpine:latest" } },
			);
			fireEvent.change(getResourceInput(document.body, "memory"), {
				target: { value: "512Mi" },
			});
			fireEvent.click(screen.getByRole("button", { name: "创 建" }));

			await waitFor(() => {
				expect(apiMocks.createComponent).toHaveBeenCalled();
			});
			expect(apiMocks.createComponent.mock.calls[0][0].resources).toMatchObject(
				{
					memory: "512Mi",
				},
			);
		},
		ANT_DESIGN_TEST_TIMEOUT_MS,
	);

	it(
		"keeps validation errors local and still allows closing the create modal",
		async () => {
			render(<ComponentManager />);

			expect(await screen.findByText("报告生成")).toBeTruthy();
			fireEvent.click(screen.getByRole("button", { name: /新建组件/ }));
			fireEvent.click(screen.getByRole("button", { name: "创 建" }));

			expect(await screen.findByText("请输入组件名称")).toBeTruthy();
			expect(apiMocks.createComponent).not.toHaveBeenCalled();

			fireEvent.click(screen.getByRole("button", { name: "关 闭" }));

			await waitFor(() => {
				expect(
					document.body.querySelector('input[placeholder="normalize-mcap"]'),
				).toBeNull();
				expect(screen.queryByText("请输入组件名称")).toBeNull();
			});
		},
		ANT_DESIGN_TEST_TIMEOUT_MS,
	);
});
