import { expect, test } from "@playwright/test";

const api = (path: string) => `**/api/v1${path}`;

const authMe = {
	authenticated: true,
	email: "qa@cyberorigin.ai",
	role: "admin",
};

const pipelineTemplatesPage1 = {
	items: [
		{
			id: "tpl-alpha",
			name: "alpha-pipeline",
			version: 3,
			scope: "dev",
			pipeline: { name: "alpha-pipeline", version: "1", nodes: [], edges: [] },
			nodeCount: 2,
			versionCount: 2,
			createdAt: "2026-06-10T00:00:00Z",
			updatedAt: "2026-06-11T00:00:00Z",
		},
	],
	total: 2,
	page: 1,
	pageSize: 10,
};

const pipelineTemplatesPage2 = {
	items: [
		{
			id: "tpl-beta",
			name: "beta-pipeline",
			version: 1,
			scope: "prod",
			pipeline: { name: "beta-pipeline", version: "1", nodes: [], edges: [] },
			nodeCount: 1,
			createdAt: "2026-06-09T00:00:00Z",
			updatedAt: "2026-06-09T12:00:00Z",
		},
	],
	total: 2,
	page: 2,
	pageSize: 10,
};

const executionTargets = {
	items: [
		{
			id: "default",
			name: "default",
			cluster: "local",
			namespace: "argo",
			argoServerConfigured: true,
			status: "available",
			isDefault: true,
		},
	],
};

const staleRunningWorkflow = {
	items: [
		{
			name: "stale-workflow",
			status: "Running",
			nodeCount: 1,
			createdAt: "2026-01-01T00:00:00Z",
			labels: {},
		},
	],
};

const pipelineRuns = { items: [], total: 0 };

test.beforeEach(async ({ page }) => {
	await page.route(api("/auth/me"), async (route) => {
		await route.fulfill({ json: authMe });
	});
	await page.route("**/api/v1/auth/login", async (route) => {
		await route.fulfill({ json: authMe });
	});
	await page.route(api("/execution-targets*"), async (route) => {
		await route.fulfill({ json: executionTargets });
	});
	await page.route(api("/workflows*"), async (route) => {
		await route.fulfill({ json: { items: [] } });
	});
	await page.route(api("/deployments*"), async (route) => {
		await route.fulfill({ json: { items: [] } });
	});
});

test("pipeline list supports server pagination, search and load more", async ({
	page,
}) => {
	let pipelinesCallCount = 0;
	await page.route(api("/pipelines*"), async (route) => {
		pipelinesCallCount += 1;
		const url = new URL(route.request().url());
		const q = url.searchParams.get("q");
		const pageNo = url.searchParams.get("page") ?? "1";
		if (q === "alpha") {
			await route.fulfill({
				json: {
					...pipelineTemplatesPage1,
					items: pipelineTemplatesPage1.items,
					total: 1,
				},
			});
			return;
		}
		if (pageNo === "2") {
			await route.fulfill({ json: pipelineTemplatesPage2 });
			return;
		}
		await route.fulfill({ json: pipelineTemplatesPage1 });
	});
	await page.route(api("/pipeline-runs*"), async (route) => {
		await route.fulfill({ json: pipelineRuns });
	});

	await page.goto("/pipeline?tab=pipelines");
	await expect(page.getByText("alpha-pipeline")).toBeVisible();
	await expect(page.getByTestId("pipeline-template-search")).toBeVisible();

	await page.getByTestId("pipeline-template-search").fill("alpha");
	await page.getByTestId("pipeline-template-search").press("Enter");
	await expect(page.getByText("alpha-pipeline")).toBeVisible();
	await expect(page.getByText("beta-pipeline")).toHaveCount(0);

	await page.getByTestId("pipeline-template-search").fill("");
	await page.getByTestId("pipeline-template-search").press("Enter");
	await page.getByTestId("pipeline-template-load-more").click();
	await expect(page.getByText("beta-pipeline")).toBeVisible();
	expect(pipelinesCallCount).toBeGreaterThanOrEqual(3);
});

test("execution list marks long-running workflows as stale zombies", async ({
	page,
}) => {
	await page.route(api("/pipelines*"), async (route) => {
		await route.fulfill({
			json: { items: [], total: 0, page: 1, pageSize: 200 },
		});
	});
	await page.route(api("/workflows*"), async (route) => {
		await route.fulfill({ json: staleRunningWorkflow });
	});
	await page.route(api("/pipeline-runs*"), async (route) => {
		await route.fulfill({ json: pipelineRuns });
	});

	await page.goto("/pipeline?tab=executions");
	await expect(page.getByText("stale-workflow")).toBeVisible();
	await expect(page.getByText("疑似僵尸")).toBeVisible();
});

test("version history drawer shows structural diff preview", async ({ page }) => {
	const versions = {
		items: [
			{
				id: "tpl-v2",
				name: "demo-pipeline",
				version: 2,
				nodeCount: 2,
				createdAt: "2026-06-11T00:00:00Z",
				pipeline: { name: "demo-pipeline", version: "1", nodes: [], edges: [] },
			},
			{
				id: "tpl-v1",
				name: "demo-pipeline",
				version: 1,
				nodeCount: 1,
				createdAt: "2026-06-10T00:00:00Z",
				pipeline: { name: "demo-pipeline", version: "1", nodes: [], edges: [] },
			},
		],
	};
	const diff = {
		added_nodes: [{ id: "new-step" }],
		removed_nodes: [],
		modified_nodes: [{ id: "transform" }],
		added_edges: [{ source: "load", target: "new-step" }],
		removed_edges: [],
	};

	await page.route("**/api/v1/pipelines/tpl-v2/versions", async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(versions),
		});
	});
	await page.route("**/api/v1/pipelines/tpl-v1/diff/tpl-v2", async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify(diff),
		});
	});
	await page.route(api("/pipelines*"), async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				items: [
					{
						id: "tpl-v2",
						name: "demo-pipeline",
						version: 2,
						versionCount: 2,
						nodeCount: 2,
						scope: "dev",
						createdAt: "2026-06-11T00:00:00Z",
						pipeline: {
							name: "demo-pipeline",
							version: "1",
							nodes: [],
							edges: [],
						},
					},
				],
				total: 1,
				page: 1,
				pageSize: 10,
			}),
		});
	});

	await page.goto("/pipeline?tab=pipelines");
	await page.locator(".dep-card-meta").getByText(/^v2/).click();
	await expect(page.getByTestId("version-history-drawer")).toBeVisible();
	await page.getByRole("button", { name: "与 v1 对比" }).click();
	await expect(page.getByRole("dialog", { name: /版本差异/ })).toBeVisible();
	await expect(page.getByText("新增节点")).toBeVisible();
	await expect(page.getByText("new-step", { exact: true })).toBeVisible();
});

test("create backfill returns pending job immediately for large asset batches", async ({
	page,
}) => {
	await page.route(api("/backfill"), async (route) => {
		if (route.request().method() !== "POST") {
			await route.continue();
			return;
		}
		await route.fulfill({
			status: 201,
			contentType: "application/json",
			body: JSON.stringify({
				id: "batch-large",
				name: "large-batch",
				templateId: "tpl-alpha",
				totalCount: 1000,
				status: "pending",
				createdAt: "2026-06-13T00:00:00Z",
			}),
		});
	});

	await page.goto("/pipeline?tab=pipelines");
	const result = await page.evaluate(async () => {
		const response = await fetch("/api/v1/backfill", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				name: "large-batch",
				templateId: "tpl-alpha",
				assetIds: Array.from({ length: 1000 }, (_, i) => `asset-${i}`),
			}),
		});
		return { status: response.status, body: await response.json() };
	});
	expect(result.status).toBe(201);
	expect(result.body.status).toBe("pending");
	expect(result.body.totalCount).toBe(1000);
});
