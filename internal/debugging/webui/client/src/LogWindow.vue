<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue"

interface LogEntry { sequence: number; timestamp: string; level: string; category: string; message: string }
interface LogBatch { entries: LogEntry[]; nextCursor: number; dropped: number; hasMore: boolean }

const token = document.querySelector<HTMLMetaElement>('meta[name="u5r-workbench-token"]')?.content ?? ""
const entries = ref<LogEntry[]>([])
const cursor = ref(0)
const dropped = ref(0)
const trimmed = ref(0)
const paused = ref(false)
const following = ref(true)
const error = ref("")
const level = ref("ALL")
const category = ref("ALL")
const search = ref("")
const logStream = ref<HTMLElement>()
let timer = 0
let controller: AbortController | undefined
const historyLimit = 2000
const categories = computed(() => [...new Set(entries.value.map((entry) => entry.category))].sort())
const visibleEntries = computed(() => entries.value.filter((entry) => (level.value === "ALL" || entry.level === level.value)
  && (category.value === "ALL" || entry.category === category.value)
  && (!search.value || `${entry.category} ${entry.message}`.toLocaleLowerCase().includes(search.value.toLocaleLowerCase()))))

async function poll(): Promise<void> {
  window.clearTimeout(timer)
  if (paused.value) return schedule()
  controller = new AbortController()
  try {
    const response = await fetch(`/api/logs?after=${cursor.value}&limit=256`, {
      cache: "no-store",
      headers: { "X-U5R-Log-Token": token },
      signal: controller.signal,
    })
    if (!response.ok) throw new Error(`log request failed: ${response.status}`)
    const batch = await response.json() as LogBatch
    cursor.value = batch.nextCursor
    dropped.value += batch.dropped
    if (batch.entries.length) {
      entries.value.push(...batch.entries)
      if (entries.value.length > historyLimit) {
        const remove = entries.value.length - historyLimit
        entries.value.splice(0, remove)
        trimmed.value += remove
      }
      if (following.value) await nextTick(() => logStream.value?.scrollTo({ top: logStream.value.scrollHeight }))
    }
    error.value = ""
    if (batch.hasMore) return void poll()
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === "AbortError")) error.value = reason instanceof Error ? reason.message : String(reason)
  }
  schedule()
}

function schedule(): void {
  window.clearTimeout(timer)
  timer = window.setTimeout(() => void poll(), 500)
}

function clearLocal(): void {
  entries.value = []
  trimmed.value = 0
}

onMounted(() => void poll())
onBeforeUnmount(() => { window.clearTimeout(timer); controller?.abort() })
</script>

<template>
  <main class="log-window">
    <header class="log-window-header"><div><p class="eyebrow">ULTIMA V REDUX :: READ-ONLY DIAGNOSTICS</p><h1>Live Log Stream</h1></div><div class="log-window-actions"><button type="button" @click="paused = !paused; if (!paused) void poll()">{{ paused ? "RESUME" : "PAUSE" }}</button><button type="button" @click="following = !following">FOLLOW {{ following ? "ON" : "OFF" }}</button><button type="button" @click="clearLocal">CLEAR VIEW</button></div></header>
    <section class="log-filter-bar"><label>Level<select v-model="level"><option>ALL</option><option v-for="item in ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']" :key="item">{{ item }}</option></select></label><label>Category<select v-model="category"><option>ALL</option><option v-for="item in categories" :key="item">{{ item }}</option></select></label><label>Filter<input v-model="search" type="search" placeholder="MESSAGE OR CATEGORY" /></label><span>{{ visibleEntries.length }} VISIBLE · CURSOR {{ cursor }}</span></section>
    <p v-if="error" class="log-window-error">{{ error }} · RETRYING</p>
    <p v-if="dropped || trimmed" class="log-window-warning">SERVER OVERWRITTEN {{ dropped }} · LOCAL TRIMMED {{ trimmed }}</p>
    <section ref="logStream" class="log-stream" aria-label="Read-only browser logs" @scroll="following = logStream ? logStream.scrollTop + logStream.clientHeight >= logStream.scrollHeight - 8 : following">
      <article v-for="entry in visibleEntries" :key="entry.sequence" :class="`log-entry level-${entry.level.toLocaleLowerCase()}`"><time>{{ new Date(entry.timestamp).toLocaleTimeString([], { hour12: false, fractionalSecondDigits: 3 }) }}</time><strong>{{ entry.level }}</strong><span>{{ entry.category }}</span><p>{{ entry.message }}</p><small>#{{ entry.sequence }}</small></article>
      <p v-if="visibleEntries.length === 0" class="empty-state">NO MATCHING LOG ENTRIES</p>
    </section>
  </main>
</template>
