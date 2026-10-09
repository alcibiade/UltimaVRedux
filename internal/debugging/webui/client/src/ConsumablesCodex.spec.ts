import { createApp, nextTick, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import ConsumablesCodex, { type ConsumableCodexEntry } from "./ConsumablesCodex.vue"

let app: VueApp<Element> | undefined
let root: HTMLElement

const entry = (overrides: Partial<ConsumableCodexEntry> = {}): ConsumableCodexEntry => ({
  kind: "potion", id: 4, name: "Orange", originalBehavior: "Sleep", reduxBehavior: "Rat transformation",
  originalContext: "Field and combat", reduxContext: "Combat unavailable", originalTarget: "Party member", reduxTarget: "Combat actor",
  originalConsumptionTiming: "Before validation", reduxConsumptionTiming: "Before context response", originalDuration: "", reduxDuration: "",
  support: "partial", deviation: "Mapping differs.", ...overrides,
})

beforeEach(() => { root = document.createElement("div"); document.body.append(root) })
afterEach(() => { app?.unmount(); app = undefined; root.remove() })

describe("Consumables Codex", () => {
  it("groups comparisons with textual support and explicit deviations", () => {
    app = createApp(ConsumablesCodex, { data: [entry(), entry({ kind: "scroll", id: 5, name: "Kal Xen Corp", support: "unavailable" })] })
    app.mount(root)

    expect(root.querySelectorAll(".consumable-group")).toHaveLength(2)
    expect(root.textContent).toContain("Original intended behavior")
    expect(root.textContent).toContain("Current Redux behavior")
    expect(root.textContent).toContain("DEVIATION")
    expect(root.textContent).toContain("unavailable")
    expect(root.textContent).toContain("CONSUME-0001")
    expect(root.textContent).toContain("CONSUME-0004")
  })

  it("filters by kind and support", async () => {
    app = createApp(ConsumablesCodex, { data: [entry(), entry({ kind: "scroll", id: 5, name: "Kal Xen Corp", support: "unavailable" })] })
    app.mount(root)
    const support = root.querySelector<HTMLSelectElement>('select[aria-label="Filter consumables by support"]')!
    support.value = "unavailable"
    support.dispatchEvent(new Event("change"))
    await nextTick()
    expect(root.textContent).toContain("Kal Xen Corp")
    expect(root.textContent).not.toContain("Orange")
  })
})
