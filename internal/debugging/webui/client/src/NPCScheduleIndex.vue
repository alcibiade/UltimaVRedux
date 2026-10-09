<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import SpritePreview from "./SpritePreview.vue"

export interface NPCScheduleEntry {
  locationID: number
  dialogNumber: number
  spriteIndex: number
  location: string
  type: string
  times: string[]
  ai: string[]
  positions: string[]
  conversationKind: string
  conversationSource: string
  conversationName: string
  conversationDescription: string
  conversationIndex: number
  hasConversation: boolean
}

const props = defineProps<{ data?: NPCScheduleEntry[] }>()
const emit = defineEmits<{ conversation: [value: { source: string; index: number }] }>()
const filter = useInstrumentDraft("npcSchedule.filter", "")
const entries = computed(() => {
  const query = filter.value.trim().toLocaleLowerCase()
  return (props.data ?? []).filter((entry) => !query || `${entry.location} ${entry.dialogNumber} ${entry.conversationName} ${entry.conversationKind}`.toLocaleLowerCase().includes(query))
})
</script>

<template>
  <section class="codex-atlas npc-schedule-index" aria-label="NPC schedule index"><header><div><span>LOADED NPC + SEMANTIC CONVERSATION</span><h3>NPC Schedule Index</h3></div><label>FILTER <input v-model="filter" type="search" aria-label="Filter NPC schedules" /></label></header>
    <p v-if="!data" class="empty-state">LOADING NPC SCHEDULE INDEX</p><div v-else class="table-scroll table-scroll--sticky table-scroll--flow" tabindex="0" role="region" aria-label="NPC Schedule Index table"><table><thead><tr><th>Sprite</th><th>Location</th><th>Dialog</th><th>Conversation</th><th>Schedule</th></tr></thead><tbody>
      <tr v-for="(entry, index) in entries" :key="`${entry.locationID}-${index}`"><td><SpritePreview :index="entry.spriteIndex" :size="24" /></td><td>#{{ entry.locationID }} {{ entry.location }}</td><td>{{ entry.dialogNumber }} · {{ entry.conversationKind }}</td><td><strong>{{ entry.conversationName || '-' }}</strong><small v-if="entry.conversationDescription">{{ entry.conversationDescription }}</small><button v-if="entry.hasConversation" type="button" class="cross-workspace-link" @click="emit('conversation', { source: entry.conversationSource, index: entry.conversationIndex })">VIEW CONVERSATION</button></td><td><span v-for="(_, slot) in entry.times" :key="slot">{{ entry.times[slot] }}: AI {{ entry.ai[slot] }} @ {{ entry.positions[slot] }}</span></td></tr>
    </tbody></table></div>
  </section>
</template>
