import { expect, test, type Page, type Route } from "@playwright/test";

const giftCardNotification = {
  id: 88,
  user_id: 7,
  sender_id: 0,
  type: 21,
  title: "notification.giftCardRedeemed.title",
  target_id: 701,
  is_read: false,
  created_at: "2026-06-22T12:30:00Z",
  detail: "legacy detail",
  gift_card_redemption: {
    product_name: "E2E Gift Card",
    face_value: "$10",
    points_spent: 30,
    code: "NOTICE-CARD-SECRET",
    redeemed_at: "2026-06-22T12:30:00Z",
  },
};

test("gift card system notification opens its detail without navigating to a post", async ({ context, page }) => {
  await context.addCookies([{ name: "xse.locale", value: "zh-CN", url: "http://127.0.0.1:3001" }]);
  await installBrowserSession(page);
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === "/api/notifications") return json(route, { data: [giftCardNotification], pagination: { page: 1, limit: 50, total: 1, pages: 1 } });
    if (path === "/api/notifications/system") return json(route, { data: [], pagination: { page: 1, limit: 50, total: 0, pages: 0 } });
    if (path === "/api/notifications/unread-count") return json(route, { notification_count: 1, system_notification_count: 0, total: 1 });
    if (path.endsWith("/read")) return json(route, {});
    return json(route, []);
  });

  await page.goto("/notifications");
  await page.getByText("系统消息", { exact: true }).click();
  await expect(page.getByTestId("gift-card-notification-detail")).toBeVisible();
  await expect(page.getByTestId("gift-card-notification-code")).toHaveText("NOTICE-CARD-SECRET");
  await expect(page).toHaveURL(/\/notifications$/);
});

for (const width of [320, 375, 390, 430, 1280]) {
  test(`wallet redemption history paginates without overflow at ${width}px`, async ({ context, page }) => {
    await page.setViewportSize({ width, height: width < 768 ? 844 : 900 });
    await context.addCookies([
      { name: "yuem_access_token", value: "e2e-token", url: "http://127.0.0.1:3001" },
      { name: "xse.locale", value: "zh-CN", url: "http://127.0.0.1:3001" },
    ]);
    await installBrowserSession(page);
    await page.route("**/api/**", (route) => handleWalletApi(route));

    await page.goto("/wallet");
    const refresh = page.getByRole("button", { name: "刷新" });
    await expect(refresh).toHaveCount(1);
    await refresh.click();

    const variant = width >= 1024 ? "desktop" : "mobile";
    const history = page.getByTestId(`gift-card-redemption-history-${variant}`);
    await expect(history).toBeVisible();
    await expect(history.getByText("PAGE-1-CARD-SECRET", { exact: true })).toBeVisible();
    await history.getByRole("button", { name: "下一页" }).click();
    await expect(history.getByText("PAGE-2-CARD-SECRET", { exact: true })).toBeVisible();

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow).toBeLessThanOrEqual(1);
  });
}

test("wallet shows the returned card code immediately after redemption", async ({ context, page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await context.addCookies([
    { name: "yuem_access_token", value: "e2e-token", url: "http://127.0.0.1:3001" },
    { name: "xse.locale", value: "zh-CN", url: "http://127.0.0.1:3001" },
  ]);
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await installBrowserSession(page);
  await page.route("**/api/**", (route) => handleWalletApi(route));

  await page.goto("/wallet");
  await page.getByRole("button", { name: "刷新" }).click();
  await page.getByRole("button", { name: "30分" }).click();
  const success = page.locator('[data-testid="gift-card-redeem-success"]:visible');
  await expect(success).toContainText("NEW-CARD-SECRET");
  await success.getByRole("button", { name: "复制卡密" }).click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("NEW-CARD-SECRET");
});

async function installBrowserSession(page: Page) {
  await page.addInitScript(() => {
    window.localStorage.setItem("yuem_access_token", "e2e-token");
    window.localStorage.setItem("yuem_user", JSON.stringify({ id: 7, user_id: "gift-card-e2e", nickname: "Gift Card E2E" }));
  });
}

async function handleWalletApi(route: Route) {
  const url = new URL(route.request().url());
  const path = url.pathname;
  if (path === "/api/balance/config") return json(route, { enabled: true });
  if (path === "/api/balance/recharge-config") return json(route, { custom_amount_enable: false, options: [] });
  if (path === "/api/balance/local-points") return json(route, { points: 100 });
  if (path === "/api/balance/user-balance") return json(route, { balance: 20, username: "gift-card-e2e" });
  if (path === "/api/withdraw/orders" || path === "/api/balance/orders" || path === "/api/points/logs") {
    return json(route, { list: [], pagination: { page: 1, limit: 8, total: 0, totalPages: 1 } });
  }
  if (path === "/api/points/overview") {
    return json(route, {
      points: 100,
      today_earned: 10,
      daily_cap: 50,
      tasks: [],
      gift_cards: [{ id: 5, name: "E2E Gift Card", face_value: "$10", points_required: 30, is_active: true, available_stock: 2 }],
    });
  }
  if (path === "/api/points/redemptions") {
    const page = Number(url.searchParams.get("page") ?? 1);
    const code = page === 2 ? "PAGE-2-CARD-SECRET" : "PAGE-1-CARD-SECRET";
    return json(route, {
      list: [{ id: page, product_id: 5, code, points_spent: 30, balance_after: 70, status: "completed", created_at: "2026-06-22T12:30:00Z", product: { id: 5, name: "E2E Gift Card", face_value: "$10", points_required: 30, is_active: true, available_stock: 1 } }],
      pagination: { page, limit: 8, total: 9, totalPages: 2 },
    });
  }
  if (path === "/api/points/gift-cards/5/redeem") {
    return json(route, { id: 99, product_id: 5, code: "NEW-CARD-SECRET", points_spent: 30, balance_after: 70, status: "completed", product: { id: 5, name: "E2E Gift Card", face_value: "$10", points_required: 30, is_active: true, available_stock: 1 } });
  }
  return json(route, []);
}

async function json(route: Route, data: unknown) {
  await route.fulfill({ contentType: "application/json", body: JSON.stringify({ code: 200, message: "success", data }) });
}
