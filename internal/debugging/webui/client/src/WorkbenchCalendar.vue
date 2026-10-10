<script setup lang="ts">
import { computed, watch } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import type { CalendarStatusData } from "./approved_tool_views"
import MoonPhaseGlyph from "./MoonPhaseGlyph.vue"

const props = defineProps<{
  data?: CalendarStatusData
  updatedAt?: Date
  pendingTime: boolean
  pendingDay: boolean
}>()

const emit = defineEmits<{
  setTime: [value: { hour: number; minute: number }]
  setDay: [value: { day: number }]
}>()

const presets = [
  { label: "Morning", hour: 5 },
  { label: "Noon", hour: 12 },
  { label: "Evening", hour: 17 },
  { label: "Dusk", hour: 20 },
  { label: "Midnight", hour: 0 },
]
const hourDraft = useInstrumentDraft("calendar.hour", 0)
const minuteDraft = useInstrumentDraft("calendar.minute", 0)
const dayDraft = useInstrumentDraft("calendar.day", 1)
const timeEditing = useInstrumentDraft("calendar.timeEditing", false)
const dayEditing = useInstrumentDraft("calendar.dayEditing", false)

const timeValid = computed(() => Number.isInteger(hourDraft.value) && hourDraft.value >= 0 && hourDraft.value <= 23
  && Number.isInteger(minuteDraft.value) && minuteDraft.value >= 0 && minuteDraft.value <= 59)
const dayValid = computed(() => Number.isInteger(dayDraft.value) && dayDraft.value >= 1 && dayDraft.value <= 28)
const timeChanged = computed(() => !!props.data && timeValid.value
  && (hourDraft.value !== props.data.hour || minuteDraft.value !== props.data.minute))
const dayChanged = computed(() => !!props.data && dayValid.value && dayDraft.value !== props.data.day)
const daylightLabel = computed(() => {
  const minutes = (props.data?.hour ?? 0) * 60 + (props.data?.minute ?? 0)
  if (minutes < 300 || minutes >= 1200) return "Night"
  if (minutes < 420) return "Dawn"
  if (minutes < 1020) return "Daylight"
  return "Twilight"
})
const moonLabel = computed(() => props.data?.moonPhase || "Unknown phase")

watch(() => props.data, (data) => {
  if (!data) return
  if (!timeEditing.value || (hourDraft.value === data.hour && minuteDraft.value === data.minute)) {
    hourDraft.value = data.hour
    minuteDraft.value = data.minute
    timeEditing.value = false
  }
  if (!dayEditing.value || dayDraft.value === data.day) {
    dayDraft.value = data.day
    dayEditing.value = false
  }
}, { immediate: true })

function editTime(): void {
  timeEditing.value = hourDraft.value !== props.data?.hour || minuteDraft.value !== props.data?.minute
}

function editDay(): void {
  dayEditing.value = dayDraft.value !== props.data?.day
}

function applyPreset(hour: number): void {
  hourDraft.value = hour
  minuteDraft.value = 0
  timeEditing.value = hour !== props.data?.hour || props.data?.minute !== 0
  if (timeEditing.value) emit("setTime", { hour, minute: 0 })
}
</script>

<template>
  <section class="approved-view calendar-console" aria-label="Calendar and time controls">
    <header class="approved-view-header">
      <div><span>AUTHORITATIVE CALENDAR</span><h3>{{ data?.date ?? 'NO DATE SIGNAL' }}</h3></div>
      <small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <div class="calendar-readout">
        <div><span>TIME</span><strong>{{ data.time }}</strong><small>TURN {{ data.turn }}</small></div>
        <dl>
          <div><dt>Light</dt><dd>{{ daylightLabel }}</dd></div>
          <div><dt>Moon</dt><dd class="moon-phase-value"><MoonPhaseGlyph :phase-index="data.moonPhaseIndex" :label="moonLabel" :size="20" />{{ moonLabel }}</dd></div>
          <div><dt>Cycle</dt><dd>Y{{ data.year }} · M{{ data.month }} · D{{ data.day }}</dd></div>
        </dl>
      </div>

      <section class="approved-panel">
        <header><strong>TIME PRESETS</strong><small>MINUTE :00</small></header>
        <div class="preset-strip">
          <button v-for="preset in presets" :key="preset.label" type="button" :disabled="pendingTime" @click="applyPreset(preset.hour)">
            <strong>{{ preset.label }}</strong><span>{{ String(preset.hour).padStart(2, '0') }}:00</span>
          </button>
        </div>
      </section>

      <div class="calendar-split">
        <form class="approved-panel exact-form" @submit.prevent="emit('setTime', { hour: hourDraft, minute: minuteDraft })">
          <header><strong>EXACT TIME</strong><small>24 HOUR</small></header>
          <div class="coordinate-controls">
            <label>Hour<input v-model.number="hourDraft" type="number" min="0" max="23" :aria-invalid="!timeValid" @input="editTime" /></label>
            <span>:</span>
            <label>Minute<input v-model.number="minuteDraft" type="number" min="0" max="59" :aria-invalid="!timeValid" @input="editTime" /></label>
          </div>
          <button type="submit" :disabled="!timeChanged || pendingTime">{{ pendingTime ? 'SETTING...' : 'SET EXACT TIME' }}</button>
        </form>
        <form class="approved-panel exact-form" @submit.prevent="emit('setDay', { day: dayDraft })">
          <header><strong>DAY OF MONTH</strong><small>1-28</small></header>
          <label>Day<input v-model.number="dayDraft" type="number" min="1" max="28" :aria-invalid="!dayValid" @input="editDay" /></label>
          <button type="submit" :disabled="!dayChanged || pendingDay">{{ pendingDay ? 'SETTING...' : 'SET DAY' }}</button>
        </form>
      </div>
    </template>
    <p v-else class="empty-state">WAITING FOR CALENDAR STATE</p>
  </section>
</template>
