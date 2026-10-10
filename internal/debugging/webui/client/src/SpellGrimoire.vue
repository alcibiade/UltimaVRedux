<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

export interface SpellGrimoireEntry {
  id: number
  circle: number
  name: string
  targeting: string
  context: string
  description: string
  reagents: string[]
  status: string
  originalBehavior: string
  reduxBehavior: string
  originalTargeting: string
  reduxTargeting: string
  actorVisual: string
  screenVisual: string
  statusIndicator: string
  turnOrder: string
  combatEvents: string
  areaEffect: string
  failure: string
  formula: string
  audio: string
  knownIssues: string[]
  opportunities: string[]
  codeReferences: string[]
  originalReferences: string[]
}

const props = defineProps<{ data?: SpellGrimoireEntry[] }>()
const search = useInstrumentDraft("grimoire.search", "")

const entries = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return props.data ?? []
  return (props.data ?? []).filter((spell) =>
    Object.values(spell).flat().join(" ").toLocaleLowerCase().includes(query),
  )
})

const circles = computed(() => Array.from({ length: 8 }, (_, index) => ({
  circle: index + 1,
  spells: entries.value.filter((spell) => spell.circle === index + 1),
})).filter((group) => group.spells.length))
</script>

<template>
  <section class="approved-view spell-grimoire" aria-label="Spell and Reagent Grimoire">
    <header class="approved-view-header">
      <div><span>GAME FACTS + REDUX IMPLEMENTATION AUDIT</span><h3>Spell and Reagent Grimoire</h3><small>{{ entries.length }} / {{ data?.length ?? 0 }} spells across 8 circles</small></div>
      <label>FILTER <input v-model="search" type="search" aria-label="Filter spells by name, reagent, or description" /></label>
    </header>
    <aside class="atlas-provenance"><strong>READING THE GRIMOIRE</strong><p>Every spell compares original behavior with current Redux behavior, including targeting, visuals, audio timing, turn effects, event text, failures, formulas, known issues, opportunities, and Redux code references. SPELL-0002 documents the complete spell catalog; MIX-0001 documents recipes. Original references identify public Game Facts rules with their scoped evidence limits; implementation facts come from the reviewed Redux code. This view does not reflect live inventory.</p></aside>
    <p v-if="!data" class="empty-state">LOADING SPELL GRIMOIRE</p>
    <template v-else>
      <section v-for="group in circles" :key="group.circle" class="spell-circle" :aria-label="`Circle ${group.circle}`">
        <header><strong>CIRCLE {{ group.circle }}</strong><small>{{ group.spells.length }} SPELLS</small></header>
        <div v-for="spell in group.spells" :key="spell.id" class="grimoire-spell-row">
          <div class="grimoire-spell-header">
            <strong>{{ spell.name }}</strong>
            <small>{{ spell.targeting }} · {{ spell.context }}</small>
            <span class="grimoire-status">{{ spell.status }}</span>
          </div>
          <p class="grimoire-spell-description">{{ spell.description }}</p>
          <div class="grimoire-reagents">
            <span class="grimoire-reagents-label">REAGENTS</span>
            <div class="grimoire-reagent-list">
              <span v-for="reagent in spell.reagents" :key="reagent" class="grimoire-reagent">{{ reagent }}</span>
              <span v-if="!spell.reagents.length" class="grimoire-no-reagents">None documented</span>
            </div>
          </div>
          <details class="grimoire-audit">
            <summary>Original vs Redux audit</summary>
            <div class="grimoire-comparison">
              <section><strong>ORIGINAL</strong><p>{{ spell.originalBehavior }}</p><p><b>Targeting:</b> {{ spell.originalTargeting }}</p></section>
              <section><strong>REDUX</strong><p>{{ spell.reduxBehavior }}</p><p><b>Targeting:</b> {{ spell.reduxTargeting }}</p></section>
            </div>
            <dl class="grimoire-facts">
              <div><dt>Actor / target visual</dt><dd>{{ spell.actorVisual }}</dd></div>
              <div><dt>Screen visual</dt><dd>{{ spell.screenVisual }}</dd></div>
              <div><dt>Status indicator</dt><dd>{{ spell.statusIndicator }}</dd></div>
              <div><dt>Turn order</dt><dd>{{ spell.turnOrder }}</dd></div>
              <div><dt>Combat Events</dt><dd>{{ spell.combatEvents }}</dd></div>
              <div><dt>Area clarity</dt><dd>{{ spell.areaEffect }}</dd></div>
              <div><dt>Failure</dt><dd>{{ spell.failure }}</dd></div>
              <div><dt>Formula</dt><dd>{{ spell.formula }}</dd></div>
              <div><dt>Audio timeline</dt><dd>{{ spell.audio }}</dd></div>
            </dl>
            <section class="grimoire-findings">
              <div><strong>KNOWN ISSUES</strong><p v-if="!spell.knownIssues.length">No confirmed spell-specific issue.</p><ul v-else><li v-for="issue in spell.knownIssues" :key="issue">{{ issue }}</li></ul></div>
              <div><strong>OPPORTUNITIES</strong><ul><li v-for="opportunity in spell.opportunities" :key="opportunity">{{ opportunity }}</li></ul></div>
            </section>
            <section class="grimoire-code-map"><strong>CODE MAP</strong><code v-for="reference in spell.codeReferences" :key="reference">{{ reference }}</code></section>
            <section class="grimoire-code-map"><strong>PUBLIC GAME FACTS RULES</strong><code v-for="reference in spell.originalReferences" :key="reference">{{ reference }}</code></section>
          </details>
        </div>
      </section>
      <p v-if="!entries.length" class="empty-state">NO MATCHING SPELLS</p>
    </template>
  </section>
</template>
