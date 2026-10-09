import type { InventoryEquipmentData, InventoryLockerStatusData } from "./approved_tool_views"
import type { CommandDefinition } from "./protocol"

export const partyEquipmentLoadoutCommandID = "party.equipment.loadout.update"

export const loadoutSlots = [
  { id: "helm", label: "Helm" },
  { id: "armour", label: "Armour" },
  { id: "weapon", label: "Weapon" },
  { id: "shield", label: "Shield" },
  { id: "ring", label: "Ring" },
  { id: "amulet", label: "Amulet" },
] as const

export type LoadoutSlotID = typeof loadoutSlots[number]["id"]
export type LoadoutAction = "unchanged" | "inventory" | "manufacture" | "unequip"
export interface LoadoutSlotDraft { action: LoadoutAction; equipmentID: string }
export type LoadoutDraft = Record<LoadoutSlotID, LoadoutSlotDraft>

export function newLoadoutDraft(): LoadoutDraft {
  return Object.fromEntries(loadoutSlots.map((slot) => [slot.id, { action: "unchanged", equipmentID: "" }])) as LoadoutDraft
}

export function equipmentForSlot(data: InventoryLockerStatusData | undefined, slotID: LoadoutSlotID): InventoryEquipmentData[] {
  return (data?.equipment ?? []).filter((equipment) => !equipment.ammo && normalizedSlot(equipment.slot) === slotID)
}

export function serializeLoadoutDraft(
  command: CommandDefinition,
  draft: LoadoutDraft,
  character: string,
  expectedSnapshot: string,
): Record<string, string> {
  const result: Record<string, string> = { character, expectedSnapshot }
  for (const slot of loadoutSlots) {
    const staged = draft[slot.id]
    if (staged.action === "unchanged") continue
    const value = staged.action === "unequip" ? "unequip" : `${staged.action}:${staged.equipmentID}`
    const parameter = command.parameters?.find((candidate) => candidate.id === slot.id)
    if (parameter?.choices?.some((choice) => choice.value === value)) result[slot.id] = value
  }
  return result
}

export function stagedLoadoutSlotCount(draft: LoadoutDraft): number {
  return loadoutSlots.filter((slot) => draft[slot.id].action !== "unchanged").length
}

function normalizedSlot(value: string): LoadoutSlotID | "" {
  const normalized = value.toLocaleLowerCase().replaceAll(/[^a-z]/g, "")
  if (normalized === "armor" || normalized === "armour") return "armour"
  if (normalized === "head") return "helm"
  if (normalized === "righthand") return "weapon"
  if (normalized === "lefthand") return "shield"
  return loadoutSlots.find((slot) => slot.id === normalized)?.id ?? ""
}
