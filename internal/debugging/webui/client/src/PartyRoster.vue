<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { PartyCharacterInspectData, PartyRosterCharacterData } from "./approved_tool_views"

const props = defineProps<{
  data?: PartyCharacterInspectData
  pendingCommands: string[]
  karmaPending: boolean
}>()

const emit = defineEmits<{
  mutate: [commandID: string, argumentsByID: Record<string, string>]
}>()

const statusDraft = useInstrumentDraft("roster.status", "good")
const selectedRosterIndices = useInstrumentDraft<number[]>("roster.targets", [])
const inactiveRosterIndex = useInstrumentDraft("roster.inactive", "")
const karmaDraft = useInstrumentDraft("roster.karma", 0)
const karmaEditing = useInstrumentDraft("roster.karmaEditing", false)
const review = ref<{ commandID: string; argumentsByID: Record<string, string>; title: string; summary: string }>()
invalidateInstrumentReview(() => { review.value = undefined })
watch([statusDraft, selectedRosterIndices, inactiveRosterIndex], () => { review.value = undefined }, { flush: "sync" })
const activeRoster = computed(() => props.data?.roster.filter((character) => character.active) ?? [])
const inactiveRoster = computed(() => props.data?.roster.filter((character) => !character.active) ?? [])
const fingerprint = computed(() => props.data?.fingerprint ?? "")
const karmaValid = computed(() => Number.isInteger(karmaDraft.value) && karmaDraft.value >= 0 && karmaDraft.value <= 99)

watch(activeRoster, (characters) => {
  const occupied = new Set(characters.map((character) => character.rosterIndex))
  const retained = selectedRosterIndices.value.filter((index) => occupied.has(index))
  if (retained.length !== selectedRosterIndices.value.length) selectedRosterIndices.value = retained
}, { immediate: true })
watch(inactiveRoster, (characters) => {
  if (!inactiveRosterIndex.value) {
    inactiveRosterIndex.value = characters[0] ? String(characters[0].rosterIndex) : ""
  }
}, { immediate: true })
watch(() => props.data?.karma, (karma) => {
  if (karma === undefined) return
  if (!karmaEditing.value) karmaDraft.value = karma
}, { immediate: true })
watch(fingerprint, (value, previous) => {
  if (review.value && previous && value !== previous) review.value = undefined
})

function pending(commandID: string): boolean {
  return props.pendingCommands.includes(commandID)
}

function diagnosticLabel(character: PartyRosterCharacterData): string {
  return `Roster index ${character.rosterIndex}`
}

function toggleStatusTarget(rosterIndex: number): void {
  selectedRosterIndices.value = selectedRosterIndices.value.includes(rosterIndex)
    ? selectedRosterIndices.value.filter((index) => index !== rosterIndex)
    : [...selectedRosterIndices.value, rosterIndex]
}

function reviewedArguments(extra: Record<string, string> = {}): Record<string, string> {
  return { expectedFingerprint: fingerprint.value, ...extra }
}

function setStatus(): void {
  if (selectedRosterIndices.value.length === 0) return
  const names = activeRoster.value.filter((character) => selectedRosterIndices.value.includes(character.rosterIndex)).map((character) => character.name)
  stageMutation("party.status.set", reviewedArguments({ rosterIndices: selectedRosterIndices.value.join(","), status: statusDraft.value }), "Review active-party condition", `${names.join(", ")} -> ${statusDraft.value}`)
}

function selectAllOccupied(): void {
  selectedRosterIndices.value = activeRoster.value.map((character) => character.rosterIndex)
}

function stageMutation(commandID: string, argumentsByID: Record<string, string>, title: string, summary: string): void {
  review.value = { commandID, argumentsByID, title, summary }
}

function confirmReview(): void {
  if (!review.value) return
  emit("mutate", review.value.commandID, review.value.argumentsByID)
  review.value = undefined
}
</script>

<template>
  <section class="party-roster" data-party-roster>
    <p v-if="!data" class="control-note">Loading the authoritative active and inactive roster...</p>
    <template v-else>
      <p class="roster-fingerprint">Reviewed roster <code>{{ fingerprint.slice(0, 12) }}</code></p>
      <div class="party-roster-layout">
        <section class="editor-group roster-active">
          <header><h4>Active positions</h4><small>Select any occupied Positions 1-6</small></header>
           <div class="roster-position-list" role="group" aria-label="Status target positions">
            <button v-for="(character, index) in activeRoster" :key="character.rosterIndex" type="button" :data-roster-position="index + 1" :data-roster-index="character.rosterIndex" :class="{ selected: selectedRosterIndices.includes(character.rosterIndex) }" :aria-pressed="selectedRosterIndices.includes(character.rosterIndex)" @click="toggleStatusTarget(character.rosterIndex)">
              <span class="roster-position">Position {{ index + 1 }}</span><strong class="roster-name">{{ character.name }}</strong><small class="roster-metadata">{{ character.class }} · {{ character.status }} · {{ diagnosticLabel(character) }}</small>
            </button>
           </div>
		  <div class="roster-selection-actions"><button type="button" data-status-target="all" @click="selectAllOccupied">SELECT ALL OCCUPIED</button><button type="button" data-status-target="none" :disabled="selectedRosterIndices.length === 0" @click="selectedRosterIndices = []">CLEAR SELECTION</button></div>
          <label for="party-reviewed-status">Explicit status</label>
          <select id="party-reviewed-status" v-model="statusDraft"><option value="good">Good</option><option value="poisoned">Poisoned</option><option value="sleep">Sleep</option><option value="charmed">Charmed</option><option value="dead">Dead</option></select>
          <button type="button" data-party-batch="status" :disabled="selectedRosterIndices.length === 0 || pending('party.status.set')" @click="setStatus">{{ pending('party.status.set') ? 'SETTING...' : `SET STATUS FOR ${selectedRosterIndices.length} SELECTED` }}</button>
        </section>

        <section class="editor-group roster-inactive">
          <header><h4>Inactive roster</h4><small>{{ inactiveRoster.length }} available</small></header>
          <ul><li v-for="character in inactiveRoster" :key="character.rosterIndex"><strong>{{ character.name }}</strong><span>{{ character.class }} · {{ character.status }}</span><small>{{ diagnosticLabel(character) }}</small></li></ul>
          <label for="party-inactive-roster">Member to add at next open position</label>
          <select id="party-inactive-roster" v-model="inactiveRosterIndex"><option v-for="character in inactiveRoster" :key="character.rosterIndex" :value="String(character.rosterIndex)">{{ character.name }} · {{ character.class }} (roster index {{ character.rosterIndex }})</option></select>
          <button type="button" data-party-batch="add" :disabled="!inactiveRosterIndex || activeRoster.length >= 6 || pending('party.member.add')" @click="stageMutation('party.member.add', reviewedArguments({ rosterIndex: inactiveRosterIndex }), 'Review roster addition', `${inactiveRoster.find((character) => String(character.rosterIndex) === inactiveRosterIndex)?.name || 'Selected member'} -> Position ${activeRoster.length + 1}`)">{{ pending('party.member.add') ? 'ADDING...' : 'REVIEW ADD TO NEXT OPEN POSITION' }}</button>
        </section>

        <section class="editor-group roster-wide-actions">
          <header><h4>Party-wide setup</h4><small>Reviewed operations</small></header>
          <div class="karma-control"><label for="party-karma-value">KARMA <small>Current {{ data.karma }}</small></label><input id="party-karma-value" v-model.number="karmaDraft" type="number" min="0" max="99" @input="karmaEditing = true" /><button type="button" :disabled="!karmaValid || karmaDraft === data.karma || karmaPending" @click="emit('mutate', 'party.karma.set', { value: String(karmaDraft) }); karmaEditing = false">{{ karmaPending ? 'SETTING...' : 'SET EXACT KARMA' }}</button></div>
          <p class="control-note">Sets exact party-wide karma without applying a moral event or advancing time.</p>
          <div class="roster-batch-actions">
            <button type="button" data-party-batch="maximize" :disabled="pending('party.maximize-levels')" @click="stageMutation('party.maximize-levels', reviewedArguments(), 'Review party progression', `Maximize ${activeRoster.length} active members to level 8`)" >REVIEW MAXIMIZE ACTIVE LEVELS</button>
            <button type="button" data-party-batch="fill" :disabled="inactiveRoster.length === 0 || activeRoster.length >= 6 || pending('party.fill')" @click="stageMutation('party.fill', reviewedArguments(), 'Review party composition', `Fill ${6 - activeRoster.length} open positions in roster order`)" >REVIEW FILL OPEN POSITIONS</button>
          </div>
        </section>

		<section v-if="review" class="editor-group roster-operation-review" data-roster-operation-review>
			<header><h4>{{ review.title }}</h4><small>Fingerprint {{ fingerprint.slice(0, 12) }}</small></header>
			<p>{{ review.summary }}</p>
			<div><button type="button" data-roster-review-cancel @click="review = undefined">BACK</button><button type="button" data-roster-review-confirm :disabled="pending(review.commandID)" @click="confirmReview">{{ pending(review.commandID) ? 'APPLYING...' : 'CONFIRM OPERATION' }}</button></div>
		</section>
      </div>
    </template>
  </section>
</template>
