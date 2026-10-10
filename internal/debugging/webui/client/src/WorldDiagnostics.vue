<script setup lang="ts">
import type { DeathStatusData, GuardStatusData, ShadowlordStatusData } from "./approved_tool_views"

defineProps<{
  guards?: GuardStatusData
  shadowlords?: ShadowlordStatusData
  death?: DeathStatusData
  pending: boolean
}>()
const emit = defineEmits<{ refresh: [] }>()
</script>

<template>
  <section class="approved-view world-diagnostics" aria-label="Exceptional world-state diagnostics">
    <header class="approved-view-header">
      <div><span>HIDDEN AUTHORITATIVE STATE</span><h3>World Diagnostics</h3></div>
      <button type="button" :disabled="pending" @click="emit('refresh')">{{ pending ? 'REFRESHING...' : 'REFRESH ALL' }}</button>
    </header>

    <div class="world-diagnostic-grid">
      <section class="approved-panel guard-diagnostics">
        <header><strong>Guard Response</strong><small>{{ guards?.supported ? guards.locationName : 'UNAVAILABLE' }}</small></header>
        <p v-if="!guards" class="empty-state">WAITING FOR GUARD STATE</p>
        <template v-else-if="guards.supported">
          <dl class="diagnostic-metrics">
            <div><dt>Criminal</dt><dd :class="{ warning: guards.criminal }">{{ guards.criminal ? 'YES' : 'NO' }}</dd></div>
            <div><dt>Friendly NPCs</dt><dd>{{ guards.friendlyNpcCount }}</dd></div>
            <div><dt>Potential Guards</dt><dd>{{ guards.potentialGuardCount }}</dd></div>
            <div><dt>Hostile Guards</dt><dd>{{ guards.hostilePotentialGuardCount }}</dd></div>
            <div><dt>Other Hostiles</dt><dd>{{ guards.otherHostileNpcCount }}</dd></div>
            <div><dt>Floor</dt><dd>{{ guards.floor }}</dd></div>
          </dl>
        </template>
        <p v-else class="read-only-notice">{{ guards.unavailableReason }}</p>
      </section>

      <section class="approved-panel death-diagnostics">
        <header><strong>Death Consequences</strong><small>{{ death?.allPartyDead ? 'PARTY DEAD' : 'PARTY ACTIVE' }}</small></header>
        <p v-if="!death" class="empty-state">WAITING FOR DEATH STATE</p>
        <template v-else>
          <dl class="diagnostic-metrics">
            <div><dt>Dead / Active</dt><dd>{{ death.deadPartyCount }} / {{ death.activePartyCount }}</dd></div>
            <div><dt>Karma</dt><dd>{{ death.karma }}</dd></div>
            <div><dt>XP Retained</dt><dd>{{ death.experienceRetainedPercent }}%</dd></div>
            <div><dt>XP Lost</dt><dd>{{ death.experienceLostPercent }}%</dd></div>
            <div><dt>Minimum Karma</dt><dd>{{ death.minimumPostDeathKarma }}</dd></div>
            <div><dt>Minimum Food</dt><dd>{{ death.minimumPostDeathFood }}</dd></div>
          </dl>
          <p class="diagnostic-route">RESURRECTION · {{ death.resurrection.locationName }} · {{ death.resurrection.x }}, {{ death.resurrection.y }} · FLOOR {{ death.resurrection.floor }}</p>
        </template>
      </section>
    </div>

    <section class="approved-panel shadowlord-diagnostics">
      <header><strong>Shadowlord State</strong><small>{{ shadowlords ? `${shadowlords.aliveCount} / ${shadowlords.shadowlords.length} ALIVE` : 'WAITING' }}</small></header>
      <p v-if="!shadowlords" class="empty-state">WAITING FOR SHADOWLORD STATE</p>
      <template v-else>
        <div class="shadowlord-cards">
          <article v-for="lord in shadowlords.shadowlords" :key="lord.id" :class="{ defeated: !lord.alive }">
            <span>SL-{{ lord.id }}</span><strong>{{ lord.name }}</strong><small>{{ lord.antiVirtue }}</small><b>{{ lord.alive ? 'ALIVE' : 'DEFEATED' }}</b>
          </article>
        </div>
        <div class="shadowlord-presence">
          <p><span>SUMMONED</span><strong>{{ shadowlords.summonedShadowlord?.name ?? 'NONE' }}</strong></p>
          <p><span>AUTOMATIC PRESENCE</span><strong>{{ shadowlords.automaticPresence?.name ?? 'NONE' }}</strong></p>
          <p><span>DAY</span><strong>{{ shadowlords.day }}</strong></p>
        </div>
        <div class="assignment-ledger">
          <strong>TODAY'S TOWN ASSIGNMENTS</strong>
          <span v-if="shadowlords.townAssignments.length === 0">No assignments.</span>
          <span v-for="assignment in shadowlords.townAssignments" :key="assignment.locationId">{{ assignment.locationName }} · {{ assignment.shadowlordName }}</span>
        </div>
      </template>
    </section>
  </section>
</template>
