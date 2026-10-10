<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { MonsterEcosystemData, MonsterEnemyData } from "./approved_tool_views"

const props = defineProps<{
  data?: MonsterEcosystemData
  pendingGeneration: boolean
  pendingClear: boolean
  pendingSpawn: boolean
}>()

const emit = defineEmits<{
  setGeneration: [value: { enabled: boolean; odds: number }]
  clear: [value: { expectedLocationId: number; expectedFloor: number; expectedEnemyCount: number }]
  spawnWhirlpool: []
  spawnPirate: []
}>()

const enabledDraft = useInstrumentDraft("ecosystem.enabled", false)
const oddsDraft = useInstrumentDraft("ecosystem.odds", 1)
const editing = useInstrumentDraft("ecosystem.editing", false)
const clearArmed = ref(false)
invalidateInstrumentReview(() => { clearArmed.value = false })

watch(() => props.data, (data, previous) => {
  if (!data) return

  if (clearArmed.value && previous && (previous.locationId !== data.locationId
    || previous.floor !== data.floor || previous.enemyCount !== data.enemyCount)) clearArmed.value = false
  const draftMatches = enabledDraft.value === data.generationEnabled && oddsDraft.value === data.generationBaseOdds
  if (!editing.value || draftMatches) {
    enabledDraft.value = data.generationEnabled
    oddsDraft.value = data.generationBaseOdds
    editing.value = false
  }
}, { immediate: true })

const generationChanged = computed(() => Boolean(props.data)
  && (enabledDraft.value !== props.data!.generationEnabled || oddsDraft.value !== props.data!.generationBaseOdds))
const oddsValid = computed(() => Number.isInteger(oddsDraft.value) && oddsDraft.value >= 1 && oddsDraft.value <= 1000)
const capacityPercent = computed(() => props.data && props.data.capacity > 0 ? Math.min(100, props.data.currentUnitCount / props.data.capacity * 100) : 0)
const largeMap = computed(() => props.data?.mapType.toLocaleLowerCase().includes("large") || props.data?.mapType.toLocaleLowerCase().includes("britannia"))

function setEnabled(value: boolean): void {
  enabledDraft.value = value
  editing.value = true
}

function setOdds(event: Event): void {
  oddsDraft.value = Number((event.target as HTMLInputElement).value)
  editing.value = true
}

function applyGeneration(): void {
  if (!generationChanged.value || !oddsValid.value) return
  emit("setGeneration", { enabled: enabledDraft.value, odds: oddsDraft.value })
}

function enemyTraits(enemy: MonsterEnemyData): string[] {
  const traits: string[] = []
  if (enemy.activelyAttacks) traits.push("AGGRO")
  if (enemy.canFlyOverWater) traits.push("FLY")
  if (enemy.canPassThroughWalls) traits.push("PHASE")
  if (enemy.doNotMove) traits.push("STATIC")
  if (enemy.hasWhirlpoolMovement) traits.push("WHIRL")
  if (enemy.skipNextTurn) traits.push("SKIP")
  if (enemy.isSandEnemy) traits.push("SAND")
  if (enemy.isWaterEnemy) traits.push("WATER")
  return traits
}

function confirmClear(): void {
  if (!props.data) return
  emit("clear", {
    expectedLocationId: props.data.locationId,
    expectedFloor: props.data.floor,
    expectedEnemyCount: props.data.enemyCount,
  })
  clearArmed.value = false
}
</script>

<template>
  <section class="approved-view monster-console" aria-label="Monster ecosystem console">
    <header class="approved-view-header">
      <div><span>CURRENT-MAP AUTHORITATIVE TELEMETRY</span><h3>Monster Ecosystem</h3></div>
      <small>{{ data ? `${data.locationName} · FLOOR ${data.floor}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <section class="ecosystem-telemetry">
        <div><span>Map</span><strong>{{ data.mapType }}</strong><small>#{{ data.locationId }}</small></div>
        <div><span>Enemies</span><strong>{{ data.enemyCount }}</strong><small>HOSTILE UNITS</small></div>
        <div><span>Friendlies</span><strong>{{ data.friendlyCount }}</strong><small>PRESERVED BY CLEAR</small></div>
        <div><span>Vehicles</span><strong>{{ data.vehicleCount }}</strong><small>PRESERVED BY CLEAR</small></div>
        <div class="capacity-gauge"><span>Map capacity</span><strong>{{ data.currentUnitCount }} / {{ data.capacity }}</strong><meter :min="0" :max="Math.max(1, data.capacity)" :value="data.currentUnitCount">{{ capacityPercent }}%</meter></div>
      </section>

      <div class="approved-split ecosystem-layout">
        <form class="approved-panel generation-console" @submit.prevent="applyGeneration">
          <header><strong>Natural generation</strong><small>{{ data.generationEnabled ? `1 IN ${data.generationBaseOdds}` : 'DISABLED' }}</small></header>
          <fieldset><legend>Explicit state</legend><div class="segment-control"><button type="button" :class="{ selected: enabledDraft }" :aria-pressed="enabledDraft" @click="setEnabled(true)">Enabled</button><button type="button" :class="{ selected: !enabledDraft }" :aria-pressed="!enabledDraft" @click="setEnabled(false)">Disabled</button></div></fieldset>
          <label>Base odds · one in X<input :value="oddsDraft" type="number" min="1" max="1000" :aria-invalid="!oddsValid" @input="setOdds" /><small>The state and odds are sent together as one live-safe update.</small></label>
          <div class="generation-preview"><span>LIVE</span><strong>{{ data.generationEnabled ? 'ENABLED' : 'DISABLED' }} · 1/{{ data.generationBaseOdds }}</strong><b>→</b><span>DRAFT</span><strong>{{ enabledDraft ? 'ENABLED' : 'DISABLED' }} · 1/{{ oddsDraft }}</strong></div>
          <button class="primary-tool-button" type="submit" :disabled="!generationChanged || !oddsValid || pendingGeneration">{{ pendingGeneration ? 'APPLYING...' : 'SET STATE & ODDS' }}</button>
        </form>

        <section class="approved-panel special-spawns">
          <header><strong>Special spawns</strong><small>RANDOM VALID POSITION</small></header>
          <p>Attempts an off-screen deep-water placement. Rejection is reported by the backend and does not mutate the map.</p>
          <div><button type="button" :disabled="pendingSpawn || !largeMap" @click="emit('spawnWhirlpool')">SPAWN WHIRLPOOL</button><button type="button" :disabled="pendingSpawn || !largeMap" @click="emit('spawnPirate')">SPAWN PIRATE</button></div>
          <p v-if="!largeMap" class="disabled-reason">Special spawns require the Britannia large map.</p>
          <p class="deferred-control"><strong>UNAVAILABLE</strong> Arbitrary monster type, coordinate placement, and per-unit mutation are deferred. Use the combat composer for controlled encounters.</p>
        </section>
      </div>

      <section class="approved-panel enemy-roster">
        <header><strong>Enemy roster</strong><small>{{ data.enemies.length }} DETACHED RECORDS</small></header>
        <div class="table-scroll table-scroll--sticky" tabindex="0" role="region" aria-label="Current-map monster roster"><table><thead><tr><th>Token</th><th>Monster</th><th>Position</th><th>Floor</th><th>Visible</th><th>Runtime traits</th></tr></thead><tbody><tr v-for="enemy in data.enemies" :key="enemy.runtimeToken"><td>#{{ enemy.runtimeToken }}</td><td><strong>{{ enemy.name }}</strong><small>TYPE {{ enemy.monsterType }}</small></td><td>{{ enemy.position.x }}, {{ enemy.position.y }}</td><td>{{ enemy.floor }}</td><td :class="enemy.visible ? 'success' : 'summary'">{{ enemy.visible ? 'YES' : 'NO' }}</td><td><span v-for="trait in enemyTraits(enemy)" :key="trait" class="trait-tag">{{ trait }}</span><span v-if="enemyTraits(enemy).length === 0" class="summary">NONE</span></td></tr><tr v-if="data.enemies.length === 0"><td colspan="6" class="empty-state">NO CURRENT-MAP ENEMIES</td></tr></tbody></table></div>
      </section>

      <section class="reset-zone" :class="{ armed: clearArmed }">
        <div><strong>{{ clearArmed ? 'ENEMY-ONLY CLEAR REVIEW' : 'CLEAR CURRENT-MAP ENEMIES' }}</strong><p>{{ clearArmed ? `Will request removal of ${data.enemyCount} hostile units. ${data.friendlyCount} friendlies and ${data.vehicleCount} vehicles are preserved.` : 'Two-stage destructive action. It never clears friendlies or vehicles.' }}</p></div>
        <button v-if="!clearArmed" type="button" :disabled="pendingClear || data.enemyCount === 0" @click="clearArmed = true">REVIEW CLEAR</button>
        <template v-else><button type="button" class="danger-button armed" :disabled="pendingClear" @click="confirmClear">{{ pendingClear ? 'CLEARING...' : 'CONFIRM ENEMY CLEAR' }}</button><button type="button" :disabled="pendingClear" @click="clearArmed = false">CANCEL</button></template>
      </section>
    </template>
    <p v-else class="empty-state">WAITING FOR MONSTER ECOSYSTEM STATUS</p>
  </section>
</template>
