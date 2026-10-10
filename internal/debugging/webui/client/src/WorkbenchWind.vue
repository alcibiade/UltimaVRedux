<script setup lang="ts">
import { computed, watch } from "vue"
import { useInstrumentDraft } from "./instrument_session"

const props = defineProps<{
  direction?: string
  updatedAt?: Date
  now: number
  paused: boolean
  pending: boolean
}>()

const emit = defineEmits<{
  apply: [direction: string]
  togglePause: []
}>()

const directions = ["Calm", "North", "East", "South", "West"]
const selectedDirection = useInstrumentDraft("wind.direction", "")
const editing = useInstrumentDraft("wind.editing", false)
const age = computed(() => props.updatedAt ? Math.max(0, Math.floor((props.now - props.updatedAt.getTime()) / 1000)) : undefined)
const changed = computed(() => editing.value && selectedDirection.value !== "" && selectedDirection.value !== props.direction)

watch(() => props.direction, (direction) => {
  if (!editing.value || selectedDirection.value === direction) {
    selectedDirection.value = direction ?? ""
    editing.value = false
  }
}, { immediate: true })

function selectDirection(direction: string): void {
  selectedDirection.value = direction
  editing.value = direction !== props.direction
}
</script>

<template>
  <section class="wind-console" aria-label="Live wind monitor">
    <header>
      <div>
        <span class="live-indicator" :class="{ paused }"><i />{{ paused ? 'PAUSED' : 'LIVE' }}</span>
        <h3>{{ direction ?? 'NO SIGNAL' }}</h3>
        <p>{{ age === undefined ? 'Waiting for first reading' : `Updated ${age}s ago` }}</p>
      </div>
      <button type="button" class="quiet-button" @click="emit('togglePause')">{{ paused ? 'RESUME' : 'PAUSE' }}</button>
    </header>

    <div class="wind-direction-grid" role="group" aria-label="Set wind direction">
      <button
        v-for="candidate in directions"
        :key="candidate"
        type="button"
        :data-wind-direction="candidate.toLocaleLowerCase()"
        :class="{ selected: selectedDirection === candidate, current: direction === candidate }"
        :aria-pressed="selectedDirection === candidate"
        @click="selectDirection(candidate)"
      >
        <span>{{ candidate === 'Calm' ? '·' : candidate.slice(0, 1) }}</span>
        <strong>{{ candidate }}</strong>
        <small v-if="direction === candidate">CURRENT</small>
      </button>
    </div>

    <footer>
      <span>{{ changed ? `${direction} → ${selectedDirection}` : 'Select a different direction to alter live state.' }}</span>
      <button type="button" :disabled="!changed || pending" data-command-id="environment.wind.set" @click="emit('apply', selectedDirection.toLocaleLowerCase())">
        {{ pending ? 'APPLYING...' : 'APPLY WIND' }}
      </button>
    </footer>
  </section>
</template>
