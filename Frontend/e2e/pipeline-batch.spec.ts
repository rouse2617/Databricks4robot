import { expect, test } from "@playwright/test";

const api = (path: string) => `**/api/v1${path}`;

const authMe = {
	authenticated: true,
	email: "qa@cyberorigin.ai",
	role: "admin",
};

const batchJobs = {
	items: [
		{
			id: "batch-job-001",
			name: "manual-batch",
			templateId: "tpl-111",
			totalCount: 2,
			completedCount: 1,
			failedCount: 0,
			status: "running",
			createdAt: "2026-06-10T02:00:00Z",
			updatedAt: "2026-06-10T02:05:00Z",
		},
	],
};

const batchDetail = {
	job: batchJobs.items[0],
};

const batchSubRuns = {
	items: [
		{
			id: "run-child-1",
			pipelineName: "demo-pipeline",
			workflowName: "demo-pipeline-abc123",
			status: "Succeeded",
			nodeCount: 1,
			templateVersion: 2,
			createdAt: "2026-06-10T02:00:10Z",
		},
		{
			id: "run-child-2",
			pipelineName: "demo-pipeline",
			workflowName: "demo-pipeline-def456",
			status: "Running",
			nodeCount: 1,
			templateVersion: 2,
			createdAt: "2026-06-10T02:00:20Z",
		},
	],
	total: 2,
	page: 1,
	pageSize: 20,
};

const singleRuns = {
	items: [
		{
			id: "run-single-1",
			pipelineName: "single-pipeline",
			workflowName: "single-run",
			status: "Succeeded",
			nodeCount: 1,
			createdAt: "2026-06-10T01:00:00Z",
		},
	],
	total: 1,
};

const demoTemplate = {
	id: "tpl-111",
	name: "demo-pipeline",
	version: 2,
	pipeline: { name: "demo-pipeline", version: "1", nodes: [], edges: [] },
	nodeCount: 1,
	createdAt: "2026-06-10T00:00:00Z",
};

const executionTargets = {
	items: [
		{
			id: "default",
			name: "Default Argo target",
			cluster: "default",
			namespace: "cyber-databrew-dev",
			argoServerConfigured: true,
			status: "available",
			isDefault: true,
		},
	],
};

async function mockPipelineShell(page: import("@playwright/test").Page) {
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
	await page.route(api("/workflows*"), (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({ items: [] }),
		}),
	);
	await page.route(api("/execution-targets*"), (route) =>
		route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(executionTargets),
		}),
	);
}

test.describe("流水线批量任务", () => {
	test.beforeEach(async ({ page }) => {
		await mockPipelineShell(page);
	});

	test("执行记录可切换批量任务并进入子任务详情", async ({ page }) => {
		await page.route(
			api("/pipeline-runs?view=summary&excludeBatch=true*"),
			(route) =>
				route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify(singleRuns),
				}),
		);
		await page.route(api("/backfill"), (route) => {
			if (route.request().method() === "GET") {
				return route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify(batchJobs),
				});
			}
			return route.continue();
		});
		await page.route(api("/backfill/batch-job-001"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify(batchDetail),
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

		await page.goto("/pipeline?tab=executions");

		await page.getByText("批量任务", { exact: true }).click();
		await expect(page.getByText("manual-batch")).toBeVisible();

		await page.getByRole("button", { name: "查看子任务" }).click();
		await expect(page).toHaveURL(/\/pipeline\/batch\/batch-job-001/);
		await expect(page.getByText("demo-pipeline-abc123")).toBeVisible();
		await expect(page.getByText("demo-pipeline-def456")).toBeVisible();
	});

	test("手动输入未注册 asset ID 可创建批量任务", async ({ page }) => {
		let createBody: Record<string, unknown> | null = null;

		await page.route(api("/pipelines*"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					items: [demoTemplate],
					total: 1,
					page: 1,
					pageSize: 20,
				}),
			}),
		);
		await page.route(api("/pipelines/tpl-111/versions*"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({ items: [demoTemplate] }),
			}),
		);
		await page.route(api("/backfill"), (route) => {
			if (route.request().method() === "POST") {
				createBody = route.request().postDataJSON() as Record<string, unknown>;
				return route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify({
						id: "batch-manual-001",
						name: createBody.name,
						templateId: "tpl-111",
						totalCount: 2,
						completedCount: 0,
						failedCount: 0,
						status: "running",
						createdAt: "2026-06-10T02:00:00Z",
						updatedAt: "2026-06-10T02:00:00Z",
					}),
				});
			}
			return route.continue();
		});
		await page.route(api("/backfill/batch-manual-001"), (route) =>
			route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					job: {
						id: "batch-manual-001",
						name: "manual-batch",
						templateId: "tpl-111",
						totalCount: 2,
						completedCount: 0,
						failedCount: 0,
						status: "running",
						createdAt: "2026-06-10T02:00:00Z",
						updatedAt: "2026-06-10T02:00:00Z",
					},
				}),
			}),
		);
		await page.route(
			api("/pipeline-runs?view=summary&batchJobId=batch-manual-001*"),
			(route) =>
				route.fulfill({
					status: 200,
					contentType: "application/json",
					body: JSON.stringify({ items: [], total: 0, page: 1, pageSize: 20 }),
				}),
		);

		await page.goto("/pipeline?tab=pipelines");
		await page.getByRole("button", { name: "运行" }).click();
		await expect(page.getByText("运行流水线")).toBeVisible();

		const searchInput = page.getByPlaceholder(
			"搜索资产（输入 asset_id 或名称）",
		);
		await searchInput.fill("custom-asset-a");
		await page
			.getByRole("button", { name: /添加「custom-asset-a」为资产 ID/ })
			.click();

		await searchInput.fill("custom-asset-b");
		await page
			.getByRole("button", { name: /添加「custom-asset-b」为资产 ID/ })
			.click();

		const modal = page.getByRole("dialog", { name: "运行流水线" });
		await expect(
			modal.getByText("将创建批量任务，共 2 个子任务"),
		).toBeVisible();

		await modal.getByRole("button", { name: "运行资产" }).click();

		await expect(page).toHaveURL(/\/pipeline\/batch\/batch-manual-001/);
		expect(createBody).toMatchObject({
			templateId: "tpl-111",
			assetIds: ["custom-asset-a", "custom-asset-b"],
		});
	});
});
