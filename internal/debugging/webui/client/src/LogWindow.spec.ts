import { createApp, nextTick } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import LogWindow from "./LogWindow.vue"

describe("read-only log window", () => {
  beforeEach(() => {
    document.head.innerHTML = '<meta name="u5r-workbench-token" content="read-token">'
    document.body.innerHTML = '<div id="log-test"></div>'
  })

  afterEach(() => vi.restoreAllMocks())

  it("polls structured logs with the read token and renders messages as text", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        entries: [{ sequence: 4, timestamp: new Date().toISOString(), level: "INFO", category: "system", message: "<b>safe text</b>" }],
        nextCursor: 4,
        dropped: 0,
        hasMore: false,
      }),
    })
    vi.stubGlobal("fetch", fetchMock)
    const app = createApp(LogWindow)
    app.mount("#log-test")
    await Promise.resolve()
    await Promise.resolve()
    await nextTick()

    expect(fetchMock).toHaveBeenCalledWith("/api/logs?after=0&limit=256", expect.objectContaining({
      headers: { "X-U5R-Log-Token": "read-token" },
    }))
    expect(document.querySelector(".log-entry p")?.textContent).toBe("<b>safe text</b>")
    expect(document.querySelector(".log-entry p b")).toBeNull()
    app.unmount()
  })
})
