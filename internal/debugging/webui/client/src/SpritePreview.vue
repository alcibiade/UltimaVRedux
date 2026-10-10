<script setup lang="ts">
import { computed } from "vue"

const props = withDefaults(defineProps<{
  index: number
  size?: number
}>(), { size: 32 })

const token = document.querySelector<HTMLMetaElement>('meta[name="u5r-workbench-token"]')?.content ?? ""
const valid = computed(() => Number.isInteger(props.index) && props.index >= 0 && props.index < 512)
const style = computed(() => {
  const column = props.index % 32
  const row = Math.floor(props.index / 32)
  return {
    width: `${props.size}px`, height: `${props.size}px`,
    backgroundImage: valid.value ? `url(/api/tile-atlas.png?token=${encodeURIComponent(token)})` : "none",
    backgroundSize: `${32 * props.size}px ${16 * props.size}px`,
    backgroundPosition: `${-column * props.size}px ${-row * props.size}px`,
  }
})
</script>

<template><span class="sprite-preview" :class="{ unavailable: !valid }" :style="style" :aria-label="valid ? `Sprite ${index}` : 'Unavailable sprite'" /></template>
