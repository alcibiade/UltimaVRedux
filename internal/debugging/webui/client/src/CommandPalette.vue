<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"

import { rankNavigationSurfaces, recentNavigationSurfaces, type NavigationSurface } from "./command_navigation"

const props = defineProps<{ surfaces: NavigationSurface[]; recents: string[] }>()
const emit = defineEmits<{ close: []; activate: [surface: NavigationSurface] }>()

const query = ref("")
const activeIndex = ref(0)
const input = ref<HTMLInputElement | null>(null)
const dialog = ref<HTMLElement | null>(null)
const matches = computed(() => query.value.trim()
  ? rankNavigationSurfaces(props.surfaces, query.value)
  : recentNavigationSurfaces(props.surfaces, props.recents))
const visibleMatches = computed(() => matches.value.slice(0, 8))
const activeID = computed(() => visibleMatches.value[activeIndex.value]?.id)

watch(visibleMatches, () => { activeIndex.value = 0 })

function activate(surface = visibleMatches.value[activeIndex.value]): void {
  if (surface) emit("activate", surface)
}

function onInputKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") emit("close")
  else if (event.key === "ArrowDown") activeIndex.value = Math.min(activeIndex.value + 1, visibleMatches.value.length - 1)
  else if (event.key === "ArrowUp") activeIndex.value = Math.max(activeIndex.value - 1, 0)
  else if (event.key === "Home") activeIndex.value = 0
  else if (event.key === "End") activeIndex.value = Math.max(visibleMatches.value.length - 1, 0)
  else if (event.key === "Enter") activate()
  else return
  event.preventDefault()
}

function trapFocus(event: KeyboardEvent): void {
  if (event.key !== "Tab") return
  const controls = [...(dialog.value?.querySelectorAll<HTMLElement>("input, button:not(:disabled):not([tabindex='-1']), [tabindex='0']") ?? [])]
  if (controls.length === 0) return
  const index = controls.indexOf(document.activeElement as HTMLElement)
  controls[(index + (event.shiftKey ? -1 : 1) + controls.length) % controls.length]?.focus()
  event.preventDefault()
}

onMounted(() => void nextTick(() => input.value?.focus()))
</script>

<template>
  <section class="command-palette-modal" role="dialog" aria-modal="true" aria-labelledby="command-palette-title" data-command-palette @keydown="trapFocus">
    <div ref="dialog" class="command-palette-dialog">
      <header><div><span>NAVIGATION ONLY</span><h2 id="command-palette-title">Go to Workbench Surface</h2></div><button type="button" aria-label="Close command palette" @click="emit('close')">ESC</button></header>
      <label class="sr-only" for="command-palette-input">Find a workspace surface</label>
      <input
        id="command-palette-input"
        ref="input"
        v-model="query"
        role="combobox"
        aria-autocomplete="list"
        aria-controls="command-palette-results"
        :aria-expanded="visibleMatches.length > 0"
        :aria-activedescendant="activeID ? `command-palette-${activeID}` : undefined"
        placeholder="Find a workspace, instrument, or intent…"
        @keydown="onInputKeydown"
      />
      <p class="command-palette-context">{{ query.trim() ? `${visibleMatches.length} MATCHES` : props.recents.length ? 'RECENT SURFACES' : 'TYPE TO FIND A SURFACE' }} · READ-ONLY REFERENCES LOAD AUTOMATICALLY · NO MUTATIONS RUN</p>
      <div id="command-palette-results" role="listbox" aria-label="Workbench navigation results">
        <button
          v-for="(surface, index) in visibleMatches"
          :id="`command-palette-${surface.id}`"
          :key="surface.id"
          type="button"
          role="option"
          tabindex="-1"
          :aria-selected="index === activeIndex"
          :class="{ active: index === activeIndex }"
          :data-palette-surface="surface.id"
          @mouseenter="activeIndex = index"
          @click="activate(surface)"
        ><span>{{ surface.workspaceLabel }}</span><strong>{{ surface.label }}</strong><small>{{ surface.description }}</small></button>
      </div>
      <p v-if="query.trim() && visibleMatches.length === 0" class="empty-state">NO MATCHING WORKBENCH SURFACES</p>
    </div>
  </section>
</template>
