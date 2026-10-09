<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import type {
  ConversationCatalogData,
  ConversationDetailData,
} from "./approved_tool_views"

const props = defineProps<{
  catalog?: ConversationCatalogData
  detail?: ConversationDetailData
  source: string
  selectedIndex: number
  catalogPending: boolean
  detailPending: boolean
}>()

const emit = defineEmits<{
  source: [source: string]
  inspect: [index: number]
}>()

const filter = useInstrumentDraft("conversations.filter", "")
const filteredScripts = computed(() => {
  const query = filter.value.trim().toLocaleLowerCase()
  if (!query) return props.catalog?.scripts ?? []
  return (props.catalog?.scripts ?? []).filter((script) => `${script.dialog} ${script.semanticId} ${script.name} ${script.description}`.toLocaleLowerCase().includes(query))
})

</script>

<template>
  <section class="approved-view conversation-explorer" aria-label="Conversation script explorer">
    <header class="approved-view-header">
      <div><span>SEMANTIC RUNTIME DOCUMENT READER</span><h3>Conversation Script Explorer</h3></div>
      <small>{{ catalogPending || detailPending ? 'READING...' : `${catalog?.scripts.length ?? 0} SCRIPTS` }}</small>
    </header>

    <div class="conversation-layout">
      <nav class="approved-panel conversation-sources" aria-label="Conversation sources">
        <header><strong>Sources</strong><small>TLK</small></header>
        <button
          v-for="item in catalog?.sources ?? []"
          :key="item.id"
          type="button"
          :class="{ selected: item.id === source }"
          :aria-pressed="item.id === source"
          :disabled="catalogPending"
          @click="emit('source', item.id)"
        ><strong>{{ item.id }}</strong><span>{{ item.scriptCount }}</span></button>
      </nav>

      <section class="approved-panel conversation-index">
        <header><strong>Dialog index</strong><small>{{ filteredScripts.length }} SHOWN</small></header>
        <label class="document-filter" for="conversation-filter">Filter dialogs<input id="conversation-filter" v-model="filter" type="search" placeholder="DIALOG, ID, NAME, DESCRIPTION" /></label>
        <div class="conversation-script-list">
          <button
            v-for="script in filteredScripts"
            :key="script.index"
            type="button"
            :class="{ selected: script.index === selectedIndex }"
            :aria-pressed="script.index === selectedIndex"
            :disabled="detailPending"
            @click="emit('inspect', script.index)"
          >
            <span>#{{ script.dialog }}</span><strong>{{ script.name || 'UNNAMED SCRIPT' }}</strong>
            <small>{{ script.description || 'No detached description' }}</small>
            <i>{{ script.topicCount }} topics · {{ script.questionCount }} questions<span v-if="script.patched"> · PATCHED</span></i>
          </button>
          <p v-if="filteredScripts.length === 0" class="empty-state">NO MATCHING SCRIPTS</p>
        </div>
      </section>

      <article class="approved-panel conversation-document">
        <header><strong>{{ detail ? `${detail.sourceId} / DIALOG ${detail.dialog}` : 'Script document' }}</strong><small>{{ detailPending ? 'LOADING' : 'READ ONLY' }}</small></header>
        <template v-if="detail">
            <section class="semantic-document">
            <div class="semantic-document-meta"><strong>{{ detail.semanticId }}</strong><span>{{ detail.schema }}</span><b :class="{ patched: detail.patched }">{{ detail.patched ? 'PROJECT PATCH APPLIED' : 'UNPATCHED IMPORT' }}</b></div>
            <pre>{{ detail.runtimeYaml }}</pre>
          </section>
        </template>
        <p v-else class="empty-state">{{ detailPending ? 'LOADING SCRIPT DOCUMENT' : 'SELECT A SCRIPT TO INSPECT' }}</p>
      </article>
    </div>

    <p class="read-only-notice"><strong>INSPECTION ONLY</strong> This is the patched semantic document executed by Redux. This explorer does not run Talk or submit player answers.</p>
  </section>
</template>
