<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import SpritePreview from "./SpritePreview.vue"

export interface TileAtlasEntry {
  id: number
  name: string
  description: string
  lookDescription: string
  speed: number
  light: number
  animationFrames: number
  animated: boolean
  blocksLight: boolean
  enemy: boolean
  npc: boolean
  building: boolean
  walk: boolean
  horse: boolean
  skiff: boolean
  frigate: boolean
  carpet: boolean
  npcPassable: boolean
}

const props = defineProps<{ data?: TileAtlasEntry[] }>()
const filter = useInstrumentDraft("tiles.filter", "")
const selectedID = useInstrumentDraft<number | undefined>("tiles.selected", undefined)
const entries = computed(() => {
  const query = filter.value.trim().toLocaleLowerCase()
  return (props.data ?? []).filter((entry) => !query || `${entry.id} ${entry.name} ${entry.description} ${entry.lookDescription}`.toLocaleLowerCase().includes(query))
})
const selected = computed(() => props.data?.find((entry) => entry.id === selectedID.value) ?? entries.value[0])
</script>

<template>
  <section class="codex-atlas" aria-label="Tile atlas">
    <header><div><span>SPRITE + RULE REFERENCE</span><h3>Tile Atlas</h3></div><label>FILTER <input v-model="filter" type="search" aria-label="Filter tiles" /></label></header>
    <div v-if="data" class="atlas-layout">
      <div class="table-scroll table-scroll--sticky table-scroll--flow" tabindex="0" role="region" aria-label="Tile Atlas table"><table><thead><tr><th>Sprite</th><th>ID</th><th>Name</th><th>Flags</th></tr></thead><tbody>
        <tr v-for="entry in entries" :key="entry.id" :class="{ selected: selected?.id === entry.id }" @click="selectedID = entry.id"><td><SpritePreview :index="entry.id" :size="24" /></td><th>{{ entry.id }}</th><td>{{ entry.name }}</td><td>{{ entry.animated ? 'ANIM ' : '' }}{{ entry.light ? `LIGHT ${entry.light} ` : '' }}{{ entry.blocksLight ? 'BLOCKS' : '' }}</td></tr>
      </tbody></table></div>
      <aside v-if="selected" class="approved-panel atlas-detail" data-sticky-detail><SpritePreview :index="selected.id" :size="64" /><h4>#{{ selected.id }} {{ selected.name }}</h4><p>{{ selected.description || 'No description.' }}</p><dl><dt>LOOK2.DAT</dt><dd>{{ selected.lookDescription || 'No LOOK2.DAT text.' }}</dd><dt>Speed</dt><dd>{{ selected.speed }}</dd><dt>Animation</dt><dd>{{ selected.animated ? `${selected.animationFrames} frames` : 'Static' }}</dd><dt>Passable</dt><dd>Walk {{ selected.walk ? 'YES' : 'NO' }} · Horse {{ selected.horse ? 'YES' : 'NO' }} · Skiff {{ selected.skiff ? 'YES' : 'NO' }} · Frigate {{ selected.frigate ? 'YES' : 'NO' }} · Carpet {{ selected.carpet ? 'YES' : 'NO' }} · NPC {{ selected.npcPassable ? 'YES' : 'NO' }}</dd></dl></aside>
    </div>
    <p v-else class="empty-state">LOADING TILE ATLAS</p>
  </section>
</template>
