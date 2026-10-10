import { createApp, h, nextTick, reactive, type App as VueApp } from "vue"
import { afterEach, describe, expect, it, vi } from "vitest"

import DungeonExplorer from "./DungeonExplorer.vue"
import WorldNavigator from "./WorldNavigator.vue"
import type { DungeonExplorerStatusData, WorldPositionStatusData } from "./approved_tool_views"
import type { Choice } from "./protocol"

let app: VueApp<Element> | undefined
let root: HTMLElement

afterEach(() => {
  app?.unmount()
  app = undefined
  root?.remove()
})

describe("World Navigator", () => {
  it("keeps ownership of an unsubmitted matching draft and edits made after Apply", async () => {
    const state = reactive({ data: worldStatus() })
    mount(() => h(WorldNavigator, { data: state.data, pending: false, landmarks: [], pendingLandmark: false }))
    await nextTick()
    setInput(inputs('.navigator-form input[type="number"]')[0], "102")
    state.data = worldStatus({ x: 102 })
    await nextTick()
    state.data = worldStatus({ x: 103 })
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]')[0]?.value).toBe("102")
    click('.navigator-form button[type="submit"]')
    setInput(inputs('.navigator-form input[type="number"]')[0], "104")
    state.data = worldStatus({ x: 102 })
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]')[0]?.value).toBe("104")
    click('.navigator-form .reset-button')
    await nextTick()
    state.data = worldStatus({ x: 105 })
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]')[0]?.value).toBe("105")
  })

  it("preserves route-only edits in exact placement until explicitly reset", async () => {
    const state = reactive({ data: worldStatus({ locationId: 33, locationName: "Deceit", mapType: "dungeon", x: 1, y: 1, bounds: dungeonBounds }) })
    mount(() => h(WorldNavigator, { data: state.data, pending: false, landmarks: [], pendingLandmark: false }))
    await nextTick()
    setSelect('[aria-label="Dungeon entry route"]', "underworld")
    state.data = { ...state.data, sourceFingerprint: "refreshed" }
    await nextTick()
    expect(select('[aria-label="Dungeon entry route"]').value).toBe("underworld")
    click('.navigator-form .reset-button')
    await nextTick()
    expect(select('[aria-label="Dungeon entry route"]').value).toBe("britannia")
  })

  it("Quick teleports to the selected small-map start using fresh preconditions", async () => {
    const apply = vi.fn()
    const state = reactive({ data: worldStatus(), pending: false })
    mount(() => h(WorldNavigator, { data: state.data, pending: state.pending, landmarks: [], pendingLandmark: false, onApply: apply }))
    await nextTick()
    expect(root.querySelector('[data-world-position-quick]')).toBeNull()
    click('[aria-label="Show all world destinations"]')
    await nextTick()
    click('[data-world-destination="2"]')
    await nextTick()
    setSelect('[aria-label="Destination floor"]', "1")
    setInput(inputs('.navigator-form input[type="number"]')[0], "3")
    click('input[value="worldEntrance"]')
    state.data = worldStatus({ x: 102, sourceFingerprint: "fresh" })
    state.pending = true
    await nextTick()
    click('[data-world-position-quick]')
    expect(apply).not.toHaveBeenCalled()
    state.pending = false
    await nextTick()
    click('[data-world-position-quick]')
    expect(apply).toHaveBeenCalledExactlyOnceWith({
      placement: "exact", locationId: 2, floor: 0, x: 15, y: 30,
      expectedLocationId: 0, expectedFloor: 0, expectedX: 102, expectedY: 101, expectedSourceFingerprint: "fresh",
    })
    state.data = worldStatus({ locationId: 2, locationName: "Britain", mapType: "small", x: 15, y: 30 })
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["15", "30"])
    expect(root.querySelector<HTMLButtonElement>('[data-world-position-quick]')?.disabled).toBe(true)
    click('[aria-label="Show all world destinations"]')
    await nextTick()
    click('[data-world-destination="33"]')
    await nextTick()
    expect(root.querySelector('[data-world-position-quick]')).toBeNull()
  })

  it("preserves one complete draft through a refreshed snapshot and reconciles only after a complete match", async () => {
    const apply = vi.fn()
    const refresh = vi.fn()
    const state = reactive<{ data: WorldPositionStatusData; pending: boolean }>({ data: worldStatus(), pending: false })
    mount(() => h(WorldNavigator, {
      data: state.data,
      pending: state.pending,
      landmarks: [],
      pendingLandmark: false,
      onApply: apply,
      onRefresh: refresh,
    }))
    await nextTick()

    click("[data-world-position-refresh]")
    expect(refresh).toHaveBeenCalledOnce()

    const location = root.querySelector<HTMLInputElement>('[aria-label="World destination"]')
    if (!location) throw new Error("missing world destination combobox")
    location.click()
    await nextTick()
    expect(root.querySelectorAll('[role="listbox"] [role="option"]')).toHaveLength(destinations.length)
    setInput(location, "dec")
    await nextTick()
    expect([...root.querySelectorAll('[role="listbox"] [role="option"]')].map((option) => option.textContent)).toEqual(["Deceitdungeon #33"])
    location.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
    await nextTick()
    expect(select('[aria-label="Destination floor"]').options).toHaveLength(8)
    expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["1", "1"])
    setSelect('[aria-label="Destination floor"]', "7")
    setInput(inputs('.navigator-form input[type="number"]')[0], "6")
    setInput(inputs('.navigator-form input[type="number"]')[1], "7")
    await nextTick()
		location.click()
		await nextTick()
		location.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
		await nextTick()
		expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["6", "7"])

    state.data = worldStatus({ sourceFingerprint: "world-source-2", x: 102, y: 103 })
    await nextTick()
    expect(location.value).toBe("Deceit")
    expect(select('[aria-label="Destination floor"]').value).toBe("7")
    expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["6", "7"])

    click('.navigator-form button[type="submit"]')
    expect(apply).toHaveBeenCalledWith({
      placement: "exact",
      locationId: 33, floor: 7, x: 6, y: 7,
      expectedLocationId: 0, expectedFloor: 0, expectedX: 102, expectedY: 103,
      expectedSourceFingerprint: "world-source-2",
    })

    state.data = worldStatus({ locationId: 33, locationName: "Deceit", mapType: "dungeon", floor: 7, x: 6, y: 7, bounds: dungeonBounds })
    await nextTick()
    state.data = worldStatus({ locationId: 33, locationName: "Deceit", mapType: "dungeon", floor: 7, x: 1, y: 2, bounds: dungeonBounds })
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["1", "2"])

    setInput(inputs('.navigator-form input[type="number"]')[0], "5")
    await nextTick()
    click(".navigator-form .reset-button")
    await nextTick()
    expect(inputs('.navigator-form input[type="number"]').map((input) => input.value)).toEqual(["1", "2"])
  })

  it("submits authoritative dungeon route intents without deriving target coordinates", async () => {
    const apply = vi.fn()
    const state = reactive<{ data: WorldPositionStatusData }>({ data: worldStatus() })
    mount(() => h(WorldNavigator, {
      data: state.data, pending: false, landmarks: [], pendingLandmark: false, onApply: apply,
    }))
    await nextTick()

    click('[aria-label="World destination"]')
    await nextTick()
    click('[data-world-destination="33"]')
    await nextTick()
    expect(select('[aria-label="Dungeon entry route"]').value).toBe("britannia")
    click('input[name="world-placement"][value="dungeon_outside_entrance"]')
    await nextTick()
    expect(root.querySelector(".dungeon-route-summary")?.textContent).toContain("Britannia")
    expect(root.querySelector(".dungeon-route-summary")?.textContent).toContain("240, 73")
    click('.navigator-form button[type="submit"]')
    expect(apply).toHaveBeenLastCalledWith({
      placement: "dungeon_outside_entrance", route: "britannia", locationId: 33,
      expectedLocationId: 0, expectedFloor: 0, expectedX: 100, expectedY: 101,
      expectedSourceFingerprint: "world-source-1",
    })

    setSelect('[aria-label="Dungeon entry route"]', "underworld")
    await nextTick()
    click('input[name="world-placement"][value="dungeon_underworld_exit"]')
    await nextTick()
    state.data = worldStatus({ sourceFingerprint: "world-source-2", x: 102, y: 103 })
    await nextTick()
    expect(root.querySelector<HTMLInputElement>('input[value="dungeon_underworld_exit"]')?.checked).toBe(true)
    expect(root.querySelector(".dungeon-route-summary")?.textContent).toContain("UNDERWORLD EXIT LADDER")
    expect(root.querySelector(".dungeon-route-summary")?.textContent).toContain("7, 7")
    expect(root.querySelector(".dungeon-route-summary")?.textContent).toContain("FacingWest")
    click('.navigator-form button[type="submit"]')
    expect(apply).toHaveBeenLastCalledWith({
      placement: "dungeon_underworld_exit", route: "underworld", locationId: 33,
      expectedLocationId: 0, expectedFloor: 0, expectedX: 102, expectedY: 103,
      expectedSourceFingerprint: "world-source-2",
    })

    click('[aria-label="Show all world destinations"]')
    await nextTick()
    click('[data-world-destination="40"]')
    await nextTick()
    expect(select('[aria-label="Dungeon entry route"]').value).toBe("underworld")
    expect(root.querySelector<HTMLInputElement>('input[value="dungeon_underworld_exit"]')?.disabled).toBe(true)
    expect(root.querySelector(".dungeon-route-targets")?.textContent).toContain("no down-ladder Underworld endpoint")
  })

	it("preserves an active location search through polling and cancels exact matches with Escape", async () => {
		const apply = vi.fn()
		const state = reactive<{ data: WorldPositionStatusData }>({ data: worldStatus() })
		mount(() => h(WorldNavigator, {
			data: state.data, pending: false, landmarks: [], pendingLandmark: false, onApply: apply,
		}))
		await nextTick()
		const location = root.querySelector<HTMLInputElement>('[aria-label="World destination"]')
		if (!location) throw new Error("missing world destination combobox")

		location.click()
		await nextTick()
		expect(location.getAttribute("aria-controls")).toBe("world-destination-options")
		setInput(location, "d")
		await nextTick()
		location.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }))
		state.data = worldStatus({ sourceFingerprint: "world-source-2", x: 102, y: 103 })
		await nextTick()
		expect(location.value).toBe("d")
		expect(location.getAttribute("aria-activedescendant")).toBe("world-destination-40")
		location.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
		await nextTick()
		expect(location.value).toBe("Doom")
		expect(location.hasAttribute("aria-controls")).toBe(false)
		location.click()
		await nextTick()
		setInput(location, "doom")
		location.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
		await nextTick()
		expect(location.value).toBe("Doom")

		location.click()
		await nextTick()
		setInput(location, "Deceit")
		location.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
		await nextTick()
		expect(location.value).toBe("Doom")
		expect(root.querySelector('[aria-label="World destinations"]')).toBeNull()
		const home = new KeyboardEvent("keydown", { key: "Home", bubbles: true, cancelable: true })
		location.dispatchEvent(home)
		expect(home.defaultPrevented).toBe(false)

		click('.navigator-form button[type="submit"]')
		expect(apply).toHaveBeenCalledWith(expect.objectContaining({ locationId: 40, expectedSourceFingerprint: "world-source-2" }))
	})
})

describe("Dungeon Explorer", () => {
  it("renders Y-major semantic cells and preserves then reconciles its exact typed setup draft", async () => {
    const apply = vi.fn()
    const state = reactive<{ data: DungeonExplorerStatusData; pending: boolean }>({ data: dungeonStatus(), pending: false })
    mount(() => h(DungeonExplorer, {
      data: state.data,
      pending: state.pending,
      dungeons,
      facings,
      onApply: apply,
    }))
    await nextTick()

    const cells = [...root.querySelectorAll<HTMLElement>(".dungeon-grid [role=gridcell]")]
    expect(cells).toHaveLength(64)
    expect(cells.slice(0, 10).map((cell) => `${cell.dataset.x},${cell.dataset.y}`)).toEqual([
      "0,0", "1,0", "2,0", "3,0", "4,0", "5,0", "6,0", "7,0", "0,1", "1,1",
    ])
    click('.dungeon-grid [data-cell-index="18"]')
    await nextTick()
    expect(root.querySelector(".dungeon-cell-inspector")?.textContent).toContain("0xD8")
    expect(root.querySelector(".dungeon-cell-inspector")?.textContent).toContain("hidden_door #13")
    expect(root.querySelector(".dungeon-cell-inspector")?.textContent).toContain("Payload8")
    expect(root.querySelector(".dungeon-cell-inspector")?.textContent).toContain("Bit 0x08SET")

    setSelect('[aria-label="Dungeon"]', "40")
    setSelect('[aria-label="Dungeon floor"]', "7")
    setSelect('[aria-label="Dungeon facing"]', "west")
    setSelect('[aria-label="Dungeon lighting"]', "false")
    await nextTick()
    state.data = dungeonStatus({ sourceFingerprint: "dungeon-source-2" })
    await nextTick()
    expect(select('[aria-label="Dungeon"]').value).toBe("40")
    expect(select('[aria-label="Dungeon floor"]').value).toBe("7")
    expect(inputs(".dungeon-setup-form input").map((input) => input.value)).toEqual(["2", "2"])

    click('.dungeon-setup-form button[type="submit"]')
    expect(apply).toHaveBeenCalledWith({
      dungeon: "40", floor: 7, x: 2, y: 2, facing: "west", lit: false,
      expectedLocationId: 33, expectedFloor: 2, expectedX: 3, expectedY: 4, expectedDungeonRevision: 12,
      expectedSourceFingerprint: "dungeon-source-2",
    })

    state.data = dungeonStatus({ locationId: 40, locationName: "Doom", floor: 7, displayLevel: 8, x: 2, y: 2, facingId: 3, facingLabel: "West", lastDirectionId: 3, lastDirectionLabel: "West", lit: false, revision: 0 })
    await nextTick()
    state.data = dungeonStatus({ locationId: 40, locationName: "Doom", floor: 7, displayLevel: 8, x: 5, y: 6, facingId: 3, facingLabel: "West", lastDirectionId: 3, lastDirectionLabel: "West", lit: false, revision: 1 })
    await nextTick()
    expect(inputs(".dungeon-setup-form input").map((input) => input.value)).toEqual(["5", "6"])
  })

  it("shows the setup blocker and disables apply", async () => {
    const state = dungeonStatus({ setupAvailable: false, setupUnavailableReason: "a dungeon command cycle is active" })
    mount(() => h(DungeonExplorer, { data: state, pending: false, dungeons, facings }))
    await nextTick()

    expect(root.querySelector(".disabled-reason")?.textContent).toContain("dungeon command cycle")
    expect(root.querySelector<HTMLButtonElement>('.dungeon-setup-form button[type="submit"]')?.disabled).toBe(true)
  })
})

const dungeonBounds = { minimumX: 0, maximumX: 7, minimumY: 0, maximumY: 7 }
const destinations = [{
  locationId: 0, locationName: "Britannia / Underworld", mapType: "large", floors: [-1, 0],
  bounds: { minimumX: 0, maximumX: 255, minimumY: 0, maximumY: 255 }, defaultSetup: { floor: 0, x: 80, y: 100 },
}, {
  locationId: 2, locationName: "Britain", mapType: "small", floors: [-1, 0, 1],
  bounds: { minimumX: 0, maximumX: 31, minimumY: 0, maximumY: 31 }, defaultSetup: { floor: 0, x: 15, y: 30 },
  worldEntrance: { floor: 0, x: 85, y: 107 },
}, {
  locationId: 25, locationName: "Ararat", mapType: "small", floors: [0],
  bounds: { minimumX: 0, maximumX: 31, minimumY: 0, maximumY: 31 }, defaultSetup: { floor: 0, x: 15, y: 30 },
  worldEntrance: { floor: -1, x: 42, y: 43 },
}, {
  locationId: 33, locationName: "Deceit", mapType: "dungeon", floors: [0, 1, 2, 3, 4, 5, 6, 7],
  bounds: dungeonBounds, defaultSetup: { floor: 0, x: 1, y: 1 }, placementTargets: [
    routeTarget("dungeon_outside_entrance", "Outside dungeon entrance", "britannia", "Britannia", 0, { floor: 0, x: 240, y: 73 }),
    routeTarget("dungeon_inside_start", "Inside dungeon start", "britannia", "Britannia", 33, { floor: 0, x: 1, y: 1 }, "East", "Arrived from above"),
    routeTarget("dungeon_outside_entrance", "Outside dungeon entrance", "underworld", "Underworld", 0, { floor: -1, x: 240, y: 73 }),
    routeTarget("dungeon_inside_start", "Inside dungeon start", "underworld", "Underworld", 33, { floor: 7, x: 7, y: 7 }, "West", "Arrived from below"),
    routeTarget("dungeon_underworld_exit", "Underworld exit ladder", "underworld", "Underworld", 33, { floor: 7, x: 7, y: 7 }, "West", "West"),
  ],
}, {
  locationId: 40, locationName: "Doom", mapType: "dungeon", floors: [0, 1, 2, 3, 4, 5, 6, 7],
  bounds: dungeonBounds, defaultSetup: { floor: 0, x: 1, y: 1 }, placementTargets: [
    routeTarget("dungeon_outside_entrance", "Outside dungeon entrance", "underworld", "Underworld", 0, { floor: -1, x: 128, y: 128 }),
    routeTarget("dungeon_inside_start", "Inside dungeon start", "underworld", "Underworld", 40, { floor: 0, x: 1, y: 1 }, "East", "Arrived from above"),
    { placement: "dungeon_underworld_exit", placementLabel: "Underworld exit ladder", route: "underworld", routeLabel: "Underworld", available: false, unavailableReason: "Doom has no down-ladder Underworld endpoint; its floor-zero Up route exits to Britannia", locationId: 40 },
  ],
}]
const dungeons: Choice[] = [
  { value: "33", label: "Deceit" }, { value: "34", label: "Despise" }, { value: "35", label: "Destard" }, { value: "36", label: "Wrong" },
  { value: "37", label: "Covetous" }, { value: "38", label: "Shame" }, { value: "39", label: "Hythloth" }, { value: "40", label: "Doom" },
]
const facings: Choice[] = [
  { value: "north", label: "North" }, { value: "east", label: "East" }, { value: "south", label: "South" }, { value: "west", label: "West" },
]

function worldStatus(overrides: Partial<WorldPositionStatusData> = {}): WorldPositionStatusData {
  return {
    sourceFingerprint: "world-source-1",
    locationId: 0, locationName: "Britannia / Underworld", mapType: "large", floor: 0, x: 100, y: 101,
    bounds: { minimumX: 0, maximumX: 255, minimumY: 0, maximumY: 255 }, destinations, ...overrides,
  }
}

function dungeonStatus(overrides: Partial<DungeonExplorerStatusData> = {}): DungeonExplorerStatusData {
  const cells = Array.from({ length: 64 }, (_, index) => {
    const x = index % 8
    const y = Math.floor(index / 8)
    if (index === 18) return { x, y, raw: 0xd8, kindId: 13, kindName: "hidden_door", payload: 8, bit08: true, player: false }
    return { x, y, raw: 0, kindId: 0, kindName: "passage", payload: 0, bit08: false, player: x === 3 && y === 4 }
  })
  return {
    sourceFingerprint: "dungeon-source-1",
    active: true, setupAvailable: true, locationId: 33, locationName: "Deceit", floor: 2, displayLevel: 3,
    x: 3, y: 4, facingId: 2, facingLabel: "South", lastDirectionId: 2, lastDirectionLabel: "South",
    lit: true, torchMinutesRemaining: 17, magicLightMinutesRemaining: 23, revision: 12, cells, ...overrides,
  }
}

function routeTarget(
  placement: string,
  placementLabel: string,
  route: string,
  routeLabel: string,
  locationId: number,
  setup: { floor: number; x: number; y: number },
  facingLabel?: string,
  lastDirectionLabel?: string,
): NonNullable<WorldPositionStatusData["destinations"][number]["placementTargets"]>[number] {
  return { placement, placementLabel, route, routeLabel, available: true, locationId, setup, facingLabel, lastDirectionLabel }
}

function mount(render: () => ReturnType<typeof h>): void {
  root = document.createElement("div")
  document.body.append(root)
  app = createApp({ setup: () => render })
  app.mount(root)
}

function select(selector: string): HTMLSelectElement {
  const control = root.querySelector<HTMLSelectElement>(selector)
  if (!control) throw new Error(`missing select: ${selector}`)
  return control
}

function inputs(selector: string): HTMLInputElement[] {
  return [...root.querySelectorAll<HTMLInputElement>(selector)]
}

function setSelect(selector: string, value: string): void {
  const control = select(selector)
  control.value = value
  control.dispatchEvent(new Event("change", { bubbles: true }))
}

function setInput(control: HTMLInputElement | undefined, value: string): void {
  if (!control) throw new Error("missing input")
  control.value = value
  control.dispatchEvent(new Event("input", { bubbles: true }))
}

function click(selector: string): void {
  const control = root.querySelector<HTMLElement>(selector)
  if (!control) throw new Error(`missing control: ${selector}`)
  control.click()
}
