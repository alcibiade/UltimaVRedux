import { createApp, nextTick, type App as VueApp } from "vue"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import initialCatalog from "./initial_catalog.fixture.json"
import InventoryLocker from "./InventoryLocker.vue"
import type { InventoryLockerStatusData } from "./approved_tool_views"
import type { CommandDefinition } from "./protocol"

let app: VueApp<Element> | undefined
let root: HTMLElement
const mutate = vi.fn()
const commands = initialCatalog as CommandDefinition[]
const data: InventoryLockerStatusData = {
  snapshotFingerprint: "locker-1",
  provisions: [{ name: "gold", quantity: 12, maximum: 9999 }],
  reagents: [],
  potions: [],
  scrolls: [],
  equipment: [],
  equipped: [],
  usablePresetReview: { reviewFingerprint: "usable-1", changes: [] },
}

beforeEach(() => {
  root = document.createElement("div")
  document.body.append(root)
  mutate.mockReset()
  app = createApp(InventoryLocker, { data, commands, pending: false, onMutate: mutate })
  app.mount(root)
})

afterEach(() => {
  app?.unmount()
  app = undefined
  root.remove()
})

describe("InventoryLocker review authority", () => {
  it.each(["", "1.5", "-1", "10000"])("shows invalid target %j without granting review authority", async (value) => {
    await stageGoldTarget("20")
    await stageGoldTarget(value)
    await nextTick()
    expect(root.querySelector('[role="alert"]')?.textContent).toContain("whole quantity from 0 to 9999")
    expect(root.querySelector("[data-locker-review-apply]")).toBeNull()
    expect(mutate).not.toHaveBeenCalled()
  })
  it("replaces a target review with a preset review and dispatches only the preset", async () => {
    await stageGoldTarget("20")
    click(".locker-presets button")
    await nextTick()

    expect(root.querySelector(".locker-review")?.textContent).toContain('Apply Provision Preset "Set All Provisions to Maximum"')
    expect(reviewApply().textContent).toBe("APPLY: Set All Provisions to Maximum")
    reviewApply().click()

    expect(mutate).toHaveBeenCalledOnce()
    expect(mutate).toHaveBeenCalledWith({
      commandID: "inventory.provision.preset",
      argumentsByID: { preset: "provisions_max", expectedSnapshot: "locker-1" },
    })
  })

  it("replaces a preset review with a target review and dispatches only the target", async () => {
    click(".locker-presets button")
    await stageGoldTarget("20")
    await nextTick()

    expect(root.querySelector(".locker-review")?.textContent).toContain("Set gold from 12 to 20")
    expect(reviewApply().textContent).toBe("APPLY STAGED CHANGE")
    reviewApply().click()

    expect(mutate).toHaveBeenCalledOnce()
    expect(mutate).toHaveBeenCalledWith({
      commandID: "inventory.provision.set",
      argumentsByID: { provision: "gold", targetQuantity: "20", expectedQuantity: "12" },
    })
  })

  it("cancels the sole review and cannot submit stale authority", async () => {
    await stageGoldTarget("20")
    const staleApply = reviewApply()
    click("[data-locker-review-cancel]")
    await nextTick()

    expect(root.querySelector(".locker-review")).toBeNull()
    expect(root.querySelector("[data-locker-review-apply]")).toBeNull()
    staleApply.click()
    expect(mutate).not.toHaveBeenCalled()
  })
})

async function stageGoldTarget(value: string): Promise<void> {
  const input = root.querySelector<HTMLInputElement>('[data-locker-provision="gold"]')!
  input.value = value
  input.dispatchEvent(new Event("input", { bubbles: true }))
  await nextTick()
  input.parentElement?.querySelector<HTMLButtonElement>("button")?.click()
}

function click(selector: string): void {
  const element = root.querySelector<HTMLButtonElement>(selector)
  if (!element) throw new Error(`missing ${selector}`)
  element.click()
}

function reviewApply(): HTMLButtonElement {
  const button = root.querySelector<HTMLButtonElement>("[data-locker-review-apply]")
  if (!button) throw new Error("missing review apply button")
  return button
}
