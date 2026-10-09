import { describe, expect, it } from "vitest"

import initialCatalog from "./initial_catalog.fixture.json"
import type { InventoryLockerStatusData } from "./approved_tool_views"
import type { CommandDefinition } from "./protocol"
import {
  equipmentForSlot,
  newLoadoutDraft,
  serializeLoadoutDraft,
  stagedLoadoutSlotCount,
} from "./party_equipment_loadout"

const command = (initialCatalog as CommandDefinition[]).find((candidate) => candidate.id === "party.equipment.loadout.update")!
const locker = {
  equipment: [
    { id: 16, name: "Dagger", category: "weapon", slot: "right_hand", ammo: false, quantity: 2, maximum: 99 },
    { id: 27, name: "Arrows", category: "ammo", slot: "right_hand", ammo: true, quantity: 20, maximum: 99 },
    { id: 4, name: "Small Shield", category: "shield", slot: "left_hand", ammo: false, quantity: 1, maximum: 99 },
  ],
} as InventoryLockerStatusData

describe("party equipment loadout helpers", () => {
  it("filters locker equipment by normalized slot and excludes ammo", () => {
    expect(equipmentForSlot(locker, "weapon").map((item) => item.name)).toEqual(["Dagger"])
    expect(equipmentForSlot(locker, "shield").map((item) => item.name)).toEqual(["Small Shield"])
  })

  it("serializes changed slots from mixed sources into catalog wire values", () => {
    const draft = newLoadoutDraft()
    draft.weapon = { action: "inventory", equipmentID: "16" }
    draft.helm = { action: "manufacture", equipmentID: "0" }
    draft.shield = { action: "unequip", equipmentID: "" }

    expect(stagedLoadoutSlotCount(draft)).toBe(3)
    expect(serializeLoadoutDraft(command, draft, "0.QXZhdGFy", "locker-1")).toEqual({
      character: "0.QXZhdGFy",
      expectedSnapshot: "locker-1",
      helm: "manufacture:0",
      weapon: "inventory:16",
      shield: "unequip",
    })
  })
})
