<script setup lang="ts">
import { computed } from "vue"

import runicFontURL from "../../../../text/assets/runic_font.png"

const props = withDefaults(defineProps<{
  phaseIndex: number
  label: string
  size?: number
}>(), {
  size: 24,
})

const phase = computed(() => Math.min(7, Math.max(0, Math.trunc(props.phaseIndex))))
const glyphIndex = computed(() => 0x30 + phase.value)
const glyphStyle = computed(() => ({
  width: `${props.size}px`,
  height: `${props.size}px`,
  backgroundImage: `url(${runicFontURL})`,
  backgroundPosition: `${-phase.value * props.size}px ${-3 * props.size}px`,
  backgroundSize: `${16 * props.size}px ${8 * props.size}px`,
}))
</script>

<template>
  <span
    class="moon-phase-glyph"
    role="img"
    :aria-label="label"
    :title="label"
    :data-glyph-index="`0x${glyphIndex.toString(16)}`"
    :style="glyphStyle"
  />
</template>
