<script setup lang="ts">
import type { DebugOptionsStatusData } from "./approved_tool_views"

defineProps<{ data?: DebugOptionsStatusData; updatedAt?: Date; pending: boolean }>()
const emit = defineEmits<{ setOption: [commandID: string, enabled: boolean] }>()
</script>

<template>
  <section class="approved-view debug-options-console" aria-label="Debug options">
    <header class="approved-view-header">
      <div><span>EXPLICIT RETRY-SAFE FLAGS</span><h3>Debug Options</h3></div>
      <small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
    </header>
    <p v-if="!data" class="empty-state">WAITING FOR DEBUG OPTIONS</p>
    <div v-else class="option-switch-grid">
      <article>
        <div><strong>Free Move</strong><p>Bypass normal terrain and object movement restrictions. Also available in the top bar.</p></div>
        <button type="button" role="switch" :aria-checked="data.freeMove" :disabled="pending" @click="emit('setOption', 'movement.free-move.set', !data.freeMove)">{{ data.freeMove ? 'ON' : 'OFF' }}</button>
      </article>
      <article>
        <div><strong>Text Placeholders</strong><p>Reveal empty and newline placeholders in game-window conversation output.</p></div>
        <button type="button" role="switch" :aria-checked="data.textPlaceholdersEnabled" :disabled="pending" @click="emit('setOption', 'overlay.text-placeholders.set', !data.textPlaceholdersEnabled)">{{ data.textPlaceholdersEnabled ? 'ON' : 'OFF' }}</button>
      </article>
      <article>
        <div><strong>Dialog Bounds</strong><p>Draw diagnostic boundaries around dialog text output regions.</p></div>
        <button type="button" role="switch" :aria-checked="data.dialogBoundsEnabled" :disabled="pending" @click="emit('setOption', 'overlay.dialog-bounds.set', !data.dialogBoundsEnabled)">{{ data.dialogBoundsEnabled ? 'ON' : 'OFF' }}</button>
      </article>
      <article>
        <div><strong>Unlimited Casting Resources</strong><p>Ignore prepared-spell and current MP costs for magical characters.</p></div>
        <button type="button" role="switch" :aria-checked="data.unlimitedCastingResources" :disabled="pending" @click="emit('setOption', 'magic.unlimited-casting-resources.set', !data.unlimitedCastingResources)">{{ data.unlimitedCastingResources ? 'ON' : 'OFF' }}</button>
      </article>
      <article>
        <div><strong>Spell Level Bypass</strong><p>Allow magical characters to use spell circles above their current level.</p></div>
        <button type="button" role="switch" :aria-checked="data.bypassSpellLevelRestrictions" :disabled="pending" @click="emit('setOption', 'magic.spell-level-restrictions-bypass.set', !data.bypassSpellLevelRestrictions)">{{ data.bypassSpellLevelRestrictions ? 'ON' : 'OFF' }}</button>
      </article>
    </div>
  </section>
</template>
