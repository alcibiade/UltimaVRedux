<script setup lang="ts">
import { computed, watch } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import type { VFXStatusData } from "./approved_tool_views"

type PositionMode = "player" | "coordinates"
type VFXTab = "kapow" | "projectile" | "earthquake"

const props = defineProps<{
  status?: VFXStatusData
  pendingKapow: boolean
  pendingProjectile: boolean
  pendingEarthquake: boolean
  pendingLightingOverlay: boolean
}>()

const emit = defineEmits<{
  kapow: [argumentsByID: Record<string, string>]
  projectile: [argumentsByID: Record<string, string>]
  earthquake: [argumentsByID: Record<string, string>]
  setLightingOverlay: [enabled: boolean]
}>()

const projectileTypes = ["arrow", "axe", "rock", "cannonball", "violet", "red", "blue", "green"] as const
const tab = useInstrumentDraft<VFXTab>("vfx.tab", "kapow")
const targetMode = useInstrumentDraft<PositionMode>("vfx.targetMode", "player")
const targetX = useInstrumentDraft("vfx.targetX", 0)
const targetY = useInstrumentDraft("vfx.targetY", 0)
const fromMode = useInstrumentDraft<PositionMode>("vfx.fromMode", "player")
const fromX = useInstrumentDraft("vfx.fromX", 0)
const fromY = useInstrumentDraft("vfx.fromY", 0)
const toX = useInstrumentDraft("vfx.toX", 0)
const toY = useInstrumentDraft("vfx.toY", 0)
const projectileType = useInstrumentDraft<(typeof projectileTypes)[number]>("vfx.projectile", "arrow")
const earthquakeDuration = useInstrumentDraft("vfx.duration", 20)
const earthquakeIntensity = useInstrumentDraft("vfx.intensity", 8)
const editing = useInstrumentDraft("vfx.editing", false)

watch(() => props.status, (status) => {
  if (!status || editing.value) return
  targetX.value = status.player.x
  targetY.value = status.player.y
  fromX.value = status.player.x
  fromY.value = status.player.y
  toX.value = Math.min(status.maximumX, status.player.x + 4)
  toY.value = status.player.y
}, { immediate: true })

const fromPosition = computed(() => fromMode.value === "player" ? props.status?.player : { x: fromX.value, y: fromY.value })
const coordinatesValid = computed(() => {
  if (!props.status) return false
  const bounded = (x: number, y: number) => Number.isInteger(x) && Number.isInteger(y) && x >= 0 && y >= 0 && x <= props.status!.maximumX && y <= props.status!.maximumY
  if (tab.value === "kapow") return targetMode.value === "player" || bounded(targetX.value, targetY.value)
  return (fromMode.value === "player" || bounded(fromX.value, fromY.value)) && bounded(toX.value, toY.value)
})
const activeProjectile = computed(() => Boolean(props.status?.activeProjectile?.active))
const earthquakeValid = computed(() => Number.isInteger(earthquakeDuration.value) && earthquakeDuration.value >= 1 && earthquakeDuration.value <= 100
  && Number.isInteger(earthquakeIntensity.value) && earthquakeIntensity.value >= 1 && earthquakeIntensity.value <= 20)

function alter(action: () => void): void {
  action()
  editing.value = true
}

function numberValue(event: Event): number {
  return Number((event.target as HTMLInputElement).value)
}

function runKapow(): void {
  if (!coordinatesValid.value) return
  const args: Record<string, string> = { targetMode: targetMode.value }
  if (targetMode.value === "coordinates") {
    args.x = String(targetX.value)
    args.y = String(targetY.value)
  }
  emit("kapow", args)
}

function runProjectile(): void {
  if (!coordinatesValid.value || activeProjectile.value) return
  const args: Record<string, string> = {
    fromMode: fromMode.value,
    toX: String(toX.value),
    toY: String(toY.value),
    projectileType: projectileType.value,
  }
  if (fromMode.value === "coordinates") {
    args.fromX = String(fromX.value)
    args.fromY = String(fromY.value)
  }
  emit("projectile", args)
}

function runEarthquake(): void {
  if (!earthquakeValid.value || props.status?.earthquakeActive) return
  emit("earthquake", { durationTenths: String(earthquakeDuration.value), intensityTenths: String(earthquakeIntensity.value) })
}
</script>

<template>
  <section class="approved-view vfx-bench" aria-label="Visual effects test bench">
    <header class="approved-view-header">
      <div><span>PRESENTATION LANE · NO GAMEPLAY DAMAGE</span><h3>VFX Test Bench</h3></div>
      <small>{{ status ? `${status.mapType} · ${status.maximumX + 1}×${status.maximumY + 1}` : 'WAITING' }}</small>
    </header>
    <section v-if="status" class="instrument-setting"><div><strong>LIGHT-DISTANCE OVERLAY</strong><p>Visualizes primary and secondary lighting distance maps in the game window.</p></div><button type="button" :disabled="pendingLightingOverlay" @click="emit('setLightingOverlay', !status.lightingOverlayEnabled)">{{ status.lightingOverlayEnabled ? 'HIDE OVERLAY' : 'SHOW OVERLAY' }}</button></section>

    <div class="vfx-tabs" role="tablist" aria-label="Visual effect type">
      <button id="vfx-tab-kapow" type="button" role="tab" :aria-selected="tab === 'kapow'" :class="{ selected: tab === 'kapow' }" @click="tab = 'kapow'">Kapow</button>
      <button id="vfx-tab-projectile" type="button" role="tab" :aria-selected="tab === 'projectile'" :class="{ selected: tab === 'projectile' }" @click="tab = 'projectile'">Projectile</button>
      <button id="vfx-tab-earthquake" type="button" role="tab" :aria-selected="tab === 'earthquake'" :class="{ selected: tab === 'earthquake' }" @click="tab = 'earthquake'">Earthquake</button>
    </div>

    <div v-if="status" class="approved-split vfx-layout">
      <form v-if="tab === 'kapow'" class="approved-panel vfx-control-panel" role="tabpanel" aria-labelledby="vfx-tab-kapow" @submit.prevent="runKapow">
        <header><strong>Kapow target</strong><small>500 MS FEEDBACK</small></header>
        <fieldset><legend>Target mode</legend><div class="segment-control"><button type="button" :class="{ selected: targetMode === 'player' }" :aria-pressed="targetMode === 'player'" @click="alter(() => targetMode = 'player')">Player</button><button type="button" :class="{ selected: targetMode === 'coordinates' }" :aria-pressed="targetMode === 'coordinates'" @click="alter(() => targetMode = 'coordinates')">Coordinates</button></div></fieldset>
        <div v-if="targetMode === 'coordinates'" class="coordinate-controls"><label>X<input :value="targetX" type="number" min="0" :max="status.maximumX" @input="alter(() => targetX = numberValue($event))" /></label><span>×</span><label>Y<input :value="targetY" type="number" min="0" :max="status.maximumY" @input="alter(() => targetY = numberValue($event))" /></label></div>
        <p>Resolved target: <strong>{{ targetMode === 'player' ? `${status.player.x}, ${status.player.y} (PLAYER)` : `${targetX}, ${targetY}` }}</strong></p>
        <button class="primary-tool-button" type="submit" :disabled="!coordinatesValid || pendingKapow">{{ pendingKapow ? 'QUEUING...' : 'QUEUE KAPOW' }}</button>
      </form>

      <form v-else-if="tab === 'projectile'" class="approved-panel vfx-control-panel" role="tabpanel" aria-labelledby="vfx-tab-projectile" @submit.prevent="runProjectile">
        <header><strong>Projectile vector</strong><small>SERIALIZED</small></header>
        <fieldset><legend>Origin mode</legend><div class="segment-control"><button type="button" :class="{ selected: fromMode === 'player' }" :aria-pressed="fromMode === 'player'" @click="alter(() => fromMode = 'player')">Player</button><button type="button" :class="{ selected: fromMode === 'coordinates' }" :aria-pressed="fromMode === 'coordinates'" @click="alter(() => fromMode = 'coordinates')">Coordinates</button></div></fieldset>
        <div v-if="fromMode === 'coordinates'" class="coordinate-controls"><label>From X<input :value="fromX" type="number" min="0" :max="status.maximumX" @input="alter(() => fromX = numberValue($event))" /></label><span>×</span><label>From Y<input :value="fromY" type="number" min="0" :max="status.maximumY" @input="alter(() => fromY = numberValue($event))" /></label></div>
        <div class="coordinate-controls"><label>To X<input :value="toX" type="number" min="0" :max="status.maximumX" @input="alter(() => toX = numberValue($event))" /></label><span>×</span><label>To Y<input :value="toY" type="number" min="0" :max="status.maximumY" @input="alter(() => toY = numberValue($event))" /></label></div>
        <label>Projectile<select v-model="projectileType" @change="editing = true"><option v-for="type in projectileTypes" :key="type" :value="type">{{ type }}</option></select></label>
        <div class="vector-summary" aria-label="Projectile vector summary"><span>{{ fromPosition?.x }},{{ fromPosition?.y }}</span><b>→</b><span>{{ toX }},{{ toY }}</span><strong>{{ projectileType }}</strong></div>
        <p v-if="activeProjectile" class="disabled-reason">A projectile is active. The backend permits only one serialized projectile at a time.</p>
        <button class="primary-tool-button" type="submit" :disabled="!coordinatesValid || activeProjectile || pendingProjectile">{{ pendingProjectile ? 'QUEUING...' : 'QUEUE PROJECTILE' }}</button>
      </form>

      <form v-else class="approved-panel vfx-control-panel" role="tabpanel" aria-labelledby="vfx-tab-earthquake" @submit.prevent="runEarthquake">
        <header><strong>Earthquake feedback</strong><small>PRESENTATION ONLY</small></header>
        <label>Duration (tenths)<input v-model.number="earthquakeDuration" type="number" min="1" max="100" /></label>
        <label>Intensity (tenths)<input v-model.number="earthquakeIntensity" type="number" min="1" max="20" /></label>
        <p>{{ (earthquakeDuration / 10).toFixed(1) }} seconds · intensity {{ (earthquakeIntensity / 10).toFixed(1) }} · max offset {{ (earthquakeIntensity * 1.2).toFixed(1) }} px</p>
        <p v-if="status.earthquakeActive" class="disabled-reason">An earthquake is already pending or active.</p>
        <button class="primary-tool-button" type="submit" :disabled="!earthquakeValid || status.earthquakeActive || pendingEarthquake">{{ pendingEarthquake ? 'QUEUING...' : 'QUEUE EARTHQUAKE' }}</button>
      </form>

      <aside class="approved-panel vfx-telemetry">
        <header><strong>Live effects</strong><small>{{ status.visualEffects.length }} ACTIVE</small></header>
        <ul class="effect-ledger"><li v-for="(effect, index) in status.visualEffects" :key="index"><strong>{{ effect.type }}</strong><span>{{ effect.worldPosition ? `${effect.worldPosition.x},${effect.worldPosition.y}` : `SCREEN ${effect.screenX},${effect.screenY}` }}</span><small>{{ effect.elapsed.toFixed(0) }} / {{ effect.duration.toFixed(0) }}</small></li><li v-if="status.visualEffects.length === 0">NO ACTIVE VISUAL EFFECTS</li></ul>
        <h4>Projectile telemetry</h4>
        <dl v-if="status.activeProjectile" class="compact-metrics vertical"><div><dt>State</dt><dd :class="status.activeProjectile.active ? 'success' : 'summary'">{{ status.activeProjectile.active ? 'ACTIVE' : 'INACTIVE' }}</dd></div><div><dt>Type</dt><dd>{{ status.activeProjectile.type }} (#{{ status.activeProjectile.typeId }})</dd></div><div><dt>Vector</dt><dd>{{ status.activeProjectile.fromTileX.toFixed(2) }},{{ status.activeProjectile.fromTileY.toFixed(2) }} → {{ status.activeProjectile.toTileX.toFixed(2) }},{{ status.activeProjectile.toTileY.toFixed(2) }}</dd></div><div><dt>Current</dt><dd>{{ status.activeProjectile.currentTileX.toFixed(2) }}, {{ status.activeProjectile.currentTileY.toFixed(2) }}</dd></div></dl>
        <p v-else class="empty-state">NO PROJECTILE TELEMETRY</p>
        <h4>Earthquake</h4><p :class="status.earthquakeActive ? 'success' : 'summary'">{{ status.earthquakeActive ? 'ACTIVE / QUEUED' : 'IDLE' }}</p>
      </aside>
    </div>
    <p v-else class="empty-state">WAITING FOR VFX STATUS</p>
  </section>
</template>
