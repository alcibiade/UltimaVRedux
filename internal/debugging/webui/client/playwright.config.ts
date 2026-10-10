import { defineConfig } from "@playwright/test"

export default defineConfig({
  testDir: "./browser",
  fullyParallel: true,
  workers: 1,
  reporter: "list",
  outputDir: "./browser/results",
  globalSetup: "./browser/server.ts",
  use: {
    browserName: "chromium", baseURL: "http://127.0.0.1:4179", viewport: { width: 1280, height: 900 },
    screenshot: "only-on-failure",
  },
})
