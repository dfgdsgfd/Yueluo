import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { expect, test } from "@playwright/test";

const fullMarkdown = readFileSync(resolve(process.cwd(), "../testdata/Full-Markdown.md"), "utf8");

test("full Markdown fixture renders every Mermaid diagram across theme changes", async ({ context, page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await context.addCookies([{ name: "xse.locale", value: "zh-CN", url: "http://127.0.0.1:3001" }]);
  await page.addInitScript(() => {
    window.localStorage.setItem("yuem_access_token", "e2e-token");
    window.localStorage.setItem("yuem_explore_theme", "light");
  });
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/posts/protection-config") {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          code: 200,
          data: {
            enabled: false,
            maxImages: 100,
            maxContentLength: 100000,
            noticeEnabled: true,
            selectAllEnabled: true,
            paymentMethods: { balance: true, points: true },
            paymentMaxPrices: { balance: 2000, points: 50000 },
          },
        }),
      });
      return;
    }
    if (path === "/api/auth/me") {
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ code: 200, data: { id: 1, user_id: "markdown-e2e", nickname: "Markdown E2E" } }) });
      return;
    }
    await route.fulfill({ contentType: "application/json", body: JSON.stringify({ code: 200, data: [] }) });
  });

  await page.goto("/publish/mobile");
  await page.locator("textarea").fill(fullMarkdown);
  await page.getByRole("button", { name: "预览 Markdown" }).click();

  const diagrams = page.locator(".mermaid-diagram");
  await expect(diagrams).toHaveCount(7);
  await expect(page.locator('.mermaid-diagram[data-mermaid-status="rendered"]')).toHaveCount(7);
  await expect(page.locator('.mermaid-diagram svg[id*="-light-"]')).toHaveCount(7);
  await expect(page.locator('.mermaid-diagram[data-mermaid-status="error"]')).toHaveCount(0);

  await page.evaluate(() => {
    document.documentElement.dataset.yuemTheme = "dark";
  });
  await expect(page.locator('.mermaid-diagram svg[id*="-dark-"]')).toHaveCount(7);
  await expect(page.locator('.mermaid-diagram[data-mermaid-status="error"]')).toHaveCount(0);
});
