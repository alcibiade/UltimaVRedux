<script setup lang="ts">
export interface CombatMapCatalogData {
  britannia: { index: number; name: string }[]
  dungeons: { index: number; name: string; roomMinimum: number; roomMaximum: number }[]
}

defineProps<{
  data?: CombatMapCatalogData
  section: string
  pending: boolean
}>()

const emit = defineEmits<{
  section: [value: string]
}>()

const sections = [
  { value: "all", label: "All Maps" },
  { value: "britannia", label: "Britannia" },
  { value: "dungeons", label: "Dungeons" },
]
</script>

<template>
  <section class="combat-catalog" aria-label="Combat map reference">
    <header>
      <div><span>IMMUTABLE REFERENCE</span><h3>Combat Maps</h3></div>
      <nav aria-label="Combat map family">
        <button
          v-for="choice in sections"
          :key="choice.value"
          type="button"
          :data-map-section="choice.value"
          :class="{ selected: section === choice.value }"
          :aria-pressed="section === choice.value"
          @click="emit('section', choice.value)"
        >{{ choice.label }}</button>
      </nav>
    </header>

    <p v-if="pending && !data" class="empty-state">LOADING COMBAT MAP REFERENCE</p>
    <div v-else-if="data" class="map-catalog">
      <section v-if="data.britannia.length">
        <header><strong>Britannia</strong><small>{{ data.britannia.length }} MAPS</small></header>
        <table>
          <thead><tr><th scope="col">Index</th><th scope="col">Map</th><th scope="col">Type</th></tr></thead>
          <tbody>
            <tr v-for="item in data.britannia" :key="item.index">
              <th scope="row">{{ String(item.index).padStart(2, '0') }}</th><td>{{ item.name }}</td><td>Britannia</td>
            </tr>
          </tbody>
        </table>
      </section>
      <section v-if="data.dungeons.length">
        <header><strong>Dungeons</strong><small>{{ data.dungeons.length }} DUNGEONS</small></header>
        <table>
          <thead><tr><th scope="col">Index</th><th scope="col">Dungeon</th><th scope="col">Rooms</th></tr></thead>
          <tbody>
            <tr v-for="item in data.dungeons" :key="item.index">
              <th scope="row">{{ String(item.index).padStart(2, '0') }}</th><td>{{ item.name }}</td><td>{{ item.roomMinimum }}-{{ item.roomMaximum }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </section>
</template>
