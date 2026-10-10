<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { CombatEncounterOptionsData } from "./approved_tool_views"

const props = defineProps<{
  options?: CombatEncounterOptionsData
  pending: boolean
}>()

const emit = defineEmits<{
  start: [argumentsByID: Record<string, string>]
}>()

const territory = useInstrumentDraft<"britannia" | "dungeon">("encounter.territory", "britannia")
const britanniaMap = useInstrumentDraft("encounter.map", 0)
const dungeon = useInstrumentDraft("encounter.dungeon", 0)
const dungeonRoom = useInstrumentDraft("encounter.room", 0)
const entryDirection = useInstrumentDraft("encounter.direction", "west")
const monsterType = useInstrumentDraft("encounter.monster", "")
const countMode = useInstrumentDraft<"automatic" | "exact" | "maximum">("encounter.mode", "automatic")
const exactCount = useInstrumentDraft("encounter.count", 1)
const friends = useInstrumentDraft("encounter.friends", true)
const seed = useInstrumentDraft("encounter.seed", "")
const editing = useInstrumentDraft("encounter.editing", false)
const reviewed = ref(false)
invalidateInstrumentReview(() => { reviewed.value = false })

const selectedDungeon = computed(() => props.options?.dungeons.find((item) => item.id === dungeon.value))
const selectedMonster = computed(() => props.options?.monsters.find((item) => String(item.id) === monsterType.value))
const selectedMap = computed(() => props.options?.britanniaMaps.find((item) => item.index === britanniaMap.value))
const validSeed = computed(() => seed.value === "" || (/^\d+$/.test(seed.value) && Number.isSafeInteger(Number(seed.value))))
const valid = computed(() => Boolean(selectedMonster.value)
  && (territory.value === "britannia" ? Boolean(selectedMap.value) : Boolean(selectedDungeon.value?.rooms.includes(dungeonRoom.value)))
  && (countMode.value !== "exact" || (Number.isInteger(exactCount.value) && exactCount.value >= 1 && exactCount.value <= 16))
  && validSeed.value)

watch(() => props.options, (options) => {
  if (!options || editing.value) return
  britanniaMap.value = options.britanniaMaps[0]?.index ?? 0
  dungeon.value = options.dungeons[0]?.id ?? 0
  dungeonRoom.value = options.dungeons[0]?.rooms[0] ?? 0
  entryDirection.value = options.entryDirections[0]?.name ?? "west"
  monsterType.value = String(options.monsters[0]?.id ?? "")
  countMode.value = (options.countModes[0]?.name as typeof countMode.value | undefined) ?? "automatic"
}, { immediate: true })

function alter(action: () => void): void {
  action()
  editing.value = true
  reviewed.value = false
}

function inputString(event: Event): string {
  return (event.target as HTMLInputElement | HTMLSelectElement).value
}

function inputNumber(event: Event): number {
  const raw = inputString(event)
  return raw.trim() ? Number(raw) : Number.NaN
}

function changeDungeon(event: Event): void {
  alter(() => {
    dungeon.value = inputNumber(event)
    dungeonRoom.value = props.options?.dungeons.find((item) => item.id === dungeon.value)?.rooms[0] ?? 0
  })
}

function reviewOrStart(): void {
  if (!valid.value) return
  if (!reviewed.value) {
    reviewed.value = true
    return
  }
  const argumentsByID: Record<string, string> = {
    territory: territory.value,
    entryDirection: entryDirection.value,
    monsterType: monsterType.value,
    countMode: countMode.value,
    friends: String(friends.value),
  }
  if (territory.value === "britannia") argumentsByID.britanniaMap = String(britanniaMap.value)
  else {
    argumentsByID.dungeon = String(dungeon.value)
    argumentsByID.dungeonRoom = String(dungeonRoom.value)
  }
  if (countMode.value === "exact") argumentsByID.exactCount = String(exactCount.value)
  if (seed.value !== "") argumentsByID.seed = seed.value
  emit("start", argumentsByID)
}
</script>

<template>
  <form class="approved-view encounter-composer" aria-label="Combat encounter composer" @submit.prevent="reviewOrStart">
    <header class="approved-view-header">
      <div><span>SYNCHRONIZED COMBAT TRANSITION</span><h3>Encounter Composer</h3></div>
      <small>{{ pending ? 'STARTING...' : reviewed ? 'REVIEW STAGED' : 'DRAFT' }}</small>
    </header>

    <div v-if="options" class="encounter-layout">
      <section class="approved-panel encounter-form">
        <header><strong>Composition</strong><small>ONE MONSTER FAMILY</small></header>
        <fieldset>
          <legend>Territory</legend>
          <div class="segment-control">
            <button v-for="choice in ['britannia', 'dungeon']" :key="choice" type="button" :class="{ selected: territory === choice }" :aria-pressed="territory === choice" @click="alter(() => territory = choice as typeof territory)">{{ choice }}</button>
          </div>
        </fieldset>
        <label v-if="territory === 'britannia'">Combat map
          <select :value="britanniaMap" @change="alter(() => britanniaMap = inputNumber($event))"><option v-for="item in options.britanniaMaps" :key="item.index" :value="item.index">{{ item.index }} · {{ item.name }}</option></select>
        </label>
        <template v-else>
          <label>Dungeon<select :value="dungeon" @change="changeDungeon"><option v-for="item in options.dungeons" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
          <label>Room<select :value="dungeonRoom" @change="alter(() => dungeonRoom = inputNumber($event))"><option v-for="room in selectedDungeon?.rooms ?? []" :key="room" :value="room">ROOM {{ room }}</option></select></label>
        </template>
        <label>Entry direction<select :value="entryDirection" @change="alter(() => entryDirection = inputString($event))"><option v-for="item in options.entryDirections" :key="item.id" :value="item.name">{{ item.name }}</option></select></label>
        <label>Monster<select :value="monsterType" @change="alter(() => monsterType = inputString($event))"><option v-for="item in options.monsters" :key="item.id" :value="item.id">{{ item.name }}</option></select></label>
        <label>Count mode<select :value="countMode" @change="alter(() => countMode = inputString($event) as typeof countMode)"><option v-for="item in options.countModes" :key="item.id" :value="item.name">{{ item.name }}</option></select></label>
        <label v-if="countMode === 'exact'">Exact count<input :value="exactCount" type="number" min="1" max="16" :aria-invalid="exactCount < 1 || exactCount > 16" @input="alter(() => exactCount = inputNumber($event))" /><small>Validated against the selected map's available positions at start.</small></label>
        <label class="check-control"><input :checked="friends" type="checkbox" @change="alter(() => friends = ($event.target as HTMLInputElement).checked)" /> Include authentic friend type</label>
        <label>Deterministic seed <span>OPTIONAL</span><input :value="seed" type="text" inputmode="numeric" pattern="[0-9]*" :aria-invalid="!validSeed" placeholder="CENTRAL RNG SEED" @input="alter(() => seed = inputString($event))" /></label>
      </section>

      <aside class="approved-panel encounter-preview">
        <header><strong>{{ reviewed ? 'Final review' : 'Structured preview' }}</strong><small>{{ reviewed ? 'STAGE 2 / 2' : 'STAGE 1 / 2' }}</small></header>
        <dl>
          <dt>Territory</dt><dd>{{ territory }}</dd>
          <dt>Battlefield</dt><dd>{{ territory === 'britannia' ? `${britanniaMap} · ${selectedMap?.name ?? 'UNKNOWN'}` : `${selectedDungeon?.name ?? 'UNKNOWN'} / ROOM ${dungeonRoom}` }}</dd>
          <dt>Party entry</dt><dd>{{ entryDirection }}</dd>
          <dt>Primary</dt><dd>{{ selectedMonster?.name ?? 'NONE' }} (#{{ monsterType }})</dd>
          <dt>Count</dt><dd>{{ countMode === 'exact' ? `${exactCount} exact` : countMode }}</dd>
          <dt>Friends</dt><dd>{{ friends && selectedMonster?.friend ? selectedMonster.friend.name : friends ? 'NONE DEFINED' : 'DISABLED' }}</dd>
          <dt>Seed</dt><dd>{{ seed || 'SESSION RNG' }}</dd>
        </dl>
        <template v-if="selectedMonster">
          <h4>Monster reference</h4>
          <dl class="monster-stat-grid"><dt>STR / DEX / INT</dt><dd>{{ selectedMonster.strength }} / {{ selectedMonster.dexterity }} / {{ selectedMonster.intelligence }}</dd><dt>ARM / DMG / HP</dt><dd>{{ selectedMonster.armour }} / {{ selectedMonster.damage }} / {{ selectedMonster.hitPoints }}</dd><dt>Authentic max group</dt><dd>{{ selectedMonster.maxGroup }}</dd></dl>
        </template>
        <p class="state-surgery-note">Starting commits a new combat scene. Arbitrary placement and active-combat editing are unavailable; positions are selected authoritatively from the combat map.</p>
      </aside>
    </div>
    <p v-else class="empty-state">WAITING FOR ENCOUNTER OPTIONS</p>

    <footer class="composer-actions">
      <span>{{ reviewed ? 'Review complete. Start will transition immediately.' : 'Review is required before start.' }}</span>
      <button v-if="reviewed" type="button" :disabled="pending" @click="reviewed = false">BACK TO DRAFT</button>
      <button type="submit" :class="{ 'danger-button': reviewed }" :disabled="!options || !valid || pending">{{ pending ? 'STARTING...' : reviewed ? 'START ENCOUNTER' : 'REVIEW ENCOUNTER' }}</button>
    </footer>
  </form>
</template>
