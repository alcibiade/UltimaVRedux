import { createApp, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import TileAtlas, { type TileAtlasEntry } from "./TileAtlas.vue"

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

describe("Tile Atlas", () => {
  it("keeps tile details sticky beside the scrolling catalog", () => {
    const entry: TileAtlasEntry = {
      id: 1,
      name: "Brick Floor",
      description: "A test tile.",
      lookDescription: "a brick floor",
      speed: 1,
      light: 0,
      animationFrames: 0,
      animated: false,
      blocksLight: false,
      enemy: false,
      npc: false,
      building: false,
      walk: true,
      horse: true,
      skiff: false,
      frigate: false,
      carpet: true,
      npcPassable: true,
    }
    app = createApp(TileAtlas, { data: [entry] })
    app.mount(root)

    const detail = root.querySelector<HTMLElement>(".atlas-detail")
    expect(detail).not.toBeNull()
    expect(detail?.hasAttribute("data-sticky-detail")).toBe(true)
    expect(root.querySelector(".atlas-layout")?.contains(root.querySelector(".table-scroll"))).toBe(true)
    expect(root.querySelector(".table-scroll")?.classList.contains("table-scroll--sticky")).toBe(true)
    expect(root.querySelector(".table-scroll")?.classList.contains("table-scroll--flow")).toBe(true)
    expect(root.querySelector(".table-scroll")?.getAttribute("aria-label")).toBe("Tile Atlas table")
    expect(detail?.textContent).toContain("LOOK2.DAT")
    expect(detail?.textContent).toContain("a brick floor")
  })
})
