import { describe, expect, it } from "vitest"

import App from "./App.vue"
import LogWindow from "./LogWindow.vue"
import { rootComponent } from "./root_component"

describe("root component", () => {
  it("uses the read-only log root only for /logs", () => {
    expect(rootComponent("/logs")).toBe(LogWindow)
    expect(rootComponent("/")).toBe(App)
  })
})
