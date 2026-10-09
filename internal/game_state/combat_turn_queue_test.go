// Tests for CombatTurnQueue — the dex-based turn priority system.
//
// These tests verify behavioral equivalence with the original Ultima V
// countdown scheduler in original combat scheduler. The core formula is:
//
//	countdown = 36 - dexterity
//
// Higher dex → lower countdown → more frequent turns. Each pass through
// the queue decrements all active countdowns; when one reaches 0, that
// combatant acts and their countdown resets.
// See ref:COMBAT-0002.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Mock CombatantProvider ─────────────────────────────────────────────────

// mockCombatant is a minimal CombatantProvider for unit testing.
type mockCombatant struct {
	dexterity   int
	isAlive     bool
	canTakeTurn bool
	isPC        bool
}

func (m *mockCombatant) GetDexterity() int { return m.dexterity }
func (m *mockCombatant) IsAlive() bool     { return m.isAlive }
func (m *mockCombatant) CanTakeTurn() bool { return m.canTakeTurn }
func (m *mockCombatant) IsPC() bool        { return m.isPC }

func newMockPC(dex int) *mockCombatant {
	return &mockCombatant{dexterity: dex, isAlive: true, canTakeTurn: true, isPC: true}
}

func newMockMonster(dex int) *mockCombatant {
	return &mockCombatant{dexterity: dex, isAlive: true, canTakeTurn: true, isPC: false}
}

// ─── Basic Turn Ordering ────────────────────────────────────────────────────

// TestCombatTurnQueue_HighDexActsFirst verifies that a combatant with higher
// dexterity gets the first turn, matching original combat scheduler behavior where
// countdown = 36 - dex means higher dex = shorter countdown.
// See ref:COMBAT-0002.
func TestCombatTurnQueue_HighDexActsFirst(t *testing.T) {
	q := NewCombatTurnQueue()

	slowPC := newMockPC(10)           // countdown = 26
	fastMonster := newMockMonster(25) // countdown = 11
	midPC := newMockPC(18)            // countdown = 18

	// Add in mixed order to verify ordering is by countdown, not insertion
	q.AddCombatant(slowPC, 0)
	q.AddCombatant(fastMonster, 0)
	q.AddCombatant(midPC, 1)

	first := q.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.Same(t, fastMonster, first.Combatant,
		"dex-25 combatant (countdown=11) should act before dex-18 (countdown=18) and dex-10 (countdown=26)")
}

func TestCombatTurnQueue_ReusedSourceSlotControlsTurnOrder(t *testing.T) {
	q := NewCombatTurnQueue()
	originalPC := newMockPC(15)
	monster := newMockMonster(15)
	q.AddCombatant(originalPC, 0)
	q.AddCombatant(monster, 0)
	original := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	monsterIdentity := CombatTurnIdentity{IsPartyMember: false, EntityIndex: 0}
	q.markFledIdentity(original)
	clone := newMockPC(15)
	q.AddClonedCombatant(clone, 0, 1, monsterIdentity)
	cloneEntry := q.GetEntryByIdentity(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0, Slot: 1})
	monsterEntry := q.GetEntryByIdentity(monsterIdentity)
	require.NotNil(t, cloneEntry)
	require.NotNil(t, monsterEntry)
	assert.Equal(t, 0, cloneEntry.sourceSlot)
	cloneEntry.turnCountdown = 1
	monsterEntry.turnCountdown = 1

	actor := q.AdvanceToNextActor()

	require.NotNil(t, actor)
	assert.Same(t, clone, actor.Combatant,
		"a clone in reused source slot 0 acts before a monster in source slot 6")
}

func TestCombatTurnQueue_ReplacingCurrentSourceWaitsUntilNextPass(t *testing.T) {
	q := NewCombatTurnQueue()
	current := newMockMonster(15)
	next := newMockMonster(15)
	q.AddCombatant(current, 0)
	q.AddCombatant(next, 1)
	q.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: 0}).turnCountdown = 1
	q.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: 1}).turnCountdown = 1
	require.Same(t, current, q.AdvanceToNextActor().Combatant)
	q.markFledIdentity(CombatTurnIdentity{EntityIndex: 0})
	replacement := newMockMonster(15)
	q.AddCombatant(replacement, 2)
	replacementEntry := q.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: 2})
	require.NotNil(t, replacementEntry)
	assert.Equal(t, sourceMonsterSlotStart, replacementEntry.sourceSlot)
	replacementEntry.turnCountdown = 1

	actor := q.AdvanceToNextActor()

	require.NotNil(t, actor)
	assert.Same(t, next, actor.Combatant,
		"a replacement in the source slot just processed must wait for the next queue pass")
}

func TestCombatTurnQueue_DeadCorpseSourceSlotRemainsOccupied(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(15), 0)
	monster := newMockMonster(15)
	q.AddCombatant(monster, 0)
	q.markDeadIdentity(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0})
	q.AddClonedCombatant(newMockPC(15), 1, 1, CombatTurnIdentity{EntityIndex: 0})
	cloneEntry := q.GetEntryByIdentity(CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1, Slot: 1})
	require.NotNil(t, cloneEntry)
	assert.Equal(t, 1, cloneEntry.sourceSlot,
		"an OBJECT_MASK corpse continues to occupy its original source slot")
}

// TestCombatTurnQueue_SameDexRoundRobin verifies that when all combatants have
// the same dexterity, they act in insertion order (round-robin), matching the
// original's linear scan of the combat queue[] array.
// See ref:COMBAT-0002.
func TestCombatTurnQueue_SameDexRoundRobin(t *testing.T) {
	q := NewCombatTurnQueue()

	pc0 := newMockPC(15)
	pc1 := newMockPC(15)
	pc2 := newMockPC(15)

	q.AddCombatant(pc0, 0)
	q.AddCombatant(pc1, 1)
	q.AddCombatant(pc2, 2)

	// All have countdown = 36 - 15 = 21. After 21 decrements, they should
	// fire in insertion order: pc0, pc1, pc2.
	first := q.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.Equal(t, 0, first.EntityIndex, "first turn should be PC 0")

	second := q.AdvanceToNextActor()
	require.NotNil(t, second)
	assert.Equal(t, 1, second.EntityIndex, "second turn should be PC 1")

	third := q.AdvanceToNextActor()
	require.NotNil(t, third)
	assert.Equal(t, 2, third.EntityIndex, "third turn should be PC 2")

	// And cycle repeats
	fourth := q.AdvanceToNextActor()
	require.NotNil(t, fourth)
	assert.Equal(t, 0, fourth.EntityIndex, "fourth turn should cycle back to PC 0")
}

// TestCombatTurnQueue_HighDexActsMoreFrequently verifies the fundamental dex
// advantage: a dex-30 combatant (countdown=6) should act roughly 3x as often
// as a dex-18 combatant (countdown=18) over many turns.
func TestCombatTurnQueue_HighDexActsMoreFrequently(t *testing.T) {
	q := NewCombatTurnQueue()

	slow := newMockPC(18)      // countdown = 18
	fast := newMockMonster(30) // countdown = 6

	q.AddCombatant(slow, 0)
	q.AddCombatant(fast, 0)

	slowTurns := 0
	fastTurns := 0

	// Run enough turns to see the pattern clearly
	for i := 0; i < 30; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry)

		if entry.Combatant == slow {
			slowTurns++
		} else {
			fastTurns++
		}
	}

	// fast (countdown=6) should act ~3x as often as slow (countdown=18)
	// Allow some tolerance for boundary effects
	assert.Greater(t, fastTurns, slowTurns*2,
		"dex-30 (countdown=6) should act at least 2x more than dex-18 (countdown=18): got fast=%d slow=%d",
		fastTurns, slowTurns)
}

// ─── Exact Sequence Verification (Original C Equivalence) ───────────────────

// TestCombatTurnQueue_ExactSequence_TwoCombatants traces slot-by-slot
// countdowns: PC Dexterity 20 begins at 16, monster Dexterity 25 begins at 11.
// The monster acts on pass 11 and resets to 11; the PC acts on pass 16.
// Scanning resumes from the retained cursor, not a restarted round. ref:COMBAT-0002
func TestCombatTurnQueue_ExactSequence_TwoCombatants(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(20)           // countdown = 16
	monster := newMockMonster(25) // countdown = 11

	q.AddCombatant(pc, 0)
	q.AddCombatant(monster, 0)

	// First actor: Monster (countdown 11 < 16)
	first := q.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.Same(t, monster, first.Combatant,
		"monster (dex=25, countdown=11) should act first")

	// Second actor: PC (countdown reaches 0 at pass 16)
	second := q.AdvanceToNextActor()
	require.NotNil(t, second)
	assert.Same(t, pc, second.Combatant,
		"PC (dex=20, countdown=16) should act second")

	// Third actor: Monster again (reset countdown 11 is shorter than PC's reset 16)
	// Monster reset at pass 11, so next action at pass 11+11=22
	// PC reset at pass 16, so next action at pass 16+16=32
	// Monster acts at 22, PC acts at 32 → Monster again
	third := q.AdvanceToNextActor()
	require.NotNil(t, third)
	assert.Same(t, monster, third.Combatant,
		"monster should get third turn (shorter countdown after reset)")
}

// TestCombatTurnQueue_ExactSequence_ThreeCombatants verifies turn order with
// 3 combatants of different dex values.
//
// Setup: PC0 (dex=15, cd=21), PC1 (dex=20, cd=16), Mon (dex=30, cd=6)
//
// Expected first 6 actions:
//
//	Action 1: Mon (cd reaches 0 at pass 6)
//	Action 2: Mon (reset cd=6, reaches 0 again at pass 12)
//	Action 3: PC1 (cd reaches 0 at pass 16)
//	Action 4: Mon (reset from pass 12, reaches 0 at pass 18)
//	Action 5: PC0 (cd reaches 0 at pass 21)
//	Action 6: Mon (reset from pass 18, reaches 0 at pass 24)
func TestCombatTurnQueue_ExactSequence_ThreeCombatants(t *testing.T) {
	q := NewCombatTurnQueue()

	pc0 := newMockPC(15)          // countdown = 21
	pc1 := newMockPC(20)          // countdown = 16
	monster := newMockMonster(30) // countdown = 6

	q.AddCombatant(pc0, 0)
	q.AddCombatant(pc1, 1)
	q.AddCombatant(monster, 0)

	expectedOrder := []struct {
		combatant CombatantProvider
		label     string
	}{
		{monster, "Mon(dex=30)"},
		{monster, "Mon(dex=30)"},
		{pc1, "PC1(dex=20)"},
		{monster, "Mon(dex=30)"},
		{pc0, "PC0(dex=15)"},
		{monster, "Mon(dex=30)"},
	}

	for i, expected := range expectedOrder {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry, "action %d should return an actor", i+1)
		assert.Same(t, expected.combatant, entry.Combatant,
			"action %d should be %s", i+1, expected.label)
	}
}

// ─── MarkDead / MarkFled ────────────────────────────────────────────────────

// TestCombatTurnQueue_MarkDead_SkipsTurns verifies that dead combatants are
// completely skipped during turn advancement.
func TestCombatTurnQueue_MarkDead_SkipsTurns(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(20)
	monster := newMockMonster(25)

	q.AddCombatant(pc, 0)
	q.AddCombatant(monster, 0)

	// Kill the monster before it gets a turn
	q.MarkDead(0, false)

	// Now only the PC should ever get turns
	for i := 0; i < 5; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry, "should still produce turns with one live combatant")
		assert.Same(t, pc, entry.Combatant,
			"only surviving PC should get turns after monster death")
	}
}

// TestCombatTurnQueue_MarkFled_SkipsTurns verifies that fled combatants are
// treated the same as dead for turn purposes.
func TestCombatTurnQueue_MarkFled_SkipsTurns(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(20)
	monster := newMockMonster(25)

	q.AddCombatant(pc, 0)
	q.AddCombatant(monster, 0)

	// Monster flees
	q.MarkFled(0, false)

	for i := 0; i < 3; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry)
		assert.Same(t, pc, entry.Combatant,
			"only remaining PC should get turns after monster flees")
	}
}

// TestCombatTurnQueue_CanTakeTurnFalse_ReachesScheduledTurn verifies that
// temporary conditions are resolved only after the actor reaches its DEX boundary.
func TestCombatTurnQueue_CanTakeTurnFalse_ReachesScheduledTurn(t *testing.T) {
	q := NewCombatTurnQueue()

	activePC := newMockPC(15)
	sleepingPC := &mockCombatant{dexterity: 20, isAlive: true, canTakeTurn: false, isPC: true}

	q.AddCombatant(activePC, 0)
	q.AddCombatant(sleepingPC, 1)

	entry := q.AdvanceToNextActor()
	require.NotNil(t, entry)
	assert.Same(t, sleepingPC, entry.Combatant,
		"sleeping PC reaches its scheduled boundary before the slower active PC")
	assert.False(t, entry.Combatant.CanTakeTurn())

	// Now wake the sleeping PC up
	sleepingPC.canTakeTurn = true

	// Both continue participating after the sleeping condition clears.
	seenSleeper := false

	for i := 0; i < 20; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry)

		if entry.Combatant == sleepingPC {
			seenSleeper = true
			break
		}
	}

	assert.True(t, seenSleeper, "woken PC should eventually get a turn")
}

// ─── CountRemaining (whatsleft) ─────────────────────────────────────────────

// TestCombatTurnQueue_CountRemaining_AllAlive verifies correct counting
// when all combatants are active.
func TestCombatTurnQueue_CountRemaining_AllAlive(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockPC(18), 1)
	q.AddCombatant(newMockMonster(12), 0)
	q.AddCombatant(newMockMonster(20), 1)
	q.AddCombatant(newMockMonster(14), 2)

	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 2, pcs, "should count 2 PCs")
	assert.Equal(t, 3, monsters, "should count 3 monsters")
}

// TestCombatTurnQueue_CountRemaining_AfterDeaths verifies counting
// after some combatants are killed.
func TestCombatTurnQueue_CountRemaining_AfterDeaths(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockPC(18), 1)
	q.AddCombatant(newMockMonster(12), 0)
	q.AddCombatant(newMockMonster(20), 1)

	// Kill one PC and one monster
	q.MarkDead(1, true)  // PC 1
	q.MarkDead(0, false) // Monster 0

	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 1, pcs, "should count 1 surviving PC")
	assert.Equal(t, 1, monsters, "should count 1 surviving monster")
}

// TestCombatTurnQueue_CountRemaining_Victory verifies that killing all
// monsters results in 0 monster count (victory condition).
func TestCombatTurnQueue_CountRemaining_Victory(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockMonster(12), 0)
	q.AddCombatant(newMockMonster(20), 1)

	q.MarkDead(0, false)
	q.MarkDead(1, false)

	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 1, pcs, "should have 1 surviving PC")
	assert.Equal(t, 0, monsters, "all monsters dead = victory condition")
}

// TestCombatTurnQueue_CountRemaining_Defeat verifies that killing all
// PCs results in 0 PC count (defeat condition).
func TestCombatTurnQueue_CountRemaining_Defeat(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockPC(18), 1)
	q.AddCombatant(newMockMonster(12), 0)

	q.MarkDead(0, true)
	q.MarkDead(1, true)

	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 0, pcs, "all PCs dead = defeat condition")
	assert.Equal(t, 1, monsters, "should have 1 surviving monster")
}

// TestCombatTurnQueue_CountRemaining_IncludesSleepingLivingCombatants verifies
// that temporary incapacity does not make a combatant dead for battle outcomes.
func TestCombatTurnQueue_CountRemaining_IncludesSleepingLivingCombatants(t *testing.T) {
	q := NewCombatTurnQueue()

	activePC := newMockPC(15)
	sleepingPC := &mockCombatant{dexterity: 18, isAlive: true, canTakeTurn: false, isPC: true}
	monster := newMockMonster(12)

	q.AddCombatant(activePC, 0)
	q.AddCombatant(sleepingPC, 1)
	q.AddCombatant(monster, 0)

	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 2, pcs, "sleeping living PC should still count as remaining")
	assert.Equal(t, 1, monsters)
}

// ─── ShouldAdvanceTime ──────────────────────────────────────────────────────

// TestCombatTurnQueue_ShouldAdvanceTime_EveryTenActions verifies one
// calendar-advance request after every ten completed actions. ref:TIME-0003
func TestCombatTurnQueue_ShouldAdvanceTime_EveryTenActions(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(30), 0) // Fast dex for quick turns

	// Take 9 actions — should NOT advance time
	for i := 0; i < 9; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry)
		assert.False(t, q.ShouldAdvanceTime(),
			"should not advance time after only %d actions", i+1)
	}

	// 10th action should trigger time advance
	entry := q.AdvanceToNextActor()
	require.NotNil(t, entry)
	assert.True(t, q.ShouldAdvanceTime(),
		"should advance time after 10 actions")

	// Counter should have reset — next 9 should not trigger
	for i := 0; i < 9; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry)
		assert.False(t, q.ShouldAdvanceTime(),
			"should not advance time after only %d actions in second cycle", i+1)
	}

	// 20th total action triggers again
	entry = q.AdvanceToNextActor()
	require.NotNil(t, entry)
	assert.True(t, q.ShouldAdvanceTime(),
		"should advance time after another 10 actions")
}

// ─── Edge Cases ─────────────────────────────────────────────────────────────

// TestCombatTurnQueue_EmptyQueue_ReturnsNil verifies that AdvanceToNextActor
// gracefully returns nil when no combatants exist.
func TestCombatTurnQueue_EmptyQueue_ReturnsNil(t *testing.T) {
	q := NewCombatTurnQueue()

	entry := q.AdvanceToNextActor()
	assert.Nil(t, entry, "empty queue should return nil")
}

// TestCombatTurnQueue_AllDead_ReturnsNil verifies that AdvanceToNextActor
// returns nil when all combatants are dead (combat should have ended).
func TestCombatTurnQueue_AllDead_ReturnsNil(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockMonster(20), 0)

	q.MarkDead(0, true)
	q.MarkDead(0, false)

	entry := q.AdvanceToNextActor()
	assert.Nil(t, entry, "all-dead queue should return nil")
}

// TestCombatTurnQueue_HighDexClamped verifies that dex >= 35 results in
// a minimum countdown of 1 (not 0 or negative), preventing infinite loops.
func TestCombatTurnQueue_HighDexClamped(t *testing.T) {
	q := NewCombatTurnQueue()

	// dex=35 → countdown = 36-35 = 1 (minimum)
	superFast := newMockPC(35)
	q.AddCombatant(superFast, 0)

	entry := q.AdvanceToNextActor()
	require.NotNil(t, entry)
	assert.Same(t, superFast, entry.Combatant)

	// dex=36 → countdown would be 0, clamped to 1
	q2 := NewCombatTurnQueue()
	tooFast := newMockMonster(36)
	q2.AddCombatant(tooFast, 0)

	entry2 := q2.AdvanceToNextActor()
	require.NotNil(t, entry2)
	assert.Same(t, tooFast, entry2.Combatant)

	// dex=50 → countdown would be -14, clamped to 1
	q3 := NewCombatTurnQueue()
	absurd := newMockPC(50)
	q3.AddCombatant(absurd, 0)

	entry3 := q3.AdvanceToNextActor()
	require.NotNil(t, entry3)
	assert.Same(t, absurd, entry3.Combatant)
}

// TestCombatTurnQueue_GetEntry_FindsCorrectEntry verifies the lookup utility.
func TestCombatTurnQueue_GetEntry_FindsCorrectEntry(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(15)
	monster := newMockMonster(20)

	q.AddCombatant(pc, 2)
	q.AddCombatant(monster, 5)

	pcEntry := q.GetEntry(2, true)
	require.NotNil(t, pcEntry)
	assert.Same(t, pc, pcEntry.Combatant)
	assert.Equal(t, 2, pcEntry.EntityIndex)

	monEntry := q.GetEntry(5, false)
	require.NotNil(t, monEntry)
	assert.Same(t, monster, monEntry.Combatant)

	// Non-existent entry
	missing := q.GetEntry(99, true)
	assert.Nil(t, missing)
}

// TestCombatTurnQueue_ActiveEntryCount verifies the active/total count utilities.
func TestCombatTurnQueue_ActiveEntryCount(t *testing.T) {
	q := NewCombatTurnQueue()

	q.AddCombatant(newMockPC(15), 0)
	q.AddCombatant(newMockPC(18), 1)
	q.AddCombatant(newMockMonster(12), 0)

	assert.Equal(t, 3, q.TotalEntryCount())
	assert.Equal(t, 3, q.ActiveEntryCount())

	q.MarkDead(1, true)

	assert.Equal(t, 3, q.TotalEntryCount(), "total should include dead entries")
	assert.Equal(t, 2, q.ActiveEntryCount(), "active should exclude dead entries")

	q.MarkFled(0, false)

	assert.Equal(t, 3, q.TotalEntryCount())
	assert.Equal(t, 1, q.ActiveEntryCount(), "active should exclude dead and fled")
}

// TestCombatTurnQueue_MidCombatDeath verifies that killing a combatant mid-combat
// correctly removes them from the turn rotation without disrupting other turns.
func TestCombatTurnQueue_MidCombatDeath(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(20)            // countdown = 16
	monster1 := newMockMonster(25) // countdown = 11
	monster2 := newMockMonster(20) // countdown = 16

	q.AddCombatant(pc, 0)
	q.AddCombatant(monster1, 0)
	q.AddCombatant(monster2, 1)

	// Monster1 (dex=25) should act first
	first := q.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.Same(t, monster1, first.Combatant)

	// Kill monster1 after it acts
	q.MarkDead(0, false)

	// Now only PC and monster2 remain, both with countdown=16.
	// PC was added first, so PC should act next.
	second := q.AdvanceToNextActor()
	require.NotNil(t, second)
	assert.Same(t, pc, second.Combatant,
		"after monster1 death, PC should get next turn")

	third := q.AdvanceToNextActor()
	require.NotNil(t, third)
	assert.Same(t, monster2, third.Combatant,
		"monster2 should get turn after PC")

	// Verify counts
	pcs, monsters := q.CountRemaining()
	assert.Equal(t, 1, pcs)
	assert.Equal(t, 1, monsters)
}

// ─── PC/Monster Ordering Mirrors Original ───────────────────────────────────

// TestCombatTurnQueue_PCsBeforeMonsters_SameDex verifies that when a PC and
// monster have the same dex, the PC acts first (matching original combat queue[]
// layout where slots 0-5 are PCs, 6-31 are monsters).
// See ref:COMBAT-0002.
func TestCombatTurnQueue_PCsBeforeMonsters_SameDex(t *testing.T) {
	q := NewCombatTurnQueue()

	pc := newMockPC(20)
	monster := newMockMonster(20)

	// PCs added first, then monsters — matches original layout
	q.AddCombatant(pc, 0)
	q.AddCombatant(monster, 0)

	first := q.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.True(t, first.Combatant.IsPC(),
		"at same dex, PC should act before monster (earlier slot in queue)")
}

// ─── Realistic Party vs Monster Group ───────────────────────────────────────

// TestCombatTurnQueue_RealisticCombat_PartyVsOrcs simulates a realistic combat
// scenario with a 3-PC party vs 3 orcs, verifying reasonable turn distribution.
func TestCombatTurnQueue_RealisticCombat_PartyVsOrcs(t *testing.T) {
	q := NewCombatTurnQueue()

	// Typical party: Avatar(dex=22), Iolo(dex=18), Shamino(dex=20)
	avatar := newMockPC(22)  // countdown = 14
	iolo := newMockPC(18)    // countdown = 18
	shamino := newMockPC(20) // countdown = 16

	// 3 orcs with dex=15
	orc0 := newMockMonster(15) // countdown = 21
	orc1 := newMockMonster(15)
	orc2 := newMockMonster(15)

	// PCs first, then monsters
	q.AddCombatant(avatar, 0)
	q.AddCombatant(iolo, 1)
	q.AddCombatant(shamino, 2)
	q.AddCombatant(orc0, 0)
	q.AddCombatant(orc1, 1)
	q.AddCombatant(orc2, 2)

	// Run 24 turns and count distribution
	pcTurns := 0
	monsterTurns := 0

	for i := 0; i < 24; i++ {
		entry := q.AdvanceToNextActor()
		require.NotNil(t, entry, "turn %d should have an actor", i+1)

		if entry.Combatant.IsPC() {
			pcTurns++
		} else {
			monsterTurns++
		}
	}

	// PCs are faster on average (dex 20 avg) vs orcs (dex 15)
	// PCs should get noticeably more turns
	assert.Greater(t, pcTurns, monsterTurns,
		"faster party (avg dex 20) should get more turns than slower orcs (dex 15): PCs=%d monsters=%d",
		pcTurns, monsterTurns)

	// First action should be avatar (fastest PC, countdown=14)
	// Verify by resetting and checking
	q2 := NewCombatTurnQueue()
	q2.AddCombatant(newMockPC(22), 0)
	q2.AddCombatant(newMockPC(18), 1)
	q2.AddCombatant(newMockPC(20), 2)
	q2.AddCombatant(newMockMonster(15), 0)

	first := q2.AdvanceToNextActor()
	require.NotNil(t, first)
	assert.Equal(t, 0, first.EntityIndex, "avatar (dex=22, countdown=14) should act first")
	assert.True(t, first.Combatant.IsPC())
}
