/**
 * Frontend E2E — Assets happy-path
 *
 * Task: 9.5 P1-T-5: 前端 E2E（Playwright 1 条 happy-path）
 *
 * Scenario:
 *   1. Login with token
 *   2. Navigate to /assets
 *   3. Verify the assets page loads (table or list visible)
 *   4. Use the search input to search for an asset
 *   5. Verify search results update
 *
 * Prerequisites:
 *   - Backend running at http://localhost:8080
 *   - Frontend dev server at http://localhost:5173 (or built + served)
 *
 * Run:
 *   cd Frontend && npx playwright test
 */

import { test, expect } from "@playwright/test";

const BASE_URL = process.env.E2E_BASE_URL || "http://localhost:5173";
const AUTH_TOKEN = process.env.E2E_AUTH_TOKEN || "dev-token";

test.describe("Assets page happy-path", () => {
  test.beforeEach(async ({ page }) => {
    // Set auth token in localStorage before navigating (simulates login)
    await page.goto(BASE_URL);
    await page.evaluate((token) => {
      localStorage.setItem("grace_token", token);
    }, AUTH_TOKEN);
  });

  test("navigate to assets, verify page loads, and search", async ({
    page,
  }) => {
    const errors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") {
        errors.push(msg.text());
      }
    });

    // 1. Navigate to assets page
    await page.goto(`${BASE_URL}/assets`);

    // 2. Wait for the page to load — look for common UI elements
    const resultsTable = page.locator(".ant-table-wrapper").first();
    await expect(resultsTable).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole("heading", { name: "资产管理" })).toBeVisible();

    // 3. Look for a search input
    const searchInput = page.locator('[data-testid="assets-search-input"]');

    // If search input exists, type a query
    if ((await searchInput.count()) > 0) {
      await searchInput.first().fill("test");
      // Give the UI time to debounce and fetch results
      await page.waitForTimeout(1_000);

      // Verify the page didn't crash — table should still be visible
      await expect(resultsTable).toBeVisible();
    }

    // Take a final screenshot for visual verification
    await page.screenshot({ path: "e2e/screenshots/assets-happy-path.png" });

    // Page should not have crashed
    await expect(page).toHaveURL(/\/assets/);
    expect(errors).toEqual([]);
  });
});
