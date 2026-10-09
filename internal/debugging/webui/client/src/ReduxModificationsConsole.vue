<script setup lang="ts">
import type { ReduxModificationsStatusData } from "./approved_tool_views"

defineProps<{ data?: ReduxModificationsStatusData; updatedAt?: Date; pending: boolean }>()
const emit = defineEmits<{
  setRepeatedDivision: [enabled: boolean]
  setPersistentDiscoveries: [enabled: boolean]
  setSubduedVisibility: [enabled: boolean]
  setSmoothSurfaceMovement: [enabled: boolean]
  setFlexibleSextantUse: [enabled: boolean]
  refresh: []
}>()
</script>

<template>
  <section class="approved-view redux-modifications-console" aria-label="Redux modifications">
    <header class="approved-view-header">
      <div><span>PERSISTED GAMEPLAY RULES</span><h3>Redux Modifications</h3></div>
      <span>
        <small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
        <button type="button" :disabled="pending" data-redux-refresh @click="emit('refresh')">REFRESH</button>
      </span>
    </header>
    <p class="redux-persistence-note">These deliberate Redux behavior choices are saved locally and persist across restarts. They are not temporary debug overrides.</p>
    <p v-if="!data" class="empty-state">WAITING FOR REDUX MODIFICATIONS</p>
    <section v-else class="approved-panel">
      <header><span>COMBAT</span><strong>Monster Behavior</strong></header>
      <div class="option-switch-grid">
        <article>
          <div>
            <strong>Repeated Monster Division</strong>
            <p>Redux Limited permits one successful child per encounter-created Slime or Gargoyle. Original Repeated lets parents and descendants continue splitting.</p>
            <small>Gameplay.Combat.AllowRepeatedMonsterDivision</small>
          </div>
          <div class="redux-setting-control">
            <span class="redux-mode-badge" :data-mode="data.combat.allowRepeatedMonsterDivision ? 'original' : 'redux'">{{ data.combat.allowRepeatedMonsterDivision ? "ORIGINAL REPEATED" : "REDUX LIMITED" }}</span>
            <button
              type="button"
              role="switch"
              data-redux-repeated-division
              :aria-checked="data.combat.allowRepeatedMonsterDivision"
              :disabled="pending"
              @click="emit('setRepeatedDivision', !data.combat.allowRepeatedMonsterDivision)"
            >{{ data.combat.allowRepeatedMonsterDivision ? "ON" : "OFF" }}</button>
          </div>
        </article>
      </div>
    </section>
    <section v-if="data" class="approved-panel">
      <header><span>DUNGEONS</span><strong>Exploration and Visibility</strong></header>
      <div class="option-switch-grid">
        <article>
          <div>
            <strong>Persistent Hidden Discoveries</strong>
            <p>Redux Persistent remembers found hidden doors and pits between visits. Original Per Visit restores them when a dungeon is entered again.</p>
            <small>Gameplay.Dungeons.PersistHiddenDiscoveries</small>
          </div>
          <div class="redux-setting-control">
            <span class="redux-mode-badge" :data-mode="data.dungeons.persistHiddenDungeonDiscoveries ? 'redux' : 'original'">{{ data.dungeons.persistHiddenDungeonDiscoveries ? "REDUX PERSISTENT" : "ORIGINAL PER VISIT" }}</span>
            <button
              type="button"
              role="switch"
              data-redux-persistent-discoveries
              :aria-checked="data.dungeons.persistHiddenDungeonDiscoveries"
              :disabled="pending"
              @click="emit('setPersistentDiscoveries', !data.dungeons.persistHiddenDungeonDiscoveries)"
            >{{ data.dungeons.persistHiddenDungeonDiscoveries ? "ON" : "OFF" }}</button>
          </div>
        </article>
        <article>
          <div>
            <strong>Subdued Unlit Visibility</strong>
            <p>Redux Subdued shows faint geometry without granting light. Original Black keeps an unlit dungeon completely black.</p>
            <small>Gameplay.Dungeons.ShowSubduedUnlitDungeons</small>
          </div>
          <div class="redux-setting-control">
            <span class="redux-mode-badge" :data-mode="data.dungeons.showSubduedUnlitDungeons ? 'redux' : 'original'">{{ data.dungeons.showSubduedUnlitDungeons ? "REDUX SUBDUED" : "ORIGINAL BLACK" }}</span>
            <button
              type="button"
              role="switch"
              data-redux-subdued-visibility
              :aria-checked="data.dungeons.showSubduedUnlitDungeons"
              :disabled="pending"
              @click="emit('setSubduedVisibility', !data.dungeons.showSubduedUnlitDungeons)"
            >{{ data.dungeons.showSubduedUnlitDungeons ? "ON" : "OFF" }}</button>
          </div>
        </article>
      </div>
    </section>
    <section v-if="data" class="approved-panel">
      <header><span>NAVIGATION</span><strong>Sextant Policy</strong></header>
      <div class="option-switch-grid">
        <article>
          <div>
            <strong>Flexible sextant use</strong>
            <p>Off by default: original nighttime world-only use. On allows daytime use and settlements on all floors, including cellars. Dungeons and the Underworld remain excluded, and a sextant must still be owned.</p>
            <small>Gameplay.Navigation.FlexibleSextantUse</small>
          </div>
          <div class="redux-setting-control">
            <span class="redux-mode-badge" :data-mode="data.navigation.flexibleSextantUse ? 'redux' : 'original'">{{ data.navigation.flexibleSextantUse ? "REDUX FLEXIBLE" : "ORIGINAL NIGHT WORLD ONLY" }}</span>
            <button
              type="button"
              role="switch"
              data-redux-flexible-sextant-use
              :aria-checked="data.navigation.flexibleSextantUse"
              :disabled="pending"
              @click="emit('setFlexibleSextantUse', !data.navigation.flexibleSextantUse)"
            >{{ data.navigation.flexibleSextantUse ? "ON" : "OFF" }}</button>
          </div>
        </article>
      </div>
    </section>
    <section v-if="data" class="approved-panel">
      <header><span>PRESENTATION</span><strong>Surface Camera</strong></header>
      <div class="option-switch-grid">
        <article>
          <div>
            <strong>Smooth Surface Movement</strong>
            <p>Optional camera scrolling while walking, riding, flying or boating in towns, overworld and underworld. Mounts animate 25% faster; automatic sailing completes each scroll within its existing cadence. Gameplay timing, combat and dungeons are unchanged. Off by default.</p>
            <small>Gameplay.Presentation.SmoothSurfaceMovement</small>
          </div>
          <div class="redux-setting-control">
            <button
              type="button"
              role="switch"
              data-redux-smooth-surface-movement
              :aria-checked="data.presentation.smoothSurfaceMovement"
              :disabled="pending"
              @click="emit('setSmoothSurfaceMovement', !data.presentation.smoothSurfaceMovement)"
            >{{ data.presentation.smoothSurfaceMovement ? "ON" : "OFF" }}</button>
          </div>
        </article>
      </div>
    </section>
  </section>
</template>
