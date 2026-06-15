import { expect, test } from "@playwright/test";

const hasIntegrationEnv = Boolean(
	process.env.E2E_DATABREW_TOKEN || process.env.VITE_DEV_ACCESS_TOKEN,
);

test.describe("批量任务 integration @integration", () => {
	test.skip(!hasIntegrationEnv, "需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN");

	async function login(page: import("@playwright/test").Page) {
		const token =
			process.env.VITE_DEV_ACCESS_TOKEN || process.env.E2E_DATABREW_TOKEN;
		if (token) {
			await page.request.post("/api/v1/auth/login", {
				data: { token },
			});
		}
	}

	test.beforeEach(async ({ page }) => {
		await login(page);
		await page.goto("/");
		await page.waitForLoadState("networkidle");
	});

	test("批次详情 reconcile 后子任务数与 totalCount 一致", async ({ page }) => {
		test.setTimeout(60_000);
		const batchesRes = await page.request.get("/api/v1/backfill");
		expect(batchesRes.ok()).toBeTruthy();
		const batchesBody = (await batchesRes.json()) as {
			items: Array<{
				id: string;
				name: string;
				totalCount: number;
			}>;
		};
		const target =
			batchesBody.items?.find(
				(job) => job.totalCount >= 1 && job.totalCount <= 100,
			) ??
			batchesBody.items?.find((job) => job.totalCount >= 1) ??
			batchesBody.items?.[0];
		test.skip(!target, "环境中无批量任务");

		const runsRes = await page.request.get(
			`/api/v1/pipeline-runs?view=summary&batchJobId=${encodeURIComponent(target.id)}&page=1&pageSize=${target.totalCount}`,
		);
		expect(runsRes.ok()).toBeTruthy();
		const runsBody = (await runsRes.json()) as { items: unknown[]; total: number };
		expect(runsBody.total).toBeGreaterThanOrEqual(target.totalCount);
		expect(runsBody.items.length).toBeGreaterThanOrEqual(
			Math.min(target.totalCount, 50),
		);

		await page.goto(`/pipeline/batch/${target.id}`);
		await expect(page.getByText("子任务执行记录")).toBeVisible({ timeout: 15000 });
		await expect(page.getByText("暂无执行记录，部署流水线后将自动生成")).toHaveCount(0);
		await expect(page.locator(".ant-table-row").first()).toBeVisible({
			timeout: 15000,
		});
	});

	test("1000 资产创建批次立即返回 pending", async ({ page }) => {
		const templatesRes = await page.request.get(
			"/api/v1/pipelines?page=1&page_size=1",
		);
		expect(templatesRes.ok()).toBeTruthy();
		const templatesBody = (await templatesRes.json()) as {
			items: Array<{ id: string; name: string }>;
		};
		const template = templatesBody.items?.[0];
		test.skip(!template, "环境中无流水线模板");

		const assetIds = Array.from({ length: 1000 }, (_, i) => `e2e-asset-${i}`);
		const started = Date.now();
		const createRes = await page.request.post("/api/v1/backfill", {
			data: {
				name: `e2e-batch-1000-${Date.now()}`,
				templateId: template.id,
				assetIds,
			},
		});
		const elapsedMs = Date.now() - started;
		expect(createRes.status()).toBe(201);
		const body = (await createRes.json()) as {
			id: string;
			status: string;
			totalCount: number;
		};
		expect(body.status).toBe("pending");
		expect(body.totalCount).toBe(1000);
		expect(elapsedMs).toBeLessThan(5000);

		const jobRes = await page.request.get(`/api/v1/backfill/${body.id}`);
		expect(jobRes.ok()).toBeTruthy();
		const jobBody = (await jobRes.json()) as {
			job: { status: string; totalCount: number };
		};
		expect(jobBody.job.totalCount).toBe(1000);
	});

	test("超过 10000 资产创建批次返回 400", async ({ page }) => {
		const templatesRes = await page.request.get(
			"/api/v1/pipelines?page=1&page_size=1",
		);
		expect(templatesRes.ok()).toBeTruthy();
		const templatesBody = (await templatesRes.json()) as {
			items: Array<{ id: string }>;
		};
		const template = templatesBody.items?.[0];
		test.skip(!template, "环境中无流水线模板");

		const assetIds = Array.from({ length: 10_001 }, (_, i) => `over-${i}`);
		const createRes = await page.request.post("/api/v1/backfill", {
			data: {
				name: "e2e-over-limit",
				templateId: template.id,
				assetIds,
			},
		});
		expect(createRes.status()).toBe(400);
		const body = (await createRes.json()) as { message?: string };
		expect(body.message ?? "").toMatch(/too many assets|max 10000/i);
	});

	test("运行弹窗批量粘贴 1000 资产并跳转批次详情", async ({ page }) => {
		const templatesRes = await page.request.get(
			"/api/v1/pipelines?page=1&page_size=1&scope=dev",
		);
		expect(templatesRes.ok()).toBeTruthy();
		const templatesBody = (await templatesRes.json()) as {
			items: Array<{ id: string; name: string }>;
		};
		const template = templatesBody.items?.[0];
		test.skip(!template, "环境中无 dev 流水线模板");

		await page.goto("/pipeline?tab=pipelines");
		await expect(page.getByText(template.name)).toBeVisible({ timeout: 20000 });

		const assetIds = Array.from({ length: 1000 }, (_, i) => `ui-e2e-${i}`);
		await page.getByRole("button", { name: "运行" }).first().click();
		const modal = page.getByRole("dialog", { name: "运行流水线" });
		await expect(modal).toBeVisible();
		await modal
			.getByText("批量粘贴 asset ID（换行 / 逗号 / 分号分隔）")
			.click();
		await modal
			.getByPlaceholder(/单次批量最多/)
			.fill(assetIds.join("\n"));
		await modal.getByPlaceholder(/单次批量最多/).blur();
		await expect(modal.getByText("将创建批量任务，共 1000 个子任务")).toBeVisible();
		await modal.getByRole("button", { name: "运行资产" }).click();

		await expect(page).toHaveURL(/\/pipeline\/batch\//, { timeout: 20000 });
		await expect(page.getByText("子任务执行记录")).toBeVisible({ timeout: 15000 });
		await expect(page.getByText("1000", { exact: true }).first()).toBeVisible();
	});
});
