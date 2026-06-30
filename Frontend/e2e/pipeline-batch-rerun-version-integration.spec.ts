import { expect, test } from "@playwright/test";

const hasIntegrationEnv = Boolean(
	process.env.E2E_DATABREW_TOKEN || process.env.VITE_DEV_ACCESS_TOKEN,
);

test.describe("批次重跑选版本 integration @integration", () => {
	test.skip(
		!hasIntegrationEnv,
		"需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN",
	);

	async function login(page: import("@playwright/test").Page) {
		const token =
			process.env.VITE_DEV_ACCESS_TOKEN || process.env.E2E_DATABREW_TOKEN;
		if (token) {
			await page.request.post("/api/v1/auth/login", {
				data: { token },
			});
		}
	}

	test("批次详情重跑弹窗可选择模板版本并 dry-run 预览", async ({ page }) => {
		test.setTimeout(90_000);
		await login(page);

		const batchesRes = await page.request.get(
			"/api/v1/backfill?page=1&page_size=50",
		);
		expect(batchesRes.ok()).toBeTruthy();
		const batchesBody = (await batchesRes.json()) as {
			items: Array<{
				id: string;
				templateId: string;
				templateVersion?: number;
				totalCount: number;
			}>;
		};

		let target: (typeof batchesBody.items)[number] | undefined;
		let versions: number[] = [];
		for (const job of batchesBody.items ?? []) {
			const versionsRes = await page.request.get(
				`/api/v1/pipelines/${job.templateId}/versions`,
			);
			if (!versionsRes.ok()) continue;
			const versionsBody = (await versionsRes.json()) as {
				items: Array<{ version: number }>;
			};
			const nums = (versionsBody.items ?? [])
				.map((item) => item.version)
				.filter((v) => v > 0);
			if (nums.length >= 2 && job.totalCount >= 1 && job.totalCount <= 200) {
				target = job;
				versions = nums;
				break;
			}
		}
		test.skip(!target, "环境中无带多版本的合适批量任务");

		const jobRes = await page.request.get(`/api/v1/backfill/${target.id}`);
		expect(jobRes.ok()).toBeTruthy();
		const jobDetail = (await jobRes.json()) as {
			job: { failedCount: number; completedCount: number };
		};
		const rerunMenuLabel =
			jobDetail.job.failedCount > 0 ? "重试全部失败" : "重试已完成";

		const currentVersion =
			target.templateVersion && target.templateVersion > 0
				? target.templateVersion
				: Math.max(...versions);
		const altVersion =
			versions.find((v) => v !== currentVersion) ?? versions[0];

		await page.goto(`/pipeline/batch/${target.id}`);
		await expect(page.getByText("子任务执行记录")).toBeVisible({
			timeout: 20000,
		});

		await page.getByRole("button", { name: "重跑" }).click();
		await page.getByText(rerunMenuLabel).click();

		const rerunDialog = page.getByRole("dialog", { name: "确认重跑" });
		await expect(rerunDialog).toBeVisible({ timeout: 15000 });
		await expect(
			rerunDialog.getByRole("combobox", { name: "重跑模板版本" }),
		).toBeVisible();
		await expect(
			rerunDialog.getByText(new RegExp(`模板版本 v${currentVersion}`)),
		).toBeVisible({ timeout: 15000 });

		const dryRunPromise = page.waitForResponse(async (res) => {
			if (
				!res.url().includes(`/api/v1/backfill/${target?.id}/rerun`) ||
				res.request().method() !== "POST"
			) {
				return false;
			}
			const body = res.request().postDataJSON() as {
				templateVersion?: number;
				dryRun?: boolean;
			} | null;
			return body?.dryRun === true && body?.templateVersion === altVersion;
		});

		await rerunDialog.locator(".ant-select").click();
		await page.getByTitle(`v${altVersion}`).click();

		const dryRunRes = await dryRunPromise;
		expect(dryRunRes.ok()).toBeTruthy();
		const dryRunBody = (await dryRunRes.json()) as {
			dryRun?: boolean;
			templateVersion?: number;
			matchedCount?: number;
		};
		expect(dryRunBody.dryRun).toBe(true);
		expect(dryRunBody.templateVersion).toBe(altVersion);
		expect(dryRunBody.matchedCount).toBeGreaterThan(0);

		await expect(
			rerunDialog.getByText(
				new RegExp(`模板版本 v${altVersion}。旧 run 记录会保留。`),
			),
		).toBeVisible();

		await rerunDialog.getByRole("button", { name: "取 消" }).click();
		await expect(rerunDialog).toHaveCount(0);
	});
});
