<script setup lang="ts">
import { computed } from "vue"
import { useInstrumentDraft } from "./instrument_session"

export interface SearchLocationAtlasEntry {
  index: number
  locationID: number
  locationName: string
  mapType: string
  floor: number
  x: number
  y: number
  rawObjectType: number
  rawPayload: number
  classification: string
  category: string
  itemID: number
  itemName: string
  pickupQuantity: number
  quantitySemantics: string
  repeatPolicy: string
  requirements: string[]
  suspicious: boolean
  encodingNote: string
}

const props = defineProps<{ data?: SearchLocationAtlasEntry[] }>()
const text = useInstrumentDraft("searchAtlas.text", "")
const location = useInstrumentDraft("searchAtlas.location", "")
const item = useInstrumentDraft("searchAtlas.item", "")
const classification = useInstrumentDraft("searchAtlas.classification", "")
const repeat = useInstrumentDraft("searchAtlas.repeat", "")
const label = (value: string) => value.replaceAll("_", " ")

const optionValues = (field: keyof SearchLocationAtlasEntry) => computed(() =>
  [...new Set((props.data ?? []).map((entry) => String(entry[field])).filter(Boolean))].sort((a, b) => a.localeCompare(b)),
)
const locations = optionValues("locationName")
const items = optionValues("itemName")
const classifications = optionValues("classification")
const repeats = optionValues("repeatPolicy")
const availabilityLabel = (value: string): string => ({
  one_time: "Once",
  conditional: "Returns when conditions are met",
  daily: "Once per day",
  repeatable: "Repeatable",
}[value] ?? label(value))
const requirementLabel = (value: string): string => ({
  no_keys: "Party has no keys",
  no_object_at_location: "No object remains at this location",
  different_day_from_last_skull_keys: "Not already collected today",
  no_glass_sword: "Party has no Glass Sword",
}[value] ?? label(value))
const entries = computed(() => {
  const query = text.value.trim().toLocaleLowerCase()
  return (props.data ?? []).filter((entry) => {
    if (location.value && entry.locationName !== location.value) return false
    if (item.value && entry.itemName !== item.value) return false
    if (classification.value && entry.classification !== classification.value) return false
    if (repeat.value && entry.repeatPolicy !== repeat.value) return false
    const searchableText = [
      entry.index,
      entry.locationName,
      entry.mapType,
      entry.itemName,
      entry.category,
      entry.classification,
      entry.repeatPolicy,
      availabilityLabel(entry.repeatPolicy),
      entry.quantitySemantics,
      ...entry.requirements,
      ...entry.requirements.map(requirementLabel),
    ].join(" ").toLocaleLowerCase()
    return !query || searchableText.includes(query)
  })
})

const hex = (value: number) => `0x${value.toString(16).toUpperCase().padStart(2, "0")}`
</script>

<template>
  <section class="approved-view search-location-atlas" aria-label="Search Location Atlas">
    <header class="approved-view-header">
      <div><span>ORIGINAL SEARCH-LOCATION CATALOG · CMD-0007</span><h3>Search Location Atlas</h3><small>{{ entries.length }} / {{ data?.length ?? 0 }} immutable locations</small></div>
    </header>
    <aside class="atlas-provenance" aria-label="Search atlas provenance">
      <strong>PROVENANCE</strong>
      <p>Coordinates and raw values describe each original search location's object, payload, map, floor, X, and Y fields. Search admission and precedence are documented in CMD-0007. Decoded identities interpret the original object/payload pair; they do not use Redux ItemType labels or live found state.</p>
    </aside>
    <div class="atlas-filters" aria-label="Search location filters">
      <label>TEXT <input v-model="text" type="search" aria-label="Filter search locations by text" /></label>
      <label>LOCATION <select v-model="location" aria-label="Filter search locations by location"><option value="">ALL</option><option v-for="value in locations" :key="value">{{ value }}</option></select></label>
      <label>ITEM <select v-model="item" aria-label="Filter search locations by item"><option value="">ALL</option><option v-for="value in items" :key="value">{{ value }}</option></select></label>
      <label>CLASSIFICATION <select v-model="classification" aria-label="Filter search locations by classification"><option value="">ALL</option><option v-for="value in classifications" :key="value" :value="value">{{ label(value) }}</option></select></label>
      <label>REPEAT <select v-model="repeat" aria-label="Filter search locations by repeat policy"><option value="">ALL</option><option v-for="value in repeats" :key="value" :value="value">{{ availabilityLabel(value) }}</option></select></label>
    </div>
    <div v-if="data" class="table-scroll table-scroll--sticky" tabindex="0" role="region" aria-label="Search Location Atlas table">
      <table><thead><tr><th>Index</th><th>Location</th><th>Position</th><th>Raw encoding</th><th>Decoded pickup</th><th>Quantity</th><th>Availability / conditions</th><th>Encoding note</th></tr></thead>
      <tbody><tr v-for="entry in entries" :key="entry.index" :class="{ suspicious: entry.suspicious }">
        <th scope="row">{{ entry.index }}</th>
        <td><strong>{{ entry.locationName }}</strong><small>ID {{ entry.locationID }} · {{ entry.mapType }}</small></td>
        <td>F{{ entry.floor }} · {{ entry.x }},{{ entry.y }}</td>
        <td><code>{{ hex(entry.rawObjectType) }} / {{ hex(entry.rawPayload) }}</code></td>
        <td><strong>{{ entry.itemName || 'No inventory item' }}</strong><small>{{ label(entry.classification) }} · {{ entry.category }}<template v-if="entry.itemID >= 0"> · ID {{ entry.itemID }}</template></small></td>
        <td>{{ entry.pickupQuantity }}<small>{{ label(entry.quantitySemantics) }}</small></td>
        <td><strong>{{ availabilityLabel(entry.repeatPolicy) }}</strong><small>{{ entry.requirements.length ? entry.requirements.map(requirementLabel).join(', ') : 'No additional condition' }}</small></td>
        <td><span v-if="entry.suspicious" class="encoding-warning">SUSPICIOUS</span>{{ entry.encodingNote || 'Direct documented decoding.' }}</td>
      </tr></tbody></table>
    </div>
    <p v-else class="empty-state">LOADING SEARCH LOCATION ATLAS</p>
    <p v-if="data && !entries.length" class="empty-state">NO MATCHING SEARCH LOCATIONS</p>
  </section>
</template>
