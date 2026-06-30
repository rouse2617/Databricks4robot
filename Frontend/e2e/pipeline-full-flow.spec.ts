import { expect, type Page, type Route, test } from "@playwright/test";

/**
 * 全流程 E2E：流水线设计 → 保存 → 部署 → 批量下发 → 批次详情一致性。
 *
 * 覆盖 CYB-2007 关注的「批量下发后子任务数收敛到逻辑总数」链路：
 *   design tab 载入示例 → 保存为模板 → 打开部署对话框 → 选 ≥2 资产
 *   → handleDeploy 走 createBatchJob(POST /backfill) → 跳转 /pipeline/batch/:id
 *   → 批次详情子任务数 == 提交资产数。
 *
 * 两个 describe：
 *  1. 「mock」确定性 happy-path —— 不依赖后端，CI 可跑（route 拦截全部 API）。
 *  2. 「@integration」—— 真连 dev 后端，需 token，无 token 自动 skip。
 *
 * 运行：
 *   cd Frontend
 *   npx playwright install --with-deps chromium
 *   npx playwright test pipeline-full-flow            # 仅 mock
 *   E2E_DATABREW_TOKEN=xxx npx playwright test pipeline-full-flow   # 含 integration
 */

const api = (path: string) => `**/api/v1${path}`;

const authMe = {
	authenticated: true,
	email: "qa@cyberorigin.ai",
	role: "admin",
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

const SAVED_TEMPLATE_ID = "tpl-e2e-fullflow-001";
const BATCH_ID = "batch-e2e-fullflow-001";
const PIPELINE_NAME = "e2e-fullflow-pipeline";
const ASSET_IDS = ["e2e-flow-asset-1", "e2e-flow-asset-2"];

// Minimal runnable pipeline (1 node, no edges) imported via the design-tab
// "导入" modal — a deterministic substitute for canvas drag-and-drop.
const IMPORT_PIPELINE = {
	name: PIPELINE_NAME,
	version: "1",
	nodes: [
		{
			id: "step-1",
			component: {
				name: "echo",
				image: "busybox:latest",
				type: "container",
				source: "custom",
				command: ["sh", "-c"],
				args: [{ name: "script", value: "echo hi" }],
			},
		},
	],
	edges: [],
};

function savedTemplate(name: string) {
	return {
		id: SAVED_TEMPLATE_ID,
		name,
		version: 1,
		versionCount: 1,
		activeVersion: 1,
		scope: "dev",
		// Return the real pipeline so post-save reload (?templateId=) keeps the canvas.
		pipeline: { ...IMPORT_PIPELINE, name },
		nodeCount: IMPORT_PIPELINE.nodes.length,
		createdAt: "2026-06-15T00:00:00Z",
		updatedAt: "2026-06-15T00:00:00Z",
	};
}

function batchJob(overrides: Record<string, unknown> = {}) {
	return {
		id: BATCH_ID,
		name: "e2e-flow-batch",
		templateId: SAVED_TEMPLATE_ID,
		templateVersion: 1,
		totalCount: ASSET_IDS.length,
		completedCount: 0,
		failedCount: 0,
		status: "pending",
		createdAt: "2026-06-15T00:00:00Z",
		updatedAt: "2026-06-15T00:00:00Z",
		...overrides,
	};
}

const nodeSummary = {
	batchJobId: BATCH_ID,
	templateId: SAVED_TEMPLATE_ID,
	templateVersion: 1,
	subtasks: {
		total: ASSET_IDS.length,
		completed: 0,
		failed: 0,
		running: 0,
		pending: ASSET_IDS.length,
		paused: false,
	},
	nodes: [],
	dataCoverage: {
		runsWithNodeRows: 0,
		runsTotal: ASSET_IDS.length,
		complete: false,
	},
	generatedAt: "2026-06-15T00:00:00Z",
};

const batchSubRuns = {
	items: ASSET_IDS.map((assetId, i) => ({
		id: `run-${i + 1}`,
		pipelineName: PIPELINE_NAME,
		workflowName: `${PIPELINE_NAME}-${i + 1}`,
		status: "Pending",
		nodeCount: 3,
		templateVersion: 1,
		assetIds: [assetId],
		assetCount: 1,
		createdAt: "2026-06-15T00:00:01Z",
	})),
	total: ASSET_IDS.length,
	page: 1,
	pageSize: 20,
};

function jsonRoute(body: unknown, status = 200) {
	return (route: Route) =>
		route.fulfill({
			status,
			contentType: "application/json",
			body: JSON.stringify(body),
		});
}

/** Populate the design canvas deterministically via the "导入" JSON modal. */
async function importPipelineJSON(page: Page) {
	await page.getByRole("button", { name: "导入" }).first().click();
	const modal = page.getByRole("dialog", { name: "导入流水线" });
	await expect(modal).toBeVisible();
	await modal.getByRole("textbox").fill(JSON.stringify(IMPORT_PIPELINE));
	// antd auto-inserts a space between two CJK chars on icon-less buttons ("导 入").
	await modal.getByRole("button", { name: /导\s*入/ }).click();
	await expect(modal).toBeHidden();
}

/** Add asset IDs manually via the deploy dialog's AssetPicker (no registry needed). */
async function addAssets(modal: ReturnType<Page["getByRole"]>, ids: string[]) {
	const search = modal.getByPlaceholder("搜索资产（输入 asset_id 或名称）");
	for (const id of ids) {
		await search.fill(id);
		await modal.getByRole("button", { name: `添加「${id}」为资产 ID` }).click();
	}
}

test.describe("流水线全流程（mock）", () => {
	let createBatchBody: Record<string, unknown> | null = null;
	let saveTemplateBody: Record<string, unknown> | null = null;

	test.beforeEach(async ({ page }) => {
		createBatchBody = null;
		saveTemplateBody = null;

		// Catch-all (registered first = lowest priority). Any /api call not matched
		// by a specific mock would otherwise hit the real dev backend, return 401,
		// trip UNAUTHORIZED_EVENT and bounce us to the login page. Log + stub empty.
		await page.route("**/api/v1/**", (route) => {
			console.log(
				`[unmocked] ${route.request().method()} ${route.request().url()}`,
			);
			return jsonRoute({ items: [], total: 0, page: 1, pageSize: 20 })(route);
		});

		await page.route(api("/auth/me"), jsonRoute(authMe));
		await page.route("**/api/v1/auth/login", jsonRoute(authMe));
		await page.route(api("/workflows*"), jsonRoute({ items: [] }));
		await page.route(api("/pipeline-components*"), jsonRoute({ items: [] }));
		await page.route(api("/execution-targets*"), jsonRoute(executionTargets));

		// Template list + single template read (after save navigate adds templateId).
		await page.route(api("/pipelines/*/versions*"), (route) =>
			jsonRoute({ items: [savedTemplate(PIPELINE_NAME)] })(route),
		);
		await page.route(api(`/pipelines/${SAVED_TEMPLATE_ID}`), (route) =>
			jsonRoute(savedTemplate(PIPELINE_NAME))(route),
		);

		// Save template (POST) + list (GET) share the /pipelines path.
		await page.route(api("/pipelines"), (route) => {
			if (route.request().method() === "POST") {
				saveTemplateBody = route.request().postDataJSON() as Record<
					string,
					unknown
				>;
				const name = (saveTemplateBody.name as string) || PIPELINE_NAME;
				return jsonRoute(savedTemplate(name))(route);
			}
			return jsonRoute({ items: [], total: 0, page: 1, pageSize: 20 })(route);
		});
		await page.route(api("/pipelines?*"), (route) => {
			if (route.request().method() === "POST") return route.fallback();
			return jsonRoute({
				items: [savedTemplate(PIPELINE_NAME)],
				total: 1,
				page: 1,
				pageSize: 20,
			})(route);
		});

		// Batch create (POST /backfill).
		await page.route(api("/backfill"), (route) => {
			if (route.request().method() === "POST") {
				createBatchBody = route.request().postDataJSON() as Record<
					string,
					unknown
				>;
				return jsonRoute(batchJob({ name: createBatchBody.name as string }))(
					route,
				);
			}
			return jsonRoute({ items: [batchJob()] })(route);
		});

		// Batch detail page fetches.
		await page.route(
			api(`/backfill/${BATCH_ID}/node-summary`),
			jsonRoute(nodeSummary),
		);
		await page.route(
			api(`/backfill/${BATCH_ID}`),
			jsonRoute({ job: batchJob() }),
		);
		await page.route(
			api(`/pipeline-runs?view=summary&batchJobId=${BATCH_ID}*`),
			jsonRoute(batchSubRuns),
		);
	});

	test("设计→保存→部署→批量下发→批次详情子任务数一致", async ({ page }) => {
		test.setTimeout(60_000);

		await page.goto("/pipeline?tab=design");

		// 1) 设计：导入流水线 JSON 填充画布。
		await importPipelineJSON(page);
		const deployBtn = page.getByRole("button", { name: "部署" });
		await expect(deployBtn).toBeEnabled();

		// 2) 保存为模板。
		await page.getByRole("button", { name: "保存" }).click();
		await expect.poll(() => saveTemplateBody).not.toBeNull();
		expect(saveTemplateBody).toMatchObject({ name: PIPELINE_NAME });

		// 3) 打开部署对话框，选 ≥2 资产 → 触发批量下发。
		await deployBtn.click();
		const modal = page.getByRole("dialog", { name: "部署流水线" });
		await expect(modal).toBeVisible();

		await addAssets(modal, ASSET_IDS);
		await expect(modal.getByText(`已选 ${ASSET_IDS.length} 个`)).toBeVisible();

		await modal.getByRole("button", { name: "运行资产" }).click();

		const confirmDialog = page.getByRole("dialog", {
			name: "确认创建批量任务",
		});
		await expect(confirmDialog).toBeVisible();
		await confirmDialog.getByRole("button", { name: "确认运行" }).click();

		// 4) 批量下发：POST /backfill 带正确 templateId + assetIds。
		await expect.poll(() => createBatchBody).not.toBeNull();
		expect(createBatchBody).toMatchObject({
			templateId: SAVED_TEMPLATE_ID,
			assetIds: ASSET_IDS,
		});

		// 5) 跳转批次详情，子任务数收敛到逻辑总数。
		await expect(page).toHaveURL(new RegExp(`/pipeline/batch/${BATCH_ID}`), {
			timeout: 20_000,
		});
		await expect(page.getByText(`批次 ID: ${BATCH_ID}`)).toBeVisible({
			timeout: 15_000,
		});
		await expect(page.getByText("子任务执行记录")).toBeVisible({
			timeout: 15_000,
		});
		await expect(
			page.getByText(String(ASSET_IDS.length), { exact: true }).first(),
		).toBeVisible();
	});
});

test.describe("流水线全流程 @integration", () => {
	const token =
		process.env.VITE_DEV_ACCESS_TOKEN || process.env.E2E_DATABREW_TOKEN || "";
	test.skip(!token, "需要 E2E_DATABREW_TOKEN 或 VITE_DEV_ACCESS_TOKEN");

	test.beforeEach(async ({ page }) => {
		await page.request.post("/api/v1/auth/login", { data: { token } });
		await page.goto("/");
		await page.waitForLoadState("networkidle");
	});

	test("真连 dev：设计→保存→部署→批量下发，批次子任务数 == 资产数", async ({
		page,
	}) => {
		test.setTimeout(120_000);
		const assetIds = Array.from({ length: 3 }, (_, i) => `e2e-flow-real-${i}`);

		await page.goto("/pipeline?tab=design");
		await importPipelineJSON(page);
		const deployBtn = page.getByRole("button", { name: "部署" });
		await expect(deployBtn).toBeEnabled();

		await page.getByRole("button", { name: "保存" }).click();
		await expect(page.getByText(/已保存为 v\d+/)).toBeVisible({
			timeout: 20_000,
		});

		await deployBtn.click();
		const modal = page.getByRole("dialog", { name: "部署流水线" });
		await expect(modal).toBeVisible();
		await addAssets(modal, assetIds);
		await expect(modal.getByText(`已选 ${assetIds.length} 个`)).toBeVisible();
		await modal.getByRole("button", { name: "运行资产" }).click();
		const confirmDialog = page.getByRole("dialog", {
			name: "确认创建批量任务",
		});
		await expect(confirmDialog).toBeVisible();
		await confirmDialog.getByRole("button", { name: "确认运行" }).click();

		// 跳转批次详情。
		await expect(page).toHaveURL(/\/pipeline\/batch\//, { timeout: 30_000 });
		const batchId = page.url().split("/pipeline/batch/")[1]?.split(/[?#]/)[0];
		expect(batchId).toBeTruthy();

		// 后端真实批次：totalCount == 提交资产数；子任务列表 total 收敛。
		const jobRes = await page.request.get(`/api/v1/backfill/${batchId}`);
		expect(jobRes.ok()).toBeTruthy();
		const jobBody = (await jobRes.json()) as {
			job: { totalCount: number };
		};
		expect(jobBody.job.totalCount).toBe(assetIds.length);

		const runsRes = await page.request.get(
			`/api/v1/pipeline-runs?view=summary&batchJobId=${batchId}&page=1&pageSize=${assetIds.length}`,
		);
		expect(runsRes.ok()).toBeTruthy();
		const runsBody = (await runsRes.json()) as { total: number };
		expect(runsBody.total).toBeLessThanOrEqual(assetIds.length);
	});
});
