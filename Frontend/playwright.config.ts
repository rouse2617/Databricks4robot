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

const previewId = (process.env.E2E_PREVIEW_ID || "").trim();
const previewHost = (
	process.env.E2E_PREVIEW_HOST || "https://cyber-databrew-dev.cyberorigin.ai"
).replace(/\/$/, "");
const apiBaseUrl = (
	process.env.E2E_API_BASE_URL ||
	process.env.VITE_API_BASE_URL ||
	""
).trim();
const devToken = (
	process.env.E2E_DATABREW_TOKEN ||
	process.env.VITE_DEV_ACCESS_TOKEN ||
	""
).trim();

const devServerEnv = [
	previewId ? `VITE_PREVIEW_ID=${previewId}` : "",
	previewId ? `VITE_PREVIEW_HOST=${previewHost}` : "",
	apiBaseUrl ? `VITE_API_BASE_URL=${apiBaseUrl}` : "",
	devToken ? `VITE_DEV_ACCESS_TOKEN=${devToken}` : "",
]
	.filter(Boolean)
	.join(" ");

export default defineConfig({
	testDir: "./e2e",
	outputDir: "./e2e/test-results",
	timeout: 30_000,
	retries: 1,
	reporter: [["html", { outputFolder: "e2e/report" }], ["list"]],

	use: {
		baseURL: process.env.E2E_BASE_URL || "http://localhost:5176",
		screenshot: "only-on-failure",
		trace: "on-first-retry",
	},

	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],

	webServer: {
		command: devServerEnv ? `${devServerEnv} npm run dev` : "npm run dev",
		url: "http://127.0.0.1:5176",
		reuseExistingServer: !previewId,
		timeout: 120_000,
	},
});
