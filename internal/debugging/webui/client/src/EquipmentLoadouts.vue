<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { InventoryLockerStatusData, PartyCharacterInspectData } from "./approved_tool_views"
import type { CommandDraft, OptionSets } from "./command_forms"
import { characterChoices, selectedCharacterMetadata, type PartyCharacterMetadata } from "./party_character_editor"
import {
  equipmentForSlot,
  loadoutSlots,
  newLoadoutDraft,
  serializeLoadoutDraft,
  stagedLoadoutSlotCount,
  type LoadoutAction,
  type LoadoutSlotID,
} from "./party_equipment_loadout"
import type { Choice, CommandDefinition } from "./protocol"

const props = defineProps<{
  command: CommandDefinition
  draft: CommandDraft
  optionSets: OptionSets
  roster?: PartyCharacterInspectData
  locker?: InventoryLockerStatusData
  pending: boolean
  successToken?: string
  unavailableReason?: string
}>()

const emit = defineEmits<{
  updateCharacter: [value: string]
  submit: [argumentsByID: Record<string, string>]
}>()

const characters = computed(() => characterChoices(props.command, props.optionSets))
const selected = computed(() => selectedCharacterMetadata(props.command, props.optionSets, props.draft))
const allTarget = computed(() => props.draft.character === "all")
const activeRoster = computed(() => props.roster?.roster.filter((character) => character.active) ?? [])
const targetCount = computed(() => allTarget.value ? activeRoster.value.length : selected.value ? 1 : 0)
const selectedRosterIndex = computed(() => selected.value ? activeRoster.value[selected.value.partyPosition - 1]?.rosterIndex : undefined)
const selectedEquipped = computed(() => props.locker?.equipped.find((character) => character.rosterIndex === selectedRosterIndex.value))
const activeEquipped = computed(() => activeRoster.value.map((character) => props.locker?.equipped.find((entry) => entry.rosterIndex === character.rosterIndex)).filter((entry) => entry !== undefined))
const targetDrafts = useInstrumentDraft<Record<string, ReturnType<typeof newLoadoutDraft>>>("loadouts.targets", {})
const targetVersions = useInstrumentDraft<Record<string, number>>("loadouts.versions", {})
const submittedDraft = useInstrumentDraft<{ target: string; version: number; previousSuccessToken?: string } | undefined>("loadouts.submitted", undefined)
const loadoutDraft = computed({
  get: () => {
    const target = props.draft.character ?? ""
    if (!targetDrafts.value[target]) targetDrafts.value[target] = newLoadoutDraft()
    return targetDrafts.value[target]
  },
  set: (value) => { targetDrafts.value[props.draft.character ?? ""] = value },
})
const reviewing = ref(false)
const reviewedSnapshot = ref("")
invalidateInstrumentReview(cancelReview)
const stagedCount = computed(() => stagedLoadoutSlotCount(loadoutDraft.value))

watch(() => props.draft.character, (character, previous) => {
  if (previous !== undefined && character !== previous) cancelReview()
})
watch(() => props.successToken, (token) => {
  const submitted = submittedDraft.value
  if (!token || !submitted || token === submitted.previousSuccessToken) return
  if ((targetVersions.value[submitted.target] ?? 0) === submitted.version) {
    targetDrafts.value[submitted.target] = newLoadoutDraft()
    if (props.draft.character === submitted.target) cancelReview()
  }
  submittedDraft.value = undefined
}, { immediate: true })
watch(() => props.locker?.snapshotFingerprint, (fingerprint, previous) => {
  if (reviewing.value && previous && fingerprint !== previous) cancelReview()
})

function metadata(choice: Choice): PartyCharacterMetadata | undefined {
  return choice.metadata as PartyCharacterMetadata | undefined
}

function loadoutEquipment(slotID: LoadoutSlotID) {
  const equipment = equipmentForSlot(props.locker, slotID)
  return loadoutDraft.value[slotID].action === "inventory"
    ? equipment.filter((item) => item.quantity >= inventoryRequired(slotID, item.id))
    : equipment
}

function currentEquipment(slotID: LoadoutSlotID) {
  const label = loadoutSlots.find((slot) => slot.id === slotID)?.label
  return selectedEquipped.value?.slots.find((slot) => slot.slot.toLocaleLowerCase() === label?.toLocaleLowerCase())
}

function currentEquipmentLabel(slotID: LoadoutSlotID): string {
  if (!allTarget.value) return currentEquipment(slotID)?.equipmentName || "None"
  const slots = targetSlots(slotID)
  if (slots.length !== targetCount.value) return "Mixed"
  const firstID = slots[0]?.equipmentId
  return slots.every((slot) => slot?.equipmentId === firstID) ? slots[0]?.equipmentName || "None" : "Mixed"
}

function targetSlots(slotID: LoadoutSlotID) {
  const label = loadoutSlots.find((slot) => slot.id === slotID)?.label.toLocaleLowerCase()
  return activeEquipped.value.map((character) => character?.slots.find((slot) => slot.slot.toLocaleLowerCase() === label))
}

function inventoryRequired(slotID: LoadoutSlotID, equipmentID: number): number {
  if (!allTarget.value) return currentEquipment(slotID)?.equipmentId === equipmentID ? 0 : 1
  return targetSlots(slotID).filter((slot) => slot?.equipmentId !== equipmentID).length
}

function carriedQuantity(equipmentID: number): number {
  return props.locker?.equipment.find((item) => item.id === equipmentID)?.quantity ?? 0
}

function setAction(slotID: LoadoutSlotID, action: LoadoutAction): void {
  const target = props.draft.character ?? ""
  targetVersions.value[target] = (targetVersions.value[target] ?? 0) + 1
  const choices = equipmentForSlot(props.locker, slotID).filter((item) => action !== "inventory" || item.quantity >= inventoryRequired(slotID, item.id))
  const currentID = loadoutDraft.value[slotID].equipmentID
  loadoutDraft.value[slotID] = {
    action,
    equipmentID: action === "inventory" || action === "manufacture"
      ? choices.some((item) => String(item.id) === currentID) ? currentID : String(choices[0]?.id ?? "")
      : "",
  }
  cancelReview()
}

function setEquipment(slotID: LoadoutSlotID, equipmentID: string): void {
  const target = props.draft.character ?? ""
  targetVersions.value[target] = (targetVersions.value[target] ?? 0) + 1
  loadoutDraft.value[slotID] = { ...loadoutDraft.value[slotID], equipmentID }
  cancelReview()
}

function actionLabel(slotID: LoadoutSlotID): string {
  const staged = loadoutDraft.value[slotID]
  if (staged.action === "unequip") return "Unequip"
  const equipment = equipmentForSlot(props.locker, slotID).find((item) => String(item.id) === staged.equipmentID)
  if (!equipment) return "Unchanged"
  if (staged.action === "manufacture") return `Manufacture and equip ${equipment.name}`
  const quantity = inventoryRequired(slotID, equipment.id)
  return `Equip ${equipment.name} from inventory${allTarget.value ? ` (${quantity} required)` : ""}`
}

function review(): void {
  if (!props.locker || stagedCount.value === 0) return
  reviewedSnapshot.value = props.locker.snapshotFingerprint
  reviewing.value = true
}

function cancelReview(): void {
  reviewing.value = false
  reviewedSnapshot.value = ""
}

function submit(): void {
  if (props.pending || !props.draft.character || !reviewedSnapshot.value || stagedCount.value === 0) return
  submittedDraft.value = { target: props.draft.character, version: targetVersions.value[props.draft.character] ?? 0, previousSuccessToken: props.successToken }
  emit("submit", serializeLoadoutDraft(props.command, loadoutDraft.value, props.draft.character, reviewedSnapshot.value))
  cancelReview()
}
</script>

<template>
  <section class="equipment-loadouts" data-equipment-loadouts>
    <fieldset class="party-picker">
      <legend>Loadout target</legend>
      <button v-for="character in characters" :key="character.value" type="button" class="party-member" :class="{ selected: character.value === draft.character, 'party-member--all': character.value === 'all' }" :data-party-character="character.value" :aria-pressed="character.value === draft.character" @click="emit('updateCharacter', character.value)">
        <template v-if="character.value === 'all'"><span class="party-position">ALL</span><span class="party-identity"><strong>All active party members</strong><small>{{ activeRoster.length }} current targets</small></span></template>
        <template v-else><span class="party-position">{{ metadata(character)?.partyPosition }}</span><span class="party-identity"><strong>{{ metadata(character)?.name }}</strong><small>{{ metadata(character)?.class }}</small></span><span class="party-status" :data-status="metadata(character)?.status">{{ metadata(character)?.status }}</span></template>
      </button>
    </fieldset>

    <section v-if="selected || allTarget" class="editor-group loadout-editor">
      <header><h4>Six-slot atomic loadout</h4><small>{{ stagedCount }} staged slots · {{ targetCount }} targets</small></header>
      <p class="control-note">Stage one or more slots, review the inventory/manufacture plan, then submit one atomic mutation across every target.</p>
      <div v-if="locker && roster" class="loadout-slots">
        <div v-for="slot in loadoutSlots" :key="slot.id" class="loadout-slot" :data-loadout-slot="slot.id">
          <header><strong>{{ slot.label }}</strong><span>Current: {{ currentEquipmentLabel(slot.id) }}</span><small v-if="currentEquipment(slot.id)">Carried: {{ carriedQuantity(currentEquipment(slot.id)!.equipmentId) }}</small></header>
          <label :for="`loadout-${slot.id}-action`">Action</label>
          <select :id="`loadout-${slot.id}-action`" :value="loadoutDraft[slot.id].action" @change="setAction(slot.id, ($event.target as HTMLSelectElement).value as LoadoutAction)">
            <option value="unchanged">Unchanged</option>
            <option value="inventory" :disabled="!equipmentForSlot(locker, slot.id).some((item) => item.quantity >= inventoryRequired(slot.id, item.id))">Inventory</option>
            <option value="manufacture" :disabled="equipmentForSlot(locker, slot.id).length === 0">Manufacture</option>
            <option value="unequip">Unequip</option>
          </select>
          <template v-if="loadoutDraft[slot.id].action === 'inventory' || loadoutDraft[slot.id].action === 'manufacture'">
            <label :for="`loadout-${slot.id}-equipment`">Equipment</label>
            <select :id="`loadout-${slot.id}-equipment`" :value="loadoutDraft[slot.id].equipmentID" @change="setEquipment(slot.id, ($event.target as HTMLSelectElement).value)">
              <option v-for="item in loadoutEquipment(slot.id)" :key="item.id" :value="String(item.id)">{{ item.name }} · carried {{ item.quantity }}</option>
            </select>
          </template>
        </div>
      </div>
      <p v-else class="control-note">Loading the reviewed party and inventory locker snapshots...</p>
      <div v-if="reviewing" class="loadout-review" data-loadout-review>
		<header><strong>Review atomic loadout</strong><small>{{ allTarget ? `${targetCount} active party members` : selected?.name }}</small></header>
		<p>Reviewed inventory fingerprint <code>{{ reviewedSnapshot }}</code></p>
        <ul><li v-for="slot in loadoutSlots.filter((item) => loadoutDraft[item.id].action !== 'unchanged')" :key="slot.id"><span>{{ slot.label }}</span><strong>{{ actionLabel(slot.id) }}</strong></li></ul>
        <p v-if="allTarget">For changed slots, inventory consumes one item per member while manufacture creates one item per member.</p>
        <p v-else>Manufactured items are created directly and do not consume carried inventory.</p>
        <div><button type="button" data-loadout-cancel @click="cancelReview">BACK TO EDIT</button><button type="button" data-loadout-submit :disabled="pending" @click="submit">{{ pending ? 'APPLYING...' : 'CONFIRM LOADOUT' }}</button></div>
      </div>
      <button v-else type="button" class="loadout-review-button" data-loadout-review-button :disabled="stagedCount === 0 || pending || !locker || !roster" @click="review">REVIEW STAGED LOADOUT</button>
      <p v-if="unavailableReason" class="disabled-reason" role="status">{{ unavailableReason }}</p>
    </section>
  </section>
</template>
