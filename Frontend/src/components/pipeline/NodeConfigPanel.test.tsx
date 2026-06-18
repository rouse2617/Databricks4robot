// @vitest-environment jsdom

import {
	cleanup,
	fireEvent,
	render,
	screen,
	waitFor,
} from "@testing-library/react";
import type { Node } from "@xyflow/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { NodeConfigPanel } from "./NodeConfigPanel";
import type { PipelineNodeData } from "./types";

const mockListRuntimeMounts = vi.fn();
const mockListConfigs = vi.fn();
const mockGetConfig = vi.fn();

vi.mock("../../api/pipelineApi", () => ({
	listRuntimeMounts: (...args: unknown[]) => mockListRuntimeMounts(...args),
}));

vi.mock("../../api/pipelineConfigs", () => ({
	pipelineConfigApi: {
		list: (...args: unknown[]) => mockListConfigs(...args),
		get: (...args: unknown[]) => mockGetConfig(...args),
	},
}));

function mockMatchMedia() {
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
}

function makeNode(): Node<PipelineNodeData> {
	return {
		id: "node-1",
		type: "pipelineStep",
		position: { x: 0, y: 0 },
		data: {
			label: "smoke-task",
			image: "busybox:latest",
			command: ["sh", "-c"],
			args: [{ name: "script", value: "echo ok" }],
			cpu: "500m",
			memory: "512Mi",
			disk: "1Gi",
		},
	};
}

describe("NodeConfigPanel", () => {
	beforeEach(() => {
		mockMatchMedia();
		vi.clearAllMocks();
		mockListConfigs.mockResolvedValue({ items: [] });
		mockGetConfig.mockResolvedValue({ versions: [] });
		mockListRuntimeMounts.mockResolvedValue({
			secrets: [],
			storage: [
				{
					id: "scratch-emptydir",
					name: "Scratch EmptyDir",
					kind: "emptyDir",
					defaultMountPath: "/workspace/scratch",
					readOnly: false,
					allowWrite: true,
				},
			],
		});
	});

	afterEach(cleanup);

	it("adds the default storage mount and saves it into node data", async () => {
		const onSave = vi.fn();
		const onCancel = vi.fn();

		render(
			<NodeConfigPanel
				open
				node={makeNode()}
				onCancel={onCancel}
				onSave={onSave}
			/>,
		);

		await waitFor(() => expect(mockListRuntimeMounts).toHaveBeenCalled());
		fireEvent.click(screen.getByRole("button", { name: /新增存储挂载/ }));

		expect(await screen.findByText(/Scratch EmptyDir/)).toBeTruthy();
		expect(screen.getByLabelText("存储挂载路径")).toHaveValue(
			"/workspace/scratch",
		);
		expect(screen.getByText("读写")).toBeTruthy();

		fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));

		await waitFor(() => {
			expect(onSave).toHaveBeenCalledWith(
				"node-1",
				expect.objectContaining({
					storageMounts: [
						{
							resourceId: "scratch-emptydir",
							mountPath: "/workspace/scratch",
							readOnly: false,
							displayName: "Scratch EmptyDir",
						},
					],
				}),
			);
		});
		expect(onCancel).toHaveBeenCalled();
	});

	it("keeps secret mounts in advanced config and saves selected catalog resource", async () => {
		mockListRuntimeMounts.mockResolvedValue({
			secrets: [
				{
					id: "platform-db-secrets",
					name: "Platform DB secrets",
					kind: "secretProviderClass",
					secretProviderClass: "databrew-platform-db-creds", // pragma: allowlist secret
					defaultMountPath: "/mnt/secrets",
					readOnly: true,
					targetIds: ["default"],
				},
			],
			storage: [],
		});
		const onSave = vi.fn();
		const onCancel = vi.fn();

		render(
			<NodeConfigPanel
				open
				node={makeNode()}
				onCancel={onCancel}
				onSave={onSave}
			/>,
		);

		await waitFor(() => expect(mockListRuntimeMounts).toHaveBeenCalled());
		expect(screen.queryByRole("button", { name: /新增密钥挂载/ })).toBeNull();

		fireEvent.click(screen.getByText("高级配置：密钥挂载"));
		fireEvent.click(screen.getByRole("button", { name: /新增密钥挂载/ }));

		expect(await screen.findByText(/Platform DB secrets/)).toBeTruthy();
		expect(screen.getByLabelText("密钥挂载路径")).toHaveValue("/mnt/secrets");

		fireEvent.click(screen.getByRole("button", { name: /保\s*存/ }));

		await waitFor(() => {
			expect(onSave).toHaveBeenCalledWith(
				"node-1",
				expect.objectContaining({
					runtimeSecrets: [
						{
							resourceId: "platform-db-secrets",
							mountPath: "/mnt/secrets",
							displayName: "Platform DB secrets",
						},
					],
				}),
			);
		});
		expect(onCancel).toHaveBeenCalled();
	});

	it("opens advanced secret config when the node already has a secret binding", async () => {
		mockListRuntimeMounts.mockResolvedValue({
			secrets: [
				{
					id: "platform-db-secrets",
					name: "Platform DB secrets",
					kind: "secretProviderClass",
					secretProviderClass: "databrew-platform-db-creds", // pragma: allowlist secret
					defaultMountPath: "/mnt/secrets",
					readOnly: true,
				},
			],
			storage: [],
		});
		const node = makeNode();
		node.data.runtimeSecrets = [
			{
				resourceId: "platform-db-secrets",
				mountPath: "/mnt/secrets",
				displayName: "Platform DB secrets",
			},
		];

		render(
			<NodeConfigPanel open node={node} onCancel={vi.fn()} onSave={vi.fn()} />,
		);

		await waitFor(() => expect(mockListRuntimeMounts).toHaveBeenCalled());

		expect(screen.getByRole("button", { name: /新增密钥挂载/ })).toBeTruthy();
		expect(screen.getByLabelText("密钥挂载路径")).toHaveValue("/mnt/secrets");
	});
});
