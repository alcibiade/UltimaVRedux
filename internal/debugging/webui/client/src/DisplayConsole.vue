<script setup lang="ts">
import { watch } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import type { Choice } from "./protocol"
import type { DisplayStatusData } from "./approved_tool_views"

const props = defineProps<{ data?: DisplayStatusData; modes: Choice[]; updatedAt?: Date; pending: boolean }>()
const emit = defineEmits<{
  setMode: [mode: string]
  setResolution: [index: number]
}>()
const mode = useInstrumentDraft("display.mode", "")
const resolutionIndex = useInstrumentDraft("display.resolution", 0)
const modeDirty = useInstrumentDraft("display.modeDirty", false)
const resolutionDirty = useInstrumentDraft("display.resolutionDirty", false)

watch(() => props.data?.requestedMode, (value) => {
	if (value !== undefined && !modeDirty.value) mode.value = value
}, { immediate: true })

watch(() => props.data, (value) => {
	if (!value || resolutionDirty.value) return
	const selected = value.presetResolutions.find((resolution) =>
		resolution.width === value.normalWindowWidth && resolution.height === value.normalWindowHeight)
	if (selected) resolutionIndex.value = selected.index
}, { immediate: true })

function requestMode(): void {
	modeDirty.value = false
	emit("setMode", mode.value)
}

function requestResolution(): void {
	resolutionDirty.value = false
	emit("setResolution", resolutionIndex.value)
}
</script>

<template>
  <section class="approved-view display-console" aria-label="Game window display controls">
    <header class="approved-view-header">
      <div><span>EBITENGINE GAME WINDOW</span><h3>Display</h3></div>
      <small>{{ data?.transitionPending ? "TRANSITION PENDING" : updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
    </header>
    <p v-if="!data" class="empty-state">WAITING FOR DISPLAY STATUS</p>
    <div v-else class="display-control-grid">
      <section class="approved-panel">
        <header><strong>Window mode</strong><small>ACTUAL · {{ data.actualMode.toUpperCase() }}</small></header>
        <label>Requested mode
			<select v-model="mode" data-display-mode @change="modeDirty = true">
            <option v-for="choice in modes" :key="choice.value" :value="choice.value">{{ choice.label }}</option>
          </select>
        </label>
		<button type="button" data-display-apply-mode :disabled="pending || data.transitionPending || mode === data.actualMode" @click="requestMode">REQUEST MODE</button>
      </section>
      <section class="approved-panel">
        <header><strong>Normal window size</strong><small>{{ data.normalWindowWidth }} × {{ data.normalWindowHeight }}</small></header>
        <label>Preset resolution
			<select v-model.number="resolutionIndex" data-display-resolution @change="resolutionDirty = true">
            <option v-for="resolution in data.presetResolutions" :key="resolution.index" :value="resolution.index">{{ resolution.width }} × {{ resolution.height }}</option>
          </select>
        </label>
		<button type="button" data-display-apply-resolution :disabled="pending || data.transitionPending || data.presetResolutions[resolutionIndex]?.width === data.normalWindowWidth && data.presetResolutions[resolutionIndex]?.height === data.normalWindowHeight" @click="requestResolution">REQUEST PRESET</button>
      </section>
      <section class="approved-panel">
        <header><strong>Observed dimensions</strong><small>{{ data.transitionPending ? 'AWAITING OS' : 'STABLE' }}</small></header>
        <p>Window {{ data.actualWindowWidth }} × {{ data.actualWindowHeight }}</p>
        <p>Monitor {{ data.monitorWidth }} × {{ data.monitorHeight }}</p>
        <p>Container {{ data.containerWidth }} × {{ data.containerHeight }}</p>
        <p>Logical game {{ data.logicalWidth }} × {{ data.logicalHeight }}</p>
      </section>
    </div>
    <p v-if="data" class="instrument-note">Runtime-only controls. Player display settings are unchanged and require confirmation in the game Configuration dialog.</p>
  </section>
</template>
