import { createApp, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import CodexCatalog from "./CodexCatalog.vue"
import NPCScheduleIndex, { type NPCScheduleEntry } from "./NPCScheduleIndex.vue"

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

describe("long reference tables", () => {
  it("opts long Codex catalogs into keyboard-accessible sticky scrolling", () => {
    app = createApp(CodexCatalog, {
      title: "Monster Bestiary",
      subtitle: "Monster reference",
      data: [{ id: 1, name: "Rat" }],
      stickyHeader: true,
    })
    app.mount(root)

    const wrapper = root.querySelector(".table-scroll")
    expect(wrapper?.classList.contains("table-scroll--sticky")).toBe(true)
    expect(wrapper?.getAttribute("tabindex")).toBe("0")
    expect(wrapper?.getAttribute("aria-label")).toBe("Monster Bestiary table")
  })

  it("keeps NPC Schedule in the normal catalog grid while enabling sticky headings", () => {
    const entry: NPCScheduleEntry = {
      locationID: 1,
      dialogNumber: 2,
      spriteIndex: 3,
      location: "Britain",
      type: "person",
      times: ["08:00"],
      ai: ["WANDER"],
      positions: ["10, 12"],
      conversationKind: "towne",
      conversationSource: "towne",
      conversationName: "Iolo",
      conversationDescription: "A bard.",
      conversationIndex: 2,
      hasConversation: true,
    }
    app = createApp(NPCScheduleIndex, { data: [entry] })
    app.mount(root)

    const wrapper = root.querySelector(".table-scroll")
    expect(wrapper?.parentElement?.classList.contains("codex-atlas")).toBe(true)
    expect(wrapper?.parentElement?.classList.contains("atlas-layout")).toBe(false)
    expect(wrapper?.classList.contains("table-scroll--sticky")).toBe(true)
    expect(wrapper?.classList.contains("table-scroll--flow")).toBe(true)
    expect(wrapper?.getAttribute("aria-label")).toBe("NPC Schedule Index table")
  })
})
