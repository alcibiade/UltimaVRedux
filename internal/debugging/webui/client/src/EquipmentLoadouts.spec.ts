import { createApp, nextTick, ref, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it } from "vitest"

import initialCatalog from "./initial_catalog.fixture.json"
import EquipmentLoadouts from "./EquipmentLoadouts.vue"
import type { CommandDefinition } from "./protocol"

let app: VueApp<Element> | undefined
let root: HTMLElement

const commands = initialCatalog as CommandDefinition[]
const loadoutCommand = commands.find((command) => command.id === "party.equipment.loadout.update")!
const character = {
  value: "0.QXZhdGFy",
  label: "Position 1 · Avatar",
  metadata: { partyPosition: 1, name: "Avatar", class: "Avatar", status: "Good", strength: 20, dexterity: 20, intelligence: 20, currentHp: 100, maximumHp: 100, currentMp: 20, experience: 500, level: 5, karma: 50 },
}
const secondCharacter = {
  value: "1.U2hhbWlubw",
  label: "Position 2 · Shamino",
  metadata: { partyPosition: 2, name: "Shamino", class: "Fighter", status: "Good", strength: 18, dexterity: 20, intelligence: 15, currentHp: 90, maximumHp: 90, currentMp: 15, experience: 400, level: 4, karma: 50 },
}
const allCharacters = { value: "all", label: "All active party members" }
const locker = {
  snapshotFingerprint: "locker-1", provisions: [], reagents: [], potions: [], scrolls: [], usablePresetReview: { reviewFingerprint: "", changes: [] },
  equipment: [
    { id: 0, name: "Leather Helm", category: "helm", slot: "Helm", ammo: false, quantity: 0, maximum: 99 },
    { id: 16, name: "Dagger", category: "weapon", slot: "Weapon", ammo: false, quantity: 2, maximum: 99 },
    { id: 30, name: "Long Sword", category: "weapon", slot: "Weapon", ammo: false, quantity: 1, maximum: 99 },
    { id: 27, name: "Arrows", category: "ammo", slot: "Weapon", ammo: true, quantity: 20, maximum: 99 },
  ],
  equipped: [
    { rosterIndex: 0, name: "Avatar", slots: [{ slot: "Weapon", equipmentId: 16, equipmentName: "Dagger" }, { slot: "Helm", equipmentId: 0, equipmentName: "Leather Helm" }] },
    { rosterIndex: 1, name: "Shamino", slots: [{ slot: "Weapon", equipmentId: -1, equipmentName: "" }, { slot: "Helm", equipmentId: 0, equipmentName: "Leather Helm" }] },
  ],
}

beforeEach(() => {
  root = document.createElement("div")
  document.body.append(root)
})

afterEach(() => {
  app?.unmount()
  app = undefined
  root.remove()
})

describe("EquipmentLoadouts", () => {
  it.each([false, true])("cleans only submitted A's exact draft version on delayed success; A edited again: %s", async (editAAgain) => {
    const target = ref(character.value)
    const pending = ref(false)
    const successToken = ref<string>()
    const submissions: Record<string, string>[] = []
    const roster = { fingerprint: "roster-1", characters: [], roster: [
      { rosterIndex: 0, name: "Avatar", class: "Avatar", status: "good", active: true },
      { rosterIndex: 1, name: "Shamino", class: "Fighter", status: "good", active: true },
    ] }
    app = createApp({
      components: { EquipmentLoadouts },
      setup: () => ({ target, pending, successToken, loadoutCommand, locker, roster,
        optionSets: { "party.equipment-loadout-targets": [character, secondCharacter] },
        submit: (args: Record<string, string>) => { submissions.push(args); pending.value = true },
      }),
      template: `<EquipmentLoadouts :command="loadoutCommand" :draft="{ character: target }" :option-sets="optionSets" :locker="locker" :roster="roster" :pending="pending" :success-token="successToken" @update-character="target = $event" @submit="submit" />`,
    })
    app.mount(root)
    setSelect("#loadout-weapon-action", "manufacture")
    await nextTick()
    click("[data-loadout-review-button]")
    await nextTick()
    click("[data-loadout-submit]")
    await nextTick()
    expect(submissions).toHaveLength(1)
    expect(submissions[0].character).toBe(character.value)
    if (editAAgain) {
      setSelect("#loadout-weapon-action", "unequip")
      await nextTick()
      // Even editing back to the submitted value is a newer desired version.
      setSelect("#loadout-weapon-action", "manufacture")
      await nextTick()
    }
    click(`[data-party-character="${secondCharacter.value}"]`)
    await nextTick()
    setSelect("#loadout-helm-action", "unequip")
    await nextTick()
    successToken.value = "success-A"
    pending.value = false
    await nextTick()
    expect(root.querySelector<HTMLSelectElement>("#loadout-helm-action")?.value).toBe("unequip")
    click(`[data-party-character="${character.value}"]`)
    await nextTick()
    expect(root.querySelector<HTMLSelectElement>("#loadout-weapon-action")?.value).toBe(editAAgain ? "manufacture" : "unchanged")
    click(`[data-party-character="${secondCharacter.value}"]`)
    await nextTick()
    successToken.value = "success-A"
    await nextTick()
    expect(root.querySelector<HTMLSelectElement>("#loadout-helm-action")?.value).toBe("unequip")
  })

  it("renders all slots and requires an explicit review that can be cancelled", async () => {
    app = createApp(EquipmentLoadouts, {
      command: loadoutCommand,
      draft: { character: character.value },
      optionSets: { "party.equipment-loadout-targets": [allCharacters, character], "party.active-positions": [character] }, pending: false, locker,
      roster: { fingerprint: "roster-1", characters: [], roster: [{ rosterIndex: 0, name: "Avatar", class: "Avatar", status: "good", active: true }] },
    })
    app.mount(root)

    expect(root.querySelectorAll("[data-loadout-slot]")).toHaveLength(6)
    expect(root.querySelector('[data-loadout-slot="weapon"]')?.textContent).toContain("Current: Dagger")
    expect(root.querySelector('[data-loadout-slot="weapon"]')?.textContent).toContain("Carried: 2")
    setSelect("#loadout-weapon-action", "inventory")
    await nextTick()
    expect(root.querySelector("#loadout-weapon-equipment")?.textContent).toContain("Dagger")
    expect(root.querySelector("#loadout-weapon-equipment")?.textContent).not.toContain("Arrows")

    click("[data-loadout-review-button]")
    await nextTick()
    expect(root.querySelector("[data-loadout-review]")?.textContent).toContain("Review atomic loadout")
    expect(root.querySelector("[data-loadout-review]")?.textContent).toContain("created directly and do not consume carried inventory")
    click("[data-loadout-cancel]")
    await nextTick()
    expect(root.querySelector("[data-loadout-review]")).toBeNull()
    expect((root.querySelector("#loadout-weapon-action") as HTMLSelectElement).value).toBe("inventory")
  })

  it("shows common and mixed all-party slots and enforces aggregate inventory visibility", async () => {
    app = createApp(EquipmentLoadouts, {
      command: loadoutCommand,
      draft: { character: "all" },
      optionSets: { "party.equipment-loadout-targets": [allCharacters, character, secondCharacter] }, pending: false, locker,
      roster: { fingerprint: "roster-1", characters: [], roster: [
        { rosterIndex: 0, name: "Avatar", class: "Avatar", status: "good", active: true },
        { rosterIndex: 1, name: "Shamino", class: "Fighter", status: "good", active: true },
      ] },
    })
    app.mount(root)

    expect(root.querySelector('[data-party-character="all"]')?.textContent).toContain("2 current targets")
    expect(root.querySelector('[data-loadout-slot="helm"]')?.textContent).toContain("Current: Leather Helm")
    expect(root.querySelector('[data-loadout-slot="weapon"]')?.textContent).toContain("Current: Mixed")
    setSelect("#loadout-weapon-action", "inventory")
    await nextTick()
    expect(root.querySelector("#loadout-weapon-equipment")?.textContent).toContain("Dagger")
    expect(root.querySelector("#loadout-weapon-equipment")?.textContent).not.toContain("Long Sword")
    click("[data-loadout-review-button]")
    await nextTick()
    expect(root.querySelector("[data-loadout-review]")?.textContent).toContain("2 active party members")
    expect(root.querySelector("[data-loadout-review]")?.textContent).toContain("(1 required)")
    expect(root.querySelector("[data-loadout-review]")?.textContent).toContain("manufacture creates one item per member")
  })
})

function click(selector: string): void {
  const element = root.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`missing ${selector}`)
  element.click()
}

function setSelect(selector: string, value: string): void {
  const element = root.querySelector<HTMLSelectElement>(selector)
  if (!element) throw new Error(`missing ${selector}`)
  element.value = value
  element.dispatchEvent(new Event("change", { bubbles: true }))
}
