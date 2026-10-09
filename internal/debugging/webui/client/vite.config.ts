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
    exclude: ["**/node_modules/**", "browser/**"],
  },
})
