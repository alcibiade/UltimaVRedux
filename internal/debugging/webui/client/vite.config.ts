import { defineConfig } from "vitest/config"
import vue from "@vitejs/plugin-vue"

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: "../dist",
    emptyOutDir: true,
    sourcemap: false,
  },
  test: {
    environment: "jsdom",
    // Keep the local quality gate's browser-DOM fixtures memory bounded.
    maxWorkers: 1,
    fileParallelism: false,
    exclude: ["**/node_modules/**", "browser/**"],
  },
})
