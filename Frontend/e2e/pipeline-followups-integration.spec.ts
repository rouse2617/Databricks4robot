import { expect, test } from "@playwright/test";

const hasIntegrationEnv = Boolean(
	process.env.E2E_DATABREW_TOKEN || process.env.VITE_DEV_ACCESS_TOKEN,
);

test.describe("pipeline follow-ups integration @integration", () => {
	test.skip(!hasIntegrationEnv, "需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN");

	test.beforeEach(async ({ page }) => {
		await page.goto("/");
		await page.waitForLoadState("networkidle");
	});

	test("pipelines API returns paginated response", async ({ page }) => {
		const body = await page.evaluate(async () => {
			const res = await fetch(
				"/api/v1/pipelines?page=1&page_size=5&sort=updated_at_desc",
			);
			if (!res.ok) {
				throw new Error(`pipelines status ${res.status}`);
			}
			return res.json() as Promise<{
				items: unknown[];
				total: number;
				page: number;
				pageSize: number;
			}>;
		});
		expect(body.page).toBe(1);
		expect(body.pageSize).toBe(5);
		expect(body.total).toBeGreaterThan(0);
		expect(body.items.length).toBeLessThanOrEqual(5);
	});

	test("pipeline management tab shows server-side search controls", async ({
		page,
	}) => {
		await page.goto("/pipeline?tab=pipelines");
		await expect(page.getByTestId("pipeline-template-search")).toBeVisible({
			timeout: 15000,
		});
		await expect(page.getByTestId("pipeline-template-sort")).toBeVisible();
	});

	test("execution list loads without duplicate refresh errors", async ({
		page,
	}) => {
		await page.goto("/pipeline?tab=executions");
		await expect(page.getByRole("heading", { name: "流水线执行记录" })).toBeVisible({
			timeout: 15000,
		});
		const table = page.locator(".ant-table");
		await expect(table).toBeVisible({ timeout: 15000 });
	});
});
