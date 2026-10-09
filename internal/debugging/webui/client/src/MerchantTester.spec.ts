import { createApp, h, nextTick, reactive, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { MerchantTesterCatalogData, MerchantTesterMerchantData } from "./approved_tool_views"
import MerchantTester from "./MerchantTester.vue"
import type { CommandDefinition } from "./protocol"

const commands: CommandDefinition[] = [
  {
    id: "merchant.tester.preview.open", category: "Merchants", name: "Open Merchant Preview",
    description: "Open the production merchant presentation.", executionModes: ["workflow"], destructive: false,
  },
  {
    id: "merchant.tester.live.open", category: "Merchants", name: "Open Live Merchant Session",
    description: "Open a reviewed production merchant session.", executionModes: ["workflow", "coordinated_alteration"], destructive: true,
  },
]

describe("MerchantTester", () => {
  let app: VueApp<Element> | undefined
  let root: HTMLDivElement

  beforeEach(() => {
    root = document.createElement("div")
    document.body.append(root)
  })

  afterEach(() => {
    app?.unmount()
    root.remove()
  })

  it("derives sorted towns and keeps all 46 unavailable merchants discoverable with their reasons", async () => {
    const merchants = Array.from({ length: 46 }, (_, index) => merchant(
      `merchant.${String(index).padStart(2, "0")}`,
      index % 2 === 0 ? 9 : 3,
      index % 2 === 0 ? "Yew" : "Britain",
      index,
      { familyId: index % 3 === 0 ? "tavern" : "inn", familyLabel: index % 3 === 0 ? "Tavern" : "Inn" },
    ))
    const harness = mountHarness(root, catalog(merchants, { locationId: 3, locationName: "Britain" }))
    app = harness.app
    await nextTick()

    expect([...root.querySelectorAll("[data-merchant-town] option")].map((option) => option.textContent)).toEqual([
      "Britain · LOCATION 3",
      "Yew · LOCATION 9",
    ])
    const discoveredMerchantIDs = new Set([...root.querySelectorAll<HTMLElement>("[data-merchant-id]")].map((row) => row.dataset.merchantId))
    expect(root.querySelectorAll("[data-merchant-id]")).toHaveLength(23)
    expect(root.querySelectorAll("[data-merchant-id].unavailable")).toHaveLength(23)
    expect(root.querySelectorAll("[data-merchant-id].unavailable em")).toHaveLength(23)
    expect([...root.querySelectorAll(".merchant-family-groups > section > header strong")].map((item) => item.textContent)).toEqual(["Inn", "Tavern"])
    expect(root.querySelector("[data-merchant-id]")?.textContent).toContain("This merchant family is not implemented.")

    setSelect(root.querySelector<HTMLSelectElement>("[data-merchant-town]"), "9")
    await nextTick()
    for (const row of root.querySelectorAll<HTMLElement>("[data-merchant-id]")) discoveredMerchantIDs.add(row.dataset.merchantId)
    expect(root.querySelectorAll("[data-merchant-id]")).toHaveLength(23)
    expect(root.querySelectorAll("[data-merchant-id].unavailable")).toHaveLength(23)
    expect(root.querySelectorAll("[data-merchant-id].unavailable em")).toHaveLength(23)
    expect(discoveredMerchantIDs.size).toBe(46)
    click(root.querySelector<HTMLElement>('[data-merchant-id="merchant.00"]'))
    await nextTick()
    expect(root.querySelector(".merchant-identity")?.textContent).toContain("merchant.00")
    expect(root.querySelector(".merchant-identity")?.textContent).toContain("This merchant family is not implemented.")
    expect(root.textContent).toContain("46 MERCHANTS · 2 TOWNS")
  })

  it("emits READY Arms Preview and reviewed Live drafts while App retains typed serialization authority", async () => {
    const preview = vi.fn()
    const live = vi.fn()
    const rows = [
      merchant("arms.britain", 3, "Britain", 0, {
        familyId: "arms", familyLabel: "Arms Dealer", implemented: true,
        previewAvailable: true, liveAvailable: true, liveExecutionMode: "workflow",
      }),
      merchant("guild.paws", 4, "Paws", 2, { implemented: true, previewAvailable: true, liveAvailable: true, liveExecutionMode: "coordinated_alteration" }),
    ]
    const harness = mountHarness(root, catalog(rows), preview, live)
    app = harness.app
    await nextTick()

    click(root.querySelector("[data-merchant-preview]"))
    expect(root.querySelector('[data-merchant-id="arms.britain"]')?.textContent).toContain("READY")
    expect(preview).toHaveBeenCalledWith({ merchantId: "arms.britain" })

    setSelect(root.querySelector<HTMLSelectElement>("[data-merchant-town]"), "4")
    await nextTick()
    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")?.textContent).toContain("Paws · LOCATION 4")
    expect(root.querySelector("[data-merchant-live-review-panel]")?.textContent).toContain("2 / 17 / 29")
    expect(root.querySelector("[data-merchant-live-review-panel]")?.textContent).toContain("COORDINATED ALTERATION")
    click(root.querySelector("[data-merchant-live-open]"))
    expect(live).toHaveBeenCalledWith({
      merchantId: "guild.paws",
      expectedLocationId: "3",
      expectedFloor: "2",
      expectedX: "17",
      expectedY: "29",
      expectedSourceFingerprint: "merchant-source-fingerprint-1234567890",
    })
  })

  it("invalidates Live review on merchant, town, and incoming source or context changes", async () => {
    const rows = [
      merchant("guild.britain.one", 3, "Britain", 1, { implemented: true, previewAvailable: true, liveAvailable: true }),
      merchant("guild.britain.two", 3, "Britain", 2, { implemented: true, previewAvailable: true, liveAvailable: true }),
      merchant("guild.paws", 4, "Paws", 3, { implemented: true, previewAvailable: true, liveAvailable: true }),
    ]
    const harness = mountHarness(root, catalog(rows))
    app = harness.app
    await nextTick()

    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).not.toBeNull()
    click(root.querySelector('[data-merchant-id="guild.britain.two"]'))
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()

    click(root.querySelector("[data-merchant-live-review]"))
    setSelect(root.querySelector<HTMLSelectElement>("[data-merchant-town]"), "4")
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()

    setSelect(root.querySelector<HTMLSelectElement>("[data-merchant-town]"), "3")
    await nextTick()
    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    harness.state.catalog = { ...harness.state.catalog, sourceFingerprint: "replacement-source" }
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()

    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    harness.state.catalog = { ...harness.state.catalog, currentContext: { ...harness.state.catalog.currentContext, x: 18 } }
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()
  })

  it("invalidates Live review when availability, effective mode, or selected action changes", async () => {
    const preview = vi.fn()
    const rows = [merchant("guild.britain", 3, "Britain", 1, {
      implemented: true, previewAvailable: true, liveAvailable: true, liveExecutionMode: "workflow",
    })]
    const harness = mountHarness(root, catalog(rows), preview)
    app = harness.app
    await nextTick()

    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    harness.state.catalog = { ...harness.state.catalog, merchants: [{ ...rows[0], liveAvailable: false }] }
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()

    harness.state.catalog = { ...harness.state.catalog, merchants: rows }
    await nextTick()
    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    harness.state.catalog = {
      ...harness.state.catalog,
      merchants: [{ ...rows[0], liveExecutionMode: "coordinated_alteration" }],
    }
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()

    harness.state.catalog = { ...harness.state.catalog, merchants: rows }
    await nextTick()
    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    click(root.querySelector("[data-merchant-preview]"))
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()
    expect(preview).toHaveBeenCalledWith({ merchantId: "guild.britain" })
  })

  it("invalidates actions when scene generation advances until the Merchant catalog refreshes", async () => {
    const rows = [merchant("guild.britain", 3, "Britain", 1, { implemented: true, previewAvailable: true, liveAvailable: true })]
    const harness = mountHarness(root, catalog(rows))
    app = harness.app
    await nextTick()

    click(root.querySelector("[data-merchant-live-review]"))
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).not.toBeNull()

    harness.state.sceneGeneration++
    await nextTick()
    expect(root.querySelector("[data-merchant-live-review-panel]")).toBeNull()
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-preview]")?.disabled).toBe(true)
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-live-review]")?.disabled).toBe(true)
    expect(root.textContent).toContain("Refresh Merchant Tester for the current scene.")

    harness.state.catalogSceneGeneration = harness.state.sceneGeneration
    harness.state.catalog = { ...harness.state.catalog }
    await nextTick()
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-preview]")?.disabled).toBe(false)
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-live-review]")?.disabled).toBe(false)
  })

  it("propagates Preview and Live pending states independently", async () => {
    const rows = [merchant("guild.britain", 3, "Britain", 1, { implemented: true, previewAvailable: true, liveAvailable: true })]
    const harness = mountHarness(root, catalog(rows))
    app = harness.app
    await nextTick()

    harness.state.pendingPreview = true
    await nextTick()
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-preview]")?.disabled).toBe(true)
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-live-review]")?.disabled).toBe(false)

    harness.state.pendingPreview = false
    harness.state.pendingLive = true
    await nextTick()
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-preview]")?.disabled).toBe(false)
    expect(root.querySelector<HTMLButtonElement>("[data-merchant-live-review]")?.disabled).toBe(true)
  })
})

function mountHarness(root: HTMLElement, catalogData: MerchantTesterCatalogData, preview = vi.fn(), live = vi.fn()) {
  const state = reactive({
    catalog: catalogData,
    catalogSceneGeneration: 7,
    commands,
    sceneGeneration: 7,
    pendingPreview: false,
    pendingLive: false,
  })
  const app = createApp({
    setup: () => () => h(MerchantTester, {
      catalog: state.catalog,
      catalogSceneGeneration: state.catalogSceneGeneration,
      commands: state.commands,
      sceneGeneration: state.sceneGeneration,
      pendingPreview: state.pendingPreview,
      pendingLive: state.pendingLive,
      onPreview: preview,
      onLive: live,
    }),
  })
  app.mount(root)
  return { app, state }
}

function catalog(merchants: MerchantTesterMerchantData[], context: Partial<MerchantTesterCatalogData["currentContext"]> = {}): MerchantTesterCatalogData {
  return {
    sourceFingerprint: "merchant-source-fingerprint-1234567890",
    currentContext: { locationId: 3, locationName: "Britain", floor: 2, x: 17, y: 29, ...context },
    merchants,
  }
}

function merchant(
  merchantId: string,
  townId: number,
  townName: string,
  dialogCode: number,
  overrides: Partial<MerchantTesterMerchantData> = {},
): MerchantTesterMerchantData {
  return {
    merchantId,
    townId,
    townName,
    shopName: `Shop ${dialogCode}`,
    ownerName: `Owner ${dialogCode}`,
    familyId: "guild",
    familyLabel: "Guild",
    dialogCode,
    implemented: false,
    unavailableReason: "This merchant family is not implemented.",
    previewAvailable: false,
    previewUnavailableReason: "This merchant family is not implemented.",
    liveAvailable: false,
    liveUnavailableReason: "This merchant family is not implemented.",
    liveExecutionMode: "coordinated_alteration",
    ...overrides,
    ...(overrides.implemented ? {
      unavailableReason: undefined,
      previewUnavailableReason: undefined,
      liveUnavailableReason: undefined,
    } : {}),
  }
}

function click(element: Element | null): void {
  if (!(element instanceof HTMLElement)) throw new Error("missing test element")
  element.dispatchEvent(new MouseEvent("click", { bubbles: true }))
}

function setSelect(control: HTMLSelectElement | null, value: string): void {
  if (!control) throw new Error("missing test select")
  control.value = value
  control.dispatchEvent(new Event("change", { bubbles: true }))
}
