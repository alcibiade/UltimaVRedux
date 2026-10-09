import { createApp, nextTick, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import ReduxModificationsConsole from "./ReduxModificationsConsole.vue"

describe("ReduxModificationsConsole", () => {
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

  it("shows the Redux default and emits the explicit original-mode value", async () => {
    const setRepeatedDivision = vi.fn()
    const setSmoothSurfaceMovement = vi.fn()
    const setFlexibleSextantUse = vi.fn()
    app = createApp(ReduxModificationsConsole, {
      data: {
        combat: { allowRepeatedMonsterDivision: false },
        presentation: { smoothSurfaceMovement: false },
        navigation: { flexibleSextantUse: false },
        dungeons: { persistHiddenDungeonDiscoveries: false, showSubduedUnlitDungeons: false },
      }, pending: false,
      onSetRepeatedDivision: setRepeatedDivision,
      onSetSmoothSurfaceMovement: setSmoothSurfaceMovement,
      onSetFlexibleSextantUse: setFlexibleSextantUse,
    })
    app.mount(root)

    expect(root.textContent).toContain("REDUX LIMITED")
    expect(root.textContent).toContain("persist across restarts")
    root.querySelector<HTMLButtonElement>("[data-redux-repeated-division]")?.click()
    await nextTick()
    expect(setRepeatedDivision).toHaveBeenCalledWith(true)
    const smooth = root.querySelector<HTMLButtonElement>("[data-redux-smooth-surface-movement]")!
    expect(smooth.getAttribute("aria-checked")).toBe("false")
    expect(root.textContent).toContain("Gameplay.Presentation.SmoothSurfaceMovement")
    expect(root.textContent).toContain("walking, riding, flying or boating")
    expect(root.textContent).toContain("Mounts animate 25% faster")
    expect(root.textContent).toContain("automatic sailing completes each scroll within its existing cadence")
    expect(root.textContent).toContain("Gameplay timing, combat and dungeons are unchanged")
    smooth.click()
    await nextTick()
    expect(setSmoothSurfaceMovement).toHaveBeenCalledWith(true)
    const sextant = root.querySelector<HTMLButtonElement>("[data-redux-flexible-sextant-use]")!
    expect(sextant.getAttribute("aria-checked")).toBe("false")
    expect(root.textContent).toContain("Flexible sextant use")
    expect(root.textContent).toContain("ORIGINAL NIGHT WORLD ONLY")
    expect(root.textContent).toContain("including cellars")
    expect(root.textContent).toContain("Dungeons and the Underworld remain excluded")
    expect(root.textContent).toContain("a sextant must still be owned")
    sextant.click()
    await nextTick()
    expect(setFlexibleSextantUse).toHaveBeenCalledWith(true)
  })

  it("shows original mode and disables mutations while pending", () => {
    app = createApp(ReduxModificationsConsole, {
      data: {
        combat: { allowRepeatedMonsterDivision: true },
        presentation: { smoothSurfaceMovement: true },
        navigation: { flexibleSextantUse: true },
        dungeons: { persistHiddenDungeonDiscoveries: true, showSubduedUnlitDungeons: true },
      }, pending: true,
    })
    app.mount(root)

    expect(root.textContent).toContain("ORIGINAL REPEATED")
    expect(root.querySelector<HTMLButtonElement>("[data-redux-repeated-division]")?.disabled).toBe(true)
    expect(root.querySelector<HTMLButtonElement>("[data-redux-smooth-surface-movement]")?.disabled).toBe(true)
    expect(root.querySelector<HTMLButtonElement>("[data-redux-flexible-sextant-use]")?.disabled).toBe(true)
  })

  it("emits false when disabling smooth surface movement", async () => {
    const setSmoothSurfaceMovement = vi.fn()
    const setFlexibleSextantUse = vi.fn()
    app = createApp(ReduxModificationsConsole, {
      data: {
        combat: { allowRepeatedMonsterDivision: false },
        dungeons: { persistHiddenDungeonDiscoveries: true, showSubduedUnlitDungeons: true },
        presentation: { smoothSurfaceMovement: true },
        navigation: { flexibleSextantUse: true },
      }, pending: false,
      onSetSmoothSurfaceMovement: setSmoothSurfaceMovement,
      onSetFlexibleSextantUse: setFlexibleSextantUse,
    })
    app.mount(root)
    const smooth = root.querySelector<HTMLButtonElement>("[data-redux-smooth-surface-movement]")!
    expect(smooth.getAttribute("aria-checked")).toBe("true")
    smooth.click()
    await nextTick()
    expect(setSmoothSurfaceMovement).toHaveBeenCalledWith(false)
    expect(root.textContent).toContain("REDUX FLEXIBLE")
    const sextant = root.querySelector<HTMLButtonElement>("[data-redux-flexible-sextant-use]")!
    expect(sextant.getAttribute("aria-checked")).toBe("true")
    sextant.click()
    await nextTick()
    expect(setFlexibleSextantUse).toHaveBeenCalledWith(false)
  })
})
