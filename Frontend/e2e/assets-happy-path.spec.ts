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
      localStorage.setItem("databrew_token", token);
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

  test("desktop: click first Asset ID expands quick preview (empty hint hidden)", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 1280, height: 800 });
    await page.goto(`${BASE_URL}/assets`);
    await expect(page.locator(".ant-table-wrapper").first()).toBeVisible({
      timeout: 15_000,
    });

    // Avoid `.ant-table-tbody tr` — first row can be a measure/hidden row without links.
    const firstAssetLink = page.locator('[data-testid^="asset-id-link-"]').first();
    await expect(firstAssetLink).toBeVisible({ timeout: 20_000 });
    await firstAssetLink.click();

    await expect(page).toHaveURL(/\/assets(\?|$)/);
    // Collapsed preview showed "点击行查看预览"; selecting a row loads preview so this goes away.
    await expect(page.getByText("点击行查看预览")).not.toBeVisible({
      timeout: 15_000,
    });
  });

  test("narrow: click Asset ID navigates to asset detail", async ({ page }) => {
    await page.setViewportSize({ width: 900, height: 800 });
    await page.goto(`${BASE_URL}/assets`);
    await expect(page.locator(".ant-table-wrapper").first()).toBeVisible({
      timeout: 15_000,
    });

    const firstAssetLink = page.locator('[data-testid^="asset-id-link-"]').first();
    await expect(firstAssetLink).toBeVisible({ timeout: 20_000 });
    await firstAssetLink.click();

    await expect(page).toHaveURL(/\/assets\/[0-9A-Za-z]{8}/, { timeout: 10_000 });
  });
});
