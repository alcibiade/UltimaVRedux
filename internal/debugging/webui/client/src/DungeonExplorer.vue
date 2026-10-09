<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useInstrumentDraft } from "./instrument_session"

import type { DungeonExplorerCellData, DungeonExplorerStatusData } from "./approved_tool_views"
import type { Choice } from "./protocol"

interface DungeonDraft {
  dungeon: string
  floor: number
  x: number
  y: number
  facing: string
  lit: boolean
}

interface DungeonApply extends DungeonDraft {
  expectedLocationId: number
  expectedFloor: number
  expectedX: number
  expectedY: number
  expectedDungeonRevision: number
  expectedSourceFingerprint: string
}

interface GridCell {
  x: number
  y: number
  semantic?: DungeonExplorerCellData
}

const props = defineProps<{
  data?: DungeonExplorerStatusData
  updatedAt?: Date
  pending: boolean
  dungeons: Choice[]
  facings: Choice[]
}>()

const emit = defineEmits<{
  apply: [value: DungeonApply]
}>()

const draft = useInstrumentDraft<DungeonDraft | undefined>("dungeon.draft", undefined)
const editing = useInstrumentDraft("dungeon.editing", false)
const appliedDraft = ref<DungeonDraft>()
const floors = Array.from({ length: 8 }, (_, floor) => floor)
const gridCells = computed<GridCell[]>(() => {
  const byCoordinate = new Map((props.data?.cells ?? []).map((cell) => [`${cell.x}:${cell.y}`, cell]))
  return Array.from({ length: 64 }, (_, index) => {
    const x = index % 8
    const y = Math.floor(index / 8)
    return { x, y, semantic: byCoordinate.get(`${x}:${y}`) }
  })
})
const selectedCell = computed(() => props.data?.cells.find((cell) => cell.x === draft.value?.x && cell.y === draft.value?.y))
const valid = computed(() => Boolean(props.data && draft.value
  && props.dungeons.some((choice) => choice.value === draft.value?.dungeon)
  && floors.includes(draft.value.floor)
  && Number.isInteger(draft.value.x) && draft.value.x >= 0 && draft.value.x <= 7
  && Number.isInteger(draft.value.y) && draft.value.y >= 0 && draft.value.y <= 7
  && props.facings.some((choice) => choice.value === draft.value?.facing)))

watch([() => props.data, () => props.dungeons, () => props.facings], ([data]) => {
  if (!data) return
  if (appliedDraft.value && matchesStatus(appliedDraft.value, data)) {
    resetToStatus(data)
    appliedDraft.value = undefined
    return
  }
  if (!draft.value || !editing.value || matchesStatus(draft.value, data)) resetToStatus(data)
}, { immediate: true })

function draftForStatus(data: DungeonExplorerStatusData): DungeonDraft {
  if (!data.active) {
    return {
      dungeon: props.dungeons[0]?.value ?? "33",
      floor: 0,
      x: 4,
      y: 4,
      facing: props.facings.find((choice) => choice.value === "east")?.value ?? props.facings[0]?.value ?? "east",
      lit: true,
    }
  }
  return {
    dungeon: String(data.locationId),
    floor: data.floor,
    x: data.x,
    y: data.y,
    facing: props.facings.find((choice) => choice.label.toLocaleLowerCase() === data.facingLabel.toLocaleLowerCase())?.value ?? "",
    lit: data.lit,
  }
}

function resetToStatus(data = props.data): void {
  if (!data) return
  draft.value = draftForStatus(data)
  editing.value = false
}

function matchesStatus(value: DungeonDraft, data: DungeonExplorerStatusData): boolean {
  const facing = props.facings.find((choice) => choice.value === value.facing)?.label ?? ""
  return data.active && Number(value.dungeon) === data.locationId && value.floor === data.floor
    && value.x === data.x && value.y === data.y && facing.toLocaleLowerCase() === data.facingLabel.toLocaleLowerCase()
    && value.lit === data.lit
}

function updateSelect(field: "dungeon" | "floor" | "facing" | "lit", event: Event): void {
  if (!draft.value) return
  const raw = (event.target as HTMLSelectElement).value
  const value = field === "floor" ? Number(raw) : field === "lit" ? raw === "true" : raw
  draft.value = { ...draft.value, [field]: value }
  markEdited()
}

function updateCoordinate(axis: "x" | "y", event: Event): void {
  if (!draft.value) return
  draft.value = { ...draft.value, [axis]: (event.target as HTMLInputElement).valueAsNumber }
  markEdited()
}

function clampCoordinate(axis: "x" | "y"): void {
  if (!draft.value || !Number.isFinite(draft.value[axis])) return
  draft.value = { ...draft.value, [axis]: Math.min(7, Math.max(0, Math.round(draft.value[axis]))) }
  markEdited()
}

function selectCell(cell: GridCell): void {
  if (!draft.value) return
  draft.value = { ...draft.value, x: cell.x, y: cell.y }
  markEdited()
}

function markEdited(): void {
  editing.value = Boolean(!props.data || !draft.value || !matchesStatus(draft.value, props.data))
  appliedDraft.value = undefined
}

function rawHex(raw: number): string {
  return `0x${raw.toString(16).toUpperCase().padStart(2, "0")}`
}

function kindClass(cell: GridCell): string {
  return cell.semantic ? `kind-${cell.semantic.kindName.replaceAll("_", "-")}` : "kind-unavailable"
}

function apply(): void {
  const source = props.data
  if (!source || !draft.value || !source.setupAvailable || props.pending || !valid.value) return
  appliedDraft.value = { ...draft.value }
  emit("apply", {
    ...draft.value,
    expectedLocationId: source.locationId,
    expectedFloor: source.floor,
    expectedX: source.x,
    expectedY: source.y,
    expectedDungeonRevision: source.revision,
    expectedSourceFingerprint: source.sourceFingerprint,
  })
}
</script>

<template>
  <section class="approved-view dungeon-explorer" aria-label="Dungeon Explorer">
    <header class="approved-view-header">
      <div><span>PACKED DUNGEON SEMANTICS</span><h3>Dungeon Explorer</h3></div>
      <small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small>
    </header>

    <template v-if="data && draft">
      <section class="dungeon-status-strip" :data-active="data.active">
        <div><span>STATUS</span><strong>{{ data.active ? 'ACTIVE' : 'INACTIVE' }}</strong></div>
        <div><span>LOCATION</span><strong>{{ data.locationName }} #{{ data.locationId }}</strong></div>
        <div><span>LEVEL</span><strong>{{ data.active ? `LEVEL ${data.displayLevel}` : `FLOOR ${data.floor}` }}</strong></div>
        <div><span>FACING</span><strong>{{ data.facingLabel || 'N/A' }}</strong><small>{{ data.lastDirectionLabel || 'NO DUNGEON DIRECTION' }}</small></div>
        <div><span>LIGHT</span><strong>{{ data.lit ? 'LIT' : 'DARK' }}</strong><small>TORCH {{ data.torchMinutesRemaining }} / MAGIC {{ data.magicLightMinutesRemaining }}</small></div>
        <div><span>REVISION</span><strong>{{ data.revision }}</strong></div>
      </section>

      <div class="dungeon-explorer-layout">
        <section class="approved-panel dungeon-map-panel">
          <header><strong>8 x 8 FLOOR GRID</strong><small>Y-MAJOR / CLICK TO TARGET</small></header>
          <div class="dungeon-grid" role="grid" aria-label="Dungeon floor cells">
            <button
              v-for="(cell, index) in gridCells"
              :key="`${cell.x}:${cell.y}`"
              type="button"
              role="gridcell"
              :data-cell-index="index"
              :data-x="cell.x"
              :data-y="cell.y"
              :data-kind="cell.semantic?.kindName"
              :class="[kindClass(cell), { selected: draft.x === cell.x && draft.y === cell.y, player: cell.semantic?.player }]"
              :aria-label="`Cell ${cell.x}, ${cell.y}${cell.semantic ? `, ${cell.semantic.kindName}, ${rawHex(cell.semantic.raw)}` : ''}`"
              @click="selectCell(cell)"
            >
              <i v-if="cell.semantic?.player">P</i><strong>{{ cell.semantic ? rawHex(cell.semantic.raw).slice(2) : '--' }}</strong><small>{{ cell.x }},{{ cell.y }}</small>
            </button>
          </div>
        </section>

        <section class="approved-panel dungeon-cell-inspector">
          <header><strong>SELECTED CELL</strong><small>{{ draft.x }}, {{ draft.y }}</small></header>
          <dl v-if="selectedCell" class="compact-metrics vertical">
            <div><dt>Raw</dt><dd>{{ rawHex(selectedCell.raw) }}</dd></div>
            <div><dt>Kind</dt><dd>{{ selectedCell.kindName }} #{{ selectedCell.kindId }}</dd></div>
            <div><dt>Payload</dt><dd>{{ selectedCell.payload }}</dd></div>
            <div><dt>Bit 0x08</dt><dd>{{ selectedCell.bit08 ? 'SET' : 'CLEAR' }}</dd></div>
            <div><dt>Player</dt><dd>{{ selectedCell.player ? 'YES' : 'NO' }}</dd></div>
          </dl>
          <p v-else class="empty-state">NO SEMANTIC CELL LOADED</p>
        </section>
      </div>

      <form class="approved-panel dungeon-setup-form" @submit.prevent="apply">
        <header><strong>ATOMIC DUNGEON SETUP</strong><small>REVIEWED WORLD + REVISION PRECONDITIONS</small></header>
        <div class="dungeon-setup-controls">
          <label>Dungeon<select :value="draft.dungeon" aria-label="Dungeon" @change="updateSelect('dungeon', $event)"><option v-for="dungeon in dungeons" :key="dungeon.value" :value="dungeon.value">{{ dungeon.label }}</option></select></label>
          <label>Floor<select :value="draft.floor" aria-label="Dungeon floor" @change="updateSelect('floor', $event)"><option v-for="floor in floors" :key="floor" :value="floor">Level {{ floor + 1 }} (internal {{ floor }})</option></select></label>
          <label>X<input :value="draft.x" aria-label="Dungeon X" type="number" min="0" max="7" step="1" :aria-invalid="!valid" @input="updateCoordinate('x', $event)" @change="clampCoordinate('x')" /></label>
          <label>Y<input :value="draft.y" aria-label="Dungeon Y" type="number" min="0" max="7" step="1" :aria-invalid="!valid" @input="updateCoordinate('y', $event)" @change="clampCoordinate('y')" /></label>
          <label>Facing<select :value="draft.facing" aria-label="Dungeon facing" @change="updateSelect('facing', $event)"><option v-for="facing in facings" :key="facing.value" :value="facing.value">{{ facing.label }}</option></select></label>
          <label>Light<select :value="String(draft.lit)" aria-label="Dungeon lighting" @change="updateSelect('lit', $event)"><option value="true">Lit</option><option value="false">Dark</option></select></label>
        </div>
        <p v-if="!valid" class="field-error">Choose a listed dungeon and facing with whole floor and coordinates from 0 through 7.</p>
        <p v-if="!data.setupAvailable" class="disabled-reason" role="status">{{ data.setupUnavailableReason || 'Dungeon setup is unavailable in the current scene.' }}</p>
        <footer><span>REVIEW {{ data.locationName }} / {{ data.floor }} / {{ data.x }},{{ data.y }} / REV {{ data.revision }}</span><div><button type="button" class="reset-button" :disabled="pending" @click="resetToStatus()">RESET</button><button type="submit" :disabled="!data.setupAvailable || pending || !valid">{{ pending ? 'APPLYING...' : 'APPLY SETUP' }}</button></div></footer>
      </form>
    </template>
    <p v-else class="empty-state">WAITING FOR DUNGEON STATUS</p>
  </section>
</template>
