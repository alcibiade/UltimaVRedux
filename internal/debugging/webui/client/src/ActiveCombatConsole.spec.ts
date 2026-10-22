import { createApp, h, nextTick, reactive } from "vue"
import { expect, it, vi } from "vitest"
import ActiveCombatConsole from "./ActiveCombatConsole.vue"
import type { ActiveCombatStatusData } from "./approved_tool_views"
import catalog from "./initial_catalog.fixture.json"
import type { CommandDefinition } from "./protocol"

function combatData(): ActiveCombatStatusData {
  return {
    active: true, snapshotFingerprint: "combat-1", encounterId: "9",
    monsterTypes: [{ id: 32, name: "Orc" }],
    map: {
      name: "Glade", index: 2, territory: "britannia", entryDirection: "north",
      victory: false, turnStarted: true, pendingAdvanceTurn: false, soloMoveMode: false,
      currentPartyIndex: 0, activeMonsterIndex: -1, targetPartyIndex: -1, objectCount: 0, cells: [],
    },
    party: [], monsters: [], forecast: [], activeMonsterCount: 0, activeHostileMonsterCount: 0,
    totalMonsterCount: 0, fixedDungeonRoom: false, defeatedExperience: 0, rngSeed: 42,
    workflowActive: false, presentationSpeedPercent: 100, configurationEditing: false,
  }
}

it("routes Active Combat speed through the string-draft preference control", async () => {
  const mutate = vi.fn()
  const props = reactive({ data: combatData(), commands: catalog as CommandDefinition[], pending: false, onMutate: mutate })
  const root = document.createElement("div")
  const app = createApp({ setup: () => () => h(ActiveCombatConsole, props) })
  app.mount(root)
  try {
    const control = root.querySelector('[data-preference="combat.presentation.speed.set"]')!
    const select = control.querySelector("select")!
    const button = control.querySelector("button")!
    expect(button.disabled).toBe(true)
    button.click()
    expect(mutate).not.toHaveBeenCalled()
    select.value = "2000"
    select.dispatchEvent(new Event("change"))
    await nextTick()
    props.data.presentationSpeedPercent = 200
    await nextTick()
    expect(select.value).toBe("2000")
    expect(button.disabled).toBe(false)
    button.click()
    expect(mutate).toHaveBeenCalledWith({ commandID: "combat.presentation.speed.set", argumentsByID: { percent: "2000" } })
    props.data.configurationEditing = true
    await nextTick()
    expect(button.disabled).toBe(true)
  } finally { app.unmount() }
})

it("keeps an optional spawn seed textual and distinguishes zero from empty", async () => {
  const mutate = vi.fn()
  const root = document.createElement("div")
  const app = createApp(ActiveCombatConsole, { data: combatData(), commands: catalog as CommandDefinition[], pending: false, onMutate: mutate })
  app.mount(root)
  try {
    const input = root.querySelector<HTMLInputElement>("[data-combat-spawn-seed]")!
    for (const seed of ["0", "123", ""]) {
      input.value = seed
      input.dispatchEvent(new Event("input"))
      await nextTick()
      const review = root.querySelector<HTMLButtonElement>("[data-combat-spawn-review]")!
      expect(review.disabled).toBe(false)
      review.click()
      await nextTick()
      root.querySelector<HTMLButtonElement>("[data-combat-spawn-apply]")!.click()
      await nextTick()
      expect(mutate).toHaveBeenLastCalledWith({
        commandID: "combat.monster.spawn",
        argumentsByID: {
          encounterId: "9", expectedSnapshot: "combat-1", monsterType: "32", countMode: "automatic", friends: "true",
          ...(seed === "" ? {} : { seed }),
        },
      })
    }
  } finally { app.unmount() }
})
