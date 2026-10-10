<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import MoonPhaseGlyph from "./MoonPhaseGlyph.vue"
import type { ScenarioComposerDraft } from "./scenarios"
import { saveSlug, type SaveSnapshotDetail, type SaveSnapshotListData, type SaveSnapshotSummary, type SaveSource } from "./saves"

const props = defineProps<{
  data?: SaveSnapshotListData
  detail?: SaveSnapshotDetail
  pendingList: boolean
  pendingDetail: boolean
  pendingMutation: boolean
  currentScene?: { locationName: string; floor: number; date: string; time: string; partyMembers: number }
  activeScenario?: ScenarioComposerDraft
  activeScenarioDirty: boolean
  loadReviewTarget?: { reference: string; revision: string; requestID: string }
}>()

const emit = defineEmits<{
  refresh: []
  inspect: [reference: string]
  capture: [value: { source: "project" | "personal"; id: string; name: string; description: string }]
  load: [value: { reference: string; expectedRevision: string }]
  delete: [value: { reference: string; expectedRevision: string }]
  useBaseline: [snapshot: SaveSnapshotSummary]
  reviewTargetConsumed: []
}>()

const captureSource = useInstrumentDraft<"project" | "personal">("saves.source", "personal")
const captureName = useInstrumentDraft("saves.name", "")
const captureID = useInstrumentDraft("saves.id", "")
const captureDescription = useInstrumentDraft("saves.description", "")
const slugEdited = useInstrumentDraft("saves.slugEdited", false)
const review = ref<"load" | "delete" | "baseline">()
const reviewedSnapshot = ref<SaveSnapshotDetail>()
invalidateInstrumentReview(() => { review.value = undefined; reviewedSnapshot.value = undefined })
watch(review, (operation) => {
  reviewedSnapshot.value = operation && props.detail ? JSON.parse(JSON.stringify(props.detail)) as SaveSnapshotDetail : undefined
}, { flush: "sync" })
const confirmation = ref("")

watch(captureName, (name) => {
  if (!slugEdited.value) captureID.value = saveSlug(name)
})
watch(() => `${props.detail?.summary.reference}:${props.detail?.summary.revision}`, () => {
  review.value = undefined
  confirmation.value = ""
})
watch(() => `${props.activeScenario?.source}:${props.activeScenario?.definition.id}:${props.activeScenario?.savedReference}`, () => {
  if (review.value === "baseline") review.value = undefined
})

const groups = computed(() => (["project", "personal", "rollback"] as SaveSource[]).map((source) => ({
  source,
  snapshots: (props.data?.snapshots ?? []).filter((snapshot) => snapshot.source === source),
})))
const selected = computed(() => props.detail?.summary)
const detail = computed(() => reviewedSnapshot.value ?? props.detail)
watch([() => props.loadReviewTarget, () => props.detail], ([target, snapshot]) => {
  if (!target || !snapshot || target.reference !== snapshot.summary.reference || target.revision !== snapshot.summary.revision) return
  review.value = "load"
  emit("reviewTargetConsumed")
}, { immediate: true })
const captureValid = computed(() => Boolean(captureName.value.trim() && captureID.value && props.data?.captureEligibility.canCapture))
const deleteValid = computed(() => Boolean(selected.value && confirmation.value === selected.value.name && selected.value.eligibility.canDelete))
const namedSnapshot = computed(() => selected.value?.source === "project" || selected.value?.source === "personal")
const baselinePortable = computed(() => !props.activeScenario || props.activeScenario.source !== "project" || selected.value?.source === "project")

function editSlug(value: string): void {
  slugEdited.value = true
  captureID.value = saveSlug(value)
}

function submitCapture(): void {
  if (!captureValid.value) return
  emit("capture", { source: captureSource.value, id: captureID.value, name: captureName.value.trim(), description: captureDescription.value.trim() })
}

function confirmReview(): void {
  const snapshot = reviewedSnapshot.value?.summary
  if (!snapshot || props.pendingMutation || snapshot.reference !== selected.value?.reference || snapshot.revision !== selected.value?.revision) return
  if (review.value === "load" && snapshot.eligibility.canLoad) emit("load", { reference: snapshot.reference, expectedRevision: snapshot.revision })
  if (review.value === "delete" && deleteValid.value) emit("delete", { reference: snapshot.reference, expectedRevision: snapshot.revision })
  if (review.value === "baseline" && baselinePortable.value) emit("useBaseline", snapshot)
  review.value = undefined
  confirmation.value = ""
}
</script>

<template>
  <section class="save-library approved-view" aria-label="Saves workspace" data-saves-workspace>
    <header class="approved-view-header">
      <div><span>IMMUTABLE GAMEPLAY CHECKPOINTS</span><h3>Saves</h3></div>
      <button type="button" :disabled="pendingList" data-save-action="refresh" @click="emit('refresh')">{{ pendingList ? 'SYNCING...' : 'REFRESH LIBRARY' }}</button>
    </header>

    <div class="save-layout">
      <aside class="save-groups" aria-label="Save snapshot library">
        <section v-for="group in groups" :key="group.source" :data-save-source="group.source">
          <header><strong>{{ group.source === 'project' ? 'PROJECT / SHARED' : group.source === 'personal' ? 'PERSONAL / LOCAL' : 'ROLLBACK / RECOVERY' }}</strong><small>{{ group.snapshots.length }}</small></header>
          <button v-for="snapshot in group.snapshots" :key="snapshot.reference" type="button" :class="{ selected: selected?.reference === snapshot.reference }" :disabled="pendingDetail" :data-save-reference="snapshot.reference" @click="emit('inspect', snapshot.reference)">
            <strong>{{ snapshot.name }}</strong><span>{{ snapshot.description || 'No description' }}</span><small>{{ snapshot.id }} · {{ new Date(snapshot.createdAt).toLocaleString() }}</small>
          </button>
          <p v-if="group.snapshots.length === 0">NO {{ group.source.toUpperCase() }} CAPTURES</p>
        </section>
      </aside>

      <main class="save-detail">
        <template v-if="detail">
          <header class="save-detail-header"><div><span>{{ detail.summary.source.toUpperCase() }} SNAPSHOT</span><h4>{{ detail.summary.name }}</h4><p>{{ detail.summary.description || 'No description provided.' }}</p></div><code>{{ detail.summary.reference }}</code></header>
          <dl class="save-metadata">
            <div><dt>Revision</dt><dd><code>{{ detail.summary.revision }}</code></dd></div>
            <div><dt>Compatibility</dt><dd :class="detail.summary.compatibility.compatible ? 'success' : 'failure'">{{ detail.summary.compatibility.compatible ? 'COMPATIBLE' : 'INCOMPATIBLE' }}<small v-if="detail.summary.compatibility.reason">{{ detail.summary.compatibility.reason }}</small></dd></div>
            <div><dt>Location</dt><dd>{{ detail.locationName }} · #{{ detail.location }} · {{ detail.x }},{{ detail.y }} · Floor {{ detail.floor }}</dd></div>
            <div><dt>Party</dt><dd>{{ detail.partyMembers }} active members</dd></div>
            <div><dt>Date / Time</dt><dd>Y{{ detail.year }} M{{ detail.month }} D{{ detail.day }} · {{ String(detail.hour).padStart(2, '0') }}:{{ String(detail.minute).padStart(2, '0') }}</dd></div>
            <div><dt>Moon</dt><dd class="save-moon"><MoonPhaseGlyph :phase-index="detail.moonPhaseIndex" :label="detail.moonPhase" :size="24" /><span>{{ detail.moonPhase }}</span></dd></div>
          </dl>
          <p v-if="detail.summary.referencedBy?.length" class="save-reference-warning"><strong>REFERENCED BY</strong> {{ detail.summary.referencedBy.join(' · ') }}. Delete is blocked while these scenarios use this baseline.</p>
          <div class="save-actions">
            <button type="button" :disabled="pendingMutation || !detail.summary.eligibility.canLoad || !detail.summary.compatibility.compatible" data-save-action="review-load" @click="review = 'load'">REVIEW &amp; LOAD</button>
            <button v-if="namedSnapshot" type="button" :disabled="pendingMutation || !activeScenario || !baselinePortable" data-save-action="review-baseline" @click="review = 'baseline'">USE AS SCENARIO BASELINE</button>
            <button v-if="detail.summary.source !== 'rollback'" type="button" class="danger-button" :disabled="pendingMutation || !detail.summary.eligibility.canDelete" data-save-action="review-delete" @click="review = 'delete'; confirmation = ''">DELETE</button>
          </div>
          <p v-if="!baselinePortable" class="disabled-reason">PROJECT SCENARIOS MAY ONLY USE PROJECT SNAPSHOT BASELINES.</p>

          <section v-if="review === 'load'" class="save-review" data-save-review="load">
            <header><strong>SCENE-BOUND LOAD REVIEW</strong><small>AUTOMATIC ROLLBACK WILL BE CAPTURED FIRST</small></header>
            <div class="save-review-comparison"><article><span>CURRENT SCENE</span><strong>{{ currentScene?.locationName ?? 'Active gameplay scene' }}</strong><small>{{ currentScene ? `Floor ${currentScene.floor} · ${currentScene.date} ${currentScene.time} · ${currentScene.partyMembers} party` : 'Live summary unavailable' }}</small></article><article><span>CANDIDATE</span><strong>{{ detail.locationName }}</strong><small>Location #{{ detail.location }} · Floor {{ detail.floor }} · Y{{ detail.year }} M{{ detail.month }} D{{ detail.day }} {{ String(detail.hour).padStart(2, '0') }}:{{ String(detail.minute).padStart(2, '0') }} · {{ detail.partyMembers }} party</small></article></div>
            <footer><button type="button" @click="review = undefined">CANCEL</button><button type="button" data-save-action="confirm-load" @click="confirmReview">CAPTURE ROLLBACK &amp; LOAD</button></footer>
          </section>
          <section v-else-if="review === 'delete'" class="save-review danger" data-save-review="delete"><header><strong>REVISION-SAFE DELETE</strong><small>TYPE THE SNAPSHOT NAME</small></header><label>CONFIRM NAME<input v-model="confirmation" autocomplete="off" :placeholder="detail.summary.name" /></label><footer><button type="button" @click="review = undefined">CANCEL</button><button type="button" class="danger-button" :disabled="!deleteValid" data-save-action="confirm-delete" @click="confirmReview">DELETE REVIEWED REVISION</button></footer></section>
          <section v-else-if="review === 'baseline'" class="save-review" data-save-review="baseline"><header><strong>SCENARIO BASELINE REVIEW</strong><small>{{ activeScenarioDirty ? 'ACTIVE DRAFT HAS UNSAVED CHANGES' : 'ACTIVE DRAFT' }}</small></header><p>Set <strong>{{ activeScenario?.definition.name }}</strong> to start from <code>{{ detail.summary.reference }}</code>. This changes only the active draft and must still be saved.</p><footer><button type="button" @click="review = undefined">CANCEL</button><button type="button" data-save-action="confirm-baseline" @click="confirmReview">USE REVIEWED SNAPSHOT</button></footer></section>
        </template>
        <p v-else class="empty-state">SELECT A CAPTURE TO INSPECT REVISION, GAME STATE, AND AVAILABLE OPERATIONS</p>
      </main>

      <aside class="save-capture">
        <header><strong>CAPTURE ACTIVE SCENE</strong><small>NEW IMMUTABLE SNAPSHOT</small></header>
        <form @submit.prevent="submitCapture">
          <label>SOURCE<select v-model="captureSource"><option value="personal">Personal / Local</option><option value="project">Project / Shared</option></select></label>
          <label>NAME<input v-model="captureName" required placeholder="Before Dungeon Entry" /></label>
          <label>SLUG<input :value="captureID" required pattern="[a-z0-9](?:[a-z0-9-]*[a-z0-9])?" data-save-capture-slug @input="editSlug(($event.target as HTMLInputElement).value)" /></label>
          <label>DESCRIPTION<textarea v-model="captureDescription" placeholder="What this checkpoint is for." /></label>
          <p v-if="data && !data.captureEligibility.canCapture" class="disabled-reason" role="status">{{ data.captureEligibility.reason || 'The active scene is not eligible for capture.' }}</p>
          <button type="submit" :disabled="pendingMutation || !captureValid" data-save-action="capture">{{ pendingMutation ? 'WORKING...' : 'CAPTURE ACTIVE SCENE' }}</button>
        </form>
      </aside>
    </div>
  </section>
</template>
