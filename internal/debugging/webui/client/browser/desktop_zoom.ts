import { chromium, test as base } from "@playwright/test"
import { mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { fileURLToPath } from "node:url"

export const test = base.extend<{ setBrowserZoom: (factor: number) => Promise<number> }>({
  context: async ({ baseURL, headless }, use) => {
    const profile = await mkdtemp(join(tmpdir(), "u5r-browser-zoom-"))
    const extension = fileURLToPath(new URL("./zoom_extension", import.meta.url))
    try {
      // Full bundled Chromium's new headless mode supports extensions. No viewport
      // or device emulation: setZoom must reflow a real, unchanged browser window.
      const context = await chromium.launchPersistentContext(profile, {
        channel: "chromium", headless, baseURL, viewport: null,
        args: ["--window-size=1280,900", `--disable-extensions-except=${extension}`, `--load-extension=${extension}`],
      })
      try { await use(context) } finally { await context.close() }
    } finally { await rm(profile, { recursive: true, force: true }) }
  },
  setBrowserZoom: async ({ context, page }, use) => {
    const worker = context.serviceWorkers()[0] ?? await context.waitForEvent("serviceworker", { timeout: 10_000 })
    await use((factor) => worker.evaluate(({ url, factor }) => {
      const zoom = globalThis as unknown as { zoomWorkbenchTab: (url: string, factor: number) => Promise<number> }
      return zoom.zoomWorkbenchTab(url, factor)
    }, { url: page.url(), factor }))
  },
})
