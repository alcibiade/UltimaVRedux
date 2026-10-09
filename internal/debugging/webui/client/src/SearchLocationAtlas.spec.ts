import { createApp, nextTick, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import SearchLocationAtlas, { type SearchLocationAtlasEntry } from "./SearchLocationAtlas.vue"

let app: VueApp<Element> | undefined
let root: HTMLElement

const entry = (overrides: Partial<SearchLocationAtlasEntry> = {}): SearchLocationAtlasEntry => ({
  index: 0, locationID: 0, locationName: "BritanniaUnderworld", mapType: "Large Map", floor: -1, x: 233, y: 233,
  rawObjectType: 11, rawPayload: 15, classification: "inventory_item", category: "Armour", itemID: 15,
  itemName: "Mystic Armour", pickupQuantity: 1, quantitySemantics: "payload_is_equipment_id", repeatPolicy: "one_time",
  requirements: [], suspicious: true, encodingNote: "Intent is unclear.", ...overrides,
})

beforeEach(() => { root = document.createElement("div"); document.body.append(root) })
afterEach(() => { app?.unmount(); app = undefined; root.remove() })

describe("Search Location Atlas", () => {
  it("shows provenance, raw encoding, suspicious notes, and no live found state", () => {
    app = createApp(SearchLocationAtlas, { data: [entry()] })
    app.mount(root)

    expect(root.textContent).toContain("CMD-0007")
    expect(root.textContent).toContain("0x0B / 0x0F")
    expect(root.textContent).toContain("SUSPICIOUS")
    expect([...root.querySelectorAll("th")].map((cell) => cell.textContent)).not.toContain("Found")
    expect(root.querySelector('[role="region"]')?.getAttribute("aria-label")).toBe("Search Location Atlas table")
  })

  it("filters independently by location, item, classification, and repeat policy", async () => {
    app = createApp(SearchLocationAtlas, { data: [entry(), entry({ index: 14, locationName: "Minoc", itemName: "Skull Keys", classification: "special_item", repeatPolicy: "daily" })] })
    app.mount(root)

    const select = root.querySelector<HTMLSelectElement>('select[aria-label="Filter search locations by repeat policy"]')!
    select.value = "daily"
    select.dispatchEvent(new Event("change"))
    await nextTick()
    expect(root.querySelector("tbody")?.textContent).toContain("Skull Keys")
    expect(root.querySelector("tbody")?.textContent).not.toContain("Mystic Armour")
  })

  it("uses raw values behind friendly classification and repeat labels", async () => {
    app = createApp(SearchLocationAtlas, { data: [
      entry(),
      entry({ index: 13, itemName: "Keys", classification: "special_item", repeatPolicy: "conditional" }),
    ] })
    app.mount(root)

    const classification = root.querySelector<HTMLSelectElement>('select[aria-label="Filter search locations by classification"]')!
    classification.value = "special_item"
    classification.dispatchEvent(new Event("change"))
    await nextTick()
    expect(root.querySelector("tbody")?.textContent).toContain("Keys")
    expect(root.querySelector("tbody")?.textContent).not.toContain("Mystic Armour")

    classification.value = ""
    classification.dispatchEvent(new Event("change"))
    const repeat = root.querySelector<HTMLSelectElement>('select[aria-label="Filter search locations by repeat policy"]')!
    repeat.value = "one_time"
    repeat.dispatchEvent(new Event("change"))
    await nextTick()
    expect(root.querySelector("tbody")?.textContent).toContain("Mystic Armour")
    expect(root.querySelector("tbody")?.textContent).not.toContain("Keys")
  })

  it("does not match a Mystic Armour row because its encoding note mentions torches", async () => {
    app = createApp(SearchLocationAtlas, { data: [
      entry({ itemName: "Mystic Armour", encodingNote: "Alternates Mystic Armour and 41 torches." }),
      entry({ index: 1, itemName: "Torches", encodingNote: "" }),
    ] })
    app.mount(root)

    const text = root.querySelector<HTMLInputElement>('input[aria-label="Filter search locations by text"]')!
    text.value = "torch"
    text.dispatchEvent(new Event("input"))
    await nextTick()
    expect(root.querySelector("tbody")?.textContent).toContain("Torches")
    expect(root.querySelector("tbody")?.textContent).not.toContain("Mystic Armour")
  })
})
