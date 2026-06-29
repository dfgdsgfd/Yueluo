import { expect, test } from "@playwright/test";

const widths = [320, 375, 390, 430];

for (const width of widths) {
  test(`mobile publish remains usable at ${width}px`, async ({ context, page }) => {
    await page.setViewportSize({ width, height: 844 });
    await context.addCookies([{ name: "yuem_access_token", value: "e2e-token", url: "http://127.0.0.1:3001" }]);
    await page.addInitScript(() => {
      window.localStorage.setItem("yuem_access_token", "e2e-token");
    });
    await page.route("**/api/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (path === "/api/auth/me") {
        await route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({ code: 200, data: { id: 7, user_id: "mobile-publish-e2e", nickname: "Mobile Publish E2E" } }),
        });
        return;
      }
      if (path === "/api/posts/protection-config") {
        await route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({
            code: 0,
            data: {
              enabled: true,
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
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ code: 0, data: [] }),
      });
    });

    await page.goto("/publish/mobile");
    const shell = page.locator(".mobile-publish-page > div");
    await expect(shell).toBeVisible();
    const liveInput = page.getByTestId("mobile-markdown-live-input");

    await expect(liveInput).toBeVisible();
    await liveInput.locator(".ProseMirror").click();
    await page.keyboard.type("Mobile visual editing works");
    await expect(liveInput.locator(".ProseMirror")).toContainText("Mobile visual editing works");

    await page.getByTestId("mobile-markdown-mode-toggle").click();
    const sourceInput = page.getByTestId("mobile-markdown-source-input");
    await expect(sourceInput).toBeVisible();
    await expect(sourceInput).toHaveValue("Mobile visual editing works");
    await sourceInput.fill("# Mobile Markdown\n\n- realtime preview");
    await page.getByTestId("mobile-markdown-mode-toggle").click();

    await expect(liveInput.locator("h1")).toContainText("Mobile Markdown");
    await expect(liveInput.locator("li")).toContainText("realtime preview");
    await expect(page.getByTestId("mobile-markdown-source-input")).toHaveCount(0);

    const metrics = await page.evaluate(() => ({
      bodyWidth: document.body.scrollWidth,
      viewportWidth: document.documentElement.clientWidth,
    }));
    expect(metrics.bodyWidth).toBeLessThanOrEqual(metrics.viewportWidth);
    const box = await shell.boundingBox();
    expect(box?.width).toBeLessThanOrEqual(width);
  });
}
