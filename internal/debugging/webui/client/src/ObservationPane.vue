<script setup lang="ts">
import type { ObservationResultData } from "./approved_tool_views"

export type ObservationCommandID = "command-test.telescope" | "command-test.spyglass" | "command-test.crystal-ball"

defineProps<{
  data?: ObservationResultData
  commandId?: ObservationCommandID
  updatedAt?: Date
  pending: boolean
}>()

const emit = defineEmits<{ run: [commandID: ObservationCommandID] }>()

const modes: Array<{ id: ObservationCommandID; name: string; requirement: string; outcome: string }> = [
  { id: "command-test.telescope", name: "Telescope", requirement: "Small map", outcome: "Day: sun damage · Night: astronomy" },
  { id: "command-test.spyglass", name: "Spyglass", requirement: "Inventory · above ground", outcome: "Day: no stars · Night: astronomy" },
  { id: "command-test.crystal-ball", name: "Crystal Ball", requirement: "Active party", outcome: "Character selection · intelligence scrying" },
]

function modeName(commandId: ObservationCommandID | undefined): string {
  return modes.find((mode) => mode.id === commandId)?.name ?? "Observation"
}
</script>

<template>
  <section class="approved-view observation-pane" aria-label="Observation command tests">
    <div class="observation-modes">
      <article v-for="mode in modes" :key="mode.id" class="observation-mode">
        <header><strong>{{ mode.name }}</strong><small>{{ mode.requirement }}</small></header>
        <p>{{ mode.outcome }}</p>
        <button
          type="button"
          :data-observation-command="mode.id"
          :aria-label="`Run ${mode.name} observation`"
          :disabled="pending"
          @click="emit('run', mode.id)"
        >{{ pending ? 'OBSERVATION PENDING...' : `RUN ${mode.name}` }}</button>
      </article>
    </div>

    <section v-if="data" class="approved-panel observation-telemetry" aria-live="polite">
      <header><strong>Latest queued telemetry</strong><small>{{ updatedAt ? updatedAt.toLocaleTimeString([], { hour12: false }) : '' }}</small></header>
      <dl>
        <div><dt>Mode</dt><dd>{{ modeName(commandId) }}</dd></div>
        <div><dt>Expected branch</dt><dd>{{ data.expectedBranch }}</dd></div>
        <div><dt>Date / time</dt><dd>{{ data.context.date }} · {{ data.context.time }}</dd></div>
        <div><dt>Location</dt><dd>{{ data.context.locationName }} · {{ data.context.mapType }}</dd></div>
        <div><dt>Floor</dt><dd>{{ data.context.floor }}</dd></div>
      </dl>
      <p>The final interaction continues in the game window.</p>
    </section>
    <p v-else class="empty-state">Choose a mode to queue its production game-window workflow. Nothing runs automatically.</p>
  </section>
</template>
