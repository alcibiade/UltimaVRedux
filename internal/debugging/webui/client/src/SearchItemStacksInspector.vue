<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { SearchCellData, SearchStatusData } from "./approved_tool_views"

const props = defineProps<{
  data?: SearchStatusData
  updatedAt?: Date
  addPending: boolean
  resetPending: boolean
  overlayPending: boolean
  searchPending: boolean
}>()

const emit = defineEmits<{
  add: [value: { x: number; y: number; item: string; quantity: number }]
  reset: []
  setOverlay: [enabled: boolean]
  search: [direction: string]
}>()

const selectedDirection = useInstrumentDraft("search.direction", "center")
const itemDraft = useInstrumentDraft("search.item", "gold")
const quantityDraft = useInstrumentDraft("search.quantity", 1)
const resetArmed = ref(false)
invalidateInstrumentReview(() => { resetArmed.value = false })
const selectedCell = computed(() => props.data?.cells.find((cell) => cell.direction.toLocaleLowerCase() === selectedDirection.value)
  ?? props.data?.cells.find((cell) => cell.direction.toLocaleLowerCase() === "center")
  ?? props.data?.cells[0])
const canAdd = computed(() => !!selectedCell.value?.inBounds && itemDraft.value.trim().length > 0
  && Number.isInteger(quantityDraft.value) && quantityDraft.value > 0)
const progressPercent = computed(() => {
  const progress = props.data?.progress
  return progress && progress.totalLocations > 0 ? Math.round(progress.foundLocations / progress.totalLocations * 100) : 0
})
const selectedSearchDirection = computed(() => {
  const direction = selectedCell.value?.direction.toLocaleLowerCase()
  const directions: Record<string, string> = { north: "up", up: "up", south: "down", down: "down", east: "right", right: "right", west: "left", left: "left" }
  return direction ? directions[direction] : undefined
})

watch(() => props.data?.cells, (cells) => {
  if (!cells?.length) return
  if (!cells.some((cell) => cell.direction.toLocaleLowerCase() === selectedDirection.value)) {
    selectedDirection.value = cells.find((cell) => cell.direction.toLocaleLowerCase() === "center")?.direction.toLocaleLowerCase()
      ?? cells[0].direction.toLocaleLowerCase()
  }
}, { immediate: true })

function cellArea(cell: SearchCellData): string {
  const direction = cell.direction.toLocaleLowerCase()
  const areas: Record<string, string> = { center: "center", north: "north", up: "north", south: "south", down: "south", east: "east", right: "east", west: "west", left: "west" }
  return areas[direction] ?? "center"
}

function addItem(): void {
  if (!canAdd.value || !selectedCell.value) return
  emit("add", { x: selectedCell.value.x, y: selectedCell.value.y, item: itemDraft.value.trim(), quantity: quantityDraft.value })
}

function resetSearch(): void {
  emit("reset")
  resetArmed.value = false
}
</script>

<template>
  <section class="approved-view search-inspector" aria-label="Search locations and item stacks inspector">
    <header class="approved-view-header">
      <div><span>LOCAL SPATIAL PROBE</span><h3>Search & Item Stacks</h3></div>
      <small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <div class="approved-split search-layout">
        <section class="approved-panel search-map-pane">
          <header><strong>{{ data.locationName }}</strong><small>{{ data.mapType }} · FLOOR {{ data.floor }}</small></header>
          <div class="search-cross" role="group" aria-label="Choose nearby search cell">
            <button
              v-for="cell in data.cells"
              :key="`${cell.direction}-${cell.x}-${cell.y}`"
              type="button"
              :style="{ gridArea: cellArea(cell) }"
              :class="{ selected: selectedCell === cell, blocked: !cell.inBounds }"
              :aria-pressed="selectedCell === cell"
              @click="selectedDirection = cell.direction.toLocaleLowerCase()"
            >
              <span>{{ cell.direction }}</span><strong>{{ cell.x }},{{ cell.y }}</strong><small>{{ cell.inBounds ? (cell.tileName ?? `TILE ${cell.tileIndex ?? '?'}`) : 'OUT OF BOUNDS' }}</small>
            </button>
          </div>
          <button type="button" :disabled="searchPending || !selectedCell?.inBounds || !selectedSearchDirection" @click="selectedSearchDirection && emit('search', selectedSearchDirection)">{{ searchPending ? 'SEARCHING...' : 'RUN PRODUCTION SEARCH' }}</button>
          <dl class="compact-metrics">
            <div><dt>Found</dt><dd>{{ data.progress.foundLocations }}/{{ data.progress.totalLocations }}</dd></div>
            <div><dt>One-time</dt><dd>{{ data.progress.foundOneTimeLocations }}/{{ data.progress.oneTimeLocations }}</dd></div>
            <div><dt>Coverage</dt><dd>{{ progressPercent }}%</dd></div>
          </dl>
        </section>

        <section v-if="selectedCell" class="approved-panel cell-inspection">
          <header><strong>{{ selectedCell.direction }} CELL</strong><small>{{ selectedCell.x }}, {{ selectedCell.y }}</small></header>
          <p v-if="!selectedCell.inBounds" class="disabled-reason">This coordinate is outside the current map and cannot receive an item.</p>
          <template v-else>
            <h4>Authentic Search Locations</h4>
            <ul class="ledger-list">
              <li v-for="location in selectedCell.authenticLocations" :key="location.index">
                <span :class="location.found ? 'success' : 'summary'">{{ location.found ? 'FOUND' : 'HIDDEN' }}</span>
				<strong>{{ location.itemName || location.category }}</strong>
				<small>0x{{ location.rawObjectType.toString(16).padStart(2, '0') }} / 0x{{ location.rawPayload.toString(16).padStart(2, '0') }} · {{ location.quantity }} · {{ location.repeatable ? 'REPEATABLE' : 'ONE-TIME' }}</small>
				<small>{{ location.discoveryText }}</small>
              </li>
              <li v-if="selectedCell.authenticLocations.length === 0" class="empty-ledger">NO AUTHENTIC SEARCH LOCATION</li>
            </ul>
            <h4>Runtime Item Stack</h4>
            <ul class="ledger-list">
              <li v-for="item in selectedCell.itemStack" :key="`${item.itemType}-${item.itemId}`">
                <span>#{{ item.itemId }}</span><strong>{{ item.itemName }}</strong><small>×{{ item.quantity }} · {{ item.itemType }}</small>
              </li>
              <li v-if="selectedCell.itemStack.length === 0" class="empty-ledger">STACK EMPTY</li>
            </ul>
            <form class="inline-add-form" @submit.prevent="addItem">
              <label>Item<select v-model="itemDraft"><option value="gold">Gold</option><option value="torches">Torches</option><option value="keys">Keys</option><option value="food">Food</option></select></label>
              <label>Qty<input v-model.number="quantityDraft" type="number" min="1" /></label>
              <button type="submit" :disabled="!canAdd || addPending">{{ addPending ? 'ADDING...' : 'ADD TO CELL' }}</button>
            </form>
          </template>
        </section>
      </div>

      <section class="reset-zone" :class="{ armed: resetArmed }">
        <div><strong>{{ resetArmed ? 'RESET PREVIEW' : 'SEARCH PROGRESS RESET' }}</strong><p>{{ resetArmed ? `Will clear ${data.progress.foundOneTimeLocations} authentic one-time found flags.` : 'Clears only authentic one-time search flags. Runtime stacks, daily caches, moonstones, and secret doors are unchanged.' }}</p></div>
        <button v-if="!resetArmed" type="button" :disabled="resetPending" @click="resetArmed = true">PREVIEW RESET</button>
        <template v-else>
          <button type="button" class="danger-button" :disabled="resetPending" @click="resetSearch">{{ resetPending ? 'RESETTING...' : 'CONFIRM RESET' }}</button>
          <button type="button" :disabled="resetPending" @click="resetArmed = false">CANCEL</button>
        </template>
      </section>
      <section class="instrument-setting"><div><strong>GAME-WINDOW SEARCH OVERLAY</strong><p>Marks authentic search locations on the current map.</p></div><button type="button" :disabled="overlayPending" @click="emit('setOverlay', !data.overlayEnabled)">{{ data.overlayEnabled ? 'HIDE OVERLAY' : 'SHOW OVERLAY' }}</button></section>
    </template>
    <p v-else class="empty-state">WAITING FOR SEARCH STATE</p>
  </section>
</template>
