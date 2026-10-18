<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { invalidateInstrumentReview, useInstrumentDraft } from "./instrument_session"

import type { ActiveCombatStatusData } from "./approved_tool_views"
import type { CommandDefinition } from "./protocol"
import PreferenceNumberControl from "./PreferenceNumberControl.vue"

interface CombatMutation { commandID: string; argumentsByID: Record<string, string> }

const props = defineProps<{ data?: ActiveCombatStatusData; updatedAt?: Date; pending: boolean; commands?: CommandDefinition[] }>()
const emit = defineEmits<{ mutate: [mutation: CombatMutation]; openEncounter: [] }>()
const selectedMonsterID = useInstrumentDraft("combat.monster", "")
const selectedEncounterID = useInstrumentDraft("combat.selectedEncounter", "")
const reviewedCondition = ref<{ monsterID: string; name: string; asleep: boolean; expectedAsleep: boolean; snapshot: string }>()
const reviewedExit = ref<{ encounterID: string; snapshot: string }>()
const spawnMonsterType = useInstrumentDraft("combat.spawnType", "32")
const spawnCountMode = useInstrumentDraft("combat.spawnMode", "automatic")
const spawnExactCount = useInstrumentDraft("combat.spawnCount", 1)
const spawnFriends = useInstrumentDraft("combat.spawnFriends", true)
const spawnSeed = useInstrumentDraft("combat.spawnSeed", "")
const reviewedSpawn = ref<{ encounterID: string; snapshot: string; monsterType: string; countMode: string; exactCount?: number; friends: boolean; seed?: string }>()
const damageAmount = useInstrumentDraft("combat.damage", 1)
const rngSeed = useInstrumentDraft("combat.seed", 42)
const reviewedSeed = ref<{ encounterID: string; snapshot: string; beforeSeed: number; afterSeed: number }>()
const reviewedMonsterMutation = ref<{ kind: "damage" | "kill"; encounterID: string; snapshot: string; monsterID: string; name: string; expectedHP: number; damage?: number }>()
const reviewedBatch = ref<{ kind: "kill-all" | "clear" | "clear-room"; encounterID: string; snapshot: string; activeCount: number; totalCount: number }>()
function disarm(): void {
  reviewedCondition.value = undefined
  reviewedExit.value = undefined
  reviewedSpawn.value = undefined
  reviewedSeed.value = undefined
  reviewedMonsterMutation.value = undefined
  reviewedBatch.value = undefined
}
invalidateInstrumentReview(disarm)
watch([selectedMonsterID, spawnMonsterType, spawnCountMode, spawnExactCount, spawnFriends, spawnSeed, damageAmount, rngSeed], disarm, { flush: "sync" })

const selectedMonster = computed(() => props.data?.monsters.find((monster) => monster.id === selectedMonsterID.value))
const mutationUnavailable = computed(() => !props.data?.active || props.data.workflowActive
  || props.data.map?.pendingAdvanceTurn || props.data.map?.activeMonsterIndex !== -1)

watch(() => props.data?.snapshotFingerprint, (fingerprint) => {
  if (!props.data) return
  if (reviewedCondition.value && reviewedCondition.value.snapshot !== fingerprint) reviewedCondition.value = undefined
  if (reviewedExit.value && reviewedExit.value.snapshot !== fingerprint) reviewedExit.value = undefined
  if (reviewedSpawn.value && reviewedSpawn.value.snapshot !== fingerprint) reviewedSpawn.value = undefined
	  if (reviewedSeed.value && reviewedSeed.value.snapshot !== fingerprint) reviewedSeed.value = undefined
	  if (reviewedMonsterMutation.value && reviewedMonsterMutation.value.snapshot !== fingerprint) reviewedMonsterMutation.value = undefined
	  if (reviewedBatch.value && reviewedBatch.value.snapshot !== fingerprint) reviewedBatch.value = undefined
  if (props.data.encounterId && selectedEncounterID.value !== props.data.encounterId) {
    selectedMonsterID.value = selectedEncounterID.value ? "" : props.data.monsters.find((monster) => !monster.dead && monster.active)?.id ?? ""
    selectedEncounterID.value = props.data.encounterId
    disarm()
  }
}, { immediate: true })


function stageSleep(asleep: boolean): void {
  const monster = selectedMonster.value
  if (!props.data?.encounterId || !monster || monster.dead || monster.asleep === asleep) return
  reviewedCondition.value = {
    monsterID: monster.id,
    name: monster.name,
    asleep,
    expectedAsleep: monster.asleep,
    snapshot: props.data.snapshotFingerprint,
  }
}

function stageExit(): void {
  if (!props.data?.encounterId || mutationUnavailable.value) return
  reviewedExit.value = { encounterID: props.data.encounterId, snapshot: props.data.snapshotFingerprint }
}

function applyExit(): void {
  const review = reviewedExit.value
  if (!review) return
  emit("mutate", { commandID: "combat.exit-debug", argumentsByID: { encounterId: review.encounterID, expectedSnapshot: review.snapshot } })
  reviewedExit.value = undefined
}

function stageSpawn(): void {
  if (!props.data?.encounterId || mutationUnavailable.value || !spawnMonsterType.value) return
  reviewedSpawn.value = {
    encounterID: props.data.encounterId,
    snapshot: props.data.snapshotFingerprint,
    monsterType: spawnMonsterType.value,
    countMode: spawnCountMode.value,
    ...(spawnCountMode.value === "exact" ? { exactCount: spawnExactCount.value } : {}),
    friends: spawnFriends.value,
    ...(spawnSeed.value.trim() ? { seed: spawnSeed.value.trim() } : {}),
  }
}

function applySpawn(): void {
  const review = reviewedSpawn.value
  if (!review) return
  emit("mutate", {
    commandID: "combat.monster.spawn",
    argumentsByID: {
      encounterId: review.encounterID,
      expectedSnapshot: review.snapshot,
      monsterType: review.monsterType,
      countMode: review.countMode,
      friends: String(review.friends),
      ...(review.exactCount !== undefined ? { exactCount: String(review.exactCount) } : {}),
      ...(review.seed !== undefined ? { seed: review.seed } : {}),
    },
  })
  reviewedSpawn.value = undefined
}

function applyCondition(): void {
  const review = reviewedCondition.value
  if (!review || !props.data?.encounterId) return
  emit("mutate", {
    commandID: "combat.monster.sleep.set",
    argumentsByID: {
      encounterId: props.data.encounterId,
      monsterId: review.monsterID,
      asleep: String(review.asleep),
      expectedAsleep: String(review.expectedAsleep),
      expectedSnapshot: review.snapshot,
    },
  })
  reviewedCondition.value = undefined
}

function stageSeed(): void {
	if (!props.data?.encounterId || mutationUnavailable.value || rngSeed.value < 1 || rngSeed.value > 999999) return
	reviewedSeed.value = {
		encounterID: props.data.encounterId, snapshot: props.data.snapshotFingerprint,
		beforeSeed: props.data.rngSeed, afterSeed: rngSeed.value,
	}
}

function applySeed(): void {
	const review = reviewedSeed.value
	if (!review) return
	emit("mutate", { commandID: "rng.seed.set", argumentsByID: {
		encounterId: review.encounterID, expectedSnapshot: review.snapshot, seed: String(review.afterSeed),
	} })
	reviewedSeed.value = undefined
}

function stageMonsterMutation(kind: "damage" | "kill"): void {
	const monster = selectedMonster.value
	if (!props.data?.encounterId || !monster || monster.dead || mutationUnavailable.value) return
	if (kind === "damage" && (damageAmount.value < 1 || damageAmount.value > 999)) return
	reviewedMonsterMutation.value = {
		kind, encounterID: props.data.encounterId, snapshot: props.data.snapshotFingerprint,
		monsterID: monster.id, name: monster.name, expectedHP: monster.currentHp,
		...(kind === "damage" ? { damage: damageAmount.value } : {}),
	}
}

function applyMonsterMutation(): void {
	const review = reviewedMonsterMutation.value
	if (!review) return
	emit("mutate", {
		commandID: review.kind === "damage" ? "combat.monster.damage-raw" : "combat.monster.kill",
		argumentsByID: {
			encounterId: review.encounterID, monsterId: review.monsterID,
			expectedCurrentHp: String(review.expectedHP), expectedSnapshot: review.snapshot,
			...(review.damage !== undefined ? { damage: String(review.damage) } : {}),
		},
	})
	reviewedMonsterMutation.value = undefined
}

function stageBatch(kind: "kill-all" | "clear" | "clear-room"): void {
	if (!props.data?.encounterId || mutationUnavailable.value || props.data.totalMonsterCount === 0) return
	if (kind === "clear-room" && (!props.data.fixedDungeonRoom || props.data.map?.victory || props.data.activeHostileMonsterCount === 0)) return
	reviewedBatch.value = {
		kind, encounterID: props.data.encounterId, snapshot: props.data.snapshotFingerprint,
		activeCount: props.data.activeMonsterCount, totalCount: props.data.totalMonsterCount,
	}
}

function applyBatch(): void {
	const review = reviewedBatch.value
	if (!review) return
	emit("mutate", {
		commandID: review.kind === "kill-all" ? "combat.monster.kill-all"
			: review.kind === "clear-room" ? "combat.dungeon-room.clear" : "combat.monster.clear",
		argumentsByID: { encounterId: review.encounterID, expectedSnapshot: review.snapshot },
	})
	reviewedBatch.value = undefined
}

function cellClass(cell: NonNullable<ActiveCombatStatusData["map"]>["cells"][number]): Record<string, boolean> {
  const monster = cell.occupantKind === "monster" ? props.data?.monsters.find((entry) => entry.id === cell.occupantId) : undefined
  const party = cell.occupantKind === "party" ? props.data?.party.find((entry) => String(entry.index) === cell.occupantId) : undefined
  return {
    party: cell.occupantKind === "party",
    monster: cell.occupantKind === "monster",
    current: Boolean(monster?.currentActor || party?.currentActor),
    target: Boolean(party?.currentTarget),
  }
}
</script>

<template>
  <section class="approved-view active-combat-console" aria-label="Active combat console">
    <header class="approved-view-header">
      <div><span>DETACHED TACTICAL TELEMETRY</span><h3>Active Combat</h3></div>
      <small>{{ updatedAt ? `UPDATED ${updatedAt.toLocaleTimeString([], { hour12: false })}` : "AWAITING STATUS" }}</small>
    </header>
    <p v-if="data?.configurationEditing" role="status">Close Configuration with Apply or Cancel before changing combat speed.</p>
    <PreferenceNumberControl v-if="data" command-id="combat.presentation.speed.set" :command="commands?.find(command => command.id === 'combat.presentation.speed.set')" :value="data.presentationSpeedPercent" :disabled="pending || data.configurationEditing" @apply="argumentsByID => emit('mutate', { commandID: 'combat.presentation.speed.set', argumentsByID })" />
    <p v-if="!data" class="empty-state">WAITING FOR ACTIVE COMBAT STATUS</p>
    <section v-else-if="!data.active" class="approved-panel combat-unavailable"><strong>NO ACTIVE ENCOUNTER</strong><p>{{ data.unavailableReason }}</p><button type="button" :disabled="pending" @click="emit('openEncounter')">OPEN ENCOUNTER OPTIONS</button></section>
    <template v-else>
      <section class="combat-telemetry">
        <div><span>Map</span><strong>{{ data.map?.name }}</strong><small>{{ data.map?.territory }} #{{ data.map?.index }}</small></div>
        <div><span>Entry</span><strong>{{ data.map?.entryDirection }}</strong><small>ENCOUNTER {{ data.encounterId }}</small></div>
        <div><span>Monsters</span><strong>{{ data.activeMonsterCount }} / {{ data.totalMonsterCount }}</strong><small>ACTIVE / TOTAL</small></div>
        <div><span>State</span><strong>{{ data.map?.victory ? "VICTORY" : data.workflowActive ? "WORKFLOW" : "LIVE" }}</strong><small>{{ data.map?.pendingAdvanceTurn ? "TURN ADVANCE PENDING" : "QUEUE STABLE" }}</small></div>
        <div><span>Objects</span><strong>{{ data.map?.objectCount }}</strong><small>CHESTS / CORPSES</small></div>
      </section>

      <div class="active-combat-layout">
        <section class="approved-panel tactical-map"><header><strong>11 × 11 tactical map</strong><small>ACTOR YELLOW · TARGET RED</small></header><div class="combat-grid"><button v-for="cell in data.map?.cells" :key="`${cell.x}-${cell.y}`" type="button" :class="cellClass(cell)" :title="cell.occupantName ? `${cell.x},${cell.y} · ${cell.occupantName}` : `${cell.x},${cell.y} · tile ${cell.tileId}`" @click="cell.occupantKind === 'monster' && (selectedMonsterID = cell.occupantId ?? '')"><b v-if="cell.occupantKind === 'party'">P{{ Number(cell.occupantId) + 1 }}</b><b v-else-if="cell.occupantKind === 'monster'">M{{ cell.occupantId }}</b><small v-else>{{ cell.tileId }}</small></button></div></section>

        <section class="approved-panel combat-forecast"><header><strong>DEX queue forecast</strong><small>NEXT {{ data.forecast.length }}</small></header><ol><li v-for="entry in data.forecast" :key="`${entry.order}-${entry.kind}-${entry.id}`"><b>{{ entry.order }}</b><strong>{{ entry.name }}</strong><span>DEX {{ entry.dexterity }}</span><span>RESET {{ entry.resetCountdown }}</span><small>{{ entry.canAct ? "CAN ACT" : "SKIP" }}</small></li></ol></section>
      </div>

      <section class="approved-panel combat-party"><header><strong>Party rotation</strong><small>{{ data.party.length }} QUEUED</small></header><div class="combatant-cards"><article v-for="member in data.party" :key="member.index" :class="{ current: member.currentActor, target: member.currentTarget }"><header><strong>P{{ member.index + 1 }} · {{ member.name }}</strong><small>{{ member.status }}</small></header><p>HP {{ member.currentHp }}/{{ member.maximumHp }} · MP {{ member.currentMp }}/{{ member.maximumMp }}</p><p>DEX {{ member.dexterity }} · QUEUE {{ member.queueCountdown }} · {{ member.canAct ? "READY" : "SKIP" }}</p><small>{{ member.x }},{{ member.y }} · {{ member.queueStatus }}</small></article></div></section>

      <section class="approved-panel combat-monsters"><header><strong>Monster roster</strong><small>STABLE ENCOUNTER IDs</small></header><div class="table-scroll"><table><thead><tr><th>ID</th><th>Monster</th><th>HP</th><th>DEX</th><th>Position</th><th>Conditions</th><th>Queue</th></tr></thead><tbody><tr v-for="monster in data.monsters" :key="monster.id" :class="{ selected: selectedMonsterID === monster.id, defeated: monster.dead }" @click="selectedMonsterID = monster.id"><td>M{{ monster.id }}<small>IDX {{ monster.managerIndex }}</small></td><td><strong>{{ monster.name }}</strong><small>TYPE {{ monster.typeId }}</small></td><td>{{ monster.currentHp }}/{{ monster.maximumHp }}</td><td>{{ monster.currentDexterity }}<small>BASE {{ monster.baseDexterity }}</small></td><td>{{ monster.x }}, {{ monster.y }}</td><td><span v-if="monster.dead" class="trait-tag">DEAD</span><span v-if="monster.asleep" class="trait-tag">ASLEEP</span><span v-if="monster.poisoned" class="trait-tag">POISONED</span><span v-if="monster.invisible" class="trait-tag">INVISIBLE</span></td><td>{{ monster.queueCountdown }} · {{ monster.canAct ? "READY" : "SKIP" }}</td></tr></tbody></table></div><footer v-if="selectedMonster"><div><strong>Selected M{{ selectedMonster.id }} · {{ selectedMonster.name }}</strong><small>{{ selectedMonster.abilities.length ? selectedMonster.abilities.join(" · ") : "NO SPECIAL ABILITIES" }}</small></div><label>Damage<input v-model.number="damageAmount" data-combat-damage type="number" min="1" max="999" /></label><button type="button" data-combat-damage-review :disabled="pending || mutationUnavailable || selectedMonster.dead || damageAmount < 1 || damageAmount > 999" @click="stageMonsterMutation('damage')">REVIEW DAMAGE</button><button type="button" data-combat-kill-review :disabled="pending || mutationUnavailable || selectedMonster.dead" @click="stageMonsterMutation('kill')">REVIEW KILL</button><button type="button" :disabled="pending || mutationUnavailable || selectedMonster.asleep || selectedMonster.dead" @click="stageSleep(true)">REVIEW SLEEP</button><button type="button" :disabled="pending || mutationUnavailable || !selectedMonster.asleep || selectedMonster.dead" @click="stageSleep(false)">REVIEW WAKE</button></footer></section>

	  <section class="instrument-setting"><div><strong>GLOBAL GAMEPLAY RNG</strong><p>Resetting this seed affects all subsequent gameplay randomness, including after combat.</p></div><label>Seed<input v-model.number="rngSeed" data-combat-rng-seed type="number" min="1" max="999999" /></label><span>CURRENT {{ data.rngSeed }}</span><button type="button" data-combat-seed-review :disabled="pending || mutationUnavailable || rngSeed < 1 || rngSeed > 999999" @click="stageSeed">REVIEW RNG RESET</button></section>
	  <section v-if="reviewedSeed" class="combat-condition-review danger-card"><div><strong>RESET GLOBAL GAMEPLAY RNG</strong><p>{{ reviewedSeed.beforeSeed }} → {{ reviewedSeed.afterSeed }}. Post-combat randomness is also reset.</p></div><button type="button" data-combat-seed-apply :disabled="pending" @click="applySeed">APPLY RNG RESET</button><button type="button" :disabled="pending" @click="reviewedSeed = undefined">CANCEL</button></section>
	  <section v-if="reviewedMonsterMutation" class="combat-condition-review danger-card"><div><strong>{{ reviewedMonsterMutation.kind === 'damage' ? `DAMAGE ${reviewedMonsterMutation.damage}` : 'KILL' }} M{{ reviewedMonsterMutation.monsterID }} · {{ reviewedMonsterMutation.name }}</strong><p>Requires reviewed HP {{ reviewedMonsterMutation.expectedHP }}. Lethal debug damage synchronizes the queue and victory, but awards no XP and creates no corpse or treasure.</p></div><button type="button" data-combat-monster-apply :disabled="pending" @click="applyMonsterMutation">APPLY {{ reviewedMonsterMutation.kind.toUpperCase() }}</button><button type="button" :disabled="pending" @click="reviewedMonsterMutation = undefined">CANCEL</button></section>
	  <section v-if="reviewedBatch" class="combat-condition-review danger-card"><div><strong>{{ reviewedBatch.kind === 'kill-all' ? 'KILL ALL REVIEWED MONSTERS' : reviewedBatch.kind === 'clear-room' ? 'CLEAR FIXED DUNGEON ROOM' : 'CLEAR REVIEWED MONSTERS' }}</strong><p>Encounter {{ reviewedBatch.encounterID }} · {{ reviewedBatch.activeCount }} active / {{ reviewedBatch.totalCount }} total. {{ reviewedBatch.kind === 'kill-all' ? 'No XP or objects; queue synchronization may trigger victory.' : reviewedBatch.kind === 'clear-room' ? 'Defeats hostile enemies with no rewards and declares victory without exiting combat.' : 'Outcome-free: removes manager and queue monsters with no XP, objects, or victory.' }}</p></div><button type="button" data-combat-batch-apply :disabled="pending" @click="applyBatch">CONFIRM {{ reviewedBatch.kind === 'kill-all' ? 'KILL ALL' : reviewedBatch.kind === 'clear-room' ? 'CLEAR ROOM' : 'CLEAR' }}</button><button type="button" :disabled="pending" @click="reviewedBatch = undefined">CANCEL</button></section>
	  <section v-else class="instrument-setting"><div><strong>BATCH MONSTER CONTROL</strong><p>Both operations require a second confirmation against this exact reviewed encounter snapshot.</p></div><button type="button" data-combat-kill-all-review :disabled="pending || mutationUnavailable || data.activeMonsterCount === 0" @click="stageBatch('kill-all')">REVIEW KILL ALL</button><button type="button" data-combat-clear-review :disabled="pending || mutationUnavailable || data.totalMonsterCount === 0" @click="stageBatch('clear')">REVIEW CLEAR</button></section>
	  <section class="instrument-setting"><div><strong>FIXED DUNGEON ROOM</strong><p v-if="data.fixedDungeonRoom">Dungeon {{ data.dungeonIndex }} · room {{ data.dungeonRoom }} · {{ data.activeHostileMonsterCount }} hostile. Declares victory but does not exit combat.</p><p v-else>Requires combat entered from an authoritative A* or F* dungeon cell.</p></div><button type="button" data-combat-clear-room-review :disabled="pending || mutationUnavailable || !data.fixedDungeonRoom || data.map?.victory || data.activeHostileMonsterCount === 0" @click="stageBatch('clear-room')">REVIEW CLEAR ROOM</button></section>

      <section class="instrument-setting combat-spawn-setting"><div><strong>REINFORCEMENTS</strong><p>Uses unoccupied authoritative CBT positions and commits the manager and DEX queue as one batch.</p></div><label>Monster<select v-model="spawnMonsterType"><option v-for="monsterType in data.monsterTypes" :key="monsterType.id" :value="String(monsterType.id)">{{ monsterType.name }} · TYPE {{ monsterType.id }}</option></select></label><label>Count<select v-model="spawnCountMode"><option value="automatic">Automatic</option><option value="exact">Exact</option><option value="maximum">Maximum</option></select></label><label v-if="spawnCountMode === 'exact'">Exact<input v-model.number="spawnExactCount" type="number" min="1" max="16" /></label><label>Friends<input v-model="spawnFriends" type="checkbox" /></label><label>Seed<input :value="spawnSeed" @input="spawnSeed = ($event.target as HTMLInputElement).value" data-combat-spawn-seed type="number" min="0" max="2147483647" placeholder="session RNG" /></label><button type="button" data-combat-spawn-review :disabled="pending || mutationUnavailable || !spawnMonsterType || (spawnCountMode === 'exact' && (spawnExactCount < 1 || spawnExactCount > 16))" @click="stageSpawn">REVIEW SPAWN</button></section>
      <section v-if="reviewedSpawn" class="combat-condition-review"><div><strong>SPAWN TYPE {{ reviewedSpawn.monsterType }} · {{ reviewedSpawn.countMode.toUpperCase() }}</strong><p>{{ reviewedSpawn.exactCount ? `${reviewedSpawn.exactCount} monsters` : "Count resolved at commit" }} · {{ reviewedSpawn.friends ? "friends enabled" : "primary type only" }} · {{ reviewedSpawn.seed ? `seed ${reviewedSpawn.seed}` : "session RNG" }}</p></div><button type="button" data-combat-spawn-apply :disabled="pending" @click="applySpawn">APPLY SPAWN</button><button type="button" :disabled="pending" @click="reviewedSpawn = undefined">CANCEL</button></section>
      <section v-if="reviewedCondition" class="combat-condition-review"><div><strong>{{ reviewedCondition.asleep ? "SLEEP" : "WAKE" }} M{{ reviewedCondition.monsterID }} · {{ reviewedCondition.name }}</strong><p>Applies only if the encounter snapshot and current condition still match this review.</p></div><button type="button" :disabled="pending" @click="applyCondition">APPLY CONDITION</button><button type="button" :disabled="pending" @click="reviewedCondition = undefined">CANCEL</button></section>
      <section v-if="reviewedExit" class="combat-condition-review danger-card"><div><strong>DISCARD ENCOUNTER {{ reviewedExit.encounterID }}</strong><p>Returns to the pre-combat location without resolving victory, defeat, evacuation, overworld contact, or pirate salvage.</p></div><button type="button" :disabled="pending" @click="applyExit">DISCARD ENCOUNTER</button><button type="button" :disabled="pending" @click="reviewedExit = undefined">CANCEL</button></section>
      <section v-else class="instrument-setting"><div><strong>DEBUG EXIT</strong><p>Discard this reviewed tactical encounter. Its overworld source remains unchanged and may immediately re-engage.</p></div><button type="button" :disabled="pending || mutationUnavailable" @click="stageExit">REVIEW DEBUG EXIT</button></section>
	  <p class="deferred-control"><strong>DEBUG DEFEAT POLICY</strong> Lethal damage and kill award no XP and create no corpse or treasure; they synchronize the DEX queue and trigger victory when appropriate. Clear is outcome-free and creates no victory.</p>
    </template>
  </section>
</template>
