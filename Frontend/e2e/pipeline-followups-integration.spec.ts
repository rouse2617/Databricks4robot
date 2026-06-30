import { expect, test } from "@playwright/test";

const hasIntegrationEnv = Boolean(
	process.env.E2E_DATABREW_TOKEN || process.env.VITE_DEV_ACCESS_TOKEN,
);

test.describe("pipeline follow-ups integration @integration", () => {
	test.skip(
		!hasIntegrationEnv,
		"需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN",
	);

	async function gotoPipelinesTab(page: import("@playwright/test").Page) {
		const token =
			process.env.VITE_DEV_ACCESS_TOKEN || process.env.E2E_DATABREW_TOKEN;
		if (token) {
			await page.request.post("/api/v1/auth/login", {
				data: { token },
			});
		}
		await page.goto("/pipeline?tab=pipelines");
		await expect(page.getByTestId("pipeline-template-search")).toBeVisible({
			timeout: 20000,
		});
	}

	test("pipelines API returns paginated response", async ({ page }) => {
		await gotoPipelinesTab(page);
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

	test("pipelines API respects page_size and scope filters", async ({
		page,
	}) => {
		await gotoPipelinesTab(page);
		const paged = await page.evaluate(async () => {
			const res = await fetch("/api/v1/pipelines?page=1&page_size=10");
			if (!res.ok) throw new Error(`status ${res.status}`);
			return res.json() as Promise<{ items: unknown[]; total: number }>;
		});
		expect(paged.items.length).toBeLessThanOrEqual(10);
		expect(paged.total).toBeGreaterThan(paged.items.length);

		const prodOnly = await page.evaluate(async () => {
			const res = await fetch("/api/v1/pipelines?scope=prod&page_size=200");
			if (!res.ok) throw new Error(`status ${res.status}`);
			return res.json() as Promise<{
				items: Array<{ scope?: string }>;
			}>;
		});
		for (const item of prodOnly.items) {
			expect(item.scope).toBe("prod");
		}
	});

	test("pipelines API search q reduces result set", async ({ page }) => {
		await gotoPipelinesTab(page);
		const baseline = await page.evaluate(async () => {
			const res = await fetch("/api/v1/pipelines?page_size=200");
			if (!res.ok) throw new Error(`status ${res.status}`);
			return res.json() as Promise<{ total: number }>;
		});
		const filtered = await page.evaluate(async () => {
			const res = await fetch("/api/v1/pipelines?q=pipeline&page_size=200");
			if (!res.ok) throw new Error(`status ${res.status}`);
			return res.json() as Promise<{ total: number; items: unknown[] }>;
		});
		expect(filtered.total).toBeGreaterThan(0);
		expect(filtered.total).toBeLessThanOrEqual(baseline.total);
	});

	test("pipeline tab shows total count and load more when needed", async ({
		page,
	}) => {
		await gotoPipelinesTab(page);
		const countBadge = page.locator(".deploy-section-title .count");
		await expect(countBadge).toBeVisible({ timeout: 15000 });
		const total = Number((await countBadge.textContent())?.trim() ?? "0");
		expect(total).toBeGreaterThan(10);
		await expect(page.getByTestId("pipeline-template-load-more")).toBeVisible();
		const section = page.locator(".deploy-panel__section-card").first();
		await expect(section.locator(".dep-card")).toHaveCount(10);
	});

	test("pipeline tab issues single pipelines list request on mount", async ({
		page,
	}) => {
		let pipelinesCalls = 0;
		page.on("request", (request) => {
			if (
				request.method() === "GET" &&
				request.url().includes("/api/v1/pipelines") &&
				!request.url().includes("/versions")
			) {
				pipelinesCalls += 1;
			}
		});
		await gotoPipelinesTab(page);
		expect(pipelinesCalls).toBeLessThanOrEqual(1);
	});

	test("pipeline management tab shows server-side search controls", async ({
		page,
	}) => {
		await gotoPipelinesTab(page);
		await expect(page.getByTestId("pipeline-template-search")).toBeVisible({
			timeout: 15000,
		});
		await expect(page.getByTestId("pipeline-template-sort")).toBeVisible();
	});

	test("execution list loads without duplicate refresh errors", async ({
		page,
	}) => {
		const token =
			process.env.VITE_DEV_ACCESS_TOKEN || process.env.E2E_DATABREW_TOKEN;
		if (token) {
			await page.request.post("/api/v1/auth/login", {
				data: { token },
			});
		}
		await page.goto("/pipeline?tab=executions");
		await expect(
			page.getByRole("heading", { name: "流水线执行记录" }),
		).toBeVisible({
			timeout: 15000,
		});
		const table = page.locator(".ant-table");
		await expect(table).toBeVisible({ timeout: 15000 });
	});
});
