import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 30_000,
  expect: {
    timeout: 5_000,
  },
  fullyParallel: true,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:3001",
    trace: "retain-on-failure",
  },
  webServer: {
    command: "npm run dev -- -p 3001",
    reuseExistingServer: true,
    timeout: 120_000,
    url: "http://127.0.0.1:3001",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
