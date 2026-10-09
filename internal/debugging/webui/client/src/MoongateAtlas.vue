<script setup lang="ts">
import { computed } from "vue"

import type { MoongateStatusData } from "./approved_tool_views"
import MoonPhaseGlyph from "./MoonPhaseGlyph.vue"

const props = defineProps<{
  data?: MoongateStatusData
  updatedAt?: Date
  overlayPending: boolean
  travelPending: boolean
}>()

const emit = defineEmits<{
  navigate: [value: { x: number; y: number }]
  setOverlay: [enabled: boolean]
  travel: [value: { expectedSourceFingerprint: string; expectedRouteFingerprint: string }]
}>()

const phases = Array.from({ length: 8 }, (_, index) => index)
const phaseLabels = ["New Moon", "Crescent Waxing", "First Quarter", "Gibbous Waxing", "Full Moon", "Gibbous Waning", "Last Quarter", "Crescent Waning"]
const sourceStatus = computed(() => {
  if (!props.data) return "WAITING"
  if (props.data.teleportationActive) return "TRANSIT ACTIVE"
  if (props.data.playerOnActiveGate) return "PLAYER ON ACTIVE GATE"
  if (props.data.visible) return "GATE VISIBLE"
  return "NO VISIBLE GATE"
})
</script>

<template>
  <section class="approved-view moongate-atlas" aria-label="Moongate atlas">
    <header class="approved-view-header">
      <div><span>MOON PHASE ROUTING REFERENCE</span><h3>Moongate Atlas</h3></div>
      <small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <ol class="phase-rail" aria-label="Moon phase rail">
        <li v-for="phase in phases" :key="phase" :class="{ active: phase === data.phaseIndex }">
          <MoonPhaseGlyph :phase-index="phase" :label="phaseLabels[phase]" />
          <span>{{ phase }}</span><small>{{ phaseLabels[phase] }}</small>
        </li>
      </ol>

      <div class="approved-split moongate-layout">
        <section class="position-card gate-source">
          <span>ACTIVE SOURCE</span><h4>{{ sourceStatus }}</h4>
          <div class="current-moon-phase"><MoonPhaseGlyph :phase-index="data.phaseIndex" :label="data.phase" :size="32" /><strong>{{ data.phase }}</strong><small>PHASE {{ data.phaseIndex }}</small></div>
          <dl>
            <div><dt>Visible</dt><dd>{{ data.visible ? 'YES' : 'NO' }}</dd></div>
            <div><dt>Player on gate</dt><dd>{{ data.playerOnActiveGate ? 'YES' : 'NO' }}</dd></div>
            <div><dt>Teleportation</dt><dd>{{ data.teleportationActive ? 'ACTIVE' : 'IDLE' }}</dd></div>
          </dl>
          <p v-if="data.unavailableReason" class="disabled-reason">{{ data.unavailableReason }}</p>
        </section>

        <section class="approved-panel destination-card">
          <header><strong>CURRENT ROUTE</strong><small>{{ data.route ? 'RESOLVED' : 'UNAVAILABLE' }}</small></header>
          <template v-if="data.route?.kind === 'moonstone' && data.route.destination">
            <h4>{{ data.route.destination.locationName }}</h4>
            <dl class="compact-metrics vertical">
              <div><dt>Coordinate</dt><dd>{{ data.route.destination.x }}, {{ data.route.destination.y }}</dd></div>
              <div><dt>Map / level</dt><dd>{{ data.route.destination.map }} / {{ data.route.destination.level }}</dd></div>
              <div><dt>Route</dt><dd>MOONSTONE</dd></div>
            </dl>
            <button type="button" class="navigator-link" @click="emit('navigate', { x: data.route.destination.x, y: data.route.destination.y })">OPEN IN NAVIGATOR →</button>
          </template>
          <template v-else-if="data.route?.kind === 'spirituality'">
            <h4>Shrine of {{ data.route.shrineName ?? 'Spirituality' }}</h4>
            <dl class="compact-metrics vertical">
              <div><dt>Route</dt><dd>SHRINE WORKFLOW</dd></div>
              <div><dt>Shrine index</dt><dd>{{ data.route.shrineIndex }}</dd></div>
            </dl>
          </template>
          <p v-else class="empty-state">{{ data.unavailableReason ?? 'No destination resolves for the current phase.' }}</p>
        </section>
      </div>

      <section class="approved-panel moonstone-table">
        <header><strong>MOONSTONE ARRAY</strong><small>{{ data.moonstones.length }} RECORDS</small></header>
        <div class="table-scroll"><table>
          <thead><tr><th scope="col">Index</th><th scope="col">Phase</th><th scope="col">Coordinate</th><th scope="col">Map / level</th><th scope="col">State</th><th scope="col">Found</th></tr></thead>
          <tbody><tr v-for="stone in data.moonstones" :key="stone.index"><th scope="row">{{ stone.index }}</th><td>{{ stone.phase }}</td><td>{{ stone.x }}, {{ stone.y }}</td><td>{{ stone.map }} / {{ stone.level }}</td><td>{{ stone.state }}</td><td :class="stone.found ? 'success' : 'summary'">{{ stone.found ? 'YES' : 'NO' }}</td></tr></tbody>
        </table></div>
      </section>

      <aside class="unsupported-tools" aria-label="Unavailable moongate operations">
        <div><strong>TRAVEL THROUGH GATE</strong><p>Queues the production route only if source readiness and routing still match this review.</p><button type="button" :disabled="travelPending || !data.playerOnActiveGate || data.teleportationActive || !data.route" @click="data.route && emit('travel', { expectedSourceFingerprint: data.sourceFingerprint, expectedRouteFingerprint: data.route.fingerprint })">{{ travelPending ? 'QUEUEING...' : 'TRAVEL' }}</button></div>
        <div><strong>MOONGATE OVERLAY</strong><p>Shows moonstone records in the game window; rendering is visible only on the large map.</p><button type="button" :disabled="overlayPending" @click="emit('setOverlay', !data.overlayEnabled)">{{ data.overlayEnabled ? 'HIDE OVERLAY' : 'SHOW OVERLAY' }}</button></div>
      </aside>
    </template>
    <p v-else class="empty-state">WAITING FOR MOONGATE STATE</p>
  </section>
</template>
