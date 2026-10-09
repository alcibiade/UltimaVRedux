<script setup lang="ts">
import { computed } from "vue"

import type { NavigationSurface } from "./command_navigation"
import { scenarioBaselineLabel, type ScenarioRunStatus, type ScenarioRunStatusResult } from "./scenarios"

const props = defineProps<{
  recents: NavigationSurface[]
  runStatus?: ScenarioRunStatusResult
  connected: boolean
}>()

const emit = defineEmits<{
  openPalette: []
  openSurface: [surface: NavigationSurface]
  openScenarios: [task: "library" | "latest-run", runId?: string]
  reviewAgain: [run: ScenarioRunStatus]
}>()

const recentRuns = computed(() => {
  const runs = props.runStatus?.runs?.length ? props.runStatus.runs : props.runStatus?.run ? [props.runStatus.run] : []
  return runs.slice(0, 3)
})

function runTimestamp(run: ScenarioRunStatus): string {
  return run.finishedAt ?? run.startedAt
}

function formatTimestamp(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? "UNKNOWN TIME" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date)
}
</script>

<template>
  <section class="workbench-home" aria-labelledby="workbench-home-title" data-home-workspace>
    <header class="home-intro">
      <div><span>BRITANNIAN SWITCHBOARD</span><h2 id="workbench-home-title">Workbench Home</h2><p>Choose a recent surface or search the control plane. Navigation never runs a mutation.</p></div>
      <button type="button" data-home-go-to @click="emit('openPalette')">GO TO A SURFACE… <kbd>CTRL/⌘K</kbd></button>
    </header>

    <div class="home-grid">
      <section class="home-panel home-recents" aria-labelledby="home-recents-title">
        <header><h3 id="home-recents-title">Recent Surfaces</h3><small>{{ recents.length }} / 8</small></header>
        <div v-if="recents.length" class="home-recent-list">
          <button v-for="surface in recents" :key="surface.id" type="button" :data-home-recent="surface.id" @click="emit('openSurface', surface)">
            <span>{{ surface.workspaceLabel }}</span><strong>{{ surface.label }}</strong><small>{{ surface.description }}</small><i aria-hidden="true">→</i>
          </button>
        </div>
        <div v-else class="home-empty"><strong>NO RECENT SURFACES</strong><p>Use Go To or choose a workspace above to begin.</p></div>
      </section>

      <div class="home-side-column">
        <section class="home-panel home-latest-run" aria-labelledby="home-run-title">
          <header><h3 id="home-run-title">Recent Scenario Runs</h3><small>{{ runStatus ? `${connected ? 'DURABLE' : 'LAST RECEIVED'} · ${recentRuns.length} / ${runStatus.capacity || recentRuns.length}` : connected ? 'LOADING' : 'LAST RECEIVED' }}</small></header>
          <div v-if="recentRuns.length" class="home-run-list">
             <div v-for="run in recentRuns" :key="run.runId" class="home-run-item">
               <button type="button" :data-home-run="run.runId" :data-run-state="run.state" @click="emit('openScenarios', 'latest-run', run.runId)">
                 <span :class="`state-${run.state}`">{{ run.state.toUpperCase() }}</span>
                 <strong :title="run.reference">{{ run.reference }}</strong>
                 <small>{{ run.nextStep }} / {{ run.totalSteps }} · {{ scenarioBaselineLabel(run.baseline) }}</small>
                 <time :datetime="runTimestamp(run)">{{ formatTimestamp(runTimestamp(run)) }}</time>
                 <i aria-hidden="true">→</i>
               </button>
               <button type="button" class="home-run-play" :data-home-play="run.reference" :aria-label="`Review Again ${run.reference}`" :disabled="!run.launch" :title="run.launch ? 'Review the actual recorded launch values; confirmation is always required.' : 'Historical launch values are unknown. Review Again is unsupported for this old record.'" @click="emit('reviewAgain', run)">REVIEW AGAIN<span v-if="!run.launch"> / UNKNOWN LAUNCH</span></button>
             </div>
            <button type="button" class="home-run-history-link" data-home-open-latest-run @click="emit('openScenarios', 'latest-run')">OPEN RUN HISTORY →</button>
          </div>
          <div v-else-if="runStatus" class="home-empty"><strong>NO DURABLE RUNS YET</strong><p>Completed scenario runs will be retained here on disk.</p><button type="button" data-home-open-scenarios @click="emit('openScenarios', 'library')">OPEN SCENARIOS →</button></div>
          <div v-else class="home-empty"><strong>{{ connected ? 'LOADING RUN HISTORY' : 'RUN HISTORY UNAVAILABLE' }}</strong><p>Durable runs are loaded from the game's private Workbench storage.</p></div>
          <p v-if="runStatus?.historyError" class="home-history-error">HISTORY ERROR · {{ runStatus.historyError }}</p>
        </section>

        <section class="home-panel home-getting-started" aria-labelledby="home-start-title">
          <header><h3 id="home-start-title">Getting Started</h3></header>
          <p><kbd>CTRL/⌘K</kbd> searches every Workbench surface.</p>
          <p>Opening a surface may load read-only telemetry. Mutations still require their normal controls and review.</p>
        </section>
      </div>
    </div>
  </section>
</template>
