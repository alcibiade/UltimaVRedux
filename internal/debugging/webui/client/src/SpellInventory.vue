<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft, useInstrumentError } from "./instrument_session"

import type { SpellInventoryStatusData } from "./approved_tool_views"

interface SpellMutation { commandID: string; argumentsByID: Record<string, string> }
interface ReviewedSpell {
  id: number
  name: string
  targetQuantity: string
  expectedQuantity: number
}

const props = defineProps<{ data?: SpellInventoryStatusData; updatedAt?: Date; pending: boolean }>()
const emit = defineEmits<{ mutate: [mutation: SpellMutation] }>()
const drafts = useInstrumentDraft<Record<number, string>>("spells.quantities", {})
const dirtySpellIDs = useInstrumentDraft("spells.dirty", new Set<number>())
const reviewedSpell = ref<ReviewedSpell>()
const reviewedPresetSnapshot = ref<string>()
const effectTurnsDraft = useInstrumentDraft("spells.turns", "1")
const lightMinutesDraft = useInstrumentDraft("spells.light", "1")
const issue = ref("")
const reportError = useInstrumentError()
function disarm(): void { reviewedSpell.value = undefined; reviewedPresetSnapshot.value = undefined }
invalidateInstrumentReview(disarm)
function fail(message: string): void { disarm(); issue.value = message; reportError(message) }

const circles = computed(() => Array.from({ length: 8 }, (_, index) => ({
  circle: index + 1,
  spells: (props.data?.spells ?? []).filter((spell) => spell.circle === index + 1),
})))

watch(() => props.data?.snapshotFingerprint, () => {
  disarm()
  if (!props.data) return
  const nextDrafts = { ...drafts.value }
  const nextDirty = new Set(dirtySpellIDs.value)
  for (const spell of props.data.spells) {
    if (nextDirty.has(spell.id) && nextDrafts[spell.id] === String(spell.quantity)) nextDirty.delete(spell.id)
    if (!nextDirty.has(spell.id)) nextDrafts[spell.id] = String(spell.quantity)
  }
  drafts.value = nextDrafts
  dirtySpellIDs.value = nextDirty
}, { immediate: true })

function updateDraft(id: number, value: string, quantity: number): void {
  drafts.value = { ...drafts.value, [id]: value }
  const nextDirty = new Set(dirtySpellIDs.value)
  if (value === String(quantity)) nextDirty.delete(id); else nextDirty.add(id)
  dirtySpellIDs.value = nextDirty
  if (reviewedSpell.value?.id === id) reviewedSpell.value = undefined
}

function stageSpell(spell: SpellInventoryStatusData["spells"][number]): void {
  const targetQuantity = drafts.value[spell.id] ?? String(spell.quantity)
  const parsed = Number(targetQuantity)
  if (!targetQuantity.trim() || !Number.isInteger(parsed) || parsed < 0 || parsed > spell.maximum || parsed === spell.quantity) {
    fail(`${spell.name}: enter a changed whole quantity from 0 to ${spell.maximum}. Blank is not zero.`)
    return
  }
  issue.value = ""
  reviewedPresetSnapshot.value = undefined
  reviewedSpell.value = { id: spell.id, name: spell.name, targetQuantity, expectedQuantity: spell.quantity }
}

function applySpell(): void {
  if (!reviewedSpell.value) return
  emit("mutate", {
    commandID: "inventory.spell.set",
    argumentsByID: {
      spell: String(reviewedSpell.value.id),
      targetQuantity: reviewedSpell.value.targetQuantity,
      expectedQuantity: String(reviewedSpell.value.expectedQuantity),
    },
  })
  reviewedSpell.value = undefined
}

function stagePreset(): void {
  if (!props.data) return
  reviewedSpell.value = undefined
  reviewedPresetSnapshot.value = props.data.snapshotFingerprint
}

function applyPreset(): void {
  if (!reviewedPresetSnapshot.value) return
  emit("mutate", {
    commandID: "inventory.spell.preset-max",
    argumentsByID: { expectedSnapshot: reviewedPresetSnapshot.value },
  })
  reviewedPresetSnapshot.value = undefined
}

function setEffectTurns(turns: number): void {
	if (!props.data?.activeSpell.active || !Number.isInteger(turns) || turns < 1 || turns > 9999) { fail("Exact turns: choose an active effect and a whole number from 1 to 9999."); return }
  emit("mutate", {
    commandID: "spell.effect.set-remaining",
    argumentsByID: {
		effectKind: "active_spell",
		expectedEffect: props.data.activeSpell.effectType,
		expectedTurns: String(props.data.activeSpell.turnsRemaining),
		expectedLightMinutes: String(props.data.magicLight.minutesRemaining),
      remaining: String(turns),
    },
  })
}

function setLightMinutes(minutes: number): void {
  if (!props.data || !Number.isInteger(minutes) || minutes < 0 || minutes > 9999) { fail("Exact minutes: enter a whole number from 0 to 9999."); return }
  emit("mutate", {
    commandID: "spell.effect.set-remaining",
    argumentsByID: {
		effectKind: "magic_light",
		expectedEffect: props.data.activeSpell.effectType,
		expectedTurns: String(props.data.activeSpell.turnsRemaining),
		expectedLightMinutes: String(props.data.magicLight.minutesRemaining),
		remaining: String(minutes),
    },
  })
}

function activateEffect(effect: "light" | "time_stop"): void {
	if (!props.data) return
	emit("mutate", {
		commandID: "spell.effect.activate",
		argumentsByID: {
			effect,
			expectedEffect: props.data.activeSpell.effectType,
			expectedTurns: String(props.data.activeSpell.turnsRemaining),
			expectedLightMinutes: String(props.data.magicLight.minutesRemaining),
		},
	})
}
</script>

<template>
  <section class="approved-view spell-inventory">
    <header class="approved-view-header">
      <div><span>AUTHORITATIVE PREPARED SPELLS</span><h3>Prepared Spell Inventory</h3></div>
      <small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
    </header>
    <p v-if="issue" class="error-line" role="alert">{{ issue }}</p>
    <p v-if="!data" class="read-only-notice">Loading prepared spells and the live duration effect.</p>
    <template v-else>
		<section class="duration-effect-card" :class="{ active: data.magicLight.active }">
		  <header><span>MAGICAL LIGHT</span><strong>{{ data.magicLight.active ? "ACTIVE" : "INACTIVE" }}</strong></header>
		  <div class="duration-effect-details">
			<p><strong>IN LOR / VAS LOR</strong><span>{{ data.magicLight.minutesRemaining }} MINUTES REMAINING</span></p>
			<div class="duration-effect-controls">
			  <button type="button" :disabled="pending || data.magicLight.minutesRemaining === 0" @click="setLightMinutes(0)">EXTINGUISH</button>
			  <label>Exact minutes<input v-model="lightMinutesDraft" type="number" min="0" max="9999" step="1" /></label>
			  <button type="button" :disabled="pending" @click="String(lightMinutesDraft).trim() ? setLightMinutes(Number(lightMinutesDraft)) : fail('Exact minutes: enter a whole number. Blank is not zero.')">SET EXACT</button>
			</div>
		  </div>
		</section>

		<section class="duration-effect-card" :class="{ active: data.activeSpell.active }">
		  <header><span>ACTIVE DURATION SPELL</span><strong>{{ data.activeSpell.active ? data.activeSpell.friendlyName : "INACTIVE" }}</strong></header>
		  <template v-if="data.activeSpell.active">
			<div class="duration-effect-details">
			  <p><strong>{{ data.activeSpell.incantations.length ? data.activeSpell.incantations.join(" / ") : "UNKNOWN INCANTATION" }}</strong><span>{{ data.activeSpell.turnsRemaining }} TURNS REMAINING</span></p>
			  <p>{{ data.activeSpell.description }}</p>
			  <small v-if="data.activeSpell.sourceAmbiguous">Source incantation is not retained.</small>
			<div class="duration-effect-controls">
				<button type="button" :disabled="pending || data.activeSpell.turnsRemaining === 1" @click="setEffectTurns(1)">SET 1 TURN</button>
				<label>Exact turns<input v-model="effectTurnsDraft" type="number" min="1" max="9999" step="1" /></label>
				<button type="button" :disabled="pending" @click="String(effectTurnsDraft).trim() ? setEffectTurns(Number(effectTurnsDraft)) : fail('Exact turns: enter a whole number. Blank is not zero.')">SET EXACT</button>
			</div>
          </div>
        </template>
		  <p v-else class="inactive-effect">No duration spell effect is active.</p>
		<div class="duration-activation-controls">
			<span>ACTIVATE AUTHENTIC SCROLL DURATION</span>
			<button type="button" :disabled="pending" @click="activateEffect('light')">LIGHT · 240</button>
			<button type="button" :disabled="pending" @click="activateEffect('time_stop')">TIME STOP · 20</button>
		</div>
      </section>

      <section class="spell-preset">
        <p>Set every castable prepared spell to its maximum. Nox is excluded.</p>
        <button type="button" :disabled="pending" @click="stagePreset">STAGE: SET ALL 48 SPELLS TO 99</button>
      </section>

      <div class="spell-circle-list">
        <section v-for="circle in circles" :key="circle.circle" class="spell-circle">
          <header><strong>CIRCLE {{ circle.circle }}</strong><small>{{ circle.spells.length }} SPELLS</small></header>
          <div v-for="spell in circle.spells" :key="spell.id" class="spell-inventory-row">
            <div><strong>{{ spell.name }}</strong><p>{{ spell.description }}</p><small>{{ spell.targeting }} / {{ spell.context }}</small></div>
            <span class="spell-current"><small>CURRENT</small><strong>{{ spell.quantity }}</strong></span>
            <label><span>TARGET</span><input :data-spell-quantity="spell.id" type="number" min="0" max="99" step="1" :value="drafts[spell.id] ?? spell.quantity" @input="updateDraft(spell.id, ($event.target as HTMLInputElement).value, spell.quantity)" /></label>
            <button type="button" :disabled="pending || !dirtySpellIDs.has(spell.id)" @click="stageSpell(spell)">STAGE</button>
          </div>
        </section>
      </div>

      <section v-if="reviewedSpell || reviewedPresetSnapshot" class="locker-review spell-review" aria-live="polite">
        <div v-if="reviewedSpell">
          <p>Set {{ reviewedSpell.name }} from {{ reviewedSpell.expectedQuantity }} to {{ reviewedSpell.targetQuantity }}.</p>
          <small>The current quantity is sent as a stale-state precondition.</small>
        </div>
        <div v-else>
          <p>Set all 48 castable spells to 99. Nox is excluded.</p>
          <small>The complete reviewed snapshot is checked first. If any spell changed, the whole operation is stale and nothing is applied.</small>
        </div>
        <button type="button" :disabled="pending" @click="reviewedSpell ? applySpell() : applyPreset()">{{ reviewedSpell ? "APPLY STAGED SPELL" : "APPLY ALL 48 TO 99" }}</button>
      </section>
    </template>
  </section>
</template>
