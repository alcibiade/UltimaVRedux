<script setup lang="ts">
import type { LoggingStatusData } from "./approved_tool_views"

interface LoggingMutation { commandID: string; argumentsByID: Record<string, string> }

defineProps<{ data?: LoggingStatusData; updatedAt?: Date; pending: boolean }>()
const emit = defineEmits<{ mutate: [mutation: LoggingMutation]; refresh: [] }>()

function setBoolean(commandID: string, enabled: boolean): void {
  emit("mutate", { commandID, argumentsByID: { enabled: String(enabled) } })
}
</script>

<template>
  <section class="approved-view logging-console" aria-label="Logging console">
    <header class="approved-view-header">
      <div><span>PROCESS DIAGNOSTICS</span><h3>Logging Control</h3></div>
      <small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
    </header>
    <div v-if="!data" class="empty-state"><p>WAITING FOR LOGGING STATUS</p><button type="button" :disabled="pending" @click="emit('refresh')">LOAD STATUS</button></div>
    <template v-else>
      <p class="logging-persistence-note">CHANGES ARE SAVED AUTOMATICALLY AND RESTORED ON THE NEXT LAUNCH.</p>
      <section class="logging-summary">
        <article :class="data.enabled ? 'is-enabled' : 'is-disabled'"><span>Master</span><strong>{{ data.enabled ? "ENABLED" : "DISABLED" }}</strong><button type="button" :disabled="pending" @click="setBoolean('logging.enabled.set', !data.enabled)">{{ data.enabled ? "DISABLE" : "ENABLE" }}</button></article>
        <article :class="data.stdoutEnabled ? 'is-enabled' : 'is-disabled'"><span>Stdout</span><strong>{{ data.stdoutEnabled ? "ENABLED" : "DISABLED" }}</strong><button type="button" :disabled="pending" @click="setBoolean('logging.stdout.set', !data.stdoutEnabled)">{{ data.stdoutEnabled ? "DISABLE" : "ENABLE" }}</button></article>
        <article><label for="logging-level">Minimum level</label><select id="logging-level" :value="data.minimumLevel" :disabled="pending" @change="emit('mutate', { commandID: 'logging.level.set', argumentsByID: { level: ($event.target as HTMLSelectElement).value } })"><option v-for="level in ['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']" :key="level" :value="level">{{ level }}</option></select></article>
      </section>
      <p v-if="!data.enabled" class="deferred-control"><strong>MASTER DISABLED</strong> Category choices remain configured but no categorized messages are currently accepted.</p>
      <section class="approved-panel logging-categories">
        <header><div><strong>Category channels</strong><small>{{ Object.values(data.categoryStates).filter(Boolean).length }} / {{ Object.keys(data.categoryStates).length }} ENABLED</small></div><div class="logging-category-actions"><button type="button" :disabled="pending || Object.values(data.categoryStates).every(Boolean)" @click="setBoolean('logging.categories.set-all', true)">ENABLE ALL</button><button type="button" :disabled="pending || Object.values(data.categoryStates).every((enabled) => !enabled)" @click="setBoolean('logging.categories.set-all', false)">DISABLE ALL</button></div></header>
        <div><article v-for="(enabled, category) in data.categoryStates" :key="category" :class="enabled ? 'is-enabled' : 'is-disabled'"><span>{{ category }}</span><strong>{{ enabled ? "ON" : "OFF" }}</strong><button type="button" :disabled="pending" @click="emit('mutate', { commandID: 'logging.category.set', argumentsByID: { category: String(category), enabled: String(!enabled) } })">{{ enabled ? "DISABLE" : "ENABLE" }}</button></article></div>
      </section>
      <section class="instrument-setting"><div><strong>READ-ONLY LOG WINDOW</strong><p>Open a separate multi-reader browser stream without consuming the Workbench controller connection.</p></div><a class="button-link" href="/logs" target="_blank" rel="noopener">OPEN LOG WINDOW</a></section>
    </template>
  </section>
</template>
