import { createApp, nextTick, ref, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import PartyRoster from "./PartyRoster.vue"

let app: VueApp<Element> | undefined
let root: HTMLElement

beforeEach(() => {
  root = document.createElement("div")
  document.body.append(root)
})

afterEach(() => {
  app?.unmount()
  app = undefined
  root.remove()
})

describe("PartyRoster active-position layout", () => {
  it.each([true, false])("retains review on identical telemetry with selected targets: %s", async (selected) => {
    const data = ref({ fingerprint: "roster-1", karma: 50, characters: [], roster: [
      { rosterIndex: 0, name: "Avatar", class: "Avatar", status: "good", active: true },
    ] })
    app = createApp({
      components: { PartyRoster }, setup: () => ({ data }),
      template: '<PartyRoster :data="data" :pending-commands="[]" :karma-pending="false" />',
    })
    app.mount(root)
    if (selected) {
      root.querySelector<HTMLButtonElement>('[data-roster-position="1"]')!.click()
      await nextTick()
    }
    root.querySelector<HTMLButtonElement>(`[data-party-batch="${selected ? "status" : "maximize"}"]`)!.click()
    await nextTick()
    const review = root.querySelector("[data-roster-operation-review]")
    expect(review).not.toBeNull()
    data.value = JSON.parse(JSON.stringify(data.value))
    await nextTick()
    expect(root.querySelector("[data-roster-operation-review]")).toBe(review)
    expect(root.querySelector('[data-roster-position="1"]')?.getAttribute("aria-pressed")).toBe(String(selected))
    data.value = { ...data.value, fingerprint: "roster-2" }
    await nextTick()
    expect(root.querySelector("[data-roster-operation-review]")).toBeNull()
  })

  it("assigns position, name, and metadata to distinct responsive grid areas", () => {
    app = createApp(PartyRoster, {
      data: {
        fingerprint: "roster-1", karma: 50, characters: [],
        roster: [{ rosterIndex: 0, name: "Avatar with a deliberately long name", class: "Avatar", status: "good", active: true }],
      },
      pendingCommands: [], karmaPending: false,
    })
    app.mount(root)

    const row = root.querySelector('[data-roster-position="1"]')
    expect(row?.querySelector(":scope > .roster-position")?.textContent).toContain("Position 1")
    expect(row?.querySelector(":scope > .roster-name")?.textContent).toContain("Avatar")
    expect(row?.querySelector(":scope > .roster-metadata")?.textContent).toContain("Roster index 0")
    expect(row?.children).toHaveLength(3)
  })
})
