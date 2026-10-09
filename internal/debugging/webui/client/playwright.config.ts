import { defineConfig } from "@playwright/test"

export default defineConfig({
  testDir: "./browser",
  fullyParallel: true,
  workers: 1,
  reporter: "list",
  outputDir: "./browser/results",
  use: { browserName: "chromium", baseURL: "http://127.0.0.1:4179", viewport: { width: 1280, height: 900 } },
  webServer: {
    command: "npm run dev -- --host 127.0.0.1 --port 4179 --strictPort",
    url: "http://127.0.0.1:4179",
    reuseExistingServer: false,
  },
})
