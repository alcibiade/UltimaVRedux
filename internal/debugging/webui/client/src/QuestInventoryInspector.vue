<script setup lang="ts">
import { computed, ref } from "vue"
import { invalidateInstrumentReview } from "./instrument_session"

import type { QuestItemData, QuestItemStatusData } from "./approved_tool_views"

const props = defineProps<{
  data?: QuestItemStatusData
  updatedAt?: Date
  pending: boolean
}>()

const emit = defineEmits<{
  ensure: [item: string]
  remove: [item: string]
	useShard: [argumentsByID: Record<string, string>]
}>()

const categoryOrder = ["quest", "artifact", "shard", "tool", "key", "other"]
const removeConfirmation = ref("")
const shardConfirmation = ref("")
invalidateInstrumentReview(() => { removeConfirmation.value = ""; shardConfirmation.value = "" })
const groups = computed(() => {
  const byCategory = new Map<string, QuestItemData[]>()
  for (const item of props.data?.items ?? []) {
    const category = item.category || "other"
    byCategory.set(category, [...(byCategory.get(category) ?? []), item])
  }
  return [...byCategory].sort(([a], [b]) => {
    const aIndex = categoryOrder.indexOf(a.toLocaleLowerCase())
    const bIndex = categoryOrder.indexOf(b.toLocaleLowerCase())
    return (aIndex < 0 ? categoryOrder.length : aIndex) - (bIndex < 0 ? categoryOrder.length : bIndex) || a.localeCompare(b)
  }).map(([category, items]) => ({ category, items }))
})

function requestRemove(item: QuestItemData): void {
  if (removeConfirmation.value !== item.key) {
    removeConfirmation.value = item.key
    return
  }
  emit("remove", item.key)
  removeConfirmation.value = ""
}

function requestShardUse(item: QuestItemData): void {
	const review = item.shardUseReview
	if (!review || review.quantity === 0) return
	if (shardConfirmation.value !== item.key) {
		shardConfirmation.value = item.key
		return
	}
	emit("useShard", {
		shard: review.shard,
		expectedQuantity: String(review.quantity),
		expectedLocationId: String(review.locationId),
		expectedX: String(review.x),
		expectedY: String(review.y),
		expectedFloor: String(review.floor),
		expectedFlame: review.flame,
		expectedShadowlordSummoned: String(review.shadowlordSummoned),
		expectedShadowlord: review.shadowlord,
		expectedShadowlordAlive: String(review.shadowlordAlive),
	})
	shardConfirmation.value = ""
}
</script>

<template>
  <section class="approved-view quest-inventory" aria-label="Quest inventory inspector">
    <header class="approved-view-header">
      <div><span>FIXED AUTHORITATIVE LEDGER</span><h3>Quest Inventory</h3></div>
      <small>{{ updatedAt ? `SYNC ${updatedAt.toLocaleTimeString([], { hour12: false })}` : 'WAITING' }}</small>
    </header>

    <template v-if="data">
      <div class="quest-ledger">
        <section v-for="group in groups" :key="group.category" class="approved-panel">
          <header><strong>{{ group.category }}</strong><small>{{ group.items.length }} ENTRIES</small></header>
          <div class="quest-ledger-row quest-ledger-heading" aria-hidden="true"><span>State</span><span>Item</span><span>Canonical placement</span><span>Operation</span></div>
          <article v-for="item in group.items" :key="item.key" class="quest-ledger-row">
            <span class="inventory-count" :class="item.quantity > 0 ? 'success' : 'summary'">×{{ item.quantity }}</span>
            <div><strong>{{ item.name }}</strong><small>{{ item.key }}</small></div>
            <span class="placement">
              <template v-if="item.canonicalPlacement">{{ item.canonicalPlacement.locationName }} · {{ item.canonicalPlacement.x }},{{ item.canonicalPlacement.y }} · F{{ item.canonicalPlacement.floor }}</template>
              <template v-else>NO CANONICAL PLACEMENT</template>
            </span>
            <div class="row-actions">
              <button type="button" :disabled="pending || item.quantity > 0" @click="emit('ensure', item.key)">{{ item.quantity > 0 ? 'OWNED' : 'GRANT' }}</button>
			  <button v-if="item.shardUseReview" type="button" :class="{ armed: shardConfirmation === item.key }" :disabled="pending || item.quantity === 0" @click="requestShardUse(item)">{{ shardConfirmation === item.key ? `CONFIRM ${item.shardUseReview.expectedOutcome.toUpperCase()}` : 'USE' }}</button>
              <button type="button" class="danger-button" :class="{ armed: removeConfirmation === item.key }" :disabled="pending || item.quantity === 0" @click="requestRemove(item)">{{ removeConfirmation === item.key ? 'CONFIRM' : 'REMOVE' }}</button>
            </div>
          </article>
        </section>
      </div>

      <aside class="unsupported-tools" aria-label="Unavailable quest workflows">
        <div><strong>GRANT MISSING</strong><p>No typed bulk grant operation is exposed by the backend.</p><button type="button" disabled>GRANT MISSING</button></div>
      </aside>
    </template>
    <p v-else class="empty-state">WAITING FOR QUEST INVENTORY</p>
  </section>
</template>
