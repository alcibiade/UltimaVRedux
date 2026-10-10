<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft, useInstrumentError } from "./instrument_session"
import { validateDraft } from "./command_forms"

import type { CommandDefinition } from "./protocol"
import type { InventoryLockerStatusData } from "./approved_tool_views"

type LockerTab = "provisions" | "reagents" | "potions" | "scrolls" | "armory"
type TargetKind = "provision" | "reagent" | "equipment"

interface LockerMutation { commandID: string; argumentsByID: Record<string, string> }
interface ReviewedTarget {
  operation: "target"
  targetKind: TargetKind
  key: string
  label: string
  target: string
  expectedQuantity: number
}
interface ReviewedPreset {
  operation: "preset"
  commandID: string
  actionLabel: string
  preset?: string
  presetLabel: string
  expectedSnapshot?: string
  expectedFingerprint?: string
  detail?: string
}
type ReviewedOperation = ReviewedTarget | ReviewedPreset

const props = defineProps<{ data?: InventoryLockerStatusData; updatedAt?: Date; commands: CommandDefinition[]; pending: boolean }>()
const emit = defineEmits<{ mutate: [mutation: LockerMutation] }>()
const tab = useInstrumentDraft<LockerTab>("inventory.tab", "provisions")
const targets = useInstrumentDraft<Record<TargetKind, Record<string, string>>>("inventory.targets", { provision: {}, reagent: {}, equipment: {} })
const additiveAmounts = useInstrumentDraft<Record<string, string>>("inventory.additions", {})
const dirtyTargets = useInstrumentDraft<Record<TargetKind, Set<string>>>("inventory.dirty", { provision: new Set(), reagent: new Set(), equipment: new Set() })
const reviewedOperation = ref<ReviewedOperation>()
const issue = ref("")
const reportError = useInstrumentError()
invalidateInstrumentReview(() => { reviewedOperation.value = undefined })
function fail(message: string): void {
  reviewedOperation.value = undefined
  issue.value = message
  reportError(message)
}
watch(tab, () => { reviewedOperation.value = undefined })
const tabs: { id: LockerTab; label: string }[] = [
  { id: "provisions", label: "Provisions" }, { id: "reagents", label: "Reagents" }, { id: "potions", label: "Potions" },
  { id: "scrolls", label: "Scrolls" }, { id: "armory", label: "Armory" },
]

watch(() => props.data?.snapshotFingerprint, () => {
  if (!props.data) return
  const nextTargets = {
    provision: { ...targets.value.provision },
    reagent: { ...targets.value.reagent },
    equipment: { ...targets.value.equipment },
  }
  const nextDirty = {
    provision: new Set(dirtyTargets.value.provision),
    reagent: new Set(dirtyTargets.value.reagent),
    equipment: new Set(dirtyTargets.value.equipment),
  }
  const reconcile = (kind: TargetKind, key: string, quantity: number) => {
    if (nextDirty[kind].has(key) && nextTargets[kind][key] === String(quantity)) nextDirty[kind].delete(key)
    if (!nextDirty[kind].has(key)) nextTargets[kind][key] = String(quantity)
  }
  for (const provision of props.data.provisions) reconcile("provision", provision.name, provision.quantity)
  for (const reagent of props.data.reagents) reconcile("reagent", String(reagent.id), reagent.quantity)
  for (const equipment of props.data.equipment) reconcile("equipment", String(equipment.id), equipment.quantity)
  targets.value = nextTargets
  dirtyTargets.value = nextDirty
  reviewedOperation.value = undefined
}, { immediate: true })

const currentTabItems = computed(() => !props.data ? [] : tab.value === "reagents" ? props.data.reagents : tab.value === "potions" ? props.data.potions : props.data.scrolls)
const equippedItems = computed(() => (props.data?.equipped ?? []).flatMap((character) => character.slots
  .filter((slot) => slot.equipmentName !== "")
  .map((slot) => ({ ...slot, rosterIndex: character.rosterIndex, characterName: character.name }))))
function presets(commandID: string): { value: string; label: string }[] {
  return props.commands.find((candidate) => candidate.id === commandID)?.parameters?.find((parameter) => parameter.id === "preset")?.choices ?? []
}
function updateTarget(kind: TargetKind, key: string, value: string, quantity: number): void {
  issue.value = ""
  targets.value = { ...targets.value, [kind]: { ...targets.value[kind], [key]: value } }
  const nextDirty = new Set(dirtyTargets.value[kind])
  if (value === String(quantity)) nextDirty.delete(key); else nextDirty.add(key)
  dirtyTargets.value = { ...dirtyTargets.value, [kind]: nextDirty }
  reviewedOperation.value = undefined
}
function stageTarget(kind: TargetKind, key: string, label: string, quantity: number, maximum: number): void {
  const target = targets.value[kind][key] ?? String(quantity)
  const parsed = Number(target)
  if (!target.trim() || !Number.isInteger(parsed) || parsed < 0 || parsed > maximum) {
    fail(`${label}: enter a whole quantity from 0 to ${maximum}. Blank is not zero.`)
    return
  }
  if (parsed === quantity) { fail(`${label}: the quantity already matches.`); return }
  issue.value = ""
  reviewedOperation.value = { operation: "target", targetKind: kind, key, label, target, expectedQuantity: quantity }
}
function stagePreset(commandID: string, presetLabel: string, preset?: string, detail?: string): void {
  if (!props.data) return
  issue.value = ""
  reviewedOperation.value = {
    operation: "preset",
    commandID,
    actionLabel: props.commands.find((candidate) => candidate.id === commandID)?.name ?? "Inventory Preset",
    preset,
    presetLabel,
    expectedSnapshot: props.data.snapshotFingerprint,
    detail,
  }
}
function applyReview(): void {
  const review = reviewedOperation.value
  if (!review) return
  if (review.operation === "target") {
    emit("mutate", {
      commandID: `inventory.${review.targetKind}.set`,
      argumentsByID: { [review.targetKind]: review.key, targetQuantity: review.target, expectedQuantity: String(review.expectedQuantity) },
    })
  } else {
    emit("mutate", {
      commandID: review.commandID,
      argumentsByID: {
		  ...(review.preset ? { preset: review.preset } : {}),
		  ...(review.expectedSnapshot ? { expectedSnapshot: review.expectedSnapshot } : {}),
		  ...(review.expectedFingerprint ? { expectedFingerprint: review.expectedFingerprint } : {}),
	    },
    })
  }
  reviewedOperation.value = undefined
}
function stageUsablePreset(): void {
	if (!props.data) return
  issue.value = ""
	const review = props.data.usablePresetReview
	const categories = [...new Set(review.changes.map((change) => change.category))]
	reviewedOperation.value = {
		operation: "preset",
		commandID: "quest.inventory.preset-usable",
		actionLabel: "Apply Usable Inventory Preset",
		presetLabel: "Usable Inventory",
		expectedFingerprint: review.reviewFingerprint,
		detail: `${review.changes.length} exact changes across ${categories.join(", ") || "no categories"}. Review fingerprint: ${review.reviewFingerprint}`,
	}
}
function addItem(commandID: string, itemKey: "potion" | "scroll", id: number): void {
  const key = `${itemKey}-${id}`
  const amount = additiveAmounts.value[key] ?? "1"
  const command = props.commands.find((candidate) => candidate.id === commandID)
  const errors = command ? Object.values(validateDraft(command, { [itemKey]: String(id), amount })) : ["This operation is unavailable in the catalog."]
  if (!amount.trim() || !Number.isInteger(Number(amount)) || Number(amount) < 1 || errors.length) {
    fail(errors.join(" ") || "Enter a whole amount of at least 1. Blank is not zero.")
    return
  }
  issue.value = ""
  emit("mutate", { commandID, argumentsByID: { [itemKey]: String(id), amount } })
  additiveAmounts.value = { ...additiveAmounts.value, [key]: "1" }
}
</script>

<template>
  <section class="approved-view inventory-locker">
    <header class="approved-view-header"><div><span>AUTHORITATIVE INVENTORY SNAPSHOT</span><h3>Inventory Locker</h3></div><small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small></header>
    <nav class="locker-tabs" aria-label="Inventory locker category"><button v-for="item in tabs" :key="item.id" type="button" :class="{ selected: tab === item.id }" @click="tab = item.id">{{ item.label }}</button></nav>
    <p v-if="issue" class="error-line" role="alert">{{ issue }}</p>
    <p v-if="!data" class="read-only-notice">Loading the complete locker snapshot.</p>
    <template v-else>
      <section v-if="tab === 'provisions'" class="approved-panel locker-panel"><header><strong>Absolute Provision Targets</strong><small>STAGING REQUIRED</small></header><div v-for="item in data.provisions" :key="item.name" class="locker-row"><strong>{{ item.name.replaceAll('_', ' ') }}</strong><small>{{ item.quantity }} / {{ item.maximum }}</small><input :aria-label="`${item.name.replaceAll('_', ' ')} target quantity`" :data-locker-provision="item.name" type="number" min="0" :max="item.maximum" :value="targets.provision[item.name] ?? item.quantity" @input="updateTarget('provision', item.name, ($event.target as HTMLInputElement).value, item.quantity)" /><button type="button" :disabled="pending || !dirtyTargets.provision.has(item.name)" @click="stageTarget('provision', item.name, item.name.replaceAll('_', ' '), item.quantity, item.maximum)">STAGE CHANGE</button></div></section>
      <section v-else-if="tab === 'armory'" class="approved-panel locker-panel"><header><strong>Carried Armory Inventory</strong><small>EXACT QUANTITIES</small></header><p class="read-only-notice">Exact carried quantities are editable. Full clearing destroys both carried items and every roster loadout after review.</p><div v-if="equippedItems.length" class="locker-equipped"><strong>Equipped across roster</strong><span v-for="item in equippedItems" :key="`${item.rosterIndex}-${item.slot}`">{{ item.characterName || `Roster ${item.rosterIndex}` }} / {{ item.slot }}: {{ item.equipmentName }}</span></div><div v-for="item in data.equipment" :key="item.id" class="locker-row armory-row"><strong>{{ item.name }}</strong><span>{{ item.category }} / {{ item.slot }}{{ item.ammo ? " / ammo" : "" }}</span><small>{{ item.quantity }} / {{ item.maximum }}</small><input :aria-label="`${item.name} target quantity`" :data-locker-equipment="item.id" type="number" min="0" :max="item.maximum" :value="targets.equipment[String(item.id)] ?? item.quantity" @input="updateTarget('equipment', String(item.id), ($event.target as HTMLInputElement).value, item.quantity)" /><button type="button" :disabled="pending || !dirtyTargets.equipment.has(String(item.id))" @click="stageTarget('equipment', String(item.id), item.name, item.quantity, item.maximum)">STAGE CHANGE</button></div></section>
      <section v-else class="approved-panel locker-panel"><header><strong>{{ tabs.find((item) => item.id === tab)?.label }}</strong><small>{{ currentTabItems.length }} TYPES</small></header><div v-for="item in currentTabItems" :key="item.id" class="locker-row"><strong>{{ item.name }}</strong><small>{{ item.quantity }} / {{ item.maximum }}</small><template v-if="tab === 'reagents'"><input :aria-label="`${item.name} target quantity`" :data-locker-reagent="item.id" type="number" min="0" :max="item.maximum" :value="targets.reagent[String(item.id)] ?? item.quantity" @input="updateTarget('reagent', String(item.id), ($event.target as HTMLInputElement).value, item.quantity)" /><button type="button" :disabled="pending || !dirtyTargets.reagent.has(String(item.id))" @click="stageTarget('reagent', String(item.id), item.name, item.quantity, item.maximum)">STAGE CHANGE</button></template><template v-else><input :aria-label="`${item.name} ${tab === 'potions' ? 'potion' : 'scroll'} amount to add`" :data-locker-add="`${tab}-${item.id}`" type="number" min="1" :value="additiveAmounts[`${tab === 'potions' ? 'potion' : 'scroll'}-${item.id}`] ?? '1'" @input="additiveAmounts = { ...additiveAmounts, [`${tab === 'potions' ? 'potion' : 'scroll'}-${item.id}`]: ($event.target as HTMLInputElement).value }" /><button type="button" :disabled="pending" @click="addItem(tab === 'potions' ? 'inventory.potion.add' : 'inventory.scroll.add', tab === 'potions' ? 'potion' : 'scroll', item.id)">ADD</button></template></div></section>
      <section class="locker-presets"><template v-for="presetCommandID in tab === 'provisions' ? ['inventory.provision.preset'] : tab === 'reagents' ? ['inventory.reagent.preset'] : tab === 'potions' ? ['inventory.potion.preset'] : tab === 'scrolls' ? ['inventory.scroll.preset'] : ['inventory.equipment.preset', 'inventory.equipment.grant-preset']" :key="presetCommandID"><button v-for="preset in presets(presetCommandID)" :key="preset.value" type="button" :disabled="pending" @click="stagePreset(presetCommandID, preset.label, preset.value)">STAGE: {{ preset.label }}</button></template><button type="button" data-usable-inventory-preset :disabled="pending" @click="stageUsablePreset">STAGE: USABLE INVENTORY</button><button v-if="tab === 'armory'" type="button" :disabled="pending" @click="stagePreset('inventory.equipment.clear', 'Clear All Carried and Equipped Items', undefined, 'This destroys all carried equipment and clears every roster loadout.')">STAGE: CLEAR ALL EQUIPMENT</button></section>
      <section v-if="reviewedOperation" class="locker-review" aria-live="polite"><div><p v-if="reviewedOperation.operation === 'target'">Set {{ reviewedOperation.label }} from {{ reviewedOperation.expectedQuantity }} to {{ reviewedOperation.target }}.</p><p v-else>Apply {{ reviewedOperation.actionLabel }} "{{ reviewedOperation.presetLabel }}" to the current inventory. {{ reviewedOperation.detail }}</p><small>The reviewed snapshot is sent as a stale-state precondition; a fresh inventory update requires restaging.</small></div><button type="button" data-locker-review-apply :disabled="pending" @click="applyReview">{{ reviewedOperation.operation === 'preset' ? `APPLY: ${reviewedOperation.presetLabel}` : 'APPLY STAGED CHANGE' }}</button><button type="button" data-locker-review-cancel :disabled="pending" @click="reviewedOperation = undefined">CANCEL</button></section>
    </template>
  </section>
</template>
