<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

const props = defineProps<{
  title: string
  subtitle: string
  data?: Record<string, unknown>[]
  stickyHeader?: boolean
}>()

const searches = useInstrumentDraft<Record<string, string>>("codex.filters", {})
const search = computed({ get: () => searches.value[props.title] ?? "", set: (value) => { searches.value[props.title] = value } })
const entries = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return props.data ?? []
  return (props.data ?? []).filter((entry) => JSON.stringify(entry).toLocaleLowerCase().includes(query))
})
const columns = computed(() => Object.keys(entries.value[0] ?? props.data?.[0] ?? {}))

function format(value: unknown): string {
  if (Array.isArray(value)) return value.length ? value.join(", ") : "-"
  if (typeof value === "boolean") return value ? "YES" : "-"
  return String(value ?? "-")
}

function label(column: string): string {
  return column.replace(/([A-Z])/g, " $1").trim().toLocaleUpperCase()
}
</script>

<template>
  <section class="combat-catalog" :aria-label="title">
    <header>
      <div><span>IMMUTABLE DEVELOPER REFERENCE</span><h3>{{ title }}</h3><small>{{ subtitle }}</small></div>
      <label>FILTER <input v-model="search" type="search" :aria-label="`Filter ${title}`" /></label>
    </header>
    <p v-if="!data" class="empty-state">LOADING CODEX REFERENCE</p>
    <div v-else class="table-scroll" :class="stickyHeader ? ['table-scroll--sticky', 'table-scroll--flow'] : []" :tabindex="stickyHeader ? 0 : undefined" role="region" :aria-label="stickyHeader ? `${title} table` : undefined"><table>
      <thead><tr><th v-for="column in columns" :key="column" scope="col">{{ label(column) }}</th></tr></thead>
      <tbody><tr v-for="(entry, index) in entries" :key="String(entry.id ?? entry.offset ?? index)"><td v-for="column in columns" :key="column">{{ format(entry[column]) }}</td></tr></tbody>
    </table></div>
    <p v-if="data && !entries.length" class="empty-state">NO MATCHING REFERENCE ENTRIES</p>
  </section>
</template>
