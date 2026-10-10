<script setup lang="ts">
import { computed, shallowRef, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type {
  MerchantTesterCatalogData,
  MerchantTesterContextData,
  MerchantTesterMerchantData,
} from "./approved_tool_views"
import type { CommandDefinition } from "./protocol"

const previewCommandID = "merchant.tester.preview.open"
const liveCommandID = "merchant.tester.live.open"

const props = defineProps<{
  catalog?: MerchantTesterCatalogData
  catalogSceneGeneration?: number
  commands: CommandDefinition[]
  sceneGeneration: number
  pendingPreview: boolean
  pendingLive: boolean
}>()

const emit = defineEmits<{
  preview: [argumentsByID: Record<string, string>]
  live: [argumentsByID: Record<string, string>]
}>()

interface TownChoice {
  id: number
  name: string
}

interface MerchantGroup {
  id: string
  label: string
  merchants: MerchantTesterMerchantData[]
}

type MerchantActionMode = "preview" | "live"

interface LiveReview {
  actionMode: MerchantActionMode
  available: boolean
  merchantId: string
  townId: number
  townName: string
  context: MerchantTesterContextData
  sourceFingerprint: string
  effectiveMode: string
  sceneGeneration: number
}

const selectedTownId = useInstrumentDraft<number | undefined>("merchant.town", undefined)
const selectedMerchantId = useInstrumentDraft("merchant.selected", "")
const liveReview = shallowRef<LiveReview>()
invalidateInstrumentReview(() => { liveReview.value = undefined })

const previewCommand = computed(() => props.commands.find((command) => command.id === previewCommandID))
const liveCommand = computed(() => props.commands.find((command) => command.id === liveCommandID))
const towns = computed<TownChoice[]>(() => {
  const choices = new Map<number, string>()
  for (const merchant of props.catalog?.merchants ?? []) choices.set(merchant.townId, merchant.townName)
  return [...choices].map(([id, name]) => ({ id, name }))
    .sort((left, right) => left.name.localeCompare(right.name) || left.id - right.id)
})
const townMerchants = computed(() => (props.catalog?.merchants ?? [])
  .filter((merchant) => merchant.townId === selectedTownId.value)
  .sort(compareMerchants))
const merchantGroups = computed<MerchantGroup[]>(() => {
  const groups = new Map<string, MerchantGroup>()
  for (const merchant of townMerchants.value) {
    const group = groups.get(merchant.familyId) ?? { id: merchant.familyId, label: merchant.familyLabel, merchants: [] }
    group.merchants.push(merchant)
    groups.set(merchant.familyId, group)
  }
  return [...groups.values()].sort((left, right) => left.label.localeCompare(right.label) || left.id.localeCompare(right.id))
})
const selectedMerchant = computed(() => townMerchants.value.find((merchant) => merchant.merchantId === selectedMerchantId.value))
const catalogIsCurrent = computed(() => props.catalogSceneGeneration === props.sceneGeneration)
const previewUnavailableReason = computed(() => {
  if (!previewCommand.value) return "Preview command is unavailable in Discovery."
  if (!selectedMerchant.value) return "Select a merchant."
  if (!catalogIsCurrent.value) return "Refresh Merchant Tester for the current scene."
  return selectedMerchant.value.previewUnavailableReason || selectedMerchant.value.unavailableReason || ""
})
const liveUnavailableReason = computed(() => {
  if (!liveCommand.value) return "Live command is unavailable in Discovery."
  if (!selectedMerchant.value) return "Select a merchant."
  if (!catalogIsCurrent.value) return "Refresh Merchant Tester for the current scene."
  return selectedMerchant.value.liveUnavailableReason || selectedMerchant.value.unavailableReason || ""
})
const previewActionAvailable = computed(() => Boolean(selectedMerchant.value?.previewAvailable && !previewUnavailableReason.value))
const liveActionAvailable = computed(() => Boolean(selectedMerchant.value?.liveAvailable && !liveUnavailableReason.value))
const reviewIsCurrent = computed(() => Boolean(liveReview.value
  && props.catalog
  && selectedMerchant.value
  && liveReview.value.actionMode === "live"
  && liveReview.value.available === liveActionAvailable.value
  && liveReview.value.merchantId === selectedMerchant.value.merchantId
  && liveReview.value.townId === selectedMerchant.value.townId
  && liveReview.value.townName === selectedMerchant.value.townName
  && liveReview.value.sourceFingerprint === props.catalog.sourceFingerprint
  && liveReview.value.effectiveMode === selectedMerchant.value.liveExecutionMode
  && liveReview.value.sceneGeneration === props.sceneGeneration
  && contextIdentity(liveReview.value.context) === contextIdentity(props.catalog.currentContext)))

watch(reviewIsCurrent, (current) => {
  if (!current && liveReview.value) liveReview.value = undefined
})

watch(() => props.catalog, (catalog, previous) => {
  if (!catalog) return
  if (previous && catalog && (previous.sourceFingerprint !== catalog.sourceFingerprint
    || contextIdentity(previous.currentContext) !== contextIdentity(catalog.currentContext))) {
    liveReview.value = undefined
  }

  const availableTowns = towns.value
  const preferredTown = selectedTownId.value
    ?? availableTowns.find((town) => town.id === catalog.currentContext.locationId)?.id ?? availableTowns[0]?.id
  if (selectedTownId.value !== preferredTown) {
    selectedTownId.value = preferredTown
    liveReview.value = undefined
  }

  const availableMerchants = (catalog?.merchants ?? []).filter((merchant) => merchant.townId === selectedTownId.value).sort(compareMerchants)
  if (!selectedMerchantId.value) {
    selectedMerchantId.value = availableMerchants[0]?.merchantId ?? ""
    liveReview.value = undefined
  }
}, { immediate: true })

function compareMerchants(left: MerchantTesterMerchantData, right: MerchantTesterMerchantData): number {
  return left.familyLabel.localeCompare(right.familyLabel)
    || left.shopName.localeCompare(right.shopName)
    || left.ownerName.localeCompare(right.ownerName)
    || left.merchantId.localeCompare(right.merchantId)
}

function contextIdentity(context: MerchantTesterContextData): string {
  return `${context.locationId}:${context.locationName}:${context.floor}:${context.x}:${context.y}`
}

function selectTown(event: Event): void {
  const townId = Number((event.target as HTMLSelectElement).value)
  if (townId === selectedTownId.value) return
  selectedTownId.value = townId
  selectedMerchantId.value = townMerchants.value[0]?.merchantId ?? ""
  liveReview.value = undefined
}

function selectMerchant(merchantId: string): void {
  if (merchantId === selectedMerchantId.value) return
  selectedMerchantId.value = merchantId
  liveReview.value = undefined
}

function openPreview(): void {
  const merchant = selectedMerchant.value
  if (!merchant || !previewActionAvailable.value || props.pendingPreview) return
  liveReview.value = undefined
  emit("preview", { merchantId: merchant.merchantId })
}

function reviewLive(): void {
  if (!props.catalog || !selectedMerchant.value || !liveActionAvailable.value || props.pendingLive) return
  liveReview.value = {
    actionMode: "live",
    available: liveActionAvailable.value,
    merchantId: selectedMerchant.value.merchantId,
    townId: selectedMerchant.value.townId,
    townName: selectedMerchant.value.townName,
    context: { ...props.catalog.currentContext },
    sourceFingerprint: props.catalog.sourceFingerprint,
    effectiveMode: selectedMerchant.value.liveExecutionMode,
    sceneGeneration: props.sceneGeneration,
  }
}

function openLive(): void {
  const review = liveReview.value
  if (!review || !reviewIsCurrent.value || props.pendingLive) return
  emit("live", {
    merchantId: review.merchantId,
    expectedLocationId: String(review.context.locationId),
    expectedFloor: String(review.context.floor),
    expectedX: String(review.context.x),
    expectedY: String(review.context.y),
    expectedSourceFingerprint: review.sourceFingerprint,
  })
}

function compactFingerprint(value: string): string {
  return value.length <= 20 ? value : `${value.slice(0, 10)}...${value.slice(-6)}`
}

function modeLabel(value: string): string {
  return value.replaceAll("_", " ").toLocaleUpperCase()
}
</script>

<template>
  <section class="approved-view merchant-tester" aria-label="Merchant Tester">
    <header class="approved-view-header merchant-tester-header">
      <div><span>CANONICAL SHOP DIRECTORY / PRODUCTION DIALOG LAUNCHER</span><h3>Merchant Tester</h3></div>
      <small>{{ catalog ? `${catalog.merchants.length} MERCHANTS · ${towns.length} TOWNS` : 'WAITING' }}</small>
    </header>

    <template v-if="catalog">
      <section class="merchant-context-strip" aria-label="Current merchant tester context">
        <div><span>Current context</span><strong>{{ catalog.currentContext.locationName }}</strong><small>LOCATION {{ catalog.currentContext.locationId }}</small></div>
        <div><span>Floor / position</span><strong>{{ catalog.currentContext.floor }} / {{ catalog.currentContext.x }}, {{ catalog.currentContext.y }}</strong><small>REVIEW SOURCE COORDINATES</small></div>
        <div><span>Source identity</span><code :title="catalog.sourceFingerprint">{{ compactFingerprint(catalog.sourceFingerprint) }}</code><small>AUTHORITATIVE FINGERPRINT</small></div>
      </section>

      <div class="merchant-tester-layout">
        <nav class="approved-panel merchant-directory" aria-label="Canonical merchants">
          <header><strong>Merchant directory</strong><small>{{ townMerchants.length }} IN TOWN</small></header>
          <label class="merchant-town-filter">Target town
            <select :value="selectedTownId" data-merchant-town @change="selectTown">
              <option v-for="town in towns" :key="town.id" :value="town.id">{{ town.name }} · LOCATION {{ town.id }}</option>
            </select>
          </label>
          <div class="merchant-family-groups">
            <section v-for="group in merchantGroups" :key="group.id">
              <header><strong>{{ group.label }}</strong><small>{{ group.id }} · {{ group.merchants.length }}</small></header>
              <button
                v-for="merchant in group.merchants"
                :key="merchant.merchantId"
                type="button"
                :class="{ selected: merchant.merchantId === selectedMerchantId, unavailable: !merchant.implemented }"
                :aria-pressed="merchant.merchantId === selectedMerchantId"
                :data-merchant-id="merchant.merchantId"
                @click="selectMerchant(merchant.merchantId)"
              >
                <span><strong>{{ merchant.shopName }}</strong><small>{{ merchant.ownerName }}</small></span>
                <b>{{ merchant.implemented ? 'READY' : 'UNIMPLEMENTED' }}</b>
                <em v-if="merchant.unavailableReason">{{ merchant.unavailableReason }}</em>
              </button>
            </section>
          </div>
        </nav>

        <section v-if="selectedMerchant" class="merchant-console" aria-live="polite">
          <section class="approved-panel merchant-identity">
            <header><strong>{{ selectedMerchant.shopName }}</strong><small>{{ selectedMerchant.familyLabel }}</small></header>
            <div class="merchant-owner"><span>Shop owner</span><strong>{{ selectedMerchant.ownerName }}</strong><small>{{ selectedMerchant.townName }}</small></div>
            <dl>
              <div><dt>Canonical ID</dt><dd><code>{{ selectedMerchant.merchantId }}</code></dd></div>
              <div><dt>Family</dt><dd>{{ selectedMerchant.familyLabel }} <small>{{ selectedMerchant.familyId }}</small></dd></div>
              <div><dt>Dialog code</dt><dd>{{ selectedMerchant.dialogCode }}</dd></div>
              <div><dt>Target town</dt><dd>{{ selectedMerchant.townName }} <small>LOCATION {{ selectedMerchant.townId }}</small></dd></div>
            </dl>
            <p v-if="selectedMerchant.unavailableReason" class="merchant-family-unavailable">{{ selectedMerchant.unavailableReason }}</p>
          </section>

          <section class="merchant-operation-lanes">
            <article class="approved-panel merchant-operation preview-operation">
              <header><strong>{{ previewCommand?.name ?? 'Merchant Preview' }}</strong><small>SAFE DIRECT ACTION</small></header>
              <p>{{ previewCommand?.description ?? 'The Preview command is not present in Discovery.' }}</p>
              <div class="merchant-availability" :class="{ available: previewActionAvailable }">
                <span>SERVER AVAILABILITY</span><strong>{{ previewActionAvailable ? 'AVAILABLE' : 'UNAVAILABLE' }}</strong>
                <small v-if="previewUnavailableReason">{{ previewUnavailableReason }}</small>
              </div>
              <button type="button" data-merchant-preview :disabled="!previewActionAvailable || pendingPreview" @click="openPreview">
                {{ pendingPreview ? 'OPENING PREVIEW...' : 'OPEN PREVIEW' }}
              </button>
            </article>

            <article class="approved-panel merchant-operation live-operation">
              <header><strong>{{ liveCommand?.name ?? 'Live Merchant Session' }}</strong><small>REVIEW REQUIRED</small></header>
              <p>{{ liveCommand?.description ?? 'The Live command is not present in Discovery.' }}</p>
              <div class="merchant-availability" :class="{ available: liveActionAvailable }">
                <span>SERVER AVAILABILITY</span><strong>{{ liveActionAvailable ? 'AVAILABLE' : 'UNAVAILABLE' }}</strong>
                <small v-if="liveUnavailableReason">{{ liveUnavailableReason }}</small>
              </div>
              <div class="merchant-live-mode"><span>Advertised effective mode</span><strong>{{ modeLabel(selectedMerchant.liveExecutionMode) }}</strong></div>
              <button type="button" data-merchant-live-review :disabled="!liveActionAvailable || pendingLive" @click="reviewLive">
                {{ pendingLive ? 'OPENING LIVE CONTEXT...' : 'REVIEW LIVE CONTEXT' }}
              </button>
            </article>
          </section>

          <section v-if="liveReview && reviewIsCurrent" class="merchant-live-review" data-merchant-live-review-panel>
            <header><strong>Live context review</strong><small>STAGE 2 / 2 · {{ modeLabel(liveReview.effectiveMode) }}</small></header>
            <dl>
              <div><dt>Merchant</dt><dd>{{ selectedMerchant.shopName }} / {{ selectedMerchant.ownerName }}</dd></div>
              <div><dt>Target town</dt><dd>{{ liveReview.townName }} · LOCATION {{ liveReview.townId }}</dd></div>
              <div><dt>Reviewed location</dt><dd>{{ liveReview.context.locationName }} · LOCATION {{ liveReview.context.locationId }}</dd></div>
              <div><dt>Floor / X / Y</dt><dd>{{ liveReview.context.floor }} / {{ liveReview.context.x }} / {{ liveReview.context.y }}</dd></div>
              <div><dt>Source identity</dt><dd><code :title="liveReview.sourceFingerprint">{{ compactFingerprint(liveReview.sourceFingerprint) }}</code></dd></div>
              <div><dt>Effective mode</dt><dd>{{ modeLabel(liveReview.effectiveMode) }}</dd></div>
            </dl>
            <footer><button type="button" :disabled="pendingLive" @click="liveReview = undefined">BACK</button><button class="danger-button" type="button" data-merchant-live-open :disabled="pendingLive" @click="openLive">{{ pendingLive ? 'OPENING...' : 'OPEN LIVE CONTEXT' }}</button></footer>
          </section>
        </section>
      </div>
    </template>
    <p v-else class="empty-state">WAITING FOR MERCHANT TESTER CATALOG</p>
  </section>
</template>
