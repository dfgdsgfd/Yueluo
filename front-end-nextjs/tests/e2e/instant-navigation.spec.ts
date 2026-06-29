import { expect, test } from "@playwright/test";
import { instant } from "@next/playwright";

test("admin shell commits client navigation in Next instant mode", async ({ page }) => {
  await page.goto("/admin");
  await expect(page.locator("form")).toBeVisible();
  const homeLink = page.locator('a[href="/"]').first();

  await instant(page, async () => {
    await homeLink.click();
    await expect(page).toHaveURL(/\/$/);
    await expect(page.locator("body")).toBeVisible();
  });
});

for (const width of [320, 375, 390, 430, 1280]) {
  test(`message flows avoid horizontal blocking at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: width < 768 ? 800 : 900 });

    for (const route of ["/messages", "/notifications"]) {
      await page.goto(route);
      await expect(page.locator("body")).toBeVisible();
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - window.innerWidth,
      );
      expect(overflow).toBeLessThanOrEqual(1);
    }
  });
}
