<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

export type ConsumableSupport = "complete" | "partial" | "unavailable"
export interface ConsumableCodexEntry {
  kind: "potion" | "scroll"
  id: number
  name: string
  originalBehavior: string
  reduxBehavior: string
  originalContext: string
  reduxContext: string
  originalTarget: string
  reduxTarget: string
  originalConsumptionTiming: string
  reduxConsumptionTiming: string
  originalDuration: string
  reduxDuration: string
  support: ConsumableSupport
  deviation: string
}

const props = defineProps<{ data?: ConsumableCodexEntry[] }>()
const kind = useInstrumentDraft("consumables.kind", "")
const support = useInstrumentDraft("consumables.support", "")
const entries = computed(() => (props.data ?? []).filter((entry) =>
  (!kind.value || entry.kind === kind.value) && (!support.value || entry.support === support.value),
))
const groups = computed(() => ["potion", "scroll"].map((value) => ({
  kind: value,
  entries: entries.value.filter((entry) => entry.kind === value),
})).filter((group) => group.entries.length))
</script>

<template>
  <section class="approved-view consumables-codex" aria-label="Consumables Codex">
    <header class="approved-view-header"><div><span>ORIGINAL VS CURRENT REDUX</span><h3>Consumables Codex</h3><small>8 potions · 8 scrolls · immutable behavior reference</small></div></header>
    <aside class="atlas-provenance"><strong>READING THE COMPARISON</strong><p>Original behavior and consumption timing are documented by CONSUME-0001 through CONSUME-0004 in the public Potions and Scrolls rules. Redux behavior documents current implementation only; this view does not alter potion or scroll mechanics. Support is always stated in text as well as color.</p></aside>
    <div class="atlas-filters compact" aria-label="Consumable filters">
      <label>KIND <select v-model="kind" aria-label="Filter consumables by kind"><option value="">ALL</option><option value="potion">POTIONS</option><option value="scroll">SCROLLS</option></select></label>
      <label>SUPPORT <select v-model="support" aria-label="Filter consumables by support"><option value="">ALL</option><option value="complete">COMPLETE</option><option value="partial">PARTIAL</option><option value="unavailable">UNAVAILABLE</option></select></label>
    </div>
    <section v-for="group in groups" :key="group.kind" class="consumable-group" :aria-labelledby="`consumable-${group.kind}`">
      <header><h4 :id="`consumable-${group.kind}`">{{ group.kind }}s</h4><span>{{ group.entries.length }} entries</span></header>
      <div class="consumable-card-grid">
        <article v-for="entry in group.entries" :key="`${entry.kind}-${entry.id}`" class="consumable-card">
          <header><div><span>#{{ entry.id }} {{ entry.kind }}</span><h5>{{ entry.name }}</h5></div><strong class="support-badge" :class="`support-${entry.support}`">{{ entry.support }}</strong></header>
          <dl class="behavior-comparison">
            <div><dt>Original intended behavior</dt><dd>{{ entry.originalBehavior }}</dd></div>
            <div><dt>Current Redux behavior</dt><dd>{{ entry.reduxBehavior }}</dd></div>
            <div><dt>Context</dt><dd><b>Original:</b> {{ entry.originalContext }}<br /><b>Redux:</b> {{ entry.reduxContext }}</dd></div>
            <div><dt>Target</dt><dd><b>Original:</b> {{ entry.originalTarget }}<br /><b>Redux:</b> {{ entry.reduxTarget }}</dd></div>
            <div><dt>Consumption</dt><dd><b>Original:</b> {{ entry.originalConsumptionTiming }}<br /><b>Redux:</b> {{ entry.reduxConsumptionTiming }}</dd></div>
            <div v-if="entry.originalDuration || entry.reduxDuration"><dt>Duration</dt><dd><b>Original:</b> {{ entry.originalDuration || 'Immediate' }}<br /><b>Redux:</b> {{ entry.reduxDuration || 'Immediate' }}</dd></div>
          </dl>
          <p class="deviation"><strong>DEVIATION</strong> {{ entry.deviation }}</p>
        </article>
      </div>
    </section>
    <p v-if="data && !entries.length" class="empty-state">NO MATCHING CONSUMABLES</p>
    <p v-if="!data" class="empty-state">LOADING CONSUMABLES CODEX</p>
  </section>
</template>
