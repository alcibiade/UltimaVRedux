<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { VehicleStatusData } from "./approved_tool_views"
import type { Choice, CommandDefinition } from "./protocol"

const props = defineProps<{
  data?: VehicleStatusData
  pendingDamage: boolean
  pendingRepair: boolean
	pendingSkiffs: boolean
	commands: CommandDefinition[]
	pendingSpawn: boolean
	pendingForceBoard: boolean
}>()

const emit = defineEmits<{
  damage: [amount: number]
  repair: []
	setSkiffs: [quantity: number]
	spawn: [argumentsByID: Record<string, string>]
	forceBoard: [type: string]
}>()

const damageDraft = useInstrumentDraft("ships.damage", 1)
const editing = useInstrumentDraft("ships.editing", false)
const reviewed = ref(false)
const skiffDraft = useInstrumentDraft("ships.skiffs", 0)
const lastSkiffs = useInstrumentDraft<number | undefined>("ships.lastSkiffs", undefined)
const spawnType = useInstrumentDraft("ships.spawnType", "frigate")
const spawnPlacement = useInstrumentDraft("ships.placement", "avatar")
const spawnDock = useInstrumentDraft("ships.dock", "jhelom")
const forceBoardType = useInstrumentDraft("ships.forceBoard", "horse")
invalidateInstrumentReview(() => { reviewed.value = false })

watch(() => props.data, (data, previous) => {
  if (!data) return
  const maximum = Math.min(98, Math.max(1, data.hullHp - 1))
  if (previous && previous.hullHp !== data.hullHp) {
    reviewed.value = false
  }
  if (!editing.value) {
    damageDraft.value = Math.min(damageDraft.value, maximum)
  }
	if (lastSkiffs.value === undefined || skiffDraft.value === lastSkiffs.value) skiffDraft.value = data.skiffQuantity
  lastSkiffs.value = data.skiffQuantity
}, { immediate: true })

const isFrigate = computed(() => props.data?.vehicleName.toLocaleLowerCase() === "frigate")
const maxDamage = computed(() => Math.min(98, Math.max(0, (props.data?.hullHp ?? 0) - 1)))
const damageValid = computed(() => Number.isInteger(damageDraft.value) && damageDraft.value >= 1 && damageDraft.value <= maxDamage.value)
const hullPercent = computed(() => props.data && props.data.maximumHullHp > 0 ? Math.round(props.data.hullHp / props.data.maximumHullHp * 100) : 0)
const previewHull = computed(() => Math.max(1, (props.data?.hullHp ?? 1) - damageDraft.value))
const skiffValid = computed(() => Number.isInteger(skiffDraft.value) && skiffDraft.value >= 0 && skiffDraft.value <= 12)
const spawnCommand = computed(() => props.commands.find((command) => command.id === "vehicle.spawn"))
const forceBoardCommand = computed(() => props.commands.find((command) => command.id === "vehicle.force-board"))
const spawnTypes = computed(() => parameterChoices(spawnCommand.value, "type"))
const spawnPlacements = computed(() => parameterChoices(spawnCommand.value, "placement"))
const spawnDocks = computed(() => parameterChoices(spawnCommand.value, "dock"))
const forceBoardTypes = computed(() => parameterChoices(forceBoardCommand.value, "type"))

function parameterChoices(command: CommandDefinition | undefined, parameterID: string): Choice[] {
  return command?.parameters?.find((parameter) => parameter.id === parameterID)?.choices ?? []
}

function spawnVehicle(): void {
  const argumentsByID: Record<string, string> = { type: spawnType.value, placement: spawnPlacement.value }
  if (spawnPlacement.value === "dock") argumentsByID.dock = spawnDock.value
  emit("spawn", argumentsByID)
}

function editDamage(event: Event): void {
  const raw = (event.target as HTMLInputElement).value
  damageDraft.value = raw.trim() ? Number(raw) : Number.NaN
  editing.value = true
  reviewed.value = false
}

function reviewOrDamage(): void {
  if (!damageValid.value || !props.data?.rawHullDamageAvailable) return
  if (!reviewed.value) {
    reviewed.value = true
    return
  }
  emit("damage", damageDraft.value)
}
</script>

<template>
  <section class="approved-view vehicle-inspector" aria-label="Ships console">
    <header class="approved-view-header">
		<div><span>FRIGATE OPERATIONS / VEHICLE TELEMETRY</span><h3>Ships</h3></div>
      <small>{{ data ? `${data.position.x}, ${data.position.y} · FLOOR ${data.floor}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <section class="vehicle-hero" :class="`vehicle-${data.vehicleName.toLocaleLowerCase()}`">
        <div><span>ACTIVE VEHICLE · TYPE {{ data.vehicleType }}</span><h4>{{ data.vehicleName }}</h4><p>{{ data.ownedByNpc ? 'NPC-owned vehicle context' : 'Party-owned vehicle context' }}</p></div>
        <dl><div><dt>Direction</dt><dd>{{ data.currentDirection }}</dd></div><div><dt>Previous</dt><dd>{{ data.previousDirection }}</dd></div><div><dt>Sail state</dt><dd>{{ data.sailState }}</dd></div><div><dt>Skiffs</dt><dd>{{ data.skiffQuantity }}</dd></div><div><dt>HMS Cape</dt><dd>{{ data.hmsCapeActive ? 'ACTIVE' : 'INACTIVE' }}</dd></div></dl>
      </section>

      <section v-if="isFrigate" class="approved-panel hull-console">
        <header><strong>Frigate hull</strong><small>{{ hullPercent }}% INTEGRITY</small></header>
        <div class="hull-meter"><div><span>0</span><strong>{{ data.hullHp }} / {{ data.maximumHullHp }} HP</strong><span>{{ data.maximumHullHp }}</span></div><meter min="0" :max="Math.max(1, data.maximumHullHp)" :value="data.hullHp">{{ hullPercent }}%</meter></div>
        <div class="hull-operations">
          <form @submit.prevent="reviewOrDamage">
            <header><strong>Raw non-sinking damage</strong><small>1–{{ maxDamage }}</small></header>
            <label>Damage amount<input :value="damageDraft" type="number" min="1" :max="maxDamage" :aria-invalid="!damageValid" @input="editDamage" /></label>
            <div class="damage-preview" :class="{ reviewed }"><span>BEFORE<strong>{{ data.hullHp }} HP</strong></span><b>− {{ damageDraft }}</b><span>AFTER<strong>{{ damageValid ? previewHull : 'INVALID' }}{{ damageValid ? ' HP' : '' }}</strong></span></div>
            <p>Damage equal to or greater than current hull is rejected. This instrument cannot sink the ship.</p>
            <div class="vehicle-actions"><button v-if="reviewed" type="button" :disabled="pendingDamage" @click="reviewed = false">EDIT</button><button type="submit" :class="{ 'danger-button': reviewed }" :disabled="!damageValid || !data.rawHullDamageAvailable || pendingDamage">{{ pendingDamage ? 'DAMAGING...' : reviewed ? 'APPLY DAMAGE' : 'REVIEW DAMAGE' }}</button></div>
          </form>
          <section>
            <header><strong>Full repair</strong><small>AUTHORITATIVE MAXIMUM</small></header>
            <p>Restore hull directly to {{ data.maximumHullHp }} HP. No resources, time, or presentation are consumed.</p>
            <button class="primary-tool-button" type="button" :disabled="!data.fullHullRepairAvailable || data.hullHp >= data.maximumHullHp || pendingRepair" @click="emit('repair')">{{ pendingRepair ? 'REPAIRING...' : 'REPAIR TO FULL' }}</button>
          </section>
        </div>
      </section>

		<section v-if="isFrigate" class="approved-panel skiff-console">
			<header><strong>Attached skiffs</strong><small>REDUX CAPACITY · 0–12</small></header>
			<div class="skiff-operation">
				<div><span>CURRENT</span><strong>{{ data.skiffQuantity }}</strong></div>
				<label>Exact quantity<input v-model.number="skiffDraft" type="number" min="0" max="12" step="1" :aria-invalid="!skiffValid" /></label>
				<button type="button" :disabled="!skiffValid || skiffDraft === data.skiffQuantity || pendingSkiffs" @click="emit('setSkiffs', skiffDraft)">{{ pendingSkiffs ? 'SETTING...' : 'SET SKIFFS' }}</button>
			</div>
			<p>Redux limits each frigate to 12 attached skiffs. The original materialized byte had no designed gameplay cap.</p>
		</section>

		<section class="approved-panel vehicle-creation-console">
			<header><strong>Spawn vehicle</strong><small>AUTHORITATIVE MAP INSERTION</small></header>
			<form @submit.prevent="spawnVehicle">
				<label>Type<select v-model="spawnType" data-vehicle-spawn-type><option v-for="choice in spawnTypes" :key="choice.value" :value="choice.value">{{ choice.label }}</option></select></label>
				<label>Placement<select v-model="spawnPlacement" data-vehicle-spawn-placement><option v-for="choice in spawnPlacements" :key="choice.value" :value="choice.value">{{ choice.label }}</option></select></label>
				<label v-if="spawnPlacement === 'dock'">Dock<select v-model="spawnDock" data-vehicle-spawn-dock required><option v-for="choice in spawnDocks" :key="choice.value" :value="choice.value">{{ choice.label }}</option></select></label>
				<button type="submit" :disabled="pendingSpawn || !spawnCommand">{{ pendingSpawn ? 'SPAWNING...' : 'SPAWN VEHICLE' }}</button>
			</form>
			<p>Dock placement uses the selected dock's authoritative overworld coordinates. Avatar placement uses the current map and floor.</p>
		</section>

		<section class="approved-panel force-board-console">
			<header><strong>Force board</strong><small>TERRAIN CHECKS BYPASSED</small></header>
			<label>Type<select v-model="forceBoardType" data-vehicle-force-board-type><option v-for="choice in forceBoardTypes" :key="choice.value" :value="choice.value">{{ choice.label }}</option></select></label>
			<p>This debug operation intentionally bypasses terrain restrictions. Core authority still rejects combat, camping, dungeon, stale-controller, occupied-release, and full-map contexts.</p>
			<button class="danger-button" type="button" :disabled="pendingForceBoard || !forceBoardCommand" @click="emit('forceBoard', forceBoardType)">{{ pendingForceBoard ? 'BOARDING...' : 'FORCE BOARD' }}</button>
		</section>

      <section v-if="!isFrigate" class="approved-panel no-hull-context">
        <header><strong>Hull operations</strong><small>NOT APPLICABLE</small></header>
        <p>{{ data.vehicleName === 'none' ? 'The party is not currently boarded.' : `${data.vehicleName.toLocaleUpperCase()} has no frigate hull state.` }} Damage and repair controls remain visible but unavailable for this context.</p>
        <div><button type="button" disabled>REVIEW DAMAGE</button><button type="button" disabled>REPAIR TO FULL</button></div>
      </section>

      <section class="unsupported-tools">
        <div><strong>SAIL TOGGLE</strong><p>Deferred. Sail-state mutation is unavailable in this batch.</p><button type="button" disabled>UNAVAILABLE</button></div>
        <div><strong>DIRECTION OVERRIDE</strong><p>Deferred. Current and previous direction are telemetry only.</p><button type="button" disabled>UNAVAILABLE</button></div>
      </section>
    </template>
    <p v-else class="empty-state">WAITING FOR VEHICLE STATUS</p>
  </section>
</template>
