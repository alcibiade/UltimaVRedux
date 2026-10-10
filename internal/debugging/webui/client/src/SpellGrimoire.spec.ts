import { createApp, nextTick, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import SpellGrimoire, { type SpellGrimoireEntry } from "./SpellGrimoire.vue"

let app: VueApp<Element> | undefined
let root: HTMLElement

const entry = (overrides: Partial<SpellGrimoireEntry> = {}): SpellGrimoireEntry => ({
  id: 0, circle: 1, name: "In Lor", targeting: "Self", context: "Non-Combat",
  description: "Creates magical light", reagents: ["Sulfurous Ash"], status: "Complete with deviation",
  originalBehavior: "Light for 100 minutes.", reduxBehavior: "Light for 100 turns.",
  originalTargeting: "Automatic party effect.", reduxTargeting: "Self; immediate.",
  actorVisual: "None.", screenVisual: "Inverted flash.", statusIndicator: "Light icon.",
  turnOrder: "Advances one minute.", combatEvents: "Not combat-capable.", areaEffect: "Party/global.",
  failure: "Common validation.", formula: "Set duration to 100.", audio: "SoundMagic at flash start.",
  knownIssues: ["Minutes are modeled as turns."], opportunities: ["Test expiration."],
  codeReferences: ["GameScene.thenCastInLor (cmd/ultimav/gamescene_cast_spell.go)"],
  originalReferences: ["SPELL-0002", "SPELL-0004"], ...overrides,
})

beforeEach(() => { root = document.createElement("div"); document.body.append(root) })
afterEach(() => { app?.unmount(); app = undefined; root.remove() })

describe("Spell Grimoire", () => {
  it("groups spells by circle and shows reagent recipes prominently", () => {
    app = createApp(SpellGrimoire, { data: [
      entry(),
      entry({ id: 7, circle: 2, name: "An Nox", description: "Cures poison.", reagents: ["Garlic", "Ginseng"] }),
    ] })
    app.mount(root)

    const circles = root.querySelectorAll(".spell-circle")
    expect(circles).toHaveLength(2)
    expect(circles[0].querySelector("header")?.textContent).toContain("CIRCLE 1")
    expect(circles[1].querySelector("header")?.textContent).toContain("CIRCLE 2")

    expect(root.querySelector(".grimoire-spell-row strong")?.textContent).toBe("In Lor")
    expect(root.querySelector(".grimoire-reagent")?.textContent).toBe("Sulfurous Ash")

    const secondRow = circles[1].querySelector(".grimoire-spell-row")
    const reagents = secondRow?.querySelectorAll(".grimoire-reagent")
    expect(reagents).toHaveLength(2)
    expect(reagents?.[0].textContent).toBe("Garlic")
    expect(reagents?.[1].textContent).toBe("Ginseng")
  })

  it("filters by spell name, reagent, or description", async () => {
    app = createApp(SpellGrimoire, { data: [
      entry({ name: "In Lor", reagents: ["Sulfurous Ash"] }),
      entry({ id: 7, circle: 2, name: "An Nox", reagents: ["Garlic", "Ginseng"] }),
    ] })
    app.mount(root)

    const input = root.querySelector<HTMLInputElement>('input[aria-label="Filter spells by name, reagent, or description"]')!
    input.value = "garlic"
    input.dispatchEvent(new Event("input"))
    await nextTick()

    expect(root.querySelector(".grimoire-spell-row strong")?.textContent).toBe("An Nox")
    const rows = root.querySelectorAll(".grimoire-spell-row")
    expect(rows).toHaveLength(1)
  })

  it("shows the complete audit in a collapsible section", () => {
    app = createApp(SpellGrimoire, { data: [
      entry({ id: 13, circle: 3, name: "Vas Flam", reduxTargeting: "Crosshair, range 15", formula: "Red fireball always hits.", reagents: ["Sulfurous Ash", "Black Pearl"] }),
    ] })
    app.mount(root)

    const details = root.querySelector(".grimoire-audit")
    expect(details).not.toBeNull()
    expect(details?.querySelector("summary")?.textContent).toContain("Original vs Redux audit")
    expect(details?.textContent).toContain("SoundMagic at flash start")
    expect(details?.textContent).toContain("GameScene.thenCastInLor")
    expect(details?.textContent).toContain("PUBLIC GAME FACTS RULES")
    expect(details?.textContent).toContain("SPELL-0002")
    expect(details?.textContent).toContain("SPELL-0004")
  })

  it("shows known issues and opportunities for non-combat spells", () => {
    app = createApp(SpellGrimoire, { data: [entry()] })
    app.mount(root)

    const details = root.querySelector(".grimoire-audit")
    expect(details?.textContent).toContain("Minutes are modeled as turns")
    expect(details?.textContent).toContain("Test expiration")
  })

  it("shows empty state when no data is provided", () => {
    app = createApp(SpellGrimoire, {})
    app.mount(root)

    expect(root.textContent).toContain("LOADING SPELL GRIMOIRE")
  })
})
