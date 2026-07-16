// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import { unstableSetRender } from "antd";
import { StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
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
	Cluster,
	ElasticQuota,
	ExecutionTarget,
} from "../../api/pipelineApi";
import PoolManager from "./PoolManager";

const mockListExecutionTargets = vi.hoisted(() => vi.fn());
const mockCreateExecutionTarget = vi.hoisted(() => vi.fn());
const mockUpdateExecutionTarget = vi.hoisted(() => vi.fn());
const mockDeleteExecutionTarget = vi.hoisted(() => vi.fn());
const mockListElasticQuotas = vi.hoisted(() => vi.fn());
const mockListClusters = vi.hoisted(() => vi.fn());

// ClusterManager is a sibling admin panel PoolManager renders above its own
// table. It calls useAuth(), which needs an AuthProvider the PoolManager tests
// don't (and shouldn't) set up — so stub it out to isolate PoolManager. It has
// its own test coverage.
vi.mock("./ClusterManager", () => ({
	default: () => null,
}));

vi.mock("../../api/pipelineApi", () => ({
	listExecutionTargets: (...args: unknown[]) =>
		mockListExecutionTargets(...args),
	createExecutionTarget: (...args: unknown[]) =>
		mockCreateExecutionTarget(...args),
	updateExecutionTarget: (...args: unknown[]) =>
		mockUpdateExecutionTarget(...args),
	deleteExecutionTarget: (...args: unknown[]) =>
		mockDeleteExecutionTarget(...args),
	listElasticQuotas: (...args: unknown[]) => mockListElasticQuotas(...args),
	listClusters: (...args: unknown[]) => mockListClusters(...args),
}));

const mockMessage = vi.hoisted(() => ({
	success: vi.fn(),
	error: vi.fn(),
	warning: vi.fn(),
	info: vi.fn(),
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

// PoolManager.fetchData() also calls listClusters(); the cluster picker needs at
// least one entry because clusterId is a required form field.
const defaultCluster: Cluster = {
	id: "cluster-default",
	name: "default",
	displayName: "Default cluster",
	isDefault: true,
	status: "active",
	koordInstalled: true,
};

// antd v5 static methods (Modal.confirm — used by the delete flow) need a React
// 19 render adapter; without it jsdom renders nothing and the confirm dialog
// never appears. The real app wires this up in its entrypoint; register it here
// so the confirm-based tests can drive the dialog.
const antdRoots = new WeakMap<Element | DocumentFragment, Root>();
unstableSetRender((node, container) => {
	let root = antdRoots.get(container);
	if (!root) {
		root = createRoot(container);
		antdRoots.set(container, root);
	}
	root.render(node);
	return async () => {
		await new Promise((resolve) => setTimeout(resolve, 0));
		root?.unmount();
		antdRoots.delete(container);
	};
});

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
	mockListElasticQuotas.mockReset();
	mockListClusters.mockReset();
	mockMessage.success.mockClear();
	mockMessage.error.mockClear();

	mockListExecutionTargets.mockResolvedValue([nonDefaultTarget, defaultTarget]);
	mockCreateExecutionTarget.mockResolvedValue(nonDefaultTarget);
	mockUpdateExecutionTarget.mockResolvedValue(nonDefaultTarget);
	mockDeleteExecutionTarget.mockResolvedValue(undefined);
	mockListElasticQuotas.mockResolvedValue([] as ElasticQuota[]);
	mockListClusters.mockResolvedValue([defaultCluster]);

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
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());
		expect(await screen.findByText(/tol: compute-tier=med/)).toBeDefined();
		expect(screen.getByText(/tol: durability=spot/)).toBeDefined();
	});

	it("distinguishes toleration vs nodeSelector in the scheduling column (CYB-3486)", async () => {
		// Regression: the list column flattened tolerations and nodeSelector into
		// identical key=value capsules, so the SAME key (durability) looked the
		// same whether it was a soft toleration or a hard node pin. Prefixes must
		// split them — tol: vs sel:.
		mockListExecutionTargets.mockResolvedValue([
			{
				id: "mixed-1",
				name: "mixed-pool",
				cluster: "default",
				namespace: "video-proc-prod",
				argoServerConfigured: true,
				status: "available",
				isDefault: false,
				resourceDefaults: {
					templateTolerations: [
						{
							key: "durability",
							operator: "Equal",
							value: "spot",
							effect: "NoSchedule",
						},
					],
					templateNodeSelector: { durability: "standard" },
				},
			},
			defaultTarget,
		]);
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());
		expect(await screen.findByText(/tol: durability=spot/)).toBeDefined();
		expect(screen.getByText(/sel: durability=standard/)).toBeDefined();
	});

	it("prefills tolerations when opening edit modal", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

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
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("edit-video-proc-dev"));

		// Change the first toleration value from "med" to "high"
		const valueInputs =
			await screen.findAllByPlaceholderText(/value\(如 med\)/);
		fireEvent.change(valueInputs[0], { target: { value: "high" } });

		// Click 保存
		const okBtn = screen.getByRole("button", { name: /保\s*存/ });
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
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("edit-video-proc-dev"));

		// Clear first toleration's value
		const valueInputs =
			await screen.findAllByPlaceholderText(/value\(如 med\)/);
		fireEvent.change(valueInputs[0], { target: { value: "" } });

		fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));

		// Wait a tick — validation should fail synchronously; update must NOT fire
		await new Promise((resolve) => setTimeout(resolve, 50));
		expect(mockUpdateExecutionTarget).not.toHaveBeenCalled();
	});

	it("does not render delete/edit buttons for default target", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());
		expect(screen.queryByLabelText("delete-Default Argo target")).toBeNull();
		expect(screen.queryByLabelText("edit-Default Argo target")).toBeNull();
	});
});

describe("PoolManager — delete", () => {
	it("calls deleteExecutionTarget after confirm", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("delete-video-proc-dev"));

		// antd Modal.confirm renders a Modal with 删除 / 取消 buttons
		const confirmBtn = await screen.findByRole("button", { name: /删\s*除/ });
		fireEvent.click(confirmBtn);

		await waitFor(() =>
			expect(mockDeleteExecutionTarget).toHaveBeenCalledTimes(1),
		);
		expect(mockDeleteExecutionTarget).toHaveBeenCalledWith(
			"a03ad932-397f-4a3b-a3d8-1cfb13a6dd54",
		);
	});
});

describe("PoolManager — ElasticQuota panel", () => {
	const quotas: ElasticQuota[] = [
		{
			name: "cyberorigin-delivery-high",
			namespace: "cyber-databrew-dev",
			min: { cpu: "4", memory: "8Gi" },
			max: { cpu: "24", memory: "48Gi" },
			used: { cpu: "6", memory: "12Gi" },
			utilizationPercent: { cpu: 25, memory: 25 },
		},
		{
			name: "cyberorigin-delivery-mid",
			namespace: "cyber-databrew-dev",
			min: { cpu: "2", memory: "4Gi" },
			max: { cpu: "14", memory: "28Gi" },
			used: { cpu: "0", memory: "0" },
			utilizationPercent: { cpu: 0, memory: 0 },
		},
		{
			name: "cyberorigin-delivery-low",
			namespace: "cyber-databrew-dev",
			min: { cpu: "1", memory: "2Gi" },
			max: { cpu: "10", memory: "20Gi" },
			used: { cpu: "0", memory: "0" },
			utilizationPercent: { cpu: 0, memory: 0 },
		},
	];

	it("renders quota rows when ElasticQuotas exist", async () => {
		mockListElasticQuotas.mockResolvedValue(quotas);
		renderPoolManager();
		await waitFor(() => expect(mockListElasticQuotas).toHaveBeenCalled());
		expect(
			await screen.findByText("Koordinator 弹性配额池 (ElasticQuota)"),
		).toBeDefined();
		expect(screen.getByText("cyberorigin-delivery-high")).toBeDefined();
		expect(screen.getByText("cyberorigin-delivery-mid")).toBeDefined();
		expect(screen.getByText("cyberorigin-delivery-low")).toBeDefined();
		// usage triples "used / min / max" render
		expect(screen.getByText("6 / 4 / 24")).toBeDefined();
	});

	it("hides the section entirely when no ElasticQuotas exist", async () => {
		mockListElasticQuotas.mockResolvedValue([]);
		renderPoolManager();
		await waitFor(() => expect(mockListElasticQuotas).toHaveBeenCalled());
		expect(
			screen.queryByText("Koordinator 弹性配额池 (ElasticQuota)"),
		).toBeNull();
	});

	it("hides the section when the API call rejects (未装 Koordinator)", async () => {
		mockListElasticQuotas.mockRejectedValue(new Error("crd missing"));
		renderPoolManager();
		await waitFor(() => expect(mockListElasticQuotas).toHaveBeenCalled());
		expect(
			screen.queryByText("Koordinator 弹性配额池 (ElasticQuota)"),
		).toBeNull();
	});
});

describe("PoolManager — create", () => {
	it("submits create with new toleration", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		const newBtn = await screen.findByRole("button", { name: /新建/ });
		fireEvent.click(newBtn);

		// Fill name / namespace (namespace is an AutoComplete — its placeholder is
		// not a real input attribute, so locate it by its form label instead).
		const nameInput = await screen.findByPlaceholderText("例如: 客户A生产池");
		fireEvent.change(nameInput, { target: { value: "test-pool" } });
		fireEvent.change(screen.getByLabelText("K8s 命名空间"), {
			target: { value: "test-ns" },
		});

		// Node constraints are collapsed by default — expand before editing them.
		fireEvent.click(screen.getByRole("button", { name: /节点约束/ }));

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

		fireEvent.click(screen.getByRole("button", { name: /创\s*建/ }));

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

// CYB-3486 pool.3: scheduler / priorityclass / pod labels / annotations are now
// stored under resource_defaults.scheduling (JSONB). The dedicated top-level
// elastic_quota_name / priority_class_name columns were dropped in #439 — the
// UI must never write them back.
describe("PoolManager — scheduling config (resource_defaults.scheduling)", () => {
	const schedulingTarget: ExecutionTarget = {
		id: "sched-target-1",
		name: "koord-pool",
		cluster: "default",
		namespace: "video-proc-dev",
		argoServerConfigured: true,
		status: "available",
		isDefault: false,
		description: "pool wired to a koord elastic quota",
		resourceDefaults: {
			computeTier: "cpu-low",
			scheduling: {
				schedulerName: "koord-scheduler",
				priorityClassName: "cyber-databrew-prod",
				podLabels: {
					"quota.scheduling.koordinator.sh/name": "cyberorigin-delivery-low",
				},
			},
		},
	};

	beforeEach(() => {
		mockListExecutionTargets.mockResolvedValue([
			schedulingTarget,
			defaultTarget,
		]);
	});

	it("prefills scheduler / priorityclass and splits the EQ label into the pool picker", async () => {
		mockListElasticQuotas.mockResolvedValue([
			{
				name: "cyberorigin-delivery-low",
				namespace: "video-proc-dev",
				min: { cpu: "1", memory: "2Gi" },
				max: { cpu: "10", memory: "20Gi" },
				used: { cpu: "0", memory: "0" },
				utilizationPercent: { cpu: 0, memory: 0 },
			},
		]);
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("edit-koord-pool"));

		const schedulerInput = (await screen.findByPlaceholderText(
			/koord-scheduler/,
		)) as HTMLInputElement;
		expect(schedulerInput.value).toBe("koord-scheduler");

		const prioInput = screen.getByPlaceholderText(
			/cyber-databrew-prod/,
		) as HTMLInputElement;
		expect(prioInput.value).toBe("cyber-databrew-prod");

		// scheduler=koord-scheduler → the Koord pool picker is shown, and it OWNS
		// the EQ label: it must NOT leak into the manual podLabels editor as a row.
		expect(screen.getByText("Koord 资源池(选填)")).toBeDefined();
		expect(
			screen.queryByDisplayValue("quota.scheduling.koordinator.sh/name"),
		).toBeNull();
	});

	it("writes edits into resource_defaults.scheduling and no legacy top-level fields", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("edit-koord-pool"));

		// Edit priorityClass — scheduler stays koord-scheduler, so the EQ picker
		// (and the EQ label it owns) persists across the save.
		const prioInput = await screen.findByPlaceholderText(/cyber-databrew-prod/);
		fireEvent.change(prioInput, {
			target: { value: "cyber-databrew-canary" },
		});

		fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
		await waitFor(() =>
			expect(mockUpdateExecutionTarget).toHaveBeenCalledTimes(1),
		);

		const [id, body] = mockUpdateExecutionTarget.mock.calls[0];
		expect(id).toBe("sched-target-1");
		const scheduling = body.resourceDefaults?.scheduling;
		expect(scheduling?.priorityClassName).toBe("cyber-databrew-canary");
		// Untouched scheduling keys survive the PUT.
		expect(scheduling?.schedulerName).toBe("koord-scheduler");
		// The EQ label — owned by the picker — is folded back into podLabels.
		expect(scheduling?.podLabels).toEqual({
			"quota.scheduling.koordinator.sh/name": "cyberorigin-delivery-low",
		});
		// Sibling resource_defaults keys preserved.
		expect(body.resourceDefaults?.computeTier).toBe("cpu-low");
		// The dropped columns must not reappear at the top level.
		expect("elasticQuotaName" in body).toBe(false);
		expect(body.priorityClassName).toBeUndefined();
	});

	it("emptying scheduler unsets it while other scheduling keys remain", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByLabelText("edit-koord-pool"));
		const schedulerInput =
			await screen.findByPlaceholderText(/koord-scheduler/);
		fireEvent.change(schedulerInput, { target: { value: "" } });

		fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));
		await waitFor(() =>
			expect(mockUpdateExecutionTarget).toHaveBeenCalledTimes(1),
		);

		const [, body] = mockUpdateExecutionTarget.mock.calls[0];
		expect(body.resourceDefaults?.scheduling?.schedulerName).toBeUndefined();
		expect(body.resourceDefaults?.scheduling?.priorityClassName).toBe(
			"cyber-databrew-prod",
		);
	});

	it("adds an arbitrary pod label on create — no hardcoded koord key", async () => {
		mockListExecutionTargets.mockResolvedValue([defaultTarget]);
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByRole("button", { name: /新建/ }));
		fireEvent.change(await screen.findByPlaceholderText("例如: 客户A生产池"), {
			target: { value: "custom-pool" },
		});
		fireEvent.change(screen.getByLabelText("K8s 命名空间"), {
			target: { value: "custom-ns" },
		});
		fireEvent.change(screen.getByPlaceholderText(/koord-scheduler/), {
			target: { value: "my-scheduler" },
		});

		// Admin supplies both the label key and value — nothing is hardcoded.
		fireEvent.click(screen.getByRole("button", { name: /添加 pod label/ }));
		const keyInput = await screen.findByPlaceholderText("label key");
		fireEvent.change(keyInput, { target: { value: "example.com/pool" } });
		const valInput = screen.getByPlaceholderText("label value");
		fireEvent.change(valInput, { target: { value: "gold" } });

		fireEvent.click(screen.getByRole("button", { name: /创\s*建/ }));
		await waitFor(() =>
			expect(mockCreateExecutionTarget).toHaveBeenCalledTimes(1),
		);

		const [body] = mockCreateExecutionTarget.mock.calls[0];
		expect(body.resourceDefaults?.scheduling?.schedulerName).toBe(
			"my-scheduler",
		);
		expect(body.resourceDefaults?.scheduling?.podLabels).toEqual({
			"example.com/pool": "gold",
		});
	});
});

// CYB-3486 pool UX: the "Koord 资源池" picker turns the raw EQ pod-label into a
// cluster-pulled dropdown. It only appears for scheduler=koord-scheduler, and on
// pick it folds the choice into podLabels[EQ key] and aligns the pool namespace
// to the EQ's namespace (koord EQs are namespace-scoped).
describe("PoolManager — Koord 资源池 picker", () => {
	const koordEqs: ElasticQuota[] = [
		{
			name: "cyberorigin-delivery-low",
			namespace: "cyber-delivery-prod",
			min: { cpu: "1", memory: "2Gi" },
			max: { cpu: "10", memory: "20Gi" },
			used: { cpu: "0", memory: "0" },
			utilizationPercent: { cpu: 0, memory: 0 },
		},
	];

	beforeEach(() => {
		mockListExecutionTargets.mockResolvedValue([defaultTarget]);
		mockListElasticQuotas.mockResolvedValue(koordEqs);
	});

	it("stays hidden until the scheduler is koord-scheduler", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByRole("button", { name: /新建/ }));
		expect(screen.queryByText("Koord 资源池(选填)")).toBeNull();

		fireEvent.change(screen.getByPlaceholderText(/koord-scheduler/), {
			target: { value: "koord-scheduler" },
		});
		expect(await screen.findByText("Koord 资源池(选填)")).toBeDefined();
	});

	it("populates the picker with the cluster's ElasticQuotas (name + usage)", async () => {
		renderPoolManager();
		await waitFor(() => expect(mockListExecutionTargets).toHaveBeenCalled());

		fireEvent.click(await screen.findByRole("button", { name: /新建/ }));
		fireEvent.change(screen.getByPlaceholderText(/koord-scheduler/), {
			target: { value: "koord-scheduler" },
		});

		// The picker pulls the selected cluster's EQs and lists each by name with
		// its live usage, so the admin picks a pool instead of typing a raw label.
		fireEvent.mouseDown(screen.getByLabelText("Koord 资源池(选填)"));
		expect(
			await screen.findByRole("option", {
				name: /cyberorigin-delivery-low.*CPU 0%/,
			}),
		).toBeDefined();
	});
});
