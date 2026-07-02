import { expect, test } from "@playwright/test";

const api = (path: string) => `**/api/v1${path}`;

const authMe = {
	authenticated: true,
	email: "qa@cyberorigin.ai",
	role: "admin",
};

const batchJob = {
	id: "batch-job-001",
	name: "manual-batch",
	templateId: "tpl-111",
	templateVersion: 4,
	totalCount: 2,
	completedCount: 1,
	failedCount: 1,
	pilotCount: 0,
	pilotPhase: "none",
	status: "running",
	createdAt: "2026-06-10T02:00:00Z",
	updatedAt: "2026-06-10T02:05:00Z",
};

const nodeSummary = {
	batchJobId: "batch-job-001",
	templateId: "tpl-111",
	templateVersion: 4,
	subtasks: {
		total: 2,
		completed: 1,
		failed: 1,
		running: 0,
		pending: 0,
		paused: false,
	},
	nodes: [
		{
			pipelineNodeId: "step-extract",
			displayName: "抽特征",
			dagOrder: 1,
			counts: {
				Pending: 0,
				Running: 0,
				Succeeded: 1,
				Failed: 1,
				Error: 0,
				Skipped: 0,
				Omitted: 0,
			},
			attempted: 2,
			failureRate: 0.5,
		},
	],
	dataCoverage: {
		runsWithNodeRows: 2,
		runsTotal: 2,
		complete: true,
	},
	generatedAt: "2026-06-10T02:05:00Z",
};

const nodeFailures = {
	items: [
		{
			backfillItemId: "item-2",
			assetId: "asset-b",
			runId: "run-child-2",
			workflowName: "demo-pipeline-def456",
			pipelineNodeId: "step-extract",
			displayName: "抽特征",
			status: "Failed",
			message: "exit code 1",
			startedAt: "2026-06-10T02:01:00Z",
			finishedAt: "2026-06-10T02:01:05Z",
		},
	],
	total: 1,
	page: 1,
	pageSize: 20,
};

const batchSubRuns = {
	items: [
		{
			id: "run-child-1",
			templateId: "tpl-111",
			pipelineName: "demo-pipeline",
			workflowName: "demo-pipeline-abc123",
			status: "Succeeded",
			nodeCount: 1,
			templateVersion: 4,
			scope: "dev",
			assetIds: ["asset-a"],
			createdAt: "2026-06-10T02:00:10Z",
		},
		{
			id: "run-child-2",
			templateId: "tpl-111",
			pipelineName: "demo-pipeline",
			workflowName: "demo-pipeline-def456",
			status: "Failed",
			nodeCount: 1,
			templateVersion: 4,
			scope: "dev",
			assetIds: ["asset-b"],
			createdAt: "2026-06-10T02:00:20Z",
		},
	],
	total: 2,
	page: 1,
	pageSize: 20,
};

const templates = {
	items: [
		{
			id: "tpl-111",
			name: "demo-pipeline",
			version: 4,
			pipeline: { name: "demo-pipeline", nodes: [], edges: [] },
			nodeCount: 1,
			scope: "dev",
			createdAt: "2026-06-10T00:00:00Z",
		},
	],
	total: 1,
	page: 1,
	pageSize: 20,
};

const templateVersions = {
	items: [
		{
			id: "tpl-111",
			name: "demo-pipeline",
			version: 3,
			pipeline: { name: "demo-pipeline", nodes: [], edges: [] },
			nodeCount: 1,
			scope: "dev",
			createdAt: "2026-06-09T00:00:00Z",
		},
		{
			id: "tpl-111",
			name: "demo-pipeline",
			version: 4,
			pipeline: { name: "demo-pipeline", nodes: [], edges: [] },
			nodeCount: 1,
			scope: "dev",
			createdAt: "2026-06-10T00:00:00Z",
		},
	],
};

async function mockShell(page: import("@playwright/test").Page) {
	await page.route(api("/auth/me"), (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(authMe),
		}),
	);
	await page.route("**/api/v1/auth/login", (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(authMe),
		}),
	);
	await page.route(api("/pipelines/tpl-111/versions"), (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(templateVersions),
		}),
	);
	await page.route(api("/pipelines*"), (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(templates),
		}),
	);
}

test.describe("batch node summary", () => {
	test.beforeEach(async ({ page }) => {
		await mockShell(page);
		await page.route(api("/backfill/batch-job-001"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({ job: batchJob }),
			}),
		);
		await page.route(api("/backfill/batch-job-001/node-summary"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(nodeSummary),
			}),
		);
		await page.route(api("/backfill/batch-job-001/node-failures*"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(nodeFailures),
			}),
		);
		await page.route(
			api("/pipeline-runs?view=summary&batchJobId=batch-job-001*"),
			(route) =>
				route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify(batchSubRuns),
				}),
		);
	});

	test("shows node summary and drills into failures", async ({ page }) => {
		await page.goto("/pipeline/batch/batch-job-001");

		await expect(page.getByText("节点概览", { exact: true })).toBeVisible();
		await expect(page.getByRole("cell", { name: "1. 抽特征" })).toBeVisible();
		await page.getByRole("button", { name: "1 失败" }).click();
		await expect(
			page.getByRole("cell", { name: "asset-b", exact: true }),
		).toBeVisible();
		await expect(page.getByText("exit code 1")).toBeVisible();
	});

	test("runs dry-run for selected rerun and allows cancel", async ({
		page,
	}) => {
		const rerunBodies: Array<Record<string, unknown>> = [];
		await page.route(api("/backfill/batch-job-001/rerun"), async (route) => {
			rerunBodies.push(
				route.request().postDataJSON() as Record<string, unknown>,
			);
			const body = route.request().postDataJSON() as Record<string, unknown>;
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					status: "accepted",
					dryRun: true,
					matchedCount: 1,
					templateId: "tpl-111",
					templateVersion: body.templateVersion ?? 4,
					skipped: [],
				}),
			});
		});

		await page.goto("/pipeline/batch/batch-job-001");

		await page
			.locator('tr[data-row-key="demo-pipeline-abc123"] .ant-checkbox-input')
			.setChecked(true, { force: true });
		await page.getByRole("button", { name: "重跑" }).click();
		await page.getByText("重试选中项 (1)").click();
		await expect(page.getByRole("dialog", { name: "确认重跑" })).toBeVisible();
		await expect(
			page.getByText("将重新提交 1 条子任务，模板版本 v4。旧 run 记录会保留。"),
		).toBeVisible();
		await page.getByRole("button", { name: "取 消" }).click();

		await expect.poll(() => rerunBodies.length).toBe(1);
		expect(rerunBodies[0].dryRun).toBe(true);
		expect(rerunBodies[0].scope).toBe("custom");
		expect(rerunBodies[0].assetIds).toEqual(["asset-a"]);
		expect(rerunBodies[0].templateVersion).toBe(4);
	});

	test("lets user pick template version before rerun", async ({ page }) => {
		const rerunBodies: Array<Record<string, unknown>> = [];
		await page.route(api("/backfill/batch-job-001/rerun"), async (route) => {
			rerunBodies.push(
				route.request().postDataJSON() as Record<string, unknown>,
			);
			const body = route.request().postDataJSON() as Record<string, unknown>;
			const isDryRun = Boolean(body.dryRun);
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					status: "accepted",
					dryRun: isDryRun,
					matchedCount: body.templateVersion === 3 ? 2 : 1,
					templateId: "tpl-111",
					templateVersion: body.templateVersion ?? 4,
					skipped: [],
				}),
			});
		});

		await page.goto("/pipeline/batch/batch-job-001");

		await page.getByRole("button", { name: "重跑" }).click();
		await page.getByText("重试全部失败").click();
		const rerunDialog = page.getByRole("dialog", { name: "确认重跑" });
		await expect(rerunDialog).toBeVisible();
		await expect(
			rerunDialog.getByText(
				"将重新提交 1 条子任务，模板版本 v4。旧 run 记录会保留。",
			),
		).toBeVisible();

		await rerunDialog.locator(".ant-select").click();
		await page.getByTitle("v3").click();
		await expect(
			rerunDialog.getByText(
				"将重新提交 2 条子任务，模板版本 v3。旧 run 记录会保留。",
			),
		).toBeVisible();

		await expect.poll(() => rerunBodies.length).toBe(2);
		expect(rerunBodies[1].dryRun).toBe(true);
		expect(rerunBodies[1].templateVersion).toBe(3);

		await page.getByRole("button", { name: "确认重跑" }).click();
		await expect.poll(() => rerunBodies.length).toBe(3);
		expect(rerunBodies[2].dryRun).toBeUndefined();
		expect(rerunBodies[2].templateVersion).toBe(3);
		expect(rerunBodies[2].scope).toBe("failed");
	});
});
