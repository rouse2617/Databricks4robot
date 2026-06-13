import { expect, test } from "@playwright/test";

const hasIntegrationEnv = Boolean(
	process.env.E2E_DATABREW_TOKEN || process.env.VITE_DEV_ACCESS_TOKEN,
);

test.describe("批量任务 integration @integration", () => {
	test.skip(!hasIntegrationEnv, "需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN");

	test.beforeEach(async ({ page }) => {
		await page.goto("/");
		await page.waitForLoadState("networkidle");
	});

	test("批次详情 reconcile 后子任务数与 totalCount 一致", async ({ page }) => {
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
			batchesBody.items?.find((job) => job.totalCount >= 1) ??
			batchesBody.items?.[0];
		test.skip(!target, "环境中无批量任务");

		const runsRes = await page.request.get(
			`/api/v1/pipeline-runs?view=summary&batchJobId=${encodeURIComponent(target.id)}&page=1&pageSize=50`,
		);
		expect(runsRes.ok()).toBeTruthy();
		const runsBody = (await runsRes.json()) as { items: unknown[]; total: number };
		expect(runsBody.total).toBeGreaterThanOrEqual(target.totalCount);
		expect(runsBody.items.length).toBeGreaterThanOrEqual(target.totalCount);

		await page.goto(`/pipeline/batch/${target.id}`);
		await expect(page.getByText("子任务执行记录")).toBeVisible({ timeout: 15000 });
		await expect(page.getByText("暂无执行记录，部署流水线后将自动生成")).toHaveCount(0);
		await expect(page.locator(".ant-table-row").first()).toBeVisible({
			timeout: 15000,
		});
	});
});
