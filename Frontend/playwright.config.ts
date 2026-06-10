/**
 * Playwright configuration for Frontend E2E tests.
 *
 * Task: 9.5 P1-T-5: 前端 E2E（Playwright 1 条 happy-path）
 *
 * Run:
 *   cd Frontend
 *   npx playwright install --with-deps chromium
 *   npx playwright test
 */

import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
	testDir: "./e2e",
	outputDir: "./e2e/test-results",
	timeout: 30_000,
	retries: 1,
	reporter: [["html", { outputFolder: "e2e/report" }], ["list"]],

	use: {
		baseURL: process.env.E2E_BASE_URL || "http://localhost:5173",
		screenshot: "only-on-failure",
		trace: "on-first-retry",
	},

	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
