// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { StrictMode } from "react";
import {
	afterEach,
	beforeAll,
	beforeEach,
	describe,
	expect,
	it,
	vi,
} from "vitest";
import type { ExecutionTarget } from "../../api/pipelineApi";
import PoolManager from "./PoolManager";

const mockListExecutionTargets = vi.hoisted(() => vi.fn());
const mockCreateExecutionTarget = vi.hoisted(() => vi.fn());
const mockUpdateExecutionTarget = vi.hoisted(() => vi.fn());
const mockDeleteExecutionTarget = vi.hoisted(() => vi.fn());

vi.mock("../../api/pipelineApi", () => ({
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	createExecutionTarget: (...args: unknown[]) =>
		mockCreateExecutionTarget(...args),
	updateExecutionTarget: (...args: unknown[]) =>
		mockUpdateExecutionTarget(...args),
	deleteExecutionTarget: (...args: unknown[]) =>
		mockDeleteExecutionTarget(...args),
}));

const mockMessage = vi.hoisted(() => ({
	success: vi.fn(),
	error: vi.fn(),
	warning: vi.fn(),
}));

vi.mock("antd", async (importOriginal) => {
	const actual = await importOriginal<typeof import("antd")>();
	return {
		...actual,
		message: mockMessage,
	};
});

const nonDefaultTarget: ExecutionTarget = {
	id: "a03ad932-397f-4a3b-a3d8-1cfb13a6dd54",
	name: "video-proc-dev",
	cluster: "default",
	namespace: "video-proc-dev",
	argoServerConfigured: true,
	status: "available",
	isDefault: false,
	description: "Video processing dev namespace",
	resourceDefaults: {
		computeTier: "cpu-low",
		templateTolerations: [
			{
				key: "compute-tier",
				operator: "Equal",
				value: "med",
				effect: "NoSchedule",
			},
			{
				key: "durability",
				operator: "Equal",
				value: "spot",
				effect: "NoSchedule",
			},
		],
	},
};

const defaultTarget: ExecutionTarget = {
	id: "default",
	name: "Default Argo target",
	cluster: "default",
	namespace: "cyber-databrew-dev",
	argoServerConfigured: true,
	status: "available",
	isDefault: true,
	description: "Current backend-configured Argo workflow namespace.",
};

beforeAll(() => {
	Object.defineProperty(window, "matchMedia", {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			addListener: vi.fn(),
			removeListener: vi.fn(),
			dispatchEvent: vi.fn(),
		})),
	});
	// Silence ResizeObserver not defined in jsdom
	class MockResizeObserver {
		observe() {}
		unobserve() {}
		disconnect() {}
	}
	// biome-ignore lint/suspicious/noExplicitAny: jsdom polyfill
	(window as any).ResizeObserver = MockResizeObserver;
});

beforeEach(() => {
	mockListExecutionTargets.mockReset();
	mockCreateExecutionTarget.mockReset();
	mockUpdateExecutionTarget.mockReset();
	mockDeleteExecutionTarget.mockReset();
	mockMessage.success.mockClear();
	mockMessage.error.mockClear();

	mockListExecutionTargets.mockResolvedValue([nonDefaultTarget, defaultTarget]);
	mockCreateExecutionTarget.mockResolvedValue(nonDefaultTarget);
	mockUpdateExecutionTarget.mockResolvedValue(nonDefaultTarget);
	mockDeleteExecutionTarget.mockResolvedValue(undefined);

	// Mock the resource-quotas fetch (component calls fetch directly for it)
	global.fetch = vi.fn().mockResolvedValue({
		ok: true,
		json: async () => ({ items: {} }),
	}) as unknown as typeof fetch;
});

afterEach(() => {
	cleanup();
});

function renderPoolManager() {
	return render(
		<StrictMode>
			<PoolManager />
		</StrictMode>,
	);
}

describe("PoolManager — templateTolerations editor", () => {
	it("renders existing tolerations as tags in the scheduling column", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);
		expect(await screen.findByText("compute-tier=med")).toBeDefined();
		expect(screen.getByText("durability=spot")).toBeDefined();
	});

	it("prefills tolerations when opening edit modal", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);

		const editBtn = await screen.findByLabelText("edit-video-proc-dev");
		fireEvent.click(editBtn);

		// Two toleration rows pre-populated — key inputs visible
		const keyInputs = await screen.findAllByPlaceholderText(
			/key\(如 compute-tier\)/,
		);
		expect(keyInputs.length).toBe(2);
		expect((keyInputs[0] as HTMLInputElement).value).toBe("compute-tier");
		expect((keyInputs[1] as HTMLInputElement).value).toBe("durability");
	});

	it("submits update with modified toleration value", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);

		fireEvent.click(await screen.findByLabelText("edit-video-proc-dev"));

		// Change the first toleration value from "med" to "high"
		const valueInputs =
			await screen.findAllByPlaceholderText(/value\(如 med\)/);
		fireEvent.change(valueInputs[0], { target: { value: "high" } });

		// Click 保存
		const okBtn = screen.getByRole("button", { name: "保存" });
		fireEvent.click(okBtn);

		await waitFor(() =>
			expect(mockUpdateExecutionTarget).toHaveBeenCalledTimes(1),
		);
		const [id, body] = mockUpdateExecutionTarget.mock.calls[0];
		expect(id).toBe("a03ad932-397f-4a3b-a3d8-1cfb13a6dd54");
		const tolerations = body.resourceDefaults?.templateTolerations;
		expect(tolerations).toBeDefined();
		expect(tolerations?.[0]).toMatchObject({
			key: "compute-tier",
			operator: "Equal",
			value: "high",
			effect: "NoSchedule",
		});
		expect(tolerations?.[1]).toMatchObject({
			key: "durability",
			operator: "Equal",
			value: "spot",
		});
	});

	it("rejects submit when operator=Equal but value is empty", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);

		fireEvent.click(await screen.findByLabelText("edit-video-proc-dev"));

		// Clear first toleration's value
		const valueInputs =
			await screen.findAllByPlaceholderText(/value\(如 med\)/);
		fireEvent.change(valueInputs[0], { target: { value: "" } });

		fireEvent.click(screen.getByRole("button", { name: "保存" }));

		// Wait a tick — validation should fail synchronously; update must NOT fire
		await new Promise((resolve) => setTimeout(resolve, 50));
		expect(mockUpdateExecutionTarget).not.toHaveBeenCalled();
	});

	it("does not render delete/edit buttons for default target", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);
		expect(screen.queryByLabelText("delete-Default Argo target")).toBeNull();
		expect(screen.queryByLabelText("edit-Default Argo target")).toBeNull();
	});
});

describe("PoolManager — delete", () => {
	it("calls deleteExecutionTarget after confirm", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);

		fireEvent.click(await screen.findByLabelText("delete-video-proc-dev"));

		// antd Modal.confirm renders a Modal with 删除 / 取消 buttons
		const confirmBtn = await screen.findByRole("button", { name: "删除" });
		fireEvent.click(confirmBtn);

		await waitFor(() =>
			expect(mockDeleteExecutionTarget).toHaveBeenCalledTimes(1),
		);
		expect(mockDeleteExecutionTarget).toHaveBeenCalledWith(
			"a03ad932-397f-4a3b-a3d8-1cfb13a6dd54",
		);
	});
});

describe("PoolManager — create", () => {
	it("submits create with new toleration", async () => {
		renderPoolManager();
		await waitFor(() =>
			expect(mockListExecutionTargets).toHaveBeenCalledTimes(1),
		);

		const newBtn = await screen.findByRole("button", { name: /新建/ });
		fireEvent.click(newBtn);

		// Fill name / namespace
		const nameInput = await screen.findByPlaceholderText("例如: 客户A生产池");
		fireEvent.change(nameInput, { target: { value: "test-pool" } });
		fireEvent.change(screen.getByPlaceholderText("例如: pool-customer-a"), {
			target: { value: "test-ns" },
		});

		// Add one toleration
		const addTolBtn = screen.getByRole("button", { name: /添加 toleration/ });
		fireEvent.click(addTolBtn);
		const keyInputs = await screen.findAllByPlaceholderText(
			/key\(如 compute-tier\)/,
		);
		fireEvent.change(keyInputs[0], { target: { value: "environment" } });
		const valueInputs =
			await screen.findAllByPlaceholderText(/value\(如 med\)/);
		fireEvent.change(valueInputs[0], { target: { value: "prod" } });

		fireEvent.click(screen.getByRole("button", { name: "创建" }));

		await waitFor(() =>
			expect(mockCreateExecutionTarget).toHaveBeenCalledTimes(1),
		);
		const [body] = mockCreateExecutionTarget.mock.calls[0];
		expect(body.name).toBe("test-pool");
		expect(body.namespace).toBe("test-ns");
		expect(body.resourceDefaults?.templateTolerations?.[0]).toMatchObject({
			key: "environment",
			value: "prod",
			operator: "Equal",
			effect: "NoSchedule",
		});
	});
});
