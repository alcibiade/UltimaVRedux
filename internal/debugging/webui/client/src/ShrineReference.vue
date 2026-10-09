<script setup lang="ts">
import { computed } from "vue"

import type { ShrineStatusData } from "./shrine_views"

const props = defineProps<{
  data?: ShrineStatusData
  selectedIndex: number
}>()

const emit = defineEmits<{
  select: [index: number]
  inspect: []
}>()

const selected = computed(() => props.data?.shrines.find((shrine) => shrine.index === props.selectedIndex))
</script>

<template>
  <section class="shrine-reference-view" aria-label="Shrine reference">
    <nav aria-label="Virtue shrines">
      <header><strong>SHRINES</strong><small>{{ data?.shrines.length ?? 0 }} VIRTUES</small></header>
      <button
        v-for="shrine in data?.shrines ?? []"
        :key="shrine.index"
        type="button"
        :data-shrine-index="shrine.index"
        :class="{ selected: shrine.index === selectedIndex }"
        :aria-current="shrine.index === selectedIndex ? 'true' : undefined"
        @click="emit('select', shrine.index)"
      >
        <span>{{ shrine.index + 1 }}</span>
        <strong>{{ shrine.virtueName }}</strong>
        <small>{{ shrine.mantra.toLocaleUpperCase() }}</small>
      </button>
    </nav>

    <article v-if="selected" class="shrine-reference-detail">
      <header>
        <div><span>IMMUTABLE CODEX REFERENCE</span><h3>{{ selected.virtueName }}</h3></div>
        <button type="button" class="cross-workspace-link" @click="emit('inspect')">INSPECT LIVE PROGRESS →</button>
      </header>
      <dl>
        <dt>Virtue key</dt><dd>{{ selected.virtueKey }}</dd>
        <dt>Mantra</dt><dd>{{ selected.mantra.toLocaleUpperCase() }}</dd>
        <dt>Word of power</dt><dd>{{ selected.wordOfPower }}</dd>
      </dl>
      <h4>Canonical Location</h4>
      <dl>
        <dt>World coordinates</dt><dd>{{ selected.worldX }}, {{ selected.worldY }}</dd>
      </dl>
      <h4>Completion Reward</h4>
      <dl>
        <dt>Strength</dt><dd>+{{ selected.strBonus }}</dd>
        <dt>Dexterity</dt><dd>+{{ selected.dexBonus }}</dd>
        <dt>Intelligence</dt><dd>+{{ selected.intBonus }}</dd>
        <dt>Karma</dt><dd>+{{ selected.karmaBonus }}</dd>
      </dl>
      <p>Codex values describe original game rules and cannot be edited. Current save progression belongs in Inspect.</p>
    </article>

    <p v-else class="empty-state">WAITING FOR SHRINE REFERENCE</p>
  </section>
</template>
