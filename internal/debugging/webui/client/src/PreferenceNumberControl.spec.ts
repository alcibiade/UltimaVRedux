import { createApp, h, nextTick, reactive } from "vue"
import { expect, it, vi } from "vitest"
import PreferenceNumberControl from "./PreferenceNumberControl.vue"
import catalog from "./initial_catalog.fixture.json"
import type { CommandDefinition } from "./protocol"

it("validates speed steps, retains drafts on refresh, and emits only allowed values", async () => {
  const commandID = "combat.presentation.speed.set"
  const apply = vi.fn()
  const props = reactive({ commandId: commandID, command: (catalog as CommandDefinition[]).find(command => command.id === commandID), value: 100, disabled: false, onApply: apply })
  const root = document.createElement("div")
  const app = createApp({ setup: () => () => h(PreferenceNumberControl, props) })
  app.mount(root)
  try {
    const input = root.querySelector("input")!
    const button = root.querySelector("button")!
    input.value = "75"
    input.dispatchEvent(new Event("input"))
    await nextTick()
    expect(button.disabled).toBe(true)
    expect(root.textContent).toContain("steps of 50")
    input.value = "250"
    input.dispatchEvent(new Event("input"))
    await nextTick()
    props.value = 200
    await nextTick()
    expect(input.value).toBe("250")
    button.click()
    expect(apply).toHaveBeenCalledWith({ percent: "250" })
    props.disabled = true
    await nextTick()
    button.click()
    expect(apply).toHaveBeenCalledTimes(1)
  } finally { app.unmount() }
})

it.each([
  ["combat.presentation.speed.set", ""],
  ["combat.presentation.speed.set", "not-a-number"],
  ["combat.presentation.speed.set", "50.5"],
  ["combat.presentation.speed.set", "49"],
  ["combat.presentation.speed.set", "401"],
  ["redux.presentation.action-popups.set", ""],
  ["redux.presentation.action-popups.set", "0.5"],
  ["redux.presentation.action-popups.set", "-1"],
  ["redux.presentation.action-popups.set", "4"],
])("rejects invalid %s draft %j without coercing blanks to zero", async (commandId, value) => {
  const apply = vi.fn()
  const root = document.createElement("div")
  const command = (catalog as CommandDefinition[]).find(command => command.id === commandId)!
  const app = createApp(PreferenceNumberControl, { commandId, command, value: Number(command.parameters![0]!.maximum), disabled: false, onApply: apply })
  app.mount(root)
  try {
    const input = root.querySelector("input")!
    input.value = value
    input.dispatchEvent(new Event("input"))
    await nextTick()
    const button = root.querySelector("button")!
    expect(button.disabled).toBe(true)
    expect(root.querySelector('[role="status"]')?.textContent).toBeTruthy()
    button.click()
    expect(apply).not.toHaveBeenCalled()
  } finally { app.unmount() }
})

it.each([
  ["combat.presentation.speed.set", "percent", 100, 50, 400],
  ["redux.presentation.action-popups.set", "count", 1, 0, 3],
] as const)("retains blank and boundary drafts through %s refreshes", async (commandId, parameterID, initial, minimum, maximum) => {
  const apply = vi.fn()
  const props = reactive({ commandId, command: (catalog as CommandDefinition[]).find(command => command.id === commandId), value: Number(initial), disabled: false, onApply: apply })
  const root = document.createElement("div")
  const app = createApp({ setup: () => () => h(PreferenceNumberControl, props) })
  app.mount(root)
  try {
    const input = root.querySelector("input")!
    const button = root.querySelector("button")!
    input.value = ""
    input.dispatchEvent(new Event("input"))
    await nextTick()
    props.value = maximum
    await nextTick()
    expect(input.value).toBe("")
    expect(button.disabled).toBe(true)
    button.click()
    expect(apply).not.toHaveBeenCalled()

    input.value = String(minimum)
    input.dispatchEvent(new Event("input"))
    await nextTick()
    expect(button.disabled).toBe(false)
    props.value = initial
    await nextTick()
    expect(input.value).toBe(String(minimum))
    button.click()
    expect(apply).toHaveBeenLastCalledWith({ [parameterID]: String(minimum) })

    props.value = minimum
    await nextTick()
    expect(button.disabled).toBe(true)
    props.value = maximum
    await nextTick()
    expect(input.value).toBe(String(maximum))
    expect(button.disabled).toBe(true)

    input.value = String(initial)
    input.dispatchEvent(new Event("input"))
    await nextTick()
    props.value = minimum
    await nextTick()
    input.value = String(maximum)
    input.dispatchEvent(new Event("input"))
    await nextTick()
    expect(button.disabled).toBe(false)
    button.click()
    expect(apply).toHaveBeenLastCalledWith({ [parameterID]: String(maximum) })
    expect(apply).toHaveBeenCalledTimes(2)
  } finally { app.unmount() }
})
